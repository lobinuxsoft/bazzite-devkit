#! /bin/bash

# verbose
#export PS4='${LINENO}: '
#set -x

set -uo pipefail
shopt -s failglob

me=$(basename "$(readlink -f "$0")")
log () {
    echo "${me}[$$]: $*" >&2 || :
}

if [ -n "${STEAM_RUNTIME-}" ]; then
  # The devkit tool is setup to run at host level, possibly started from the CLI
  # The tool expects a suitable version of python installed
  if [ -n "${SRT_LAUNCHER_SERVICE_ALONGSIDE_STEAM-}" ]; then
    log 'Running in SLR environment, relaunching at host level'
    log "${STEAM_RUNTIME}/amd64/usr/bin/steam-runtime-launch-client" --alongside-steam --host -- "$0" "$@"
    exec "${STEAM_RUNTIME}/amd64/usr/bin/steam-runtime-launch-client" --alongside-steam --host -- "$0" "$@"
    # unreachable
  fi
  log 'Running in LDLP environment, relaunching with runtime disabled'
  log "${STEAM_RUNTIME}/scripts/switch-runtime.sh" --runtime="" -- "$0" "$@"
  exec "${STEAM_RUNTIME}/scripts/switch-runtime.sh" --runtime="" -- "$0" "$@"
  # unreachable
fi

# Find a suitable OS-level python interpreter
for p in python3.13 python3.12 python3.11 python3 python; do
  if which "$p" &>/dev/null; then
    VERS=$("$p" 2>/dev/null -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
    for v in 13 12 11; do
      if [ "$VERS" == "3.$v" ]; then
        PYTHON=( "$p" )
        PYZ=devkit-gui-cp3${v}.pyz
        break 2
      fi
    done
  fi
done

# No suitable python found, check for a working pyenv
if [ -z "${PYTHON-}" ]; then
  if which pyenv >/dev/null; then
    for v in 12 11 10 9; do
      PYENV_VERSION=$(pyenv versions | grep -o -m1 "3\.${v}\.[0-9]*")
      if [ -n "${PYENV_VERSION-}" ]; then
        export PYENV_VERSION
        PYTHON=( pyenv exec "python3.${v}" )
        PYZ=devkit-gui-cp3${v}.pyz
        break
      fi
    done

    if [ -z "${PYTHON-}" ]; then
      pyenv versions | grep '^\s*3\.\(12\|11\|10\|9\)'
      log "pyenv installed but no python 3.12, 3.11, 3.10 or 3.9 versions found"
      log "Run 'pyenv install <version>' with a version listed above"
      exit 1
    fi
  fi
fi

if [ -z "${PYTHON-}" ]; then
  log "No usable python found"
  log "Please install python 3.12, 3.11, 3.10 or 3.9 from your package manager or via pyenv"
  log "e.g apt install python3.11"
  log "    pacman -S pyenv"
  exit
fi

pushd "$(dirname "$0")/linux-client" > /dev/null || exit 1
if [ -z "${DEVKIT_DEBUG-}" ]; then
  "${PYTHON[@]}" "$PYZ" &>/dev/null &
  disown %1
else
  "${PYTHON[@]}" "$PYZ"
fi
popd > /dev/null || exit 1
