# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Run one engine on one Amazon variant; print one JSON line.
usage: AMAZON_DB=DIR python3 amazon_one.py ENGINE VARIANT   (ENGINE: oracle go-ref go-fast souffle-compiled
souffle-interp jaxrun cozo; VARIANT: flat nested; run `oracle` first, it writes the reference decisions)"""
import json, os, shutil, subprocess, sys, tempfile, time
HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..")); sys.path.insert(0, HERE)
import cost, gen, oracle
from common import facts, assemble
import run_souffle, run_cozo, run_starlark
AG = os.environ.get("AGOGO", os.path.join(HERE, "..", "agogo", "agogo")); JAXRUN = os.environ.get("JAXRUN", "jaxrun")
engine, variant = sys.argv[1], sys.argv[2]
D = os.environ.get("AMAZON_DB", os.path.join(HERE, "..", "data", "amazon-db")) + os.sep   # where amazon_to_db.py writes
db = json.load(open(D + variant + ".json"))
def dump(o):
    f = tempfile.NamedTemporaryFile("w", suffix=".json", delete=False); json.dump(o, f); f.close(); return f.name
def go(impl):
    p = dump(db)
    try:
        r = subprocess.run([AG, "eval", "-impl", impl, p], capture_output=True, text=True)
        if r.returncode: raise RuntimeError(r.stderr[:200])
        return json.loads(r.stdout)
    finally: os.unlink(p)
def souffle_compiled():
    d = tempfile.mkdtemp(prefix="sfc_")
    try:
        for rel, rows in facts(db).items():
            with open(os.path.join(d, rel + ".facts"), "w") as f: f.write("".join("\t".join(r) + "\n" for r in rows))
        o = os.path.join(d, "out"); os.mkdir(o)
        r = subprocess.run([os.environ.get("AG_COMPILED", os.path.join(HERE, "ag_compiled")), "-F", d, "-D", o], capture_output=True, text=True)
        if r.returncode: raise RuntimeError(r.stderr[:300])
        rd = lambda n: [l.rstrip("\n").split("\t") for l in open(os.path.join(o, n + ".csv"))]
        den = {}
        for a, b in rd("den"): den.setdefault(a, set()).add(b)
        return assemble(db, {x[0] for x in rd("supported")}, {a: b for a, b in rd("via")}, den)
    finally: shutil.rmtree(d, ignore_errors=True)
def jaxrun():
    pkg = gen.build(db, steps=int(os.environ["STEPS"])) if os.environ.get("STEPS") else gen.build(db)
    p = dump(pkg)
    try:
        r = subprocess.run([JAXRUN, p], capture_output=True, text=True)
        if r.returncode: raise RuntimeError((r.stdout + r.stderr)[:200])
        return json.loads(r.stdout)
    finally: os.unlink(p)
ENG = {"oracle": lambda: oracle.evaluate(db), "go-ref": lambda: go("ref"), "go-fast": lambda: go("fast"),
       "souffle-compiled": souffle_compiled, "souffle-interp": lambda: run_souffle.run(db),
       "jaxrun": jaxrun, "cozo": lambda: run_cozo.run(db), "starlark": lambda: run_starlark.run(db)}
wantf = D + "want-" + variant + ".json"
t = time.perf_counter()
try:
    out = ENG[engine]()
    sec = time.perf_counter() - t
    if engine == "oracle":
        json.dump(out, open(wantf, "w")); equal = True
    else:
        equal = out == json.load(open(wantf))
    print(json.dumps({"engine": engine, "variant": variant, "seconds": round(sec, 2), "equal_to_reference": equal, "decisions": len(out)}))
except Exception as e:
    print(json.dumps({"engine": engine, "variant": variant, "seconds": round(time.perf_counter() - t, 2), "error": str(e)[:160]}))
