# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Time the evaluators end to end on generated databases.

  python3 sweep.py SIZES [ENGINE_FILTER]

SIZES is a comma list of tiny, small, medium, large, xlarge. ENGINE_FILTER is a comma list of
substrings of engine names (default: all). Every engine must equal the reference on every
decision or the run stops. Needs the engines of engines/README.md.
"""
import json, os, re, subprocess, sys, tempfile, time, shutil
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, ".."))
sys.path.insert(0, HERE)
import synth
import gen, jx, oracle
import run_souffle, run_cozo, run_starlark
from common import facts

AG = os.environ.get("AGOGO", os.path.join(HERE, "..", "agogo", "agogo"))

def dumpdb(db):
    f = tempfile.NamedTemporaryFile("w", suffix=".json", delete=False); json.dump(db, f); f.close(); return f.name

def go_oracle(impl):
    def run(db):
        p = dumpdb(db)
        try:
            r = subprocess.run([AG, "eval", "-impl", impl, p], capture_output=True, text=True)
            if r.returncode: raise RuntimeError(r.stderr[:200])
            return json.loads(r.stdout)
        finally: os.unlink(p)
    return run

def souffle_compiled(db):
    d = tempfile.mkdtemp(prefix="sfc_")
    try:
        for rel, rows in facts(db).items():
            with open(os.path.join(d, rel + ".facts"), "w") as f:
                f.write("".join("\t".join(r) + "\n" for r in rows))
        o = os.path.join(d, "out"); os.mkdir(o)
        r = subprocess.run([os.environ.get("AG_COMPILED", os.path.join(HERE, "ag_compiled")), "-F", d, "-D", o], capture_output=True, text=True)
        if r.returncode: raise RuntimeError(r.stderr[:300])
        rd = lambda n: [l.rstrip("\n").split("\t") for l in open(os.path.join(o, n + ".csv"))]
        from common import assemble
        den = {}
        for a, b in rd("den"): den.setdefault(a, set()).add(b)
        return assemble(db, {x[0] for x in rd("supported")}, {a: b for a, b in rd("via")}, den)
    finally: shutil.rmtree(d, ignore_errors=True)

def jaxrun(db):
    pkg = gen.build(db)
    p = dumpdb(pkg)
    try:
        r = subprocess.run([os.environ.get("JAXRUN", "jaxrun"), p], capture_output=True, text=True)
        if r.returncode: raise RuntimeError((r.stdout + r.stderr)[:200])
        return json.loads(r.stdout)
    finally: os.unlink(p)

ENG = [
    ("oracle.py (Python)",               lambda db: oracle.evaluate(db)),
    ("Go oracle, line-for-line port",    go_oracle("ref")),
    ("Go oracle, optimized",             go_oracle("fast")),
    ("Soufflé, interpreter",             run_souffle.run),
    ("Soufflé, compiled",                souffle_compiled),
    ("Jaxson program on jaxrun",         jaxrun),
    ("CozoDB",                           run_cozo.run),
    ("Starlark (starlark-go)",           run_starlark.run),
    ("Jaxson program on jx.py",          lambda db: jx.run(gen.build(db))[0]),
]
only = sys.argv[2].split(",") if len(sys.argv) > 2 else None

def timed(fn, db, want):
    best = None
    for i in range(3):
        t = time.perf_counter(); out = fn(db); d = time.perf_counter() - t
        assert out == want, "output differs from oracle.py"
        best = d if best is None else min(best, d)
        if d > 1.0: break
    return best * 1000

rows = {n: [] for n, _ in ENG}; comp = {"oracle.py": [], "Go ref": [], "Go fast": []}
sizes = sys.argv[1].split(",")
for size in sizes:
    dbs = [synth.generate(s, size) for s in (range(6) if size in ("tiny", "small", "medium") else range(1))]
    wants = [oracle.evaluate(d) for d in dbs]
    for name, fn in ENG:
        if only and not any(o in name for o in only): rows[name].append(float("nan")); continue
        rows[name].append(sum(timed(fn, d, w) for d, w in zip(dbs, wants)) / len(dbs))
    c = {"oracle.py": 0.0, "Go ref": 0.0, "Go fast": 0.0}
    for d in dbs:
        t = time.perf_counter(); oracle.evaluate(d); c["oracle.py"] += (time.perf_counter() - t) * 1000
        p = dumpdb(d)
        for k, impl in (("Go ref", "ref"), ("Go fast", "fast")):
            r = subprocess.run([AG, "eval", "-impl", impl, "-repeat", "3", p], capture_output=True, text=True)
            c[k] += float(re.search(r"evaluate: ([\d.]+) ms", r.stderr).group(1))
        os.unlink(p)
    for k in c: comp[k].append(c[k] / len(dbs))
print("END TO END, ms per database (called from Python with the database in memory; mean over %s)" % ("6 databases for tiny to medium, 1 for large and xlarge"))
print("%-34s" % "engine" + "".join("%12s" % s for s in sizes))
for n, _ in ENG: print("%-34s" % n + "".join("%12.1f" % v for v in rows[n]))
print("\nCOMPUTE ONLY, ms per database (inside the process, index building included, best of 3)")
for k in comp: print("%-34s" % k + "".join("%12.2f" % v for v in comp[k]))
