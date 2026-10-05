// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// ORACLE TEST: SCHEDULED FOR DELETION together with tree_oracle.go (see the
// header of that file for the list).
//
// The closure compiler (compile.go) must be indistinguishable from the old
// tree-walking interpreter (tree_oracle.go): same output, same step total
// and same failure (category, code and message) for every package. This file
// checks that three ways:
//
//  1. every core fixture (jaxson-v0.1.0-fixtures.json),
//  2. every showcase fixture, when the file is present,
//  3. a deterministic stream of random programs covering every operator and
//     every core instruction, with step limits low enough that the limit is
//     hit part of the time (so the order of charging is exercised, not only
//     the totals).
//
// Both interpreters are run on fresh copies of the same package; the step
// total is read from the machine through a profile hook, which sees it
// whether the run succeeds or fails.

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"
)

// outcome is everything observable about one run.
type outcome struct {
	out   string
	steps int64
	err   string
}

func (o outcome) String() string {
	if o.err != "" {
		return fmt.Sprintf("error %s after %d steps", o.err, o.steps)
	}
	return fmt.Sprintf("output %s after %d steps", o.out, o.steps)
}

// stepProbe captures the machine of one run so its step total can be read.
type stepProbe struct {
	NoHooks
	m *Machine
}

func (p *stepProbe) Start(m *Machine) { p.m = m }

// runBoth runs the package once per interpreter, each on its own deep copy,
// and returns the compiled and the tree outcome.
func runBoth(p map[string]any) (compiled, tree outcome) {
	one := func(useTree bool) outcome {
		UseTreeInterpreter(useTree)
		defer UseTreeInterpreter(false)
		probe := &stepProbe{}
		pr := CoreProfile()
		pr.Begin = func(map[string]any) Hooks { return probe }
		out, e := pr.Run(Clone(p).(map[string]any))
		o := outcome{}
		if probe.m != nil {
			o.steps = probe.m.Steps()
		}
		if e != nil {
			o.err = e.Error()
		} else {
			o.out = show(out)
		}
		return o
	}
	return one(false), one(true)
}

func TestOracleCoreFixtures(t *testing.T) {
	files := []string{"jaxson-v0.1.0-fixtures.json"}
	if path := os.Getenv("JAXSON_SHOWCASE"); path != "" {
		files = append(files, path)
	} else if _, err := os.Stat("../../../../examples/showcase/showcase-fixtures.json"); err == nil {
		files = append(files, "../../../../examples/showcase/showcase-fixtures.json")
	}
	total := 0
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		v, perr := ParseJSON(raw)
		if perr != nil {
			t.Fatalf("%s: %v", f, perr)
		}
		for _, c := range v.([]any) {
			fx := c.(map[string]any)
			p := map[string]any{}
			for k, val := range fx {
				if k != "name" && k != "note" && k != "expect" {
					p[k] = val
				}
			}
			name := fmt.Sprint(fx["name"])
			a, b := runBoth(p)
			if a != b {
				t.Errorf("%s: compiled: %s\n   tree: %s", name, a, b)
			}
			total++
		}
	}
	if total < 28 {
		t.Fatalf("only %d fixtures compared", total)
	}
	t.Logf("%d fixtures: compiled and tree runs agree", total)
}

// ---------------------------------------------------------------- random programs

// pgen builds random but mostly well-formed packages. It aims at valid
// programs (so that execution, not the static check, is what differs if
// anything does) but deliberately allows type errors, missing paths, bad
// indices and division by zero at run time.
type pgen struct {
	r *rand.Rand
	// numeric locals and string locals currently in scope
	nums []string
	strs []string
	id   int
}

func (g *pgen) pick(n int) int { return g.r.Intn(n) }
func (g *pgen) chance(p int) bool {
	return g.r.Intn(100) < p
}

func rat(n, d int64) *big.Rat { return big.NewRat(n, d) }

