# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import json, os, sys
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import spike

ents = spike.load_entities()
kb = {}
for e in ents:
    kb.setdefault(e["domain"], []).append(e)
P, LP, C = spike.P, spike.LP, spike.C
pkg = {
    "shaxon": "3.1",
    "limits": {"steps": 200000, "maxShapeDepth": 8},
    "indices": {
        "ingredients": {"source": P("state", "ingredients"), "key": LP("item"), "multi": True},
        "compoundElements": {"source": P("input", "elementCompound"), "key": LP("item", "element"), "multi": True},
    },
    "shapes": {
        "PharmaProps": {
            "kind": "object", "closed": False,
            "required": ["active_ingredient", "indication", "contraindications", "max_daily_dose_mg", "availability"],
            "fields": {
                "active_ingredient": {"kind": "reference", "index": "ingredients"},
                "indication": {"kind": "array", "items": {"kind": "string"}},
                "contraindications": {"kind": "array", "items": {"kind": "string"}},
                "max_daily_dose_mg": {"kind": "number", "min": 1},
                "availability": {"kind": "string", "enum": ["otc", "prescription"]},
            },
        },
        "PharmaEntity": {
            "kind": "object", "closed": False, "required": ["id", "name", "domain", "props"],
            "fields": {"domain": {"kind": "string", "enum": ["pharma"]}, "props": {"shape": "PharmaProps"}},
        },
        "BiochemProps": {
            "kind": "object", "closed": False,
            "fields": {
                "contains_element": {"kind": "reference", "index": "compoundElements"},
                "deficiency_symptoms": {"kind": "array", "items": {"kind": "string"}},
            },
        },
        "BiochemEntity": {
            "kind": "object", "closed": False, "required": ["id", "name", "domain", "props"],
            "fields": {"domain": {"kind": "string", "enum": ["biochemical"]}, "props": {"shape": "BiochemProps"}},
        },
    },
    "validate": [
        {"target": {"$each": ["input", "kb", d]}, "unique": {"field": f, "id": "%s_%s_UNIQUE" % (d.upper(), f.upper())}, "mode": "report"}
        for d in ("biochemical", "chemistry", "pharma", "medicine") for f in ("id", "name")
    ],
    "program": [
        {"op": "set", "path": ["state", "ingredients"], "value": []},
        {"op": "for", "in": P("input", "kb", "chemistry"), "as": "c", "do": [
            {"op": "append", "path": ["state", "ingredients"], "value": LP("c", "name")},
            {"op": "if", "cond": C({"o": LP("c", "props")}, ["has", {"$v": "o"}, "chemical_name"]),
             "then": [{"op": "append", "path": ["state", "ingredients"], "value": LP("c", "props", "chemical_name")}]},
        ]},
        {"op": "for", "in": P("input", "kb", "pharma"), "as": "p", "index": "pi", "do": [
            {"op": "check", "target": P("input", "kb", "pharma", LP("pi")), "shape": "PharmaEntity", "mode": "report"}]},
        {"op": "for", "in": P("input", "kb", "biochemical"), "as": "b", "index": "bi", "do": [
            {"op": "check", "target": P("input", "kb", "biochemical", LP("bi")), "shape": "BiochemEntity", "mode": "report"}]},
        {"op": "set", "path": ["output"], "value": {"$lit": "kb checked"}},
    ],
    "input": {"kb": kb, "elementCompound": spike.make_input(ents)["elementCompound"]},
}
json.dump(pkg, open(os.path.join(HERE, "kb_shaxon.json"), "w"))
print("kb package written:", {k: len(v) for k, v in kb.items()})
