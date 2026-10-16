// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Strict numeric mode (Jaxson core design, section 2): a JSON number that is
// not a Real is a PARSE_ERROR, and is rejected in time proportional to the
// length of its text. A Real has at most 38 digits once leading zeros are
// removed and at most 200 digits after the decimal point.

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

// isOutOfRange reports whether e is the range error, whose message begins
// "number out of range" and goes on to name the limit.
func isOutOfRange(e *Err) bool {
	return e != nil && strings.HasPrefix(e.Msg, "number out of range")
}

type realCase struct {
	lit  string
	real bool
}

func realCases() []realCase {
	return []realCase{
		{"0", true}, {"-0", true}, {"1", true}, {"0.5", true}, {"12.5E-1", true}, {"1.50", true},
		{strings.Repeat("9", 38), true},
		{"-" + strings.Repeat("9", 38), true},
		{strings.Repeat("9", 39), false},
		{"0." + strings.Repeat("9", 38), true},
		{"0." + strings.Repeat("9", 39), false},
		{"1e37", true}, {"1e38", false}, {"-1e38", false},
		{"123456789012345678901234567890e8", true},        // 38 digits
		{"123456789012345678901234567890e9", false},       // 39 digits
		{"1000000000000000000000000000000000000e0", true}, // 10^36
		{"100e-2", true}, // 1
		{"1e-200", true}, {"1e-201", false},
		{"0." + strings.Repeat("0", 199) + "1", true},  // 200 digits after the point
		{"0." + strings.Repeat("0", 200) + "1", false}, // 201
		{"1.5e-200", false}, {"15e-201", false},
		{"0e999999", true}, {"-0e5", true}, {"0.000e-999999", true}, // zero is a Real however it is written
		{"1e999999", false}, {"1e-999999", false}, {"1e1000001", false}, {"1e99999999999", false},
	}
}

func TestStrictRealBoundaries(t *testing.T) {
	for _, c := range realCases() {
		doc := []byte("[" + c.lit + "]")
		_, perr := ParseJSON(doc)
		_, derr := ParseData(doc)
		_, nerr := ParseNumber(c.lit)
		if (perr == nil) != c.real || (derr == nil) != c.real || (nerr == nil) != c.real {
			t.Errorf("%.60q: Real=%v, but ParseJSON err=%v, ParseData err=%v, ParseNumber err=%v", c.lit, c.real, perr, derr, nerr)
		}
		if !c.real && perr != nil && (perr.Cat != "PARSE_ERROR" || !isOutOfRange(perr)) {
			t.Errorf("%.60q: got %v, want PARSE_ERROR: number out of range", c.lit, perr)
		}
	}
}

// The cost of a literal is bounded by its length, however large its exponent
// or digit string. The bound here is generous; the cost measured on the
// prototype was a few milliseconds for each case.
func TestStrictRejectionCost(t *testing.T) {
	rep := func(s string, n int) string { return "[" + strings.TrimSuffix(strings.Repeat(s+",", n), ",") + "]" }
	for _, h := range []struct {
		name, doc string
		valid     bool
	}{
		{"100 x 1e999999", rep("1e999999", 100), false},
		{"100 x 1e-999999", rep("1e-999999", 100), false},
		{"one 1,000,000-digit integer", "[" + strings.Repeat("9", 1000000) + "]", false},
		{"one literal with 1,000,000 fraction digits", "[0." + strings.Repeat("1", 1000000) + "]", false},
		{"100 x 0e999999 (zero is valid)", rep("0e999999", 100), true},
	} {
		start := time.Now()
		_, err := ParseJSON([]byte(h.doc))
		el := time.Since(start)
		if h.valid && err != nil {
			t.Errorf("%s: %v", h.name, err)
		}
		if !h.valid && (err == nil || err.Cat != "PARSE_ERROR") {
			t.Errorf("%s: got %v, want PARSE_ERROR", h.name, err)
		}
		if el > time.Second {
			t.Errorf("%s took %v: the cost of a literal must be proportional to its text", h.name, el)
		}
	}
}

func recovered(f func()) (err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	f()
	return nil
}

func TestNormalizeRejectsNonReal(t *testing.T) {
	err := recovered(func() { Normalize(map[string]any{"n": json.Number("1e999999")}) })
	if err == nil || err.Cat != "PARSE_ERROR" {
		t.Errorf("Normalize of 1e999999 gave %v, want PARSE_ERROR", err)
	}
	if err := recovered(func() { Normalize(map[string]any{"n": json.Number("1.5e3")}) }); err != nil {
		t.Errorf("Normalize of 1.5e3 gave %v", err)
	}
}

func TestToNumberRejectsNonRealQuickly(t *testing.T) {
	pkg := `{"jaxson":"1.0","inputSchema":{"type":"any"},"outputSchema":{"type":"any"},"input":"` + strings.Repeat("9", 1000000) + `",
	 "program":[{"op":"set","path":["output"],"value":{"$compute":{"with":{"s":{"$path":["input"]}},"expr":["to_number",{"$v":"s"}]}}}]}`
	start := time.Now()
	_, err := RunJSON([]byte(pkg))
	if err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("to_number of a 1,000,000-digit string gave %v, want EXECUTION_ERROR/NUMERIC_OVERFLOW", err)
	}
	if el := time.Since(start); el > time.Second {
		t.Errorf("took %v", el)
	}
}

func TestHugeRatIsRejectedInConstantTime(t *testing.T) {
	huge := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(999999), nil))
	tiny := new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(999999), nil))
	for name, r := range map[string]*big.Rat{"10^999999": huge, "10^-999999": tiny} {
		start := time.Now()
		err := recovered(func() { checkNum(r) })
		if err == nil || err.Code != "NUMERIC_OVERFLOW" {
			t.Errorf("checkNum(%s) gave %v, want NUMERIC_OVERFLOW", name, err)
		}
		if _, ok := NumberFromRat(r); ok {
			t.Errorf("NumberFromRat accepted %s", name)
		}
		if el := time.Since(start); el > 100*time.Millisecond {
			t.Errorf("%s took %v: a Rat's size must be tested without building its digits", name, el)
		}
	}
}

func TestNumberFromNilRat(t *testing.T) {
	if _, ok := NumberFromRat(nil); ok {
		t.Error("NumberFromRat(nil) accepted")
	}
	if _, ok := AsNumber((*big.Rat)(nil)); ok {
		t.Error("AsNumber((*big.Rat)(nil)) accepted")
	}
}

// referenceHasNonReal reports whether the encoding/json reading of raw holds a
// number that is not a Real. It lets the differential tests accept the one
// disagreement strict mode introduces: the reference reads such a number, the
// scanner rejects it.
func referenceHasNonReal(raw []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return false
	}
	var walk func(any) bool
	walk = func(x any) bool {
		switch t := x.(type) {
		case json.Number:
			r, ok := new(big.Rat).SetString(t.String())
			if !ok {
				return true
			}
			c, _, ok := decParts(r)
			return !ok || len(new(big.Int).Abs(c).String()) > maxDigits
		case []any:
			for _, e := range t {
				if walk(e) {
					return true
				}
			}
		case map[string]any:
			for _, e := range t {
				if walk(e) {
					return true
				}
			}
		}
		return false
	}
	return walk(v)
}
