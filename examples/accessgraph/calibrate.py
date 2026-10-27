# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Calibrate the cost model (cost.py): fit it on measured step counts, derive the
margin, and check it on data the fit never saw.

  python3 calibrate.py [--write] [--cache FILE]

Step counts come from the stand-in, which counts exactly like the real runner
(test_policy.py and README). The coefficients are the non-negative least squares
fit of relative error, so a small database counts as much as a large one. The
margin is the worst measured-over-predicted ratio on the calibration set plus
5%, rounded up to the next 0.05. The held-out set includes a preset larger than
anything fitted (xlarge, 5,000 users, against at most 1,000 in the fit); the
limit must hold on all of it. --write stores the model in cost_model.json with a
fingerprint of the generated program.
"""
import copy
import json
import math
import os
import random
import sys
import time

import numpy as np

import cost
import gen
import jx
import oracle
import policy
import synth

CALIBRATION = [("tiny", range(0, 30)), ("small", range(0, 15)), ("medium", range(0, 8)), ("large", range(0, 2))]
# Generated databases grow every dimension together, so a fit on them alone cannot tell the
# terms apart (it gave resources, and earlier roles, a coefficient of zero). These stretch one
# dimension of a tiny database at a time, which makes each term identifiable.
CONTROLLED = ["resources", "resources-deep", "roles", "grants", "denies", "delegations", "requests", "users", "groups"]
CALIBRATION += [("ctl:" + n, range(0, 2)) for n in CONTROLLED]
HELD_OUT = [("tiny", range(100, 110)), ("small", range(100, 110)), ("medium", range(100, 104)),
            ("large", range(7, 9)), ("xlarge", range(0, 1)), ("ratio", range(0, 1))]
PRESETS = ["tiny", "small", "medium", "large", "xlarge"]
AMAZON = os.path.join(os.path.dirname(os.path.abspath(__file__)), "data", "amazon-db")

# The proportions of the Amazon data (users, groups, resources, grants, delegations,
# requests per user) at 1,500 users: a synthetic database that is held out of the fit
# and has the shape that broke the first model.
synth.PRESETS["ratio"] = dict(users=1500, groups=252, roles=16, resources=900, grants=17550, denies=340,
                              delegations=1070, chains=100, requests=416)


def controlled(name, seed):
    """A tiny database stretched in one dimension."""
    rng = random.Random(1000 + seed)
    db = copy.deepcopy(synth.generate(seed, "tiny"))
    users, groups = list(db["users"]), list(db["groups"])
    roles, res = [r["id"] for r in db["roles"]], [r["id"] for r in db["resources"]]
    root = res[0]

    def grant(effect):
        return {"id": policy.grant_id(len(db["grants"]) + 1), "subject": rng.choice(users + groups),
                "role": rng.choice(roles), "scope": rng.choice(res), "effect": effect}

    if name == "resources":
        db["resources"] += [{"id": "rz%04d" % i, "parent": root} for i in range(150)]
    elif name == "resources-deep":
        for c in range(30):
            prev = root
            for d in range(4):
                db["resources"].append({"id": "rz%04d" % (c * 4 + d), "parent": prev})
                prev = "rz%04d" % (c * 4 + d)
    elif name == "roles":
        for i in range(40):
            db["roles"].append({"id": "rz%02d" % i, "perms": sorted(rng.sample(synth.ACTIONS, rng.choice([1, 2]))),
                                "inherits": ["rz%02d" % (i - 1)] if i % 6 else []})
    elif name in ("grants", "denies"):
        for _ in range(400 if name == "grants" else 200):
            db["grants"].append(grant("allow" if name == "grants" else "deny"))
    elif name == "delegations":
        for _ in range(300):
            frm = rng.choice(users)
            to = rng.choice([u for u in users if u != frm])
            db["delegations"].append({"id": policy.deleg_id(len(db["delegations"]) + 1), "from": frm, "to": to,
                                      "role": rng.choice(roles), "scope": rng.choice(res)})
    elif name == "requests":
        for _ in range(300):
            db["requests"].append({"id": "q%05d" % (len(db["requests"]) + 1), "user": rng.choice(users),
                                   "action": rng.choice(synth.ACTIONS), "resource": rng.choice(res)})
    elif name == "users":
        for i in range(150):
            u = "uz%04d" % i
            db["users"].append(u)
            for g in rng.sample(groups, rng.choice([1, 2])):
                db["members"].append({"member": u, "group": g})
    elif name == "groups":
        for i in range(30):
            g = "gz%03d" % i
            db["groups"].append(g)
            db["members"].append({"member": g, "group": "gz%03d" % (i - 1) if i % 5 else rng.choice(groups)})
            for u in rng.sample(users, 2):
                db["members"].append({"member": u, "group": g})
    else:
        raise ValueError(name)
    oracle.check_refs(db)
    assert not oracle.over_limits(db), (name, oracle.over_limits(db))
    return db


def make(size, seed):
    return controlled(size[4:], seed) if size.startswith("ctl:") else synth.generate(seed, size)


def measure(size, seed):
    db = make(size, seed)
    steps = jx.run(gen.build(db, steps=2000000000))[1]
    return {"size": size, "seed": seed, "steps": steps, "features": cost.features(db),
            "users": len(db["users"])}


def collect(plan, cache, name):
    out = []
    for size, seeds in plan:
        for seed in seeds:
            key = "%s/%s/%d" % (name, size, seed)
            if key not in cache:
                cache[key] = measure(size, seed)
            rec = dict(cache[key])
            rec["features"] = cost.features(make(size, seed))   # always fresh: only step counts are cached
            out.append(rec)
    return out


def nnls(X, y):
    keep = list(range(X.shape[1]))
    while True:
        beta, *_ = np.linalg.lstsq(X[:, keep], y, rcond=None)
        if (beta >= 0).all():
            break
        keep.pop(int(np.argmin(beta)))
    full = np.zeros(X.shape[1])
    full[keep] = beta
    return full


def matrix(rows):
    return np.array([[cost.terms(r["features"])[k] for k in cost.TERMS] for r in rows], dtype=float)


def main(argv):
    cache_file = argv[argv.index("--cache") + 1] if "--cache" in argv else None
    cache = json.load(open(cache_file)) if cache_file and __import__("os").path.exists(cache_file) else {}
    t0 = time.time()
    cal = collect(CALIBRATION, cache, "cal")
    held = collect(HELD_OUT, cache, "held")
    if cache_file:
        json.dump(cache, open(cache_file, "w"))
    print("measured %d calibration and %d held-out databases in %.0f s" % (len(cal), len(held), time.time() - t0))

    X = matrix(cal)
    y = np.array([r["steps"] for r in cal], dtype=float)
    coef = nnls(X / y[:, None], np.ones(len(y)))
    pred = X @ coef
    worst_cal = float((y / pred).max())
    margin = math.ceil((worst_cal + 0.05) / 0.05 - 1e-9) * 0.05
    margin = round(margin, 2)
    print("coefficients:", {k: round(float(c), 4) for k, c in zip(cost.TERMS, coef)})
    print("calibration: measured / predicted from %.3f to %.3f; margin %.2f" % ((y / pred).min(), worst_cal, margin))

    Xh = matrix(held)
    yh = np.array([r["steps"] for r in held], dtype=float)
    ph = Xh @ coef
    declared = np.ceil(margin * ph)
    exceeded = int((yh > declared).sum())
    print("held out: %d databases, measured / predicted from %.3f to %.3f, %d over the declared limit"
          % (len(held), (yh / ph).min(), (yh / ph).max(), exceeded))
    xl = [i for i, r in enumerate(held) if r["size"] == "xlarge"]
    for i in xl:
        print("  xlarge (never fitted, 5x the largest fitted size): measured %d, predicted %d, declared %d"
              % (yh[i], ph[i], declared[i]))

    print("\nmeasured against declared steps, seed 0 of every preset")
    print("%-8s %7s %12s %12s %7s" % ("preset", "users", "measured", "declared", "used"))
    seed0 = {r["size"]: (r, "cal") for r in cal if r["seed"] == 0}
    seed0.update({r["size"]: (r, "held") for r in held if r["seed"] == 0})
    for size in PRESETS:
        r, _ = seed0[size]
        d = math.ceil(margin * sum(coef[i] * cost.terms(r["features"])[k] for i, k in enumerate(cost.TERMS)))
        print("%-8s %7d %12d %12d %6.0f%%" % (size, r["users"], r["steps"], d, 100.0 * r["steps"] / d))

    unseen = []
    for v in ("flat", "nested"):
        dbf, sf = os.path.join(AMAZON, v + ".json"), os.path.join(AMAZON, "steps-%s.json" % v)
        if os.path.exists(dbf) and os.path.exists(sf):
            f = cost.terms(cost.features(json.load(open(dbf))))
            unseen.append((v, json.load(open(sf))["steps"], sum(coef[i] * f[k] for i, k in enumerate(cost.TERMS))))
    if unseen:
        print("\nreal-shaped data, never in the fit (the Amazon databases of provision/amazon_to_db.py):")
        for v, s, p in unseen:
            d = math.ceil(margin * p)
            print("  %-7s measured %d, predicted %.0f, measured / predicted %.3f, declared %d: %s"
                  % (v, s, p, s / p, d, "fits" if s <= d else "EXCEEDED"))
            exceeded += s > d

    if "--write" in argv:
        model = {"program_sha256": cost.program_sha256(),
                 "coef": {k: float(c) for k, c in zip(cost.TERMS, coef)},
                 "margin": margin,
                 "calibration": {"databases": len(cal), "worst_measured_over_predicted": round(worst_cal, 4),
                                 "held_out_databases": len(held), "held_out_exceeded": exceeded,
                                 "held_out_worst_measured_over_predicted": round(float((yh / ph).max()), 4),
                                 "unseen_real_shaped": [{"variant": v, "measured": s, "predicted": round(p)} for v, s, p in unseen]}}
        json.dump(model, open(cost.MODEL, "w"), indent=2)
        print("\nwrote", cost.MODEL)
    return 1 if exceeded else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
