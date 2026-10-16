// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// The parse entry points with a numeric mode (Jaxson core design, section 2).
// ParseJSON, ParseData and ParsePackage are these in strict mode. In relaxed
// mode a number that is not a Real is read as a float64 if a float64 can hold
// it, and is a PARSE_ERROR if not.

func parseDoc(p *scanner) (v any, err *Err) {
	defer recoverErr(&err)
	p.ws()
	v = p.value(0)
	p.ws()
	if p.i < len(p.b) {
		fail("PARSE_ERROR", "", "unexpected data after the JSON document")
	}
	return v, nil
}

// ParseJSONMode is ParseJSON in the given numeric mode.
func ParseJSONMode(raw []byte, mode NumericMode) (any, *Err) {
	return parseDoc(&scanner{b: raw, rlx: mode == Relaxed})
}

// ParseDataMode is ParseData in the given numeric mode.
func ParseDataMode(raw []byte, mode NumericMode) (any, *Err) {
	return parseDoc(&scanner{b: raw, data: true, rlx: mode == Relaxed})
}

// ParsePackageMode is ParsePackage in the given numeric mode.
func ParsePackageMode(raw []byte, mode NumericMode) (any, *Err) {
	return parseDoc(&scanner{b: raw, pkg: true, rlx: mode == Relaxed})
}
