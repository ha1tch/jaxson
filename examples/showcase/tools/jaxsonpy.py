"""
jaxsonpy.py -- an independent Python reference executor for Jaxson 1.0,
ported directly from this repo's pkg/jaxson Go source (values.go,
machine.go, compute.go, schema.go, program.go), for use as a cross-check
in an environment with no Go toolchain available.

Not a product, not a substitute for the Go implementation or for
running go test against pkg/jaxson/jaxson-v0.1.0-fixtures.json -- a
second, independently-written implementation of the same spec, used
here only to execute a batch of new example packages and record their
actual output as each one's "expect" block.

Numbers: Python's fractions.Fraction, exactly analogous to math/big.Rat.
Values: None/bool/str/list/dict/Fraction, exactly analogous to the Go
side's any/*big.Rat.
"""
from fractions import Fraction
import json


class JaxsonError(Exception):
    def __init__(self, cat, code, msg=""):
        self.cat = cat
        self.code = code
        self.msg = msg
        super().__init__(f"{cat}/{code}: {msg}")


def fail(cat, code, msg=""):
    raise JaxsonError(cat, code, msg)


def exec_fail(code, msg=""):
    fail("EXECUTION_ERROR", code, msg)


def prog_fail(msg=""):
    fail("PROGRAM_ERROR", "", msg)


MAX_DIGITS = 38


# ---------------------------------------------------------------- numbers

def dec_parts(r: Fraction):
    n = r
    s = 0
    while n.denominator != 1:
        n *= 10
        s += 1
        if s > 200:
            return None, 0
    return n.numerator, s


def check_num(r: Fraction) -> Fraction:
    coeff, _ = dec_parts(r)
    if coeff is None or len(str(abs(coeff))) > MAX_DIGITS:
        exec_fail("NUMERIC_OVERFLOW", f"number exceeds {MAX_DIGITS} digits")
    return r


def format_decimal(r: Fraction) -> str:
    coeff, s = dec_parts(r)
    neg = coeff < 0
    d = str(abs(coeff))
    if s > 0:
        d = d.rjust(s + 1, "0")
        d = d[: len(d) - s] + "." + d[len(d) - s :]
    if neg:
        d = "-" + d
    return d


def round_rat(x: Fraction, scale: int, mode: str) -> Fraction:
    p = Fraction(10) ** scale
    m = x * p
    neg = m < 0
    a = abs(m)
    fl = a.numerator // a.denominator
    rem = a - fl
    half = Fraction(1, 2)
    if mode == "half_up":
        inc = rem >= half
    elif mode == "half_even":
        c = (rem > half) - (rem < half)
        inc = c > 0 or (c == 0 and fl % 2 == 1)
    elif mode == "down":
        inc = False
    else:
        exec_fail("BAD_MODE", f"unknown rounding mode {mode!r}")
    if inc:
        fl += 1
    if neg:
        fl = -fl
    return Fraction(fl, p.numerator) if p.denominator == 1 else Fraction(fl) / p


# ---------------------------------------------------------------- values

def clone(v):
    if isinstance(v, list):
        return [clone(x) for x in v]
    if isinstance(v, dict):
        return {k: clone(x) for k, x in v.items()}
    return v  # Fraction, str, bool, None are all immutable / copied by value


def jx_equal(a, b) -> bool:
    if a is None:
        return b is None
    if isinstance(a, bool):
        return isinstance(b, bool) and a == b
    if isinstance(a, str):
        return isinstance(b, str) and a == b
    if isinstance(a, Fraction):
        return isinstance(b, Fraction) and a == b
    if isinstance(a, list):
        if not isinstance(b, list) or len(a) != len(b):
            return False
        return all(jx_equal(x, y) for x, y in zip(a, b))
    if isinstance(a, dict):
        if not isinstance(b, dict) or len(a) != len(b):
            return False
        for k, xv in a.items():
            if k not in b or not jx_equal(xv, b[k]):
                return False
        return True
    return False


def type_name(v) -> str:
    if v is None:
        return "null"
    if isinstance(v, bool):
        return "boolean"
    if isinstance(v, Fraction):
        return "number"
    if isinstance(v, str):
        return "string"
    if isinstance(v, list):
        return "array"
    if isinstance(v, dict):
        return "object"
    return "unknown"


