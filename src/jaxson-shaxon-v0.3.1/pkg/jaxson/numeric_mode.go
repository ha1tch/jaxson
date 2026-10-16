// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// The numeric mode of the engine (Jaxson core design, section 2). Strict is
// the default. In relaxed mode a number that is not a Real is an IEEE 754
// binary64 float, and determinism across architectures is not guaranteed.
// This runtime computes a relaxed result exactly and rounds it once, so its
// relaxed results are reproducible across architectures; the standard does not
// require that of a runtime. A command line or a host API asks for a mode with
// a switch; the mode itself is carried by Profile.Numbers.

import "errors"

// NumericMode is strict or relaxed.
type NumericMode int

const (
	// Strict is the engine default: every number is a Real.
	Strict NumericMode = iota
	// Relaxed is the optional mode that allows IEEE 754 floats beyond the Real
	// limits, without a guarantee of determinism across architectures.
	Relaxed
)

func (m NumericMode) String() string {
	if m == Relaxed {
		return "relaxed"
	}
	return "strict"
}

// ModeFromFlags reads a pair of command-line switches: strict (true unless
// given) and relaxed (false unless given), with strictSet true if -strict was
// given explicitly. -relaxed overrides the default -strict; giving both is an
// error, as is -strict=false, which names no mode.
func ModeFromFlags(strict, relaxed, strictSet bool) (NumericMode, error) {
	switch {
	case relaxed && strictSet:
		return Strict, errors.New("-strict and -relaxed are mutually exclusive")
	case relaxed:
		return Relaxed, nil
	case !strict:
		return Strict, errors.New("-strict=false names no mode: use -relaxed")
	}
	return Strict, nil
}
