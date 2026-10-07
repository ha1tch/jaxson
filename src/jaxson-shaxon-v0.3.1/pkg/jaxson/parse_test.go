// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson_test

import (
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func TestParseJSONAcceptsAndNormalises(t *testing.T) {
	v, err := jaxson.ParseJSON([]byte(` {"a": [1, 2.50, {"b": null}], "c": "x", "d": true, "e": {}} `))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if got := jaxson.Show(v); got != `{"a":[1,2.5,{"b":null}],"c":"x","d":true,"e":{}}` {
		t.Fatalf("got %s", got)
	}
}

func TestParseJSONKeepsDecimalsExact(t *testing.T) {
	v, err := jaxson.ParseJSON([]byte(`[0.1, 12345678901234567890.123456789, 1e3]`))
	if err != nil {
		t.Fatal(err)
	}
	if got := jaxson.Show(v); got != `[0.1,12345678901234567890.123456789,1000]` {
		t.Fatalf("got %s", got)
	}
}

func TestParseJSONRejectsDuplicateKeysAtAnyDepth(t *testing.T) {
	for name, src := range map[string]string{
		"top":       `{"a":1,"a":2}`,
		"nested":    `{"x":{"a":1,"b":2,"a":3}}`,
		"in array":  `{"x":[{"k":1,"k":1}]}`,
		"same data": `{"a":1,"a":1}`,
	} {
		_, err := jaxson.ParseJSON([]byte(src))
		if err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("%s: got %v, want PARSE_ERROR", name, err)
		}
	}
}

func TestParseJSONAllowsTheSameKeyInDifferentObjects(t *testing.T) {
	if _, err := jaxson.ParseJSON([]byte(`{"a":{"k":1},"b":{"k":2},"c":[{"k":3},{"k":4}]}`)); err != nil {
		t.Fatalf("rejected a legitimate document: %v", err)
	}
}

func TestParseJSONRejectsNotJSONAndTrailingData(t *testing.T) {
	for name, src := range map[string]string{
		"empty":         ``,
		"garbage":       `nope`,
		"unterminated":  `{"a":`,
		"trailing":      `{"a":1} {"b":2}`,
		"trailing junk": `[1] x`,
		"bad close":     `{"a":1`,
	} {
		_, err := jaxson.ParseJSON([]byte(src))
		if err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("%s: got %v, want PARSE_ERROR", name, err)
		}
	}
}
