// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package build_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	jb "github.com/ha1tch/jaxson/pkg/jaxson/build"
)

// sumOfPrices is the Jaxson fixture of the same name, written with the
// builder instead of JSON.
func sumOfPrices() jb.Package {
	var (
		item  = jb.Loop("item")
		t, p  = jb.Var("t"), jb.Var("p")
		total = jb.State("total")
	)
	return jb.NewPackage().
		InputSchema(jb.Object().
			Req("items", jb.Array(jb.Object().Req("price", jb.Number())))).
		OutputSchema(jb.Object().Req("total", jb.Number())).
		Input(map[string]any{"items": []any{
			map[string]any{"price": 10}, map[string]any{"price": 20}, map[string]any{"price": 30}}}).
		Program(
			jb.Set(total, jb.Int(0)),
			jb.For(jb.Input("items")).As(item).Do(
				jb.Set(total, jb.Calc(jb.Add(t, p)).
					With(t, total).
					With(p, item.Key("price"))),
			),
			jb.Set(jb.Output(), jb.Tpl().Set("total", total)),
		)
}

func TestBuilderReproducesFixture(t *testing.T) {
	raw, err := os.ReadFile("../jaxson-v0.1.0-fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var cases []any
	if err := d.Decode(&cases); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	for _, c := range jaxson.Normalize(cases).([]any) {
		if m := c.(map[string]any); m["name"] == "sum-of-prices" {
			want = m
		}
	}
	if want == nil {
		t.Fatal("fixture sum-of-prices not found")
	}
	delete(want, "name")
	delete(want, "expect")
	got, err := sumOfPrices().Map()
	if err != nil {
		t.Fatal(err)
	}
	if !jaxson.Equal(want, got) {
		t.Fatalf("builder output differs from fixture\n got: %s\nwant: %s", jaxson.Show(got), jaxson.Show(want))
	}
}

func TestRunProducesExpectedOutput(t *testing.T) {
	out, err := sumOfPrices().Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("output rendered as: %s", jaxson.Show(out))
	if !strings.Contains(jaxson.Show(out), "60") {
		t.Fatalf("unexpected output %s", jaxson.Show(out))
	}
}

func TestJSONIsDeterministicAndRunnable(t *testing.T) {
	a, err := sumOfPrices().JSON()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := sumOfPrices().JSON()
	if !bytes.Equal(a, b) {
		t.Fatal("JSON output is not deterministic")
	}
	var back map[string]any
	if json.Unmarshal(a, &back) != nil || back["jaxson"] != "1.0" {
		t.Fatalf("JSON is not a Jaxson package: %s", a)
	}
}

// Every operator constructor, at its minimum arity, must produce an
// expression the runtime accepts: this catches a misspelt operator name.
func TestEveryOperatorIsAcceptedByTheRuntime(t *testing.T) {
	i, s, b := jb.Int(2), jb.Str("a"), jb.Bool(true)
	exprs := map[string]jb.Expr{
		"add": jb.Add(i, i), "sub": jb.Sub(i, i), "mul": jb.Mul(i, i), "neg": jb.Neg(i), "abs": jb.Abs(i),
		"mod": jb.Mod(i, i), "min": jb.Min(i), "max": jb.Max(i),
		"div": jb.Div(i, i, 2), "divmode": jb.Div(i, i, 0, jb.Down), "round": jb.Round(i, 0, jb.HalfUp),
		"eq": jb.Eq(i, i), "ne": jb.Ne(i, i), "lt": jb.Lt(i, i), "le": jb.Le(i, i), "gt": jb.Gt(i, i), "ge": jb.Ge(i, i),
		"and": jb.And(b), "or": jb.Or(b), "not": jb.Not(b), "select": jb.Select(b, i, i),
		"concat": jb.Concat(s), "len": jb.Len(s), "to_string": jb.ToString(i), "to_number": jb.ToNumber(s),
		"list": jb.List(), "get": jb.Get(jb.List(i), jb.Int(0)), "has": jb.Has(jb.List(i), jb.Int(0)),
		"get_or": jb.GetOr(jb.List(i), jb.Int(0), i), "keys": jb.Keys(jb.List(i)), "type_of": jb.TypeOf(i),
	}
	for name, e := range exprs {
		p := jb.NewPackage().InputSchema(jb.Anything()).OutputSchema(jb.Anything()).
			Program(jb.Set(jb.State("x"), jb.Calc(e)))
		if err := p.Check(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func base() jb.Package {
	return jb.NewPackage().InputSchema(jb.Anything()).OutputSchema(jb.Anything())
}

func TestAssemblyRejectsWhatTheRuntimeRejects(t *testing.T) {
	x := jb.Var("x")
	loop := jb.Loop("k")
	cases := map[string]jb.Package{
		"unbound variable":  base().Program(jb.Set(jb.State("a"), jb.Calc(jb.Add(x, jb.Int(1))))),
		"write to input":    base().Program(jb.Set(jb.Input("a"), jb.Int(1))),
		"shadowed loop":     base().Program(jb.For(jb.Data([]int{1})).As(loop).Do(jb.For(jb.Data([]int{1})).As(loop).Do(jb.Halt()))),
		"form inside data":  base().Program(jb.Set(jb.State("a"), jb.Data(map[string]any{"$path": []any{"input"}, "z": 1}))),
		"missing in schema": jb.NewPackage().OutputSchema(jb.Anything()),
		"bad sample input":  jb.NewPackage().InputSchema(jb.Integer()).OutputSchema(jb.Anything()).Input("text"),
		"bad schema":        jb.NewPackage().InputSchema(jb.Object().Req("a", jb.Anything()).Open()).OutputSchema(jb.AnyOf()),
	}
	for name, p := range cases {
		if err := p.Check(); err == nil {
			t.Errorf("%s: accepted, want an error", name)
		}
	}
}

func TestBuildersAreImmutable(t *testing.T) {
	base := jb.Object().Req("a", jb.Integer())
	withB := base.Req("b", jb.String())
	_ = withB
	p1 := jb.NewPackage().InputSchema(base).OutputSchema(jb.Anything()).Input(map[string]any{"a": 1})
	if err := p1.Check(); err != nil {
		t.Fatalf("extending a shared schema leaked into the original: %v", err)
	}
	c := jb.Calc(jb.Add(jb.Var("a"), jb.Int(1)))
	c1 := c.With("a", jb.Int(1))
	_ = c1
	if err := base_(c).Check(); err == nil {
		t.Fatal("With mutated the receiver: original Calc now has a binding")
	}
}

func base_(c jb.Compute) jb.Package {
	return base().Program(jb.Set(jb.State("r"), c))
}

func TestIfElseAndInsertDeleteAndOptRoundTrip(t *testing.T) {
	p := base().Input(nil).Program(
		jb.Set(jb.State("xs"), jb.Data([]int{1, 2, 3})),
		jb.Insert(jb.State("xs"), jb.Int(0), jb.Int(9)),
		jb.Delete(jb.State("xs").Idx(1)),
		jb.If(jb.Bool(true)).Then(jb.Set(jb.State("f"), jb.Int(1))).Else(jb.Set(jb.State("f"), jb.Int(2))),
		jb.Assert(jb.Calc(jb.Eq(jb.Var("n"), jb.Int(3))).With("n", jb.Calc(jb.Len(jb.Var("l"))).With("l", jb.State("xs")))).Msg("len"),
		jb.Set(jb.Output(), jb.Tpl().Set("xs", jb.State("xs")).Opt("gone", jb.State("nope")).Set("n", jb.Tpl().Set("f", jb.State("f")))),
	)
	out, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rendered: %s", jaxson.Show(out))
}
