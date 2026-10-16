// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// The conformance fixtures for strict numeric mode (Jaxson core design,
// sections 1 and 2). They are in the package format with `expect`, like the
// main fixture file, except that a fixture which tests how a runtime reads a
// package gives it as text in `packageText`, in place of the package members:
// the runtime is handed that text as it would be handed a file.

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestStrictNumberFixtures(t *testing.T) {
	raw, err := os.ReadFile("jaxson-v0.1.0-strict-numbers-fixtures.json")
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
	if len(cases) != 11 {
		t.Errorf("expected 11 fixtures, found %d", len(cases))
	}
	for _, c := range cases {
		cm := c.(map[string]any)
		name, _ := cm["name"].(string)
		t.Run(name, func(t *testing.T) {
			var out any
			var e *Err
			if text, ok := cm["packageText"].(string); ok {
				out, e = RunJSON([]byte(text))
			} else {
				out, e = Run(cm)
			}
			ok, detail := fixtureCheck(cm["expect"].(map[string]any), out, e)
			if !ok {
				t.Errorf("%s: %s", name, detail)
			}
		})
	}
}
