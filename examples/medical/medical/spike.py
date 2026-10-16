# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
#!/usr/bin/env python3
"""Spike: port the medical coordinator's recommendation pipeline to a Jaxson package.

1. convert the tosid/5 facts of the four Prolog knowledge bases to JSON;
2. generate the Jaxson package (the inference is Jaxson code over that data);
3. run it, and compare with the output of the original Prolog system.
"""
import json
import re
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))

SRC = os.path.join(HERE, "..", "sources", "medical") + os.sep
KBS = ["biochemical.pl", "chemistry.pl", "pharma.pl", "medicine.pl"]


# ---------------------------------------------------------------- Prolog facts -> JSON
def tokenize(s):
    toks, i, n = [], 0, len(s)
    while i < n:
        c = s[i]
        if c.isspace():
            i += 1
        elif c == "%":
            while i < n and s[i] != "\n":
                i += 1
        elif c == "'":
            j = i + 1
            buf = []
            while True:
                if s[j] == "'" and j + 1 < n and s[j + 1] == "'":
                    buf.append("'")
                    j += 2
                elif s[j] == "'":
                    break
                else:
                    buf.append(s[j])
                    j += 1
            toks.append(("atom", "".join(buf)))
            i = j + 1
        elif c.isdigit() or (c == "-" and i + 1 < n and s[i + 1].isdigit()):
            m = re.compile(r"-?\d+(\.\d+)?").match(s, i)
            txt = m.group(0)
            toks.append(("num", float(txt) if m.group(1) else int(txt)))
            i = m.end()
        elif c.isalpha() or c == "_":
            m = re.compile(r"[A-Za-z0-9_]+").match(s, i)
            toks.append(("atom", m.group(0)))
            i = m.end()
        elif c in "()[],.":
            toks.append((c, c))
            i += 1
        else:
            toks.append(("op", c))
            i += 1
    return toks


def parse_term(t, k):
    kind, val = t[k]
    if kind in ("atom", "num"):
        if kind == "atom" and k + 1 < len(t) and t[k + 1][0] == "(":
            args, k = [], k + 2
            while True:
                a, k = parse_term(t, k)
                args.append(a)
                if t[k][0] == ",":
                    k += 1
                    continue
                assert t[k][0] == ")", t[k]
                return ("compound", val, args), k + 1
        return (kind, val), k + 1
    if kind == "[":
        items, k = [], k + 1
        if t[k][0] == "]":
            return ("list", []), k + 1
        while True:
            a, k = parse_term(t, k)
            items.append(a)
            if t[k][0] == ",":
                k += 1
                continue
            assert t[k][0] == "]", t[k]
            return ("list", items), k + 1
    raise ValueError("unexpected token %r" % (t[k],))


def plain(term):
    kind = term[0]
    if kind in ("atom", "num"):
        return term[1]
    if kind == "list":
        return [plain(x) for x in term[1]]
    raise ValueError(term)


def load_entities():
    ents = []
    for f in KBS:
        toks = tokenize(open(SRC + f, encoding="utf-8").read())
        stmts, cur = [], []
        for tk in toks:
            if tk[0] == ".":
                stmts.append(cur)
                cur = []
            else:
                cur.append(tk)
        for st in stmts:
            if not st or st[0] != ("atom", "tosid"):
                continue
            term, _ = parse_term(st, 0)
            a = term[2]
            props = {}
            for p in a[4][1]:
                assert p[0] == "compound", p
                args = [plain(x) for x in p[2]]
                props[p[1]] = args[0] if len(args) == 1 else args
            ents.append({"taxonomy": plain(a[0]), "netmask": plain(a[1]), "id": plain(a[2]),
                         "name": plain(a[3]), "domain": props["domain"], "props": props})
    return ents


# ---------------------------------------------------------------- the helper facts of the coordinator, as data
TABLES = {
    "symptomIndication": [("headache", "pain"), ("head_pain", "pain"), ("muscle_pain", "pain"), ("body_aches", "pain"),
                          ("irregular_heartbeat", "magnesium_deficiency"), ("sleep_disorders", "mood_support"),
                          ("depression", "mood_support")],
    "elementCompound": [("iron", "ferrous_sulfate"), ("magnesium", "magnesium_oxide")],
    "functionIndication": [("oxygen_transport", "iron_deficiency"), ("calcium_regulation", "bone_health"),
                           ("dna_synthesis", "b12_deficiency")],
    "vitaminIngredient": [("Vitamin D3", "cholecalciferol"), ("Vitamin B12", "cyanocobalamin")],
}
INTERACTIONS = [("ferrous_sulfate", "ascorbic_acid", "synergistic", "high"), ("calcium", "ferrous_sulfate", "antagonistic", "moderate"),
                ("acetylsalicylic_acid", "ibuprofen", "antagonistic", "moderate"), ("acetylsalicylic_acid", "acetaminophen", "neutral", "low")]
