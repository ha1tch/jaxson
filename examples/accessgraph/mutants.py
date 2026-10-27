# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Plant faults in the generated Jaxson program and require the checker to notice.

  python3 mutants.py --jaxrun PATH [N_SEEDS]        (N_SEEDS defaults to 24)

Each mutant changes one thing about how the package is built: a delegation
chain one hop too short or too long, one closure round fewer than the database
needs for groups, roles or resource ancestry, or the deny grants dropped from
the input. A round mutant comes in two forms. Silent: the program's assertion
that its input fits is left out, so it answers from a truncated closure and the
answers must differ from the reference. Asserted: the assertion stays in, so
the program must refuse the database (ASSERTION_FAILED) and not answer. The
unmutated build is the control and must agree with the reference on every
database. Every mutant must disagree with the reference on at least one
database. Exit status is 1 if either condition fails.

A mutant detected on few databases points at a thinly exercised part of the
rules; the count per mutant is printed for that reason.
"""
import json
import os
import subprocess
import sys
import tempfile

import gen
import oracle
import policy
import synth

SIZES = ["tiny", "small", "small", "medium"]

MUTANTS = [
    ("one hop short", {"hops": policy.MAX_HOPS - 1}),
    ("one hop extra", {"hops": policy.MAX_HOPS + 1}),
    ("group round short, silent", {"bound": "group_rounds", "detect": False}),
    ("role round short, silent", {"bound": "role_rounds", "detect": False}),
    ("ancestor walk short, silent", {"bound": "res_depth", "detect": False}),
    ("group round short, asserted", {"bound": "group_rounds"}),
    ("role round short, asserted", {"bound": "role_rounds"}),
    ("ancestor walk short, asserted", {"bound": "res_depth"}),
    ("denies ignored", {"deny_off": True}),
]


def run_real(jaxrun, pkg):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(pkg, f)
    try:
        r = subprocess.run([jaxrun, f.name], capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    if r.returncode:
        raise RuntimeError("jaxrun failed: " + (r.stdout + r.stderr).strip()[:300])
    return json.loads(r.stdout)


def package(db, mutant):
    """The package for db with one mutation applied (none for the control)."""
    rounds = None
    if "bound" in mutant:
        rounds = dict(policy.LIMITS)
        key = mutant["bound"]
        rounds[key] = max(oracle.bounds(db)[key] - 1, 0)   # one fewer than this database needs
    pkg = gen.build(db, hops=mutant.get("hops", policy.MAX_HOPS), rounds=rounds,
                    detect=mutant.get("detect", True))
    if mutant.get("deny_off"):
        pkg["input"]["deny"] = {s: [] for s in pkg["input"]["deny"]}
    return pkg


def main(argv):
    args = list(argv[1:])
    if "--jaxrun" not in args:
        sys.stderr.write(__doc__)
        return 2
    i = args.index("--jaxrun")
    jaxrun = args[i + 1]
    del args[i:i + 2]
    n = int(args[0]) if args else 24

    cases = [("control (unmutated)", {})] + MUTANTS
    hits = {name: 0 for name, _ in cases}
    for seed in range(n):
        db = synth.generate(seed, SIZES[seed % len(SIZES)])
        want = oracle.evaluate(db)
        for name, mutant in cases:
            try:
                got = run_real(jaxrun, package(db, mutant))
            except RuntimeError as e:
                if "ASSERTION_FAILED" not in str(e):
                    raise
                got = None                      # the program refused its input
            if got != want:
                hits[name] += 1

    print("%-24s %s" % ("case", "databases with a mismatch, of %d" % n))
    for name, _ in cases:
        print("%-24s %d" % (name, hits[name]))
    control_clean = hits["control (unmutated)"] == 0
    all_caught = all(hits[name] > 0 for name, _ in MUTANTS)
    print("control clean: %s | every mutant detected: %s" % (control_clean, all_caught))
    return 0 if control_clean and all_caught else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
