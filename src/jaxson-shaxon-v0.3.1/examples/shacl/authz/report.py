#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""Merge the *.jsonl files that scale.py (via bench.sh) wrote into one report.

    python report.py [DIR]          # DIR defaults to ./results; writes DIR/REPORT.md and prints it

For each example it prints the end-to-end table and the engine-only table, with
how much faster or slower Shaxon is than each SHACL engine. A cell that the
time budget skipped is shown as "skipped". If a file holds several runs of the
same cell, the last one wins. Nothing here times anything.
"""
import glob, json, os, sys

ORDER = ["rolling-quota", "chinese-wall", "four-eyes-release", "delegation-chain", "break-glass"]


def load(d):
    cells, hosts = {}, []
    for path in sorted(glob.glob(os.path.join(d, "*.jsonl"))):
        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                r = json.loads(line)
                if r.get("kind") == "host":
                    h = {k: v for k, v in r.items() if k not in ("kind", "when", "profile", "examples", "sizes", "budget_s", "repeat", "engine_runs")}
                    if h not in hosts:
                        hosts.append(h)
                    continue
                cells[(r["profile"], r["example"], r["events"], r["engine"])] = r
    return cells, hosts


def num(ms):
    if ms is None:
        return "skipped"
    if ms < 1:
        return "%.2f" % ms
    if ms < 10:
        return "%.1f" % ms
    return "{:,.0f}".format(ms)


def versus(sx, other):
    if sx is None or other is None:
        return "-"
    q = other / sx
    if q >= 1:
        return ("%.0fx faster" % q) if q >= 10 else ("%.1fx faster" % q)
    q = 1 / q
    return ("%.0fx slower" % q) if q >= 10 else ("%.1fx slower" % q)


def table(cells, profile, example, sizes):
    hdr = "| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |"
    if profile == "engine":
        hdr = "| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |"
    out = [hdr, "|" + "---|" * (hdr.count("|") - 1)]
    for n in sizes:
        g = lambda e: cells.get((profile, example, n, e))
        sx, py, je = g("shaxon"), g("pyshacl"), g("jena")
        if not (sx or py or je):
            continue
        ms = lambda r: None if r is None else r.get("ms")
        row = "| %s | %s | %s | %s | %s | %s | %s |" % (
            "{:,}".format(n), (sx or {}).get("steps", "-") if sx else "-", num(ms(sx)) if sx else "-",
            num(ms(py)) if py else "-", num(ms(je)) if je else "-", versus(ms(sx), ms(py)), versus(ms(sx), ms(je)))
        if profile == "engine":
            lift = (py or {}).get("lift_ms")
            row = row + " %s |" % (num(lift) if lift is not None else "-")
        out.append(row)
    return out


def verdict_check(cells):
    bad = []
    for (profile, ex, n, eng), r in cells.items():
        if eng == "shaxon" or "allow" not in r:
            continue
        sx = cells.get((profile, ex, n, "shaxon"))
        if sx and "allow" in sx and sx["allow"] != r["allow"]:
            bad.append("%s %s %d: shaxon %s, %s %s" % (profile, ex, n, sx["allow"], eng, r["allow"]))
        if profile == "end-to-end" and sx and "reasons" in sx and "reasons" in r and sx["reasons"] != r["reasons"]:
            bad.append("%s %d reasons differ: shaxon %s, %s %s" % (ex, n, sx["reasons"], eng, r["reasons"]))
    return bad


def main():
    d = sys.argv[1] if len(sys.argv) > 1 else "results"
    cells, hosts = load(d)
    if not cells:
        sys.exit("report.py: no results in %s (run bench.sh first)" % d)
    sizes = sorted({k[2] for k in cells})
    examples = [e for e in ORDER if any(k[1] == e for k in cells)] + sorted({k[1] for k in cells} - set(ORDER))
    L = ["# Shaxon against pyshacl and Apache Jena", ""]
    for h in hosts:
        L.append("Host: " + ", ".join("%s %s" % (k, v) for k, v in h.items()) + "  ")
    L += ["", "Times are milliseconds. End to end is the JSON input to a verdict (Shaxon: the whole process; pyshacl and Jena include the RDF lift, and Jena the JVM start). "
          "Engine only is validation alone on data already in each engine's form, best of several runs after a warm-up. "
          "\"skipped\" means the harness's time budget projected that cell too slow to run.", ""]
    for ex in examples:
        L += ["## " + ex, "", "### End to end", ""] + table(cells, "end-to-end", ex, sizes) + ["", "### Engine only", ""] + table(cells, "engine", ex, sizes) + [""]
    bad = verdict_check(cells)
    L.append("## Agreement")
    L.append("")
    L.append("Every cell where more than one engine ran returned the same allow/deny verdict (and, end to end, the same reasons)." if not bad else "DISAGREEMENTS:\n\n" + "\n".join("- " + b for b in bad))
    text = "\n".join(L) + "\n"
    with open(os.path.join(d, "REPORT.md"), "w") as f:
        f.write(text)
    print(text)
    sys.exit(1 if bad else 0)


if __name__ == "__main__":
    main()
