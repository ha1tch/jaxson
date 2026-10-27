# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Compare the oracle with the generated package, run by the stand-in
interpreter and, with --jaxrun, by the real runner.

  python3 check.py [N_SEEDS] [--jaxrun PATH]
"""
import json
import os
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import gen, jx, oracle, synth  # noqa: E402

FIXTURES = os.path.join(HERE, "..", "..", "jaxson-v0.1.0-fixtures.json")


def real(jaxrun, pkg):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(pkg, f)
    try:
        r = subprocess.run([jaxrun, f.name], capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    if r.returncode != 0:
        raise RuntimeError("jaxrun failed: " + (r.stdout + r.stderr).strip()[:300])
    return json.loads(r.stdout)


def main(argv):
    args = argv[1:]
    jaxrun = None
    if "--jaxrun" in args:
        i = args.index("--jaxrun")
        jaxrun = args[i + 1]
        del args[i:i + 2]
    n = int(args[0]) if args else 12
    if os.path.exists(FIXTURES) and not jx.fixtures(FIXTURES):
        print("stand-in interpreter fails the fixtures; its results mean nothing")
        return 1
    sizes = ["tiny", "small", "small", "medium"]
    bad = decisions = worst = 0
    t0 = time.time()
    for seed in range(n):
        size = sizes[seed % len(sizes)]
        db = synth.generate(seed, size)
        pkg = gen.build(db)
        want = oracle.evaluate(db)
        outs = {"stand-in": jx.run(pkg)}
        if jaxrun:
            outs["jaxrun"] = (real(jaxrun, pkg), None)
        for who, (out, steps) in outs.items():
            if out != want:
                bad += 1
                diff = [q for q in want if out.get(q) != want[q]]
                print("MISMATCH %s seed=%d size=%s: %d/%d requests differ, first %s: got %s want %s" % (
                    who, seed, size, len(diff), len(want), diff[0], out.get(diff[0]), want[diff[0]]))
            if steps:
                worst = max(worst, steps / pkg["limits"]["steps"])
        decisions += len(want)
    print("%d databases, %d decisions per runtime, %d mismatches, %.0fs" % (n, decisions, bad, time.time() - t0))
    print("runtimes: stand-in%s; at most %.0f%% of the declared limit used (stand-in's count)" % (
        " + jaxrun" if jaxrun else "", 100 * worst))
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
