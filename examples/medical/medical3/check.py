# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))

def akey(a):
    return (a["s"]["k"], a["s"]["v"], a["p"], a["o"]["k"], a["o"]["v"], a["status"], a["ps"], a["ind"], a["conflict"],
            tuple(sorted((x["d"], x["st"]) for x in a["defeaters"])), a["completeness"], a["n6"], a["n9"])

def tkey(r):
    return (r["set"], r["tier"], r["p"], r["name"], r["sym"], r["status"], r["ps"], r["ind"], r["conflict"],
            tuple(sorted((x["d"], x["st"]) for x in r["defeaters"])), r["completeness"],
            tuple(sorted((x["d"], x["st"]) for x in r["why"])))

def compare(exp, got, label=""):
    EA, GA = {akey(a) for a in exp["assessments"]}, {akey(a) for a in got["assessments"]}
    ET, GT = {tkey(r) for r in exp["tiers"]}, {tkey(r) for r in got["tiers"]}
    ok = EA == GA and ET == GT
    print("%s assessments %d vs %d: %s | tier rows %d vs %d: %s" % (label, len(GA), len(EA), "SAME" if EA == GA else "DIFFER", len(GT), len(ET), "SAME" if ET == GT else "DIFFER"))
    if EA != GA:
        for k in sorted(GA - EA)[:3]: print("   jaxson only:", k)
        for k in sorted(EA - GA)[:3]: print("   prolog only:", k)
    if ET != GT:
        for k in sorted(GT - ET)[:3]: print("   jaxson only (tier):", k)
        for k in sorted(ET - GT)[:3]: print("   prolog only (tier):", k)
    return ok

if __name__ == "__main__":
    exp = json.load(open(os.path.join(HERE, "data", "m3.json"))); got = json.load(open(os.path.join(HERE, "data", "m3_out.json")))
    raise SystemExit(0 if compare(exp, got, "shipped") else 1)