def sorted_keys(m: dict):
    return sorted(m.keys())


def jx_order(a, b) -> int:
    if isinstance(a, Fraction) and isinstance(b, Fraction):
        return (a > b) - (a < b)
    if isinstance(a, str) and isinstance(b, str):
        return (a > b) - (a < b)
    exec_fail("TYPE_ERROR", f"cannot order {type_name(a)} and {type_name(b)}")


def show(v) -> str:
    if v is None:
        return "null"
    if isinstance(v, bool):
        return "true" if v else "false"
    if isinstance(v, str):
        return json.dumps(v, ensure_ascii=False)
    if isinstance(v, Fraction):
        return format_decimal(v)
    if isinstance(v, list):
        return "[" + ",".join(show(x) for x in v) + "]"
    if isinstance(v, dict):
        parts = []
        for k in sorted_keys(v):
            parts.append(json.dumps(k, ensure_ascii=False) + ":" + show(v[k]))
        return "{" + ",".join(parts) + "}"
    return "?"


# ---------------------------------------------------------------- compute operators

def num(x) -> Fraction:
    if not isinstance(x, Fraction):
        exec_fail("TYPE_ERROR", f"expected number, got {type_name(x)}")
    return x


def s_(x) -> str:
    if not isinstance(x, str):
        exec_fail("TYPE_ERROR", f"expected string, got {type_name(x)}")
    return x


def boo(x) -> bool:
    if not isinstance(x, bool):
        exec_fail("TYPE_ERROR", f"expected boolean, got {type_name(x)}")
    return x


def integer(x) -> int:
    r = num(x)
    if r.denominator != 1:
        exec_fail("TYPE_ERROR", f"expected integer, got {format_decimal(r)}")
    return r.numerator


def scale_of(x) -> int:
    r = num(x)
    if r.denominator != 1 or r < 0 or r > MAX_DIGITS:
        exec_fail("BAD_SCALE", f"scale must be an integer from 0 to {MAX_DIGITS}")
    return r.numerator


def lookup(container, key):
    if isinstance(container, dict):
        k = s_(key)
        return container.get(k), k in container
    if isinstance(container, list):
        i = to_index(num(key))
        if i < len(container):
            return container[i], True
        return None, False
    exec_fail("TYPE_ERROR", f"cannot look up in {type_name(container)}")


def to_index(r: Fraction) -> int:
    if r.denominator != 1 or r < 0 or r.numerator.bit_length() > 31:
        exec_fail("BAD_INDEX", f"invalid index {format_decimal(r)}")
    return r.numerator


def has_get_or(op, args):
    val, found = lookup(args[0], args[1])
    if op == "has":
        return found
    if found:
        return val
    if op == "get_or":
        return args[2]
    exec_fail("MISSING_PATH", "get: key not found")


def min_max(op, args):
    best = num(args[0])
    for x in args[1:]:
        c = jx_order(num(x), best)
        if (op == "min" and c < 0) or (op == "max" and c > 0):
            best = num(x)
    return Fraction(best)


