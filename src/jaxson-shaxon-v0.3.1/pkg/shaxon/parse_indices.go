// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Core section 3. source and key are Jaxson operands, not literal path
// arrays: "source": {"$path": [...]} is the ordinary form, and either
// could in principle be any operand that resolves appropriately at
// evaluation time. source is checked with no extra locals; key is
// checked with "item" in scope, since it is evaluated once per source
// element with local.item bound — the same shape as a `for` binding.
//
// What this file does NOT check, because it needs real data and Phase 2
// runs before any exists: "must resolve to an array" (source), "must
// resolve to a scalar" (key), and "a repeated key in a non-multi index is
// SHAPE_ERROR at build time" — all three are explicitly build-time in
// core section 3, and belong to Phase 4's indices.go.

import "github.com/ha1tch/jaxson/pkg/jaxson"

func parseIndices(pkg map[string]any, r *Registries) {
	raw, has := pkg["indices"]
	if !has {
		return
	}
	m, ok := raw.(map[string]any)
	if !ok {
		failLoad("indices must be an object")
	}
	for _, name := range jaxson.SortedKeys(m) {
		entry, ok := m[name].(map[string]any)
		if !ok {
			failLoad("indices.%s must be an object", name)
		}
		for k := range entry {
			if k != "source" && k != "key" && k != "multi" {
				failLoad("indices.%s: unknown key %q", name, k)
			}
		}
		source, hasSource := entry["source"]
		if !hasSource {
			failLoad("indices.%s: needs a source", name)
		}
		if e := jaxson.CheckOperand(source); e != nil {
			failLoad("indices.%s.source: %s", name, e.Msg)
		}
		key, hasKey := entry["key"]
		if !hasKey {
			failLoad("indices.%s: needs a key", name)
		}
		if e := jaxson.CheckOperand(key, "item"); e != nil {
			failLoad("indices.%s.key: %s", name, e.Msg)
		}
		multi := false
		if mv, present := entry["multi"]; present {
			b, ok := mv.(bool)
			if !ok {
				failLoad("indices.%s: multi must be a boolean", name)
			}
			multi = b
		}
		r.Indices[name] = IndexDecl{Name: name, Source: source, Key: key, Multi: multi}
	}
}
