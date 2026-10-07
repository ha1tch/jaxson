// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Core section 4a. Unlike indices.source/key (Jaxson operands), a
// relation's from.path/to.path are literal path-segment arrays — the same
// shape jaxson.CheckPath expects — and from.field/to.key are plain field
// names (bare strings), not operands at all.
//
// This file also carries the relations-vs-relations half of section 4c's
// redundancy check (two relations entries whose `to` sides normalise to
// the same (path, key)). The other half — an inline reference shorthand
// colliding with a relation's `to` side — needs the shapes registry
// parsed first, so it runs afterward, in parse_shapes.go.
//
// Deferred to Phase 4 (needs real data): "a repeated to.key value,"
// "cardinality: one-to-one uniqueness on from.field."

import "github.com/ha1tch/jaxson/pkg/jaxson"

var validCardinality = map[string]bool{"one-to-many": true, "one-to-one": true}

func parseRelations(pkg map[string]any, r *Registries) {
	raw, has := pkg["relations"]
	if !has {
		return
	}
	m, ok := raw.(map[string]any)
	if !ok {
		failLoad("relations must be an object")
	}
	type toSide struct {
		path []any
		key  string
	}
	seen := map[string]toSide{} // relation name -> its to side, for pairwise redundancy comparison
	for _, name := range jaxson.SortedKeys(m) {
		entry, ok := m[name].(map[string]any)
		if !ok {
			failLoad("relations.%s must be an object", name)
		}
		for k := range entry {
			if k != "from" && k != "to" && k != "cardinality" {
				failLoad("relations.%s: unknown key %q", name, k)
			}
		}
		fromPath, fromField := parseRelationSide(name, "from", entry, "field")
		toPath, toKey := parseRelationSide(name, "to", entry, "key")

		cardinality := "one-to-many"
		if cv, present := entry["cardinality"]; present {
			cs, ok := cv.(string)
			if !ok || !validCardinality[cs] {
				failLoad("relations.%s: cardinality must be \"one-to-many\" or \"one-to-one\"", name)
			}
			cardinality = cs
		}

		for other, side := range seen {
			if pathsEqual(side.path, toPath) && side.key == toKey {
				failLoad("relations.%s and relations.%s both build an index over the same (path, key) — point every consuming field at one of them", other, name)
			}
		}
		seen[name] = toSide{toPath, toKey}

		r.Relations[name] = RelationDecl{
			Name: name, FromPath: fromPath, FromField: fromField,
			ToPath: toPath, ToKey: toKey, Cardinality: cardinality,
		}
	}
}

// parseRelationSide validates one of "from"/"to": {"path": [...], <fieldKey>: "name"}.
func parseRelationSide(relName, side string, entry map[string]any, fieldKey string) (path []any, field string) {
	raw, has := entry[side]
	if !has {
		failLoad("relations.%s: needs %q", relName, side)
	}
	sm, ok := raw.(map[string]any)
	if !ok {
		failLoad("relations.%s.%s must be an object", relName, side)
	}
	for k := range sm {
		if k != "path" && k != fieldKey {
			failLoad("relations.%s.%s: unknown key %q", relName, side, k)
		}
	}
	p, has := sm["path"]
	if !has {
		failLoad("relations.%s.%s: needs a path", relName, side)
	}
	path, ok = p.([]any)
	if !ok {
		failLoad("relations.%s.%s: path must be an array", relName, side)
	}
	if e := jaxson.CheckPath(path); e != nil {
		failLoad("relations.%s.%s.path: %s", relName, side, e.Msg)
	}
	fv, has := sm[fieldKey]
	if !has {
		failLoad("relations.%s.%s: needs a %s", relName, side, fieldKey)
	}
	field, ok = fv.(string)
	if !ok || field == "" {
		failLoad("relations.%s.%s.%s must be a plain field name", relName, side, fieldKey)
	}
	return path, field
}

func pathsEqual(a, b []any) bool {
	return jaxson.Equal(any(a), any(b))
}
