// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Numbers handed to the runtime as Go values (Jaxson core design, section 2):
// a host that passes numbers other than as JSON text has them converted under
// the same rule as text, and one that is not a Real is NUMERIC_OVERFLOW. A
// floating-point value is converted through its shortest decimal that reads
// back as the same float, so 0.1 is 0.1, not its binary expansion.

import (
	"encoding/json"
	"math"
	"math/big"
	"strconv"
)

// hostNumber converts a Go number type to the machine's number, a *big.Rat. It
// reports false for any other value.
func hostNumber(v any, mode NumericMode) (any, bool) {
	switch t := v.(type) {
	case Number:
		return t.Rat(), true
	case json.Number:
		return numFromText(t.String(), mode), true
	case int:
		return new(big.Rat).SetInt64(int64(t)), true
	case int8:
		return new(big.Rat).SetInt64(int64(t)), true
	case int16:
		return new(big.Rat).SetInt64(int64(t)), true
	case int32:
		return new(big.Rat).SetInt64(int64(t)), true
	case int64:
		return new(big.Rat).SetInt64(t), true
	case uint:
		return ratFromUint64(uint64(t)), true
	case uint8:
		return ratFromUint64(uint64(t)), true
	case uint16:
		return ratFromUint64(uint64(t)), true
	case uint32:
		return ratFromUint64(uint64(t)), true
	case uint64:
		return ratFromUint64(t), true
	case float32:
		return numFromFloat(float64(t), 32, mode), true
	case float64:
		return numFromFloat(t, 64, mode), true
	}
	return nil, false
}

func ratFromUint64(u uint64) *big.Rat {
	return new(big.Rat).SetInt(new(big.Int).SetUint64(u))
}

// numFromFloat converts a Go float to a number. In strict mode it must be a
// Real; in relaxed mode one that is not a Real stays a float64.
func numFromFloat(f float64, bits int, mode NumericMode) any {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		execFail("NUMERIC_OVERFLOW", "%v is not a finite number", f)
	}
	text := strconv.FormatFloat(f, 'g', -1, bits)
	if mode == Relaxed && numberOutOfRange([]byte(text)) {
		return f
	}
	return ratFromText(text)
}

// numFromText reads a number written as text; in relaxed mode one that is not
// a Real is a float64 if a float64 can hold it.
func numFromText(s string, mode NumericMode) any {
	if mode == Relaxed && numberOutOfRange([]byte(s)) {
		if f, ok := parseFloatLiteral(s); ok {
			return f
		}
	}
	return ratFromText(s)
}

// ratFromText reads a number written as text, refusing one that is not a Real
// before any value is built.
func ratFromText(s string) *big.Rat {
	if numberOutOfRange([]byte(s)) {
		execFail("NUMERIC_OVERFLOW", "%s", msgNotReal)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		execFail("NUMERIC_OVERFLOW", "not a number")
	}
	return r
}

// checkHostRat fails with NUMERIC_OVERFLOW unless r is a Real. An integer of
// at most 126 bits is a Real without further work; any other value is tested
// exactly.
func checkHostRat(r *big.Rat) {
	if r.IsInt() && r.Num().BitLen() <= 126 {
		return
	}
	checkNum(r)
}
