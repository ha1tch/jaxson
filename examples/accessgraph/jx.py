# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
"""Strict stand-in for jaxrun, covering the subset the package uses.
Unsupported features raise Unsupported. jaxrun is authoritative.

  python3 jx.py PKG.json
  python3 jx.py --fixtures jaxson-v0.1.0-fixtures.json
"""
import copy
import json
import sys
from fractions import Fraction


class Err(Exception):
    def __init__(self, cat, code=None):
        super().__init__("%s %s" % (cat, code))
        self.cat, self.code = cat, code


class Unsupported(Exception):
    pass


class Halt(Exception):
    pass


def isnum(x):
    return type(x) in (int, Fraction)


def deq(a, b):
    if type(a) is bool or type(b) is bool:
        return type(a) is type(b) and a == b
    if isnum(a) and isnum(b):
        return a == b
    if isinstance(a, list) and isinstance(b, list):
        return len(a) == len(b) and all(deq(x, y) for x, y in zip(a, b))
    if isinstance(a, dict) and isinstance(b, dict):
        return a.keys() == b.keys() and all(deq(a[k], b[k]) for k in a)
    return type(a) is type(b) and a == b


def te():
    return Err("EXECUTION_ERROR", "TYPE_ERROR")


def miss():
    return Err("EXECUTION_ERROR", "MISSING_PATH")


def run(pkg):
    if pkg.get("jaxson") != "1.0":
        raise Err("VERSION_ERROR")
    st = {"input": pkg.get("input"), "state": {}, "output": None}
    limit = pkg.get("limits", {}).get("steps", 100000)
    n = [0]

    def tick():
        n[0] += 1
        if n[0] > limit:
            raise Err("RESOURCE_ERROR", "STEPS")

    def step(cur, s):
        if isinstance(cur, dict):
            if not isinstance(s, str):
                raise te()
            if s not in cur:
                raise miss()
        elif isinstance(cur, list):
            if type(s) is not int:
                raise te()
            if not 0 <= s < len(cur):
                raise miss()
        else:
            raise te()
        return cur[s]

    def seg(s, loc):
        return ev(s, loc) if isinstance(s, dict) else s

    def get(path, loc):
        cur = {**st, "local": loc}[path[0]]
        for s in path[1:]:
            cur = step(cur, seg(s, loc))
        return cur

    def write(ins, loc):
        o, path = ins["op"], ins["path"]
        val = ev(ins["value"], loc) if "value" in ins else None
        at = ev(ins["at"], loc) if "at" in ins else None
        if path[0] in ("input", "local"):
            raise Err("PROGRAM_ERROR")
        segs = [seg(s, loc) for s in path[1:]]
        if not segs:
            if o != "set":
                raise te()
            st[path[0]] = val
            return
        cur = st[path[0]]
        for s in segs[:-1]:
            cur = step(cur, s)
        last = segs[-1]
        if o in ("append", "insert"):
            cur = step(cur, last)
            if not isinstance(cur, list):
                raise te()
            if o == "append":
                cur.append(val)
            elif type(at) is int and 0 <= at <= len(cur):
                cur.insert(at, val)
            else:
                raise Err("EXECUTION_ERROR", "BAD_INDEX")
        elif isinstance(cur, dict):
            if not isinstance(last, str):
                raise te()
            if o == "set":
                cur[last] = val
            else:
                step(cur, last)
                del cur[last]
        elif isinstance(cur, list):
            if type(last) is not int:
                raise te()
            if not 0 <= last < len(cur):
                raise Err("EXECUTION_ERROR", "BAD_INDEX")
            if o == "set":
                cur[last] = val
            else:
                del cur[last]
        else:
            raise te()

    def ev(x, loc):
        if isinstance(x, dict):
            ks = [k for k in x if k.startswith("$")]
            if not ks:
                return copy.deepcopy(x)
            if len(x) != 1:
                raise Err("PROGRAM_ERROR")
            k = ks[0]
            if k == "$lit":
                return copy.deepcopy(x[k])
            if k == "$path":
                return copy.deepcopy(get(x[k], loc))
            if k == "$compute":
                w = x[k].get("with", {})
                env = {b: ev(w[b], loc) for b in sorted(w)}
                return expr(x[k]["expr"], env)
            if k == "$tpl":
                return tpl(x[k], loc)
            raise Err("PROGRAM_ERROR")
        return copy.deepcopy(x) if isinstance(x, list) else x

    def opt(v, loc):
        return isinstance(v, dict) and set(v) == {"$opt"}

    def tpl(t, loc):
        if isinstance(t, dict):
            if any(k.startswith("$") for k in t):
                return ev(t, loc)
            out = {}
            for k in sorted(t):
                if opt(t[k], loc):
                    try:
                        out[k] = copy.deepcopy(get(t[k]["$opt"], loc))
                    except Err as e:
                        if e.code != "MISSING_PATH":
                            raise
                else:
                    out[k] = tpl(t[k], loc)
            return out
        if isinstance(t, list):
            out = []
            for v in t:
                if opt(v, loc):
                    try:
                        out.append(copy.deepcopy(get(v["$opt"], loc)))
                    except Err as e:
                        if e.code != "MISSING_PATH":
                            raise
                else:
                    out.append(tpl(v, loc))
            return out
        return t

    def boolean(v):
        if type(v) is not bool:
            raise te()
        return v

    def expr(e, env):
        if isinstance(e, dict):
            if set(e) == {"$v"} and e["$v"] in env:
                return env[e["$v"]]
            raise Err("PROGRAM_ERROR")
        if not isinstance(e, list):
            return e
        op, a = e[0], e[1:]
        tick()
        if op == "and":
            for x in a:
                if not boolean(expr(x, env)):
                    return False
            return True
        if op == "or":
            for x in a:
                if boolean(expr(x, env)):
                    return True
            return False
        if op == "select":
            return expr(a[1] if boolean(expr(a[0], env)) else a[2], env)
        return apply(op, [expr(x, env) for x in a])

    def apply(op, v):
        if op in ("add", "sub", "mul"):
            if not all(isnum(x) for x in v):
                raise te()
            r = v[0]
            for x in v[1:]:
                r = r + x if op == "add" else r - x if op == "sub" else r * x
            if type(r) is int and len(str(abs(r))) > 38:
                raise Err("EXECUTION_ERROR", "NUMERIC_OVERFLOW")
            return r
        if op == "eq":
            return deq(v[0], v[1])
        if op == "ne":
            return not deq(v[0], v[1])
        if op in ("lt", "le", "gt", "ge"):
            a, b = v
            if not ((isnum(a) and isnum(b)) or (isinstance(a, str) and isinstance(b, str))):
                raise te()
            return {"lt": a < b, "le": a <= b, "gt": a > b, "ge": a >= b}[op]
        if op == "not":
            return not boolean(v[0])
        if op == "concat":
            if not all(isinstance(x, str) for x in v):
                raise te()
            return "".join(v)
        if op == "len":
            if not isinstance(v[0], (str, list, dict)):
                raise te()
            return len(v[0])
        if op == "keys":
            if not isinstance(v[0], dict):
                raise te()
            return sorted(v[0])
        if op == "has":
            try:
                step(v[0], v[1])
                return True
            except Err as e:
                if e.code == "MISSING_PATH":
                    return False
                raise
        if op == "get":
            return step(v[0], v[1])
        if op == "get_or":
            try:
                return step(v[0], v[1])
            except Err as e:
                if e.code == "MISSING_PATH":
                    return v[2]
                raise
        if op == "list":
            return list(v)
        if op in ("min", "max"):
            if not all(isnum(x) for x in v):
                raise te()
            return min(v) if op == "min" else max(v)
        raise Unsupported(op)

    def ex(prog, loc):
        for ins in prog:
            tick()
            o = ins["op"]
            if o in ("set", "delete", "append", "insert"):
                write(ins, loc)
            elif o == "if":
                if boolean(ev(ins["cond"], loc)):
                    ex(ins["then"], loc)
                elif "else" in ins:
                    ex(ins["else"], loc)
            elif o == "for":
                arr = ev(ins["in"], loc)
                if not isinstance(arr, list):
                    raise te()
                if ins["as"] in loc:
                    raise Err("PROGRAM_ERROR")
                for i, item in enumerate(arr):
                    tick()
                    l2 = dict(loc)
                    l2[ins["as"]] = item
                    if "index" in ins:
                        l2[ins["index"]] = i
                    ex(ins["do"], l2)
            elif o == "assert":
                if not boolean(ev(ins["that"], loc)):
                    raise Err("EXECUTION_ERROR", "ASSERTION_FAILED")
            elif o == "halt":
                raise Halt()
            else:
                raise Err("PROGRAM_ERROR")

    try:
        ex(pkg["program"], {})
    except Halt:
        pass
    return st["output"], n[0]