def apply_op(op, args):
    if op == "and":
        return all(args)
    if op == "or":
        return any(args)
    if op == "add":
        s = Fraction(0)
        for x in args:
            s += num(x)
        return check_num(s)
    if op == "sub":
        s = Fraction(num(args[0]))
        for x in args[1:]:
            s -= num(x)
        return check_num(s)
    if op == "mul":
        s = Fraction(1)
        for x in args:
            s *= num(x)
        return check_num(s)
    if op == "neg":
        return -num(args[0])
    if op == "abs":
        return abs(num(args[0]))
    if op == "min":
        return min_max("min", args)
    if op == "max":
        return min_max("max", args)
    if op == "mod":
        x, y = integer(args[0]), integer(args[1])
        if y == 0:
            exec_fail("DIV_ZERO", "modulo by zero")
        # Go's big.Int.Rem: truncated division, remainder takes sign of dividend
        r = abs(x) % abs(y)
        if x < 0:
            r = -r
        return Fraction(r)
    if op == "div":
        x, y = num(args[0]), num(args[1])
        if y == 0:
            exec_fail("DIV_ZERO", "division by zero")
        mode = s_(args[3]) if len(args) == 4 else "half_even"
        return check_num(round_rat(x / y, scale_of(args[2]), mode))
    if op == "round":
        mode = s_(args[2]) if len(args) == 3 else "half_even"
        return check_num(round_rat(num(args[0]), scale_of(args[1]), mode))
    if op == "eq":
        return jx_equal(args[0], args[1])
    if op == "ne":
        return not jx_equal(args[0], args[1])
    if op == "lt":
        return jx_order(args[0], args[1]) < 0
    if op == "le":
        return jx_order(args[0], args[1]) <= 0
    if op == "gt":
        return jx_order(args[0], args[1]) > 0
    if op == "ge":
        return jx_order(args[0], args[1]) >= 0
    if op == "not":
        return not boo(args[0])
    if op == "concat":
        return "".join(s_(x) for x in args)
    if op == "len":
        v = args[0]
        if isinstance(v, str):
            return Fraction(len(v))
        if isinstance(v, (list, dict)):
            return Fraction(len(v))
        exec_fail("TYPE_ERROR", f"len of {type_name(v)}")
    if op == "to_string":
        return format_decimal(num(args[0]))
    if op == "to_number":
        st = s_(args[0])
        import re as _re

        if not _re.match(r"^-?(0|[1-9][0-9]*)(\.[0-9]+)?$", st):
            exec_fail("BAD_NUMBER", f"not a JSON number: {st!r}")
        return check_num(Fraction(st))
    if op == "list":
        return list(args)
    if op == "keys":
        v = args[0]
        if not isinstance(v, dict):
            exec_fail("TYPE_ERROR", f"keys of {type_name(v)}")
        return sorted_keys(v)
    if op == "type_of":
        return type_name(args[0])
    if op == "has":
        return has_get_or("has", args)
    if op == "get":
        return has_get_or("get", args)
    if op == "get_or":
        return has_get_or("get_or", args)
    raise AssertionError(f"unimplemented op {op}")


LAZY_OPS = {"and", "or", "select"}


def eval_expr(e, env):
    if isinstance(e, dict):
        return env[e["$v"]]
    if isinstance(e, list):
        op = e[0]
        args_raw = e[1:]
        if op == "and":
            for a in args_raw:
                if not boo(eval_expr(a, env)):
                    return False
            return True
        if op == "or":
            for a in args_raw:
                if boo(eval_expr(a, env)):
                    return True
            return False
        if op == "select":
            cond, then, els = args_raw
            return eval_expr(then, env) if boo(eval_expr(cond, env)) else eval_expr(els, env)
        args = [eval_expr(a, env) for a in args_raw]
        return apply_op(op, args)
    return e  # scalar literal (nil/bool/string/number)


# ---------------------------------------------------------------- machine

