// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson_test

// Profile tests. A toy dialect ("toy": "0.1", its own limit "depth", its
// own instruction "bind") is built from outside the package, so this
// proves the pipeline seam is sufficient for a host such as Shaxon before
// that host depends on it.

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
	jb "github.com/ha1tch/jaxson/pkg/jaxson/build"
)

type toy struct {
	jaxson.NoHooks
	trace  *[]string
	failAt string
	got    string
}

func (h *toy) note(s string) { *h.trace = append(*h.trace, s) }

func (h *toy) fail(at string) {
	if h.failAt == at {
		jaxson.Fail("TOY_ERROR", "AT_"+at, "toy failure at %s", at)
	}
}

func (h *toy) Static(map[string]any) { h.note("static"); h.fail("static") }
func (h *toy) Host() any             { h.note("host"); return "toy-host" }
func (h *toy) Start(*jaxson.Machine) { h.note("start") }
func (h *toy) AfterInput(*jaxson.Machine) {
	h.note("afterInput")
	h.fail("input")
}
func (h *toy) AfterOutput(*jaxson.Machine) {
	h.note("afterOutput")
	h.fail("output")
}

func (h *toy) Instructions() map[string]jaxson.InstructionDef {
	h.note("instructions")
	tbl := jaxson.CoreInstructions()
	tbl["bind"] = jaxson.InstructionDef{
		Name: "bind", Req: []string{"value", "body"},
		Check: func(in map[string]any, c *jaxson.Checker) {
			if c.Host != "toy-host" {
				jaxson.Fail("TOY_ERROR", "NO_HOST", "Checker.Host not delivered")
			}
			if e := jaxson.CheckOperand(in["value"]); e != nil {
				jaxson.Fail(e.Cat, e.Code, "%s", e.Msg)
			}
			if e := jaxson.CheckOperand(in["body"], "focus"); e != nil {
				jaxson.Fail(e.Cat, e.Code, "%s", e.Msg)
			}
		},
		Exec: func(m *jaxson.Machine, in map[string]any) {
			m.Step()
			h.fail("exec")
			v := m.Eval(in["value"])
			m.WithLocal("focus", v, func() {
				outer := jaxson.Show(m.Eval(in["body"]))
				var inner string
				m.WithLocal("focus", jaxson.Normalize(json.Number("8")), func() {
					inner = jaxson.Show(m.Eval(in["body"]))
				})
				h.got = outer + "," + inner + "," + jaxson.Show(m.Eval(in["body"]))
			})
		},
	}
	return tbl
}

func toyProfile(trace *[]string, failAt string, last **toy) jaxson.Profile {
	return jaxson.Profile{
		Key: "toy", Versions: []string{"0.1"}, Limits: []string{"depth"},
		Begin: func(map[string]any) jaxson.Hooks {
			*trace = append(*trace, "begin")
			t := &toy{trace: trace, failAt: failAt}
			if last != nil {
				*last = t
			}
			return t
		},
	}
}

// corePkg is a small valid core package, assembled by the public builder
// so the schema vocabulary is not guessed here.
func corePkg(t testing.TB) map[string]any {
	t.Helper()
	m, err := jb.NewPackage().
		InputSchema(jb.Anything()).OutputSchema(jb.Anything()).
		Input(map[string]any{}).
		Program(jb.Set(jb.Output(), jb.Tpl().Set("a", jb.Int(1)))).
		Map()
	if err != nil {
		t.Fatalf("building the core package: %v", err)
	}
	return m
}

func toyPkg(t testing.TB) map[string]any {
	m := corePkg(t)
	delete(m, "jaxson")
	m["toy"] = "0.1"
	m["limits"] = dec(t, `{"steps":100,"depth":3}`).(map[string]any)
	m["program"] = dec(t, `[{"op":"bind","value":7,"body":{"$path":["local","focus"]}}]`)
	return m
}

func TestProfileHookOrderAndHostInstruction(t *testing.T) {
	var trace []string
	var h *toy
	if _, err := toyProfile(&trace, "", &h).Run(toyPkg(t)); err != nil {
		t.Fatalf("toy run failed: %v", err)
	}
	want := []string{"begin", "static", "instructions", "host", "start", "afterInput", "afterOutput"}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("hook order %v, want %v", trace, want)
	}
	if h.got != "7,8,7" {
		t.Fatalf("WithLocal nesting: got %q, want 7,8,7", h.got)
	}
}

