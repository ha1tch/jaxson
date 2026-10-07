// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 6: runs shaxon-v0.3.1-fixtures.json, the conformance fixtures.
//
// A fixture is one complete Shaxon package plus three bookkeeping members
// that are not part of the package: "name", "note" (why the fixture exists
// and what it would catch), and "expect". The package is run exactly as
// written, so a fixture is also a runnable example and any other
// implementation can read the same file.
//
// "expect" has one of two shapes:
//
//	{"output": V}                      the run succeeds and its output equals V
//	{"error": {"category": C, "code": X}}   the run fails with that category
//	                                   and, if "code" is given, that code
//
// and may add, to the success shape only:
//
//	"report": R    the combined report (conforms and violations) the run
//	               returns alongside its output; "report": null means that no
//	               report is returned at all. "conforms" must be equal and
//	               "violations" must have the same length, in the same order.
//	               Each expected violation lists only the members the core
//	               fixes for that case (a message the core leaves to the
//	               runtime is simply not listed); every listed member must be
//	               equal in the run's violation, and {"$absent": true} as a
//	               value requires the member to be missing (not null).
//	"steps": N     the exact total charged to the step counter (used only
//	               where the core fixes the charge: core section 3)
//
// Cost fixtures that would otherwise need an exact total use a step limit
// instead: a limit that a runtime obeying the rule stays under (or exceeds)
// and a runtime ignoring it does not, so they stay valid whatever the
// ordinary instruction charges are.

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const fixturesFile = "shaxon-v0.3.1-fixtures.json"

// reportMismatch compares an expected report with the one a run returned and
// returns "" if they match under the rules in the header, or what differs.
func reportMismatch(want, got map[string]any) string {
	if !jaxson.Equal(want["conforms"], got["conforms"]) {
		return "conforms differs"
	}
	wv, _ := want["violations"].([]any)
	gv, _ := got["violations"].([]any)
	if len(wv) != len(gv) {
		return fmt.Sprintf("%d violations, want %d", len(gv), len(wv))
	}
	for i := range wv {
		w := wv[i].(map[string]any)
		g := gv[i].(map[string]any)
		for k, val := range w {
			if m, ok := val.(map[string]any); ok && len(m) == 1 && m["$absent"] == true {
				if _, present := g[k]; present {
					return fmt.Sprintf("violation %d has %q, which must be absent", i, k)
				}
				continue
			}
			have, present := g[k]
			if !present || !jaxson.Equal(val, have) {
				return fmt.Sprintf("violation %d: %q is %s, want %s", i, k, jaxson.Show(have), jaxson.Show(val))
			}
		}
	}
	return ""
}

func TestFixtures(t *testing.T) {
	raw, err := os.ReadFile(fixturesFile)
	if err != nil {
		t.Fatalf("reading fixtures: %v", err)
	}
	v, perr := jaxson.ParseJSON(raw)
	if perr != nil {
		t.Fatalf("%s: %v", fixturesFile, perr)
	}
	cases, ok := v.([]any)
	if !ok || len(cases) == 0 {
		t.Fatalf("%s must be a non-empty array", fixturesFile)
	}
	seen := map[string]bool{}
	for _, c := range cases {
		fx := c.(map[string]any)
		name, _ := fx["name"].(string)
		if name == "" || seen[name] {
			t.Fatalf("fixture names must be present and unique, got %q", name)
		}
		seen[name] = true
		if _, has := fx["note"].(string); !has {
			t.Errorf("%s: a fixture must say why it exists (note)", name)
		}
		t.Run(name, func(t *testing.T) {
			expect, _ := fx["expect"].(map[string]any)
			if expect == nil {
				t.Fatal("no expect")
			}
			pkg := map[string]any{}
			for k, val := range fx {
				if k != "name" && k != "note" && k != "expect" {
					pkg[k] = val
				}
			}
			res, e := Run(pkg)
			if want, has := expect["error"].(map[string]any); has {
				if e == nil {
					t.Fatalf("want error %v, got output %s", jaxson.Show(want), jaxson.Show(res.Output))
				}
				if e.Cat != want["category"] {
					t.Fatalf("want category %v, got %s", want["category"], e)
				}
				if code, has := want["code"]; has && code != e.Code {
					t.Fatalf("want code %v, got %s", code, e)
				}
				return
			}
			if e != nil {
				t.Fatalf("unexpected error: %s", e)
			}
			if !jaxson.Equal(expect["output"], res.Output) {
				t.Errorf("output %s, want %s", jaxson.Show(res.Output), jaxson.Show(expect["output"]))
			}
			if wantReport, has := expect["report"]; has {
				switch {
				case wantReport == nil && res.Report != nil:
					t.Errorf("want no report, got %s", jaxson.Show(res.Report.Value()))
				case wantReport != nil && res.Report == nil:
					t.Errorf("want report %s, got none", jaxson.Show(wantReport))
				case wantReport != nil:
					if msg := reportMismatch(wantReport.(map[string]any), res.Report.Value()); msg != "" {
						t.Errorf("report %s: %s\nwant %s", jaxson.Show(res.Report.Value()), msg, jaxson.Show(wantReport))
					}
				}
			}
			if wantSteps, has := expect["steps"]; has {
				if !jaxson.Equal(wantSteps, new(big.Rat).SetInt64(res.Steps)) {
					t.Errorf("steps %d, want %s", res.Steps, jaxson.Show(wantSteps))
				}
			}
		})
	}
}
