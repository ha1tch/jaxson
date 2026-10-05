// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.1 (plan section 7): index build, reuse, and the lookups that
// consume them. Core sections 3, 4a, 4c and 5.
//
// An IndexSet belongs to one run. It owns every built index of that run,
// sees every mutation the Machine performs (it installs itself on
// Machine.OnMutate, chaining whatever was there), and rebuilds an index
// only when a mutation since its last build overlaps something the build
// depends on. That is the mandatory reuse rule of section 3, and it is the
// only thing that decides whether a build is charged.
//
// ====================================================================
// Decisions where the spec is silent or ambiguous (P1..P6)
// ====================================================================
//
//	P1  Reuse is invalidated by OVERLAP, not by the literal wording. Core
//	    section 3 says a build is reused "whenever no mutation since that
//	    build targeted a path equal to, or a prefix of, its declared
//	    source". Read literally, a `set` on state.customers[3].id — a path
//	    the source state.customers is a prefix OF, not the other way round —
//	    would leave the index stale, which contradicts the same section's
//	    "always validated against current data, never a stale snapshot".
//	    Here a mutation invalidates a build when either path is a prefix of
//	    the other. Whenever the literal rule reuses, this rule reuses too;
//	    it recharges only in the cases where the literal rule would serve
//	    wrong data. NEEDS A SPEC RULING: it changes step counts relative to
//	    a literal reading.
//	P2  Dependencies are the source path AND every non-local path the
//	    source and key operands read (a key such as
//	    {"$path": ["state", "rate"]} depends on state.rate). They are found
//	    by static scan of the operands; a path with a computed segment is
//	    truncated at that segment, so it is treated as the whole subtree.
//	P3  A relation is one unit: its `to` index, its `from` index and, for
//	    cardinality one-to-one, the uniqueness check on from.field are built
//	    together whichever direction is used, and charged for both sides.
//	    Section 4a says the one-to-one check is made "at the same build
//	    step" as the to.key check, which requires reading the from side.
//	P4  Index keys are strings, numbers, booleans and null. They are ordered
//	    null < false < true < numbers (numerically) < strings (by code
//	    point). Section 3 names only the sorted-key convention; mixed-type
//	    keys are not otherwise ordered.
//	P5  An index is "addressable" when its source is a `$path` whose
//	    segments resolve to literals or to operands that evaluate to
//	    strings or non-negative integers: only then does each source element
//	    have a path, which `$indexed` and `$inverse` need. A source that is
//	    any other operand (a `$compute`, say) can still serve `reference`
//	    but is a SHAPE_ERROR when `$indexed`/`$inverse` asks for its paths.
//	P6  Building is charged one `index.element` per element of `source`,
//	    before the key is evaluated (charge before effect). A duplicate key
//	    found part-way has already been charged for the elements before it.