func (g *pgen) numLit() any {
	switch g.pick(6) {
	case 0:
		return rat(int64(g.pick(5)), 1)
	case 1:
		return rat(int64(g.pick(200)-100), 1)
	case 2:
		return rat(int64(g.pick(2000)-1000), 100)
	case 3:
		return rat(int64(g.pick(30)+1), 8)
	case 4:
		return rat(3, 4)
	}
	return rat(int64(g.pick(10)), 1)
}

func (g *pgen) strLit() any {
	return []string{"", "a", "bc", "12", "-3.5", "x y", "héllo", "1e2"}[g.pick(8)]
}

// ---- compute expressions

func (g *pgen) numExpr(d int, names []string) any {
	if d <= 0 || g.chance(25) {
		if len(names) > 0 && g.chance(60) {
			return map[string]any{"$v": names[g.pick(len(names))]}
		}
		return g.numLit()
	}
	n := func() any { return g.numExpr(d-1, names) }
	b := func() any { return g.boolExpr(d-1, names) }
	switch g.pick(16) {
	case 0, 1:
		return []any{"add", n(), n()}
	case 2:
		return []any{"add", n(), n(), n()}
	case 3:
		return []any{"sub", n(), n()}
	case 4:
		return []any{"mul", n(), n()}
	case 5:
		return []any{"neg", n()}
	case 6:
		return []any{"abs", n()}
	case 7:
		return []any{"min", n(), n()}
	case 8:
		return []any{"max", n(), n(), n()}
	case 9:
		if g.chance(85) {
			return []any{"mod", []any{"round", n(), rat(0, 1)}, rat(int64(1+g.pick(7)), 1)}
		}
		return []any{"mod", n(), n()}
	case 10:
		e := []any{"div", n(), n(), rat(int64(g.pick(4)), 1)}
		if g.chance(40) {
			e = append(e, []string{"half_even", "half_up", "down", "half_even", "half_up", "down", "bogus"}[g.pick(7)])
		}
		return e
	case 11:
		e := []any{"round", n(), rat(int64(g.pick(4)), 1)}
		if g.chance(40) {
			e = append(e, []string{"half_even", "half_up", "down"}[g.pick(3)])
		}
		return e
	case 12:
		return []any{"select", b(), n(), n()}
	case 13:
		return []any{"len", g.strExpr(d-1, names)}
	case 14:
		if g.chance(80) {
			return []any{"to_number", []any{"to_string", n()}}
		}
		return []any{"to_number", g.strExpr(d-1, names)}
	}
	return []any{"len", []any{"list", n(), n()}}
}

func (g *pgen) boolExpr(d int, names []string) any {
	if d <= 0 || g.chance(15) {
		return g.chance(50)
	}
	n := func() any { return g.numExpr(d-1, names) }
	b := func() any { return g.boolExpr(d-1, names) }
	switch g.pick(12) {
	case 0:
		return []any{"eq", n(), n()}
	case 1:
		return []any{"ne", n(), n()}
	case 2:
		return []any{"lt", n(), n()}
	case 3:
		return []any{"le", n(), n()}
	case 4:
		return []any{"gt", n(), n()}
	case 5:
		return []any{"ge", n(), n()}
	case 6:
		return []any{"and", b(), b(), b()}
	case 7:
		return []any{"or", b(), b()}
	case 8:
		return []any{"not", b()}
	case 9:
		return []any{"select", b(), b(), b()}
	case 10:
		return []any{"has", []any{"list", n(), n()}, n()}
	}
	return []any{"eq", g.strExpr(d-1, names), g.strExpr(d-1, names)}
}

func (g *pgen) strExpr(d int, names []string) any {
	if d <= 0 || g.chance(30) {
		return g.strLit()
	}
	switch g.pick(5) {
	case 0:
		return []any{"concat", g.strExpr(d-1, names), g.strExpr(d-1, names)}
	case 1:
		return []any{"to_string", g.numExpr(d-1, names)}
	case 2:
		return []any{"type_of", g.anyExpr(d-1, names)}
	case 3:
		return []any{"select", g.boolExpr(d-1, names), g.strExpr(d-1, names), g.strExpr(d-1, names)}
	}
	return []any{"concat", g.strLit(), []any{"to_string", g.numExpr(d-1, names)}}
}

