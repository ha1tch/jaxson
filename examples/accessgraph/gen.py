# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Build the accessgraph Jaxson package from a synthetic database.

  python3 gen.py SEED [tiny|small|medium] > pkg.json

The input is the database re-keyed for lookup; nothing derived is
precomputed. Closures are unrolled to the fixed rounds in policy.py, so the
program is the same for every database, and it asserts that the data fits.
"""
import json
import sys

import cost
import oracle
import policy
import synth


def P(*s): return {"$path": list(s)}
def L(v, *r): return P("local", v, *r)
def V(n): return {"$v": n}
def C(w, e): return {"$compute": {"with": w, "expr": e}}
def op2(op, a, b): return C({"a": a, "b": b}, [op, V("a"), V("b")])
def has(o, k): return op2("has", o, k)
def no(x): return C({"b": x}, ["not", V("b")])
def keys(o): return C({"o": o}, ["keys", V("o")])
def cat(*x):
    e = x[0]
    for v in x[1:]:
        e = op2("concat", e, v)
    return e
def S(path, v): return {"op": "set", "path": path, "value": v}
def A(path, v): return {"op": "append", "path": path, "value": v}
def F(i, v, body): return {"op": "for", "in": i, "as": v, "do": body}
def I(c, t, e=None): return dict({"op": "if", "cond": c, "then": t}, **({"else": e} if e else {}))
def st(*s): return P("state", *s)


def less(h, c, eh, ec):  # (h, c) < (eh, ec)
    return C({"h": h, "c": c, "eh": eh, "ec": ec},
             ["or", ["lt", V("h"), V("eh")], ["and", ["eq", V("h"), V("eh")], ["lt", V("c"), V("ec")]]])


def closure(out, owners, edges, rounds, detect=None):
    p = [S(["state", out], {}),
         F(P("input", owners), "o", [S(["state", out, L("o")], {}), S(["state", out, L("o"), L("o")], True)])]
    for _ in range(rounds):
        p.append(F(P("input", owners), "o", [F(keys(st(out, L("o"))), "x", [
            F(P("input", edges, L("x")), "y", [
                I(no(has(st(out, L("o")), L("y"))), [S(["state", out, L("o"), L("y")], True)])])])]))
    if detect:   # one more round: anything it adds means the data is deeper than the rounds
        over = out + "Over"
        p += [S(["state", over], {}), F(P("input", owners), "o", [F(keys(st(out, L("o"))), "x", [
            F(P("input", edges, L("x")), "y", [
                I(no(has(st(out, L("o")), L("y"))), [S(["state", over, L("o")], True)])])])]),
              {"op": "assert", "that": C({"k": keys(st(over))}, ["eq", ["len", V("k")], 0]), "msg": detect}]
    return p


def upsert(hu, role, scope, h, c):  # hu: path of the user's held map
    e = {"$tpl": {"role": role, "scope": scope, "h": h, "c": c}}
    ex = lambda f: P(*hu, st("k"), f)
    return [S(["state", "k"], cat(role, "|", scope)),
            I(has(P(*hu), st("k")),
              [I(less(h, c, ex("h"), ex("c")), [S(hu + [st("k")], e)])],
              [S(hu + [st("k")], e)])]


def program(b, hops, detect=True):
    p = closure("sub", "users", "up", b["group_rounds"],
                "membership is nested deeper than GROUP_ROUNDS" if detect else None)
    p += closure("rc", "roles", "roleInh", b["role_rounds"],
                 "role inheritance is deeper than ROLE_ROUNDS" if detect else None)
    p += [S(["state", "perm"], {}), F(P("input", "roles"), "r", [
        S(["state", "perm", L("r")], {}),
        F(keys(st("rc", L("r"))), "x", [F(P("input", "rolePerms", L("x")), "a", [
            S(["state", "perm", L("r"), L("a")], True)])])])]
    walk = [I(op2("ne", st("cur"), ""), [S(["state", "ancs", L("r"), st("cur")], True),
                                         S(["state", "cur"], P("input", "parent", st("cur")))])]
    p += [S(["state", "ancs"], {}), F(P("input", "resources"), "r", [
        S(["state", "ancs", L("r")], {}), S(["state", "cur"], L("r"))] + walk * (b["res_depth"] + 1) + (
        [{"op": "assert", "that": op2("eq", st("cur"), ""), "msg": "a resource is deeper than RES_DEPTH"}]
        if detect else []))]
    hu = ["state", "held", L("u")]
    p += [S(["state", "held"], {}), F(P("input", "users"), "u", [
        S(hu, {}),
        F(keys(st("sub", L("u"))), "s", [F(P("input", "allow", L("s")), "g", upsert(
            hu, L("g", "role"), L("g", "scope"), 0, L("g", "id")))])])]
    for k in range(1, hops + 1):
        hf = ["state", "held", L("d", "from"), L("k")]
        ok = C({"er": P(*hf, "role"), "dr": L("d", "role"), "eh": P(*hf, "h"), "es": P(*hf, "scope"),
                "cov": st("ancs", L("d", "scope"))},
               ["and", ["eq", V("er"), V("dr")], ["and", ["eq", V("eh"), k - 1], ["has", V("cov"), V("es")]]])
        p.append(F(P("input", "delegs"), "d", [F(keys(st("held", L("d", "from"))), "k", [I(ok, upsert(
            ["state", "held", L("d", "to")], L("d", "role"), L("d", "scope"), k,
            cat(P(*hf, "c"), ">", L("d", "id"))))])]))
    he = ["state", "held", L("q", "user"), L("k")]
    cover = lambda scope, role: C({"A": st("ancs", L("q", "resource")), "s": scope, "pm": st("perm", role),
                                   "a": L("q", "action")},
                                  ["and", ["has", V("A"), V("s")], ["has", V("pm"), V("a")]])
    nden = C({"k": keys(st("den"))}, ["len", V("k")])
    p += [S(["output"], {}), F(P("input", "requests"), "q", [
        S(["state", "sup"], False), S(["state", "den"], {}),
        F(keys(st("held", L("q", "user"))), "k", [I(cover(P(*he, "scope"), P(*he, "role")), [
            I(no(st("sup")),
              [S(["state", "sup"], True), S(["state", "bh"], P(*he, "h")), S(["state", "bc"], P(*he, "c"))],
              [I(less(P(*he, "h"), P(*he, "c"), st("bh"), st("bc")),
                 [S(["state", "bh"], P(*he, "h")), S(["state", "bc"], P(*he, "c"))])])])]),
        F(keys(st("sub", L("q", "user"))), "s", [F(P("input", "deny", L("s")), "g", [
            I(cover(L("g", "scope"), L("g", "role")), [S(["state", "den", L("g", "id")], True)])])]),
        I(C({"s": st("sup"), "n": nden}, ["and", V("s"), ["eq", V("n"), 0]]),
          [S(["output", L("q", "id")], {"$tpl": {"allow": True, "via": st("bc"), "denied_by": []}})],
          [S(["output", L("q", "id")], {"$tpl": {"allow": False, "via": None, "denied_by": keys(st("den"))}})])])]
    return p


def make_input(db):
    subj = db["users"] + db["groups"]
    up, allow, deny = ({s: [] for s in subj} for _ in range(3))
    for m in db["members"]:
        up[m["member"]].append(m["group"])
    for g in db["grants"]:
        (allow if g["effect"] == "allow" else deny)[g["subject"]].append(
            {"id": g["id"], "role": g["role"], "scope": g["scope"]})
    return {"users": db["users"], "roles": [r["id"] for r in db["roles"]],
            "resources": [r["id"] for r in db["resources"]], "up": up, "allow": allow, "deny": deny,
            "roleInh": {r["id"]: r["inherits"] for r in db["roles"]},
            "rolePerms": {r["id"]: r["perms"] for r in db["roles"]},
            "parent": {r["id"]: r["parent"] or "" for r in db["resources"]},
            "delegs": db["delegations"], "requests": db["requests"]}


def build(db, steps=None, hops=policy.MAX_HOPS, rounds=None, detect=True):
    """The package for db. The program is the same for every database: its rounds
    are policy.LIMITS. A database deeper than the limits is refused here
    (LimitExceeded) and, if it got past, by the program itself.

    rounds and detect exist for the fault-planting script: other round counts,
    and the assertion left out. Passing rounds skips the check made here.

    steps is the declared limit. By default it is derived from db before the run
    (cost.py). The fault-planting variants are not the shipped program, so they
    get twice the derived limit."""
    if rounds is None:
        oracle.check_limits(db)
    if steps is None:
        steps = cost.limit(db) * (1 if rounds is None and hops == policy.MAX_HOPS and detect else 2)
    return {"jaxson": "1.0", "limits": {"steps": steps}, "inputSchema": {"type": "any"},
            "outputSchema": {"type": "any"}, "input": make_input(db),
            "program": program(rounds or policy.LIMITS, hops, detect)}


if __name__ == "__main__":
    json.dump(build(synth.generate(int(sys.argv[1]), sys.argv[2] if len(sys.argv) > 2 else "small")),
              sys.stdout, sort_keys=True)
    sys.stdout.write("\n")
