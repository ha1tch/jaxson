# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Reference oracle for the accessgraph example.

Computes the decision for every request in a synthetic access-graph database
by top-down enumeration of derivations. It is deliberately a different
algorithm from the generated Jaxson package (bottom-up rounds), so that
agreement between the two means something.

Semantics (see README.md):

  subjects(u)   u, plus every group reachable from u through membership edges
                (nesting and cycles allowed; reachability is unbounded here).
  held(u)       (role, scope) pairs u holds, each with its best derivation:
                  hops 0: an allow grant whose subject is in subjects(u);
                  hops k+1: a delegation d to u, where d.from holds d.role on
                  a scope that is d.scope or an ancestor of it, at hops k.
                Delegation chains are bounded by max_hops. Holding a role that
                inherits d.role does not entitle the holder to delegate it.
  best          minimum (hops, chain) where chain is the fixed-width id string
                "G000007>D000003>..." (grant id, then one delegation id per hop).
  allow(u,a,r)  some held (role, scope) has scope in anc_or_self(r) and a in
                perms(role closure); and no deny grant does the same for a
                subject in subjects(u). Denies are not transitive through
                delegation and do not stop a delegator from delegating.
"""
import re

import policy
from policy import MAX_HOPS

_GRANT_ID = re.compile(r"G[0-9]{%d}\Z" % policy.ID_WIDTH)
_DELEG_ID = re.compile(r"D[0-9]{%d}\Z" % policy.ID_WIDTH)


def build_index(db):
    ix = {"parent": {r["id"]: r["parent"] for r in db["resources"]},
          "role": {r["id"]: r for r in db["roles"]},
          "up": {}, "allow_by_subject": {}, "deny_by_subject": {}, "deleg_to": {}}
    for m in db["members"]:
        ix["up"].setdefault(m["member"], []).append(m["group"])
    for g in db["grants"]:
        side = "allow_by_subject" if g["effect"] == "allow" else "deny_by_subject"
        ix[side].setdefault(g["subject"], []).append(g)
    for d in db["delegations"]:
        ix["deleg_to"].setdefault(d["to"], []).append(d)
    return ix


def check_refs(db):
    """Raise ValueError on the first dangling reference. The oracle is only
    defined on referentially clean databases."""
    users, groups = set(db["users"]), set(db["groups"])
    roles = {r["id"] for r in db["roles"]}
    res = {r["id"] for r in db["resources"]}
    subj = users | groups

    def need(cond, what):
        if not cond:
            raise ValueError("dangling reference: " + what)

    for r in db["roles"]:
        for p in r["inherits"]:
            need(p in roles, "role %s inherits %s" % (r["id"], p))
    for r in db["resources"]:
        need(r["parent"] is None or r["parent"] in res, "resource %s parent" % r["id"])
    for m in db["members"]:
        need(m["member"] in subj and m["group"] in groups, "member %r" % (m,))
    for g in db["grants"]:
        need(g["subject"] in subj and g["role"] in roles and g["scope"] in res
             and g["effect"] in ("allow", "deny"), "grant %s" % g["id"])
    for d in db["delegations"]:
        need(d["from"] in users and d["to"] in users and d["role"] in roles
             and d["scope"] in res, "delegation %s" % d["id"])
    for q in db["requests"]:
        need(q["user"] in users and q["resource"] in res, "request %s" % q["id"])


def anc_or_self(parent, res):
    out, cur = [], res
    while cur is not None and len(out) <= len(parent):
        out.append(cur)
        cur = parent[cur]
    return out


def _reach(start, edges):
    seen, stack = {start}, [start]
    while stack:
        x = stack.pop()
        for y in edges.get(x, []):
            if y not in seen:
                seen.add(y)
                stack.append(y)
    return seen


def evaluate(db, max_hops=MAX_HOPS, detailed=False, enforce=True):
    """Return {request_id: {"allow", "via", "denied_by"}} (plus "supported"
    and "hops" when detailed).

    Defined on databases within the policy limits (policy.py): a deeper one
    raises LimitExceeded. enforce=False lifts that, for tests that show what an
    unbounded evaluation would answer."""
    check_refs(db)
    if enforce:
        check_limits(db)
    ix = build_index(db)
    inherits = {r["id"]: r["inherits"] for r in db["roles"]}
    subj_cache, perm_cache, memo = {}, {}, {}

    def subjects(u):
        if u not in subj_cache:
            subj_cache[u] = _reach(u, ix["up"])
        return subj_cache[u]

    def perms(role):
        if role not in perm_cache:
            perm_cache[role] = {a for r in _reach(role, inherits)
                                for a in ix["role"][r]["perms"]}
        return perm_cache[role]

    def chains(u, budget):
        key = (u, budget)
        if key in memo:
            return memo[key]
        out = []
        for s in subjects(u):
            for g in ix["allow_by_subject"].get(s, []):
                out.append((g["role"], g["scope"], 0, g["id"]))
        if budget > 0:
            for d in ix["deleg_to"].get(u, []):
                covering = set(anc_or_self(ix["parent"], d["scope"]))
                for role, scope, h, c in chains(d["from"], budget - 1):
                    if role == d["role"] and scope in covering:
                        out.append((role, d["scope"], h + 1, c + ">" + d["id"]))
        memo[key] = out
        return out

    held_cache = {}

    def held(u):
        if u not in held_cache:
            best = {}
            for role, scope, h, c in chains(u, max_hops):
                k = (role, scope)
                if k not in best or (h, c) < best[k]:
                    best[k] = (h, c)
            held_cache[u] = best
        return held_cache[u]

    result = {}
    for q in db["requests"]:
        u, a = q["user"], q["action"]
        anc = set(anc_or_self(ix["parent"], q["resource"]))
        supports = [hc for (role, scope), hc in held(u).items()
                    if scope in anc and a in perms(role)]
        denied_by = sorted(g["id"] for s in subjects(u)
                           for g in ix["deny_by_subject"].get(s, [])
                           if g["scope"] in anc and a in perms(g["role"]))
        allow = bool(supports) and not denied_by
        out = {"allow": allow,
               "via": min(supports)[1] if allow else None,
               "denied_by": denied_by}
        if detailed:
            out["supported"] = bool(supports)
            out["hops"] = min(supports)[0] if supports else None
        result[q["id"]] = out
    return result


def _bfs_depth(start, edges):
    dist, frontier, d = {start: 0}, [start], 0
    while frontier:
        nxt = []
        for x in frontier:
            for y in edges.get(x, []):
                if y not in dist:
                    dist[y] = d + 1
                    nxt.append(y)
        frontier, d = nxt, d + 1
    return max(dist.values())


class LimitExceeded(ValueError):
    """A database is deeper than the policy limits, so it is outside the domain."""


def over_limits(db):
    """The users, roles and resources deeper than the policy limits, and the
    grant and delegation ids that are not policy.ID_WIDTH digits, by kind; an
    empty dict when the database is within them."""
    ix = build_index(db)
    inherits = {r["id"]: r["inherits"] for r in db["roles"]}
    over = {
        "group_rounds": [u for u in db["users"] if _bfs_depth(u, ix["up"]) > policy.GROUP_ROUNDS],
        "role_rounds": [r for r in inherits if _bfs_depth(r, inherits) > policy.ROLE_ROUNDS],
        "res_depth": [r for r in ix["parent"] if len(anc_or_self(ix["parent"], r)) - 1 > policy.RES_DEPTH],
        "id_width": [g["id"] for g in db["grants"] if not _GRANT_ID.match(g["id"])]
                    + [d["id"] for d in db["delegations"] if not _DELEG_ID.match(d["id"])],
    }
    return {k: v for k, v in over.items() if v}


def check_limits(db):
    over = over_limits(db)
    if over:
        raise LimitExceeded("database exceeds the policy limits: " + "; ".join(
            "%s: %d over (first %s)" % (k, len(v), v[0]) for k, v in over.items()))


def bounds(db):
    """Round counts a bounded bottom-up evaluator needs to be complete on this
    database: longest shortest-path through membership and role inheritance,
    and the deepest resource."""
    ix = build_index(db)
    inherits = {r["id"]: r["inherits"] for r in db["roles"]}
    return {
        "group_rounds": max([_bfs_depth(u, ix["up"]) for u in db["users"]] + [0]),
        "role_rounds": max([_bfs_depth(r, inherits) for r in inherits] + [0]),
        "res_depth": max(len(anc_or_self(ix["parent"], r)) - 1 for r in ix["parent"]),
    }
