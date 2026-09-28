// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Runs examples/showcase/showcase-fixtures.json (28 cases: 12 example
// packages plus 16 variants covering other branches and failure paths)
// through the real implementation. Same fixture format, and the same
// fixtureCheck comparison, as TestFixtures.
//
// The expected values in that file were recorded from an independent
// Python port of this package's semantics (which itself passes all 28
// official fixtures), not from this Go code — so a failure here means
// either the Go implementation or the Python port is wrong, and is worth
// investigating rather than "fixing" by regenerating the expectations.
//
// Path: JAXSON_SHOWCASE overrides; otherwise the default assumes the
// repository layout jaxson/src/<tree>/pkg/jaxson -> jaxson/examples/showcase.
// The test skips (not fails) if the file cannot be found.

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestShowcaseFixtures(t *testing.T) {
	path := os.Getenv("JAXSON_SHOWCASE")
	if path == "" {
		path = "../../../../examples/showcase/showcase-fixtures.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("showcase fixtures not found at %s (set JAXSON_SHOWCASE): %v", path, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var cases []any
	if err := dec.Decode(&cases); err != nil {
		t.Fatalf("showcase fixtures are not valid JSON: %v", err)
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
