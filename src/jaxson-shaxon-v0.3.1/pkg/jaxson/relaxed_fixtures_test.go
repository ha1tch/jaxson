// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// The conformance fixtures for relaxed numeric mode (Jaxson core design,
// section 2). They are in the package format with `expect`, like the strict
// fixtures, and are read and run in relaxed mode: the file itself holds numbers
// beyond the Real limits, which a strict reader would refuse.

import (
	"os"
	"testing"
)

func TestRelaxedNumberFixtures(t *testing.T) {
	raw, err := os.ReadFile("jaxson-v0.1.0-relaxed-numbers-fixtures.json")
	if err != nil {
		t.Fatalf("reading fixtures: %v", err)
	}
	v, perr := ParseJSONMode(raw, Relaxed)
	if perr != nil {
		t.Fatalf("fixtures are not valid JSON in relaxed mode: %v", perr)
	}
	cases := v.([]any)
	if len(cases) != 14 {
		t.Errorf("expected 14 fixtures, found %d", len(cases))
	}
	pr := CoreProfile()
	pr.Numbers = Relaxed
	for _, c := range cases {
		cm := c.(map[string]any)
		name, _ := cm["name"].(string)
		t.Run(name, func(t *testing.T) {
			var out any
			var e *Err
			if text, ok := cm["packageText"].(string); ok {
				out, e = pr.RunJSON([]byte(text))
			} else {
				out, e = pr.Run(cm)
			}
			ok, detail := fixtureCheck(cm["expect"].(map[string]any), out, e)
			if !ok {
				t.Errorf("%s: %s", name, detail)
			}
		})
	}
}