import (
	"fmt"
	"math/big"
	"slices"
	"strconv"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// EventIndexElement is the cost-table event for visiting one source element
// while building an index (core section 3, "Cost").
const EventIndexElement = "index.element"

type mutation struct {
	root string
	segs []any
}

// indexSpec is the normalised form of anything that builds an index: a
// named `indices` entry, a relation side, or an inline reference shorthand.
type indexSpec struct {
	label  string // for messages, e.g. `index "customersById"`
	source any
	key    any
	multi  bool
}

type indexEntry struct {
	key  any     // the scalar key, as evaluated
	pos  []int   // positions in source, ascending
	flag flagKey // canonical form, the map key
	one  [1]int  // backing store for pos while the key has a single element
}

// flagKey is a scalar key's canonical form: its kind and, for strings and
// numbers, its text. A struct rather than a prefixed string so that looking up
// or storing a string key does not build a new string.
type flagKey struct {
	kind byte
	text string
}

// indexData is one built index.
type indexData struct {
	spec    indexSpec
	byKey   map[flagKey]*indexEntry
	ordered []*indexEntry // ascending key order (P4)
	srcPath []any         // root + segments of the source; nil if not addressable (P5)
}

// unit is what the set builds, reuses and invalidates as one: an index, or
// a relation's two sides (P3).
type unit struct {
	primary *indexData // the index; for a relation, its `to` side
	inverse *indexData // a relation's `from` side; nil otherwise
	deps    [][]any    // each: root followed by a literal prefix (P2)
	logPos  int        // how much of the mutation log has been checked
	built   bool
	working bool // being built now: a key that asks for its own index is a cycle (F7)
}

// IndexSet builds and serves the indices of one run.
type IndexSet struct {
	m     *jaxson.Machine
	reg   *Registries
	log   []mutation
	units map[string]*unit

	ambient *Ambient // publishes the element path to closure forms in a key (forms.go, F3)
}

// NewIndexSet returns an IndexSet over m and reg and attaches it to m's
// mutation hook. Create it before anything runs and keep it for the whole
// run, so that it sees every mutation and builds are reused across
// `validate` entries and `check` instructions.
func NewIndexSet(m *jaxson.Machine, reg *Registries) *IndexSet {
	s := &IndexSet{m: m, reg: reg, units: map[string]*unit{}}
	prev := m.OnMutate
	m.OnMutate = func(root string, segs []any) {
		s.log = append(s.log, mutation{root: root, segs: append([]any(nil), segs...)})
		if prev != nil {
			prev(root, segs)
		}
	}
	return s
}

// ---------------------------------------------------------------- identity

func refID(t ReferenceTarget) string {
	switch {
	case t.Index != "":
		return "i:" + t.Index
	case t.Relation != "":
		return "r:" + t.Relation
	}
	return "s:" + jaxson.Show(t.Of[0]) + "|" + jaxson.Show(any(t.By))
}

// specsFor returns the unit's specs: one for an index or shorthand, two
// (to, from) for a relation.
func (s *IndexSet) specsFor(t ReferenceTarget) (primary indexSpec, inverse *indexSpec, oneToOne bool) {
	switch {
	case t.Index != "":
		d := s.reg.Indices[t.Index]
		return indexSpec{label: fmt.Sprintf("index %q", d.Name), source: d.Source, key: d.Key, multi: d.Multi}, nil, false
	case t.Relation != "":
		r := s.reg.Relations[t.Relation]
		item := func(f string) any { return map[string]any{"$path": []any{"local", "item", f}} }
		primary = indexSpec{label: fmt.Sprintf("relation %q (to side)", r.Name),
			source: map[string]any{"$path": r.ToPath}, key: item(r.ToKey)}
		inv := indexSpec{label: fmt.Sprintf("relation %q (from side)", r.Name),
			source: map[string]any{"$path": r.FromPath}, key: item(r.FromField), multi: true}
		return primary, &inv, r.Cardinality == "one-to-one"
	}
	by := []any{"local", "item"}
	by = append(by, t.By...)
	return indexSpec{label: "inline reference shorthand", source: t.Of[0],
		key: map[string]any{"$path": by}}, nil, false
}

// ---------------------------------------------------------------- freshness

// ensure returns the unit for t, building it if it has never been built and
// rebuilding it if a mutation since the last build overlaps its
// dependencies (P1). It charges only when it builds.
func (s *IndexSet) ensure(t ReferenceTarget) *unit {
	id := refID(t)
	u, ok := s.units[id]
	if !ok {
		u = &unit{}
		s.units[id] = u
	}
	if u.built && !s.stale(u) {
		return u
	}
	if u.working {
		jaxson.Fail(CatShapeError, "", "an index whose key asks for its own $inverse (or one that leads back to it) cannot be built (F7)")
	}
	u.working = true
	defer func() { u.working = false }()
	primary, inverse, oneToOne := s.specsFor(t)
	u.primary = s.build(primary)
	u.inverse = nil
	u.deps = depsOf(primary)
	if inverse != nil {
		u.inverse = s.build(*inverse)
		u.deps = append(u.deps, depsOf(*inverse)...)
		if oneToOne {
			s.requireUniqueEntries(u.inverse, "cardinality one-to-one requires from.field to be unique")
		}
	}
	u.logPos = len(s.log)
	u.built = true
	return u
}

// stale reports whether a mutation after the unit's last check overlaps one
// of its dependencies, and advances the checked position when none does.
func (s *IndexSet) stale(u *unit) bool {
	for _, mu := range s.log[u.logPos:] {
		for _, d := range u.deps {
			if overlaps(mu.root, mu.segs, d) {
				return true
			}
		}
	}
	u.logPos = len(s.log)
	return false
}

// overlaps: do the mutated path and a dependency (root + literal prefix)
// share a prefix — one is equal to, or an ancestor of, the other?
func overlaps(root string, segs []any, dep []any) bool {
	if dep[0].(string) != root {
		return false
	}
	d := dep[1:]
	n := len(segs)
	if len(d) < n {
		n = len(d)
	}
	for i := 0; i < n; i++ {
		if fmt.Sprint(segs[i]) != fmt.Sprint(d[i]) {
			return false
		}
	}
	return true
}

// depsOf lists what a build reads: the source's own path and every
// non-local path in the source and key operands (P2).
func depsOf(sp indexSpec) [][]any {
	var out [][]any
	collectReads(sp.source, &out)
	collectReads(sp.key, &out)
	return out
}

func collectReads(x any, out *[][]any) {
	switch t := x.(type) {
	case map[string]any:
		if p, ok := t["$path"].([]any); ok && len(t) == 1 {
			if root, _ := p[0].(string); root != "" && root != "local" {
				dep := []any{root}
				for _, seg := range p[1:] {
					if _, computed := seg.(map[string]any); computed {
						break
					}
					dep = append(dep, seg)
				}
				*out = append(*out, normaliseSegs(dep))
			}
			return
		}
		if _, lit := t["$lit"]; lit {
			return
		}
		for _, k := range jaxson.SortedKeys(t) {
			collectReads(t[k], out)
		}
	case []any:
		for _, e := range t {
			collectReads(e, out)
		}
	}
}

// normaliseSegs turns decimal segments into ints so that dependency and
// mutation paths compare uniformly.
func normaliseSegs(p []any) []any {
	out := make([]any, len(p))
	for i, v := range p {
		if r, ok := v.(*big.Rat); ok && r.IsInt() {
			out[i] = int(r.Num().Int64())
		} else {
			out[i] = v
		}
	}
	return out
}

// ---------------------------------------------------------------- building

func (s *IndexSet) build(sp indexSpec) *indexData {
	srcPath, arr := s.resolveSource(sp)
	d := &indexData{spec: sp, byKey: map[flagKey]*indexEntry{}, srcPath: srcPath}
	for i, el := range arr {
		s.m.ChargeEvent(EventIndexElement, 0)
		var key any
		s.ambient.withElem(srcPath, i, func() {
			s.m.WithLocal("item", el, func() { key = s.m.Eval(sp.key) })
		})
		flag, ok := keyFlag(key)
		if !ok {
			jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: the key of element %d must be a scalar, got %s", sp.label, i, jaxson.TypeName(key))
		}
		if e, seen := d.byKey[flag]; seen {
			if !sp.multi {
				jaxson.Fail(CatShapeError, "", "%s: key %s repeats (elements %d and %d); an index is a function unless it declares multi", sp.label, jaxson.Show(key), e.pos[0], i)
			}
			e.pos = append(e.pos, i)
			continue
		}
		e := &indexEntry{key: key, flag: flag}
		e.one[0] = i
		e.pos = e.one[:1]
		d.byKey[flag] = e
		d.ordered = append(d.ordered, e)
	}
	// Keys are distinct (equal keys share an entry), so no two entries
	// compare equal and an unstable sort gives the one order.
	slices.SortFunc(d.ordered, func(a, b *indexEntry) int { return compareKeys(a.key, b.key) })
	return d
}

// resolveSource evaluates a spec's source to an array. When the source is
// a `$path`, it also returns the path (root first), which makes the index
// addressable (P5); otherwise the path is nil.
func (s *IndexSet) resolveSource(sp indexSpec) (path []any, arr []any) {
	if mp, ok := sp.source.(map[string]any); ok && len(mp) == 1 {
		if p, ok := mp["$path"].([]any); ok {
			root, segs := evalPath(s.m, sp.label, p)
			v, err := s.m.Walk(root, segs)
			if err != nil {
				panic(err)
			}
			a, ok := v.([]any)
			if !ok {
				jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: source must be an array, got %s", sp.label, jaxson.TypeName(v))
			}
			return append([]any{root}, segs...), a
		}
	}
	v := s.m.Eval(sp.source)
	a, ok := v.([]any)
	if !ok {
		jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: source must be an array, got %s", sp.label, jaxson.TypeName(v))
	}
	return nil, a
}

// requireUniqueEntries raises SHAPE_ERROR if any key of d has more than one
// element (the one-to-one check, P3).
func (s *IndexSet) requireUniqueEntries(d *indexData, why string) {
	for _, e := range d.ordered {
		if len(e.pos) > 1 {
			jaxson.Fail(CatShapeError, "", "%s: %s; key %s is shared by elements %d and %d", d.spec.label, why, jaxson.Show(e.key), e.pos[0], e.pos[1])
		}
	}
}

// ---------------------------------------------------------------- keys

// keyFlag is a key's canonical string, or false if it is not a scalar.
func keyFlag(v any) (flagKey, bool) {
	switch t := v.(type) {
	case string:
		return flagKey{'s', t}, true
	case *big.Rat:
		if t.IsInt() && t.Num().IsInt64() {
			return flagKey{'n', strconv.FormatInt(t.Num().Int64(), 10)}, true
		}
		return flagKey{'n', t.RatString()}, true
	case bool:
		if t {
			return flagKey{'t', ""}, true
		}
		return flagKey{'f', ""}, true
	case nil:
		return flagKey{'z', ""}, true
	}
	return flagKey{}, false
}

func keyRank(v any) int {
	switch v.(type) {
	case nil:
		return 0
	case bool:
		return 1
	case *big.Rat:
		return 2
	}
	return 3
}

// compareKeys is the P4 order.
func compareKeys(a, b any) int {
	ra, rb := keyRank(a), keyRank(b)
	if ra != rb {
		return ra - rb
	}
	switch x := a.(type) {
	case bool:
		if x == b.(bool) {
			return 0
		}
		if !x {
			return -1
		}
		return 1
	case *big.Rat:
		return x.Cmp(b.(*big.Rat))
	case string:
		return jaxson.Order(x, b.(string))
	}
	return 0
}

// ---------------------------------------------------------------- lookups

// Lookup reports whether key is present in the target index (or in a
// relation's `to` side). It implements IndexResolver. The build, if one is
// due, is charged here.
func (s *IndexSet) Lookup(t ReferenceTarget, key any) bool {
	flag, ok := keyFlag(key)
	if !ok {
		jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "a reference must be a scalar, got %s", jaxson.TypeName(key))
	}
	_, found := s.ensure(t).primary.byKey[flag]
	return found
}