class Machine:
    def __init__(self, input_, state, output, limit):
        self.input = input_
        self.state = state
        self.output = output
        self.locals = {}
        self.steps = 0
        self.limit = limit

    def step(self):
        self.steps += 1
        if self.steps > self.limit:
            fail("RESOURCE_ERROR", "STEPS", f"step limit {self.limit} exceeded")

    def root(self, name):
        return {"input": self.input, "state": self.state, "output": self.output}[name]

    def walk(self, root, segs):
        if root == "local":
            cur = self.locals[segs[0]]
            segs = segs[1:]
        else:
            cur = self.root(root)
        for s in segs:
            if isinstance(s, str):
                if not isinstance(cur, dict):
                    exec_fail("TYPE_ERROR", f"cannot read member {s!r} of {type_name(cur)}")
                if s not in cur:
                    exec_fail("MISSING_PATH", f"member {s!r} not found")
                cur = cur[s]
            else:  # int index
                if not isinstance(cur, list):
                    exec_fail("TYPE_ERROR", f"cannot index {type_name(cur)}")
                if s >= len(cur):
                    exec_fail("MISSING_PATH", f"index {s} out of range")
                cur = cur[s]
        return cur

    def get_at(self, root, segs):
        return self.walk(root, segs)

    def put_at(self, root, segs, v):
        if len(segs) == 0:
            if root == "state":
                self.state = v
            elif root == "output":
                self.output = v
            return
        par = self.get_at(root, segs[:-1])
        last = segs[-1]
        if isinstance(last, str):
            if not isinstance(par, dict):
                exec_fail("TYPE_ERROR", f"cannot set member {last!r} on {type_name(par)}")
            par[last] = v
        else:
            if not isinstance(par, list):
                exec_fail("TYPE_ERROR", f"cannot set an index on {type_name(par)}")
            if last >= len(par):
                exec_fail("BAD_INDEX", f"index {last} out of range")
            par[last] = v

    def segs(self, path):
        root = path[0]
        out = []
        for s in path[1:]:
            v = self.eval(s) if isinstance(s, dict) else s
            if isinstance(v, str):
                out.append(v)
            elif isinstance(v, Fraction):
                out.append(to_index(v))
            else:
                exec_fail("TYPE_ERROR", f"a path segment must be a string or an integer, got {type_name(v)}")
        return root, out

    def eval(self, x):
        if isinstance(x, dict) and len(x) == 1:
            (k, v), = x.items()
            if k == "$path":
                return self.read(v)
            if k == "$lit":
                return clone(v)
            if k == "$compute":
                return clone(self.compute(v))
            if k == "$tpl":
                return self.tpl(v)
        return clone(x)

    def read(self, path):
        r, s = self.segs(path)
        return clone(self.get_at(r, s))

    def tpl(self, v):
        if isinstance(v, dict) and len(v) == 1:
            (k, a), = v.items()
            if k == "$opt":
                try:
                    r, s = self.segs(a)
                    val = self.walk(r, s)
                except JaxsonError as e:
                    if e.code == "MISSING_PATH":
                        return _OMIT
                    raise
                return clone(val)
            if isinstance(k, str) and k.startswith("$"):
                return self.eval(v)
        if isinstance(v, dict):
            out = {}
            for k, a in v.items():
                r = self.tpl(a)
                if r is _OMIT:
                    continue
                out[k] = r
            return out
        if isinstance(v, list):
            out = []
            for a in v:
                r = self.tpl(a)
                if r is _OMIT:
                    continue
                out.append(r)
            return out
        return clone(v)

    def compute(self, c):
        env = {}
        for n in sorted_keys(c.get("with", {})):
            env[n] = self.eval(c["with"][n])
        return self.expr(c["expr"], env)

    def expr(self, e, env):
        if isinstance(e, dict):
            return env[e["$v"]]
        if isinstance(e, list):
            self.step()
            return self.apply(e[0], e[1:], env)
        return e

    def apply(self, op, arg_forms, env):
        if op == "and":
            for a in arg_forms:
                if not boo(self.expr(a, env)):
                    return False
            return True
        if op == "or":
            for a in arg_forms:
                if boo(self.expr(a, env)):
                    return True
            return False
        if op == "select":
            cond, then, els = arg_forms
            return self.expr(then, env) if boo(self.expr(cond, env)) else self.expr(els, env)
        args = [self.expr(a, env) for a in arg_forms]
        return apply_op(op, args)

    def run(self, block):
        for instr in block:
            self.step()
            op = instr["op"]
            if op == "set":
                val = self.eval(instr["value"])
                r, s = self.segs(instr["path"])
                self.put_at(r, s, val)
            elif op == "append":
                val = self.eval(instr["value"])
                r, s = self.segs(instr["path"])
                ar = self.get_at(r, s)
                if not isinstance(ar, list):
                    exec_fail("TYPE_ERROR", "append target is not an array")
                new_ar = ar + [val]
                self.put_at(r, s, new_ar)
            elif op == "insert":
                val = self.eval(instr["value"])
                at = to_index(num(self.eval(instr["at"])))
                r, s = self.segs(instr["path"])
                ar = self.get_at(r, s)
                if not isinstance(ar, list):
                    exec_fail("TYPE_ERROR", "insert target is not an array")
                if at > len(ar):
                    exec_fail("BAD_INDEX", f"insert position {at} beyond length {len(ar)}")
                new_ar = ar[:at] + [val] + ar[at:]
                self.put_at(r, s, new_ar)
            elif op == "delete":
                path = instr["path"]
                r, s = self.segs(path)
                par = self.get_at(r, s[:-1])
                last = s[-1]
                if isinstance(last, str):
                    if not isinstance(par, dict):
                        exec_fail("TYPE_ERROR", f"delete member of {type_name(par)}")
                    if last not in par:
                        exec_fail("MISSING_PATH", f"member {last!r} not found")
                    del par[last]
                else:
                    if not isinstance(par, list):
                        exec_fail("TYPE_ERROR", f"delete index of {type_name(par)}")
                    if last >= len(par):
                        exec_fail("BAD_INDEX", f"index {last} out of range")
                    new_ar = par[:last] + par[last + 1 :]
                    self.put_at(r, s[:-1], new_ar)
            elif op == "if":
                if boo(self.eval(instr["cond"])):
                    self.run(instr["then"])
                elif "else" in instr:
                    self.run(instr["else"])
            elif op == "for":
                arr = self.eval(instr["in"])
                if not isinstance(arr, list):
                    exec_fail("TYPE_ERROR", "for: in must be an array")
                as_ = instr["as"]
                idx = instr.get("index")
                for i, el in enumerate(arr):
                    self.step()
                    self.locals[as_] = el
                    if idx:
                        self.locals[idx] = Fraction(i)
                    self.run(instr["do"])
                del self.locals[as_]
                if idx:
                    del self.locals[idx]
            elif op == "assert":
                if not boo(self.eval(instr["that"])):
                    exec_fail("ASSERTION_FAILED", instr.get("msg", ""))
            elif op == "halt":
                raise _Halt()
            else:
                exec_fail("TYPE_ERROR", f"unknown op {op!r} at execution")


