// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// ParseJSON: the strict reader for a package or an input document.
//
// The core design (jaxson-v0.1.0-core-design.md, and shaxon core section 2)
// says duplicate object keys anywhere are a PARSE_ERROR. encoding/json
// silently keeps the last duplicate, so a plain Decode cannot be used for
// that; this reader is its own scanner. It also rejects anything after the
// document.
//
// Numbers become *big.Rat exactly as Normalize makes them, so the result
// can be handed straight to Run or to a Profile. The scanner builds the
// value in one pass over the bytes. What it accepts, and the value it
// builds, are the same as the encoding/json-based reader it replaced: that
// reader is kept in parse_reference_test.go and parse_diff_test.go compares
// the two on generated and mutated documents, and on every fixture.
//
// Where it differs from a general reader, on purpose:
//   - a number with at most 18 significant digits and no exponent is turned
//     into a Rat directly (an integer mantissa over a power of ten); anything
//     else goes through big.Rat.SetString, as before;
//   - object keys are interned per document: a trail of events repeats the
//     same few member names thousands of times, and each is stored once;
//   - nesting deeper than 10000 is a PARSE_ERROR (encoding/json's limit),
//     so a hostile document cannot exhaust the stack.

import (
	"math/big"
	"unicode/utf16"
	"unicode/utf8"
)

const maxJSONDepth = 10000

// ParseJSON decodes raw as one JSON document into the value model Run
// expects. It fails with PARSE_ERROR if raw is not JSON, has a duplicate
// object key at any depth, or has anything after the document.
func ParseJSON(raw []byte) (v any, err *Err) {
	defer recoverErr(&err)
	p := &scanner{b: raw}
	p.ws()
	v = p.value(0)
	p.ws()
	if p.i < len(p.b) {
		fail("PARSE_ERROR", "", "unexpected data after the JSON document")
	}
	return v, nil
}

// ParseData is ParseJSON producing the machine's own representation: objects
// are *Object, not map[string]any. Use it for a document that will be handed
// to Run as the package's input; it saves the conversion Run would otherwise
// make. Convert the result with Legacy for map form.
func ParseData(raw []byte) (v any, err *Err) {
	defer recoverErr(&err)
	p := &scanner{b: raw, data: true}
	p.ws()
	v = p.value(0)
	p.ws()
	if p.i < len(p.b) {
		fail("PARSE_ERROR", "", "unexpected data after the JSON document")
	}
	return v, nil
}

// ParsePackage is ParseJSON for a package document: the package itself is
// map[string]any, as Run expects, but its "input" member, the one large
// data-bearing part, is parsed straight into the machine's representation.
func ParsePackage(raw []byte) (v any, err *Err) {
	defer recoverErr(&err)
	p := &scanner{b: raw, pkg: true}
	p.ws()
	v = p.value(0)
	p.ws()
	if p.i < len(p.b) {
		fail("PARSE_ERROR", "", "unexpected data after the JSON document")
	}
	return v, nil
}

type scanner struct {
	b    []byte
	i    int
	keys map[string]string // interned object keys
	data bool              // build *Object, not map[string]any
	pkg  bool              // the top-level object's "input" member is built as data
	kk   map[string]Key    // Keys made so far (data mode)
}

func (p *scanner) bad(what string) {
	fail("PARSE_ERROR", "", "not valid JSON: %s at offset %d", what, p.i)
}

