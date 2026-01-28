#!/usr/bin/env python3
"""Resolve devkit game shortcuts with Steam.

This module synchronizes devkit games between disk and Steam shortcuts.
Modified for Bazzite/Universal devkit - uses steam-shortcut-manager instead of Steam IPC.
"""

import sys
import os
import logging

from . import (
    validate_steam_client,
    add_steam_shortcut,
    remove_steam_shortcut,
    list_steam_shortcuts,
    shortcut_manager_available,
)

import logging as logging_module
logger = logging_module.getLogger(__name__)

# Prefix used to identify devkit shortcuts in Steam
DEVKIT_SHORTCUT_PREFIX = 'Devkit: '


def resolve_shortcuts():
    """Synchronize devkit games between disk and Steam shortcuts.

    This function:
    1. Scans ~/devkit-game for installed devkit games
    2. Gets the list of registered devkit shortcuts from Steam
    3. Removes orphaned shortcuts (registered but not on disk)
    4. Adds missing shortcuts (on disk but not registered)
    """
    # Verify Steam is installed
    validate_steam_client()

    # Check that steam-shortcut-manager is available
    if not shortcut_manager_available():
        logger.warning(
            'steam-shortcut-manager binary not found. '
            'Shortcuts cannot be synchronized without it.'
        )
        return

    # Scan the devkit games on disk
    devkit_game_path = os.path.expanduser('~/devkit-game')
    installed_gameids = set()

    if not os.path.exists(devkit_game_path):
        logger.info('%r does not exist, creating', devkit_game_path)
        os.makedirs(devkit_game_path, exist_ok=True)

    entries = sorted(os.scandir(devkit_game_path), key=lambda entry: entry.name)
    directories = [e for e in entries if e.is_dir()]

    for d in directories:
        gameid = d.name
        file_names = [f.name for f in entries if f.is_file() and f.name.startswith(gameid)]
        has_argv = '{0}-argv.json'.format(gameid) in file_names
        has_settings = '{0}-settings.json'.format(gameid) in file_names

        if not has_argv and not has_settings:
            logger.info(
                'Subfolder %r in %r is not accompanied by devkit configuration files, ignoring',
                d.name, devkit_game_path
            )
            continue

        logger.info('Found installed Devkit Game: %r', gameid)
        installed_gameids.add(gameid)

    # Get registered devkit shortcuts from Steam
    registered_gameids = set()
    try:
        shortcuts = list_steam_shortcuts()
        for sc in shortcuts:
            app_name = sc.get('AppName', '') or sc.get('appname', '') or sc.get('name', '')
            if app_name.startswith(DEVKIT_SHORTCUT_PREFIX):
                gameid = app_name[len(DEVKIT_SHORTCUT_PREFIX):]
                logger.info('Found Devkit Game registered with Steam: %r', gameid)
                registered_gameids.add(gameid)
    except Exception as e:
        logger.warning('Failed to list Steam shortcuts: %s', e)
        logger.info('Will attempt to add missing shortcuts anyway')

    logger.info(
        'Steam has %d registered devkit game(s), disk has %d installed',
        len(registered_gameids), len(installed_gameids)
    )

    # Remove orphaned shortcuts (registered in Steam but not on disk)
    for remove_gameid in registered_gameids - installed_gameids:
        shortcut_name = '{0}{1}'.format(DEVKIT_SHORTCUT_PREFIX, remove_gameid)
        logger.info('Removing stale registered Devkit Game: %r', remove_gameid)
        try:
            remove_steam_shortcut(shortcut_name)
            logger.info('Successfully removed shortcut for %r', remove_gameid)
        except Exception as e:
            logger.warning('Failed to remove shortcut for %r: %s', remove_gameid, e)

    # Add missing shortcuts (on disk but not registered in Steam)
    for add_gameid in installed_gameids - registered_gameids:
        game_dir = os.path.join(devkit_game_path, add_gameid)
        shortcut_name = '{0}{1}'.format(DEVKIT_SHORTCUT_PREFIX, add_gameid)

        # Try to find an executable in the game directory
        exe = _find_game_executable(game_dir, add_gameid)

        logger.info('Registering installed Devkit Game: %r', add_gameid)
        try:
            add_steam_shortcut(
                name=shortcut_name,
                exe=exe,
                start_dir=game_dir,
                tags=['devkit']
            )
            logger.info('Successfully registered shortcut for %r', add_gameid)
        except Exception as e:
            logger.warning('Failed to register shortcut for %r: %s', add_gameid, e)


def _find_game_executable(game_dir, gameid):
    """Find the game executable in the game directory.

    Args:
        game_dir: Path to the game directory
        gameid: The game identifier

    Returns:
        Path to the executable (launch.sh, or a detected executable)
    """
    # Prefer launch.sh if it exists
    launch_script = os.path.join(game_dir, 'launch.sh')
    if os.path.exists(launch_script):
        return launch_script

    # Try gameid as executable name
    gameid_exe = os.path.join(game_dir, gameid)
    if os.path.exists(gameid_exe) and os.access(gameid_exe, os.X_OK):
        return gameid_exe

    # Try gameid.sh
    gameid_sh = os.path.join(game_dir, gameid + '.sh')
    if os.path.exists(gameid_sh):
        return gameid_sh

    # Look for any executable file in the directory
    try:
        for entry in os.scandir(game_dir):
            if entry.is_file() and os.access(entry.path, os.X_OK):
                # Skip common non-game executables
                if entry.name in ('configure', 'install', 'uninstall'):
                    continue
                return entry.path
    except OSError:
        pass

    # Fallback: return game_dir and let Steam figure it out
    logger.warning(
        'Could not find executable for %r in %r, using directory as exe',
        gameid, game_dir
    )
    return game_dir


if __name__ == '__main__':
    logging.basicConfig(format='%(message)s', level=logging.DEBUG)
    resolve_shortcuts()
