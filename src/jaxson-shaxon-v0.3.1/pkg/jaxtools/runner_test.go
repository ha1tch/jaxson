// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import (
	"reflect"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	jb "github.com/ha1tch/jaxson/pkg/jaxson/build"
)

// Both shapes a host will pass must be assignable to Runner; this is a
// compile-time check.
var (
	_ Runner = jaxson.Run
	_ Runner = jaxson.Profile{}.Run
)

func TestNewSessionWithUsesTheGivenRunner(t *testing.T) {
	var seen []map[string]any
	run := func(p map[string]any) (any, *jaxson.Err) {
		seen = append(seen, p)
		return "ok", nil
	}
	pkg := map[string]any{"toy": "0.1", "input": "discarded"}
	s := NewSessionWith(pkg, run)

	in := map[string]any{"x": "y"}
	out, err := s.Step(in)
	if err != nil || out != "ok" {
		t.Fatalf("Step = %v, %v", out, err)
	}
	if len(seen) != 1 || seen[0]["toy"] != "0.1" || !reflect.DeepEqual(seen[0]["input"], in) {
		t.Fatalf("runner saw %v, want the package with this turn's input", seen)
	}
	if pkg["input"] != "discarded" || len(pkg) != 2 {
		t.Errorf("the caller's package map was mutated: %v", pkg)
	}
	if s.Turn() != 1 || s.Last() != "ok" || s.LastErr() != nil {
		t.Errorf("state after success: turn %d last %v err %v", s.Turn(), s.Last(), s.LastErr())
	}
}

func TestNewSessionWithFailureHoldsState(t *testing.T) {
	calls := 0
	run := func(map[string]any) (any, *jaxson.Err) {
		calls++
		if calls == 2 {
			return nil, &jaxson.Err{Cat: "TOY_ERROR", Code: "NO", Msg: "refused"}
		}
		return calls, nil
	}
	s := NewSessionWith(map[string]any{}, run)
	if _, err := s.Step(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Step(nil); err == nil || err.Cat != "TOY_ERROR" {
		t.Fatalf("second turn: %v", err)
	}
	if s.Turn() != 1 || s.Last() != 1 || s.LastErr() == nil {
		t.Errorf("a rejected turn must leave state alone: turn %d last %v err %v", s.Turn(), s.Last(), s.LastErr())
	}
	if _, err := s.Step(nil); err != nil || s.Turn() != 2 || s.LastErr() != nil {
		t.Errorf("retry after a rejected turn: %v turn %d err %v", err, s.Turn(), s.LastErr())
	}
}

func TestNewSessionWithNilRunnerIsTheCore(t *testing.T) {
	pkg, err := jb.NewPackage().
		InputSchema(jb.Anything()).OutputSchema(jb.Anything()).
		Program(jb.Set(jb.Output(), jb.Tpl().Set("a", jb.Int(1)))).
		Map()
	if err != nil {
		t.Fatal(err)
	}
	out, e := NewSessionWith(pkg, nil).Step(map[string]any{})
	if e != nil || jaxson.Show(out) != `{"a":1}` {
		t.Fatalf("nil runner: %v, %v", jaxson.Show(out), e)
	}
}
