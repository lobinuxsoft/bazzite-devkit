#!/usr/bin/env python
# encoding: utf-8
"""Utility functions for the Steam client hook scripts

Modified for Bazzite/Universal Devkit - uses steam-shortcut-manager instead of Steam IPC pipe.
"""

import sys
import os
import traceback
import tempfile
import json
import logging
import errno
import contextlib
import time
import subprocess
import platform

# fcntl is only available on Unix
try:
    import fcntl
    HAS_FCNTL = True
except ImportError:
    HAS_FCNTL = False


import logging as logging_module
logger = logging_module.getLogger(__name__)


# ============================================================================
# steam-shortcut-manager integration
# ============================================================================

def get_shortcut_manager_binary():
    """Returns path to the steam-shortcut-manager binary"""
    base_dir = os.path.dirname(__file__)
    if platform.system() == 'Windows':
        return os.path.join(base_dir, 'bin', 'steam-shortcut-manager.exe')
    return os.path.join(base_dir, 'bin', 'steam-shortcut-manager-linux')


def _check_shortcut_manager_available():
    """Check if steam-shortcut-manager binary is available"""
    binary = get_shortcut_manager_binary()
    if not os.path.exists(binary):
        raise FileNotFoundError(
            f'steam-shortcut-manager binary not found at {binary}. '
            'Please ensure the binary is compiled and placed in the bin/ directory.'
        )
    if not os.access(binary, os.X_OK) and platform.system() != 'Windows':
        raise PermissionError(
            f'steam-shortcut-manager binary at {binary} is not executable. '
            'Please run: chmod +x {binary}'
        )
    return binary


def add_steam_shortcut(name, exe, start_dir, launch_options='', icon='', tags=None):
    """Add a shortcut to Steam using steam-shortcut-manager

    Args:
        name: Display name for the shortcut in Steam
        exe: Path to the executable
        start_dir: Working directory for the executable
        launch_options: Optional launch arguments
        icon: Optional path to icon file
        tags: Optional list of tags/categories

    Returns:
        stdout from steam-shortcut-manager

    Raises:
        Exception: If the shortcut could not be added
    """
    binary = _check_shortcut_manager_available()
    cmd = [binary, 'add', name, exe, '--start-dir', start_dir]

    if launch_options:
        cmd.extend(['--launch-options', launch_options])
    if icon:
        cmd.extend(['--icon', icon])
    if tags:
        cmd.extend(['--tags', ','.join(tags)])

    logger.debug('Running steam-shortcut-manager: %s', ' '.join(cmd))
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        error_msg = result.stderr.strip() or result.stdout.strip() or 'Unknown error'
        raise Exception(f'Error adding shortcut: {error_msg}')

    logger.info('Added Steam shortcut: %s', name)
    return result.stdout


def remove_steam_shortcut(name):
    """Remove a shortcut from Steam using steam-shortcut-manager

    Args:
        name: Display name of the shortcut to remove

    Returns:
        stdout from steam-shortcut-manager

    Raises:
        Exception: If the shortcut could not be removed
    """
    binary = _check_shortcut_manager_available()
    cmd = [binary, 'remove', name]

    logger.debug('Running steam-shortcut-manager: %s', ' '.join(cmd))
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        error_msg = result.stderr.strip() or result.stdout.strip() or 'Unknown error'
        raise Exception(f'Error removing shortcut: {error_msg}')

    logger.info('Removed Steam shortcut: %s', name)
    return result.stdout


def list_steam_shortcuts():
    """List all shortcuts from Steam using steam-shortcut-manager

    Returns:
        List of shortcut dictionaries with keys like 'AppName', 'Exe', etc.

    Raises:
        Exception: If shortcuts could not be listed
    """
    binary = _check_shortcut_manager_available()
    cmd = [binary, 'list', '-o', 'json']

    logger.debug('Running steam-shortcut-manager: %s', ' '.join(cmd))
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        error_msg = result.stderr.strip() or result.stdout.strip() or 'Unknown error'
        raise Exception(f'Error listing shortcuts: {error_msg}')

    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as e:
        logger.error('Failed to parse steam-shortcut-manager output: %s', result.stdout)
        raise Exception(f'Error parsing shortcut list: {e}')


def shortcut_manager_available():
    """Check if the steam-shortcut-manager binary is available

    Returns:
        True if the binary exists and is executable, False otherwise
    """
    try:
        _check_shortcut_manager_available()
        return True
    except (FileNotFoundError, PermissionError):
        return False


# ============================================================================
# File locking helpers (cross-platform)
# ============================================================================

def _lock_file(f, exclusive=True):
    """Lock a file (Unix only, no-op on Windows)"""
    if HAS_FCNTL:
        fcntl.flock(f, fcntl.LOCK_EX if exclusive else fcntl.LOCK_SH)


