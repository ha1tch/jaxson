// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import (
	"fmt"
	"math/big"
	"sort"
	"testing"
)

func TestObjectSetGetDelete(t *testing.T) {
	// Sizes straddle the inline store (4) and the index threshold (12).
	for _, n := range []int{0, 1, 3, 4, 5, 11, 12, 13, 40, 300} {
		o := NewObject(0)
		ref := map[string]any{}
		for i := 0; i < n; i++ {
			k := fmt.Sprintf("k%d", i)
			o.Set(k, big.NewRat(int64(i), 1))
			ref[k] = big.NewRat(int64(i), 1)
		}
		if n > 0 {
			o.Set("k0", "again") // overwrite does not add
			ref["k0"] = "again"
		}
		if o.Len() != len(ref) {
			t.Fatalf("n=%d: Len %d want %d", n, o.Len(), len(ref))
		}
		for k, want := range ref {
			got, ok := o.Get(k)
			if !ok || !Equal(got, want) {
				t.Fatalf("n=%d: Get(%q) = %v, %v", n, k, got, ok)
			}
			if g2, ok := o.GetKey(MakeKey(k)); !ok || !Equal(g2, want) {
				t.Fatalf("n=%d: GetKey(%q) = %v, %v", n, k, g2, ok)
			}
		}
		if o.Has("absent") {
			t.Fatalf("n=%d: Has(absent)", n)
		}
		want := make([]string, 0, len(ref))
		for k := range ref {
			want = append(want, k)
		}
		sort.Strings(want)
		if got := o.SortedKeys(); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("n=%d: SortedKeys %v want %v", n, got, want)
		}
		// delete every other member, then check the rest are still found
		for i := 0; i < n; i += 2 {
			k := fmt.Sprintf("k%d", i)
			if !o.Delete(k) {
				t.Fatalf("n=%d: Delete(%q) false", n, k)
			}
			delete(ref, k)
			if o.Delete(k) {
				t.Fatalf("n=%d: second Delete(%q) true", n, k)
			}
		}
		if o.Len() != len(ref) {
			t.Fatalf("n=%d: after deletes Len %d want %d", n, o.Len(), len(ref))
		}
		for k, w := range ref {
			if g, ok := o.Get(k); !ok || !Equal(g, w) {
				t.Fatalf("n=%d: after deletes Get(%q) = %v, %v", n, k, g, ok)
			}
		}
		for i := 0; i < n; i += 2 {
			if o.Has(fmt.Sprintf("k%d", i)) {
				t.Fatalf("n=%d: deleted member still present", n)
			}
		}
	}
}

func TestObjectNilReadsAsEmpty(t *testing.T) {
	var o *Object
	if o.Len() != 0 || o.Has("a") {
		t.Fatal("nil Object is not empty")
	}
	if _, ok := o.Get("a"); ok {
		t.Fatal("nil Object has a member")
	}
}

func TestKeyIdentity(t *testing.T) {
	a, b := MakeKey("same"+fmt.Sprint(1)), MakeKey("same1")
	if a != b {
		t.Fatal("equal names gave different Keys")
	}
	if a.Value() != "same1" || MakeKey("other") == a {
		t.Fatal("Key Value or distinctness wrong")
	}
}

func TestObjectCloneIsDeep(t *testing.T) {
	v, err := ParseData([]byte(`{"a":{"b":[1,{"c":2}]},"d":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	c := Clone(v).(*Object)
	if !Equal(v, c) {
		t.Fatal("clone not equal")
	}
	inner, _ := c.Get("a")
	inner.(*Object).Set("b", "changed")
	orig, _ := v.(*Object).Get("a")
	if got, _ := orig.(*Object).Get("b"); Show(got) == `"changed"` {
		t.Fatal("clone shares an inner object with the original")
	}
	if Equal(v, c) {
		t.Fatal("change to the clone should show")
	}
}

func TestObjectEqualAcrossRepresentations(t *testing.T) {
	m := map[string]any{"a": big.NewRat(1, 1), "b": []any{map[string]any{"c": "x"}}}
	o := Data(m)
	if _, isObj := o.(*Object); !isObj {
		t.Fatalf("Data gave %T", o)
	}
	if !Equal(o, m) || !Equal(m, o) || !Equal(o, Data(m)) {
		t.Fatal("map and Object forms of one value compare unequal")
	}
	m2 := map[string]any{"a": big.NewRat(1, 1), "b": []any{map[string]any{"c": "y"}}}
	if Equal(o, m2) || Equal(m2, o) {
		t.Fatal("different values compare equal")
	}
	if TypeName(o) != "object" || TypeName(m) != "object" {
		t.Fatal("TypeName of an object")
	}
	if Show(o) != Show(m) {
		t.Fatalf("Show differs: %s vs %s", Show(o), Show(m))
	}
}

func TestDataLegacyRoundTrip(t *testing.T) {
	m := map[string]any{"n": nil, "t": true, "arr": []any{big.NewRat(1, 3), map[string]any{}}, "o": map[string]any{"x": "y"}}
	back := Legacy(Data(m))
	if !sameValue(back, m) {
		t.Fatal("round trip changed the value")
	}
	if _, isMap := back.(map[string]any)["o"].(map[string]any); !isMap {
		t.Fatal("Legacy left an Object")
	}
	// a value with no map in it is returned as it is, not copied
	arr := []any{"a", big.NewRat(2, 1)}
	if d := Data(arr).([]any); &d[0] != &arr[0] {
		t.Fatal("Data copied an array with no object in it")
	}
	// the argument is not changed
	if _, isMap := m["o"].(map[string]any); !isMap {
		t.Fatal("Data changed its argument")
	}
}

func TestParseDataRejectsDuplicateKeys(t *testing.T) {
	for _, doc := range []string{`{"a":1,"a":2}`, `{"x":{"k":1,"k":1}}`, `{"a":[{"b":1,"b":2}]}`} {
		if _, err := ParseData([]byte(doc)); err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("ParseData(%s) = %v", doc, err)
		}
		if _, err := ParsePackage([]byte(`{"input":` + doc + `}`)); err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("ParsePackage(input %s) = %v", doc, err)
		}
	}
}

func TestParsePackageOnlyConvertsInput(t *testing.T) {
	v, err := ParsePackage([]byte(`{"input":{"a":{"b":1}},"program":[{"op":"set"}],"input2":{"x":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if _, ok := m["input"].(*Object); !ok {
		t.Fatalf("input is %T", m["input"])
	}
	if _, ok := m["program"].([]any)[0].(map[string]any); !ok {
		t.Fatal("program was converted")
	}
	if _, ok := m["input2"].(map[string]any); !ok {
		t.Fatal("a member merely named like input was converted")
	}
	// "input" nested deeper is not the package's input
	v, err = ParsePackage([]byte(`{"x":{"input":{"a":1}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(map[string]any)["x"].(map[string]any)["input"].(map[string]any); !ok {
		t.Fatal("nested input member converted")
	}
}

func BenchmarkObjectGetFixed(b *testing.B) {
	v, _ := ParseData([]byte(`{"t":1,"actor":"u1","mb":0.5,"x":2}`))
	o := v.(*Object)
	k := MakeKey("mb")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := o.GetKey(k); !ok {
			b.Fatal("missing")
		}
	}
}

func BenchmarkObjectSetDynamic(b *testing.B) {
	names := make([]string, 256)
	for i := range names {
		names[i] = fmt.Sprintf("actor-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o := NewObject(0)
		for _, n := range names {
			o.Set(n, true)
		}
	}
}
