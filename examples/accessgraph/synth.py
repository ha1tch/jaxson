# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Seeded synthetic access-graph database for the accessgraph example.

Pure fact generation. Nothing here derives a permission: that is the job of
oracle.py (reference) and of the generated Jaxson package. The generator
deliberately plants the awkward cases: nested groups, cyclic and self
membership, deny grants below allow grants, delegations that are invalid
because the delegator lacks the role, and delegation chains longer than
MAX_HOPS.

Usage:  python3 synth.py SEED [tiny|small|medium|large|xlarge] > db.json
"""
import json
import random
import sys

ACTIONS = ["read", "write", "delete", "approve", "admin"]
from policy import MAX_HOPS, RES_DEPTH as MAX_RES_DEPTH, MAX_IDS, grant_id, deleg_id  # policy (policy.py)
import oracle  # noqa: E402

SEED_STRIDE = 1000003  # distance between the draws made for one seed

PRESETS = {
    "tiny":   dict(users=12,  groups=5,  roles=6,  resources=14,  grants=16,
                   denies=4,  delegations=8,   chains=2,  requests=40),
    "small":  dict(users=60,  groups=14, roles=10, resources=60,  grants=120,
                   denies=18, delegations=40,  chains=6,  requests=200),
    "medium": dict(users=200, groups=36, roles=16, resources=200, grants=450,
                   denies=60, delegations=140, chains=16, requests=400),
    "large":  dict(users=1000, groups=180, roles=16, resources=1000, grants=2250,
                   denies=300, delegations=700, chains=80, requests=2000),
    "xlarge": dict(users=5000, groups=900, roles=16, resources=5000, grants=8000,
                   denies=900, delegations=2800, chains=320, requests=10000),
}

# Ids are fixed width (g%03d, u%04d, ...; grants and delegations take
# policy.ID_WIDTH digits, G000001). A count past the width would make an id
# that sorts wrongly as a string, which breaks the meaning of the chain
# tie-break, so the generator refuses instead.
ID_LIMITS = dict(users=9999, groups=999, roles=99, resources=9999, grants=MAX_IDS,
                 delegations=MAX_IDS)


def _check_width(what, n):
    if n > ID_LIMITS[what]:
        raise ValueError("%d %s exceed the fixed id width (limit %d)" % (n, what, ID_LIMITS[what]))


def _draw(seed, size, cycles):
    p = PRESETS[size]
    for what, n in (("users", p["users"]), ("groups", p["groups"]), ("roles", p["roles"]),
                    ("resources", p["resources"]), ("grants", p["grants"] + p["denies"])):
        _check_width(what, n)
    rng = random.Random(seed)

    users = ["u%04d" % i for i in range(p["users"])]
    groups = ["g%03d" % i for i in range(p["groups"])]
    role_ids = ["r%02d" % i for i in range(p["roles"])]

    # roles: acyclic inheritance (a role inherits only from lower-numbered ones)
    roles = []
    for i, rid in enumerate(role_ids):
        perms = sorted(rng.sample(ACTIONS, rng.choice([1, 1, 2, 2, 3])))
        k = min(i, rng.choice([0, 0, 1, 1, 2]))
        inherits = sorted(rng.sample(role_ids[:i], k)) if k else []
        roles.append({"id": rid, "perms": perms, "inherits": inherits})

    # resources: a tree, bounded depth, biased towards deep chains
    resources = [{"id": "res0000", "parent": None}]
    depth = {"res0000": 0}
    for i in range(1, p["resources"]):
        rid = "res%04d" % i
        pool = [r["id"] for r in resources if depth[r["id"]] < MAX_RES_DEPTH]
        recent = pool[-10:]
        parent = rng.choice(recent if rng.random() < 0.5 else pool)
        resources.append({"id": rid, "parent": parent})
        depth[rid] = depth[parent] + 1
    children = {}
    for r in resources:
        if r["parent"] is not None:
            children.setdefault(r["parent"], []).append(r["id"])

    def desc_or_self(res):
        out, stack = [], [res]
        while stack:
            x = stack.pop()
            out.append(x)
            stack.extend(children.get(x, []))
        return sorted(out)

    # membership: groups nest into lower-numbered groups; users join 0-3 groups
    members = []
    for j in range(1, len(groups)):
        for parent in sorted(rng.sample(groups[:j], min(j, rng.choice([0, 1, 1, 2])))):
            members.append({"member": groups[j], "group": parent})
    for u in users:
        if rng.random() < 0.1:
            continue
        for g in sorted(rng.sample(groups, rng.choice([1, 1, 2, 3]))):
            members.append({"member": u, "group": g})
    if cycles:
        gg = [m for m in members if m["member"] in set(groups)]
        if gg:
            m = rng.choice(gg)
            members.append({"member": m["group"], "group": m["member"]})   # 2-cycle
        x = rng.choice(groups)
        members.append({"member": x, "group": x})                          # self loop

    direct_users = {}
    for m in members:
        if m["member"] in set(users):
            direct_users.setdefault(m["group"], []).append(m["member"])

    # grants (allow and deny share one id space)
    raw = []
    for effect, n in (("allow", p["grants"]), ("deny", p["denies"])):
        for _ in range(n):
            subject = rng.choice(users) if rng.random() < 0.5 else rng.choice(groups)
            raw.append({"subject": subject, "role": rng.choice(role_ids),
                        "scope": rng.choice(resources)["id"], "effect": effect})
    rng.shuffle(raw)
    grants = [dict(id=grant_id(i + 1), **g) for i, g in enumerate(raw)]
    allows = [g for g in grants if g["effect"] == "allow"]

    def holder_of(g):
        if g["subject"] in set(users):
            return g["subject"]
        pool = direct_users.get(g["subject"])
        return rng.choice(pool) if pool else None

    # delegations: explicit chains (some longer than MAX_HOPS), then singles
    raw_d = []
    for _ in range(p["chains"]):
        g = rng.choice(allows)
        cur = holder_of(g)
        if cur is None:
            continue
        scope, seen = g["scope"], {cur}
        for _ in range(rng.randint(2, MAX_HOPS + 2)):
            to = rng.choice([u for u in users if u not in seen])
            seen.add(to)
            scope = rng.choice(desc_or_self(scope))
            raw_d.append({"from": cur, "to": to, "role": g["role"], "scope": scope})
            cur = to
    for _ in range(p["delegations"]):
        if rng.random() < 0.6:
            g = rng.choice(allows)
            frm = holder_of(g) or rng.choice(users)
            role, scope = g["role"], rng.choice(desc_or_self(g["scope"]))
        else:
            frm, role, scope = rng.choice(users), rng.choice(role_ids), rng.choice(resources)["id"]
        raw_d.append({"from": frm, "to": rng.choice([u for u in users if u != frm]),
                      "role": role, "scope": scope})
    rng.shuffle(raw_d)
    _check_width("delegations", len(raw_d))
    delegations = [dict(id=deleg_id(i + 1), **d) for i, d in enumerate(raw_d)]

    # requests: some anchored on grants / delegations, the rest uniform
    requests = []
    for i in range(p["requests"]):
        roll = rng.random()
        if roll < 0.3 and grants:
            g = rng.choice(grants)
            user = holder_of(g) or rng.choice(users)
            action = rng.choice(roles[role_ids.index(g["role"])]["perms"])
            res = rng.choice(desc_or_self(g["scope"]))
        elif roll < 0.5 and delegations:
            d = rng.choice(delegations)
            user = d["to"]
            action = rng.choice(roles[role_ids.index(d["role"])]["perms"])
            res = rng.choice(desc_or_self(d["scope"]))
        else:
            user, action = rng.choice(users), rng.choice(ACTIONS)
            res = rng.choice(resources)["id"]
        requests.append({"id": "q%04d" % (i + 1), "user": user,
                         "action": action, "resource": res})

    return {"meta": {"seed": seed, "size": size, "cycles": cycles,
                     "max_hops": MAX_HOPS},
            "users": users, "groups": groups, "roles": roles,
            "resources": resources, "members": members, "grants": grants,
            "delegations": delegations, "requests": requests}


def generate(seed, size="small", cycles=True):
    """A database for the seed that is within the policy limits (policy.py).

    The first draw is the one the seed names. If its nesting is deeper than the
    limits it is drawn again from a derived seed and meta records the draw, so
    a seed whose first draw fits is unchanged."""
    for draw in range(50):
        db = _draw(seed + draw * SEED_STRIDE, size, cycles)
        if not oracle.over_limits(db):
            db["meta"]["seed"] = seed
            if draw:
                db["meta"]["draw"] = draw
            return db
    raise ValueError("no draw of seed %d fits the policy limits" % seed)


def main(argv):
    if len(argv) < 2:
        sys.stderr.write(__doc__)
        return 2
    size = argv[2] if len(argv) > 2 else "small"
    json.dump(generate(int(argv[1]), size), sys.stdout, indent=1, sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
