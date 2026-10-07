// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// A concat-keyed index is built into a byte slab (slabindex.go). The tree
// interpreter never takes that path, so it is the oracle: whatever the slab
// build answers, fails with and charges must be what the generic build does.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const pkgSlab = `{
  "indices": {
    "pair":  {"source": {"$path": ["state", "ev"]}, "key": {"$compute": {"with": {"a": {"$path": ["local", "item", "a"]}, "c": {"$path": ["local", "item", "c"]}}, "expr": ["concat", {"$v": "a"}, "|", {"$v": "c"}]}}},
    "pairs": {"source": {"$path": ["state", "ev"]}, "key": {"$compute": {"with": {"a": {"$path": ["local", "item", "a"]}, "c": {"$path": ["local", "item", "c"]}}, "expr": ["concat", {"$v": "a"}, "|", {"$v": "c"}]}}, "multi": true},
    "three": {"source": {"$path": ["state", "ev"]}, "key": {"$compute": {"with": {"a": {"$path": ["local", "item", "a"]}}, "expr": ["concat", "<", {"$v": "a"}, ">", {"$v": "a"}, "!"]}}, "multi": true}
  },
  "shapes": {"Any": {"kind": "any"}}
}`

type slabOutcome struct {
	found []bool
	elems [][]any
	inv   [][]any
	err   string
	steps int64
}

func slabRun(t *testing.T, index, state string, tree bool) (o slabOutcome) {
	t.Helper()
	jaxson.UseTreeInterpreter(tree)
	defer jaxson.UseTreeInterpreter(false)
	s, m := newSet(t, pkgSlab, state)
	tgt := ReferenceTarget{Index: index}
	if err := raise(func() {
		for _, k := range []string{"a|x", "b|y", "", "q|z", "<a>"} {
			o.found = append(o.found, s.Lookup(tgt, k))
		}
		o.found = append(o.found, s.Lookup(tgt, val(t, "1")), s.Lookup(tgt, nil))
		o.elems = s.Elements(index)
		o.inv = s.Inverse(tgt, "a|x")
	}); err != nil {
		o.err = fmt.Sprint(err)
	}
	o.steps = m.Steps()
	return o
}

func TestSlabIndexAgreesWithTheGenericBuild(t *testing.T) {
	states := map[string]string{
		"empty":        `{"ev": []}`,
		"distinct":     `{"ev": [{"a":"a","c":"x"},{"a":"b","c":"y"},{"a":"q","c":"z"}]}`,
		"repeats":      `{"ev": [{"a":"a","c":"x"},{"a":"b","c":"y"},{"a":"a","c":"x"},{"a":"q","c":"z"},{"a":"a","c":"x"}]}`,
		"empty parts":  `{"ev": [{"a":"","c":""},{"a":"a","c":"x"}]}`,
		"empty key":    `{"ev": [{"a":"","c":""}]}`,
		"unicode":      `{"ev": [{"a":"é","c":"日本"},{"a":"é","c":"日本"},{"a":"b","c":"y"}]}`,
		"number part":  `{"ev": [{"a":"a","c":"x"},{"a":"b","c":1}]}`,
		"null part":    `{"ev": [{"a":"a","c":"x"},{"a":null,"c":"y"}]}`,
		"missing part": `{"ev": [{"a":"a","c":"x"},{"a":"b"}]}`,
		"object part":  `{"ev": [{"a":"a","c":"x"},{"a":"b","c":{"k":1}}]}`,
		"not an array": `{"ev": {"a":"a"}}`,
	}
	for name, st := range states {
		for _, ix := range []string{"pair", "pairs", "three"} {
			got, want := slabRun(t, ix, st, false), slabRun(t, ix, st, true)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s / %s:\n slab:    %+v\n generic: %+v", name, ix, got, want)
			}
		}
	}
}

// A large index with many repeats and growth of the table, against the
// generic build, including its step count and the order of positions.
func TestSlabIndexAtScale(t *testing.T) {
	var ev []byte
	ev = append(ev, `{"ev": [`...)
	for i := 0; i < 5000; i++ {
		if i > 0 {
			ev = append(ev, ',')
		}
		ev = append(ev, fmt.Sprintf(`{"a":"u%d","c":"c%d"}`, (i*7)%97, (i*13)%251)...)
	}
	ev = append(ev, `]}`...)
	got, want := slabRun(t, "pairs", string(ev), false), slabRun(t, "pairs", string(ev), true)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("slab and generic builds differ at scale (steps %d vs %d)", got.steps, want.steps)
	}
	if got.steps != 5000 && got.err == "" {
		t.Logf("steps %d", got.steps)
	}
}

// The differential tests above prove nothing if both sides take the generic
// build, so check that the compiled machine really builds into a slab, and
// that a key of another shape does not.
func TestSlabPathIsTaken(t *testing.T) {
	st := `{"ev": [{"a":"a","c":"x"}], "rate": 1}`
	s, _ := newSet(t, pkgSlab, st)
	if s.ensure(ReferenceTarget{Index: "pair"}).primary.tab == nil {
		t.Error("a concat key was not built into a slab")
	}
	s, _ = newSet(t, pkgIdx, stateIdx)
	if s.ensure(ReferenceTarget{Index: "byRate"}).primary.tab != nil {
		t.Error("an arithmetic key was built into a slab")
	}
}