class _Halt(Exception):
    pass


class _Omit:
    pass


_OMIT = _Omit()


# ---------------------------------------------------------------- static checking
# Ported from paths.go / operands.go / program.go's checkBlock. Not
# every diagnostic message matches the Go source verbatim, but every
# PROGRAM_ERROR-triggering condition the spec lists (section 10) is
# checked, so a program that passes this also passes the real checker,
# and a program the real checker would reject is rejected here too.

import re as _re_mod

_NAME_RE = _re_mod.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")

ARITY = {
    "add": (2, -1), "sub": (2, -1), "mul": (2, -1), "neg": (1, 1), "abs": (1, 1),
    "min": (1, -1), "max": (1, -1), "mod": (2, 2), "div": (3, 4), "round": (2, 3),
    "eq": (2, 2), "ne": (2, 2), "lt": (2, 2), "le": (2, 2), "gt": (2, 2), "ge": (2, 2),
    "and": (1, -1), "or": (1, -1), "not": (1, 1), "select": (3, 3),
    "concat": (1, -1), "len": (1, 1), "to_string": (1, 1), "to_number": (1, 1),
    "list": (0, -1), "get": (2, 2), "has": (2, 2), "get_or": (3, 3), "keys": (1, 1), "type_of": (1, 1),
}

ROOTS = {"input", "state", "output", "local"}

INSTR_FIELDS = {
    "set": (["path", "value"], []),
    "append": (["path", "value"], []),
    "insert": (["path", "at", "value"], []),
    "delete": (["path"], []),
    "if": (["cond", "then"], ["else"]),
    "for": (["in", "as", "do"], ["index"]),
    "assert": (["that"], ["msg"]),
    "halt": ([], []),
}


def contains_form(x):
    if isinstance(x, dict):
        for k, v in x.items():
            if isinstance(k, str) and k.startswith("$"):
                return True
            if contains_form(v):
                return True
        return False
    if isinstance(x, list):
        return any(contains_form(v) for v in x)
    return False


def check_path(p, locals_, write):
    if not isinstance(p, list) or len(p) == 0:
        prog_fail("a path must be a non-empty array")
    root = p[0]
    if not isinstance(root, str) or root not in ROOTS:
        prog_fail("bad path root")
    if write and root not in ("state", "output"):
        prog_fail(f"cannot write to root {root!r}")
    if root == "local":
        if len(p) < 2 or not isinstance(p[1], str):
            prog_fail("a local path needs a name")
        if p[1] not in locals_:
            prog_fail("undeclared local")
    for s in p[1:]:
        if isinstance(s, str):
            continue
        if isinstance(s, Fraction):
            if s.denominator != 1 or s < 0:
                prog_fail("a literal index must be a non-negative integer")
            continue
        if isinstance(s, dict):
            if not any(isinstance(k, str) and k.startswith("$") for k in s):
                prog_fail("a path segment must be a string, an index or a form")
            check_operand(s, locals_)
            continue
        prog_fail("bad path segment")


