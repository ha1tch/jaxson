# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
#!/usr/bin/env python3
"""Compile medical3's vocabulary into a Jaxson program that derives, bottom-up and bounded to the
solver's depth, every headline conclusion with the parts of medical3's assessment/2 record, and the
recommendation tiers of recommend_tiers/2."""
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..", "medical"))
from spike import P, LP, V, C, set_, append, for_, if_, eq, has  # noqa: E402

M3 = json.load(open(os.path.join(HERE, "data", "m3.json")))
RANK = {"supported": 4, "conditional": 3, "conflicted": 2, "unknown": 1, "refuted": 0}
RANK_NAMES = ["refuted", "unknown", "conflicted", "conditional", "supported"]
DEPTH, SNAP, DEEP = 6, 6, 9


def get(o, k):
    return ["get", o, k]


def term_key_expr(s, o, p):
    return ["concat", get(s, "k"), ":", get(s, "v"), "|", p, "|", get(o, "k"), ":", get(o, "v")]


def cand(s_op, p, o_op, r_op, i_op, n_op, h_op):
    key = C({"s": s_op, "o": o_op}, term_key_expr(V("s"), V("o"), p))
    return append(["state", "cand"], {"$tpl": {"key": key, "s": s_op, "p": p, "o": o_op, "r": r_op, "i": i_op, "n": n_op, "h": h_op}})


def conj(exprs):
    return exprs[0] if len(exprs) == 1 else ["and"] + exprs


def guarded(withs, exprs, body):
    if not exprs:
        return body
    return [if_(C(withs, conj(exprs)), body)]


def nest_mul(names):
    e = V(names[-1])
    for n in reversed(names[:-1]):
        e = ["mul", V(n), e]
    return e


def height_of(paths):
    """1 + the tallest premise proof"""
    withs = {"h%d" % i: P(*p) for i, p in enumerate(paths)}
    names = list(withs)
    return C(withs, ["add", 1, ["max"] + [V(n) for n in names] if len(names) > 1 else V(names[0])]) if len(names) > 1 else C(withs, ["add", 1, V(names[0])])


def combine(prefix, paths, how):
    """min or product over fields of already-matched facts"""
    withs = {"%s%d" % (prefix, i): P(*p) for i, p in enumerate(paths)}
    names = list(withs)
    if how == "min":
        return C(withs, ["min"] + [V(n) for n in names]) if len(names) > 1 else C(withs, V(names[0]))
    return C(withs, nest_mul(names))


# ------------------------------------------------------------------ schemas
def gen_schema(sc):
    rels = [g for g in sc["body"] if "guard" not in g]
    guards = [g for g in sc["body"] if "guard" in g]
    counter = [0]

    def fresh():
        counter[0] += 1
        return "w%d" % counter[0]

    def operand(env, v):
        return P(*env[v][1])

    def gen(j, env, slots):
        if j == len(rels):
            return finish(env, slots)
        goal = rels[j]
        kvar, slot = "k%d" % j, "g%d" % j
        withs, exprs, new = {}, [], {}
        for comp in ("s", "o"):
            pat = goal[comp]
            gpath = ["state", slot, comp]
            if "var" in pat:
                v = pat["var"]
                if v in env:
                    a, b = fresh(), fresh()
                    withs[a], withs[b] = P(*gpath), operand(env, v)
                    exprs.append(["eq", V(a), V(b)])
                else:
                    new[v] = ("term", gpath)
            elif "concept" in pat:
                inner = pat["concept"]
                a = fresh()
                withs[a] = P(*gpath, "k")
                exprs.append(["eq", V(a), "c"])
                if "var" in inner:
                    v = inner["var"]
                    if v in env:
                        x, y = fresh(), fresh()
                        withs[x], withs[y] = P(*gpath, "v"), operand(env, v)
                        exprs.append(["eq", V(x), V(y)])
                    else:
                        new[v] = ("str", gpath + ["v"])
                else:
                    x = fresh()
                    withs[x] = P(*gpath, "v")
                    exprs.append(["eq", V(x), inner["atom"]])
            else:
                raise ValueError("atom patterns are not used by the shipped vocabulary")
        env2 = dict(env)
        env2.update(new)
        inner_code = gen(j + 1, env2, slots + [slot])
        body = [set_(["state", slot], P("state", "F", LP(kvar)))] + guarded(withs, exprs, inner_code)
        return [for_(P("state", "byP", goal["p"]), kvar, body)]

    def guard_expr(g, env, withs):
        args = []
        for a in g["args"]:
            name = fresh()
            withs[name] = operand(env, a["var"]) if "var" in a else a["atom"]
            args.append(V(name))
        ent = fresh()
        withs[ent] = P("state", "ent")
        e = V(ent)
        if g["guard"] == "distinct":
            return ["ne", args[0], args[1]]
        if g["guard"] == "before":
            return ["lt", get(args[0], "v"), get(args[1], "v")]
        if g["guard"] == "is_product":
            row = get(e, get(args[0], "v"))
            return ["and", ["has", e, get(args[0], "v")], ["eq", get(row, "domain"), "pharma"],
                    ["eq", get(row, "type"), "pharmaceutical"]]
        if g["guard"] == "entity_domain":
            row = get(e, get(args[0], "v"))
            return ["and", ["has", e, get(args[0], "v")], ["eq", get(row, "domain"), args[1]]]
        raise ValueError(g)

    def head_operand(env, pat):
        if "var" in pat:
            return operand(env, pat["var"])
        if "concept" in pat:
            inner = pat["concept"]
            return {"$tpl": {"k": "c", "v": operand(env, inner["var"]) if "var" in inner else inner["atom"]}}
        return {"k": "e", "v": pat["atom"]}

    def finish(env, slots):
        withs = {}
        exprs = [guard_expr(g, env, withs) for g in guards]
        rwith = {"c": RANK[sc["ceiling"]]}
        for i, s in enumerate(slots):
            rwith["r%d" % i] = P("state", s, "r")
        r_op = C(rwith, ["min"] + [V(n) for n in rwith])
        i_op = combine("i", [["state", s, "i"] for s in slots], "min")
        n_op = combine("n", [["state", s, "n"] for s in slots], "mul")
        h = sc["head"]
        h_op = height_of([["state", s, "h"] for s in slots])
        code = [cand(head_operand(env, h["s"]), h["p"], head_operand(env, h["o"]), r_op, i_op, n_op, h_op)]
        return guarded(withs, exprs, code)

    return gen(0, {}, [])


