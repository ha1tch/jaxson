// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Phase 1: operand static-checking ($path/$lit/$compute/$tpl/$opt and the
// compute-expression arity table), split out of checks.go per that
// file's own "Phase 1 splits this into..." note. checkOperand/checkForm/
// checkTpl/checkCompute now thread a *Checker instead of a bare
// map[string]bool — mechanical, same reasoning as paths.go. checkExpr
// never took locals in the first place (a compute expr's only names are
// $compute's own "with" bindings), so it's unchanged beyond moving file.

import (
	"math/big"
	"regexp"
	"strings"
)

var nameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var arity = map[string][2]int{
	"add": {2, -1}, "sub": {2, -1}, "mul": {2, -1}, "neg": {1, 1}, "abs": {1, 1},
	"min": {1, -1}, "max": {1, -1}, "mod": {2, 2}, "div": {3, 4}, "round": {2, 3},
	"eq": {2, 2}, "ne": {2, 2}, "lt": {2, 2}, "le": {2, 2}, "gt": {2, 2}, "ge": {2, 2},
	"and": {1, -1}, "or": {1, -1}, "not": {1, 1}, "select": {3, 3},
	"concat": {1, -1}, "len": {1, 1}, "to_string": {1, 1}, "to_number": {1, 1},
	"list": {0, -1}, "get": {2, 2}, "has": {2, 2}, "get_or": {3, 3}, "keys": {1, 1}, "type_of": {1, 1},
}

func checkOperand(x any, c *Checker) {
	if mp, ok := x.(map[string]any); ok {
		for k := range mp {
			if strings.HasPrefix(k, "$") {
				checkForm(mp, c)
				return
			}
		}
	}
	if containsForm(x) {
		progFail("a literal contains a $-form; use $tpl or $lit")
	}
}

func checkForm(mp map[string]any, c *Checker) {
	if d, ok := formOf(c.Forms, mp); ok {
		d.Check(mp, c)
		return
	}
	if len(mp) != 1 {
		progFail("a $-form must have exactly one key")
	}
	for k, v := range mp {
		switch k {
		case "$path":
			checkPath(v, c, false)
		case "$lit":
		case "$compute":
			checkCompute(v, c)
		case "$tpl":
			checkTpl(v, c, true)
		case "$opt":
			progFail("$opt is only valid inside $tpl")
		case "$v":
			progFail("$v is only valid inside a compute expr")
		default:
			progFail("unknown form %q", k)
		}
	}
}

func checkTpl(v any, c *Checker, top bool) {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 1 {
			for k, a := range t {
				if k == "$opt" {
					if top {
						progFail("a $tpl root cannot be $opt")
					}
					checkPath(a, c, false)
					return
				}
				if strings.HasPrefix(k, "$") {
					checkForm(t, c)
					return
				}
			}
		}
		for k, a := range t {
			if strings.HasPrefix(k, "$") {
				progFail("a $-key must be the only key of its object")
			}
			checkTpl(a, c, false)
		}
	case []any:
		for _, a := range t {
			checkTpl(a, c, false)
		}
	}
}

func checkCompute(v any, c *Checker) {
	cm, ok := v.(map[string]any)
	if !ok {
		progFail("$compute must be an object")
	}
	for k := range cm {
		if k != "with" && k != "expr" {
			progFail("$compute: unknown key %q", k)
		}
	}
	if _, ok := cm["expr"]; !ok {
		progFail("$compute needs an expr")
	}
	names := map[string]bool{}
	if w, has := cm["with"]; has {
		wm, ok := w.(map[string]any)
		if !ok {
			progFail("$compute: with must be an object")
		}
		for n, o := range wm {
			if !nameRe.MatchString(n) {
				progFail("$compute: bad binding name %q", n)
			}
			names[n] = true
			checkOperand(o, c)
		}
	}
	checkExpr(cm["expr"], names)
}

func checkExpr(e any, names map[string]bool) {
	switch t := e.(type) {
	case nil, bool, string, *big.Rat:
		return
	case map[string]any:
		n, ok := t["$v"].(string)
		if len(t) != 1 || !ok || !names[n] {
			progFail("bad $v reference in expr")
		}
	case []any:
		if len(t) == 0 {
			progFail("an empty application in expr")
		}
		op, ok := t[0].(string)
		ar, known := arity[op]
		if !ok || !known {
			progFail("unknown compute operation %v", t[0])
		}
		n := len(t) - 1
		if n < ar[0] || (ar[1] >= 0 && n > ar[1]) {
			progFail("wrong arity for %s", op)
		}
		for _, a := range t[1:] {
			checkExpr(a, names)
		}
	}
}