def check_operand(x, locals_):
    if isinstance(x, dict) and any(isinstance(k, str) and k.startswith("$") for k in x):
        check_form(x, locals_)
        return
    if contains_form(x):
        prog_fail("a literal contains a $-form; use $tpl or $lit")


def check_form(mp, locals_):
    if len(mp) != 1:
        prog_fail("a $-form must have exactly one key")
    (k, v), = mp.items()
    if k == "$path":
        check_path(v, locals_, False)
    elif k == "$lit":
        pass
    elif k == "$compute":
        check_compute(v, locals_)
    elif k == "$tpl":
        check_tpl(v, locals_, True)
    elif k == "$opt":
        prog_fail("$opt is only valid inside $tpl")
    elif k == "$v":
        prog_fail("$v is only valid inside a compute expr")
    else:
        prog_fail(f"unknown form {k!r}")


def check_tpl(v, locals_, top):
    if isinstance(v, dict):
        if len(v) == 1:
            (k, a), = v.items()
            if k == "$opt":
                if top:
                    prog_fail("a $tpl root cannot be $opt")
                check_path(a, locals_, False)
                return
            if isinstance(k, str) and k.startswith("$"):
                check_form(v, locals_)
                return
        for k, a in v.items():
            if isinstance(k, str) and k.startswith("$"):
                prog_fail("a $-key must be the only key of its object")
            check_tpl(a, locals_, False)
    elif isinstance(v, list):
        for a in v:
            check_tpl(a, locals_, False)


def check_compute(v, locals_):
    if not isinstance(v, dict):
        prog_fail("$compute must be an object")
    for k in v:
        if k not in ("with", "expr"):
            prog_fail(f"$compute: unknown key {k!r}")
    if "expr" not in v:
        prog_fail("$compute needs an expr")
    names = set()
    if "with" in v:
        wm = v["with"]
        if not isinstance(wm, dict):
            prog_fail("$compute: with must be an object")
        for n, o in wm.items():
            if not _NAME_RE.match(n):
                prog_fail(f"$compute: bad binding name {n!r}")
            names.add(n)
            check_operand(o, locals_)
    check_expr(v["expr"], names)


def check_expr(e, names):
    if e is None or isinstance(e, (bool, str, Fraction)):
        return
    if isinstance(e, dict):
        if len(e) != 1 or "$v" not in e or e["$v"] not in names:
            prog_fail("bad $v reference in expr")
        return
    if isinstance(e, list):
        if len(e) == 0:
            prog_fail("an empty application in expr")
        op = e[0]
        if not isinstance(op, str) or op not in ARITY:
            prog_fail(f"unknown compute operation {op!r}")
        lo, hi = ARITY[op]
        n = len(e) - 1
        if n < lo or (hi >= 0 and n > hi):
            prog_fail(f"wrong arity for {op}")
        for a in e[1:]:
            check_expr(a, names)


def check_block(block, locals_):
    if not isinstance(block, list):
        prog_fail("a block must be an array")
    for instr in block:
        if not isinstance(instr, dict):
            prog_fail("an instruction must be an object")
        op = instr.get("op")
        if not isinstance(op, str):
            prog_fail("instruction without a string op")
        if op not in INSTR_FIELDS:
            prog_fail(f"unknown op {op!r}")
        req, opt = INSTR_FIELDS[op]
        allowed = {"op", *req, *opt}
        for f in req:
            if f not in instr:
                prog_fail(f"{op}: missing field {f!r}")
        for k in instr:
            if k not in allowed:
                prog_fail(f"{op}: unknown field {k!r}")
        if op in ("set", "append", "insert", "delete"):
            check_path(instr["path"], locals_, True)
            if op == "delete" and len(instr["path"]) < 2:
                prog_fail("delete needs a path below a root")
            if "value" in instr:
                check_operand(instr["value"], locals_)
            if op == "insert":
                check_operand(instr["at"], locals_)
        elif op == "if":
            check_operand(instr["cond"], locals_)
            check_block(instr["then"], locals_)
            if "else" in instr:
                check_block(instr["else"], locals_)
        elif op == "for":
            check_operand(instr["in"], locals_)
            as_ = instr.get("as")
            if not isinstance(as_, str) or not _NAME_RE.match(as_) or as_ in locals_:
                prog_fail("for: bad or shadowed binding name")
            inner = set(locals_)
            inner.add(as_)
            if "index" in instr:
                ix = instr["index"]
                if not isinstance(ix, str) or not _NAME_RE.match(ix) or ix in locals_ or ix == as_:
                    prog_fail("for: bad or shadowed index name")
                inner.add(ix)
            check_block(instr["do"], inner)
        elif op == "assert":
            check_operand(instr["that"], locals_)
            if "msg" in instr and not isinstance(instr["msg"], str):
                prog_fail("assert: msg must be a string")
        elif op == "halt":
            pass


