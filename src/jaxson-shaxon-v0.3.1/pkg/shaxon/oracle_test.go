// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// ORACLE TEST: SCHEDULED FOR DELETION together with pkg/jaxson/tree_oracle.go.
//
// The closure compiler must be indistinguishable from the tree-walking
// interpreter it replaced. Here that is checked at the Shaxon level, where
// the host forms, `check`, indices, shapes, relations and aggregate
// expansions all run through the compiler's fall-back paths:
//
//   - every conformance fixture, and every authorisation example case, is
//     run once per interpreter on its own deep copy and the output, the
//     report, the step total and the error must be identical;
//   - then every run that succeeded is repeated under a series of tighter
//     step limits (1, a third, a half, all but one of its steps), so the
//     order in which things are charged is compared, not only the total:
//     the same limit must stop both interpreters with the same error.

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

type shaxonOutcome struct {
	out, report, err string
	steps            int64
}

func (o shaxonOutcome) String() string {
	if o.err != "" {
		return "error " + o.err
	}
	return fmt.Sprintf("output %s, report %s, %d steps", o.out, o.report, o.steps)
}

func runShaxon(pkg map[string]any, useTree bool) shaxonOutcome {
	jaxson.UseTreeInterpreter(useTree)
	defer jaxson.UseTreeInterpreter(false)
	res, e := Run(jaxson.Clone(pkg).(map[string]any))
	if e != nil {
		return shaxonOutcome{err: e.Error()}
	}
	o := shaxonOutcome{out: jaxson.Show(res.Output), steps: res.Steps, report: "none"}
	if res.Report != nil {
		o.report = jaxson.Show(res.Report.Value())
	}
	return o
}

// withLimit returns a shallow copy of the package with limits.steps set.
func withLimit(pkg map[string]any, n int64) map[string]any {
	cp := make(map[string]any, len(pkg))
	for k, v := range pkg {
		cp[k] = v
	}
	lim := map[string]any{}
	if old, ok := pkg["limits"].(map[string]any); ok {
		for k, v := range old {
			lim[k] = v
		}
	}
	lim["steps"] = new(big.Rat).SetInt64(n)
	cp["limits"] = lim
	return cp
}

// compareBoth runs pkg under both interpreters, at its own limit and then at
// tighter ones, and reports every difference.
func compareBoth(t *testing.T, label string, pkg map[string]any) (runs int) {
	t.Helper()
	a, b := runShaxon(pkg, false), runShaxon(pkg, true)
	runs++
	if a != b {
		t.Errorf("%s:\n compiled: %s\n     tree: %s", label, a, b)
		return
	}
	if a.err != "" || a.steps < 2 {
		return
	}
	for _, n := range []int64{1, a.steps / 3, a.steps / 2, a.steps - 1} {
		if n < 1 {
			continue
		}
		q := withLimit(pkg, n)
		x, y := runShaxon(q, false), runShaxon(q, true)
		runs++
		if x != y {
			t.Errorf("%s at limit %d:\n compiled: %s\n     tree: %s", label, n, x, y)
		}
	}
	return
}

func TestOracleFixtures(t *testing.T) {
	raw, err := os.ReadFile(fixturesFile)
	if err != nil {
		t.Fatal(err)
	}
	v, perr := jaxson.ParseJSON(raw)
	if perr != nil {
		t.Fatal(perr)
	}
	cases := v.([]any)
	runs := 0
	for _, c := range cases {
		fx := c.(map[string]any)
		pkg := map[string]any{}
		for k, val := range fx {
			if k != "name" && k != "note" && k != "expect" {
				pkg[k] = val
			}
		}
		runs += compareBoth(t, fmt.Sprint(fx["name"]), pkg)
	}
	if len(cases) < 161 {
		t.Fatalf("only %d fixtures compared", len(cases))
	}
	t.Logf("%d fixtures, %d runs (including tightened step limits): compiled and tree agree", len(cases), runs)
}

func TestOracleAuthzExamples(t *testing.T) {
	files, err := filepath.Glob("../../examples/shaxon/authz/*.cases.json")
	if err != nil || len(files) != 5 {
		t.Fatalf("want the five example case files, got %v (%v)", files, err)
	}
	runs, cases := 0, 0
	for _, cf := range files {
		pkgPath := strings.TrimSuffix(cf, ".cases.json") + ".json"
		name := filepath.Base(strings.TrimSuffix(cf, ".cases.json"))
		for _, c := range readValue(t, cf).([]any) {
			cm := c.(map[string]any)
			pkg := readValue(t, pkgPath).(map[string]any)
			in := pkg["input"].(map[string]any)
			if override, ok := cm["input"].(map[string]any); ok {
				for k, v := range override {
					in[k] = v
				}
			}
			runs += compareBoth(t, name+": "+fmt.Sprint(cm["name"]), pkg)
			cases++
		}
	}
	t.Logf("%d authz cases, %d runs: compiled and tree agree", cases, runs)
}
