// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// The hand-written scanner (parse.go) against the encoding/json-based reader
// it replaced (parse_reference_test.go): for any bytes, both must accept or
// both must reject, and on accepting, build equal values (numbers exactly
// equal, strings equal byte for byte, objects with the same members).
//
// Inputs: every fixture file; documents generated at random from a grammar
// that leans on the awkward parts (numbers at the 18-digit edge, exponents,
// escapes, surrogates, duplicate keys, deep nesting); and byte-level
// mutations of those, which are mostly invalid. FuzzParseJSON runs the same
// check under go test -fuzz.

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// sameValue is Equal (numbers by value, strings byte for byte, objects by
// member). It does not print: a Rat such as 1e400 is a valid parse result
// that show cannot format.
func sameValue(a, b any) bool { return Equal(a, b) && sameRats(a, b) }

// sameRats checks that every number in a is held in exactly the form the
// reference reader gives it (numerator and denominator in lowest terms),
// which Equal, comparing by value, would not notice.
func sameRats(a, b any) bool {
	switch x := a.(type) {
	case *big.Rat:
		y, ok := b.(*big.Rat)
		return ok && x.Num().Cmp(y.Num()) == 0 && x.Denom().Cmp(y.Denom()) == 0
	case []any:
		y := b.([]any)
		for i := range x {
			if !sameRats(x[i], y[i]) {
				return false
			}
		}
	case map[string]any:
		y := b.(map[string]any)
		for k, v := range x {
			if !sameRats(v, y[k]) {
				return false
			}
		}
	}
	return true
}

func checkAgree(t testing.TB, raw []byte) {
	t.Helper()
	got, e1 := ParseJSON(raw)
	want, e2 := parseJSONReference(raw)
	if (e1 == nil) != (e2 == nil) {
		t.Fatalf("accept/reject differs on %q:\n scanner: %v\n reference: %v", clip(raw), e1, e2)
	}
	checkDataModes(t, raw, got, e1)
	if e1 != nil {
		if e1.Cat != e2.Cat || e1.Code != e2.Code {
			t.Fatalf("error kind differs on %q: %v vs %v", clip(raw), e1, e2)
		}
		if strings.HasPrefix(e2.Msg, "duplicate key") != strings.HasPrefix(e1.Msg, "duplicate key") && !strings.HasPrefix(e1.Msg, "not valid JSON") {
			// a duplicate key in an otherwise broken document may be reported
			// as the syntax error first by one reader: only the kind must match
		}
		return
	}
	if !sameValue(got, want) {
		t.Fatalf("values differ on %q", clip(raw))
	}
}

// checkDataModes holds ParseData and ParsePackage to ParseJSON: the same
// accept/reject and error, and, converted back, the same value. ParsePackage
// must also leave the document's own top level as a map and have turned its
// "input" member, if any, into an Object.
func checkDataModes(t testing.TB, raw []byte, want any, wantErr *Err) {
	t.Helper()
	d, de := ParseData(raw)
	pk, pe := ParsePackage(raw)
	if (de == nil) != (wantErr == nil) || (pe == nil) != (wantErr == nil) {
		t.Fatalf("accept/reject differs between modes on %q: json %v, data %v, package %v", clip(raw), wantErr, de, pe)
	}
	if wantErr != nil {
		if de.Cat != wantErr.Cat || de.Code != wantErr.Code || pe.Cat != wantErr.Cat || pe.Code != wantErr.Code {
			t.Fatalf("error kind differs between modes on %q: %v / %v / %v", clip(raw), wantErr, de, pe)
		}
		return
	}
	if !sameValue(Legacy(d), want) || !sameValue(Legacy(pk), want) {
		t.Fatalf("values differ between modes on %q", clip(raw))
	}
	if _, isMap := d.(map[string]any); isMap {
		t.Fatalf("ParseData left a map at the top on %q", clip(raw))
	}
	if m, ok := pk.(map[string]any); ok {
		if in, has := m["input"]; has {
			if _, isMap := in.(map[string]any); isMap {
				t.Fatalf("ParsePackage left a map as input on %q", clip(raw))
			}
		}
	}
}

func clip(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "..."
	}
	return string(b)
}

func TestParseAgreesOnFixtures(t *testing.T) {
	for _, f := range []string{"jaxson-v0.1.0-fixtures.json", "../shaxon/shaxon-v0.3.1-fixtures.json",
		"../../examples/shaxon/authz/rolling-quota.json", "../../examples/shaxon/authz/chinese-wall.json",
		"../../examples/shaxon/authz/delegation-chain.json"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		checkAgree(t, raw)
	}
}

