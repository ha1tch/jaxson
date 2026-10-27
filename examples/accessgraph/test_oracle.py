# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Checks on the accessgraph reference oracle and generator.

1. Hand-built adversarial cases, with answers derived from the prose rules.
2. A third, independent algorithm (naive global fixpoint) agrees with the
   oracle on many generated databases.
3. The generator is deterministic, and its output exercises every category of
   decision (so agreement on it is not vacuous).

Usage:  python3 test_oracle.py [N_SEEDS]
"""
import json
import os
import sys
from collections import Counter

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import oracle  # noqa: E402
import synth  # noqa: E402
from synth import MAX_HOPS  # noqa: E402


# ------------------------------------------------------------ hand-built cases
EXPECT = [
    # (user, action, resource, allow, via, denied_by)
    ("u0001", "read",  "res0001", True,  "G000001", []),          # via g001 -> g002, despite the membership cycle
    ("u0001", "read",  "res0002", False, None,    ["G000002"]),   # deny on a descendant overrides inherited allow
    ("u0001", "write", "res0002", True,  "G000001", []),          # the deny covers read only
    ("u0001", "write", "res0003", False, None,    []),          # res0003 is not under res0001: no support
    ("u0002", "admin", "res0003", True,  "G000003", []),
    ("u0002", "read",  "res0002", True,  "G000003", []),          # admin inherits editor inherits viewer
    ("u0003", "write", "res0003", True,  "G000003>D000002", []),    # D000001 is invalid: delegates r01 but holds r02
    ("u0003", "read",  "res0001", False, None,    []),          # D000006 invalid: delegator's scope is narrower
    ("u0005", "admin", "res0003", True,  "G000003>D000002>D000003>D000004", []),   # exactly MAX_HOPS
    ("u0006", "admin", "res0003", False, None,    []),          # one hop too many
    ("u0004", "read",  "res0002", True,  "G000001>D000007", []),    # u0001's deny is not carried to the delegate
    ("u0007", "read",  "res0002", True,  "G000004", []),
    ("u0007", "read",  "res0003", False, None,    []),
]


def hand_db():
    return {
        "users": ["u%04d" % i for i in range(1, 8)],
        "groups": ["g001", "g002"],
        "roles": [{"id": "r00", "perms": ["read"], "inherits": []},
                  {"id": "r01", "perms": ["write"], "inherits": ["r00"]},
                  {"id": "r02", "perms": ["admin"], "inherits": ["r01"]}],
        "resources": [{"id": "res0000", "parent": None},
                      {"id": "res0001", "parent": "res0000"},
                      {"id": "res0002", "parent": "res0001"},
                      {"id": "res0003", "parent": "res0000"}],
        "members": [{"member": "u0001", "group": "g001"},
                    {"member": "g001", "group": "g002"},
                    {"member": "g002", "group": "g001"}],
        "grants": [
            {"id": "G000001", "subject": "g002", "role": "r01", "scope": "res0001", "effect": "allow"},
            {"id": "G000002", "subject": "u0001", "role": "r00", "scope": "res0002", "effect": "deny"},
            {"id": "G000003", "subject": "u0002", "role": "r02", "scope": "res0000", "effect": "allow"},
            {"id": "G000004", "subject": "u0007", "role": "r00", "scope": "res0002", "effect": "allow"}],
        "delegations": [
            {"id": "D000001", "from": "u0002", "to": "u0003", "role": "r01", "scope": "res0003"},
            {"id": "D000002", "from": "u0002", "to": "u0003", "role": "r02", "scope": "res0003"},
            {"id": "D000003", "from": "u0003", "to": "u0004", "role": "r02", "scope": "res0003"},
            {"id": "D000004", "from": "u0004", "to": "u0005", "role": "r02", "scope": "res0003"},
            {"id": "D000005", "from": "u0005", "to": "u0006", "role": "r02", "scope": "res0003"},
            {"id": "D000006", "from": "u0007", "to": "u0003", "role": "r00", "scope": "res0001"},
            {"id": "D000007", "from": "u0001", "to": "u0004", "role": "r01", "scope": "res0002"}],
        "requests": [{"id": "q%04d" % (i + 1), "user": e[0], "action": e[1], "resource": e[2]}
                     for i, e in enumerate(EXPECT)],
    }


def test_hand_cases():
    db = hand_db()
    got = oracle.evaluate(db)
    for i, (u, a, r, allow, via, den) in enumerate(EXPECT):
        want = {"allow": allow, "via": via, "denied_by": den}
        have = got["q%04d" % (i + 1)]
        assert have == want, "%s %s %s: want %r have %r" % (u, a, r, want, have)
    # the hop bound is a parameter, not an accident: one more hop reaches u0006
    db["requests"] = [{"id": "x", "user": "u0006", "action": "admin", "resource": "res0003"}]
    assert oracle.evaluate(db, max_hops=MAX_HOPS + 1)["x"] == {
        "allow": True, "via": "G000003>D000002>D000003>D000004>D000005", "denied_by": []}
    assert oracle.bounds(hand_db()) == {"group_rounds": 2, "role_rounds": 2, "res_depth": 2}


# ------------------------------------------------- third algorithm: naive fixpoint
def naive(db, max_hops=MAX_HOPS):
    parent = {r["id"]: r["parent"] for r in db["resources"]}

    def anc(x):
        out = set()
        while x is not None:
            out.add(x)
            x = parent[x]
        return out

    inh = {r["id"]: set(r["inherits"]) for r in db["roles"]}
    own = {r["id"]: set(r["perms"]) for r in db["roles"]}
    clo = {r: {r} for r in inh}
    changed = True
    while changed:
        changed = False
        for r in inh:
            for p in list(clo[r]):
                new = inh[p] - clo[r]
                if new:
                    clo[r] |= new
                    changed = True
    perms = {r: set().union(*[own[x] for x in clo[r]]) for r in inh}

    ups = {}
    for m in db["members"]:
        ups.setdefault(m["member"], set()).add(m["group"])
    subj = {u: {u} for u in db["users"]}
    changed = True
    while changed:
        changed = False
        for u in subj:
            for s in list(subj[u]):
                new = ups.get(s, set()) - subj[u]
                if new:
                    subj[u] |= new
                    changed = True

    held = {u: {} for u in db["users"]}

    def better(u, key, cand):
        if key not in held[u] or cand < held[u][key]:
            held[u][key] = cand
            return True
        return False

    changed = True
    while changed:
        changed = False
        for g in db["grants"]:
            if g["effect"] == "allow":
                for u in held:
                    if g["subject"] in subj[u]:
                        changed |= better(u, (g["role"], g["scope"]), (0, g["id"]))
        for d in db["delegations"]:
            cov = anc(d["scope"])
            for (role, scope), (h, c) in list(held[d["from"]].items()):
                if role == d["role"] and scope in cov and h < max_hops:
                    changed |= better(d["to"], (role, d["scope"]), (h + 1, c + ">" + d["id"]))

    out = {}
    for q in db["requests"]:
        u, a, A = q["user"], q["action"], anc(q["resource"])
        sup = [hc for (role, scope), hc in held[u].items() if scope in A and a in perms[role]]
        den = sorted(g["id"] for g in db["grants"]
                     if g["effect"] == "deny" and g["subject"] in subj[u]
                     and g["scope"] in A and a in perms[g["role"]])
        allow = bool(sup) and not den
        out[q["id"]] = {"allow": allow, "via": min(sup)[1] if allow else None, "denied_by": den}
    return out


def test_hand_cases_against_naive():
    db = hand_db()
    assert naive(db) == oracle.evaluate(db)


def test_generated(n_seeds):
    cov, worst = Counter(), Counter()
    sizes = ["tiny"] * (n_seeds // 2) + ["small"] * (n_seeds // 2 - n_seeds // 10) \
        + ["medium"] * (n_seeds // 10)
    for seed, size in enumerate(sizes):
        db = synth.generate(seed, size)
        fast = oracle.evaluate(db, detailed=True)
        plain = {k: {f: v[f] for f in ("allow", "via", "denied_by")} for k, v in fast.items()}
        slow = naive(db)
        assert plain == slow, "oracle and naive fixpoint disagree: seed %d size %s" % (seed, size)
        wider = oracle.evaluate(db, max_hops=MAX_HOPS + 2)
        b = oracle.bounds(db)
        for k, v in b.items():
            worst[k] = max(worst[k], v)
        assert b["res_depth"] <= synth.MAX_RES_DEPTH
        for qid, r in fast.items():
            if r["allow"]:
                cov["allow_direct" if r["hops"] == 0 else "allow_delegated"] += 1
            elif r["supported"] and r["denied_by"]:
                cov["deny_overrides_allow"] += 1
            elif r["denied_by"]:
                cov["deny_without_support"] += 1
            else:
                cov["no_support"] += 1
            if wider[qid]["allow"] != r["allow"]:
                cov["flips_with_more_hops"] += 1
        if size != "tiny":
            assert any(m["member"] == m["group"] for m in db["members"]), "no self-loop planted"
    for k in ("allow_direct", "allow_delegated", "deny_overrides_allow",
              "no_support", "flips_with_more_hops"):
        assert cov[k] > 0, "generator never produced category %r" % k
    return cov, worst, len(sizes)


def test_determinism():
    a = json.dumps(synth.generate(7, "small"), sort_keys=True)
    b = json.dumps(synth.generate(7, "small"), sort_keys=True)
    c = json.dumps(synth.generate(8, "small"), sort_keys=True)
    assert a == b and a != c


def main(argv):
    n = int(argv[1]) if len(argv) > 1 else 200
    test_hand_cases()
    print("ok   hand-built cases (%d expectations, hop-bound parameter, bounds)" % len(EXPECT))
    test_hand_cases_against_naive()
    print("ok   naive fixpoint agrees on the hand-built database")
    test_determinism()
    print("ok   generator deterministic")
    cov, worst, runs = test_generated(n)
    print("ok   oracle == naive fixpoint on %d generated databases" % runs)
    total = sum(cov.values()) - cov["flips_with_more_hops"]
    for k in sorted(cov):
        print("       %-22s %6d" % (k, cov[k]))
    print("       decisions checked      %6d" % total)
    print("       worst-case bounds: %s" % dict(worst))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