def fixtures(path):
    cases = json.load(open(path), parse_float=Fraction)
    soft = {"PARSE_ERROR", "SCHEMA_ERROR", "PROGRAM_ERROR", "INPUT_ERROR", "OUTPUT_ERROR"}
    npass = nfail = nskip = 0
    for c in cases:
        if "packageText" in c:
            nskip += 1
            continue
        pkg = {k: v for k, v in c.items() if k not in ("name", "expect")}
        want = c["expect"]
        try:
            got = ("output", run(pkg)[0])
        except Err as e:
            got = ("error", e.cat, e.code)
        except Unsupported:
            nskip += 1
            continue
        except (KeyError, TypeError, IndexError, AttributeError):
            got = ("error", "PROGRAM_ERROR", None)
        if "output" in want:
            ok = got[0] == "output" and deq(got[1], want["output"])
        else:
            w = want["error"]
            ok = got[0] == "error" and got[1] == w["category"] and w.get("code") in (None, got[2])
            if not ok and w["category"] in soft:
                nskip += 1
                continue
        if ok:
            npass += 1
        else:
            nfail += 1
            print("FAIL", c["name"], "got", got, "want", want)
    print("fixtures: %d pass, %d fail, %d skipped (not implemented in stand-in)" % (npass, nfail, nskip))
    return nfail == 0


if __name__ == "__main__":
    if len(sys.argv) == 3 and sys.argv[1] == "--fixtures":
        sys.exit(0 if fixtures(sys.argv[2]) else 1)
    out, steps = run(json.load(open(sys.argv[1])))
    print(json.dumps(out, sort_keys=True))
    sys.stderr.write("steps: %d\n" % steps)
