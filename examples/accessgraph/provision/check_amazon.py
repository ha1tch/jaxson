#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Check the Amazon-derived databases and run the in-tree parts of the T-39 comparison.

  python3 check_amazon.py [--shaxonrun PATH] [--run-program --jaxrun PATH] [--steps-jaxrun PATH]

Needs the databases from amazon_to_db.py (which needs the data from fetch.sh).
Without options it checks the databases against the policy limits and
references, shows that the nested variant has the same reach for every user and
the same decisions as the flat one, and prints the declared step limit the cost
model derives before any run.

  --shaxonrun PATH    also run the integrity gate, without and with the depth check
  --run-program       run the generated program on jaxrun at the declared limit and
                      report whether it fits (needs --jaxrun PATH)
  --steps-jaxrun PATH a jaxrun that prints its step count on stderr as "steps: N"
                      (the repository's jaxrun does not; a scratch build with one
                      line added does): run at a raised limit, compare with the
                      derived prediction, and check the answers against the reference

The engines outside the tree (the Go ports, Soufflé, CozoDB) are compared by the
trial scripts that go with them.
"""
import json
import os
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))
import cost  # noqa: E402
import gen  # noqa: E402
import integrity  # noqa: E402
import oracle  # noqa: E402
import policy  # noqa: E402

DATA = os.path.join(HERE, "..", "data", "amazon-db")
RAISED = 2000000000


def arg(argv, name):
    return argv[argv.index(name) + 1] if name in argv else None


def reach(db):
    up = {}
    for m in db["members"]:
        up.setdefault(m["member"], []).append(m["group"])
    return {u: cost._reach(up, u) for u in db["users"]}


def run_package(runner, pkg):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(pkg, f)
    try:
        t = time.time()
        r = subprocess.run([runner, f.name], capture_output=True, text=True)
        return r, time.time() - t
    finally:
        os.unlink(f.name)


def gate(runner, db, limits):
    with tempfile.TemporaryDirectory() as d:
        json.dump(integrity.package(limits), open(d + "/p.json", "w"))
        json.dump(db, open(d + "/i.json", "w"))
        t = time.time()
        r = subprocess.run([runner, "-steps", d + "/p.json", d + "/i.json"], capture_output=True, text=True)
        sec = time.time() - t
    if r.returncode:
        return "failed: " + (r.stdout + r.stderr).strip()[:120]
    n = len(json.loads(r.stdout)["report"]["violations"])
    return "%d findings, %s, %.0f s" % (n, r.stderr.strip().splitlines()[-1], sec)


def main(argv):
    if not all(os.path.exists(os.path.join(DATA, v + ".json")) for v in ("flat", "nested")):
        sys.stderr.write("no databases in %s: run fetch.sh and amazon_to_db.py first\n" % DATA)
        return 2
    dbs = {v: json.load(open(os.path.join(DATA, v + ".json"))) for v in ("flat", "nested")}
    failed = 0
    reaches = {}
    for v, db in dbs.items():
        over = oracle.over_limits(db)
        oracle.check_refs(db)
        reaches[v] = reach(db)
        print("%-7s %d users, %d groups, %d members, %d grants, %d delegations, %d requests" % (
            v, len(db["users"]), len(db["groups"]), len(db["members"]), len(db["grants"]),
            len(db["delegations"]), len(db["requests"])))
        print("        within the policy limits: %s; true bounds %s; references clean" % (
            "yes" if not over else "NO " + str(over), oracle.bounds(db)))
        failed += bool(over)
    diff = sum(1 for u in reaches["flat"] if reaches["flat"][u] != reaches["nested"][u])
    print("reach differs between flat and nested for %d of %d users" % (diff, len(reaches["flat"])))
    failed += diff != 0
    want = {v: oracle.evaluate(db) for v, db in dbs.items()}
    same = sum(1 for q in want["flat"] if want["flat"][q] == want["nested"][q])
    allowed = sum(1 for r in want["flat"].values() if r["allow"])
    chained = sum(1 for r in want["flat"].values() if r["via"] and ">D" in r["via"])
    print("reference decisions: %d of %d identical between the variants; %d allowed, %d denied, %d through a delegation chain"
          % (same, len(want["flat"]), allowed, len(want["flat"]) - allowed, chained))
    failed += same != len(want["flat"])
    for v, db in dbs.items():
        print("%-7s declared step limit derived before any run: %d (predicted %.0f)" % (v, cost.limit(db), cost.predict(db)))

    runner = arg(argv, "--shaxonrun") or os.environ.get("SHAXONRUN")
    if "--shaxonrun" in argv or os.environ.get("SHAXONRUN"):
        for v, db in dbs.items():
            print("gate %-7s references and ids only: %s" % (v, gate(runner, db, None)))
            print("gate %-7s with the depth check:    %s" % (v, gate(runner, db, (policy.GROUP_ROUNDS, policy.ROLE_ROUNDS, policy.RES_DEPTH))))

    jaxrun = arg(argv, "--jaxrun")
    if "--run-program" in argv and jaxrun:
        for v, db in dbs.items():
            r, sec = run_package(jaxrun, gen.build(db))
            ok = r.returncode == 0 and json.loads(r.stdout) == want[v]
            print("program %-7s at the declared limit: %s (%.0f s)" % (
                v, "fits, answers equal the reference" if ok else "stopped: " + (r.stdout + r.stderr).strip()[:70], sec))
            failed += not ok

    counter = arg(argv, "--steps-jaxrun")
    if counter:
        for v, db in dbs.items():
            r, sec = run_package(counter, gen.build(db, steps=RAISED))
            steps = int(r.stderr.strip().splitlines()[-1].split()[-1])
            equal = json.loads(r.stdout) == want[v]
            p, lim = cost.predict(db), cost.limit(db)
            json.dump({"variant": v, "steps": steps, "predicted": round(p), "declared_limit": lim,
                       "equal_to_reference": equal}, open(os.path.join(DATA, "steps-%s.json" % v), "w"))
            print("steps   %-7s %d measured, %.0f predicted (%.2f times), declared %d (%s); answers %s the reference" % (
                v, steps, p, steps / p, lim, "fits" if steps <= lim else "EXCEEDED", "equal" if equal else "DIFFER FROM"))
            failed += (not equal) or steps > lim
    print("\n%s" % ("all checks passed" if not failed else "%d check(s) failed" % failed))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