func (p *scanner) ws() {
	for p.i < len(p.b) {
		switch p.b[p.i] {
		case ' ', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

func (p *scanner) value(depth int) any {
	if p.i >= len(p.b) {
		p.bad("unexpected end of input")
	}
	switch c := p.b[p.i]; {
	case c == '{':
		return p.object(depth + 1)
	case c == '[':
		return p.array(depth + 1)
	case c == '"':
		return p.str()
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	case c == 't':
		p.lit("true")
		return true
	case c == 'f':
		p.lit("false")
		return false
	case c == 'n':
		p.lit("null")
		return nil
	}
	p.bad("unexpected character")
	return nil
}

func (p *scanner) lit(word string) {
	if len(p.b)-p.i < len(word) || string(p.b[p.i:p.i+len(word)]) != word {
		p.bad("invalid literal")
	}
	p.i += len(word)
}

func (p *scanner) object(depth int) any {
	if depth > maxJSONDepth {
		p.bad("nesting too deep")
	}
	if p.data {
		return p.objectData(depth)
	}
	p.i++ // {
	obj := map[string]any{}
	p.ws()
	if p.i < len(p.b) && p.b[p.i] == '}' {
		p.i++
		return obj
	}
	for {
		p.ws()
		if p.i >= len(p.b) || p.b[p.i] != '"' {
			p.bad("object key expected")
		}
		key := p.key()
		if _, dup := obj[key]; dup {
			fail("PARSE_ERROR", "", "duplicate key %q", key)
		}
		p.ws()
		if p.i >= len(p.b) || p.b[p.i] != ':' {
			p.bad("':' expected")
		}
		p.i++
		p.ws()
		if p.pkg && depth == 1 && key == "input" {
			p.data = true
			obj[key] = p.value(depth)
			p.data = false
		} else {
			obj[key] = p.value(depth)
		}
		p.ws()
		if p.i >= len(p.b) {
			p.bad("unexpected end of input")
		}
		switch p.b[p.i] {
		case ',':
			p.i++
		case '}':
			p.i++
			return obj
		default:
			p.bad("',' or '}' expected")
		}
	}
}

// objectData is object building an *Object.
func (p *scanner) objectData(depth int) any {
	p.i++ // {
	obj := NewObject(0)
	p.ws()
	if p.i < len(p.b) && p.b[p.i] == '}' {
		p.i++
		return obj
	}
	for {
		p.ws()
		if p.i >= len(p.b) || p.b[p.i] != '"' {
			p.bad("object key expected")
		}
		name := p.key()
		k, ok := p.kk[name]
		if !ok {
			k = MakeKey(name)
			if p.kk == nil {
				p.kk = map[string]Key{}
			}
			if len(p.kk) < 4096 {
				p.kk[name] = k
			}
		}
		if obj.findKey(k) >= 0 {
			fail("PARSE_ERROR", "", "duplicate key %q", name)
		}
		p.ws()
		if p.i >= len(p.b) || p.b[p.i] != ':' {
			p.bad("':' expected")
		}
		p.i++
		p.ws()
		obj.add(k, p.value(depth))
		p.ws()
		if p.i >= len(p.b) {
			p.bad("unexpected end of input")
		}
		switch p.b[p.i] {
		case ',':
			p.i++
		case '}':
			p.i++
			return obj
		default:
			p.bad("',' or '}' expected")
		}
	}
}

func (p *scanner) array(depth int) any {
	if depth > maxJSONDepth {
		p.bad("nesting too deep")
	}
	p.i++ // [
	arr := []any{}
	p.ws()
	if p.i < len(p.b) && p.b[p.i] == ']' {
		p.i++
		return arr
	}
	for {
		p.ws()
		arr = append(arr, p.value(depth))
		p.ws()
		if p.i >= len(p.b) {
			p.bad("unexpected end of input")
		}
		switch p.b[p.i] {
		case ',':
			p.i++
		case ']':
			p.i++
			return arr
		default:
			p.bad("',' or ']' expected")
		}
	}
}

// ---------------------------------------------------------------- strings

// key reads an object key. A key that is plain (no escapes, valid UTF-8) is
// looked up in the document's table without allocating; the first occurrence
// of each is stored.
func (p *scanner) key() string {
	raw, plain := p.scanString()
	if !plain {
		return p.unquote(raw)
	}
	if s, ok := p.keys[string(raw)]; ok {
		return s
	}
	s := string(raw)
	if len(s) <= 64 && len(p.keys) < 4096 {
		if p.keys == nil {
			p.keys = map[string]string{}
		}
		p.keys[s] = s
	}
	return s
}

func (p *scanner) str() any {
	raw, plain := p.scanString()
	if plain {
		return string(raw)
	}
	return p.unquote(raw)
}

// scanString consumes a string literal starting at the opening quote and
// returns the bytes between the quotes. plain is true if they contain no
// escape and are valid UTF-8, so they are the value as they stand.
func (p *scanner) scanString() (raw []byte, plain bool) {
	p.i++ // opening quote
	start := p.i
	plain = true
	high := false
	for j := start; j < len(p.b); j++ {
		c := p.b[j]
		switch {
		case c == '"':
			raw = p.b[start:j]
			p.i = j + 1
			if high && !utf8.Valid(raw) {
				plain = false
			}
			return raw, plain
		case c == '\\':
			// Find the end, honouring escapes, and let unquote validate.
			plain = false
			for j += 2; j < len(p.b); j++ { // j+1 is the escaped character
				if p.b[j] == '\\' {
					j++
					continue
				}
				if p.b[j] == '"' {
					raw = p.b[start:j]
					p.i = j + 1
					return raw, false
				}
				if p.b[j] < 0x20 {
					p.i = j
					p.bad("control character in string")
				}
			}
			p.i = len(p.b)
			p.bad("unterminated string")
		case c < 0x20:
			p.i = j
			p.bad("control character in string")
		case c >= 0x80:
			high = true
		}
	}
	p.i = len(p.b)
	p.bad("unterminated string")
	return nil, false
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

func (p *scanner) hex4(s []byte, at int) rune {
	if at+4 > len(s) {
		p.bad("bad \\u escape")
	}
	var r rune
	for k := 0; k < 4; k++ {
		h := hexVal(s[at+k])
		if h < 0 {
			p.bad("bad \\u escape")
		}
		r = r<<4 | rune(h)
	}
	return r
}

// unquote decodes the bytes of a string literal that has escapes or invalid
// UTF-8. Invalid UTF-8 and unpaired surrogate escapes become U+FFFD, as
// encoding/json does.
func (p *scanner) unquote(s []byte) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\':
			i++
			if i >= len(s) {
				p.bad("bad escape")
			}
			switch s[i] {
			case '"', '\\', '/':
				out = append(out, s[i])
				i++
			case 'b':
				out = append(out, '\b')
				i++
			case 'f':
				out = append(out, '\f')
				i++
			case 'n':
				out = append(out, '\n')
				i++
			case 'r':
				out = append(out, '\r')
				i++
			case 't':
				out = append(out, '\t')
				i++
			case 'u':
				r := p.hex4(s, i+1)
				i += 5
				if utf16.IsSurrogate(r) {
					r2 := rune(-1)
					if i+1 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
						r2 = p.hex4(s, i+2)
					}
					if dec := utf16.DecodeRune(r, r2); dec != utf8.RuneError {
						r = dec
						i += 6
					} else {
						r = utf8.RuneError
					}
				}
				out = utf8.AppendRune(out, r)
			default:
				p.bad("bad escape")
			}
		case c < utf8.RuneSelf:
			out = append(out, c)
			i++
		default:
			r, w := utf8.DecodeRune(s[i:])
			if r == utf8.RuneError && w == 1 {
				out = utf8.AppendRune(out, utf8.RuneError)
			} else {
				out = append(out, s[i:i+w]...)
			}
			i += w
		}
	}
	return string(out)
}

