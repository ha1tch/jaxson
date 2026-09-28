// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Phase 0: this replaces the original jaxrun.go main()'s printed
// pass/fail report with an equivalent go test. The comparison logic
// (check) is moved here unchanged from the original file; it is a test
// helper, not part of the library's public API, so it stays in
// _test.go rather than in cmd/jaxrun.

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// check mirrors the original jaxrun.go's fixture-comparison helper
// verbatim (originally the package-level function `check`, lines
// 1285-1305 of jaxrun.go).
func fixtureCheck(exp map[string]any, out any, e *Err) (bool, string) {
	if want, has := exp["error"].(map[string]any); has {
		if e == nil {
			return false, "expected an error, got output " + show(out)
		}
		if e.Cat != want["category"] {
			return false, "wrong category: " + e.Error()
		}
		if c, has := want["code"]; has && c != e.Code {
			return false, "wrong code: " + e.Error()
		}
		return true, "-> " + e.Msg
	}
	if e != nil {
		return false, "unexpected error: " + e.Error()
	}
	if !Equal(exp["output"], out) {
		return false, "output " + show(out) + " != expected " + show(exp["output"])
	}
	return true, ""
}

func TestFixtures(t *testing.T) {
	raw, err := os.ReadFile("jaxson-v0.1.0-fixtures.json")
	if err != nil {
		t.Fatalf("reading fixtures: %v", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var cases []any
	if err := dec.Decode(&cases); err != nil {
		t.Fatalf("fixtures are not valid JSON: %v", err)
	}
	norm(cases)
	for _, c := range cases {
		cm := c.(map[string]any)
		name, _ := cm["name"].(string)
		t.Run(name, func(t *testing.T) {
			out, e := Run(cm)
			ok, detail := fixtureCheck(cm["expect"].(map[string]any), out, e)
			if !ok {
				t.Errorf("%s: %s", name, detail)
			} else if detail != "" {
				t.Logf("%s", detail)
			}
		})
	}
}