# ------------------------------------------------------------------ axioms, composition
def two_min(a, b, f):
    return C({"x": P("state", a, f), "y": P("state", b, f)}, ["min", V("x"), V("y")])


def gen_compose(r):
    p1, p2, p3 = r["p1"], r["p2"], r["p3"]
    return [for_(P("state", "byP", p1), "ka", [
        set_(["state", "ga"], P("state", "F", LP("ka"))),
        for_(P("state", "byP", p2), "kb", [
            set_(["state", "gb"], P("state", "F", LP("kb"))),
            if_(C({"a": P("state", "ga", "o"), "b": P("state", "gb", "s")}, ["eq", V("a"), V("b")]),
                [cand(P("state", "ga", "s"), p3, P("state", "gb", "o"), two_min("ga", "gb", "r"), two_min("ga", "gb", "i"),
                      C({"x": P("state", "ga", "n"), "y": P("state", "gb", "n")}, ["mul", V("x"), V("y")]),
                      height_of([["state", "ga", "h"], ["state", "gb", "h"]]))])])])]


def gen_axiom(ax):
    kind, a = ax["kind"], ax["args"]
    if kind == "symmetric":   # the stored converse is the only premise: its own strength, one proof
        return [for_(P("state", "baseByP", a[0]), "ka", [
            set_(["state", "ga"], P("state", "F", LP("ka"))),
            if_(C({"s": P("state", "ga", "s"), "o": P("state", "ga", "o")}, ["ne", V("s"), V("o")]),
                [cand(P("state", "ga", "o"), a[0], P("state", "ga", "s"), P("state", "ga", "br"), P("state", "ga", "bi"), 1, 1)])])]
    if kind == "subrel":
        return [for_(P("state", "byP", a[0]), "ka", [
            set_(["state", "ga"], P("state", "F", LP("ka"))),
            cand(P("state", "ga", "s"), a[1], P("state", "ga", "o"), P("state", "ga", "r"), P("state", "ga", "i"), P("state", "ga", "n"),
                 height_of([["state", "ga", "h"]]))])]
    if kind == "inverse":
        return [for_(P("state", "byP", a[1]), "ka", [
            set_(["state", "ga"], P("state", "F", LP("ka"))),
            cand(P("state", "ga", "o"), a[0], P("state", "ga", "s"), P("state", "ga", "r"), P("state", "ga", "i"), P("state", "ga", "n"),
                 height_of([["state", "ga", "h"]]))])]
    if kind == "transitive":
        p = a[0]
        return [for_(P("state", "edges", p), "ke", [
            set_(["state", "ge"], LP("ke")),
            for_(P("state", "byP", p), "kb", [
                set_(["state", "gb"], P("state", "F", LP("kb"))),
                if_(C({"m": P("state", "ge", "m"), "s": P("state", "gb", "s"), "es": P("state", "ge", "s"), "o": P("state", "gb", "o")},
                      ["and", ["eq", V("m"), V("s")], ["ne", V("es"), V("o")], ["ne", V("m"), V("es")]]),
                    [cand(P("state", "ge", "s"), p, P("state", "gb", "o"), two_min("ge", "gb", "r"), two_min("ge", "gb", "i"),
                          P("state", "gb", "n"), height_of([["state", "gb", "h"]]))])])])]
    return []  # defeats is applied when the conclusions are assessed


