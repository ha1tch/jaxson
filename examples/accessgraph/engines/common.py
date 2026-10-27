# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import os
import sys
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))
import oracle, synth

def facts(db):
    f = {}
    f["user"] = [(u,) for u in db["users"]]
    f["up"] = [(m["member"], m["group"]) for m in db["members"]]
    f["role"] = [(r["id"],) for r in db["roles"]]
    f["inh"] = [(r["id"], s) for r in db["roles"] for s in r["inherits"]]
    f["rperm"] = [(r["id"], a) for r in db["roles"] for a in r["perms"]]
    f["res"] = [(r["id"],) for r in db["resources"]]
    f["par"] = [(r["id"], r["parent"]) for r in db["resources"] if r["parent"]]
    f["grant"] = [(g["id"], g["subject"], g["role"], g["scope"], g["effect"]) for g in db["grants"]]
    f["deleg"] = [(d["id"], d["from"], d["to"], d["role"], d["scope"]) for d in db["delegations"]]
    f["req"] = [(q["id"], q["user"], q["action"], q["resource"]) for q in db["requests"]]
    return f

def assemble(db, supported, via, den):
    """supported: set of q; via: {q: chain}; den: {q: set(grant ids)} -> oracle-shaped result."""
    out = {}
    for q in db["requests"]:
        i = q["id"]
        d = sorted(den.get(i, ()))
        ok = i in supported and not d
        out[i] = {"allow": ok, "via": via.get(i) if ok else None, "denied_by": d}
    return out

def compare(name, run, n, sizes=("tiny", "small", "small", "medium")):
    bad = dec = 0
    first = None
    for seed in range(n):
        db = synth.generate(seed, sizes[seed % len(sizes)])
        want = oracle.evaluate(db)
        got = run(db)
        diff = [q for q in want if got.get(q) != want[q]]
        dec += len(want)
        if diff:
            bad += 1
            if first is None:
                q = diff[0]; first = (seed, q, got.get(q), want[q])
    print("%s: %d databases, %d decisions, %d databases with mismatches" % (name, n, dec, bad))
    if first: print("  first mismatch seed=%d %s: got %s want %s" % first)
    return bad