// ---------------------------------------------------------------- numbers

var pow10 = [...]int64{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000,
	1000000000, 10000000000, 100000000000, 1000000000000, 10000000000000,
	100000000000000, 1000000000000000, 10000000000000000, 100000000000000000,
	1000000000000000000}

// number scans a JSON number (the strict grammar) and returns it as a *big.Rat.
func (p *scanner) number() any {
	start := p.i
	b := p.b
	i := p.i
	neg := false
	if b[i] == '-' {
		neg = true
		i++
	}
	if i >= len(b) {
		p.bad("invalid number")
	}
	var mant int64
	digits := 0 // significant digits accumulated in mant
	switch {
	case b[i] == '0':
		i++
	case b[i] >= '1' && b[i] <= '9':
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			if digits < 18 {
				mant = mant*10 + int64(b[i]-'0')
			}
			digits++
			i++
		}
	default:
		p.i = i
		p.bad("invalid number")
	}
	scale := 0
	exact := true
	if i < len(b) && b[i] == '.' {
		i++
		if i >= len(b) || b[i] < '0' || b[i] > '9' {
			p.i = i
			p.bad("invalid number")
		}
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			// Leading zeros of a fraction below 1 are not significant digits
			// (they add to scale only).
			if digits < 18 {
				mant = mant*10 + int64(b[i]-'0')
				if mant != 0 {
					digits++
				}
			} else {
				exact = false
			}
			scale++
			i++
		}
	}
	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		exact = false
		i++
		if i < len(b) && (b[i] == '+' || b[i] == '-') {
			i++
		}
		if i >= len(b) || b[i] < '0' || b[i] > '9' {
			p.i = i
			p.bad("invalid number")
		}
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
		}
	}
	p.i = i
	if exact && digits <= 18 && scale <= 18 {
		if neg {
			mant = -mant
		}
		if scale == 0 {
			return big.NewRat(mant, 1)
		}
		return decimalRat(mant, pow10[scale])
	}
	r, ok := new(big.Rat).SetString(string(b[start:i]))
	if !ok {
		fail("PARSE_ERROR", "", "bad number %s", b[start:i])
	}
	return r
}

// decimalRat is mant/den in lowest terms, where den is a power of ten. The
// only primes in den are 2 and 5, so dividing them out of both gives lowest
// terms with no gcd; the Rat is then built without the normalising step that
// SetFrac64 would repeat (that gcd was a third of the cost of reading a
// document of decimals).
func decimalRat(mant, den int64) *big.Rat {
	for den%2 == 0 && mant%2 == 0 {
		mant /= 2
		den /= 2
	}
	for den%5 == 0 && mant%5 == 0 {
		mant /= 5
		den /= 5
	}
	r := new(big.Rat).SetInt64(mant)
	if den != 1 {
		// Denom of a Rat that has been assigned a value is a reference to its
		// denominator (math/big documents this); the value stays normalised
		// because mant and den are coprime and den is positive.
		r.Denom().SetInt64(den)
	}
	return r
}