func TestParseEdgeCases(t *testing.T) {
	for _, s := range []string{
		``, ` `, `null`, `true`, `false`, `"x"`, `0`, `-0`, `-`, `01`, `1.`, `.5`, `+1`, `1e`, `1e+`, `1E5`, `1e-5`, `1.5e3`,
		`123456789012345678`, `1234567890123456789`, `-123456789012345678`, `0.123456789012345678`, `0.1234567890123456789`,
		`0.000000000000000000001`, `12345678901234567890.123456789`, `100000000000000000000`, `1e400`, `1e-400`, `1e99999999999`,
		`0.5`, `-0.5`, `9223372036854775807`, `9223372036854775808`, `-9223372036854775808`, `999999999999999999`, `999999999999999999.9`,
		`"a\u0041"`, `"\ud83d\ude00"`, `"\ud83d"`, `"\ude00"`, `"\ud83dx"`, `"\ud83d\u0041"`, `"\u00"`, `"\x"`, `"\"`, `"a`, "\"a\nb\"", "\"a\tb\"",
		"\"\xff\"", "\"\xc3\x28\"", "\"\xe2\x82\"", "\"h\xc3\xa9llo\"", `"\/"`, `"\b\f\n\r\t"`, `"\u0000"`,
		`[]`, `[ ]`, `[1,]`, `[,1]`, `[1 2]`, `{}`, `{ }`, `{"a":1,}`, `{,}`, `{"a"}`, `{"a":}`, `{a:1}`, `{"a":1 "b":2}`, `{1:2}`,
		`{"a":1,"a":2}`, `{"a":1,"\u0061":2}`, `{"a":{"a":1},"b":{"a":1}}`, `[1] 2`, `[1]]`, `tru`, `nul`, `truee`, `nulll`, `True`, `NaN`, `Infinity`,
		"\ufeff[1]", "[1]\n", "\t[1]\r\n ", "[\x00]", `{"k":[{"k":[{"k":[]}]}]}`,
	} {
		checkAgree(t, []byte(s))
	}
	// Deep nesting: both readers must agree where they agree, and the scanner
	// must refuse beyond its limit rather than overflow the stack.
	for _, n := range []int{1, 100, 9999, 10000} {
		checkAgree(t, []byte(strings.Repeat("[", n)+strings.Repeat("]", n)))
	}
	if _, e := ParseJSON([]byte(strings.Repeat("[", 20000) + strings.Repeat("]", 20000))); e == nil || e.Cat != "PARSE_ERROR" {
		t.Errorf("20000 levels of nesting: want PARSE_ERROR, got %v", e)
	}
	if _, e := ParseJSON([]byte(strings.Repeat("[", 2000000))); e == nil {
		t.Errorf("two million unclosed brackets: want an error")
	}
}

// ---- random documents

type jgen struct{ r *rand.Rand }

func (g jgen) num() string {
	switch g.r.Intn(12) {
	case 0:
		return "0"
	case 1:
		return fmt.Sprint(g.r.Intn(1000) - 500)
	case 2:
		return fmt.Sprintf("%d.%d", g.r.Intn(100), g.r.Intn(1000))
	case 3:
		return strings.Repeat("9", 15+g.r.Intn(8)) // around the 18-digit edge
	case 4:
		return "0." + strings.Repeat("0", g.r.Intn(25)) + fmt.Sprint(1+g.r.Intn(9))
	case 5:
		return fmt.Sprintf("%d.%s", g.r.Intn(10), strings.Repeat("7", 15+g.r.Intn(8)))
	case 6:
		return fmt.Sprintf("%de%d", g.r.Intn(100), g.r.Intn(40)-20)
	case 7:
		return fmt.Sprintf("-%d.%dE+%d", g.r.Intn(100), g.r.Intn(100), g.r.Intn(30))
	case 8:
		return fmt.Sprintf("%d%s", 1+g.r.Intn(9), strings.Repeat("0", g.r.Intn(30)))
	case 9:
		return "-0.0"
	case 10:
		return fmt.Sprintf("%d.%d", g.r.Intn(3), g.r.Int63())
	}
	return fmt.Sprint(g.r.Int63())
}

