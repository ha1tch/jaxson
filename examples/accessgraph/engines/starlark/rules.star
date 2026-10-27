# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
#
# The access-decision rules in Starlark: a port of ../../oracle.py within the
# language as shipped (no sets, no recursion, no while). Sets are dicts, the
# delegation chains are built level by level instead of recursively, and every
# loop that follows a graph is bounded with range, which is the same device the
# Jaxson program uses.

MAX_HOPS = 3
BIG = 1000000

def reach(start, edges):
    seen = {start: True}
    frontier = [start]
    for _ in range(BIG):
        if not frontier:
            break
        nxt = []
        for x in frontier:
            for y in edges.get(x, []):
                if y not in seen:
                    seen[y] = True
                    nxt.append(y)
        frontier = nxt
    return seen

def anc_or_self(parent, res):
    out = []
    cur = res
    for _ in range(len(parent) + 1):
        if cur == None:
            break
        out.append(cur)
        cur = parent[cur]
    return out

def decide(db):
    parent = {r["id"]: r["parent"] for r in db["resources"]}
    role = {r["id"]: r for r in db["roles"]}
    inherits = {r["id"]: r["inherits"] for r in db["roles"]}
    up = {}
    for m in db["members"]:
        up.setdefault(m["member"], []).append(m["group"])
    allow_by = {}
    deny_by = {}
    for g in db["grants"]:
        side = allow_by if g["effect"] == "allow" else deny_by
        side.setdefault(g["subject"], []).append(g)
    deleg_to = {}
    for d in db["delegations"]:
        deleg_to.setdefault(d["to"], []).append(d)

    subj_cache = {}
    perm_cache = {}

    def subjects(u):
        if u not in subj_cache:
            subj_cache[u] = reach(u, up)
        return subj_cache[u]

    def perms(r):
        if r not in perm_cache:
            acts = {}
            for x in reach(r, inherits):
                for a in role[x]["perms"]:
                    acts[a] = True
            perm_cache[r] = acts
        return perm_cache[r]

    # which (user, budget) pairs the chains need: the requesters at the full budget,
    # and the delegators of everyone needed at the next budget up
    needed = [{} for _ in range(MAX_HOPS + 1)]
    for q in db["requests"]:
        needed[MAX_HOPS][q["user"]] = True
    for b in range(MAX_HOPS, 0, -1):
        for u in needed[b]:
            for d in deleg_to.get(u, []):
                needed[b - 1][d["from"]] = True

    chains = {}
    for b in range(MAX_HOPS + 1):
        for u in needed[b]:
            out = []
            for s in subjects(u):
                for g in allow_by.get(s, []):
                    out.append((g["role"], g["scope"], 0, g["id"]))
            if b > 0:
                for d in deleg_to.get(u, []):
                    covering = {x: True for x in anc_or_self(parent, d["scope"])}
                    for e in chains[(d["from"], b - 1)]:
                        if e[0] == d["role"] and e[1] in covering:
                            out.append((e[0], d["scope"], e[2] + 1, e[3] + ">" + d["id"]))
            chains[(u, b)] = out

    held = {}

    def held_of(u):
        if u not in held:
            best = {}
            for e in chains[(u, MAX_HOPS)]:
                k = (e[0], e[1])
                cand = (e[2], e[3])
                if k not in best or cand < best[k]:
                    best[k] = cand
            held[u] = best
        return held[u]

    result = {}
    for q in db["requests"]:
        u = q["user"]
        a = q["action"]
        anc = {x: True for x in anc_or_self(parent, q["resource"])}
        supports = [hc for k, hc in held_of(u).items() if k[1] in anc and a in perms(k[0])]
        denied = []
        for s in subjects(u):
            for g in deny_by.get(s, []):
                if g["scope"] in anc and a in perms(g["role"]):
                    denied.append(g["id"])
        denied = sorted(denied)
        allow = len(supports) > 0 and len(denied) == 0
        result[q["id"]] = {"allow": allow, "via": min(supports)[1] if allow else None, "denied_by": denied}
    return result

OUT = json.encode(decide(json.decode(INPUT)))
