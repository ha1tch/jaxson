// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// parseJSONReference is the reader ParseJSON was before it became a
// hand-written scanner: encoding/json's token stream, with duplicate-key and
// trailing-data checks added. It is kept as the independent reference the
// scanner is compared against (parse_diff_test.go). It is not scheduled for
// deletion: encoding/json is exactly the thing worth agreeing with.

import (
	"bytes"
	"encoding/json"
	"io"
)

func parseJSONReference(raw []byte) (v any, err *Err) {
	defer recoverErr(&err)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v = refTokens(dec)
	if _, e := dec.Token(); e != io.EOF {
		fail("PARSE_ERROR", "", "unexpected data after the JSON document")
	}
	return norm(v), nil
}

func refTokens(dec *json.Decoder) any {
	tok, e := dec.Token()
	if e != nil {
		fail("PARSE_ERROR", "", "not valid JSON: %v", e)
	}
	d, isDelim := tok.(json.Delim)
	if !isDelim {
		return tok
	}
	switch d {
	case '{':
		obj := map[string]any{}
		for dec.More() {
			kt, e := dec.Token()
			if e != nil {
				fail("PARSE_ERROR", "", "not valid JSON: %v", e)
			}
			key, _ := kt.(string)
			if _, dup := obj[key]; dup {
				fail("PARSE_ERROR", "", "duplicate key %q", key)
			}
			obj[key] = refTokens(dec)
		}
		if _, e := dec.Token(); e != nil {
			fail("PARSE_ERROR", "", "not valid JSON: %v", e)
		}
		return obj
	default:
		arr := []any{}
		for dec.More() {
			arr = append(arr, refTokens(dec))
		}
		if _, e := dec.Token(); e != nil {
			fail("PARSE_ERROR", "", "not valid JSON: %v", e)
		}
		return arr
	}
}