def check_program(program):
    check_block(program, set())


# ---------------------------------------------------------------- schema

SCHEMA_KEYS = {
    "any": [], "null": [], "boolean": [],
    "number": ["int", "min", "max"], "string": ["minLen", "maxLen", "enum"],
    "array": ["items", "minItems", "maxItems"], "object": ["fields", "required", "extra"],
}


def check_schema(s):
    if not isinstance(s, dict):
        fail("SCHEMA_ERROR", "", "a schema must be an object")
    if "anyOf" in s:
        for k in s:
            if k not in ("anyOf", "nullable"):
                fail("SCHEMA_ERROR", "", f"anyOf schema has extra keyword {k!r}")
        arr = s["anyOf"]
        if not isinstance(arr, list) or len(arr) == 0:
            fail("SCHEMA_ERROR", "", "anyOf needs a non-empty array")
        for sub in arr:
            check_schema(sub)
        return
    t = s.get("type")
    if t not in SCHEMA_KEYS:
        fail("SCHEMA_ERROR", "", "unknown or missing schema type")
    allowed = {"type", "nullable"}
    allowed.update(SCHEMA_KEYS[t])
    for k in s:
        if k not in allowed:
            fail("SCHEMA_ERROR", "", f"keyword {k!r} is not valid for type {t}")
    if t == "array" and "items" in s:
        check_schema(s["items"])
    if t == "object":
        fields = s.get("fields", {})
        for f in fields.values():
            check_schema(f)
        if "required" in s:
            req = s["required"]
            if not isinstance(req, list):
                fail("SCHEMA_ERROR", "", "required must be an array")
            for n in req:
                if n not in fields:
                    fail("SCHEMA_ERROR", "", "required names an undeclared field")
        if "extra" in s and s["extra"] not in ("reject", "allow"):
            fail("SCHEMA_ERROR", "", "extra must be reject or allow")


def rat_of(v):
    return v if isinstance(v, Fraction) else None


def check_number(r, want_int, mn, mx):
    if want_int and r.denominator != 1:
        return "expected an integer"
    if mn is not None and r < mn:
        return "below minimum"
    if mx is not None and r > mx:
        return "above maximum"
    return ""


def check_string_len(s, min_len, max_len, enum):
    n = len(s)
    if min_len is not None and n < min_len:
        return "too short"
    if max_len is not None and n > max_len:
        return "too long"
    if enum is not None:
        if s not in enum:
            return "not in enum"
    return ""


def check_array_len(n, min_items, max_items):
    if min_items is not None and n < min_items:
        return "too few items"
    if max_items is not None and n > max_items:
        return "too many items"
    return ""