CASES = [("case1", ["fatigue", "shortness_of_breath", "pale_skin"]), ("case2", ["headache", "fever"]),
         ("case3", ["depression", "sleep_disorders", "fatigue"]), ("case4", ["muscle_cramps", "irregular_heartbeat"]),
         ("case5", ["fatigue", "memory_problems", "weakness"])]


def make_input(ents):
    def rows(keys, tab):
        return [dict(zip(keys, r)) for r in tab]
    return {
        "entities": ents,
        "symptomIndication": rows(["symptom", "indication"], TABLES["symptomIndication"]),
        "elementCompound": rows(["element", "ingredient"], TABLES["elementCompound"]),
        "functionIndication": rows(["function", "indication"], TABLES["functionIndication"]),
        "vitaminIngredient": rows(["entity", "ingredient"], TABLES["vitaminIngredient"]),
        "interactions": rows(["a", "b", "type", "significance"], INTERACTIONS),
        "cases": [{"id": i, "symptoms": s} for i, s in CASES],
    }


# ---------------------------------------------------------------- a tiny builder for Jaxson programs
class Gen:
    def __init__(self):
        self.n = 0

    def u(self, p):
        self.n += 1
        return "%s%d" % (p, self.n)


def P(*segs):
    return {"$path": list(segs)}


def LP(var, *rest):
    return P("local", var, *rest)


def V(n):
    return {"$v": n}


def C(withs, expr):
    return {"$compute": {"with": withs, "expr": expr}}


def eq(a, b):
    return C({"a": a, "b": b}, ["eq", V("a"), V("b")])


def has(obj, key):
    return C({"o": obj}, ["has", V("o"), key])


def set_(path, value):
    return {"op": "set", "path": path, "value": value}


def append(path, value):
    return {"op": "append", "path": path, "value": value}


def for_(in_, var, body, index=None):
    d = {"op": "for", "in": in_, "as": var, "do": body}
    if index:
        d["index"] = index
    return d


def if_(cond, then, els=None):
    d = {"op": "if", "cond": cond, "then": then}
    if els:
        d["else"] = els
    return d


def rec(sym, treat, mech, typ, conf):
    return {"$tpl": {"symptom": sym, "treatment": treat, "mechanism": mech, "type": typ, "confidence": conf}}