func TestProfileVersionKeyAndLimitOwnership(t *testing.T) {
	var trace []string
	p := toyProfile(&trace, "", nil)

	if _, err := jaxson.Run(toyPkg(t)); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("core Run accepted a toy package: %v", err)
	}

	bad := toyPkg(t)
	bad["toy"] = "9.9"
	if _, err := p.Run(bad); err == nil || err.Cat != "VERSION_ERROR" || !strings.Contains(err.Msg, "toy") {
		t.Errorf("unlisted version: %v", err)
	}
	if len(trace) != 0 {
		t.Errorf("Begin ran before the version gate: %v", trace)
	}

	fromCore := corePkg(t)
	fromCore["toy"] = "0.1"
	if _, err := p.Run(fromCore); err != nil {
		t.Errorf("a toy package that also carries jaxson: %v", err)
	}
	onlyCore := corePkg(t)
	if _, err := p.Run(onlyCore); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("toy profile accepted a package with no toy member: %v", err)
	}

	unowned := toyPkg(t)
	unowned["limits"] = dec(t, `{"steps":100,"bogus":1}`)
	if _, err := p.Run(unowned); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("unowned limit accepted: %v", err)
	}

	coreWithDepth := corePkg(t)
	coreWithDepth["limits"] = dec(t, `{"depth":3}`)
	if _, err := jaxson.Run(coreWithDepth); err == nil || err.Cat != "VERSION_ERROR" {
		t.Errorf("core Run accepted a limit only the toy dialect owns: %v", err)
	}
}

func TestProfileFailurePropagatesAndStopsLaterStages(t *testing.T) {
	cases := []struct {
		failAt string
		want   []string
	}{
		{"static", []string{"begin", "static"}},
		{"input", []string{"begin", "static", "instructions", "host", "start", "afterInput"}},
		{"exec", []string{"begin", "static", "instructions", "host", "start", "afterInput"}},
		{"output", []string{"begin", "static", "instructions", "host", "start", "afterInput", "afterOutput"}},
	}
	for _, c := range cases {
		var trace []string
		_, err := toyProfile(&trace, c.failAt, nil).Run(toyPkg(t))
		if err == nil || err.Cat != "TOY_ERROR" || err.Code != "AT_"+c.failAt {
			t.Errorf("%s: got %v, want TOY_ERROR/AT_%s", c.failAt, err, c.failAt)
		}
		if !reflect.DeepEqual(trace, c.want) {
			t.Errorf("%s: stages ran %v, want %v", c.failAt, trace, c.want)
		}
	}
}

func TestHostInstructionStaticCheckSeesDeclaredLocalsOnly(t *testing.T) {
	var trace []string
	pkg := toyPkg(t)
	pkg["program"] = dec(t, `[{"op":"bind","value":7,"body":{"$path":["local","other"]}}]`)
	if _, err := toyProfile(&trace, "", nil).Run(pkg); err == nil || err.Cat != "PROGRAM_ERROR" {
		t.Fatalf("undeclared local in a host operand: %v", err)
	}
	if err := jaxson.CheckOperand(dec(t, `{"$path":["local","focus"]}`)); err == nil {
		t.Error("CheckOperand accepted a local nobody declared")
	}
	if err := jaxson.CheckOperand(dec(t, `{"$path":["local","focus"]}`), "focus"); err != nil {
		t.Errorf("CheckOperand refused a declared local: %v", err)
	}
	if err := jaxson.CheckPath(dec(t, `["local","focus"]`), "focus"); err != nil {
		t.Errorf("CheckPath refused a declared local: %v", err)
	}
	if err := jaxson.CheckPath(dec(t, `["local","focus"]`)); err == nil {
		t.Error("CheckPath accepted a local nobody declared")
	}
}

func TestZeroProfileIsTheCore(t *testing.T) {
	a, ea := jaxson.Run(corePkg(t))
	b, eb := jaxson.Profile{}.Run(corePkg(t))
	if ea != nil || eb != nil || jaxson.Show(a) != jaxson.Show(b) || jaxson.Show(a) != `{"a":1}` {
		t.Fatalf("Run=%v,%v  Profile{}.Run=%v,%v", jaxson.Show(a), ea, jaxson.Show(b), eb)
	}
}

func TestProfileConcurrentRunsShareNothing(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var trace []string
			var h *toy
			if _, err := toyProfile(&trace, "", &h).Run(toyPkg(t)); err != nil || h.got != "7,8,7" {
				t.Errorf("concurrent run: %v %q", err, h.got)
			}
		}()
	}
	wg.Wait()
}