def validate(s, v, path):
    if s.get("nullable") and v is None:
        return ""
    if "anyOf" in s:
        for sub in s["anyOf"]:
            if validate(sub, v, path) == "":
                return ""
        return f"{path}: matches no anyOf alternative"
    t = s["type"]
    if t == "any":
        return ""
    got = type_name(v)
    if got != t:
        return f"{path}: expected {t}, got {got}"
    if t == "number":
        msg = check_number(v, s.get("int", False), rat_of(s.get("min")), rat_of(s.get("max")))
        if msg:
            return f"{path}: {msg}"
    elif t == "string":
        enum = s.get("enum")
        msg = check_string_len(v, rat_of(s.get("minLen")), rat_of(s.get("maxLen")), enum)
        if msg:
            return f"{path}: {msg}"
    elif t == "array":
        msg = check_array_len(len(v), rat_of(s.get("minItems")), rat_of(s.get("maxItems")))
        if msg:
            return f"{path}: {msg}"
        if "items" in s:
            for i, e in enumerate(v):
                msg = validate(s["items"], e, f"{path}[{i}]")
                if msg:
                    return msg
    elif t == "object":
        fields = s.get("fields", {})
        if "required" in s:
            for n in s["required"]:
                if n not in v:
                    return f"{path}: missing required {n!r}"
        for k in sorted_keys(v):
            if k in fields:
                msg = validate(fields[k], v[k], f"{path}.{k}")
                if msg:
                    return msg
            elif s.get("extra") != "allow":
                return f"{path}: unexpected member {k!r}"
    return ""


# ---------------------------------------------------------------- top level

def normalize(v):
    return v  # numbers already parsed as Fraction; nothing else to convert


def run_package(pkg):
    """Runs a Jaxson package (already parsed with Fraction numbers).
    Returns (output, None) on success or (None, JaxsonError) on failure."""
    try:
        if pkg.get("jaxson") != "1.0":
            fail("VERSION_ERROR", "", "unsupported or missing jaxson version")
        check_schema(pkg["inputSchema"])
        check_schema(pkg["outputSchema"])
        check_program(pkg["program"])
        limit = pkg.get("limits", {}).get("steps", 100000)
        if isinstance(limit, Fraction):
            limit = int(limit)
        msg = validate(pkg["inputSchema"], pkg["input"], "$")
        if msg:
            fail("INPUT_ERROR", "", msg)
        m = Machine(clone(pkg["input"]), {}, None, limit)
        try:
            m.run(pkg["program"])
        except _Halt:
            pass
        msg = validate(pkg["outputSchema"], m.output, "$")
        if msg:
            fail("OUTPUT_ERROR", "", msg)
        return m.output, None
    except JaxsonError as e:
        return None, e


def loads(text):
    return json.loads(text, parse_float=Fraction, parse_int=Fraction)


def to_jsonable(v):
    """Converts Fraction back to plain int/float-free JSON-serialisable
    form for dumping: integers as int, non-integers as a float-tagged
    marker string handled by our custom encoder below."""
    if isinstance(v, Fraction):
        return _NumberLiteral(format_decimal(v))
    if isinstance(v, list):
        return [to_jsonable(x) for x in v]
    if isinstance(v, dict):
        return {k: to_jsonable(x) for k, x in v.items()}
    return v


class _NumberLiteral(str):
    """A string that json.dumps should emit unquoted, verbatim -- used
    to serialise our exact decimals without going through float."""


class ExactJSONEncoder(json.JSONEncoder):
    def default(self, o):
        if isinstance(o, _NumberLiteral):
            return o
        return super().default(o)

    def iterencode(self, o, _one_shot=False):
        # Intercept _NumberLiteral before the base encoder stringifies it.
        def fix(x):
            if isinstance(x, _NumberLiteral):
                return _Raw(str(x))
            if isinstance(x, dict):
                return {k: fix(v) for k, v in x.items()}
            if isinstance(x, list):
                return [fix(v) for v in x]
            return x

        return super().iterencode(fix(o), _one_shot)


class _Raw(float):
    """Cheap trick: subclass float so json emits it unquoted, but override
    __repr__ via a wrapping approach isn't reliable across json versions,
    so instead we special-case dumping ourselves (see dumps_exact)."""


def dumps_exact(v, **kwargs):
    """Serialise a value (with Fraction numbers) to JSON text, emitting
    numbers via format_decimal rather than Python float repr."""

    def enc(x):
        if isinstance(x, Fraction):
            return format_decimal(x)
        if isinstance(x, bool):
            return "true" if x else "false"
        if x is None:
            return "null"
        if isinstance(x, str):
            return json.dumps(x, ensure_ascii=False)
        if isinstance(x, list):
            return "[" + ",".join(enc(e) for e in x) + "]"
        if isinstance(x, dict):
            return "{" + ",".join(json.dumps(k, ensure_ascii=False) + ":" + enc(v) for k, v in x.items()) + "}"
        raise TypeError(type(x))

    return enc(v)