def build_program():
    g = Gen()

    def member(value, list_operand, then):
        """if value is in the list, run `then` (a scan: Jaxson has no membership operator)."""
        flag, x = g.u("f"), g.u("x")
        return [set_(["state", flag], False),
                for_(list_operand, x, [if_(eq(LP(x), value), [set_(["state", flag], True)])]),
                if_(P("state", flag), then)]

    prog = [set_(["output"], [])]

    def mechanisms(s, p):
        """the three alternatives of symptom_relieved_by_drug's mechanism, in order, each yielding solutions"""
        out = []
        ce1, ce2 = g.u("ce"), g.u("ce")
        # (i) the chemistry entity *named* as the active ingredient, with a mechanism
        out.append(for_(P("input", "entities"), ce1, [
            if_(eq(LP(ce1, "name"), LP(p, "props", "active_ingredient")), [
                if_(eq(LP(ce1, "domain"), "chemistry"), [
                    if_(has(LP(ce1, "props"), "mechanism"),
                        [append(["state", "recs"], rec(LP(s), LP(p, "name"), LP(ce1, "props", "mechanism"), "symptomatic_relief", "moderate"))])])])]))
        # (ii) any chemistry entity whose chemical_name is the active ingredient, with a mechanism
        out.append(for_(P("input", "entities"), ce2, [
            if_(eq(LP(ce2, "domain"), "chemistry"), [
                if_(has(LP(ce2, "props"), "chemical_name"), [
                    if_(eq(LP(ce2, "props", "chemical_name"), LP(p, "props", "active_ingredient")), [
                        if_(has(LP(ce2, "props"), "mechanism"),
                            [append(["state", "recs"], rec(LP(s), LP(p, "name"), LP(ce2, "props", "mechanism"), "symptomatic_relief", "moderate"))])])])])]))
        # (iii) the fallback, always a solution
        out.append(append(["state", "recs"], rec(LP(s), LP(p, "name"), "symptomatic_relief", "symptomatic_relief", "moderate")))
        return out

    def per_symptom(s):
        ins = []
        # ---- deficiency path: symptom_suggests_deficiency + deficiency_treated_by_supplement
        e, p = g.u("e"), g.u("p")
        ins.append(for_(P("input", "entities"), e, [
            if_(eq(LP(e, "domain"), "biochemical"), [
                if_(has(LP(e, "props"), "deficiency_symptoms"),
                    member(LP(s), LP(e, "props", "deficiency_symptoms"), [
                        for_(P("input", "entities"), p, [
                            if_(eq(LP(p, "domain"), "pharma"), [
                                if_(has(LP(p, "props"), "indication"), pharma_alternatives(s, e, p))])])]))])]))
        # ---- symptomatic path: symptom_relieved_by_drug
        p2 = g.u("p")
        ins.append(for_(P("input", "entities"), p2, [
            if_(eq(LP(p2, "domain"), "pharma"), [
                if_(has(LP(p2, "props"), "indication"), [
                    if_(has(LP(p2, "props"), "active_ingredient"), symptomatic(s, p2))])])]))
        return ins

    def pharma_alternatives(s, e, p):
        out = []
        # (a) element replacement
        if_a = []
        ec = g.u("ec")
        if_a.append(for_(P("input", "elementCompound"), ec, [
            if_(eq(LP(ec, "element"), LP(e, "props", "contains_element")), [
                if_(eq(LP(ec, "ingredient"), LP(p, "props", "active_ingredient")),
                    [append(["state", "recs"], rec(LP(s), LP(p, "name"), "element_replacement", "deficiency_correction", "high"))])])]))
        out.append(if_(has(LP(e, "props"), "contains_element"), [if_(has(LP(p, "props"), "active_ingredient"), if_a)]))
        # (b)(c) vitamin replacement
        vi = g.u("vi")
        out.append(if_(has(LP(p, "props"), "active_ingredient"), [for_(P("input", "vitaminIngredient"), vi, [
            if_(eq(LP(vi, "entity"), LP(e, "name")), [
                if_(eq(LP(vi, "ingredient"), LP(p, "props", "active_ingredient")),
                    [append(["state", "recs"], rec(LP(s), LP(p, "name"), "vitamin_replacement", "deficiency_correction", "high"))])])])]))
        # (d) functional support
        fi = g.u("fi")
        out.append(if_(has(LP(e, "props"), "function"), [for_(P("input", "functionIndication"), fi, [
            if_(eq(LP(fi, "function"), LP(e, "props", "function")),
                member(LP(fi, "indication"), LP(p, "props", "indication"),
                       [append(["state", "recs"], rec(LP(s), LP(p, "name"), "functional_support", "deficiency_correction", "high"))]))])]))
        return out

    def symptomatic(s, p):
        out = []
        # clause 1: the symptom is itself an indication
        out.extend(member(LP(s), LP(p, "props", "indication"), mechanisms(s, p)))
        # clause 2: symptom_matches_indication
        si = g.u("si")
        out.append(for_(P("input", "symptomIndication"), si, [
            if_(eq(LP(si, "symptom"), LP(s)),
                member(LP(si, "indication"), LP(p, "props", "indication"), mechanisms(s, p)))]))
        return out

    # ---- per case
    case_body = [set_(["state", "recs"], []), set_(["state", "clean"], []), set_(["state", "treat"], []),
                 set_(["state", "inter"], []), set_(["state", "inter2"], []), set_(["state", "cost"], 0)]
    sv = g.u("s")
    case_body.append(for_(LP("c", "symptoms"), sv, per_symptom(sv)))
    # remove_exact_duplicates keeps the LAST occurrence of each rec
    i1, r1, j1, r2, dup = "i1", "r1", "j1", "r2", "dup"
    case_body.append(for_(P("state", "recs"), r1, [
        set_(["state", "dup"], False),
        for_(P("state", "recs"), r2, [
            if_(C({"j": LP("j1"), "i": LP("i1"), "a": LP(r1), "b": LP(r2)},
                  ["and", ["gt", V("j"), V("i")], ["eq", V("a"), V("b")]]), [set_(["state", "dup"], True)])], index="j1"),
        if_(C({"d": P("state", "dup")}, ["not", V("d")]), [append(["state", "clean"], LP(r1))])], index="i1"))
    # unique treatments (list_to_set keeps first occurrences)
    cr = g.u("cr")
    case_body.append(for_(P("state", "clean"), cr, member_not(g, LP(cr, "treatment"), ["state", "treat"],
                                                              [append(["state", "treat"], LP(cr, "treatment"))])))
    # interactions between recommended drugs
    d1, d2, e1, e2, ii = "d1", "d2", "e1", "e2", "ii"
    case_body.append(for_(P("state", "treat"), d1, [for_(P("state", "treat"), d2, [
        if_(C({"a": LP(d1), "b": LP(d2)}, ["ne", V("a"), V("b")]), [
            for_(P("input", "entities"), e1, [
                if_(eq(LP(e1, "name"), LP(d1)), [if_(has(LP(e1, "props"), "active_ingredient"), [
                    for_(P("input", "entities"), e2, [
                        if_(eq(LP(e2, "name"), LP(d2)), [if_(has(LP(e2, "props"), "active_ingredient"), [
                            for_(P("input", "interactions"), ii, [
                                if_(C({"ia": LP(ii, "a"), "ib": LP(ii, "b"), "x": LP(e1, "props", "active_ingredient"),
                                       "y": LP(e2, "props", "active_ingredient")},
                                      ["or", ["and", ["eq", V("ia"), V("x")], ["eq", V("ib"), V("y")]],
                                       ["and", ["eq", V("ia"), V("y")], ["eq", V("ib"), V("x")]]]), [
                                    append(["state", "inter"], {"$tpl": {"a": LP(d1), "b": LP(d2), "type": LP(ii, "type"),
                                                                         "significance": LP(ii, "significance")}})])])])])])])])])])])]))
    # remove_duplicate_interactions: drop an interaction if its mirror comes later
    k1, q1, k2, q2, dup2 = "k1", "q1", "k2", "q2", "dup2"
    case_body.append(for_(P("state", "inter"), q1, [
        set_(["state", "dup2"], False),
        for_(P("state", "inter"), q2, [
            if_(C({"j": LP("k2"), "i": LP("k1"), "a1": LP(q1, "a"), "b1": LP(q1, "b"), "a2": LP(q2, "a"), "b2": LP(q2, "b"),
                   "t1": LP(q1, "type"), "t2": LP(q2, "type"), "s1": LP(q1, "significance"), "s2": LP(q2, "significance")},
                  ["and", ["gt", V("j"), V("i")], ["eq", V("a1"), V("b2")], ["eq", V("b1"), V("a2")],
                   ["eq", V("t1"), V("t2")], ["eq", V("s1"), V("s2")]]), [set_(["state", "dup2"], True)])], index="k2"),
        if_(C({"d": P("state", "dup2")}, ["not", V("d")]), [append(["state", "inter2"], LP(q1))])], index="k1"))
    # cost: summed over the recommendations (not the unique treatments), tablets only
    cr2, ce = g.u("cr"), g.u("ce")
    case_body.append(for_(P("state", "clean"), cr2, [for_(P("input", "entities"), ce, [
        if_(eq(LP(ce, "name"), LP(cr2, "treatment")), [if_(has(LP(ce, "props"), "cost_per_tablet"), [
            set_(["state", "cost"], C({"t": P("state", "cost"), "p": LP(ce, "props", "cost_per_tablet")}, ["add", V("t"), V("p")]))])])])]))
    case_body.append(append(["output"], {"$tpl": {
        "id": LP("c", "id"), "symptoms": LP("c", "symptoms"), "recs": P("state", "clean"),
        "interactions": P("state", "inter2"),
        "cost": C({"t": P("state", "cost")}, ["round", V("t"), 2, "half_up"])}}))
    prog.append(for_(P("input", "cases"), "c", case_body))
    return prog


def member_not(g, value, list_path, then):
    """if value is NOT in the array at list_path, run `then`"""
    flag, x = g.u("f"), g.u("x")
    return [set_(["state", flag], False),
            for_(P(*list_path), x, [if_(eq(LP(x), value), [set_(["state", flag], True)])]),
            if_(C({"f": P("state", flag)}, ["not", V("f")]), then)]


def main():
    ents = load_entities()
    by = {}
    for e in ents:
        by[e["domain"]] = by.get(e["domain"], 0) + 1
    print("entities:", len(ents), by)
    pkg = {"jaxson": "1.0", "limits": {"steps": 5000000}, "inputSchema": {"type": "any"}, "outputSchema": {"type": "any"},
           "input": make_input(ents), "program": build_program()}
    json.dump(pkg, open(os.path.join(HERE, "medical_pkg.json"), "w"), indent=1)
    print("package written:", len(json.dumps(pkg)), "bytes,", len(json.dumps(pkg["program"])), "bytes of program")


if __name__ == "__main__":
    main()