func (g *pgen) anyExpr(d int, names []string) any {
	switch g.pick(8) {
	case 0:
		return g.strExpr(d, names)
	case 1:
		return g.boolExpr(d, names)
	case 2:
		return []any{"list", g.numExpr(d-1, names), g.strExpr(d-1, names), g.boolExpr(d-1, names)}
	case 3:
		return []any{"len", []any{"list", g.anyExpr(d-1, names)}}
	case 4:
		return []any{"get_or", []any{"list", g.numExpr(d-1, names)}, rat(int64(g.pick(3)), 1), g.numExpr(d-1, names)}
	case 5:
		return []any{"get", []any{"list", g.numExpr(d-1, names), g.numExpr(d-1, names)}, rat(int64(g.pick(3)), 1)}
	case 6:
		return nil
	}
	return g.numExpr(d, names)
}

// compute builds a $compute island. It binds a few names to operands and
// writes an expression over them.
func (g *pgen) compute(d int, kind string) map[string]any {
	with := map[string]any{}
	var names []string
	for _, c := range [][2]string{{"p", "num"}, {"q", "num"}, {"r", "num"}} {
		if g.chance(55) {
			with[c[0]] = g.operand(d-1, "num")
			names = append(names, c[0])
		}
	}
	var expr any
	if kind == "any" {
		kind = []string{"bool", "str", "num", "num"}[g.pick(4)]
	}
	switch kind {
	case "bool":
		expr = g.boolExpr(3, names)
	case "str":
		expr = g.strExpr(3, names)
	default:
		expr = g.numExpr(3, names)
	}
	if len(with) == 0 && g.chance(50) {
		return map[string]any{"expr": expr}
	}
	return map[string]any{"with": with, "expr": expr}
}

// ---- operands

// path returns a $path operand reading something that mostly exists.
func (g *pgen) readPath(kind string) any {
	if kind == "num" || kind == "any" {
		if len(g.nums) > 0 && g.chance(40) {
			return map[string]any{"$path": []any{"local", g.nums[g.pick(len(g.nums))]}}
		}
		switch g.pick(7) {
		case 0:
			return map[string]any{"$path": []any{"input", "a"}}
		case 1:
			return map[string]any{"$path": []any{"input", "l", rat(int64(g.pick(4)), 1)}}
		case 2:
			return map[string]any{"$path": []any{"state", "n0"}}
		case 3:
			return map[string]any{"$path": []any{"state", "n1"}}
		case 4:
			return map[string]any{"$path": []any{"state", "o", "k"}}
		case 5:
			// computed segment
			return map[string]any{"$path": []any{"input", "l", map[string]any{"$compute": map[string]any{"expr": []any{"mod", []any{"abs", []any{"round", g.numExpr(2, nil), rat(0, 1)}}, rat(4, 1)}}}}}
		}
		return map[string]any{"$path": []any{"state", "l", rat(int64(g.pick(3)), 1)}}
	}
	return map[string]any{"$path": []any{"input", "s"}}
}

func (g *pgen) operand(d int, kind string) any {
	if d <= 0 || g.chance(20) {
		switch g.pick(4) {
		case 0:
			if kind == "str" {
				return g.strLit()
			}
			return g.numLit()
		case 1:
			if kind == "any" {
				return map[string]any{"$lit": []any{g.numLit(), g.strLit(), map[string]any{"k": g.numLit()}}}
			}
			return map[string]any{"$lit": g.numLit()}
		default:
			return g.readPath(kind)
		}
	}
	switch g.pick(6) {
	case 0, 1:
		return map[string]any{"$compute": g.compute(d, kind)}
	case 2:
		return g.readPath(kind)
	case 3, 4:
		if kind == "num" || kind == "str" {
			return map[string]any{"$compute": g.compute(d, kind)}
		}
	}
	if kind != "any" {
		if kind == "str" {
			return g.strLit()
		}
		return g.numLit()
	}
	switch g.pick(2) {
	case 0:
		return map[string]any{"$tpl": map[string]any{
			"a": g.operand(d-1, "num"),
			"b": []any{g.operand(d-1, "num"), map[string]any{"$opt": []any{"input", "missing"}}, g.strLit()},
			"c": map[string]any{"$path": []any{"input", "s"}},
		}}
	case 1:
		return map[string]any{"$tpl": []any{map[string]any{"$compute": g.compute(d, "any")}, g.numLit()}}
	}
	if kind == "str" {
		return g.strLit()
	}
	return g.numLit()
}