def gen_edges(p, axioms):
    sym = {ax["args"][0] for ax in axioms if ax["kind"] == "symmetric"}
    sources = [p] + [ax["args"][0] for ax in axioms if ax["kind"] == "subrel" and ax["args"][1] == p]
    code = []
    for q in sources:
        body = [set_(["state", "ga"], P("state", "F", LP("ka"))),
                append(["state", "edges", p], {"$tpl": {"s": P("state", "ga", "s"), "m": P("state", "ga", "o"),
                                                         "r": P("state", "ga", "br"), "i": P("state", "ga", "bi")}})]
        if q in sym:
            body.append(if_(C({"s": P("state", "ga", "s"), "o": P("state", "ga", "o")}, ["ne", V("s"), V("o")]),
                            [append(["state", "edges", p], {"$tpl": {"s": P("state", "ga", "o"), "m": P("state", "ga", "s"),
                                                                     "r": P("state", "ga", "br"), "i": P("state", "ga", "bi")}})]))
        code.append(for_(P("state", "baseByP", q), "ka", body))
    return code


def build():
    preds = [r["p"] for r in M3["preds"]]
    init = [set_(["state", "F"], {}), set_(["state", "byP"], {}), set_(["state", "baseByP"], {}), set_(["state", "ent"], {}),
            set_(["state", "edges"], {}), set_(["state", "cand"], []), set_(["state", "acc"], {}), set_(["state", "tiers"], [])]
    for p in preds:
        init += [set_(["state", "byP", p], []), set_(["state", "baseByP", p], [])]
    for ax in M3["axioms"]:
        if ax["kind"] == "transitive":
            init.append(set_(["state", "edges", ax["args"][0]], []))
    init.append(for_(P("input", "entities"), "e", [set_(["state", "ent", LP("e", "id")], LP("e"))]))
    # base facts: best strength among the sources, and the number of distinct origins behind the fact
    init.append(for_(P("input", "facts"), "f", [
        set_(["state", "best"], -1),
        set_(["state", "origs"], {}),
        for_(LP("f", "ev"), "e", [
            if_(C({"rk": P("input", "ranks"), "s": LP("e", "strength"), "b": P("state", "best")}, ["gt", ["get", V("rk"), V("s")], V("b")]),
                [set_(["state", "best"], C({"rk": P("input", "ranks"), "s": LP("e", "strength")}, ["get", V("rk"), V("s")]))]),
            set_(["state", "lg"], C({"l": LP("e", "lineage")}, ["len", V("l")])),
            if_(C({"n": P("state", "lg")}, ["eq", V("n"), 1]),
                [set_(["state", "og"], C({"l": LP("e", "lineage")}, ["get", V("l"), 0]))],
                [if_(C({"n": P("state", "lg")}, ["eq", V("n"), 0]),
                     [set_(["state", "og"], C({"m": P("input", "lineageOf"), "s": LP("e", "src")}, ["get_or", V("m"), V("s"), V("s")]))],
                     [set_(["state", "og"], LP("e", "src"))])]),
            set_(["state", "origs", P("state", "og")], True),
        ]),
        set_(["state", "ind"], C({"o": P("state", "origs")}, ["len", ["keys", V("o")]])),
        set_(["state", "cur"], {"$tpl": {"s": LP("f", "s"), "p": LP("f", "p"), "o": LP("f", "o"), "r": P("state", "best"),
                                         "i": P("state", "ind"), "n": 1, "b": 1, "br": P("state", "best"), "bi": P("state", "ind"), "h": 0}}),
        set_(["state", "ckey"], C({"s": LP("f", "s"), "o": LP("f", "o"), "p": LP("f", "p")}, term_key_expr(V("s"), V("o"), V("p")))),
        set_(["state", "F", P("state", "ckey")], P("state", "cur")),
        append(["state", "byP", LP("f", "p")], P("state", "ckey")),
        append(["state", "baseByP", LP("f", "p")], P("state", "ckey")),
    ]))
    for ax in M3["axioms"]:
        if ax["kind"] == "transitive":
            init += gen_edges(ax["args"][0], M3["axioms"])
    round_body = []
    for r in M3["compose"]:
        round_body += gen_compose(r)
    for ax in M3["axioms"]:
        round_body += gen_axiom(ax)
    for sc in M3["schemas"]:
        round_body += gen_schema(sc)
    # merge: strength and independence are the best over proofs; the proof count is recomputed each round
    round_body.append(for_(P("state", "cand"), "c", [
        if_(C({"f": P("state", "F"), "k": LP("c", "key")}, ["has", V("f"), V("k")]),
            [if_(C({"n": LP("c", "r"), "o": P("state", "F", LP("c", "key"), "r")}, ["gt", V("n"), V("o")]),
                 [set_(["state", "F", LP("c", "key"), "r"], LP("c", "r"))]),
             if_(C({"n": LP("c", "i"), "o": P("state", "F", LP("c", "key"), "i")}, ["gt", V("n"), V("o")]),
                 [set_(["state", "F", LP("c", "key"), "i"], LP("c", "i"))]),
             if_(C({"n": LP("c", "h"), "o": P("state", "F", LP("c", "key"), "h")}, ["gt", V("n"), V("o")]),
                 [set_(["state", "F", LP("c", "key"), "h"], LP("c", "h"))])],
            [set_(["state", "F", LP("c", "key")], {"$tpl": {"s": LP("c", "s"), "p": LP("c", "p"), "o": LP("c", "o"), "r": LP("c", "r"),
                                                            "i": LP("c", "i"), "n": 0, "b": 0, "br": -1, "bi": 0, "h": LP("c", "h")}}),
             append(["state", "byP", LP("c", "p")], LP("c", "key"))]),
        set_(["state", "acc", LP("c", "key")],
             C({"a": P("state", "acc"), "k": LP("c", "key"), "n": LP("c", "n")}, ["add", ["get_or", V("a"), V("k"), 0], V("n")])),
    ]))
    round_body.append(for_(C({"a": P("state", "acc")}, ["keys", V("a")]), "ak", [
        set_(["state", "F", LP("ak"), "n"],
             C({"b": P("state", "F", LP("ak"), "b"), "a": P("state", "acc", LP("ak"))}, ["add", V("b"), V("a")]))]))
    round_body += [set_(["state", "cand"], []), set_(["state", "acc"], {}),
                   if_(C({"r": LP("round")}, ["eq", V("r"), SNAP]), [set_(["state", "F6"], P("state", "F"))])]
    return init + [for_(list(range(1, DEEP + 1)), "round", round_body)] + assess_code() + tier_code()


