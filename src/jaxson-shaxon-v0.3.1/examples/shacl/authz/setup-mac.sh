#!/bin/sh
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
#
# Sets up everything the SHACL harnesses and bench.sh need on macOS, with
# Homebrew. Run it from this folder. It asks before it installs anything.
#
#   ./setup-mac.sh --check      report what is present and what would be installed; changes nothing
#   ./setup-mac.sh              install what is missing (after asking), then run ./setup.sh --jena
#   ./setup-mac.sh --yes        the same without the question
#
# What it may install with brew (only if missing; sizes are approximate):
#   go        the Go toolchain, version 1.25 or later (about 250 MB)
#   openjdk   a JDK, for Jena and for compiling JenaTime.java (about 300 MB)
#   maven     only to fetch Jena's jars (about 10 MB, plus about 40 MB of jars
#             cached under ~/.m2 and 17 MB under .jena/lib)
#   python@3.13   only if there is no python3 of version 3.9 or later
# pyshacl and rdflib go into a local .venv (about 36 MB). Nothing is installed
# outside Homebrew's own directories, ~/.m2 and this folder; to undo it: brew
# uninstall what you do not want, and delete .venv, .jena and ~/.m2.
#
# Homebrew itself is not installed by this script. If it is missing the script
# prints the one-line installer from https://brew.sh and stops.
#
# Why no sudo and no symlink into /Library/Java: the JDK stays inside
# Homebrew's prefix and env-mac.sh puts it on PATH and sets JAVA_HOME, which is
# all that java, javac and Maven need.
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
cd "$HERE"

CHECK=0
YES=0
for arg in "$@"; do
    case "$arg" in
        --check) CHECK=1 ;;
        --yes|-y) YES=1 ;;
        -h|--help) sed -n '5,28p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "setup-mac.sh: unknown argument: $arg" >&2; exit 2 ;;
    esac
done

if [ "$(uname -s)" != Darwin ] && [ -z "${SETUP_MAC_FORCE:-}" ]; then
    echo "setup-mac.sh is for macOS. On Linux use ./setup.sh --jena (and your package manager for Java, Maven and Go)." >&2
    exit 1
fi

# Homebrew
BREW=
for c in /opt/homebrew/bin/brew /usr/local/bin/brew; do
    if [ -x "$c" ]; then BREW=$c; break; fi
done
if [ -z "$BREW" ] && command -v brew >/dev/null 2>&1; then BREW=$(command -v brew); fi
if [ -z "$BREW" ]; then
    echo "Homebrew is not installed. Install it first, then run this again:" >&2
    echo '  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"' >&2
    echo "(from https://brew.sh; it will tell you to add brew to your PATH when it finishes)" >&2
    exit 1
fi
eval "$("$BREW" shellenv)"

# Pick up anything already installed, so the checks see it.
# shellcheck disable=SC1091
. "$HERE/env-mac.sh"

have() { command -v "$1" >/dev/null 2>&1; }

go_ok() {
    have go || return 1
    v=$(go version | sed -n 's/.*go\([0-9][0-9]*\)\.\([0-9][0-9]*\).*/\1 \2/p')
    [ -n "$v" ] || return 1
    set -- $v
    [ "$1" -gt 1 ] || [ "$2" -ge 25 ]
}

java_ok() {
    have java && have javac || return 1
    v=$(JAVA_TOOL_OPTIONS= java -version 2>&1 | sed -n 's/.*version "\([0-9][0-9]*\).*/\1/p' | head -n 1)
    [ -n "$v" ] && [ "$v" -ge 17 ]
}

py_ok() {
    have python3 || return 1
    python3 -c 'import sys; sys.exit(0 if sys.version_info >= (3, 9) else 1)' 2>/dev/null
}

TODO=
if go_ok;   then echo "  go:      $(go version | awk '{print $3}')";             else echo "  go:      MISSING or older than 1.25 (brew install go)"; TODO="$TODO go"; fi
if java_ok; then echo "  java:    $(JAVA_TOOL_OPTIONS= java -version 2>&1 | sed -n 's/.*version "\(.*\)".*/\1/p' | head -n 1), with javac"; else echo "  java:    MISSING, older than 17, or no javac (brew install openjdk)"; TODO="$TODO openjdk"; fi
if have mvn; then echo "  maven:   present";                                      else echo "  maven:   MISSING (brew install maven)"; TODO="$TODO maven"; fi
if py_ok;   then echo "  python3: $(python3 --version 2>&1)";                    else echo "  python3: MISSING or older than 3.9 (brew install python@3.13)"; TODO="$TODO python@3.13"; fi

if [ "$CHECK" -eq 1 ]; then
    if [ -n "$TODO" ]; then echo "would install with brew:$TODO"; else echo "nothing to install"; fi
    exit 0
fi

if [ -n "$TODO" ]; then
    echo "About to run: brew install$TODO"
    if [ "$YES" -ne 1 ]; then
        printf 'Go ahead? [y/N] '
        read -r ans
        case "$ans" in y|Y|yes|YES) ;; *) echo "nothing installed"; exit 1 ;; esac
    fi
    # shellcheck disable=SC2086
    "$BREW" install $TODO
    # the new tools may be new on PATH
    # shellcheck disable=SC1091
    . "$HERE/env-mac.sh"
fi

go_ok   || { echo "setup-mac.sh: Go 1.25 or later is still not on PATH" >&2; exit 1; }
java_ok || { echo "setup-mac.sh: a JDK 17 or later with javac is still not on PATH (is JAVA_HOME set? see env-mac.sh)" >&2; exit 1; }
have mvn || { echo "setup-mac.sh: mvn is still not on PATH" >&2; exit 1; }

echo "== ./setup.sh --jena (pyshacl into .venv, Jena's jars into .jena/lib)"
./setup.sh --jena

cat <<'DONE'

Done. In a new shell, either run things through bench.sh (it loads env-mac.sh
itself) or load it by hand first:  . ./env-mac.sh

Next:
  ./bench.sh list                  the batches
  ./bench.sh --sizes "100 1000" run     a quick first run (a few minutes)
  ./bench.sh run                   the full run, resumable: run it again if interrupted
  ./bench.sh report                merge the results into results/REPORT.md
DONE
