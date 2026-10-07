// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

func TestNumberParseAndString(t *testing.T) {
	for in, want := range map[string]string{
		"0": "0", "-0": "0", "5": "5", "-0.25": "-0.25", "1.50": "1.5", "1e3": "1000",
		"12.5E-1": "1.25", "100": "100", "0.0001": "0.0001",
	} {
		n, err := ParseNumber(in)
		if err != nil {
			t.Errorf("ParseNumber(%q): %v", in, err)
			continue
		}
		if n.String() != want {
			t.Errorf("ParseNumber(%q).String() = %q, want %q", in, n.String(), want)
		}
	}
	for _, bad := range []string{"", "+1", "01", "1.", ".5", "0x10", "NaN", "1e", " 1", "1 ", "1/3",
		strings.Repeat("9", 39), "1e400"} {
		if _, err := ParseNumber(bad); err == nil {
			t.Errorf("ParseNumber(%q) accepted", bad)
		}
	}
	var zero Number
	if zero.String() != "0" || zero.Sign() != 0 {
		t.Error("the zero Number is not 0")
	}
}

func TestNumberIsImmutable(t *testing.T) {
	src := big.NewRat(3, 2)
	n, ok := NumberFromRat(src)
	if !ok {
		t.Fatal("1.5 refused")
	}
	src.SetInt64(99) // the caller changes the Rat it passed in
	if n.String() != "1.5" {
		t.Fatalf("Number followed its source: %s", n)
	}
	n.Rat().SetInt64(7) // and the one it got out
	if n.String() != "1.5" {
		t.Fatalf("Number followed its copy: %s", n)
	}
}

func TestNumberConversions(t *testing.T) {
	if _, ok := NumberFromRat(big.NewRat(1, 3)); ok {
		t.Error("1/3 is not a decimal but was accepted")
	}
	if n, ok := AsNumber(big.NewRat(5, 2)); !ok || n.String() != "2.5" {
		t.Errorf("AsNumber(*big.Rat) = %v, %v", n, ok)
	}
	if _, ok := AsNumber("5"); ok {
		t.Error("AsNumber accepted a string")
	}
	n := NumberFromInt64(-42)
	if i, ok := n.Int64(); !ok || i != -42 || !n.IsInt() || n.Sign() != -1 {
		t.Error("Int64 round trip")
	}
	if h, _ := ParseNumber("0.5"); h.IsInt() {
		t.Error("0.5 is not an integer")
	} else if _, ok := h.Int64(); ok {
		t.Error("0.5 gave an int64")
	}
	big1, _ := ParseNumber("9223372036854775808") // 2^63
	if _, ok := big1.Int64(); ok {
		t.Error("2^63 fitted an int64")
	}
	if f, exact := mustNum(t, "0.5").Float64(); f != 0.5 || !exact {
		t.Error("0.5 is exact in float64")
	}
	if _, exact := mustNum(t, "0.1").Float64(); exact {
		t.Error("0.1 is not exact in float64")
	}
	if a, b := mustNum(t, "1.0"), mustNum(t, "1"); a.Cmp(b) != 0 || a != b && a.String() != b.String() {
		t.Error("1.0 and 1 differ")
	}
	if mustNum(t, "2").Cmp(mustNum(t, "10")) >= 0 {
		t.Error("2 should be less than 10, not compared as text")
	}
}

func TestNumberJSON(t *testing.T) {
	out, err := json.Marshal(map[string]any{"n": mustNum(t, "0.30"), "big": mustNum(t, "123456789012345678901234567890.5")})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"big":123456789012345678901234567890.5,"n":0.3}` {
		t.Errorf("got %s", out)
	}
}

func mustNum(t *testing.T, s string) Number {
	t.Helper()
	n, err := ParseNumber(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