// computedWritePath is a write path whose last segment is computed, so that
// the order in which a set or delete evaluates its value and its path (and
// what each charges) is observable.
func (g *pgen) computedWritePath() []any {
	if g.chance(50) {
		return []any{"state", "l", map[string]any{"$compute": map[string]any{"expr": []any{"mod", []any{"abs", []any{"round", g.numExpr(2, nil), rat(0, 1)}}, rat(3, 1)}}}}
	}
	return []any{"state", "o", map[string]any{"$compute": map[string]any{"expr": []any{"select", g.boolExpr(2, nil), "k", g.strLit()}}}}
}

func (g *pgen) at() any {
	if g.chance(85) {
		return rat(int64(g.pick(3)), 1)
	}
	return g.operand(1, "num")
}

func (g *pgen) boolOperand(d int) any {
	if g.chance(25) {
		return map[string]any{"$path": []any{"input", "flag"}}
	}
	return map[string]any{"$compute": g.compute(d, "bool")}
}

// ---- instructions

func (g *pgen) block(d, n int) []any {
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, g.instruction(d))
	}
	return out
}

func (g *pgen) instruction(d int) any {
	writeRoot := "state"
	if g.chance(15) {
		writeRoot = "output"
	}
	switch k := g.pick(14); {
	case k <= 3:
		path := []any{"state", fmt.Sprintf("n%d", g.pick(3))}
		if g.chance(15) {
			path = []any{writeRoot, "o", "k"}
		} else if g.chance(15) {
			path = g.computedWritePath()
		}
		return map[string]any{"op": "set", "path": path, "value": g.operand(2, "num")}
	case k == 4:
		return map[string]any{"op": "append", "path": []any{"state", "l"}, "value": g.operand(2, "num")}
	case k == 5:
		return map[string]any{"op": "insert", "path": []any{"state", "l"}, "at": g.at(), "value": g.operand(2, "num")}
	case k == 6:
		if g.chance(30) {
			return map[string]any{"op": "delete", "path": g.computedWritePath()}
		}
		if g.chance(50) {
			return map[string]any{"op": "delete", "path": []any{"state", "l", rat(int64(g.pick(3)), 1)}}
		}
		return map[string]any{"op": "delete", "path": []any{"state", "o", "k"}}
	case k == 7 && d > 0:
		in := map[string]any{"op": "if", "cond": g.boolOperand(2), "then": g.block(d-1, 1+g.pick(2))}
		if g.chance(50) {
			in["else"] = g.block(d-1, 1+g.pick(2))
		}
		return in
	case (k == 8 || k == 9) && d > 0:
		g.id++
		el, ix := fmt.Sprintf("e%d", g.id), fmt.Sprintf("i%d", g.id)
		in := map[string]any{"op": "for", "as": el}
		if g.chance(40) {
			in["in"] = map[string]any{"$path": []any{"state", "l"}} // a loop over something its body appends to
		} else {
			in["in"] = map[string]any{"$path": []any{"input", "l"}}
		}
		saved := g.nums
		g.nums = append(append([]string{}, g.nums...), el)
		if g.chance(50) {
			in["index"] = ix
			g.nums = append(g.nums, ix)
		}
		in["do"] = g.block(d-1, 1+g.pick(3))
		g.nums = saved
		return in
	case k == 10:
		in := map[string]any{"op": "assert", "that": g.boolOperand(2)}
		if g.chance(50) {
			in["msg"] = "m"
		}
		return in
	case k == 11 && g.chance(30):
		return map[string]any{"op": "halt"}
	case k == 12:
		return map[string]any{"op": "set", "path": []any{writeRoot, "t"}, "value": g.operand(3, "any")}
	}
	return map[string]any{"op": "set", "path": []any{"state", "n2"}, "value": g.operand(3, "num")}
}

