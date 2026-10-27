# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import os, sys, io, contextlib, tempfile
from common import *
import run_souffle, run_cozo, run_goeng
N = 24
def count(run):
    bad = 0
    retired = False
    for seed in range(N):
        db = synth.generate(seed, ("tiny", "small", "small", "medium")[seed % 4])
        try:
            if run(db) != oracle.evaluate(db): bad += 1
        except RuntimeError as e:
            if "retired" not in str(e): raise
            return "retired"
    return bad
MUT = {  # name: (souffle, cozo, goeng) substitution pairs (old, new)
 "one hop short": (("H < 3", "H < 2"), ("h0 < 3", "h0 < 2"), ("H0 < 3", "H0 < 2")),
 "one hop extra": (("H < 3", "H < 4"), ("h0 < 3", "h0 < 4"), ("H0 < 3", "H0 < 4")),
 "denies ignored": (('"deny"', '"denx"'), ("'deny'", "'denx'"), ('"deny"', '"denx"')),
 "group nesting dropped": (("subj(U, G) :- subj(U, H), up(H, G).", "subj(U, G) :- user(U), up(U, G)."),
                           ("subj[u, g] := subj[u, h], *up[h, g]", "subj[u, g] := *user[u], *up[u, g]"),
                           ("subj(U, G) :- subj(U, H), up(H, G).", "subj(U, G) :- user(U), up(U, G).")),
 "ancestor walk dropped": (("anc(R, P) :- anc(R, C), par(C, P).", ""),
                           ("anc[r, p] := anc[r, c], *par[c, p]", ""),
                           ("anc(R, P) :- anc(R, C), par(C, P).", "")),
}
HERE = os.path.dirname(os.path.abspath(__file__))
BASE_DL = os.path.join(HERE, "ag.dl")
MUT_DL = os.path.join(tempfile.gettempdir(), "ag_mut.dl")
base_dl = open(BASE_DL).read()
print("%-24s %8s %8s %8s   (databases with a mismatch, of %d)" % ("mutant", "souffle", "cozo", "go-eng", N))
print("%-24s %8s %8s %8s" % ("control", count(run_souffle.run), count(run_cozo.run), count(run_goeng.run)))
for name, (sf, cz, ge) in MUT.items():
    if base_dl.count(sf[0]) != 1 or run_cozo.RULES.count(cz[0]) != 1 or run_goeng.RULES.count(ge[0]) != 1:
        print("GUARD FAILED for", name); sys.exit(1)
    open(MUT_DL, "w").write(base_dl.replace(*sf)); os.environ["AG_DL"] = MUT_DL
    a = count(run_souffle.run); os.environ["AG_DL"] = BASE_DL
    c0, g0 = run_cozo.RULES, run_goeng.RULES
    run_cozo.RULES = c0.replace(*cz); b = count(run_cozo.run); run_cozo.RULES = c0
    run_goeng.RULES = g0.replace(*ge); c = count(run_goeng.run); run_goeng.RULES = g0
    print("%-24s %8s %8s %8s" % (name, a, b, c))
