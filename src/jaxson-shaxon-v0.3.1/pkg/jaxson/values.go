// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

import (
	"encoding/json"
	"math/big"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------- numbers

var ten = big.NewRat(10, 1)

// numRe matches a JSON number literal, used by to_number to validate a
// string before parsing it as a *big.Rat. Moved here from checks.go
// (Phase 1) — it's a number-format primitive, not a static-check rule,
// so it belongs beside FormatDecimal/checkNum rather than with checkPath.
var numRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

const maxDigits = 38

// decParts returns the coefficient and the minimal scale of a decimal Rat.
func decParts(r *big.Rat) (*big.Int, int, bool) {
	n := new(big.Rat).Set(r)
	s := 0
	for !n.IsInt() {
		if s > 200 {
			return nil, 0, false
		}
		n.Mul(n, ten)
		s++
	}
	return new(big.Int).Set(n.Num()), s, true
}

func checkNum(r *big.Rat) *big.Rat {
	c, _, ok := decParts(r)
	if !ok || len(new(big.Int).Abs(c).String()) > maxDigits {
		execFail("NUMERIC_OVERFLOW", "number exceeds %d digits", maxDigits)
	}
	return r
}

func FormatDecimal(r *big.Rat) string {
	c, s, _ := decParts(r)
	neg := c.Sign() < 0
	d := new(big.Int).Abs(c).String()
	if s > 0 {
		for len(d) <= s {
			d = "0" + d
		}
		d = d[:len(d)-s] + "." + d[len(d)-s:]
	}
	if neg {
		d = "-" + d
	}
	return d
}

func roundRat(x *big.Rat, scale int, mode string) *big.Rat {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	m := new(big.Rat).Mul(x, new(big.Rat).SetInt(p))
	neg := m.Sign() < 0
	a := new(big.Rat).Abs(m)
	fl := new(big.Int).Div(a.Num(), a.Denom())
	rem := new(big.Rat).Sub(a, new(big.Rat).SetInt(fl))
	half := big.NewRat(1, 2)
	inc := false
	switch mode {
	case "half_up":
		inc = rem.Cmp(half) >= 0
	case "half_even":
		c := rem.Cmp(half)
		inc = c > 0 || (c == 0 && fl.Bit(0) == 1)
	case "down":
		inc = false
	default:
		execFail("BAD_MODE", "unknown rounding mode %q", mode)
	}
	if inc {
		fl.Add(fl, big.NewInt(1))
	}
	if neg {
		fl.Neg(fl)
	}
	return new(big.Rat).SetFrac(fl, p)
}

// ---------------------------------------------------------------- values

func norm(v any) any {
	switch t := v.(type) {
	case json.Number:
		r, ok := new(big.Rat).SetString(t.String())
		if !ok {
			fail("PARSE_ERROR", "", "bad number %s", t)
		}
		return r
	case []any:
		for i := range t {
			t[i] = norm(t[i])
		}
		return t
	case map[string]any:
		for k, x := range t {
			t[k] = norm(x)
		}
		return t
	}
	return v
}

func Clone(v any) any {
	switch t := v.(type) {
	case *big.Rat:
		return new(big.Rat).Set(t)
	case []any:
		o := make([]any, len(t))
		for i, x := range t {
			o[i] = Clone(x)
		}
		return o
	case map[string]any:
		o := make(map[string]any, len(t))
		for k, x := range t {
			o[k] = Clone(x)
		}
		return o
	}
	return v
}

func Equal(a, b any) bool {
	switch x := a.(type) {
	case nil:
		return b == nil
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case *big.Rat:
		y, ok := b.(*big.Rat)
		return ok && x.Cmp(y) == 0
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !Equal(x[i], y[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, xv := range x {
			yv, ok := y[k]
			if !ok || !Equal(xv, yv) {
				return false
			}
		}
		return true
	}
	return false
}

func TypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case *big.Rat:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
}

func SortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func show(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case bool, string:
		b, _ := json.Marshal(t)
		return string(b)
	case *big.Rat:
		return FormatDecimal(t)
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			parts[i] = show(x)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		var parts []string
		for _, k := range SortedKeys(t) {
			kb, _ := json.Marshal(k)
			parts = append(parts, string(kb)+":"+show(t[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return "?"
}

// Order compares two jaxson values for lt/le/gt/ge, per the same
// ordering the original jaxrun.go used. Moved here from machine.go
// (Phase 1) — the plan's own §2 layout places it in values.go, alongside
// Clone/Equal/TypeName/SortedKeys, since it's a pure value comparison
// with no dependency on a running Machine.
func Order(a, b any) int {
	switch x := a.(type) {
	case *big.Rat:
		if y, ok := b.(*big.Rat); ok {
			return x.Cmp(y)
		}
	case string:
		if y, ok := b.(string); ok {
			return strings.Compare(x, y)
		}
	}
	execFail("TYPE_ERROR", "cannot order %s and %s", TypeName(a), TypeName(b))
	return 0
}

// Normalize, Show, Clone, Equal, TypeName, SortedKeys, Order, and
// FormatDecimal are Phase 1's export surface: Phase 0 added only
// Normalize and Show (the minimum needed for an external caller to get
// a package's raw JSON into the value model Run() expects and back out
// again); everything else — including these six — stayed unexported
// pending this pass. Capitalizing them closes the last item the Phase 1
// assessment flagged: Phase 2's pkg/shaxon will need at least Equal (to
// compare shape-checked values) and FormatDecimal (to render one in a
// violation report), and nothing in Phase 1 depended on deferring this.

// Normalize converts a value decoded with json.Decoder.UseNumber() (so
// that json.Number values appear instead of float64) into the value
// model jaxson.Run expects: json.Number becomes *big.Rat, recursively
// through arrays and objects. Call it once on a decoded package before
// calling Run.
func Normalize(v any) any { return norm(v) }

// Show renders a jaxson value (as returned by Run, or read from a
// package) as canonical JSON text: exact decimals, sorted object keys,
// no whitespace. Unlike encoding/json.Marshal, it understands the
// *big.Rat number representation Run() produces.
func Show(v any) string { return show(v) }
