// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// The `aggregate` instruction: sugar for the fold every counting or summing
// program otherwise writes by hand (see examples/shaxon/authz/rolling-quota).
//
//	{"op": "aggregate",
//	 "in": <array operand>, "as": "e", "into": ["state", "window"],
//	 "where": <boolean operand over local.e>,                      (optional)
//	 "measures": {"n": {"count": true}, "total": {"sum": <number operand>},
//	              "lo": {"min": <number operand>}, "hi": {"max": <number operand>}}}
//
// It is inflated, before the program is checked, into
//
//	set into {"n": 0, "total": 0, "lo": null, "hi": null}
//	for in as e do [ if where then [ one `set` per measure ] ]
//
// using only the core's own instructions and `$compute` islands. Nothing
// downstream knows `aggregate` exists: the static check, the cost table,
// the step limit, the mutation log and index rebuilds all see the expansion,
// so an aggregate costs exactly what the expansion costs and cannot do
// anything a hand-written fold could not.
//
// Decisions (A1..A5)
//
//	A1  The result is an object written at `into` with one member per
//	    measure, replacing whatever was there (it is a `set`).
//	A2  Over no matching element: count and sum are 0, min and max are null.
//	A3  sum, min and max require number operands; anything else is the
//	    TYPE_ERROR the compute operator raises. The result of sum is exact.
//	    There is no average: `div` loses digits, so it stays an explicit
//	    choice in the program, made on the aggregate's members.
//	A4  A malformed aggregate (missing or unknown member, a measure that is
//	    not exactly one of count/sum/min/max) is a PROGRAM_ERROR naming it.
//	    Errors in the operands themselves are reported by the ordinary
//	    checks, against the expansion.
//	A5  `as` is an ordinary `for` binding: the same shadowing rules apply
//	    and it is local to the aggregate.

import (
	"math/big"
	"sort"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// expandAggregates returns prog with every `aggregate` replaced by its
// expansion. It never modifies prog; blocks it rewrites are fresh.
func expandAggregates(prog any) any {
	block, ok := prog.([]any)
	if !ok {
		return prog // the ordinary check reports a program that is not a block
	}
	out := make([]any, 0, len(block))
	for _, raw := range block {
		in, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		switch in["op"] {
		case "aggregate":
			out = append(out, expandAggregate(in)...)
		case "if":
			cp := shallowCopy(in)
			for _, k := range []string{"then", "else"} {
				if b, has := in[k]; has {
					cp[k] = expandAggregates(b)
				}
			}
			out = append(out, cp)
		case "for":
			cp := shallowCopy(in)
			if b, has := in["do"]; has {
				cp["do"] = expandAggregates(b)
			}
			out = append(out, cp)
		default:
			out = append(out, raw)
		}
	}
	return out
}

func shallowCopy(m map[string]any) map[string]any {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func aggFail(format string, a ...any) {
	jaxson.Fail("PROGRAM_ERROR", "", "aggregate: "+format, a...)
}

var aggregateMeasures = map[string]bool{"count": true, "sum": true, "min": true, "max": true}

func expandAggregate(in map[string]any) []any {
	for k := range in {
		switch k {
		case "op", "in", "as", "into", "where", "measures":
		default:
			aggFail("unknown field %q", k)
		}
	}
	for _, k := range []string{"in", "as", "into", "measures"} {
		if _, has := in[k]; !has {
			aggFail("missing field %q", k)
		}
	}
	if _, ok := in["as"].(string); !ok {
		aggFail("as must be a string")
	}
	into, ok := in["into"].([]any)
	if !ok || len(into) == 0 {
		aggFail("into must be a non-empty path array")
	}
	measures, ok := in["measures"].(map[string]any)
	if !ok || len(measures) == 0 {
		aggFail("measures must be a non-empty object")
	}

	// Members in sorted order, so the expansion (and its cost) does not
	// depend on how a decoder happened to order the object.
	names := make([]string, 0, len(measures))
	for n := range measures {
		names = append(names, n)
	}
	sort.Strings(names)

	zero := func() any { return new(big.Rat) }
	init := map[string]any{}
	var updates []any
	for _, name := range names {
		spec, ok := measures[name].(map[string]any)
		if !ok || len(spec) != 1 {
			aggFail("measure %q must be an object with exactly one of count, sum, min, max", name)
		}
		var kind string
		var arg any
		for k, v := range spec {
			if !aggregateMeasures[k] {
				aggFail("measure %q: unknown aggregate %q (count, sum, min, max)", name, k)
			}
			kind, arg = k, v
		}
		at := append(jaxson.Clone(into).([]any), name)
		cur := map[string]any{"$path": jaxson.Clone(at)}
		var value any
		switch kind {
		case "count":
			if arg != true {
				aggFail("measure %q: count takes true", name)
			}
			init[name] = zero()
			value = aggCompute(map[string]any{"c": cur},
				[]any{"add", aggVar("c"), big.NewRat(1, 1)})
		case "sum":
			init[name] = zero()
			value = aggCompute(map[string]any{"s": cur, "x": jaxson.Clone(arg)},
				[]any{"add", aggVar("s"), aggVar("x")})
		default: // min, max
			init[name] = nil
			value = aggCompute(map[string]any{"cur": cur, "x": jaxson.Clone(arg)},
				[]any{"select",
					[]any{"eq", []any{"type_of", aggVar("cur")}, "null"},
					[]any{kind, aggVar("x")},
					[]any{kind, aggVar("cur"), aggVar("x")}})
		}
		updates = append(updates, map[string]any{"op": "set", "path": at, "value": value})
	}

	body := updates
	if w, has := in["where"]; has {
		body = []any{map[string]any{"op": "if", "cond": jaxson.Clone(w), "then": updates}}
	}
	return []any{
		map[string]any{"op": "set", "path": jaxson.Clone(into), "value": map[string]any{"$lit": init}},
		map[string]any{"op": "for", "in": jaxson.Clone(in["in"]), "as": in["as"], "do": body},
	}
}

func aggVar(name string) any { return map[string]any{"$v": name} }

func aggCompute(with map[string]any, expr []any) any {
	return map[string]any{"$compute": map[string]any{"with": with, "expr": expr}}
}