def _unlock_file(f):
    """Unlock a file (Unix only, no-op on Windows)"""
    if HAS_FCNTL:
        fcntl.flock(f, fcntl.LOCK_UN)


# ============================================================================
# Output wrapping and Steam client validation
# ============================================================================

@contextlib.contextmanager
def wrap_outputs(stderr_prefix):
    # capture stderr to file to support debugging
    stderr_fd = sys.stderr.fileno()
    tf = tempfile.NamedTemporaryFile(
        mode='w+',
        prefix=stderr_prefix,
        delete=True)
    if sys.version_info >= (3, 4):
        # this API in the os module is only available for python3
        # but it does not seem to work with subprocess anyway
        os.set_inheritable(tf.file.fileno(), True)
        assert os.get_inheritable(tf.file.fileno())
    sys.stderr = tf.file

    # we can only write out a json response to stdout,
    # so redirect stdout to stderr,
    # and keep a handle on the original stdout for the response
    stdout_fd = os.dup(sys.stdout.fileno())
    os.dup2(sys.stderr.fileno(), sys.stdout.fileno())

    ctx = {}
    try:
        yield ctx
    except:
        logger.error(traceback.format_exc())
    finally:
        tf.flush()
        tf.seek(0)
        os.write(stderr_fd, tf.read().encode('utf-8'))
        if 'ret' in ctx:
            os.write(stdout_fd, json.dumps(ctx['ret']).encode('utf-8'))


class SteamClientNotRunningException(Exception):
    def __init__(self, error_message):
        self.error_message = error_message

    def __str__(self):
        return self.error_message


class SteamNotInstalledException(Exception):
    def __init__(self, error_message):
        self.error_message = error_message

    def __str__(self):
        return self.error_message


def validate_steam_client():
    """Verify that Steam is installed.

    For Bazzite/Universal devkit, we only check that Steam is installed,
    not that it's running with the IPC pipe. The steam-shortcut-manager
    handles shortcuts directly via the shortcuts.vdf file.
    """
    # Check common Steam installation paths
    steam_paths = [
        os.path.expanduser('~/.steam'),
        os.path.expanduser('~/.local/share/Steam'),
        os.path.expanduser('~/.steam/steam'),
    ]

    steam_found = False
    for steam_path in steam_paths:
        if os.path.exists(steam_path):
            steam_found = True
            logger.info('Found Steam installation at %s', steam_path)
            break

    if not steam_found:
        raise SteamNotInstalledException(
            'Steam does not appear to be installed. '
            'Checked paths: ' + ', '.join(steam_paths)
        )

    # Optionally check if Steam is running (informational only)
    pid_path = os.path.normpath(
        os.path.realpath(
            os.path.expanduser('~/.steam/steam.pid')))
    if os.path.exists(pid_path):
        try:
            pid = int(open(pid_path, 'rt').read())
            os.kill(pid, 0)
            logger.info('Steam client is running (pid %s)', pid)
        except (ValueError, OSError):
            logger.info('Steam client is not currently running')
    else:
        logger.info('Steam client is not currently running')


def validate_steam_client_legacy():
    """Legacy validation - requires Steam client to be running with IPC pipe.

    This is the original SteamOS validation that requires the steam.pipe.
    Kept for backward compatibility but not used in Bazzite mode.
    """
    pid_path = os.path.normpath(
        os.path.realpath(
            os.path.expanduser('~/.steam/steam.pid')))
    if not os.path.exists(pid_path):
        raise SteamClientNotRunningException('{0} does not exist'.format(pid_path))
    try:
        pid = int(open(pid_path, 'rt').read())
    except Exception:
        raise SteamClientNotRunningException('{0} is invalid'.format(pid_path))
    try:
        os.kill(pid, 0)
    except OSError:
        raise SteamClientNotRunningException('{0} does not refer to a valid process'.format(pid_path))
    logger.info('Found steam client pid %s', pid)


def execute_steam_client_command(cmd):
    """Send a command to the steam client over the IPC pipe.

    NOTE: This is the legacy SteamOS IPC method. For Bazzite/Universal devkit,
    use add_steam_shortcut(), remove_steam_shortcut(), and list_steam_shortcuts()
    which use steam-shortcut-manager instead.
    """
    pipe_path = os.path.normpath(
        os.path.realpath(
            os.path.expanduser('~/.steam/steam.pipe')))
    try:
        pipe = open(pipe_path, 'wb+', 0)
    except IOError:
        raise Exception('cannot open steam client pipe')
    session_token = open(os.path.expanduser('~/.steam/steam.token')).read()
    #pipe_cmd = 'steam://{0}'.format(cmd)
    # ^ hack to execute a normal command over the IPC directly - sometimes useful
    pipe_cmd = 'devkit-1 steam://devkit-1/{0}/{1}'.format(
        session_token,
        cmd
    )
    logger.debug('Sending command line:')
    logger.debug(pipe_cmd)
    pipe.write('{0}\n'.format(pipe_cmd).encode('utf-8'))
    pipe.close()


