# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
import subprocess, sys, tempfile, os
from common import *
RULES = """
subj(U, U) :- user(U).
subj(U, G) :- subj(U, H), up(H, G).
rc(R, R) :- role(R).
rc(R, S) :- rc(R, T), inh(T, S).
perm(R, A) :- rc(R, S), rperm(S, A).
anc(R, R) :- res(R).
anc(R, P) :- anc(R, C), par(C, P).
sup(U, R, S, 0, G) :- subj(U, X), grant(G, X, R, S, "allow").
sup(U, R, S, H, C) :- deleg(D, F, U, R, S), sup(F, R, S2, H0, C0), anc(S, S2), H0 < 3, H is H0 + 1, C is C0 * 10000 + D.
cov(Q, K) :- req(Q, U, A, X), sup(U, R, S, H, C), anc(X, S), perm(R, A), K is H * 10000000000000000 + C.
den(Q, G) :- req(Q, U, A, X), subj(U, Y), grant(G, Y, R, S, "deny"), anc(X, S), perm(R, A).
via(Q, K) :- K = min(K1) : cov(Q, K1).
"""
def num(i): return int(i[1:])            # G0003 -> 3, D0002 -> 2
def q(x): return '"%s"' % x
def program(db):
    out = []
    for rel, rows in facts(db).items():
        for r in rows:
            if rel == "grant":   r = (num(r[0]),) + r[1:]
            if rel == "deleg":   r = (num(r[0]),) + r[1:]
            out.append("%s(%s)." % (rel, ", ".join(str(x) if isinstance(x, int) else q(x) for x in r)))
    return "\n".join(out) + "\n" + RULES
def decode(k):
    h, c = divmod(int(k), 10**16)
    segs = []
    for _ in range(h): c, d = divmod(c, 10000); segs.append("D%04d" % d)
    return ">".join(["G%04d" % c] + segs[::-1])
def run(db):
    width = max(len(g["id"]) - 1 for g in db["grants"] + db["delegations"])
    if width > 4:
        raise RuntimeError("retired at ids of %d digits (T-37): a chain of four segments needs %d digits "
                           "and does not fit 64 bits" % (width, 4 * width))
    with tempfile.NamedTemporaryFile("w", suffix=".dl", delete=False) as f:
        f.write(program(db))
    try:
        r = subprocess.run([os.environ.get("DLENG", os.path.join(os.path.dirname(os.path.abspath(__file__)), "goeng", "dleng")), f.name], capture_output=True, text=True)
    finally:
        os.unlink(f.name)
    if r.returncode: raise RuntimeError(r.stderr[:500])
    via, den = {}, {}
    for line in r.stdout.splitlines():
        k, a, b = line.split("\t")
        if k == "via": via[a] = decode(b)
        else: den.setdefault(a, set()).add("G%04d" % int(b))
    return assemble(db, set(via), via, den)
if __name__ == "__main__":
    sys.exit(1 if compare("go-engine", run, int(sys.argv[1])) else 0)
