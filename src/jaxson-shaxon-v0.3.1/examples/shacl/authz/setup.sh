#!/bin/sh
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
#
# Sets up the SHACL harnesses in this folder. See SETUP.md.
#
#   ./setup.sh            create .venv and install pyshacl (about 36 MB)
#   ./setup.sh --jena     also fetch Apache Jena's jars into .jena/lib (about 17 MB)
#   ./setup.sh --check    change nothing; report what is present and what is missing
#
# Environment: PYTHON (interpreter used to create the venv, default python3).
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
cd "$HERE"

DO_JENA=0
CHECK=0
for arg in "$@"; do
    case "$arg" in
        --jena) DO_JENA=1 ;;
        --check) CHECK=1 ;;
        -h|--help) sed -n '5,12p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "setup.sh: unknown argument: $arg" >&2; exit 2 ;;
    esac
done

BASE_PY=${PYTHON:-python3}
VENV_PY="$HERE/.venv/bin/python"
JENA_LIB=${JENA_LIB:-$HERE/.jena/lib}

have() { command -v "$1" >/dev/null 2>&1; }

report() {
    echo "== what is present"
    if have "$BASE_PY"; then echo "  python:   $("$BASE_PY" --version 2>&1)"; else echo "  python:   MISSING ($BASE_PY)"; fi
    if [ -x "$VENV_PY" ] && "$VENV_PY" -c 'import pyshacl, rdflib' 2>/dev/null; then
        echo "  pyshacl:  $("$VENV_PY" -c 'import pyshacl, rdflib; print("pyshacl", pyshacl.__version__, "rdflib", rdflib.__version__)')"
    else
        echo "  pyshacl:  MISSING (run ./setup.sh)"
    fi
    if have java; then echo "  java:     $(java -version 2>&1 | sed -n 's/.*version "\(.*\)".*/\1/p' | head -n 1)"; else echo "  java:     MISSING (needed for Jena only)"; fi
    if have mvn; then echo "  maven:    present"; else echo "  maven:    MISSING (needed to fetch Jena only)"; fi
    if have javac; then echo "  javac:    present"; else echo "  javac:    MISSING (needed only for Jena's engine-only timing column)"; fi
    if [ -d "$JENA_LIB" ] && ls "$JENA_LIB"/jena-cmds-*.jar >/dev/null 2>&1; then
        echo "  jena:     $(ls "$JENA_LIB" | sed -n 's/^jena-cmds-\(.*\)\.jar$/\1/p' | head -n 1) in .jena/lib"
    else
        echo "  jena:     MISSING (run ./setup.sh --jena; optional)"
    fi
    if have go; then echo "  go:       $(go version | awk '{print $3}') (needed for ./run.sh timing)"; else echo "  go:       MISSING (needed for ./run.sh timing only)"; fi
}

if [ "$CHECK" -eq 1 ]; then
    report
    exit 0
fi

if ! have "$BASE_PY"; then
    echo "setup.sh: $BASE_PY not found; set PYTHON to a Python 3.9 or later interpreter" >&2
    exit 1
fi

if [ ! -x "$VENV_PY" ]; then
    echo "creating .venv"
    "$BASE_PY" -m venv "$HERE/.venv"
fi
echo "installing requirements.txt into .venv"
"$VENV_PY" -m pip install --quiet -r requirements.txt

if [ "$DO_JENA" -eq 1 ]; then
    if ! have java; then echo "setup.sh: java not found; Jena needs Java 17 or later" >&2; exit 1; fi
    if ! have mvn; then echo "setup.sh: mvn not found; it is only used to fetch the jars" >&2; exit 1; fi
    echo "fetching Apache Jena jars into .jena/lib"
    # JAVA_TOOL_OPTIONS is cleared: some environments set proxy options in it
    # that Maven does not need and that print a banner on every JVM start.
    JAVA_TOOL_OPTIONS= mvn -q -B -f "$HERE/pom.xml" dependency:copy-dependencies -DoutputDirectory="$JENA_LIB"
fi

report
echo "done. next: ./run.sh cases"
