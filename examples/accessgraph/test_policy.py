# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Tests for the policy limits (policy.py).

  python3 test_policy.py --jaxrun PATH

1. one program   the generated program is identical for every database
2. generator     generated databases are within the limits, at every preset
3. boundary      for each limit, a database exactly at it is accepted and
                 answered the same as the reference, and one past it is refused:
                 by the reference (LimitExceeded), by the generator, and by the
                 program itself (ASSERTION_FAILED) on both the stand-in and the
                 real runner
4. truncation    the same too-deep database, built without the program's
                 assertion, is answered wrongly from a truncated closure; that
                 is what the assertion prevents
5. ids           a grant or delegation id of another width than policy.ID_WIDTH
                 is refused by the reference and the generator; and a database
                 with more than 9,999 grants, past the old four-digit cap, is
                 answered the same by the reference and both runtimes
"""
import copy
import json
import os
import subprocess
import sys
import tempfile

import gen
import jx
import oracle
import policy
import synth

DIMENSIONS = [("group_rounds", policy.GROUP_ROUNDS), ("role_rounds", policy.ROLE_ROUNDS),
              ("res_depth", policy.RES_DEPTH)]


def deep(db, kind, depth):
    """db plus a new user whose answer depends on a chain exactly `depth` deep."""
    d = copy.deepcopy(db)
    root = d["resources"][0]["id"]
    role0 = d["roles"][0]
    req = {"id": "q9999", "user": "ux", "action": role0["perms"][0], "resource": root}
    d["users"].append("ux")
    if kind == "group_rounds":
        names = ["gx%d" % i for i in range(depth)]
        d["groups"] += names
        d["members"].append({"member": "ux", "group": names[0]})
        d["members"] += [{"member": names[i], "group": names[i + 1]} for i in range(depth - 1)]
        d["grants"].append({"id": policy.grant_id(9001), "subject": names[-1], "role": role0["id"], "scope": root, "effect": "allow"})
    elif kind == "role_rounds":
        ids = ["rx%d" % i for i in range(depth + 1)]
        d["roles"].append({"id": ids[0], "perms": ["read"], "inherits": []})
        d["roles"] += [{"id": ids[i], "perms": [], "inherits": [ids[i - 1]]} for i in range(1, depth + 1)]
        d["grants"].append({"id": policy.grant_id(9001), "subject": "ux", "role": ids[-1], "scope": root, "effect": "allow"})
        req["action"] = "read"
    else:
        ids = ["rsx%d" % i for i in range(depth)]
        d["resources"] += [{"id": ids[i], "parent": root if i == 0 else ids[i - 1]} for i in range(depth)]
        d["grants"].append({"id": policy.grant_id(9001), "subject": "ux", "role": role0["id"], "scope": root, "effect": "allow"})
        req["resource"] = ids[-1]
    d["requests"] = [req]
    return d


def run_real(jaxrun, pkg):
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(pkg, f)
    try:
        r = subprocess.run([jaxrun, f.name], capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    if r.returncode:
        return "refused" if "ASSERTION_FAILED" in r.stdout + r.stderr else "error: " + (r.stdout + r.stderr)[:120]
    return json.loads(r.stdout)


def run_standin(pkg):
    try:
        return jx.run(pkg)[0]
    except jx.Err as e:
        return "refused" if e.code == "ASSERTION_FAILED" else "error: %s" % e.code


def main(argv):
    if "--jaxrun" not in argv:
        sys.stderr.write(__doc__)
        return 2
    jaxrun = argv[argv.index("--jaxrun") + 1]
    failed = 0
    base = synth.generate(1, "tiny")

    # 1. one program
    programs = {json.dumps(gen.build(synth.generate(s, size))["program"], sort_keys=True)
                for size in ("tiny", "small", "medium") for s in range(8)}
    print("one program   24 databases of three sizes: %d distinct program(s)" % len(programs))
    failed += len(programs) != 1

    # 2. generator within limits
    bad = sum(1 for size, n in (("tiny", 100), ("small", 50), ("medium", 20), ("large", 4))
              for s in range(n) if oracle.over_limits(synth.generate(s, size)))
    print("generator     174 databases over four presets: %d outside the limits" % bad)
    failed += bad

    # 3 and 4. the boundary of each limit
    print("boundary      %-13s %-9s %-10s %-9s %-9s %-9s %s" % (
        "limit", "depth", "reference", "generator", "stand-in", "jaxrun", "truncated, no assertion"))
    for kind, limit in DIMENSIONS:
        for depth in (limit, limit + 1):
            db = deep(base, kind, depth)
            inside = depth <= limit
            row = []
            try:
                want = oracle.evaluate(db)
                row.append("answers" if inside else "ACCEPTED")
                if not inside:
                    failed += 1
            except oracle.LimitExceeded:
                want = oracle.evaluate(db, enforce=False)
                row.append("refuses" if not inside else "REFUSED")
                if inside:
                    failed += 1
            try:
                pkg = gen.build(db)
                row.append("builds" if inside else "BUILT")
                if not inside:
                    failed += 1
            except oracle.LimitExceeded:
                row.append("refuses" if not inside else "REFUSED")
                pkg = gen.build(db, rounds=dict(policy.LIMITS))     # past the check made at build time
                if inside:
                    failed += 1
            got = (run_standin(pkg), run_real(jaxrun, pkg))
            for g in got:
                ok = (g == want) if inside else (g == "refused")
                row.append(("answers" if inside else "refuses") if ok else "WRONG: %s" % str(g)[:30])
                failed += not ok
            silent = gen.build(db, rounds=dict(policy.LIMITS), detect=False)
            s_real = run_real(jaxrun, silent)
            differs = s_real != want
            row.append("answers wrongly" if differs else "answers the same")
            if not inside and not differs:
                failed += 1                   # the too-deep case must show truncation
            print("              %-13s %-9d %-10s %-9s %-9s %-9s %s" % ((kind, depth) + tuple(row)))
    # 5. ids
    bad4 = copy.deepcopy(base)
    bad4["grants"][0]["id"] = "G0001"
    bad7 = copy.deepcopy(base)
    bad7["delegations"][0]["id"] = policy.deleg_id(1) + "0"
    for name, db in (("four-digit grant id", bad4), ("seven-digit delegation id", bad7)):
        try:
            oracle.evaluate(db)
            outcome = ["ACCEPTED by the reference"]
        except oracle.LimitExceeded as e:
            outcome = ["refused by the reference" if "id_width" in str(e) else "WRONG: " + str(e)[:40]]
        try:
            gen.build(db)
            outcome.append("BUILT")
        except oracle.LimitExceeded:
            outcome.append("refused by the generator")
        ok = outcome == ["refused by the reference", "refused by the generator"]
        failed += not ok
        print("ids           %-26s %s" % (name, ", ".join(outcome)))
    synth.PRESETS["wideids"] = dict(users=60, groups=14, roles=10, resources=60, grants=10050,
                                    denies=18, delegations=40, chains=6, requests=200)
    big = synth.generate(2, "wideids")
    del synth.PRESETS["wideids"]
    want = oracle.evaluate(big)
    pkg = gen.build(big)
    same = run_standin(pkg) == want and run_real(jaxrun, pkg) == want
    print("ids           %d grants, largest id %s: reference, stand-in and real runner %s" % (
        len(big["grants"]), max(g["id"] for g in big["grants"]), "agree" if same else "DISAGREE"))
    failed += not same
    print("\n%s" % ("all checks passed" if not failed else "%d check(s) failed" % failed))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