// Refresh builds or rebuilds the target if it is due, without looking
// anything up. The `validate` pipeline calls it for every index an entry
// uses before the entry resolves its targets, which is where section 3 puts
// the charge.
func (s *IndexSet) Refresh(t ReferenceTarget) { s.ensure(t) }

// Elements returns the path of every source element of the named index, in
// the order `$indexed` visits them: ascending key order, and within a key
// by position (core section 7).
func (s *IndexSet) Elements(name string) [][]any {
	d := s.ensure(ReferenceTarget{Index: name}).primary
	return s.pathsOf(d, nil)
}

// Inverse returns the path of every source element the multi index (or a
// relation's `from` side) maps key to, in array order (core section 6).
func (s *IndexSet) Inverse(t ReferenceTarget, key any) [][]any {
	flag, ok := keyFlag(key)
	if !ok {
		jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "an $inverse key must be a scalar, got %s", jaxson.TypeName(key))
	}
	u := s.ensure(t)
	d := u.inverse
	if d == nil {
		d = u.primary
		if !d.spec.multi {
			jaxson.Fail(CatShapeError, "", "%s: $inverse needs a multi index", d.spec.label)
		}
	}
	e, found := d.byKey[flag]
	if !found {
		return nil
	}
	return s.pathsOf(d, e)
}

// pathsOf lists element paths of d: of every entry in key order when only
// is nil, else of that one entry.
func (s *IndexSet) pathsOf(d *indexData, only *indexEntry) [][]any {
	if d.srcPath == nil {
		jaxson.Fail(CatShapeError, "", "%s: its source is not a $path, so its elements have no paths", d.spec.label)
	}
	entries := d.ordered
	if only != nil {
		entries = []*indexEntry{only}
	}
	var out [][]any
	for _, e := range entries {
		for _, p := range e.pos {
			out = append(out, append(append([]any(nil), d.srcPath...), p))
		}
	}
	return out
}
