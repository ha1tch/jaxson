#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
"""Time Shaxon, pyshacl and (if its jars are present) Apache Jena on the same
long trails, and check that all of them reach the same verdict.

    python scale.py                 # 100, 1000 and 4000 events, both profiles
    python scale.py 500 2000        # other sizes
    python scale.py --profile engine       # only the engine-only profile
    python scale.py --profile end-to-end   # only the end-to-end profile

All five examples are scaled (rolling-quota, chinese-wall, four-eyes-release,
delegation-chain, break-glass; --example picks some). The inputs are generated
here, deterministically (a fixed seed), so a run can be repeated. Other options:
--budget SECONDS (the longest one cell may take; a size projected to exceed it
is skipped, with every larger one), --json FILE (one JSON line per cell, which
report.py merges), --repeat N, --engine-runs N. bench.sh runs this in batches.

Two profiles are printed, because the two answer different questions.

END-TO-END is what a caller who holds the JSON input pays to get a verdict.
SHACL validates an RDF graph, so for pyshacl and Jena the RDF lift (lift.py)
is part of the cost; Shaxon reads the JSON tree as it is and has no such step.
That is a real difference, and it is also the one that favours Shaxon most:
a caller who already has the data as RDF does not pay it.

ENGINE-ONLY leaves the lift out. It times only the validation itself, on data
already in each engine's own form, and prints the lift as a separate column so
the difference stays visible. See the notes at the end of the output.

END-TO-END, once per engine and size, as the best of REPEAT runs:

  shaxon    the whole `shaxonrun` process, which is built from this module
            with `go build` into a temporary directory. For rolling-quota the
            package is a temporary copy with limits.steps raised, because the
            shipped package refuses long trails on purpose.
  pyshacl   lifting the input to RDF and validating it, in this process.
  jena      the same, plus writing Turtle files and starting a JVM
            (about 0.7 s of the figure).

ENGINE-ONLY, as the best of ENGINE_RUNS runs after a warm-up:

  shaxon    shaxon.Run in-process (enginetime/main.go) on a package and input
            that are already decoded. Process start and JSON decoding are out.
  pyshacl   pyshacl.validate on an rdflib graph that is already lifted. The
            shapes graph is parsed once, but pyshacl prepares it again inside
            every call; that is part of its validate and is left in.
  jena      Shapes/ShaclValidator in one JVM (JenaTime.java, compiled on the
            fly with javac), data and shapes already parsed, after warm-up
            runs so the JIT has settled. JVM start, Turtle writing and
            parsing are out.
  lift      lift.py on the same input, shown beside them. It is the cost that
            the end-to-end profile adds for both SHACL engines.

The figures compare a Go binary, a Python library and a JVM, so they show that
all three scale about linearly, not what the languages can do.
Exit status 1 if the engines disagree on a verdict.
"""
import json, math, os, platform, random, shutil, subprocess, sys, tempfile, time
from decimal import Decimal

import run_cases as rc

HERE = os.path.dirname(os.path.abspath(__file__))
MODULE = os.path.abspath(os.path.join(HERE, "..", "..", ".."))
SHAXON = os.path.join(HERE, "..", "..", "shaxon", "authz")
REPEAT = 3
ENGINE_RUNS = 10
JENA_WARMUP = 3


