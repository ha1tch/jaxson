# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""A data-integrity gate for accessgraph databases, written as a Shaxon package.

The reference (oracle.py) is defined only on a referentially clean database, and
the generated Jaxson program fails on a dangling reference or, worse, on data
deeper than the rounds it was written for. This package checks a database
before either sees it. It is one fixed package; the database is its input.

  python3 integrity.py [--limits G,R,D|policy] [--shaxonrun PATH] DB.json
      check one database, print the findings, exit 1 if there are any
  python3 integrity.py --write [FILE] [--limits G,R,D]
      write the package (default integrity.json)

What it checks, each finding at the exact path of the offending element:

  references   members, role inheritance, resource parents, grants, delegations
               and requests must name things that exist, and an effect must be
               allow or deny (a dangling reference is SHAX_DANGLING_REFERENCE)
  uniqueness   the later occurrence of a repeated user, group, role, resource,
               grant, delegation or request id, and of a name that is both a
               user and a group
  width        a grant or delegation id with other than policy.ID_WIDTH digits after
               its letter: a mixed-width database is rejected, not mis-ordered
  depth        only with --limits G,R,D: a user whose membership chain is more
               than G rounds deep, a role whose inheritance chain is more than R
               deep, a resource more than D below its root. Each is the same
               fixed number of closure rounds the inference program unrolls, plus
               one more round that records whoever is still changing, so the
               check cannot be satisfied by data the program would truncate.