def key_of(s, o, p):
    return C({"s": s, "o": o}, term_key_expr(V("s"), V("o"), p))


def defeat_code():
    code = []
    for ax in M3["axioms"]:
        if ax["kind"] != "defeats":
            continue
        d, t = ax["args"]
        code.append(if_(C({"p": P("state", "cur", "p")}, ["eq", V("p"), t]), [
            set_(["state", "dkey"], key_of(P("state", "cur", "s"), P("state", "cur", "o"), d)),
            if_(C({"f": P("state", "F6"), "k": P("state", "dkey")}, ["has", V("f"), V("k")]), [
                set_(["state", "dr"], P("state", "F6", P("state", "dkey"), "r")),
                if_(C({"dr": P("state", "dr")}, ["ge", V("dr"), 3]), [
                    append(["state", "defs"], {"$tpl": {"d": d, "st": C({"n": P("input", "rankNames"), "r": P("state", "dr")}, ["get", V("n"), V("r")])}}),
                    if_(C({"a": P("state", "dr"), "b": P("state", "bd")}, ["gt", V("a"), V("b")]), [set_(["state", "bd"], P("state", "dr"))])])])]))
    return code


def rank_name(path):
    return C({"n": P("input", "rankNames"), "r": P(*path)}, ["get", V("n"), V("r")])