def quota_input(n):
    actors = max(n // 6, 1)
    return {
        "policy": {"windowHours": 24, "limitMb": 0.9, "maxExports": 5},
        "events": [{"t": round(i * 20 / n, 4), "actor": "u%d" % (i % actors), "mb": 0.0001} for i in range(n)],
        "request": {"actor": "u1", "t": 20.5, "mb": 0.1},
    }


def wall_input(n):
    rnd = random.Random(7)
    return {
        "classOf": {"c%d" % i: "k%d" % (i % 40) for i in range(200)},
        "events": [{"actor": "u%d" % rnd.randrange(100), "company": "c%d" % rnd.randrange(200)} for _ in range(n)],
        "request": {"actor": "u1", "company": "c1"},
    }


def four_eyes_input(n):
    """n events over n/4 payments, in `seq` order: each payment is created by
    one of 40 people and approved by another; a third are then amended and
    approved again, a tenth have an approval withdrawn. The request releases the
    last payment."""
    rnd = random.Random(7)
    people = ["p%d" % i for i in range(40)]
    events, seq = [], 0
    payments = max(n // 4, 1)
    for i in range(payments):
        pay = "P%d" % i
        creator = people[rnd.randrange(40)]
        approver = people[(people.index(creator) + 1 + rnd.randrange(39)) % 40]
        group = [("create", creator), ("approve", approver)]
        r = rnd.random()
        if r < 0.33:
            group += [("amend", creator), ("approve", approver)]
        elif r < 0.43:
            group += [("revoke", approver), ("approve", people[rnd.randrange(40)])]
        else:
            group += [("approve", people[rnd.randrange(40)]), ("approve", people[rnd.randrange(40)])]
        for typ, who in group:
            seq += 1
            events.append({"seq": seq, "type": typ, "payment": pay, "actor": who})
    return {"events": events[:max(n, 1)], "request": {"payment": "P%d" % (payments - 1), "actor": "releaser"}}


def delegation_input(n):
    """The shipped three-link chain, with n revocations and n uses of other
    grants in the trail (the chain's own links appear once each, so the verdict
    is the shipped package's allow). The chain depth is fixed by the package's
    maxShapeDepth horizon, so the trail is what grows."""
    rnd = random.Random(7)
    revocations = [{"grant": "x%d" % i} for i in range(n)]
    uses = [{"grant": "x%d" % rnd.randrange(max(n, 1))} for _ in range(n)] + [{"grant": "g3"}]
    return {
        "owner": "orla", "revocations": revocations, "uses": uses,
        "request": {"at": 100.5, "grant": {"id": "g3", "holder": "kit", "expires": 200.5, "maxUses": 2,
                    "grantedBy": {"id": "g2", "holder": "jo", "expires": 150.25, "maxUses": 5,
                                  "grantedBy": {"id": "g1", "holder": "orla", "expires": 300, "maxUses": 9}}}},
    }


def breakglass_input(n):
    """n events: break-glass uses by n/8 people (about a third of them reviewed
    by someone else, a tenth by themselves), so earlier uses of the requester are
    in the trail among many other people's."""
    rnd = random.Random(7)
    people = ["a%d" % i for i in range(max(n // 8, 2))]
    events, k = [], 0
    while len(events) < n:
        k += 1
        who = people[rnd.randrange(len(people))]
        events.append({"type": "breakglass", "id": "bg%d" % k, "actor": who, "reason": "r%d" % k})
        r = rnd.random()
        if len(events) < n and r < 0.45:
            reviewer = who if r < 0.05 else people[rnd.randrange(len(people))]
            events.append({"type": "review", "of": "bg%d" % k, "reviewer": reviewer})
    return {"events": events[:n], "request": {"actor": people[0]}}


EXAMPLES = [("rolling-quota", quota_input), ("chinese-wall", wall_input), ("four-eyes-release", four_eyes_input),
            ("delegation-chain", delegation_input), ("break-glass", breakglass_input)]


def build_shaxonrun(tmp):
    exe = os.path.join(tmp, "shaxonrun")
    try:
        subprocess.run(["go", "build", "-o", exe, "./cmd/shaxonrun"], cwd=MODULE, check=True,
                       capture_output=True, text=True)
    except FileNotFoundError:
        die("scale.py: go not found; install Go 1.25 or later, or put it on PATH")
    except subprocess.CalledProcessError as e:
        die("scale.py: go build failed:\n" + e.stderr)
    return exe


def package_for(name, tmp):
    """The package to run: a temporary copy with limits.steps raised, because
    the shipped limits refuse long trails on purpose (that is a result, not a
    timing). The package is otherwise untouched."""
    path = os.path.join(SHAXON, name + ".json")
    with open(path) as f:
        pkg = json.load(f)
    pkg.setdefault("limits", {})["steps"] = 10 ** 12
    out = os.path.join(tmp, name + ".scaled.json")
    with open(out, "w") as f:
        json.dump(pkg, f)
    return out


def run_shaxon(exe, pkg, inp_path, reps=None):
    best, verdict, steps = None, None, None
    for _ in range(reps or REPEAT):
        t0 = time.perf_counter()
        r = subprocess.run([exe, "-steps", pkg, inp_path], capture_output=True, text=True)
        ms = (time.perf_counter() - t0) * 1000
        if r.returncode != 0:
            die("scale.py: shaxonrun failed: " + (r.stdout + r.stderr)[-300:])
        best = ms if best is None else min(best, ms)
        out = json.loads(r.stdout.splitlines()[0])["output"]
        verdict = (out["decision"] == "allow", {f["constraintId"] for f in out["findings"]})
        for line in (r.stdout + r.stderr).splitlines():
            if line.startswith("steps:"):
                steps = int(line.split()[1])
    return best, verdict, steps


def run_shacl(name, inp, processor, reps=None):
    shapes = rc.Graph().parse(os.path.join(HERE, name + ".shapes.ttl"))
    best, verdict = None, None
    for _ in range(reps or REPEAT):
        t0 = time.perf_counter()
        conforms, reasons = rc.verdict(name, inp, shapes, processor)
        ms = (time.perf_counter() - t0) * 1000
        best = ms if best is None else min(best, ms)
        verdict = (conforms, reasons)
    return best, verdict


def build_enginetime(tmp):
    exe = os.path.join(tmp, "enginetime")
    try:
        subprocess.run(["go", "build", "-o", exe, "./examples/shacl/authz/enginetime"], cwd=MODULE, check=True,
                       capture_output=True, text=True)
    except FileNotFoundError:
        die("scale.py: go not found; install Go 1.25 or later, or put it on PATH")
    except subprocess.CalledProcessError as e:
        die("scale.py: go build failed:\n" + e.stderr)
    return exe


def build_jenatime(tmp, lib):
    if not shutil.which("javac"):
        return None
    env = dict(os.environ, JAVA_TOOL_OPTIONS="")
    r = subprocess.run(["javac", "-cp", os.path.join(lib, "*"), "-d", tmp, os.path.join(HERE, "JenaTime.java")],
                       capture_output=True, text=True, env=env)
    if r.returncode != 0:
        print("scale.py: JenaTime.java did not compile; the jena column is skipped:\n" + r.stderr[-300:], file=sys.stderr)
        return None
    return tmp


def engine_shaxon(exe, pkg, inp_path, runs=None):
    r = subprocess.run([exe, "-n", str(runs or ENGINE_RUNS), pkg, inp_path], capture_output=True, text=True)
    if r.returncode != 0:
        die("scale.py: enginetime failed: " + (r.stdout + r.stderr)[-300:])
    d = json.loads(r.stdout)
    out = d["output"]
    return d["ms"], (out["decision"] == "allow", {f["constraintId"] for f in out["findings"]}), d["steps"]


def engine_pyshacl(name, inp, runs=None):
    runs = runs or ENGINE_RUNS
    shapes = rc.Graph().parse(os.path.join(HERE, name + ".shapes.ttl"))
    t0 = time.perf_counter()
    data = rc.lift(name, inp)
    lift_ms = (time.perf_counter() - t0) * 1000
    best, conforms = None, None
    for i in range(runs + 1):
        t0 = time.perf_counter()
        conforms, _ = rc.run_pyshacl(data, shapes)
        ms = (time.perf_counter() - t0) * 1000
        if i > 0 or runs == 0:
            best = ms if best is None else min(best, ms)
    return best, conforms, lift_ms, data


def engine_jena(jt, lib, name, data, tmp, runs=None, warmup=None):
    dpath, spath = os.path.join(tmp, "jena-data.ttl"), os.path.join(HERE, name + ".shapes.ttl")
    data.serialize(dpath, format="turtle")
    env = dict(os.environ, JAVA_TOOL_OPTIONS="")
    r = subprocess.run(["java", "-cp", jt + os.pathsep + os.path.join(lib, "*"), "JenaTime", spath, dpath,
                        str(JENA_WARMUP if warmup is None else warmup), str(runs or ENGINE_RUNS)],
                       capture_output=True, text=True, env=env)
    if r.returncode != 0:
        die("scale.py: JenaTime failed: " + r.stderr[-300:])
    d = json.loads(r.stdout)
    return d["ms"], d["conforms"]


def die(msg):
    print(msg, file=sys.stderr)
    sys.exit(2)


def host_info():
    info = {"python": sys.version.split()[0], "platform": platform.platform(), "machine": platform.machine(),
            "cpus": os.cpu_count()}
    for cmd, key in ((["sysctl", "-n", "machdep.cpu.brand_string"], "cpu"), (["sh", "-c", "grep -m1 'model name' /proc/cpuinfo | cut -d: -f2"], "cpu")):
        if key not in info:
            try:
                r = subprocess.run(cmd, capture_output=True, text=True)
                if r.returncode == 0 and r.stdout.strip():
                    info[key] = r.stdout.strip()
            except OSError:
                pass
    env = dict(os.environ, JAVA_TOOL_OPTIONS="")
    for cmd, key in ((["go", "version"], "go"), (["java", "-version"], "java")):
        try:
            r = subprocess.run(cmd, capture_output=True, text=True, env=env)
            out = [l for l in (r.stdout + r.stderr).strip().splitlines() if "version" in l]
            info[key] = out[0].replace("go version ", "").strip() if out else "?"
        except OSError:
            info[key] = "missing"
    try:
        import pyshacl, rdflib
        info["pyshacl"], info["rdflib"] = pyshacl.__version__, rdflib.__version__
    except ImportError:
        pass
    return info


class Budget:
    """Keeps one slow engine from eating the whole run. For each (example,
    engine, profile) the sizes are visited in increasing order; after each
    measured size the time of the next is projected from the growth seen so far
    (at least linear), and a size whose projected single run exceeds the
    budget is skipped, as is every larger one. A size that does fit gets as
    many repeats as the budget allows, up to the usual number."""

    def __init__(self, seconds):
        self.limit_ms = seconds * 1000.0
        self.seen = {}      # key -> list of (n, ms of one run)
        self.stopped = set()

    def plan(self, key, n, repeats):
        """Returns the number of runs to make, or 0 to skip."""
        if key in self.stopped:
            return 0
        pts = self.seen.get(key, [])
        if not pts:
            return repeats  # the first size is always run; its first run decides nothing yet
        (n1, t1) = pts[-1]
        e = 1.0
        if len(pts) >= 2:
            (n0, t0) = pts[-2]
            if t0 > 0 and t1 > 0 and n1 > n0:
                e = max(1.0, math.log(t1 / t0) / math.log(n1 / n0))
        proj = t1 * (n / n1) ** e
        if proj > self.limit_ms:
            self.stopped.add(key)
            return 0
        return max(1, min(repeats, int(self.limit_ms // proj)))

    def record(self, key, n, ms):
        self.seen.setdefault(key, []).append((n, ms))


def fmt_cell(ms, width, unit_ms=True, digits=0):
    if ms is None:
        return "%*s" % (width + 3, "skipped")
    return "%*.*f ms" % (width, digits, ms)


def main():
    global REPEAT, ENGINE_RUNS
    args = sys.argv[1:]

    def take(flag, conv=str, default=None):
        if flag in args:
            k = args.index(flag)
            if k + 1 >= len(args):
                die("scale.py: %s needs a value" % flag)
            v = conv(args[k + 1])
            del args[k:k + 2]
            return v
        return default

    profile = take("--profile", str, "both")
    if profile not in ("end-to-end", "engine", "both"):
        die("scale.py: --profile takes end-to-end, engine or both")
    chosen = take("--example", str, "all")
    budget = Budget(take("--budget", float, 120.0))
    jpath = take("--json", str, None)
    REPEAT = take("--repeat", int, REPEAT)
    ENGINE_RUNS = take("--engine-runs", int, ENGINE_RUNS)
    names = [n for n, _ in EXAMPLES]
    picked = names if chosen == "all" else chosen.split(",")
    for n in picked:
        if n not in names:
            die("scale.py: unknown example %r; choose from %s" % (n, ", ".join(names)))
    examples = [(n, mk) for n, mk in EXAMPLES if n in picked]
    sizes = sorted(int(a) for a in args) or [100, 1000, 4000]
    lib = os.environ.get("JENA_LIB", os.path.join(HERE, ".jena", "lib"))
    os.environ["JENA_LIB"] = lib
    jena = any(f.startswith("jena-cmds-") for f in (os.listdir(lib) if os.path.isdir(lib) else []))
    disagreements = 0

    def emit(rec):
        if jpath:
            with open(jpath, "a") as f:
                f.write(json.dumps(rec, default=str) + "\n")

    emit({"kind": "host", "when": time.strftime("%Y-%m-%d %H:%M:%S"), "profile": profile, "examples": picked, "sizes": sizes,
          "budget_s": budget.limit_ms / 1000, "repeat": REPEAT, "engine_runs": ENGINE_RUNS, **host_info()})

    with tempfile.TemporaryDirectory() as tmp:
        cases = []
        for n in sizes:
            for name, make in examples:
                text = json.dumps(make(n))
                inp_path = os.path.join(tmp, f"{name}-{n}.json")
                with open(inp_path, "w") as f:
                    f.write(text)
                cases.append((n, name, inp_path, json.loads(text, parse_float=Decimal), package_for(name, tmp)))
        cases.sort(key=lambda c: ([e[0] for e in examples].index(c[1]), c[0]))

        if profile in ("end-to-end", "both"):
            exe = build_shaxonrun(tmp)
            print("END-TO-END (the JSON input to a verdict; the RDF lift is inside the SHACL figures)")
            print(f"{'example':18} {'events':>6}  {'shaxon':>9}  {'steps':>6}  {'pyshacl':>12}  " + (f"{'jena':>12}  " if jena else "") + "verdict")
            for n, name, inp_path, inp, pkg in cases:
                skey = (name, "shaxon", "e2e")
                sreps = budget.plan(skey, n, REPEAT)
                verdicts, row = [], f"{name:18} {n:>6}  "
                steps, sx_v = None, None
                if sreps == 0:
                    row += fmt_cell(None, 6) + f"  {'':>6}  "
                    emit({"kind": "cell", "profile": "end-to-end", "example": name, "events": n, "engine": "shaxon", "ms": None,
                          "skipped": "projected single run over the %.0f s budget" % (budget.limit_ms / 1000)})
                else:
                    sx_ms, sx_v, steps = run_shaxon(exe, pkg, inp_path, sreps)
                    budget.record(skey, n, sx_ms)
                    emit({"kind": "cell", "profile": "end-to-end", "example": name, "events": n, "engine": "shaxon", "ms": sx_ms, "steps": steps,
                          "repeats": sreps, "allow": sx_v[0], "reasons": sorted(sx_v[1])})
                    row += f"{sx_ms:>6.0f} ms  {steps:>6}  "
                    verdicts.append(sx_v)
                engines = [("pyshacl", "pyshacl")] + ([("jena", "jena")] if jena else [])
                for eng, proc in engines:
                    key = (name, eng, "e2e")
                    reps = budget.plan(key, n, REPEAT)
                    if reps == 0:
                        row += fmt_cell(None, 9) + "  "
                        emit({"kind": "cell", "profile": "end-to-end", "example": name, "events": n, "engine": eng, "ms": None,
                              "skipped": "projected single run over the %.0f s budget" % (budget.limit_ms / 1000)})
                        continue
                    ms, v = run_shacl(name, inp, proc, reps)
                    budget.record(key, n, ms)
                    row += fmt_cell(ms, 6) + "  "
                    verdicts.append(v)
                    emit({"kind": "cell", "profile": "end-to-end", "example": name, "events": n, "engine": eng, "ms": ms,
                          "repeats": reps, "allow": v[0], "reasons": sorted(v[1])})
                agree = all(v == verdicts[0] for v in verdicts)
                disagreements += 0 if agree else 1
                shown = (("allow" if verdicts[0][0] else "deny") + " " + str(sorted(verdicts[0][1]))) if verdicts else "-"
                print(row + (shown if agree else "DISAGREE " + str(verdicts)), flush=True)
            print(f"best of up to {REPEAT}; shaxon is the whole process, pyshacl and jena include the RDF lift; 'skipped' = projected over the budget")

        if profile in ("engine", "both"):
            if profile == "both":
                print()
            exe = build_enginetime(tmp)
            jt = build_jenatime(tmp, lib) if jena else None
            print("ENGINE-ONLY (validation alone, on data already in each engine's form; the lift is shown, not counted)")
            print(f"{'example':18} {'events':>6}  {'shaxon':>10}  {'steps':>6}  {'pyshacl':>12}  " + (f"{'jena':>12}  " if jt else "") + f"{'lift':>9}  verdict")
            for n, name, inp_path, inp, pkg in cases:
                skey = (name, "shaxon", "engine")
                sruns = budget.plan(skey, n, ENGINE_RUNS)
                conforms, row = [], f"{name:18} {n:>6}  "
                if sruns == 0:
                    row += fmt_cell(None, 7) + f"  {'':>6}  "
                    emit({"kind": "cell", "profile": "engine", "example": name, "events": n, "engine": "shaxon", "ms": None,
                          "skipped": "projected single run over the %.0f s budget" % (budget.limit_ms / 1000)})
                else:
                    sx_ms, sx_v, steps = engine_shaxon(exe, pkg, inp_path, sruns)
                    budget.record(skey, n, sx_ms)
                    emit({"kind": "cell", "profile": "engine", "example": name, "events": n, "engine": "shaxon", "ms": sx_ms, "steps": steps,
                          "repeats": sruns, "allow": sx_v[0]})
                    row += f"{sx_ms:>7.2f} ms  {steps:>6}  "
                    conforms.append(sx_v[0])
                key = (name, "pyshacl", "engine")
                reps = budget.plan(key, n, ENGINE_RUNS)
                data, lift_ms = None, None
                if reps == 0:
                    row += fmt_cell(None, 9) + "  "
                    emit({"kind": "cell", "profile": "engine", "example": name, "events": n, "engine": "pyshacl", "ms": None,
                          "skipped": "projected single run over the %.0f s budget" % (budget.limit_ms / 1000)})
                else:
                    py_ms, py_conforms, lift_ms, data = engine_pyshacl(name, inp, reps)
                    budget.record(key, n, py_ms)
                    row += fmt_cell(py_ms, 6) + "  "
                    conforms.append(py_conforms)
                    emit({"kind": "cell", "profile": "engine", "example": name, "events": n, "engine": "pyshacl", "ms": py_ms,
                          "repeats": reps, "lift_ms": lift_ms, "allow": py_conforms})
                if jt:
                    jkey = (name, "jena", "engine")
                    jreps = budget.plan(jkey, n, ENGINE_RUNS)
                    if jreps != 0 and data is None:
                        # pyshacl was skipped, so there is no lifted graph yet: lift here
                        t0 = time.perf_counter()
                        data = rc.lift(name, inp)
                        lift_ms = (time.perf_counter() - t0) * 1000
                    if jreps == 0:
                        row += fmt_cell(None, 9) + "  "
                        emit({"kind": "cell", "profile": "engine", "example": name, "events": n, "engine": "jena", "ms": None,
                              "skipped": "projected single run over the budget"})
                    else:
                        je_ms, je_conforms = engine_jena(jt, lib, name, data, tmp, jreps, min(JENA_WARMUP, jreps))
                        budget.record(jkey, n, je_ms)
                        row += fmt_cell(je_ms, 6, digits=1) + "  "
                        conforms.append(je_conforms)
                        emit({"kind": "cell", "profile": "engine", "example": name, "events": n, "engine": "jena", "ms": je_ms,
                              "repeats": jreps, "allow": je_conforms})
                row += (f"{lift_ms:>6.0f} ms  " if lift_ms is not None else f"{'':>9}  ")
                agree = all(c == conforms[0] for c in conforms)
                disagreements += 0 if agree else 1
                shown = ("allow" if conforms[0] else "deny") if conforms else "-"
                print(row + (shown if agree else "DISAGREE " + str(conforms)), flush=True)
            print(f"best of up to {ENGINE_RUNS} after a warm-up; the verdict compared is allow/deny (the end-to-end profile compares reasons too)")
            print("lift = lift.py on the same input: the cost the end-to-end profile adds for pyshacl and Jena, and Shaxon does not have")
    sys.exit(1 if disagreements else 0)


if __name__ == "__main__":
    main()
