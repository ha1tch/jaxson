#!/bin/sh
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
#
# Launches the SHACL harnesses in this folder. See SETUP.md.
#
#   ./run.sh cases          every Shaxon case through pyshacl; exit 0 if all comparable ones agree
#   ./run.sh cases-verbose  the same, one line per case
#   ./run.sh jena           the same cases through Apache Jena SHACL (needs ./setup.sh --jena)
#   ./run.sh no-structure   pyshacl with the structure shapes dropped; passes when exactly
#                           the 6 malformed-request cases differ (the fail-open result)
#   ./run.sh mutants        break each SHACL rule in turn; exit 0 if every mutant is caught
#   ./run.sh timing         Shaxon, pyshacl and (if present) Jena at 100, 1000 and 4000 events,
#                           in both profiles: end-to-end (the RDF lift counted) and engine-only
#   ./run.sh timing-end-to-end   only the first profile
#   ./run.sh timing-engine       only the second: validation alone, the lift left out
#   ./run.sh all            cases, no-structure, mutants, then jena if installed
#
# Environment: PYTHON (default .venv/bin/python), JENA_LIB (default .jena/lib).
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
cd "$HERE"

PYTHON=${PYTHON:-$HERE/.venv/bin/python}
JENA_LIB=${JENA_LIB:-$HERE/.jena/lib}
export JENA_LIB

case "${1:-}" in
    -h|--help|help) sed -n '5,20p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
esac

if [ ! -x "$PYTHON" ]; then
    echo "run.sh: no Python at $PYTHON; run ./setup.sh first (or set PYTHON)" >&2
    exit 1
fi

have_jena() { ls "$JENA_LIB"/jena-cmds-*.jar >/dev/null 2>&1; }

need_jena() {
    if ! have_jena; then
        echo "run.sh: no Jena jars in $JENA_LIB; run ./setup.sh --jena (or set JENA_LIB)" >&2
        exit 1
    fi
}

# The malformed-request cases are the ones whose expectation is EXECUTION_ERROR
# in the Shaxon case files; without the structure shapes SHACL allows them.
no_structure() {
    out=$("$PYTHON" run_cases.py --no-structure) || true
    printf '%s\n' "$out" | tail -n 3
    case "$out" in
        *"differ: 6"*) echo "no-structure: as expected, the 6 malformed requests are allowed by the rules alone" ;;
        *) echo "no-structure: UNEXPECTED, expected exactly 6 cases to differ" >&2; return 1 ;;
    esac
}

# Runs a command, prints its last line under a heading, and returns the
# command's own exit status (not that of the pipe used to trim the output).
summarise() {
    label=$1
    shift
    echo "### $label"
    if out=$("$@" 2>&1); then rc=0; else rc=$?; fi
    printf '%s\n' "$out" | tail -n 1
    return "$rc"
}

target=${1:-cases}
case "$target" in
    cases) "$PYTHON" run_cases.py ;;
    cases-verbose) "$PYTHON" run_cases.py --verbose ;;
    jena) need_jena; "$PYTHON" run_cases.py --processor jena ;;
    no-structure) no_structure ;;
    mutants) "$PYTHON" mutants.py ;;
    timing) "$PYTHON" scale.py ;;
    timing-end-to-end) "$PYTHON" scale.py --profile end-to-end ;;
    timing-engine) "$PYTHON" scale.py --profile engine ;;
    all)
        status=0
        summarise "cases (pyshacl)" "$PYTHON" run_cases.py || status=1
        echo "### no-structure"; no_structure || status=1
        summarise "mutants" "$PYTHON" mutants.py || status=1
        if have_jena; then
            summarise "cases (jena)" "$PYTHON" run_cases.py --processor jena || status=1
        else
            echo "### cases (jena): skipped, no jars in $JENA_LIB (./setup.sh --jena)"
        fi
        exit "$status"
        ;;
    *) echo "run.sh: unknown target: $target (try ./run.sh help)" >&2; exit 2 ;;
esac
