#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Adapt the Amazon Access Samples to accessgraph databases (T-39).

  python3 amazon_to_db.py [DERIVED_DIR] [OUT_DIR]

DERIVED_DIR is the `derived` directory that fetch.sh writes (default
../data/amazon-access/derived); OUT_DIR defaults to ../data/amazon-db. It writes
the same database in two variants and a record of what came from the data and
what was added:

  flat.json    membership as in the data: people in permission groups
  nested.json  the same entailments through nesting that came from the data:
               an edge from group A to group B where the members of A are
               strictly contained in the members of B (groups of at least 20
               members) and no group lies between them, and each user's direct
               membership in B dropped where it follows from another group the
               user is in. The memberships in the data are already
               closed, so adding the edges alone would change nothing (the true
               depth stays 1); dropping the implied ones makes the closure do
               the work, and every decision must equal the flat variant's
  record.json  counts of everything taken from the data and everything added

From the data, unchanged: the people, the permission groups, who is in which
group, and each direct grant of a system or host to a person. The data has no
roles, resource tree, group grants, denies or delegations, so those are
completed from the generator's own distributions (synth.py), at proportions
scaled to this size, with a fixed seed. Resource ids and group ids are renamed
to fit the example's id formats; the data's own column names are not kept.
"""
import json
import os
import random
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))
import policy  # noqa: E402
import synth  # noqa: E402

SEED = 39
FOLDERS_L1, FOLDERS_L2 = 24, 24          # root, 24 areas, 24 zones each: leaves at depth 3
GROUP_GRANTS, DENIES = 12000, 8000       # synthetic, from the generator's mix of subjects and scopes
CHAINS, SINGLES, REQUESTS = 2300, 20200, 10000   # the generator's xlarge ratios scaled to 36,063 users
MIN_NESTED_MEMBERS = 20


def read_tsv(path):
    with open(path) as f:
        return [line.rstrip("\n").split("\t") for line in f]


def main(argv):
    derived = argv[1] if len(argv) > 1 else os.path.join(HERE, "..", "data", "amazon-access", "derived")
    out_dir = argv[2] if len(argv) > 2 else os.path.join(HERE, "..", "data", "amazon-db")
    os.makedirs(out_dir, exist_ok=True)
    rng = random.Random(SEED)

    attrs = read_tsv(os.path.join(derived, "person_attrs.tsv"))
    pid_col = attrs[0].index("PERSON_ID")
    people = sorted(int(r[pid_col]) for r in attrs[1:])
    edges = read_tsv(os.path.join(derived, "edges.tsv"))[1:]
    perm = [(int(p), c) for p, k, c, v in edges if k == "permgroup"]
    direct = [(int(p), c) for p, k, c, v in edges if k in ("system", "host")]

    users = ["u%05d" % p for p in people]
    uid = {p: "u%05d" % p for p in people}
    group_cols = sorted({c for _, c in perm})
    groups = ["g%05d" % i for i in range(len(group_cols))]
    gid = dict(zip(group_cols, groups))
    members = [{"member": uid[p], "group": gid[c]} for p, c in perm]

    # roles: the generator's construction, acyclic inheritance
    role_ids = ["r%02d" % i for i in range(16)]
    roles = []
    for i, rid in enumerate(role_ids):
        perms = sorted(rng.sample(synth.ACTIONS, rng.choice([1, 1, 2, 2, 3])))
        k = min(i, rng.choice([0, 0, 1, 1, 2]))
        roles.append({"id": rid, "perms": perms, "inherits": sorted(rng.sample(role_ids[:i], k)) if k else []})
    role_perms = {r["id"]: r["perms"] for r in roles}

    # resources: the data's systems and hosts are the leaves; the tree above them is added
    res_cols = sorted({c for _, c in direct})
    rng.shuffle(res_cols)
    resources = [{"id": "res00000", "parent": None}]
    nxt, zones = 1, []
    for a in range(FOLDERS_L1):
        aid = "res%05d" % nxt
        nxt += 1
        resources.append({"id": aid, "parent": "res00000"})
        for z in range(FOLDERS_L2):
            zid = "res%05d" % nxt
            nxt += 1
            resources.append({"id": zid, "parent": aid})
            zones.append(zid)
    n_folders = len(resources)
    rid = {}
    for i, c in enumerate(res_cols):
        rid[c] = "res%05d" % nxt
        nxt += 1
        resources.append({"id": rid[c], "parent": zones[i % len(zones)]})
    all_res = [r["id"] for r in resources]
    children = {}
    for r in resources:
        if r["parent"]:
            children.setdefault(r["parent"], []).append(r["id"])

    def desc_or_self(res):
        out, stack = [], [res]
        while stack:
            x = stack.pop()
            out.append(x)
            stack.extend(children.get(x, ()))
        return out

    # grants: every direct grant of the data stays direct and allowing; the role is added
    raw = [{"subject": uid[p], "role": rng.choice(role_ids), "scope": rid[c], "effect": "allow"} for p, c in direct]
    n_direct = len(raw)
    for _ in range(GROUP_GRANTS):
        raw.append({"subject": rng.choice(groups), "role": rng.choice(role_ids), "scope": rng.choice(all_res), "effect": "allow"})
    for _ in range(DENIES):
        subject = rng.choice(users) if rng.random() < 0.5 else rng.choice(groups)
        raw.append({"subject": subject, "role": rng.choice(role_ids), "scope": rng.choice(all_res), "effect": "deny"})
    rng.shuffle(raw)
    grants = [dict(id=policy.grant_id(i + 1), **g) for i, g in enumerate(raw)]
    allows = [g for g in grants if g["effect"] == "allow"]

    user_set = set(users)
    direct_users = {}
    for m in members:
        direct_users.setdefault(m["group"], []).append(m["member"])

    def holder_of(g):
        if g["subject"] in user_set:
            return g["subject"]
        pool = direct_users.get(g["subject"])
        return rng.choice(pool) if pool else None

    def other_than(seen):
        while True:
            u = rng.choice(users)
            if u not in seen:
                return u

    raw_d = []
    for _ in range(CHAINS):
        g = rng.choice(allows)
        cur = holder_of(g)
        if cur is None:
            continue
        scope, seen = g["scope"], {cur}
        for _ in range(rng.randint(2, policy.MAX_HOPS + 2)):
            to = other_than(seen)
            seen.add(to)
            scope = rng.choice(desc_or_self(scope))
            raw_d.append({"from": cur, "to": to, "role": g["role"], "scope": scope})
            cur = to
    for _ in range(SINGLES):
        if rng.random() < 0.6:
            g = rng.choice(allows)
            frm = holder_of(g) or rng.choice(users)
            role, scope = g["role"], rng.choice(desc_or_self(g["scope"]))
        else:
            frm, role, scope = rng.choice(users), rng.choice(role_ids), rng.choice(all_res)
        raw_d.append({"from": frm, "to": other_than({frm}), "role": role, "scope": scope})
    rng.shuffle(raw_d)
    delegations = [dict(id=policy.deleg_id(i + 1), **d) for i, d in enumerate(raw_d)]

    requests = []
    for i in range(REQUESTS):
        roll = rng.random()
        if roll < 0.3:
            g = rng.choice(grants)
            user = holder_of(g) or rng.choice(users)
            action = rng.choice(role_perms[g["role"]])
            res = rng.choice(desc_or_self(g["scope"]))
        elif roll < 0.5:
            d = rng.choice(delegations)
            user, action = d["to"], rng.choice(role_perms[d["role"]])
            res = rng.choice(desc_or_self(d["scope"]))
        else:
            user, action, res = rng.choice(users), rng.choice(synth.ACTIONS), rng.choice(all_res)
        requests.append({"id": "q%05d" % (i + 1), "user": user, "action": action, "resource": res})

    def database(name, member_list):
        return {"meta": {"seed": SEED, "size": name, "cycles": False, "max_hops": policy.MAX_HOPS,
                         "source": "Amazon Access Samples (UCI 216), adapted by provision/amazon_to_db.py"},
                "users": users, "groups": groups, "roles": roles, "resources": resources,
                "members": member_list, "grants": grants,
                "delegations": delegations, "requests": requests}

    # the nested variant: strict containment of user sets among groups of at least 20 members
    index = {u: i for i, u in enumerate(users)}
    mask, size = {}, {}
    for g, us in direct_users.items():
        m = 0
        for u in us:
            m |= 1 << index[u]
        mask[g], size[g] = m, len(set(us))
    big = sorted((g for g in mask if size[g] >= MIN_NESTED_MEMBERS), key=lambda g: size[g])
    nested = []
    for i, a in enumerate(big):
        for b in big[i + 1:]:
            if size[b] > size[a] and mask[a] & mask[b] == mask[a]:
                nested.append({"member": a, "group": b})
    up = {}
    for e in nested:
        up.setdefault(e["member"], []).append(e["group"])
    depth = {}

    def chain(g):
        if g not in depth:
            depth[g] = 1 + max((chain(p) for p in up.get(g, ())), default=0)
        return depth[g]
    longest = max((chain(g) for g in big), default=0)

    implied = {}
    for e in nested:
        implied.setdefault(e["member"], set()).add(e["group"])
    held = {}
    for m in members:
        held.setdefault(m["member"], set()).add(m["group"])
    reduced = [m for m in members
               if not any(m["group"] in implied.get(a, ()) for a in held[m["member"]] if a != m["group"])]
    dropped = len(members) - len(reduced)
    covering = [e for e in nested
                if not any(e["group"] in implied.get(c, ()) for c in implied.get(e["member"], ()) if c != e["group"])]
    flat, nest = database("amazon-flat", members), database("amazon-nested", reduced + covering)
    for name, db in (("flat", flat), ("nested", nest)):
        with open(os.path.join(out_dir, name + ".json"), "w") as f:
            json.dump(db, f)
    record = {
        "seed": SEED,
        "from_the_data": {"users": len(users), "groups_with_members": len(groups), "memberships": len(members),
                          "direct_grants": n_direct, "resources_as_leaves": len(res_cols),
                          "nested_variant_containments": len(nested),
                          "nested_variant_covering_edges": len(covering),
                          "nested_variant_memberships_dropped": dropped,
                          "longest_containment_chain_in_groups": longest},
        "added": {"roles": len(roles), "resource_folders": n_folders, "resource_tree_depth": 3,
                  "group_allow_grants": GROUP_GRANTS, "denies": DENIES,
                  "delegations": len(delegations), "delegation_chains_requested": CHAINS,
                  "requests": len(requests), "role_per_direct_grant": "chosen uniformly, seeded"},
        "totals": {"grants": len(grants), "members_flat": len(members),
                   "members_nested": len(reduced) + len(covering)},
    }
    with open(os.path.join(out_dir, "record.json"), "w") as f:
        json.dump(record, f, indent=2)
    print(json.dumps(record, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
