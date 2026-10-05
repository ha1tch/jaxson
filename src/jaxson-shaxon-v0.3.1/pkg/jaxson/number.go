// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// Phase 7.1: the number type at the API boundary.
//
// Inside, a number is a *big.Rat and the machine shares those freely (a Rat
// in a result may be the very one the machine still holds, and Clone shares
// them on purpose). A caller who keeps one, or changes one through the
// Rat's own methods, would be changing the machine's values behind its back.
// Number is the value a caller is meant to hold: immutable, comparable by
// value, and always an exact finite decimal, as every number the language
// produces is. Run's results stay *big.Rat for now (changing the element
// type of every result would break each existing caller); AsNumber is the
// way from one to the other.

import (
	"errors"
	"math"
	"math/big"
	"regexp"
)

// Number is an exact decimal number. The zero value is 0. Numbers are
// immutable and safe to share between goroutines.
type Number struct{ r *big.Rat }

var jsonNumRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

// ParseNumber reads s as a JSON number. It is an error if s is not one, or
// if the number has more than 38 significant digits (the language's limit).
func ParseNumber(s string) (Number, error) {
	if !jsonNumRe.MatchString(s) {
		return Number{}, errors.New("not a JSON number: " + s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return Number{}, errors.New("not a JSON number: " + s)
	}
	n, ok := NumberFromRat(r)
	if !ok {
		return Number{}, errors.New("number out of range: " + s)
	}
	return n, nil
}

// NumberFromInt64 returns the number i.
func NumberFromInt64(i int64) Number { return Number{new(big.Rat).SetInt64(i)} }

// NumberFromRat returns a Number equal to r, which is copied. It reports
// false if r is not a finite decimal of at most 38 digits.
func NumberFromRat(r *big.Rat) (Number, bool) {
	c, _, ok := decParts(r)
	if !ok || len(new(big.Int).Abs(c).String()) > maxDigits {
		return Number{}, false
	}
	return Number{new(big.Rat).Set(r)}, true
}

// AsNumber converts a number as the machine holds it (a *big.Rat), or a
// Number, to a Number. It reports false for any other value, and for a Rat
// that is not a decimal within the language's limits.
func AsNumber(v any) (Number, bool) {
	switch t := v.(type) {
	case Number:
		return t, true
	case *big.Rat:
		return NumberFromRat(t)
	}
	return Number{}, false
}

func (n Number) rat() *big.Rat {
	if n.r == nil {
		return new(big.Rat)
	}
	return n.r
}

// String is the number in JSON's decimal notation, with no exponent and no
// trailing zeros: 5, -0.25, 1000000.
func (n Number) String() string { return FormatDecimal(n.rat()) }

// Rat returns the value as a new *big.Rat that the caller may keep and change.
func (n Number) Rat() *big.Rat { return new(big.Rat).Set(n.rat()) }

// Sign returns -1, 0 or 1.
func (n Number) Sign() int { return n.rat().Sign() }

// IsInt reports whether the number has no fractional part.
func (n Number) IsInt() bool { return n.rat().IsInt() }

// Int64 returns the value if it is an integer that fits an int64.
func (n Number) Int64() (int64, bool) {
	r := n.rat()
	if !r.IsInt() || !r.Num().IsInt64() {
		return 0, false
	}
	return r.Num().Int64(), true
}

// Float64 returns the nearest float64 and whether it is exactly the number.
// A number outside float64's range gives ±Inf and false.
func (n Number) Float64() (f float64, exact bool) {
	f, exact = n.rat().Float64()
	if math.IsInf(f, 0) {
		return f, false
	}
	return f, exact
}

// Cmp compares n and m: -1, 0 or 1.
func (n Number) Cmp(m Number) int { return n.rat().Cmp(m.rat()) }

// MarshalJSON writes the number as a JSON number, exactly.
func (n Number) MarshalJSON() ([]byte, error) { return []byte(n.String()), nil }

// MarshalText is String, so a Number is also usable as a map key in JSON.
func (n Number) MarshalText() ([]byte, error) { return []byte(n.String()), nil }
