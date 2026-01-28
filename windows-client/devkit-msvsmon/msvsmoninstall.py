#!/bin/env python3
import glob
import os
import subprocess
import logging
import argparse
import re
import json
import shutil

logging.basicConfig(format='%(levelname)s:%(message)s', level=logging.DEBUG)
logger = logging.getLogger(__name__)


class MSVSMonPatcher:
    def __init__(self, conf):
        self.conf = conf
        self.msvsmon_dirs = glob.glob(
            os.path.join(
                os.path.realpath(os.path.dirname(__file__)),
                'msvsmon2???'
            )
        )
        if len(self.msvsmon_dirs) == 0:
            logger.warning('msvsmon binaries seems to be missing!')

        self.patch_path = None
        if not self.conf.no_patch:
            patches = glob.glob(
                os.path.join(
                    os.path.realpath(os.path.dirname(__file__)),
                    'msvsmon-*.patch.gz'
                )
            )
            if len(patches) == 0:
                logger.warning('No patches matching expected pattern!')
            else:
                self.patch_path = sorted(patches)[-1]
                logger.info(f'Patch to apply: {self.patch_path}')

    def ProcessProtonFolder(self, proton_folder):
        logger.info(f'Process Proton folder: {proton_folder}')

        for msvsmon_dir in self.msvsmon_dirs:
            proton_folder_msvsmon = os.path.join(proton_folder, os.path.basename(msvsmon_dir))
            logger.info(f'Copying {msvsmon_dir} to {proton_folder_msvsmon}')
            shutil.copytree(msvsmon_dir, proton_folder_msvsmon, dirs_exist_ok=True)

        # No longer used, was pretty fragile anyway. The proton script has all needed functionality now.
        if self.patch_path is not None and not self.conf.no_patch:
            logger.info(f'Check patch {self.patch_path}')
            # lots of patchutils libraries for python, but I'm trying to avoid adding dependencies
            # This is not very robust, only meant to handle the following:
            # - proton updates and drops in a new 'proton' script that we need to patch again
            # - there is a new version of the patch that requires a rollback and re-apply
            version = re.search(r'msvsmon-(.*)\.patch\.gz', self.patch_path)[1]
            patch_state_file = os.path.join(proton_folder, 'msvsmon-patch-state.json')
            if os.path.exists(patch_state_file):
                state = json.load(open(patch_state_file, 'r'))
                if state['version'] != version:
                    logger.info(f"Folder {proton_folder} is patched to version {state['version']}, rollback and apply version {version}")
                    # if proton *also* updated, we're probably screwed here and reverting to an older version of the script!
                    subprocess.call(f"tar xvf {state['backup']}", shell=True, cwd=proton_folder)
                    os.unlink(patch_state_file)
                else:
                    # in this case, we proceed to testing if the patch can still be applied, since a Proton update could have reverted from under us
                    logger.info(f'State file reports that folder {proton_folder} has been previously patched to the intended version {version}')
            else:
                # should only need this in the transition to a state file coming from version 0.20210805.0 of the patch
                orig = os.path.join(proton_folder, 'proton.orig')
                if os.path.exists(orig):
                    logger.info(f'Restoring from {orig}, presumably from previous devkit tool patching')
                    shutil.copy(orig, os.path.join(proton_folder, 'proton'))
            try:
                output = subprocess.check_output(f'zcat {self.patch_path} | patch -N --dry-run', shell=True, cwd=proton_folder, universal_newlines=True)
            except subprocess.SubprocessError as e:
                if e.output.find('previously applied') != -1:
                    logger.info(f'Patch already applied, skip {proton_folder}')
                else:
                    logger.warning(e.output)
                    # could be other causes, but very likely the proton script has changed on us
                    raise Exception(f'The proton launch script has been modified in an incompatible way, cannot setup for remote debug.')
            else:
                # Prepare a rollback / backup archive
                backup_files = re.findall('checking file (.*)', output)
                #logger.debug(f'Backing up files about to be patched: {backup_files!r}')
                backup_archive = f'msvsmon-backup-{version}.tar'
                subprocess.check_call(f"tar -cvf {backup_archive} {' '.join(backup_files)}", shell=True, cwd=proton_folder)
                logger.info(f'Apply patch to {proton_folder}')
                subprocess.call(f'zcat {self.patch_path} | patch -b -N', shell=True, cwd=proton_folder, universal_newlines=True)
                # Write a state file to support future updates
                state = {
                    'version': version,
                    'backup': backup_archive,
                }
                json.dump(state, open(patch_state_file, 'w'), indent=2, sort_keys=True)

    def ProcessSteamLibrary(self, steam_library_folder):
        logger.info(f'Process Steam library: {steam_library_folder}')
        for proton_folder in glob.glob(os.path.join(steam_library_folder, 'steamapps/common/Proton*')):
            install = proton_folder.endswith('Proton - Experimental')
            if not install:
                try:
                    version = int(re.search(r'Proton ([0-9]*)\..*$', proton_folder).group(1))
                    if version >= 7:
                        install = True
                except:
                    pass
            if not install:
                logger.warning(f'Proton {proton_folder!r} does not support remote debugging (>= 7 or experimental required), skipping.')
            else:
                self.ProcessProtonFolder(proton_folder)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Prepare a Proton installation for remote debugging')
    parser.add_argument('--process-library', required=False, action='store', help='Process a Steam library path')
    parser.add_argument('--process-proton', required=False, action='store', help='Process a Proton folder')
    parser.add_argument('--no-patch', required=False, default=True, action='store_true', help='Disable the patching feature')
    conf = parser.parse_args()

    patcher = MSVSMonPatcher(conf)
    if conf.process_library is not None:
        patcher.ProcessSteamLibrary(conf.process_library)
    elif conf.process_proton is not None:
        patcher.ProcessProtonFolder(conf.process_proton)
    else:
        # TODO: extract the list of libraries from ~/.local/share/Steam/steamapps/libraryfolders.vdf, and iterate over that
        patcher.ProcessSteamLibrary(os.path.realpath(os.path.expanduser('~/.steam/steam')))
