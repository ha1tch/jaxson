// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Phase 1: the compute-operator table, extracted from what was previously
// one hardcoded switch in machine.go's apply(). Not required for
// v3.1 (no new operators) — jaxson-v0.1.0-core-design.md section 7
// anticipates a later named-profile operator, and the cost of adding this
// extension point now, while the file is already being restructured, is
// close to zero. Behaviour of every existing operator is unchanged.

import (
	"math/big"
	"strings"
	"unicode/utf8"
)

// OperatorDef is one compute operator. MinArity/MaxArity (-1 = unbounded)
// mirror the `arity` table in operands.go exactly — CoreOperators fills
// them from that same table below, so there is one source of truth, not
// two that could drift apart. Jaxson's own checkExpr still enforces arity
// internally before Apply ever runs; these fields exist so a host runtime
// (e.g. Shaxon, validating a `computes` entry at load time, with no
// Machine and no input yet) can check arity the same way without
// reimplementing this table itself. "and", "or", and "select" are lazy —
// Apply evaluates each argument itself, since they short-circuit — every
// other operator is eager: its Apply's first line is always
// `v := evalArgs(m, a, env)`.
type OperatorDef struct {
	Name               string
	MinArity, MaxArity int
	Apply              func(m *Machine, a []any, env map[string]any) any
}

func evalArgs(m *Machine, a []any, env map[string]any) []any {
	v := make([]any, len(a))
	for i, x := range a {
		v[i] = m.expr(x, env)
	}
	return v
}

func minMax(op string, v []any) any {
	best := num(v[0])
	for _, x := range v[1:] {
		c := num(x).Cmp(best)
		if (op == "min" && c < 0) || (op == "max" && c > 0) {
			best = num(x)
		}
	}
	return new(big.Rat).Set(best)
}

func hasGetOr(op string, v []any) any {
	val, found := lookup(v[0], v[1])
	if op == "has" {
		return found
	}
	if found {
		return val
	}
	if op == "get_or" {
		return v[2]
	}
	execFail("MISSING_PATH", "get: key not found")
	return nil
}

// CoreOperators returns Jaxson's compute operators, unchanged in
// behaviour from the original jaxrun.go.
func CoreOperators() map[string]OperatorDef {
	ops := map[string]OperatorDef{
		"and": {Name: "and", Apply: func(m *Machine, a []any, env map[string]any) any {
			for _, x := range a {
				if !boo(m.expr(x, env)) {
					return false
				}
			}
			return true
		}},
		"or": {Name: "or", Apply: func(m *Machine, a []any, env map[string]any) any {
			for _, x := range a {
				if boo(m.expr(x, env)) {
					return true
				}
			}
			return false
		}},
		"select": {Name: "select", Apply: func(m *Machine, a []any, env map[string]any) any {
			if boo(m.expr(a[0], env)) {
				return m.expr(a[1], env)
			}
			return m.expr(a[2], env)
		}},
		"add": {Name: "add", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			s := new(big.Rat)
			for _, x := range v {
				s.Add(s, num(x))
			}
			return checkNum(s)
		}},
		"sub": {Name: "sub", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			s := new(big.Rat).Set(num(v[0]))
			for _, x := range v[1:] {
				s.Sub(s, num(x))
			}
			return checkNum(s)
		}},
		"mul": {Name: "mul", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			s := big.NewRat(1, 1)
			for _, x := range v {
				s.Mul(s, num(x))
			}
			return checkNum(s)
		}},
		"neg": {Name: "neg", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return new(big.Rat).Neg(num(v[0]))
		}},
		"abs": {Name: "abs", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return new(big.Rat).Abs(num(v[0]))
		}},
		"min": {Name: "min", Apply: func(m *Machine, a []any, env map[string]any) any {
			return minMax("min", evalArgs(m, a, env))
		}},
		"max": {Name: "max", Apply: func(m *Machine, a []any, env map[string]any) any {
			return minMax("max", evalArgs(m, a, env))
		}},
		"mod": {Name: "mod", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			x, y := integer(v[0]), integer(v[1])
			if y.Sign() == 0 {
				execFail("DIV_ZERO", "modulo by zero")
			}
			return new(big.Rat).SetInt(new(big.Int).Rem(x, y))
		}},
		"div": {Name: "div", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			x, y := num(v[0]), num(v[1])
			if y.Sign() == 0 {
				execFail("DIV_ZERO", "division by zero")
			}
			mode := "half_even"
			if len(v) == 4 {
				mode = str(v[3])
			}
			return checkNum(roundRat(new(big.Rat).Quo(x, y), scaleOf(v[2]), mode))
		}},
		"round": {Name: "round", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			mode := "half_even"
			if len(v) == 3 {
				mode = str(v[2])
			}
			return checkNum(roundRat(num(v[0]), scaleOf(v[1]), mode))
		}},
		"eq": {Name: "eq", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return Equal(v[0], v[1])
		}},
		"ne": {Name: "ne", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return !Equal(v[0], v[1])
		}},
		"lt": {Name: "lt", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return Order(v[0], v[1]) < 0
		}},
		"le": {Name: "le", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return Order(v[0], v[1]) <= 0
		}},
		"gt": {Name: "gt", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return Order(v[0], v[1]) > 0
		}},
		"ge": {Name: "ge", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return Order(v[0], v[1]) >= 0
		}},
		"not": {Name: "not", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return !boo(v[0])
		}},
		"concat": {Name: "concat", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			var b strings.Builder
			for _, x := range v {
				b.WriteString(str(x))
			}
			return b.String()
		}},
		"len": {Name: "len", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			switch t := v[0].(type) {
			case string:
				return big.NewRat(int64(utf8.RuneCountInString(t)), 1)
			case []any:
				return big.NewRat(int64(len(t)), 1)
			case map[string]any:
				return big.NewRat(int64(len(t)), 1)
			}
			execFail("TYPE_ERROR", "len of %s", TypeName(v[0]))
			return nil
		}},
		"to_string": {Name: "to_string", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return FormatDecimal(num(v[0]))
		}},
		"to_number": {Name: "to_number", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			s := str(v[0])
			if !numRe.MatchString(s) {
				execFail("BAD_NUMBER", "not a JSON number: %q", s)
			}
			r, _ := new(big.Rat).SetString(s)
			return checkNum(r)
		}},
		"list": {Name: "list", Apply: func(m *Machine, a []any, env map[string]any) any {
			return evalArgs(m, a, env)
		}},
		"keys": {Name: "keys", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			mp, ok := v[0].(map[string]any)
			if !ok {
				execFail("TYPE_ERROR", "keys of %s", TypeName(v[0]))
			}
			out := []any{}
			for _, k := range SortedKeys(mp) {
				out = append(out, k)
			}
			return out
		}},
		"type_of": {Name: "type_of", Apply: func(m *Machine, a []any, env map[string]any) any {
			v := evalArgs(m, a, env)
			return TypeName(v[0])
		}},
		"has": {Name: "has", Apply: func(m *Machine, a []any, env map[string]any) any {
			return hasGetOr("has", evalArgs(m, a, env))
		}},
		"get": {Name: "get", Apply: func(m *Machine, a []any, env map[string]any) any {
			return hasGetOr("get", evalArgs(m, a, env))
		}},
		"get_or": {Name: "get_or", Apply: func(m *Machine, a []any, env map[string]any) any {
			return hasGetOr("get_or", evalArgs(m, a, env))
		}},
	}
	for name, bounds := range arity {
		d := ops[name]
		d.MinArity, d.MaxArity = bounds[0], bounds[1]
		ops[name] = d
	}
	return ops
}
