// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import (
	"fmt"
	"math/big"
	"unicode/utf8"
)

// ---------------------------------------------------------------- contract schemas

var schemaKeys = map[string][]string{
	"any": {}, "null": {}, "boolean": {},
	"number": {"int", "min", "max"}, "string": {"minLen", "maxLen", "enum"},
	"array": {"items", "minItems", "maxItems"}, "object": {"fields", "required", "extra"},
}

func checkSchema(s any) {
	m, ok := s.(map[string]any)
	if !ok {
		fail("SCHEMA_ERROR", "", "a schema must be an object")
	}
	if a, has := m["anyOf"]; has {
		for k := range m {
			if k != "anyOf" && k != "nullable" {
				fail("SCHEMA_ERROR", "", "anyOf schema has extra keyword %q", k)
			}
		}
		arr, ok := a.([]any)
		if !ok || len(arr) == 0 {
			fail("SCHEMA_ERROR", "", "anyOf needs a non-empty array")
		}
		for _, sub := range arr {
			checkSchema(sub)
		}
		return
	}
	t, ok := m["type"].(string)
	keys, known := schemaKeys[t]
	if !ok || !known {
		fail("SCHEMA_ERROR", "", "unknown or missing schema type")
	}
	allowed := map[string]bool{"type": true, "nullable": true}
	for _, k := range keys {
		allowed[k] = true
	}
	for k := range m {
		if !allowed[k] {
			fail("SCHEMA_ERROR", "", "keyword %q is not valid for type %s", k, t)
		}
	}
	switch t {
	case "array":
		if it, has := m["items"]; has {
			checkSchema(it)
		}
	case "object":
		fields, _ := m["fields"].(map[string]any)
		for _, f := range fields {
			checkSchema(f)
		}
		if req, has := m["required"]; has {
			arr, ok := req.([]any)
			if !ok {
				fail("SCHEMA_ERROR", "", "required must be an array")
			}
			for _, n := range arr {
				ns, ok := n.(string)
				if _, declared := fields[ns]; !ok || !declared {
					fail("SCHEMA_ERROR", "", "required names an undeclared field")
				}
			}
		}
		if ex, has := m["extra"]; has && ex != "reject" && ex != "allow" {
			fail("SCHEMA_ERROR", "", "extra must be reject or allow")
		}
	}
}

func ratOf(v any) *big.Rat { r, _ := v.(*big.Rat); return r }

// ---------------------------------------------------------------- primitive keyword checks
//
// Phase 1: factored out of validate() below so a host package's own
// field-shape checking (Shaxon's `kind: "string"`/"number"/"array" with
// the identical minLen/maxLen/enum, min/max/int, minItems/maxItems
// keywords) calls the same functions validate() does, rather than
// re-implementing them. Each takes the already-extracted constraint
// values (nil meaning "not declared") and returns "" on success or a
// short reason fragment on failure — the same fragments validate()
// always produced, just no longer duplicated if a second caller needs
// them.

// CheckNumber applies the number-kind keyword checks (int/min/max).
func CheckNumber(r *big.Rat, wantInt bool, min, max *big.Rat) string {
	if wantInt && !r.IsInt() {
		return "expected an integer"
	}
	if min != nil && r.Cmp(min) < 0 {
		return "below minimum"
	}
	if max != nil && r.Cmp(max) > 0 {
		return "above maximum"
	}
	return ""
}

// CheckStringLen applies the string-kind minLen/maxLen/enum keyword
// checks. enum may be nil (not declared).
func CheckStringLen(s string, minLen, maxLen *big.Rat, enum []any) string {
	n := int64(utf8.RuneCountInString(s))
	if minLen != nil && big.NewRat(n, 1).Cmp(minLen) < 0 {
		return "too short"
	}
	if maxLen != nil && big.NewRat(n, 1).Cmp(maxLen) > 0 {
		return "too long"
	}
	if enum != nil {
		found := false
		for _, e := range enum {
			if es, ok := e.(string); ok && es == s {
				found = true
				break
			}
		}
		if !found {
			return "not in enum"
		}
	}
	return ""
}

// CheckArrayLen applies the array-kind minItems/maxItems keyword checks.
func CheckArrayLen(n int, minItems, maxItems *big.Rat) string {
	nn := big.NewRat(int64(n), 1)
	if minItems != nil && nn.Cmp(minItems) < 0 {
		return "too few items"
	}
	if maxItems != nil && nn.Cmp(maxItems) > 0 {
		return "too many items"
	}
	return ""
}

// validate returns "" when v satisfies s, otherwise the first failure.
func validate(s, v any, path string) string {
	m := s.(map[string]any)
	if nl, _ := m["nullable"].(bool); nl && v == nil {
		return ""
	}
	if a, has := m["anyOf"]; has {
		for _, sub := range a.([]any) {
			if validate(sub, v, path) == "" {
				return ""
			}
		}
		return path + ": matches no anyOf alternative"
	}
	t := m["type"].(string)
	if t == "any" {
		return ""
	}
	if got := TypeName(v); got != t {
		return fmt.Sprintf("%s: expected %s, got %s", path, t, got)
	}
	switch t {
	case "number":
		i, _ := m["int"].(bool)
		if msg := CheckNumber(v.(*big.Rat), i, ratOf(m["min"]), ratOf(m["max"])); msg != "" {
			return path + ": " + msg
		}
	case "string":
		var enum []any
		if en, has := m["enum"].([]any); has {
			enum = en
		}
		if msg := CheckStringLen(v.(string), ratOf(m["minLen"]), ratOf(m["maxLen"]), enum); msg != "" {
			return path + ": " + msg
		}
	case "array":
		arr := v.([]any)
		if msg := CheckArrayLen(len(arr), ratOf(m["minItems"]), ratOf(m["maxItems"])); msg != "" {
			return path + ": " + msg
		}
		if it, has := m["items"]; has {
			for i, e := range arr {
				if msg := validate(it, e, fmt.Sprintf("%s[%d]", path, i)); msg != "" {
					return msg
				}
			}
		}
	case "object":
		obj := v.(*Object)
		fields, _ := m["fields"].(map[string]any)
		if req, has := m["required"].([]any); has {
			for _, n := range req {
				if !obj.Has(n.(string)) {
					return fmt.Sprintf("%s: missing required %q", path, n)
				}
			}
		}
		for _, k := range obj.SortedKeys() {
			if fs, declared := fields[k]; declared {
				member, _ := obj.Get(k)
				if msg := validate(fs, member, path+"."+k); msg != "" {
					return msg
				}
			} else if m["extra"] != "allow" {
				return fmt.Sprintf("%s: unexpected member %q", path, k)
			}
		}
	}
	return ""
}
