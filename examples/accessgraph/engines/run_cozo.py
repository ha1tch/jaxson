# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import sys
from cozo_embedded import CozoDbPy
from common import *
SCHEMA = {"user": ["u"], "up": ["m", "g"], "role": ["r"], "inh": ["r", "s"], "rperm": ["r", "a"],
          "res": ["r"], "par": ["r", "p"], "grant": ["id", "s", "r", "sc", "eff"],
          "deleg": ["id", "f", "t", "r", "sc"], "req": ["id", "u", "a", "x"]}
RULES = """
subj[u, u] := *user[u]
subj[u, g] := subj[u, h], *up[h, g]
rc[r, r] := *role[r]
rc[r, s] := rc[r, t], *inh[t, s]
perm[r, a] := rc[r, s], *rperm[s, a]
anc[r, r] := *res[r]
anc[r, p] := anc[r, c], *par[c, p]
sup[u, r, s, h, c] := subj[u, x], *grant[g, x, r, s, 'allow'], h = 0, c = g
sup[u, r, s, h, c] := *deleg[d, f, u, r, s], sup[f, r, s2, h0, c0], anc[s, s2], h0 < 3, h = h0 + 1, c = concat(c0, '>', d)
cov[q, h, c] := *req[q, u, a, x], sup[u, r, s, h, c], anc[x, s], perm[r, a]
den[q, g] := *req[q, u, a, x], subj[u, y], *grant[g, y, r, s, 'deny'], anc[x, s], perm[r, a]
minh[q, min(h)] := cov[q, h, _]
worse[q, h, c] := cov[q, h, c], cov[q, h, c2], c2 < c
via[q, c] := minh[q, h], cov[q, h, c], not worse[q, h, c]
supp[q] := cov[q, _, _]
?[k, q, v] := supp[q], k = 'sup', v = ''
?[k, q, v] := via[q, v], k = 'via'
?[k, q, v] := den[q, v], k = 'den'
"""
def run(db):
    c = CozoDbPy("mem", "", "{}")
    for rel, cols in SCHEMA.items():
        c.run_script(":create %s {%s}" % (rel, ", ".join(x + ": String" for x in cols)), {}, False)
    for rel, rows in facts(db).items():
        if rows:
            cols = ", ".join(SCHEMA[rel])
            c.run_script("?[%s] <- $rows :put %s {%s}" % (cols, rel, cols), {"rows": [list(r) for r in rows]}, False)
    out = c.run_script(RULES, {}, True)["rows"]
    sup = {q for k, q, v in out if k == "sup"}
    via = {q: v for k, q, v in out if k == "via"}
    den = {}
    for k, q, v in out:
        if k == "den": den.setdefault(q, set()).add(v)
    return assemble(db, sup, via, den)
if __name__ == "__main__":
    sys.exit(1 if compare("cozo", run, int(sys.argv[1])) else 0)
