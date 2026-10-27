# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Record steps and wall time at each size, on the reference and on both runtimes.

  python3 scale.py --jaxrun PATH [--sizes medium,large,xlarge] [--seed N] [--no-standin]

For each size it generates one database, evaluates it with the reference
(oracle.py), the stand-in interpreter (jx.py) and the real runner, requires
all three to give identical decisions, and prints steps and wall time. The
declared step limit is the one derived from the data (cost.py). It is never
raised to make a run pass, and a run that exceeds it stops with
RESOURCE_ERROR/STEPS; the table shows measured against declared steps.

The stand-in is an interpreter written in Python and is slow at the largest
size (about a minute at 5,000 users); --no-standin skips it, and then the step
count is not reported.
"""
import json
import os
import subprocess
import sys
import tempfile
import time

import gen
import jx
import oracle
import synth


def run_real(jaxrun, pkg):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(pkg, f)
    try:
        t = time.perf_counter()
        r = subprocess.run([jaxrun, f.name], capture_output=True, text=True)
        dt = time.perf_counter() - t
    finally:
        os.unlink(f.name)
    if r.returncode:
        raise RuntimeError("jaxrun failed: " + (r.stdout + r.stderr).strip()[:300])
    return json.loads(r.stdout), dt


def main(argv):
    args = list(argv[1:])

    def opt(name, default=None):
        if name in args:
            i = args.index(name)
            v = args[i + 1]
            del args[i:i + 2]
            return v
        return default

    jaxrun = opt("--jaxrun")
    sizes = opt("--sizes", "medium,large,xlarge").split(",")
    seed = int(opt("--seed", "0"))
    standin = "--no-standin" not in args
    if not jaxrun:
        sys.stderr.write(__doc__)
        return 2

    print("%-8s %6s %9s %12s %12s %6s %12s %11s" % (
        "size", "users", "requests", "steps", "declared", "used", "stand-in s", "jaxrun s"))
    for size in sizes:
        db = synth.generate(seed, size)
        t = time.perf_counter()
        want = oracle.evaluate(db)
        t_oracle = time.perf_counter() - t
        pkg = gen.build(db)
        steps = t_si = None
        if standin:
            t = time.perf_counter()
            out, steps = jx.run(pkg)
            t_si = time.perf_counter() - t
            if out != want:
                print("MISMATCH: stand-in differs from the reference at", size)
                return 1
        real, t_real = run_real(jaxrun, pkg)
        if real != want:
            print("MISMATCH: jaxrun differs from the reference at", size)
            return 1
        declared = pkg["limits"]["steps"]
        print("%-8s %6d %9d %12s %12d %6s %12s %11.2f   (reference %.3f s)" % (
            size, len(db["users"]), len(db["requests"]),
            "%d" % steps if steps else "-", declared,
            ("%.0f%%" % (100.0 * steps / declared)) if steps else "-",
            "%.2f" % t_si if t_si else "-", t_real, t_oracle))
    print("all runtimes agree with the reference on every decision")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
