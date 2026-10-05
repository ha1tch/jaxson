// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// The `unit` cost table must reproduce the original flat accounting — one
// step per instruction, per loop iteration begun, per operator applied —
// exactly (docs/proposals/step-cost-model.md, principle 1). This test pins
// that: testdata/unit-step-counts.json holds, for every fixture in
// jaxson-v0.1.0-fixtures.json and examples/showcase/showcase-fixtures.json,
// the exact number of steps the case consumes before reaching its outcome.
//
// The golden file was generated from the code as it stood BEFORE the cost
// table was introduced (CM-1), so a pass here means the refactor changed
// no count, not merely that two new numbers agree. Regenerate only for a
// deliberate change to the `unit` table, which is a change to the language:
//
//	JAXSON_WRITE_GOLDEN=1 go test ./pkg/jaxson -run TestUnitStepCountsFrozen
//
// Steps are measured as the smallest declared limit under which the case's
// outcome is no longer RESOURCE_ERROR/STEPS, found by bisection, so the
// measurement needs nothing from the Machine beyond the public Run.

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"sort"
	"testing"
)

const goldenSteps = "testdata/unit-step-counts.json"

func stepsNeeded(t *testing.T, p map[string]any) int {
	t.Helper()
	isSteps := func(e *Err) bool { return e != nil && e.Cat == "RESOURCE_ERROR" && e.Code == "STEPS" }
	run := func(n int) *Err {
		q := Clone(p).(map[string]any)
		l, _ := q["limits"].(map[string]any)
		if l == nil {
			l = map[string]any{}
		}
		l["steps"] = big.NewRat(int64(n), 1)
		q["limits"] = l
		_, e := Run(q)
		return e
	}
	lo, hi := 1, 1<<20
	if isSteps(run(hi)) {
		t.Fatalf("case needs more than %d steps", hi)
	}
	for lo < hi {
		mid := (lo + hi) / 2
		if isSteps(run(mid)) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func loadCases(t *testing.T, path string) []any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var cases []any
	if err := dec.Decode(&cases); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	norm(cases)
	return cases
}

func measureAll(t *testing.T) map[string]int {
	t.Helper()
	showcase := os.Getenv("JAXSON_SHOWCASE")
	if showcase == "" {
		showcase = "../../../../examples/showcase/showcase-fixtures.json"
	}
	got := map[string]int{}
	for _, src := range []struct{ prefix, path string }{
		{"core", "jaxson-v0.1.0-fixtures.json"},
		{"showcase", showcase},
	} {
		for _, c := range loadCases(t, src.path) {
			cm := c.(map[string]any)
			name, _ := cm["name"].(string)
			got[src.prefix+"/"+name] = stepsNeeded(t, cm)
		}
	}
	return got
}

func TestUnitStepCountsFrozen(t *testing.T) {
	got := measureAll(t)
	if len(got) == 0 {
		t.Fatal("no fixtures found")
	}
	if os.Getenv("JAXSON_WRITE_GOLDEN") != "" {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var buf bytes.Buffer
		buf.WriteString("{\n")
		for i, k := range keys {
			kb, _ := json.Marshal(k)
			buf.WriteString("  " + string(kb) + ": ")
			vb, _ := json.Marshal(got[k])
			buf.Write(vb)
			if i < len(keys)-1 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
		}
		buf.WriteString("}\n")
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenSteps, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d step counts to %s", len(got), goldenSteps)
		return
	}
	raw, err := os.ReadFile(goldenSteps)
	if err != nil {
		t.Fatalf("golden file missing: %v", err)
	}
	var want map[string]int
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	for k, w := range want {
		g, ok := got[k]
		if !ok {
			// The showcase file may legitimately be absent (the showcase
			// test skips in that case); core cases never may be.
			if len(k) >= 5 && k[:5] == "core/" {
				t.Errorf("%s: case no longer present", k)
			}
			continue
		}
		if g != w {
			t.Errorf("%s: %d steps, want %d", k, g, w)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: not in the golden file (regenerate deliberately if a case was added)", k)
		}
	}
}
