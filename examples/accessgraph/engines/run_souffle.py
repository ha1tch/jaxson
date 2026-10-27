# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import os, subprocess, sys, tempfile, shutil
from common import *
def run(db):
    d = tempfile.mkdtemp(prefix="sf_")
    try:
        for rel, rows in facts(db).items():
            with open(os.path.join(d, rel + ".facts"), "w") as f:
                f.write("".join("\t".join(r) + "\n" for r in rows))
        o = os.path.join(d, "out"); os.mkdir(o)
        r = subprocess.run(["souffle", "-F", d, "-D", o, os.environ.get("AG_DL", os.path.join(os.path.dirname(os.path.abspath(__file__)), "ag.dl"))], capture_output=True, text=True)
        if r.returncode: raise RuntimeError(r.stderr[:400])
        rd = lambda n: [l.rstrip("\n").split("\t") for l in open(os.path.join(o, n + ".csv"))]
        sup = {x[0] for x in rd("supported")}
        via = {a: b for a, b in rd("via")}
        den = {}
        for a, b in rd("den"): den.setdefault(a, set()).add(b)
        return assemble(db, sup, via, den)
    finally:
        shutil.rmtree(d, ignore_errors=True)
if __name__ == "__main__":
    sys.exit(1 if compare("souffle", run, int(sys.argv[1])) else 0)