The shaxonrun runner is found through --shaxonrun or the SHAXONRUN variable.
"""
import json
import os
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import policy  # noqa: E402
from gen import P, L, V, S, A, F, I, st, has, op2, keys, no  # noqa: E402

# Provisional; deriving the step limit from the data is T-38.
STEPS = 100000000

# (array, shape) pairs checked element by element
ELEMENTS = [("members", "Member"), ("roles", "Role"), ("resources", "Resource"),
            ("grants", "Grant"), ("delegations", "Deleg"), ("requests", "Request")]

# (array, field, id) uniqueness entries; field None means the element itself
UNIQUE = [("users", None, "USER_UNIQUE"), ("groups", None, "GROUP_UNIQUE"),
          ("roles", "id", "ROLE_UNIQUE"), ("resources", "id", "RESOURCE_UNIQUE"),
          ("grants", "id", "GRANT_UNIQUE"), ("delegations", "id", "DELEGATION_UNIQUE"),
          ("requests", "id", "REQUEST_UNIQUE")]


def ref(index, **extra):
    return dict({"kind": "reference", "index": index}, **extra)


def id_shape():
    """A grant or delegation id has ID_WIDTH digits after its letter."""
    return {"kind": "string", "minLen": policy.ID_WIDTH + 1, "maxLen": policy.ID_WIDTH + 1}


def shapes():
    """name -> (shape body, {field rule name: field name}); the rule names let
    the mutation check remove one rule at a time."""
    return {
        "Member": {"kind": "object", "closed": False, "required": ["member", "group"],
                   "fields": {"member": ref("subjects"), "group": ref("groups")}},
        "Role": {"kind": "object", "closed": False,
                 "fields": {"inherits": {"kind": "array", "items": ref("roles")}}},
        "Resource": {"kind": "object", "closed": False,
                     "fields": {"parent": {"or": [{"kind": "null"}, ref("resources")]}}},
        "Grant": {"kind": "object", "closed": False,
                  "required": ["subject", "role", "scope", "effect"],
                  "fields": {"id": id_shape(), "subject": ref("subjects"), "role": ref("roles"),
                             "scope": ref("resources"),
                             "effect": {"kind": "string", "enum": ["allow", "deny"]}}},
        "Deleg": {"kind": "object", "closed": False, "required": ["from", "to", "role", "scope"],
                  "fields": {"id": id_shape(), "from": ref("users"), "to": ref("users"),
                             "role": ref("roles"), "scope": ref("resources")}},
        "Request": {"kind": "object", "closed": False, "required": ["user", "action", "resource"],
                    "fields": {"user": ref("users"), "resource": ref("resources")}},
    }


def FI(src, var, idx, body):
    return {"op": "for", "in": src, "as": var, "index": idx, "do": body}


def rounds(out, owners, edges, n, detect):
    """n closure rounds over state.<edges>, then one more that only records who
    still changes. The first n are the rounds the inference program unrolls."""
    p = [S(["state", out], {}), S(["state", detect], {}),
         F(owners, "o", [S(["state", out, L("o")], {}), S(["state", out, L("o"), L("o")], True)])]

    def body(found):
        return F(owners, "o", [F(keys(st(out, L("o"))), "x", [
            I(has(P("state", edges), L("x")), [F(P("state", edges, L("x")), "y", [
                I(no(has(st(out, L("o")), L("y"))), found)])])])])
    p += [body([S(["state", out, L("o"), L("y")], True)]) for _ in range(n)]
    p.append(body([S(["state", detect, L("o")], True)]))
    return p


def depth_program(g, r, d):
    p = [S(["state", "up"], {}), S(["state", "inh"], {}), S(["state", "roleIds"], []), S(["state", "parent"], {}),
         F(P("input", "users"), "s", [S(["state", "up", L("s")], [])]),
         F(P("input", "groups"), "s", [S(["state", "up", L("s")], [])]),
         F(P("input", "members"), "m", [I(has(P("state", "up"), L("m", "member")),
                                          [A(["state", "up", L("m", "member")], L("m", "group"))])]),
         F(P("input", "roles"), "x", [A(["state", "roleIds"], L("x", "id")),
                                      S(["state", "inh", L("x", "id")], L("x", "inherits"))]),
         F(P("input", "resources"), "x", [I(op2("ne", L("x", "parent"), None),
                                            [S(["state", "parent", L("x", "id")], L("x", "parent"))],
                                            [S(["state", "parent", L("x", "id")], "")])])]
    p += rounds("sub", P("input", "users"), "up", g, "deepUsers")
    p += rounds("rc", P("state", "roleIds"), "inh", r, "deepRoles")
    walk = [I(op2("ne", st("cur"), ""), [I(has(P("state", "parent"), st("cur")),
                                           [S(["state", "cur"], P("state", "parent", st("cur")))],
                                           [S(["state", "cur"], "")])])]
    p += [S(["state", "deepRes"], {}), F(P("input", "resources"), "x", [S(["state", "cur"], L("x", "id"))]
                                         + walk * (d + 1)
                                         + [I(op2("ne", st("cur"), ""), [S(["state", "deepRes", L("x", "id")], True)])])]
    return p


def flag(cid, setname, key):
    return {"kind": "any", "check": {"id": cid, "with": {"d": P("state", setname), "k": key},
                                     "expr": ["not", ["has", V("d"), V("k")]]}}


def package(limits=None, skip=()):
    """The gate. limits is (G, R, D) or None. skip names rules to leave out, for
    the mutation check: "Shape.field" or "unique.<array>"."""
    sh = shapes()
    for name in list(sh):
        for fld in list(sh[name]["fields"]):
            if "%s.%s" % (name, fld) in skip:
                del sh[name]["fields"][fld]
    idx = lambda src, key: {"source": src, "key": key, "multi": True}
    indices = {"users": idx(P("input", "users"), L("item")), "groups": idx(P("input", "groups"), L("item")),
               "subjects": idx(P("state", "subjects"), L("item")),
               "roles": idx(P("input", "roles"), L("item", "id")),
               "resources": idx(P("input", "resources"), L("item", "id"))}
    validate = []
    for arr, fld, cid in UNIQUE:
        if "unique." + arr in skip:
            continue
        u = {"id": cid}
        if fld:
            u["field"] = fld
        validate.append({"target": {"$each": ["input", arr]}, "unique": u, "mode": "report"})
    program = [S(["state", "subjects"], []),
               F(P("input", "users"), "u", [A(["state", "subjects"], L("u"))]),
               F(P("input", "groups"), "g", [A(["state", "subjects"], L("g"))])]
    if "unique.subjects" not in skip:
        program.append({"op": "check", "target": {"$each": ["state", "subjects"]},
                        "unique": {"id": "SUBJECT_UNIQUE"}, "mode": "report"})
    for arr, shape in ELEMENTS:
        program.append(FI(P("input", arr), "e", "i", [
            {"op": "check", "target": P("input", arr, L("i")), "shape": shape, "mode": "report"}]))
    if limits:
        g, r, d = limits
        sh["UserDepth"] = flag("GROUP_NESTING_EXCEEDED", "deepUsers", P("local", "focus"))
        sh["RoleDepth"] = flag("ROLE_DEPTH_EXCEEDED", "deepRoles", P("local", "focus", "id"))
        sh["ResourceDepth"] = flag("RESOURCE_DEPTH_EXCEEDED", "deepRes", P("local", "focus", "id"))
        program += depth_program(g, r, d)
        for arr, shape in (("users", "UserDepth"), ("roles", "RoleDepth"), ("resources", "ResourceDepth")):
            program.append(FI(P("input", arr), "e", "i", [
                {"op": "check", "target": P("input", arr, L("i")), "shape": shape, "mode": "report"}]))
    program.append(S(["output"], {"$lit": "checked"}))
    return {"shaxon": "3.1", "limits": {"steps": STEPS, "maxShapeDepth": 8}, "indices": indices,
            "shapes": sh, "validate": validate, "program": program, "input": {}}


def check_db(db, shaxonrun=None, limits=None, skip=()):
    """Run the gate over a database. Returns a list of (code or constraint id,
    focus path tuple); an empty list means the database passes."""
    runner = shaxonrun or os.environ.get("SHAXONRUN", "shaxonrun")
    with tempfile.TemporaryDirectory() as d:
        json.dump(package(limits, skip), open(os.path.join(d, "p.json"), "w"))
        json.dump(db, open(os.path.join(d, "i.json"), "w"))
        r = subprocess.run([runner, os.path.join(d, "p.json"), os.path.join(d, "i.json")],
                           capture_output=True, text=True)
    if r.returncode:
        raise RuntimeError("shaxonrun failed: " + r.stderr.strip()[:400])
    out = json.loads(r.stdout)["report"]
    return [(v.get("code") or v.get("constraintId"), tuple(v["focusPath"])) for v in out["violations"]]


def main(argv):
    args = list(argv[1:])

    def opt(name):
        if name in args:
            i = args.index(name)
            v = args[i + 1]
            del args[i:i + 2]
            return v
        return None
    limits = opt("--limits")
    if limits == "policy":
        limits = (policy.GROUP_ROUNDS, policy.ROLE_ROUNDS, policy.RES_DEPTH)
    else:
        limits = tuple(int(x) for x in limits.split(",")) if limits else None
    runner = opt("--shaxonrun")
    if "--write" in args:
        args.remove("--write")
        path = args[0] if args else os.path.join(HERE, "integrity.json")
        json.dump(package(limits), open(path, "w"), sort_keys=True)
        print("wrote", path)
        return 0
    if len(args) != 1:
        sys.stderr.write(__doc__)
        return 2
    found = check_db(json.load(open(args[0])), runner, limits)
    for code, path in found:
        print("%-28s %s" % (code, "/".join(str(x) for x in path)))
    print("%d finding(s)" % len(found))
    return 1 if found else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
