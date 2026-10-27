#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""SHACL Core versions of chinese-wall and break-glass, checked against Shaxon.

The other SHACL versions in this folder use sh:sparql, so they are SHACL-SPARQL.
These two shapes files use SHACL Core only, to find out whether Core is enough.

    python core_cases.py audit             prove the shapes use only Core terms
    python core_cases.py cases             the packages' case files, through Core
    python core_cases.py random [N] [S]    N random trails per package (default 300),
                                           seed S (default 1), Shaxon as the oracle
    python core_cases.py timing            Core against SHACL-SPARQL on large trails

`random` needs a built shaxonrun: set SHAXONRUN to its path (default: the
shaxonrun on PATH). Validation runs through pyshacl with advanced features off,
so a SPARQL constraint would be ignored rather than run; `audit` is what
guarantees there is none.

A Shaxon refusal (EXECUTION_ERROR) has no SHACL meaning: SHACL reports
violations and does not raise errors. A refusal counts as met by any non-allow
verdict, as in run_cases.py; an allow is counted separately as fail-open.
"""
import json, os, random, subprocess, sys, tempfile, time
from rdflib import BNode, Graph, RDF
from pyshacl import validate
from lift import lift
from run_cases import HERE, SH, SHAXON, load, local

NAMES = ["chinese-wall", "break-glass"]
HELPERS = {"NoConflict"}

# Every term of the SHACL Core vocabulary that these shapes may use. Anything
# else in the sh: namespace (sh:sparql, sh:select, sh:SPARQLConstraint,
# sh:prefixes, sh:declare, sh:rule, sh:target, ...) fails the audit.
CORE = set("""NodeShape PropertyShape targetNode targetClass targetSubjectsOf targetObjectsOf
path inversePath alternativePath zeroOrMorePath oneOrMorePath zeroOrOnePath property node
not and or xone class datatype nodeKind minCount maxCount minExclusive minInclusive
maxExclusive maxInclusive minLength maxLength pattern flags languageIn uniqueLang equals
disjoint lessThan lessThanOrEquals qualifiedValueShape qualifiedMinCount qualifiedMaxCount
qualifiedValueShapesDisjoint closed ignoredProperties hasValue in message severity name
description order group deactivated""".split())


def core_shapes(name):
    return Graph().parse(os.path.join(HERE, name + ".core.shapes.ttl"))


def audit():
    bad_total = 0
    for name in NAMES:
        g = core_shapes(name)
        used = {str(p) for _, p, _ in g if str(p).startswith(str(SH))}
        used |= {str(o) for _, p, o in g if p == RDF.type and str(o).startswith(str(SH))}
        names = {u[len(str(SH)):] for u in used}
        bad = sorted(names - CORE)
        bad_total += len(bad)
        print(f"{name}: {len(names)} sh: terms used, {len(bad)} outside Core {bad}")
    return 1 if bad_total else 0


def core_verdict(name, inp, shapes):
    conforms, rgraph, _ = validate(lift(name, inp), shacl_graph=shapes, advanced=False,
                                   inference="none")
    reasons = set()
    for res in rgraph.subjects(SH.resultSeverity, None):
        src = rgraph.value(res, SH.sourceShape)
        # unnamed (blank) shapes are the request structure's property shapes
        reasons.add("RequestStructure" if src is None or isinstance(src, BNode) else local(src))
    # pyshacl also reports the helper value shape that a sh:node constraint
    # delegates to; a helper is not a constraint of the package, so it is left out
    return conforms, reasons - HELPERS


def sparql_decision(name, inp, shapes):
    """Decision of the existing SHACL-SPARQL version, for comparison."""
    conforms, _, _ = validate(lift(name, inp), shacl_graph=shapes, advanced=True, inference="none")
    return "allow" if conforms else "deny"


def shaxon_verdict(name, pkg, inp):
    """(decision, reasons) from the real Shaxon runner, or ('error', code)."""
    q = dict(pkg)
    q["input"] = inp
    with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
        json.dump(q, f)
    try:
        r = subprocess.run([os.environ.get("SHAXONRUN", "shaxonrun"), f.name],
                           capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    if r.returncode:
        return "error", r.stderr.strip().split(":")[0]
    out = json.loads(r.stdout)["output"]
    return out["decision"], {x["constraintId"] for x in out["findings"] if x.get("constraintId")}


def cases():
    totals = {"agree": 0, "differ": 0, "n/a": 0}
    for name in NAMES:
        pkg = load(os.path.join(SHAXON, name + ".json"))
        shapes = core_shapes(name)
        print(f"== {name} (SHACL Core)")
        for c in load(os.path.join(SHAXON, name + ".cases.json")):
            inp = dict(pkg["input"])
            inp.update(c.get("input", {}))
            conforms, reasons = core_verdict(name, inp, shapes)
            got = "allow" if conforms else "deny"
            expect = c["expect"]
            if "error" in expect:
                if expect["error"].startswith("RESOURCE_ERROR"):
                    status, note = "n/a", "no step budget in SHACL"
                else:
                    status = "agree" if got == "deny" else "differ"
                    note = f"shaxon refuses ({expect['error']}); core says {got} {sorted(reasons)}"
            else:
                want = {r.removeprefix("SHAX_") for r in expect["reasons"]}
                same = got == expect["decision"] and (reasons == want if want or got == "allow" else True)
                status = "agree" if same else "differ"
                note = f"{got} {sorted(reasons)}"
            totals[status] += 1
            if status != "agree":
                print(f"  {status:6} {c['name'][:58]}  -> {note}")
    print(", ".join(f"{k}: {v}" for k, v in totals.items()))
    return 1 if totals["differ"] else 0


def gen_chinese_wall(rng, base):
    companies = list(base["classOf"])
    actors = ["ines", "omar", "pia", "quin"]
    events = [{"actor": rng.choice(actors), "company": rng.choice(companies)}
              for _ in range(rng.randint(0, 14))]
    pool = companies + (["zzz"] if rng.random() < 0.08 else [])
    return dict(base, events=events, request={"actor": rng.choice(actors), "company": rng.choice(pool)})


def gen_break_glass(rng, base):
    actors = ["rhea", "sam", "tobi", "uma"]
    events, ids = [], []
    for _ in range(rng.randint(0, 8)):
        if ids and rng.random() < 0.45:
            events.append({"type": "review", "of": rng.choice(ids), "reviewer": rng.choice(actors)})
        else:
            ids.append("bg%d" % (len(ids) + 1))
            events.append({"type": "breakglass", "id": ids[-1], "actor": rng.choice(actors), "reason": "x"})
    return dict(base, events=events, request={"actor": rng.choice(actors)})


GENERATORS = {"chinese-wall": gen_chinese_wall, "break-glass": gen_break_glass}


def random_test(n, seed):
    bad = 0
    for name in NAMES:
        pkg = load(os.path.join(SHAXON, name + ".json"))
        shapes = core_shapes(name)
        sparql = Graph().parse(os.path.join(HERE, name + ".shapes.ttl"))
        rng = random.Random(seed)
        count = {"agree": 0, "differ": 0, "refused: core denies": 0, "refused: core allows (fail-open)": 0}
        core_vs_sparql = 0
        decisions = {"allow": 0, "deny": 0}
        shown = 0
        for _ in range(n):
            inp = GENERATORS[name](rng, pkg["input"])
            conforms, reasons = core_verdict(name, inp, shapes)
            got = "allow" if conforms else "deny"
            decision, want = shaxon_verdict(name, pkg, inp)
            if sparql_decision(name, inp, sparql) != got:
                core_vs_sparql += 1
            if decision == "error":
                count["refused: core denies" if got == "deny" else "refused: core allows (fail-open)"] += 1
                continue
            decisions[decision] += 1
            if got == decision and (reasons == want if decision == "deny" else True):
                count["agree"] += 1
            else:
                count["differ"] += 1
                if shown < 3:
                    shown += 1
                    print(f"  DIFFER {name}: shaxon {decision} {sorted(want)}, core {got} {sorted(reasons)}\n"
                          f"         input {json.dumps({k: inp[k] for k in ('events', 'request')})[:300]}")
        print(f"{name}: {n} random trails, shaxon decided allow {decisions['allow']}, deny {decisions['deny']}; {count}; "
              f"core and SHACL-SPARQL disagree on {core_vs_sparql}")
        bad += count["differ"]
    return 1 if bad else 0


def big_input(name, pkg, events, rng):
    base = pkg["input"]
    if name == "chinese-wall":
        companies = list(base["classOf"])
        actors = ["u%d" % i for i in range(events // 8 + 3)]
        return dict(base, events=[{"actor": rng.choice(actors), "company": rng.choice(companies)}
                                  for _ in range(events)],
                    request={"actor": actors[0], "company": companies[0]})
    actors = ["u%d" % i for i in range(events // 6 + 3)]
    ev, ids = [], []
    for _ in range(events):
        if ids and rng.random() < 0.5:
            ev.append({"type": "review", "of": rng.choice(ids), "reviewer": rng.choice(actors)})
        else:
            ids.append("bg%d" % len(ids))
            ev.append({"type": "breakglass", "id": ids[-1], "actor": rng.choice(actors), "reason": "x"})
    return dict(base, events=ev, request={"actor": actors[0]})


def best_ms(fn, runs=3):
    t = []
    for _ in range(runs):
        t0 = time.perf_counter()
        fn()
        t.append((time.perf_counter() - t0) * 1000)
    return min(t)


def timing():
    print(f"{'package':14} {'events':>7} {'SHACL-SPARQL':>14} {'SHACL Core':>12}   (pyshacl, validation only, best of 3)")
    for name in NAMES:
        pkg = load(os.path.join(SHAXON, name + ".json"))
        sparql = Graph().parse(os.path.join(HERE, name + ".shapes.ttl"))
        core = core_shapes(name)
        for events in (250, 1000, 4000):
            data = lift(name, big_input(name, pkg, events, random.Random(7)))
            a = best_ms(lambda: validate(data, shacl_graph=sparql, advanced=True, inference="none"))
            b = best_ms(lambda: validate(data, shacl_graph=core, advanced=False, inference="none"))
            print(f"{name:14} {events:>7} {a:>11.0f} ms {b:>9.0f} ms")
    return 0


def main():
    mode = sys.argv[1] if len(sys.argv) > 1 else ""
    if mode == "audit":
        sys.exit(audit())
    if mode == "cases":
        sys.exit(cases())
    if mode == "random":
        sys.exit(random_test(int(sys.argv[2]) if len(sys.argv) > 2 else 300,
                             int(sys.argv[3]) if len(sys.argv) > 3 else 1))
    if mode == "timing":
        sys.exit(timing())
    sys.stderr.write(__doc__)
    sys.exit(2)


if __name__ == "__main__":
    main()
