# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Tests for the derived step limit (cost.py, calibrate.py).

  python3 test_cost.py --jaxrun PATH

1. fresh data    on databases the fit never saw, the measured steps never exceed
                 the declared limit, and use at least half of it
2. loud failure  a limit a tenth of the measured steps is exceeded, and both
                 runtimes stop with RESOURCE_ERROR/STEPS; nothing raises it
3. stale model   a model whose program fingerprint no longer matches refuses to
                 give a limit, so a changed program cannot reuse old numbers
4. real-shaped   if the Amazon databases and their measured step counts are present
                 (they never were in the fit), the declared limit covers both
"""
import json
import os
import subprocess
import sys
import tempfile

import cost
import gen
import jx
import synth

FRESH = [("tiny", range(200, 215)), ("small", range(200, 205)), ("medium", range(200, 202))]


def run_real(jaxrun, pkg):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(pkg, f)
    try:
        r = subprocess.run([jaxrun, f.name], capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    return r.returncode, r.stdout + r.stderr


def main(argv):
    if "--jaxrun" not in argv:
        sys.stderr.write(__doc__)
        return 2
    jaxrun = argv[argv.index("--jaxrun") + 1]
    failed = 0
    m = cost.model()

    shares = []
    over = 0
    for size, seeds in FRESH:
        for seed in seeds:
            db = synth.generate(seed, size)
            pkg = gen.build(db)
            steps = jx.run(pkg)[1]
            shares.append(steps / pkg["limits"]["steps"])
            over += steps > pkg["limits"]["steps"]
    print("fresh data    %d databases the fit never saw: %d over the declared limit, used %.0f%% to %.0f%% of it (margin %.2f)"
          % (len(shares), over, 100 * min(shares), 100 * max(shares), m["margin"]))
    failed += over + (min(shares) < 0.5)

    amazon = os.path.join(os.path.dirname(os.path.abspath(__file__)), "data", "amazon-db")
    real = []
    for v in ("flat", "nested"):
        dbf, sf = os.path.join(amazon, v + ".json"), os.path.join(amazon, "steps-%s.json" % v)
        if os.path.exists(dbf) and os.path.exists(sf):
            d = json.load(open(dbf))
            real.append((v, json.load(open(sf))["steps"], cost.limit(d)))
    if real:
        print("real-shaped   " + "; ".join("%s: %d steps measured, %d declared (%.0f%% used)" % (v, s, l, 100.0 * s / l)
                                          for v, s, l in real))
        failed += sum(1 for v, s, l in real if s > l)
    else:
        print("real-shaped   skipped: no Amazon databases with step counts "
              "(provision/amazon_to_db.py, then provision/check_amazon.py --steps-jaxrun)")

    db = synth.generate(201, "tiny")
    pkg = gen.build(db)
    steps = jx.run(pkg)[1]
    tight = dict(pkg, limits={"steps": steps // 10})
    try:
        jx.run(tight)
        stand = "RAN"
    except jx.Err as e:
        stand = "stops: %s/%s" % (e.cat, e.code)
    rc, msg = run_real(jaxrun, tight)
    real = "stops: " + ("RESOURCE_ERROR/STEPS" if rc and "STEPS" in msg else "WRONG " + msg[:40])
    ok = stand.endswith("STEPS") and real.endswith("STEPS")
    failed += not ok
    print("loud failure  limit %d for %d measured steps: stand-in %s; jaxrun %s" % (steps // 10, steps, stand, real))

    saved = cost.MODEL
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(dict(m, program_sha256="0" * 64), f)
    cost.MODEL = f.name
    cost._cached.clear()
    try:
        cost.limit(db)
        outcome = "GAVE A LIMIT"
        failed += 1
    except RuntimeError as e:
        outcome = "refuses: " + str(e)[:48]
    finally:
        cost.MODEL = saved
        cost._cached.clear()
        os.unlink(f.name)
    print("stale model   %s" % outcome)

    print("\n%s" % ("all checks passed" if not failed else "%d check(s) failed" % failed))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