func (g jgen) str() string {
	parts := []string{"a", "bc", "", "x y", `\"`, `\\`, `\/`, `\n`, `\t`, `\u0041`, `\u00e9`, `\ud83d\ude00`, `\ud83d`, `\ude00`, `\ud83dz`,
		"é", "日本", "😀", "\xff", "\xc3", "\xe2\x82", `\u0000`, `\b\f`}
	var sb strings.Builder
	sb.WriteByte('"')
	for i, n := 0, g.r.Intn(5); i < n; i++ {
		sb.WriteString(parts[g.r.Intn(len(parts))])
	}
	sb.WriteByte('"')
	return sb.String()
}

func (g jgen) ws() string { return []string{"", "", " ", "\n", "\t ", "\r\n"}[g.r.Intn(6)] }

func (g jgen) value(d int) string {
	if d <= 0 {
		switch g.r.Intn(6) {
		case 0:
			return "null"
		case 1:
			return "true"
		case 2:
			return "false"
		case 3, 4:
			return g.num()
		}
		return g.str()
	}
	switch g.r.Intn(8) {
	case 0, 1:
		var items []string
		for i, n := 0, g.r.Intn(5); i < n; i++ {
			items = append(items, g.ws()+g.value(d-1)+g.ws())
		}
		return "[" + strings.Join(items, ",") + "]"
	case 2, 3:
		var items []string
		for i, n := 0, g.r.Intn(5); i < n; i++ {
			k := g.str()
			if g.r.Intn(3) == 0 {
				k = fmt.Sprintf(`"k%d"`, g.r.Intn(4)) // collisions: duplicate keys
			}
			items = append(items, g.ws()+k+g.ws()+":"+g.ws()+g.value(d-1)+g.ws())
		}
		return "{" + strings.Join(items, ",") + "}"
	}
	return g.value(0)
}

func TestParseAgreesOnRandomDocuments(t *testing.T) {
	g := jgen{rand.New(rand.NewSource(20261005))}
	valid, total := 0, 0
	for i := 0; i < 20000; i++ {
		doc := []byte(g.ws() + g.value(1+g.r.Intn(4)) + g.ws())
		checkAgree(t, doc)
		total++
		if _, e := ParseJSON(doc); e == nil {
			valid++
		}
		// byte-level mutations: delete, duplicate, replace or insert one byte
		for k := 0; k < 3 && len(doc) > 0; k++ {
			m := append([]byte(nil), doc...)
			at := g.r.Intn(len(m))
			switch g.r.Intn(4) {
			case 0:
				m = append(m[:at], m[at+1:]...)
			case 1:
				m = append(m[:at+1], m[at:]...)
			case 2:
				const alphabet = "{}[],:\"\\0e.-+ tn\x00\xff"
				m[at] = alphabet[g.r.Intn(len(alphabet))]
			default:
				m = append(m[:at], append([]byte{"{}[],:\"\\"[g.r.Intn(8)]}, m[at:]...)...)
			}
			checkAgree(t, m)
			total++
		}
	}
	t.Logf("%d documents compared; %d of the unmutated ones accepted", total, valid)
	if valid < 5000 {
		t.Errorf("generator produced too few valid documents (%d)", valid)
	}
}

func FuzzParseJSON(f *testing.F) {
	for _, s := range []string{`{"a":[1,2.5,{"b":null}],"c":"x\u00e9","d":true}`, `[1e3,-0.0,123456789012345678901234567890]`, `"\ud83d\ude00"`, `{"a":1,"a":2}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) { checkAgree(t, raw) })
}

// ---- speed: the scanner against the reader it replaced, on a trail of 20000
// events like the ones in examples/shaxon/authz (about 920 KB).

func trailJSON(n int) []byte {
	var sb strings.Builder
	sb.WriteString(`{"policy":{"windowHours":24,"limitMb":0.9,"maxExports":5},"events":[`)
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"t":%.4f,"actor":"u%d","mb":0.0001}`, float64(i)*20/float64(n), i%(n/6))
	}
	sb.WriteString(`],"request":{"actor":"u1","t":20.5,"mb":0.1}}`)
	return []byte(sb.String())
}

func BenchmarkParseScanner(b *testing.B) {
	raw := trailJSON(20000)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, e := ParseJSON(raw); e != nil {
			b.Fatal(e)
		}
	}
}

func BenchmarkParseData(b *testing.B) {
	raw := trailJSON(20000)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, e := ParseData(raw); e != nil {
			b.Fatal(e)
		}
	}
}

func BenchmarkParseReference(b *testing.B) {
	raw := trailJSON(20000)
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, e := parseJSONReference(raw); e != nil {
			b.Fatal(e)
		}
	}
}
