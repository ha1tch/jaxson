// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Numbers handed to the runtime as Go values (Jaxson core design, section 2).

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"
)

func TestDataConvertsHostNumbers(t *testing.T) {
	in := map[string]any{
		"a": 1, "b": 2.5, "c": uint64(18446744073709551615), "d": float32(0.1),
		"e": json.Number("1.5e3"), "f": NumberFromInt64(7), "g": int8(-3), "h": []any{1, 2.25},
	}
	want := `{"a":1,"b":2.5,"c":18446744073709551615,"d":0.1,"e":1500,"f":7,"g":-3,"h":[1,2.25]}`
	if got := Show(Data(in)); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
	if in["a"] != 1 {
		t.Error("Data changed its argument")
	}
}

func TestDataRejectsHostNonReals(t *testing.T) {
	bigRat := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(999999), nil))
	binaryFraction := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 100)) // 5^100 digits
	for name, v := range map[string]any{
		"float64 1e300": 1e300, "float64 +Inf": math.Inf(1), "float64 NaN": math.NaN(),
		"json.Number 1e999999": json.Number("1e999999"), "Rat 10^999999": bigRat,
		"Rat 1/2^100":      binaryFraction,
		"Rat of 39 digits": new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(38), nil)),
	} {
		err := recovered(func() { Data(map[string]any{"n": v}) })
		if err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != "NUMERIC_OVERFLOW" {
			t.Errorf("%s: got %v, want EXECUTION_ERROR/NUMERIC_OVERFLOW", name, err)
		}
	}
	// the largest Real is accepted
	max := new(big.Rat).SetInt(new(big.Int).Sub(new(big.Int).Exp(big.NewInt(10), big.NewInt(38), nil), big.NewInt(1)))
	if err := recovered(func() { Data(map[string]any{"n": max}) }); err != nil {
		t.Errorf("38 nines: %v", err)
	}
}

func TestRunAcceptsHostNumbersInInput(t *testing.T) {
	pkg := map[string]any{"jaxson": "1.0",
		"inputSchema": map[string]any{"type": "any"}, "outputSchema": map[string]any{"type": "any"},
		"input": map[string]any{"a": 2, "b": 3.5},
		"program": []any{map[string]any{"op": "set", "path": []any{"output"}, "value": map[string]any{"$compute": map[string]any{
			"with": map[string]any{"x": map[string]any{"$path": []any{"input", "a"}}, "y": map[string]any{"$path": []any{"input", "b"}}},
			"expr": []any{"add", map[string]any{"$v": "x"}, map[string]any{"$v": "y"}}}}}}}
	out, err := Run(pkg)
	if err != nil || Show(out) != "5.5" {
		t.Fatalf("got %s, %v", Show(out), err)
	}
	pkg["input"] = map[string]any{"a": 2, "b": math.Inf(1)}
	if _, err := Run(pkg); err == nil || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("an infinite input gave %v, want NUMERIC_OVERFLOW", err)
	}
}
