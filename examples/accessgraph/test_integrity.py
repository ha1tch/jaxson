# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Tests for the data-integrity gate (integrity.py), against exact ground truth.

  SHAXONRUN=PATH python3 test_integrity.py [N_TRIALS]        (default 60)

1. clean      generated databases pass with no finding
2. injected   random defects, 1 to 6 per database, with the findings known in
              advance; the gate's findings must equal them exactly (no missed
              defect, no extra finding)
3. mutants    each rule of the gate removed in turn must make that test fail,
              so every rule is shown to be doing something
4. depth      with limits, nothing is reported at the true bounds, and below
              them exactly the users, roles and resources whose chain is longer
              than the limit, as computed independently here

A defect that overwrites an element is applied to distinct elements; a
duplicate is made by appending a copy, so it breaks nothing else.
"""
import copy
import random
import sys

import integrity
import oracle
import synth

# Defect kinds. "field" kinds overwrite one value; "dup" kinds append a copy.
FIELD = {
    "member.member": ("members", "member"), "member.group": ("members", "group"),
    "resource.parent": ("resources", "parent"),
    "grant.subject": ("grants", "subject"), "grant.role": ("grants", "role"),
    "grant.scope": ("grants", "scope"), "grant.effect": ("grants", "effect"),
    "deleg.from": ("delegations", "from"), "deleg.to": ("delegations", "to"),
    "deleg.role": ("delegations", "role"), "deleg.scope": ("delegations", "scope"),
    "request.user": ("requests", "user"), "request.resource": ("requests", "resource"),
}
DUP = {"dup.user": "users", "dup.group": "groups", "dup.role": "roles", "dup.resource": "resources",
       "dup.grant": "grants", "dup.deleg": "delegations", "dup.request": "requests"}
WIDTH = {"grant.id": "grants", "deleg.id": "delegations"}
KINDS = list(FIELD) + ["role.inherits"] + list(DUP) + ["clash"] + list(WIDTH)
UNIQUE_ID = {"users": "USER_UNIQUE", "groups": "GROUP_UNIQUE", "roles": "ROLE_UNIQUE",
             "resources": "RESOURCE_UNIQUE", "grants": "GRANT_UNIQUE",
             "delegations": "DELEGATION_UNIQUE", "requests": "REQUEST_UNIQUE"}
# the rule of the gate that each kind exercises (for the mutation check)
RULE = {"member.member": "Member.member", "member.group": "Member.group", "role.inherits": "Role.inherits",
        "resource.parent": "Resource.parent", "grant.subject": "Grant.subject", "grant.role": "Grant.role",
        "grant.scope": "Grant.scope", "grant.effect": "Grant.effect", "deleg.from": "Deleg.from",
        "deleg.to": "Deleg.to", "deleg.role": "Deleg.role", "deleg.scope": "Deleg.scope",
        "request.user": "Request.user", "request.resource": "Request.resource",
        "dup.user": "unique.users", "dup.group": "unique.groups", "dup.role": "unique.roles",
        "dup.resource": "unique.resources", "dup.grant": "unique.grants", "dup.deleg": "unique.delegations",
        "dup.request": "unique.requests", "clash": "unique.subjects",
        "grant.id": "Grant.id", "deleg.id": "Deleg.id"}


def inject(db, rng, kinds):
    """Apply one defect per entry of kinds; return (database, expected findings)."""
    d = copy.deepcopy(db)
    used, dupsrc, expected, subject_dups = set(), set(), set(), []
    for kind in kinds:
        if kind == "role.inherits":
            cands = [(i, j) for i, r in enumerate(d["roles"]) for j in range(len(r["inherits"]))
                     if ("roles", i, j) not in used]
            if not cands:
                continue
            i, j = rng.choice(cands)
            used.add(("roles", i, j))
            d["roles"][i]["inherits"][j] = "NOPE"
            expected.add(("SHAX_DANGLING_REFERENCE", ("input", "roles", i, "inherits", j)))
        elif kind in FIELD:
            arr, fld = FIELD[kind]
            i = rng.choice([x for x in range(len(d[arr])) if (arr, x) not in used])
            used.add((arr, i))
            d[arr][i][fld] = "maybe" if kind == "grant.effect" else "NOPE"
            if kind == "grant.effect":
                expected.add((None, ("input", arr, i)))
            elif kind == "resource.parent":
                expected.add((None, ("input", arr, i, fld)))
            else:
                expected.add(("SHAX_DANGLING_REFERENCE", ("input", arr, i, fld)))
        elif kind in DUP:
            arr = DUP[kind]
            src = rng.choice([x for x in range(len(d[arr])) if (arr, x) not in used])   # an untouched element
            dupsrc.add((arr, src))
            d[arr].append(copy.deepcopy(d[arr][src]))
            expected.add((UNIQUE_ID[arr], ("input", arr, len(d[arr]) - 1)))
            if arr in ("users", "groups"):
                subject_dups.append((arr, len(d[arr]) - 1))
        elif kind in WIDTH:
            arr = WIDTH[kind]
            i = rng.choice([x for x in range(len(db[arr])) if (arr, x) not in used and (arr, x) not in dupsrc])
            used.add((arr, i))
            d[arr][i]["id"] = d[arr][i]["id"][0] + str(7000 + i)       # a short id, repeated nowhere
            expected.add((None, ("input", arr, i)))
        elif kind == "clash":
            d["groups"].append(rng.choice(d["users"]))
            subject_dups.append(("clash", len(d["groups"]) - 1))
    for what, i in subject_dups:   # positions in the users-then-groups list, known only now
        pos = i if what == "users" else len(d["users"]) + i
        expected.add(("SUBJECT_UNIQUE", ("state", "subjects", pos)))
    return d, expected


def truth(db, g, r, dd):
    ix = oracle.build_index(db)
    inh = {x["id"]: x["inherits"] for x in db["roles"]}
    t = set()
    for i, u in enumerate(db["users"]):
        if oracle._bfs_depth(u, ix["up"]) > g:
            t.add(("GROUP_NESTING_EXCEEDED", ("input", "users", i)))
    for i, x in enumerate(db["roles"]):
        if oracle._bfs_depth(x["id"], inh) > r:
            t.add(("ROLE_DEPTH_EXCEEDED", ("input", "roles", i)))
    for i, x in enumerate(db["resources"]):
        if len(oracle.anc_or_self(ix["parent"], x["id"])) - 1 > dd:
            t.add(("RESOURCE_DEPTH_EXCEEDED", ("input", "resources", i)))
    return t


def main(argv):
    n = int(argv[1]) if len(argv) > 1 else 60
    sizes = ["tiny", "small", "small", "small"]
    failed = 0

    # 1. clean
    bad = 0
    for seed in range(12):
        for size in ("tiny", "small", "medium"):
            if integrity.check_db(synth.generate(seed, size)):
                bad += 1
    print("clean      36 generated databases: %d with a finding" % bad)
    failed += bad

    # 2. injected defects, exact
    rng = random.Random(1)
    mismatch, covered, total = 0, set(), 0
    trials = []
    for t in range(n):
        db = synth.generate(100 + t, sizes[t % 4])
        kinds = [rng.choice(KINDS) for _ in range(rng.randint(1, 6))]
        mutated, expected = inject(db, rng, kinds)
        got = set(integrity.check_db(mutated))
        trials.append((mutated, expected))
        covered |= set(kinds)
        total += len(expected)
        if got != expected:
            mismatch += 1
            print("  MISMATCH trial %d %s: missed %s, extra %s" % (t, kinds, sorted(expected - got, key=str)[:3], sorted(got - expected, key=str)[:3]))
    print("injected   %d databases, %d defects planted, %d kinds of %d covered: %d with findings unequal to the plan"
          % (n, total, len(covered), len(KINDS), mismatch))
    failed += mismatch

    # 3. mutants: remove each rule and require the plan to be missed
    rules = sorted(set(RULE.values()))
    survivors = []
    for rule in rules:
        kinds = [k for k, v in RULE.items() if v == rule]
        rng2 = random.Random(7)
        killed = False
        for t in range(6):
            db = synth.generate(300 + t, "small")
            mutated, expected = inject(db, rng2, [rng2.choice(kinds)])
            if set(integrity.check_db(mutated)) != expected:
                failed += 1                       # the full gate must still match
                print("  full gate disagrees with its plan under", rule)
            if set(integrity.check_db(mutated, skip=(rule,))) != expected:
                killed = True
        if not killed:
            survivors.append(rule)
    print("mutants    %d rules removed one at a time: %d killed, survivors %s" % (len(rules), len(rules) - len(survivors), survivors))
    failed += len(survivors)

    # 4. depth
    bad = checked = 0
    for size in ("tiny", "small", "medium"):
        for seed in range(3):
            db = synth.generate(seed, size)
            b = oracle.bounds(db)
            top = (b["group_rounds"], b["role_rounds"], b["res_depth"])
            for drop in (0, 1, 2):
                lim = tuple(max(x - drop, 0) for x in top)
                got = set(integrity.check_db(db, limits=lim))
                checked += 1
                if got != truth(db, *lim):
                    bad += 1
                    print("  depth mismatch", size, seed, lim)
    print("depth      %d databases x limits at, one below and two below the true bounds: %d unequal to the independent truth" % (checked, bad))
    failed += bad

    print("\n%s" % ("all checks passed" if not failed else "%d check(s) failed" % failed))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
