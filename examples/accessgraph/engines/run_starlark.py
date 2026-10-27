# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""The access-decision rules in Starlark (starlark/rules.star), run through starlark-go.

  python3 run_starlark.py N      # N generated databases, compared with the reference

Build the runner first: `go build -o starlark_run .` in starlark/. STARLARK_RUN names another
binary, and STARLARK_STEPS sets the interpreter's step budget (a run past it stops with an error).
"""
import json, os, subprocess, sys, tempfile
from common import *

HERE = os.path.dirname(os.path.abspath(__file__))


def run_counted(db):
    """(decisions, steps) for db."""
    f = tempfile.NamedTemporaryFile("w", suffix=".json", delete=False)
    json.dump(db, f)
    f.close()
    try:
        cmd = [os.environ.get("STARLARK_RUN", os.path.join(HERE, "starlark", "starlark_run")),
               os.path.join(HERE, "starlark", "rules.star"), f.name]
        if os.environ.get("STARLARK_STEPS"):
            cmd.append(os.environ["STARLARK_STEPS"])
        r = subprocess.run(cmd, capture_output=True, text=True)
        steps = int(r.stderr.strip().splitlines()[0].split()[-1]) if r.stderr.startswith("steps:") else None
        if r.returncode:
            raise RuntimeError(r.stderr.strip()[:300])
        return json.loads(r.stdout), steps
    finally:
        os.unlink(f.name)


def run(db):
    return run_counted(db)[0]


if __name__ == "__main__":
    sys.exit(1 if compare("starlark", run, int(sys.argv[1])) else 0)
