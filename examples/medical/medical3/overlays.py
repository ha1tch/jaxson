# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
#!/usr/bin/env python3
"""Knowledge-only overlays checked against medical3's engine, and the contract properties
(docs/CONTRACT.md C4, C5, C6) checked on the Jaxson evaluator."""
import copy
import json
import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
JAXRUN = os.environ.get("JAXRUN", "jaxrun")
sys.path.insert(0, HERE)
import check  # noqa: E402
import gen  # noqa: E402

SRC = os.path.join(HERE, "..", "sources", "medical3")
lib = open(os.path.join(HERE, "export.pl")).read()
lib = re.sub(r"% BEGIN-LOAD.*?% END-LOAD\n", "", lib, flags=re.S).replace(":- initialization((main, halt)).\n", "")
open(os.path.join(HERE, "export_lib.pl"), "w").write(lib)

OVERLAYS = {
    "defeat": """
:- assertz(bridge('Ibuprofen 200mg Capsule', not_indicated_for, concept(headache), [source(curated)])).
:- assertz(bridge('Iron Supplement 65mg', not_indicated_for, concept(fatigue), [source(curated)])).
""",
    "cycle": """
:- assertz(bridge('Tryptophan', precursor_of, 'Tyrosine', [source(curated)])).
:- assertz(bridge('Tyrosine', precursor_of, 'Tryptophan', [source(curated)])).
:- assertz(tosid('10', 'E4', 'TRP-TAB-500:TRP-AMI-T01', 'Tryptophan 500mg Tablet',
     [type(pharmaceutical), domain(pharma), active_ingredient('Tryptophan'), indication([sleep_disorders])])).
""",
    "newknowledge": """
:- assertz(rel_pred(supplements, [kind(derived), polarity(positive), headline(true), doc("Product supplies a deficient entity")])).
:- assertz(schema(supplement_for_deficiency, rel(P, supplements, E),
     [rel(E, deficiency_manifests_as, _), rel(P, provides, E), guard(is_product(P))], conditional)).
:- assertz(rel_axiom(defeats(inhibits, supplements))).
:- assertz(bridge('Calcium', inhibits, 'Hemoglobin', [source(curated)])).
:- assertz(bridge('Iron Supplement 65mg', not_indicated_for, concept(anemia), [source(curated)])).
""",
    "multisource": """
:- assertz(bridge('Ferrous Sulfate', provides, 'Iron', [source(hypothesis)])).
:- assertz(bridge('Ascorbic Acid', enhances, 'Iron', [source(curated), lineage(trial_a)])).
:- assertz(bridge('Ascorbic Acid', enhances, 'Iron', [source(hypothesis), lineage(trial_b)])).
""",
    "depth": """
:- forall(between(1, 10, I), ( format(atom(Id), 'CH-~w:CH-ENT-~w', [I, I]), format(atom(Nm), 'Chain Entity ~w', [I]),
        assertz(tosid('10', 'E5', Id, Nm, [type(amino_acid), domain(biochemical)])) )).
:- forall(between(1, 9, I), ( J is I + 1, format(atom(A), 'Chain Entity ~w', [I]), format(atom(B), 'Chain Entity ~w', [J]),
        assertz(bridge(A, precursor_of, B, [source(curated)])) )).
:- assertz(tosid('10', 'E6', 'CH-PROD:CH-PRD-01', 'Chain Product 100mg',
     [type(pharmaceutical), domain(pharma), active_ingredient('Chain Entity 1'), indication([sleep])])).
""",
}


def run_prolog(name, extra):
    path = os.path.join(HERE, "overlay_%s.pl" % name)
    open(path, "w").write(":- consult('%s/load.pl').\n:- consult('%s').\n%s\n:- initialization((ingest_all, main, halt)).\n"
                          % (SRC, os.path.join(HERE, "export_lib.pl"), extra))
    r = subprocess.run(["swipl", "-q", path], cwd=SRC, capture_output=True, text=True, stdin=subprocess.DEVNULL, timeout=300)
    return json.loads(r.stdout[r.stdout.find('{"facts"'):])


def run_jaxson(d):
    pkg = gen.package(d)
    p = os.path.join(HERE, "ov_pkg.json")
    json.dump(pkg, open(p, "w"))
    r = subprocess.run([JAXRUN, p], capture_output=True, text=True, timeout=600)
    return (json.loads(r.stdout), "") if r.returncode == 0 else (None, (r.stdout + r.stderr)[:300])


def main():
    base = json.load(open(os.path.join(HERE, "data", "m3.json")))
    for name, extra in OVERLAYS.items():
        d = run_prolog(name, extra)
        got, err = run_jaxson(d)
        if got is None:
            print("%-13s jaxson failed: %s" % (name, err))
            continue
        cm = {}
        for a in d["assessments"]:
            cm[a["completeness"]] = cm.get(a["completeness"], 0) + 1
        print("%-13s %d conclusions; prolog completeness %s" % (name, len(d["assessments"]), cm))
        check.compare(d, got, "   ")
    # ---- contract properties on the Jaxson evaluator (shipped knowledge)
    base_pkg_facts = base["facts"]
    key = lambda a: (a["s"]["k"], a["s"]["v"], a["p"], a["o"]["k"], a["o"]["v"])
    ref, _ = run_jaxson(base)
    ref_status = {key(a): a["status"] for a in ref["assessments"]}
    again, _ = run_jaxson(base)
    print("C6 determinism: two runs identical:", json.dumps(ref, sort_keys=True) == json.dumps(again, sort_keys=True))
    created = 0
    for i in range(len(base_pkg_facts)):
        d = copy.deepcopy(base)
        del d["facts"][i]
        got, _ = run_jaxson(d)
        created += sum(1 for a in got["assessments"] if key(a) not in ref_status)
    print("C5 removing any one of %d facts never creates a conclusion: %s (%d created)" % (len(base_pkg_facts), created == 0, created))
    rank = {"refuted": 0, "unknown": 1, "conflicted": 2, "conditional": 3, "supported": 4}
    worse, tested = 0, 0
    for a in ref["assessments"]:
        if a["p"] != "may_address":
            continue
        d = copy.deepcopy(base)
        d["facts"].append({"s": a["s"], "p": "not_indicated_for", "o": a["o"], "ev": [{"src": "curated", "strength": "conditional", "lineage": []}]})
        got, _ = run_jaxson(d)
        new = {key(x): x["status"] for x in got["assessments"]}
        tested += 1
        if rank[new[key(a)]] > rank[ref_status[key(a)]]:
            worse += 1
    print("C4 adding a defeater never improves a status: %s (%d conclusions tested, %d improved)" % (worse == 0, tested, worse))


if __name__ == "__main__":
    main()