def assess_code():
    per = [
        set_(["state", "cur"], P("state", "F6", LP("key"))),
        if_(C({"h": P("input", "headline"), "p": P("state", "cur", "p")}, ["has", V("h"), V("p")]), [
            set_(["state", "defs"], []), set_(["state", "bd"], -1)] + defeat_code() + [
            set_(["state", "rr"], P("state", "cur", "r")),
            set_(["state", "cf"], "none"),
            if_(C({"d": P("state", "defs")}, ["gt", ["len", V("d")], 0]), [
                if_(C({"a": P("state", "bd"), "b": P("state", "cur", "r")}, ["gt", V("a"), V("b")]),
                    [set_(["state", "rr"], 0), set_(["state", "cf"], "defeated")],
                    [set_(["state", "rr"], 2), set_(["state", "cf"], "contested")])]),
            set_(["state", "n9"], P("state", "F", LP("key"), "n")),
            set_(["state", "cm"], "exhaustive"),
            if_(C({"a": P("state", "n9"), "b": P("state", "cur", "n"), "h": P("state", "cur", "h")},
                  ["or", ["ne", V("a"), V("b")], ["ge", V("h"), SNAP]]), [set_(["state", "cm"], "bound_reached")]),
            append(["state", "assess"], {"$tpl": {
                "s": P("state", "cur", "s"), "p": P("state", "cur", "p"), "o": P("state", "cur", "o"),
                "status": rank_name(["state", "rr"]), "ps": rank_name(["state", "cur", "r"]), "ind": P("state", "cur", "i"),
                "conflict": P("state", "cf"), "defeaters": P("state", "defs"), "completeness": P("state", "cm"),
                "n6": P("state", "cur", "n"), "n9": P("state", "n9")}})])]
    return [set_(["state", "assess"], []), for_(C({"o": P("state", "F6")}, ["keys", V("o")]), "key", per)]


def tier_code():
    row = [
        # membership of the objective symptom in this set
        set_(["state", "inset"], False),
        for_(P("input", "sets", LP("sn")), "sy", [
            if_(C({"a": LP("sy"), "b": LP("a", "o", "v")}, ["eq", V("a"), V("b")]), [set_(["state", "inset"], True)])]),
        if_(P("state", "inset"), [
            set_(["state", "tier"], "needs_review"),
            if_(C({"a": LP("a", "conflict")}, ["eq", V("a"), "defeated"]), [set_(["state", "tier"], "withheld")],
                [if_(C({"a": LP("a", "conflict")}, ["eq", V("a"), "contested"]), [set_(["state", "tier"], "needs_review")],
                     [if_(C({"a": LP("a", "ps")}, ["eq", V("a"), "supported"]), [set_(["state", "tier"], "supported")])])]),
            set_(["state", "why"], LP("a", "defeaters")),
            if_(C({"t": P("state", "tier")}, ["eq", V("t"), "supported"]), [set_(["state", "why"], [])]),
            append(["state", "tiers"], {"$tpl": {
                "set": LP("sn"), "tier": P("state", "tier"), "p": LP("a", "s", "v"),
                "name": C({"e": P("state", "ent"), "i": LP("a", "s", "v")}, ["get", ["get", V("e"), V("i")], "name"]),
                "sym": LP("a", "o", "v"), "status": LP("a", "status"), "ps": LP("a", "ps"), "ind": LP("a", "ind"),
                "conflict": LP("a", "conflict"), "defeaters": LP("a", "defeaters"), "completeness": LP("a", "completeness"),
                "why": P("state", "why")}})])]
    body = [if_(C({"p": LP("a", "p"), "k": LP("a", "o", "k")}, ["and", ["eq", V("p"), "may_address"], ["eq", V("k"), "c"]]), row)]
    return [for_(C({"s": P("input", "sets")}, ["keys", V("s")]), "sn", [for_(P("state", "assess"), "a", body)]),
            set_(["output"], {"$tpl": {"assessments": P("state", "assess"), "tiers": P("state", "tiers")}})]


def package(d=None):
    global M3
    if d is not None:
        M3 = d
    d = M3
    return {"jaxson": "1.0", "limits": {"steps": 400000000}, "inputSchema": {"type": "any"}, "outputSchema": {"type": "any"},
            "input": {"facts": d["facts"], "entities": d["entities"], "ranks": RANK, "rankNames": RANK_NAMES,
                      "headline": {r["p"]: True for r in d["preds"] if r["headline"]}, "lineageOf": d["lineage"], "sets": d["sets"]},
            "program": build()}


def main():
    pkg = package()
    json.dump(pkg, open(os.path.join(HERE, "data", "m3_pkg.json"), "w"))
    print("package:", len(json.dumps(pkg["program"])), "bytes of program for", len(M3["schemas"]), "schemas,", len(M3["axioms"]), "axioms,", len(M3["compose"]), "compose rules")


if __name__ == "__main__":
    main()
