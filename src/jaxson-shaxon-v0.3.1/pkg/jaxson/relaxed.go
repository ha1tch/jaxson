// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Relaxed numeric mode (Jaxson core design, section 2). In strict mode every
// number is a Real, a *big.Rat that is an exact decimal of at most 38 digits
// and 200 after the point. In relaxed mode a number that is not a Real, a
// literal beyond those limits or the result of an operation beyond them, is
// an IEEE 754 binary64 float, held as a float64. A float and a Real are both
// numbers: they compare and order by their exact values. An operation with a
// float operand, or whose result is not a Real, is done on the exact values
// and gives the float64 nearest to the exact result, unless that result is a
// Real. A float is never zero (zero is a Real) and never infinite or NaN.

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Messages for a number that is not a Real. They name the limits and the way
// out, and each begins as the older, shorter message did, so a match on that
// prefix still holds.
var (
	msgOutOfRange = fmt.Sprintf("number out of range: a Real has at most %d digits and %d after the point; relaxed mode reads a larger number as a float64", maxDigits, maxScale)
	msgNotReal    = fmt.Sprintf("not a Real: more than %d digits, or more than %d after the point (relaxed mode rounds it to a float64)", maxDigits, maxScale)
	msgNoFloat    = "number out of range: no float64 can hold it"
)

// floatRat is the exact value of f, which must be finite.
func floatRat(f float64) *big.Rat { return new(big.Rat).SetFloat64(f) }

// numberRat returns the exact value of a number of either kind.
func numberRat(x any) (*big.Rat, bool) {
	switch t := x.(type) {
	case *big.Rat:
		return t, true
	case float64:
		return floatRat(t), true
	}
	return nil, false
}

// AsRat returns the exact value of x, a Real or a float, as a *big.Rat. It
// reports false for any other value. A dialect that reads numbers out of a
// document uses it so that it handles floats in relaxed mode.
func AsRat(x any) (*big.Rat, bool) { return numberRat(x) }

// isReal reports whether r is a Real: an exact decimal of at most maxDigits
// digits and at most maxScale after the point.
func isReal(r *big.Rat) bool {
	if r.IsInt() && r.Num().BitLen() <= 126 {
		return true // an integer of at most 126 bits has at most 38 digits
	}
	c, _, ok := decParts(r)
	return ok && len(new(big.Int).Abs(c).String()) <= maxDigits
}

// fitMode gives the number r is in the given mode: r itself if it is a Real,
// in relaxed mode the float64 nearest r if it is not, and otherwise
// NUMERIC_OVERFLOW. A value no float64 can hold, too large or so small that it
// would round to zero, is NUMERIC_OVERFLOW in either mode.
func fitMode(r *big.Rat, mode NumericMode) any {
	if isReal(r) {
		return r
	}
	if mode == Relaxed {
		f, _ := r.Float64()
		if !math.IsInf(f, 0) && f != 0 {
			if fr := floatRat(f); isReal(fr) {
				return fr // a float64 whose exact value is a Real is that Real
			}
			return f
		}
		execFail("NUMERIC_OVERFLOW", "%s", msgNoFloat)
	}
	execFail("NUMERIC_OVERFLOW", "%s", msgNotReal)
	return nil
}

// SetNumericMode sets the machine's numeric mode. Strict is the default.
func (m *Machine) SetNumericMode(mode NumericMode) { m.numMode = mode }

// fit is fitMode in the machine's mode.
func (m *Machine) fit(r *big.Rat) any { return fitMode(r, m.numMode) }

// parseFloatLiteral reads a JSON number literal that is not a Real as a
// float64. It reports false if the literal is outside the range of a float64:
// too large, or so small that it rounds to zero.
func parseFloatLiteral(lit string) (float64, bool) {
	f, err := strconv.ParseFloat(lit, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) || f == 0 {
		return 0, false
	}
	return f, true
}

// formatFloat writes f as a JSON number in the shortest form that reads back
// as the same float64: plain digits from 1e-6 up to 1e21, otherwise an
// exponent with no leading zeros.
func formatFloat(f float64) string {
	abs := math.Abs(f)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	s := strconv.FormatFloat(f, 'e', -1, 64)
	i := strings.IndexByte(s, 'e')
	exp := s[i+2:] // after the sign
	for len(exp) > 1 && exp[0] == '0' {
		exp = exp[1:]
	}
	return s[:i+2] + exp
}