func (g *pgen) program() map[string]any {
	g.nums = nil
	prog := []any{
		map[string]any{"op": "set", "path": []any{"output"}, "value": map[string]any{"$lit": map[string]any{}}},
		map[string]any{"op": "set", "path": []any{"state", "n0"}, "value": rat(0, 1)},
		map[string]any{"op": "set", "path": []any{"state", "n1"}, "value": rat(1, 1)},
		map[string]any{"op": "set", "path": []any{"state", "n2"}, "value": rat(5, 2)},
		map[string]any{"op": "set", "path": []any{"state", "l"}, "value": map[string]any{"$lit": []any{rat(3, 1), rat(1, 1)}}},
		map[string]any{"op": "set", "path": []any{"state", "o"}, "value": map[string]any{"$lit": map[string]any{"k": rat(7, 1)}}},
	}
	prog = append(prog, g.block(2, 3+g.pick(6))...)
	prog = append(prog, map[string]any{"op": "set", "path": []any{"output", "state"}, "value": map[string]any{"$path": []any{"state"}}})
	limits := map[string]any{}
	switch g.pick(10) {
	case 0, 1:
		limits["steps"] = rat(int64(10+g.pick(60)), 1)
	case 2:
		limits["steps"] = rat(int64(100+g.pick(400)), 1)
	}
	p := map[string]any{
		"jaxson":       "1.0",
		"input":        map[string]any{"a": rat(4, 1), "s": "text", "flag": g.chance(50), "l": []any{rat(2, 1), rat(5, 1), rat(-1, 1), rat(1, 2)}},
		"inputSchema":  map[string]any{"type": "any"},
		"outputSchema": map[string]any{"type": "any"},
		"program":      prog,
	}
	if len(limits) > 0 {
		p["limits"] = limits
	}
	return p
}

func TestOracleRandomPrograms(t *testing.T) {
	const programs = 12000
	g := &pgen{r: rand.New(rand.NewSource(20260705))}
	var ok, failed, outOfSteps, static int
	codes := map[string]int{}
	for i := 0; i < programs; i++ {
		p := g.program()
		a, b := runBoth(p)
		if a != b {
			t.Fatalf("program %d differs\n compiled: %s\n     tree: %s\n  package: %s", i, a, b, show(p))
		}
		switch {
		case a.err == "":
			ok++
		default:
			failed++
			e := a.err
			if len(e) > 40 {
				e = e[:40]
			}
			codes[e[:indexOr(e, ':', len(e))]]++
			if len(a.err) >= 14 && a.err[:14] == "RESOURCE_ERROR" {
				outOfSteps++
			}
			if len(a.err) >= 13 && a.err[:13] == "PROGRAM_ERROR" {
				static++
			}
		}
	}
	t.Logf("%d programs: %d succeeded, %d failed (%d out of steps, %d rejected statically)", programs, ok, failed, outOfSteps, static)
	t.Logf("failures by category/code: %v", codes)
	// The generator must exercise the executing paths, not just the static check.
	if ok < programs/4 {
		t.Errorf("only %d of %d programs ran to the end; the generator is not exercising execution", ok, programs)
	}
	if outOfSteps < 100 {
		t.Errorf("only %d programs ran out of steps; the limit-order paths are barely exercised", outOfSteps)
	}
	if static > programs/4 {
		t.Errorf("%d programs were rejected statically; the generator is wasting runs", static)
	}
}

func indexOr(s string, c byte, d int) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return d
}
