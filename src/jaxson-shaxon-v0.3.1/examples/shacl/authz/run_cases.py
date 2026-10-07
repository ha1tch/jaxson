#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""Run the Shaxon authorisation cases against the SHACL versions.

Uses the very same <name>.cases.json files as the Shaxon examples
(../../shaxon/authz), merging each case's input over the package's default
input exactly as the Go test does, lifting it to RDF (lift.py) and validating
it with pyshacl and the shapes in <name>.shapes.ttl.

    python run_cases.py [--processor jena] [--no-structure] [--verbose]

--processor jena runs Apache Jena SHACL instead of pyshacl. Set JENA_LIB to
a directory holding Jena's jars (jena-cmds and its dependencies; see
README.md). Each case then starts a JVM, so it takes about a second.

--no-structure drops each example's ex:RequestStructure shape, to show what
the rules alone do with a malformed request.

Exit status 0 if every comparable case agrees with the Shaxon expectation.
A case is comparable unless its expectation is an error that has no SHACL
meaning (RESOURCE_ERROR/STEPS: SHACL has no step budget); those are listed as
not applicable. An expected EXECUTION_ERROR (a malformed request that
Shaxon refuses to judge) is met by any non-allow verdict, since SHACL reports
violations and does not raise errors.
"""
import json, os, subprocess, sys, tempfile, time
from decimal import Decimal
from rdflib import BNode, Graph, Namespace
from pyshacl import validate
from lift import lift, EX

SH = Namespace("http://www.w3.org/ns/shacl#")
HERE = os.path.dirname(os.path.abspath(__file__))
SHAXON = os.path.join(HERE, "..", "..", "shaxon", "authz")
NAMES = ["four-eyes-release", "chinese-wall", "delegation-chain", "rolling-quota", "break-glass"]


def load(path):
    with open(path) as f:
        return json.load(f, parse_float=Decimal, parse_int=int)


def drop_structure(shapes):
    """Remove ex:RequestStructure and the blank nodes hanging from it."""
    def descend(node, seen):
        for _, _, o in shapes.triples((node, None, None)):
            if isinstance(o, BNode) and o not in seen:
                seen.add(o)
                descend(o, seen)
        return seen
    doomed = descend(EX.RequestStructure, {EX.RequestStructure})
    for n in doomed:
        shapes.remove((n, None, None))
    # and the references to them, e.g. rdf:rest cells of lists
    for n in doomed:
        shapes.remove((None, None, n))


def local(term):
    return str(term).split("#")[-1].split("/")[-1]


def run_pyshacl(data, shapes):
    conforms, rgraph, _ = validate(data, shacl_graph=shapes, advanced=True, inference="none")
    return conforms, rgraph


def run_jena(data, shapes):
    lib = os.environ.get("JENA_LIB")
    if not lib:
        sys.exit("JENA_LIB must name a directory of Jena jars")
    with tempfile.TemporaryDirectory() as tmp:
        dpath, spath = os.path.join(tmp, "data.ttl"), os.path.join(tmp, "shapes.ttl")
        data.serialize(dpath, format="turtle")
        shapes.serialize(spath, format="turtle")
        env = dict(os.environ, JAVA_TOOL_OPTIONS="")
        r = subprocess.run(["java", "-cp", os.path.join(lib, "*"), "shacl.shacl", "validate",
                            "--shapes", spath, "--data", dpath],
                           capture_output=True, text=True, env=env)
        if r.returncode not in (0, 1):
            sys.exit(f"jena failed ({r.returncode}): {r.stderr[-500:]}")
        rgraph = Graph().parse(data=r.stdout, format="turtle")
    return next(rgraph.objects(None, SH.conforms)).toPython(), rgraph


PROCESSORS = {"pyshacl": run_pyshacl, "jena": run_jena}


def verdict(name, inp, shapes, processor="pyshacl"):
    conforms, rgraph = PROCESSORS[processor](lift(name, inp), shapes)
    reasons = set()
    for res in rgraph.subjects(SH.resultSeverity, None):
        src = rgraph.value(res, SH.sourceConstraint)
        if processor == "jena" and rgraph.value(res, SH.sourceConstraintComponent) == SH.SPARQLConstraintComponent:
            # Jena puts the shape in sh:sourceConstraint, not the sh:sparql
            # constraint (pyshacl puts the constraint). Every constraint here
            # has its own sh:message, so the message names the constraint.
            msg = rgraph.value(res, SH.resultMessage)
            src = next((c for c in shapes.subjects(SH.message, msg)), src)
        reasons.add(local(src) if src is not None else local(rgraph.value(res, SH.sourceConstraintComponent)))
    return conforms, reasons


def main():
    no_structure = "--no-structure" in sys.argv
    verbose = "--verbose" in sys.argv
    processor = sys.argv[sys.argv.index("--processor") + 1] if "--processor" in sys.argv else "pyshacl"
    totals = {"agree": 0, "differ": 0, "n/a": 0}
    for name in NAMES:
        pkg = load(os.path.join(SHAXON, name + ".json"))
        cases = load(os.path.join(SHAXON, name + ".cases.json"))
        shapes = Graph().parse(os.path.join(HERE, name + ".shapes.ttl"))
        if no_structure:
            drop_structure(shapes)
        print(f"== {name}")
        for c in cases:
            inp = dict(pkg["input"])
            inp.update(c.get("input", {}))
            expect = c["expect"]
            t0 = time.perf_counter()
            conforms, reasons = verdict(name, inp, shapes, processor)
            ms = (time.perf_counter() - t0) * 1000
            got = "allow" if conforms else "deny"
            if "error" in expect:
                if expect["error"].startswith("RESOURCE_ERROR"):
                    status, note = "n/a", "no step budget in SHACL"
                else:
                    status = "agree" if got == "deny" else "differ"
                    note = f"shaxon refuses to judge ({expect['error']}); shacl says {got} {sorted(reasons)}"
            else:
                want = {r.removeprefix("SHAX_") for r in expect["reasons"]}
                same = got == expect["decision"] and (reasons == want if want or got == "allow" else True)
                status = "agree" if same else "differ"
                note = f"{got} {sorted(reasons)}"
            totals[status] += 1
            if verbose or status != "agree":
                print(f"  {status:6} {ms:6.0f} ms  {c['name'][:58]}  -> {note}")
        print(f"  ({len(cases)} cases)")
    print("\n" + ", ".join(f"{k}: {v}" for k, v in totals.items()))
    sys.exit(1 if totals["differ"] else 0)


if __name__ == "__main__":
    main()