# ============================================================================
# Game configuration file management
# ============================================================================


def save_argv(gameid, argv):
    """Save command line and arguments if provided"""

    if argv is None:
        return

    argvfile = os.path.join(os.getenv("HOME"), "devkit-game",
                            gameid + "-argv.json")
    try:
        with open(argvfile, "w") as argvf:
            _lock_file(argvf)
            json.dump(argv, argvf)
            _unlock_file(argvf)
    except IOError:
        raise Exception(
            "Unable to open argv file for writing: {0}".format(argvfile))


def obtain_argv(gameid, argv):
    """Obtain command line with arguments"""

    # If present and not None or [], just return the local arguments
    if argv:
        return argv

    # From here, expect arguments to have been saved previously
    argvfile = os.path.join(os.getenv("HOME"), "devkit-game",
                            gameid + "-argv.json")
    try:
        with open(argvfile, "r") as argvf:
            _lock_file(argvf)
            argv = json.load(argvf)
            _unlock_file(argvf)
    except IOError:
        raise Exception(
            "Unable to open argv file for reading: {0}".format(argvfile))
    return argv


def save_settings(gameid, data):
    """Save settings"""
    settingsfile = os.path.join(os.getenv("HOME"), "devkit-game",
                                gameid + "-settings.json")
    settings = dict()

    if data.get('clear_settings', False):
        settings = {}
    else:
        try:
            with open(settingsfile, "r") as f:
                _lock_file(f)
                settings = json.load(f)
                _unlock_file(f)
        except IOError as e:
            if (e.errno != errno.ENOENT):
                raise

    # Merge settings from new json
    if 'settings' in data:
        settings.update(data['settings'])

    try:
        with open(settingsfile, "w") as f:
            _lock_file(f)
            json.dump(settings, f)
            _unlock_file(f)
    except (IOError):
        raise Exception(
            "Unable to open settings file for writing: {0}".format(
                settingsfile
            ))

    return settings


def load_settings(gameid):
    settingsfile = os.path.join(os.getenv("HOME"), "devkit-game", gameid + '-settings.json')

    if not os.path.isfile(settingsfile):
        return None

    with open(settingsfile, "r") as f:
        _lock_file(f)
        settings = json.load(f)
        _unlock_file(f)

    return settings


class SteamResponse_Timeout(Exception):
    pass


class SteamResponse_Error(Exception):
    def __init__(self, error_response):
        self.error_response = error_response

    def __str__(self):
        return self.error_response


@contextlib.contextmanager
def wait_on_file_response(path, timeout=5):
    """
The pipe to the Steam Client is one way.
Responses from the Steam Client are written to filesystem.
Protocol is as follows:
- Steam Client creates a 'path.lock' file
- Steam Client writes either 'path' or 'path.error' to indicate a problem
- Steam Client deletes 'path.lock'
- Caller (us) can then read the response

NOTE 1: this function is used as a context manager and will block until a response comes in or timeout.

NOTE 2: the files are created by Steam when responding to a command. If the files already exist the response protocol will break.

NOTE 3: This is a legacy SteamOS IPC function. For Bazzite/Universal devkit,
use the steam-shortcut-manager functions instead.
    """
    lock_path = '{0}.lock'.format(path)
    error_path = '{0}.error'.format(path)
    max_count = timeout
    while True:
        time.sleep(1)
        if os.path.exists(error_path) or os.path.exists(path) and not os.path.exists(lock_path):
            if os.path.exists(error_path):
                with open(error_path, 'r') as f:
                    _lock_file(f)
                    error_response = f.read()
                    _unlock_file(f)
                    raise SteamResponse_Error(error_response)
            with open(path, 'r') as f:
                _lock_file(f)
                success_response = f.read()
                yield success_response
                _unlock_file(f)
                return
        max_count -= 1
        if max_count > 0:
            continue
        raise SteamResponse_Timeout()


# Setting up as a context manager so we never miss the deletion
# Creating a temporary .lock file to guard the create operation
@contextlib.contextmanager
def create_pid(pid_path):
    os.makedirs(os.path.dirname(pid_path), exist_ok=True)
    lock_path = '{0}.lock'.format(pid_path)
    try:
        lock_file = os.open(lock_path, os.O_CREAT | os.O_EXCL)
    except IOError as e:
        logger.error('cannot create lock file %s for pid file %s', lock_path, pid_path)
        logger.error('remove the lock file manually and run again if you are confident no other instance is active')
        raise

    pid_file = open(pid_path,'w')
    pid_file.write(str(os.getpid()))
    pid_file.flush()
    os.close(lock_file)
    os.unlink(lock_path)
    try:
        yield pid_file
    finally:
        pid_file.close()
        # Assume that's atomic and all is well, no need for another .lock
        os.unlink(pid_path)
