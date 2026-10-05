// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// append: a benchmark showing it is linear (it was quadratic, copying the whole
// array each time) and the test that pins why growing in place is safe.

import (
	"math/big"
	"testing"
)

func appendProgram(n int) map[string]any {
	in := make([]any, n)
	for i := range in {
		in[i] = big.NewRat(int64(i), 1)
	}
	return map[string]any{
		"jaxson": "1.0", "input": in,
		"inputSchema": map[string]any{"type": "any"}, "outputSchema": map[string]any{"type": "any"},
		"limits": map[string]any{"steps": big.NewRat(100000000, 1)},
		"program": []any{
			map[string]any{"op": "set", "path": []any{"state", "l"}, "value": map[string]any{"$lit": []any{}}},
			map[string]any{"op": "for", "in": map[string]any{"$path": []any{"input"}}, "as": "e", "do": []any{
				map[string]any{"op": "append", "path": []any{"state", "l"}, "value": map[string]any{"$path": []any{"local", "e"}}},
			}},
			map[string]any{"op": "set", "path": []any{"output"}, "value": map[string]any{"$compute": map[string]any{"with": map[string]any{"l": map[string]any{"$path": []any{"state", "l"}}}, "expr": []any{"len", map[string]any{"$v": "l"}}}}},
		},
	}
}

func benchAppend(b *testing.B, n int) {
	p := appendProgram(n)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, e := Run(p); e != nil {
			b.Fatal(e)
		}
	}
}
func BenchmarkAppend5000(b *testing.B)  { benchAppend(b, 5000) }
func BenchmarkAppend20000(b *testing.B) { benchAppend(b, 20000) }

// TestAppendDoesNotAlias pins the property that lets append grow arrays in
// place: after an array has grown (so it has spare capacity), a copy of it
// made by a read, and a loop's snapshot of it, are not affected by later
// appends to the original, and appends to the copy do not show in the
// original.
func TestAppendDoesNotAlias(t *testing.T) {
	st := func(path ...string) map[string]any {
		p := []any{"state"}
		for _, s := range path {
			p = append(p, s)
		}
		return map[string]any{"$path": p}
	}
	app := func(to string, v any) any {
		return map[string]any{"op": "append", "path": []any{"state", to}, "value": v}
	}
	rt := func(n int64) any { return big.NewRat(n, 1) }
	p := map[string]any{
		"jaxson": "1.0", "input": nil,
		"inputSchema": map[string]any{"type": "any"}, "outputSchema": map[string]any{"type": "any"},
		"program": []any{
			map[string]any{"op": "set", "path": []any{"state", "a"}, "value": map[string]any{"$lit": []any{}}},
			app("a", rt(1)), app("a", rt(2)), app("a", rt(3)), // cap is now 4: room for one more in place
			map[string]any{"op": "set", "path": []any{"state", "b"}, "value": st("a")}, // a copy, by a read
			app("a", rt(100)),
			app("b", rt(200)),
			map[string]any{"op": "set", "path": []any{"state", "snap"}, "value": map[string]any{"$lit": []any{}}},
			// A loop over a, appending to a while it runs, sees the 4 elements it started with.
			map[string]any{"op": "for", "in": st("a"), "as": "x", "do": []any{
				app("a", map[string]any{"$path": []any{"local", "x"}}),
				app("snap", map[string]any{"$path": []any{"local", "x"}}),
			}},
			map[string]any{"op": "set", "path": []any{"output"}, "value": map[string]any{"$path": []any{"state"}}},
		},
	}
	out, e := Run(p)
	if e != nil {
		t.Fatal(e)
	}
	want := `{"a":[1,2,3,100,1,2,3,100],"b":[1,2,3,200],"snap":[1,2,3,100]}`
	if got := show(out); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}
