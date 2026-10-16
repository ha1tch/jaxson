# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Compare the Jaxson program's output with the original Prolog system's, case by case, in order."""
import json
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))
ref = open(os.path.join(HERE, "expected", "prolog_reference_output.txt"), encoding="utf-8", errors="replace").read()
blocks = re.split(r"=== TREATMENT REPORT FOR (case\d+) ===", ref)
expected = {}
for k in range(1, len(blocks), 2):
    cid, body = blocks[k], blocks[k + 1]
    recs = [{"treatment": m.group(1), "symptom": m.group(2), "mechanism": m.group(3), "type": m.group(4), "confidence": m.group(5)}
            for m in re.finditer(r"- (.+)\n  For: (.+)\n  Mechanism: (.+)\n  Type: (\S+) \| Confidence: (\S+)", body)]
    inter = [{"a": m.group(1), "b": m.group(2), "type": m.group(3), "significance": m.group(4)}
             for m in re.finditer(r"- (.+?) \+ (.+?): (\S+) \((\S+) significance\)", body)]
    cost = re.search(r"ESTIMATED DAILY COST: \$([\d.]+)", body)
    expected[cid] = {"recs": recs, "interactions": inter, "cost": float(cost.group(1))}
got = {c["id"]: c for c in json.load(open(os.path.join(HERE, "run_out.json")))}
allok = True
for cid in sorted(expected):
    e, g = expected[cid], got[cid]
    grecs = [{k: r[k] for k in ("treatment", "symptom", "mechanism", "type", "confidence")} for r in g["recs"]]
    gint = [{k: r[k] for k in ("a", "b", "type", "significance")} for r in g["interactions"]]
    ok = grecs == e["recs"] and gint == e["interactions"] and abs(g["cost"] - e["cost"]) < 1e-9
    allok &= ok
    print("%s: %d recommendations, %d interactions, cost %s: %s" % (cid, len(grecs), len(gint), g["cost"], "SAME" if ok else "DIFFER"))
print("ALL SAME" if allok else "SOME DIFFER")
raise SystemExit(0 if allok else 1)
