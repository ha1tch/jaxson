# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""The declared step limit, derived from the database before the run.

A fixed limit does not fit a program whose cost grows with the data: 5,000 users
need twice the default. Here the limit is computed from the database itself,
from counts and closure sizes, before anything runs:

    declared = ceil(margin * sum(coefficient[k] * feature[k]))

The features follow the program's loops. A closure scans every subject it has
reached and the edges of each, once per round, so its cost is the number of
rounds (plus the assertion's round, from policy.py) times the sum of those
sizes; the group closure and the role closure are the same code, so they share
one coefficient. The held table is built for every user, not only for those who
ask: each user scans every subject it reaches and every allow grant of each. A
delegation hop scans, for every delegation, the held entries of its delegator,
once per hop. A request scans the held entries of its user and the denies of
every subject its user reaches. The coefficients are fitted on measured step
counts by calibrate.py, and so is the margin; both are stored in
cost_model.json with a fingerprint of the generated program. Change the
program and the model refuses to run until it is calibrated again.

The limit is exceeded loudly when it is: the runtime stops with
RESOURCE_ERROR/STEPS. Nothing here raises a limit to make a run pass.
"""
import hashlib
import json
import math
import os

import oracle
import policy

MODEL = os.path.join(os.path.dirname(os.path.abspath(__file__)), "cost_model.json")
FEATURES = ["const", "groups", "roles", "resources", "build", "hops", "req_held", "req_deny", "requests"]
TERMS = ["const", "closure", "resources", "build", "hops", "req_held", "req_deny", "requests"]


def _reach(adj, start):
    """start and everything reachable from it."""
    seen = {start}
    todo = [start]
    while todo:
        x = todo.pop()
        for y in adj.get(x, ()):
            if y not in seen:
                seen.add(y)
                todo.append(y)
    return seen


def features(db):
    """The cost drivers of the program on db, from the data alone."""
    ix = oracle.build_index(db)
    up = ix["up"]
    inh = {r["id"]: r["inherits"] for r in db["roles"]}
    reach_u = {u: _reach(up, u) for u in db["users"]}
    reach_r = {r: _reach(inh, r) for r in inh}
    w_groups = sum(len(R) + sum(len(up.get(x, ())) for x in R) for R in reach_u.values())
    w_roles = sum(len(R) + sum(len(inh.get(x, ())) for x in R) for R in reach_r.values())
    n_allow = {s: len(gs) for s, gs in ix["allow_by_subject"].items()}
    n_deny = {s: len(gs) for s, gs in ix["deny_by_subject"].items()}
    # the held entries of a user: the allow grants of every subject it reaches. An
    # upper bound: the table has one entry per distinct (role, scope) pair, but
    # a feature counting distinct pairs fitted synthetic data more tightly and
    # failed on the real-shaped data, where almost every pair is distinct.
    held = {u: sum(n_allow.get(s, 0) for s in R) for u, R in reach_u.items()}
    deny_scan = {u: sum(1 + n_deny.get(s, 0) for s in R) for u, R in reach_u.items()}
    return {
        "const": 1,
        "groups": (policy.GROUP_ROUNDS + 1) * w_groups,
        "roles": (policy.ROLE_ROUNDS + 1) * w_roles,
        "resources": len(db["resources"]) * (policy.RES_DEPTH + 2),
        "build": sum(len(R) + held[u] for u, R in reach_u.items()),
        "hops": policy.MAX_HOPS * sum(1 + held[d["from"]] for d in db["delegations"]),
        "req_held": sum(1 + held[q["user"]] for q in db["requests"]),
        "req_deny": sum(deny_scan[q["user"]] for q in db["requests"]),
        "requests": len(db["requests"]),
    }


def terms(f):
    """The regressors of the model, from the features. The group and role
    closures are the same code, so they are one term."""
    return {"const": f["const"], "closure": f["groups"] + f["roles"], "resources": f["resources"],
            "build": f["build"], "hops": f["hops"], "req_held": f["req_held"],
            "req_deny": f["req_deny"], "requests": f["requests"]}


def program_sha256():
    import gen
    prog = gen.program(policy.LIMITS, policy.MAX_HOPS)
    return hashlib.sha256(json.dumps(prog, sort_keys=True).encode()).hexdigest()


_cached = {}


def model():
    if "m" not in _cached:
        m = json.load(open(MODEL))
        if m["program_sha256"] != program_sha256():
            raise RuntimeError("cost model is stale: the generated program changed; run calibrate.py --write")
        _cached["m"] = m
    return _cached["m"]


def predict(db, m=None):
    m = m or model()
    t = terms(features(db))
    return sum(m["coef"][k] * t[k] for k in TERMS)


def limit(db):
    """The declared step limit for db."""
    m = model()
    return int(math.ceil(m["margin"] * predict(db, m)))
