// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.3 (plan section 7): bounded path closures, `$path*` and `$path+`
// (core section 6).
//
// ====================================================================
// What a closure is here, and decisions C1..C6
// ====================================================================
//
// `innerStep` is one relative path, so each node has at most one successor:
// its own path followed by the step's segments. A closure is therefore a
// chain, base then base+step then base+step+step, and the breadth-first,
// duplicate-free ordering of core section 6 is satisfied by the chain order
// itself (a path only grows, so none can repeat). What varies from node to
// node is only which segment a step names, when it uses `local.step`.
//
//	C1  `local.step` is the VALUE at the current node, bound for the
//	    evaluation of the step's operand segments, parallel to `local.item`
//	    in an index build. {"$path": ["local","step","next"]} as a segment
//	    reads the current node's `next` member and uses it as the segment.
//	    It must give a string or a non-negative integer, else TYPE_ERROR
//	    (core section 9). An application of the step is one hop whatever
//	    its segment count.
//	C2  A hop that does not resolve ends the chain; it is not an error
//	    (section 6). That includes a hop that would descend into a scalar:
//	    the Machine's TYPE_ERROR for it is read as "does not resolve". So
//	    is a step whose own segment cannot be computed because the node
//	    lacks the member it reads (MISSING_PATH while evaluating the
//	    operand): a leaf with no `next` ends the chain. A segment that
//	    evaluates to the wrong kind of value is still TYPE_ERROR (C1).
//	C3  `maxDepth` bounds hops. When maxDepth hops have been taken and one
//	    more would still resolve, that is PATH_DEPTH_EXCEEDED: a structural
//	    violation at the last node taken, with the fixed message of section
//	    10, and in gate mode the raised error. The nodes up to the horizon
//	    are still returned as focus nodes.
//	C4  Cost. Each attempted hop, including the one that finds the chain
//	    ended or past the horizon, is one `closure.hop` event, charged before
//	    the hop is tried.
//	C5  The base of `$path*` is always part of the result; of `$path+`, never.
//	    The base must resolve in both (MISSING_PATH if not). `$path+` with no
//	    hop resolving is EXECUTION_ERROR/MISSING_PATH: "a failure only if
//	    zero hops resolve at all".
//	C6  A missing or non-positive-integer `maxDepth` is a PROGRAM_ERROR when
//	    the target is checked, as section 6 requires.

import (
	"math/big"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// EventClosureHop is the cost-table event for one attempted closure hop.
const EventClosureHop = "closure.hop"

// parseClosure statically checks a `$path*` / `$path+` target (the base is
// required: `from`). form is "$path*" or "$path+".
func parseClosure(ctx, form string, mp map[string]any, locals []string) Target {
	for k := range mp {
		if k != form && k != "from" && k != "maxDepth" {
			failLoad("%s: %s takes step, from and maxDepth; unknown key %q", ctx, form, k)
		}
	}
	from, has := mp["from"]
	if !has {
		failLoad("%s: a closure target needs a from path (there is no ambient focus yet)", ctx)
	}
	t := Target{
		Kind:     TargetClosure,
		Plus:     form == "$path+",
		Path:     targetPath(ctx+".from", from, locals),
		Step:     checkStep(ctx+"."+form, mp[form], locals),
		MaxDepth: checkMaxDepth(ctx, mp),
	}
	return t
}

// checkMaxDepth reads the mandatory maxDepth (C6).
func checkMaxDepth(ctx string, mp map[string]any) int {
	v, has := mp["maxDepth"]
	if !has {
		jaxson.Fail("PROGRAM_ERROR", "", "%s: maxDepth is mandatory on a path closure", ctx)
	}
	n, ok := v.(*big.Rat)
	if !ok || !n.IsInt() || n.Sign() <= 0 || !n.Num().IsInt64() || n.Num().Int64() > 1<<31 {
		jaxson.Fail("PROGRAM_ERROR", "", "%s: maxDepth must be a positive integer", ctx)
	}
	return int(n.Num().Int64())
}

// checkStep statically checks an innerStep: a non-empty array of segments,
// each a field name, a non-negative integer, or an operand that may read
// `local.step`.
func checkStep(ctx string, v any, locals []string) []any {
	step, ok := v.([]any)
	if !ok || len(step) == 0 {
		failLoad("%s: a step must be a non-empty array of path segments", ctx)
	}
	scope := append(append([]string(nil), locals...), "step")
	for i, seg := range step {
		switch s := seg.(type) {
		case string:
		case *big.Rat:
			if !s.IsInt() || s.Sign() < 0 {
				failLoad("%s[%d]: an integer segment must be non-negative", ctx, i)
			}
		case map[string]any:
			if e := jaxson.CheckOperand(s, scope...); e != nil {
				failLoad("%s[%d]: %s", ctx, i, e.Msg)
			}
		default:
			failLoad("%s[%d]: a segment must be a field name, a non-negative integer or an operand", ctx, i)
		}
	}
	return step
}

// stepSegs evaluates one application of the step at the node whose value is
// cur (C1). ok is false when a segment's operand cannot be evaluated
// because a path it reads is missing (C2).
func stepSegs(m *jaxson.Machine, step []any, cur any) (out []any, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			if e, is := r.(*jaxson.Err); is && e.Code == "MISSING_PATH" {
				out, ok = nil, false
				return
			}
			panic(r)
		}
	}()
	out = make([]any, 0, len(step))
	m.WithLocal("step", cur, func() {
		for _, seg := range step {
			v := seg
			if mp, ok := seg.(map[string]any); ok {
				v = m.Eval(mp)
			}
			switch u := v.(type) {
			case string:
				out = append(out, u)
			case *big.Rat:
				if !u.IsInt() || u.Sign() < 0 || !u.Num().IsInt64() {
					jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "local.step must resolve to a string or a non-negative integer, got %s", jaxson.Show(u))
				}
				out = append(out, int(u.Num().Int64()))
			default:
				jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "local.step must resolve to a string or a non-negative integer, got %s", jaxson.TypeName(v))
			}
		}
	})
	return out, true
}

// closure walks the chain from base (a path, root first) and returns its
// focus nodes in order. overflow is the path of the last node taken when
// the horizon was hit with the chain still resolving (C3), else nil.
func closure(m *jaxson.Machine, base []any, step []any, maxDepth int, plus bool) (out []Focus, overflow []any) {
	root := base[0].(string)
	cur := base
	curVal := walkOrRaise(m, root, base[1:])
	if !plus {
		out = append(out, Focus{Path: cur, Value: curVal})
	}
	for hop := 1; ; hop++ {
		m.ChargeEvent(EventClosureHop, 1)
		segs, ok := stepSegs(m, step, curVal)
		if !ok {
			break
		}
		next := append(append([]any(nil), cur...), segs...)
		v, err := m.Walk(root, next[1:])
		if err != nil { // the chain ends here (C2)
			break
		}
		if hop > maxDepth {
			return out, cur
		}
		cur, curVal = next, v
		out = append(out, Focus{Path: cur, Value: curVal})
	}
	if plus && len(out) == 0 {
		jaxson.Fail("EXECUTION_ERROR", "MISSING_PATH", "%s: no hop of the $path+ closure resolves", pathString(base))
	}
	return out, nil
}
