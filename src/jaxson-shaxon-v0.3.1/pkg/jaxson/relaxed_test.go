// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Relaxed numeric mode (Jaxson core design, section 2): a number that is not a
// Real is an IEEE 754 binary64 float.

import (
	"math/big"
	"strings"
	"testing"
)

func relaxedProfile() Profile {
	pr := CoreProfile()
	pr.Numbers = Relaxed
	return pr
}

// relaxedRun runs expr over a and b in relaxed mode and shows the result.
func relaxedRun(t *testing.T, pr Profile, input, expr string) (string, *Err) {
	t.Helper()
	pkg := `{"jaxson":"1.0","inputSchema":{"type":"any"},"outputSchema":{"type":"any"},"input":` + input +
		`,"program":[{"op":"set","path":["output"],"value":{"$compute":{"with":{"a":{"$path":["input","a"]},"b":{"$path":["input","b"]}},"expr":` + expr + `}}}]}`
	out, err := pr.RunJSON([]byte(pkg))
	if err != nil {
		return "", err
	}
	return Show(out), nil
}

func TestRelaxedParse(t *testing.T) {
	v, err := ParseJSONMode([]byte(`[1e300, 12345678901234567890123456789012345678901234567890, 1e37, 0.5, -1e300]`), Relaxed)
	if err != nil {
		t.Fatal(err)
	}
	a := v.([]any)
	if f, ok := a[0].(float64); !ok || f != 1e300 {
		t.Errorf("1e300 read as %T %v", a[0], a[0])
	}
	if _, ok := a[1].(float64); !ok {
		t.Errorf("a 50-digit integer read as %T, want a float64", a[1])
	}
	if _, ok := a[2].(*big.Rat); !ok {
		t.Errorf("1e37 is a Real and read as %T", a[2])
	}
	if _, ok := a[3].(*big.Rat); !ok {
		t.Errorf("0.5 read as %T", a[3])
	}
	if f, ok := a[4].(float64); !ok || f != -1e300 {
		t.Errorf("-1e300 read as %T %v", a[4], a[4])
	}
	for _, lit := range []string{`1e999999`, `1e-999999`, `1e309`, `-1e309`} { // no float64 holds these
		if _, err := ParseJSONMode([]byte(lit), Relaxed); err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("relaxed %s: %v, want PARSE_ERROR", lit, err)
		}
	}
	for _, lit := range []string{`1e300`, `12345678901234567890123456789012345678901234567890`} { // strict still refuses
		if _, err := ParseJSONMode([]byte(lit), Strict); err == nil || err.Cat != "PARSE_ERROR" {
			t.Errorf("strict %s: %v, want PARSE_ERROR", lit, err)
		}
	}
}

func TestRelaxedOperations(t *testing.T) {
	pr := relaxedProfile()
	a, b := `{"$v":"a"}`, `{"$v":"b"}`
	over3 := formatFloat(1e300 / 3)
	for _, c := range []struct{ name, input, expr, want string }{
		{"float + float", `{"a":1e300,"b":2e300}`, `["add",` + a + `,` + b + `]`, "3e+300"},
		{"float - itself is the Real 0", `{"a":1e300,"b":1e300}`, `["sub",` + a + `,` + b + `]`, "0"},
		{"float + 1 rounds back to the float", `{"a":1e300,"b":1}`, `["add",` + a + `,` + b + `]`, "1e+300"},
		{"Reals whose product is beyond the strict range", `{"a":1e20,"b":1e20}`, `["mul",` + a + `,` + b + `]`, "1e+40"},
		{"float / Real with a scale", `{"a":1e300,"b":3}`, `["div",` + a + `,` + b + `,2]`, over3},
		{"a float times a float that comes back in range is a Real", `{"a":1e300,"b":1e-300}`, `["mul",` + a + `,` + b + `]`, "1"},
		{"neg", `{"a":1e300,"b":0}`, `["neg",` + a + `]`, "-1e+300"},
		{"abs", `{"a":-1e300,"b":0}`, `["abs",` + a + `]`, "1e+300"},
		{"max keeps the float", `{"a":1e300,"b":5}`, `["max",` + a + `,` + b + `]`, "1e+300"},
		{"min keeps the Real", `{"a":1e300,"b":5}`, `["min",` + a + `,` + b + `]`, "5"},
		{"float > Real", `{"a":1e300,"b":5}`, `["gt",` + a + `,` + b + `]`, "true"},
		{"Real < float", `{"a":1e300,"b":5}`, `["lt",` + b + `,` + a + `]`, "true"},
		{"float = itself", `{"a":1e300,"b":1e300}`, `["eq",` + a + `,` + b + `]`, "true"},
		{"float != float", `{"a":1e300,"b":1e301}`, `["eq",` + a + `,` + b + `]`, "false"},
		{"to_string of a float", `{"a":1e300,"b":0}`, `["to_string",` + a + `]`, `"1e+300"`},
		{"mod of a float is exact", `{"a":1e300,"b":7}`, `["mod",` + a + `,` + b + `]`,
			new(big.Int).Rem(floatRat(1e300).Num(), big.NewInt(7)).String()},
	} {
		got, err := relaxedRun(t, pr, c.input, c.expr)
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	// a result no float64 can hold is NUMERIC_OVERFLOW
	_, err := relaxedRun(t, pr, `{"a":1e200,"b":1e200}`, `["mul",`+a+`,`+b+`]`)
	if err == nil || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("1e200 * 1e200: %v, want NUMERIC_OVERFLOW", err)
	}
	// strict mode refuses the Real operands' result
	_, err = relaxedRun(t, CoreProfile(), `{"a":1e20,"b":1e20}`, `["mul",`+a+`,`+b+`]`)
	if err == nil || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("strict 1e20 * 1e20: %v, want NUMERIC_OVERFLOW", err)
	}
}

func TestRelaxedToNumber(t *testing.T) {
	long := strings.Repeat("9", 300)
	in := `{"a":"` + long + `","b":0}`
	got, err := relaxedRun(t, relaxedProfile(), in, `["to_number",{"$v":"a"}]`)
	if err != nil || got != "1e+300" {
		t.Errorf("relaxed to_number of 300 nines: %q, %v", got, err)
	}
	_, err = relaxedRun(t, CoreProfile(), in, `["to_number",{"$v":"a"}]`)
	if err == nil || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("strict to_number of 300 nines: %v, want NUMERIC_OVERFLOW", err)
	}
}

func TestRelaxedSchemaChecksFloats(t *testing.T) {
	run := func(max string) *Err {
		pkg := `{"jaxson":"1.0","inputSchema":{"type":"number","max":` + max + `},"outputSchema":{"type":"any"},"input":1e300,"program":[]}`
		_, err := relaxedProfile().RunJSON([]byte(pkg))
		return err
	}
	if err := run("1e301"); err != nil {
		t.Errorf("1e300 within a max of 1e301: %v", err)
	}
	if err := run("1e299"); err == nil || err.Cat != "INPUT_ERROR" {
		t.Errorf("1e300 above a max of 1e299: %v, want INPUT_ERROR", err)
	}
}

func TestAsNumberLeavesFloatsAlone(t *testing.T) {
	if _, ok := AsNumber(1e300); ok {
		t.Error("a float64 is not a Number")
	}
	if n, ok := AsNumber(new(big.Rat).SetInt64(7)); !ok || n.String() != "7" {
		t.Errorf("a Real is a Number: %v %v", n, ok)
	}
}

func TestFloatIsNeverAnIndex(t *testing.T) {
	pkg := `{"jaxson":"1.0","input":{"a":[1,2],"i":1e300},"inputSchema":{"type":"any"},"outputSchema":{"type":"any"},` +
		`"program":[{"op":"set","path":["output"],"value":{"$path":["input","a",{"$path":["input","i"]}]}}]}`
	_, err := relaxedProfile().RunJSON([]byte(pkg))
	if err == nil || err.Code != "TYPE_ERROR" || !strings.Contains(err.Msg, "float") {
		t.Errorf("a float as an index: %v, want TYPE_ERROR naming a float", err)
	}
}

func TestRelaxedHostNumbers(t *testing.T) {
	huge := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(300), nil))
	beyond := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(400), nil))
	d := DataMode(map[string]any{"f": 1e300, "r": huge, "small": 2.5, "i": 7}, Relaxed)
	if got := Show(d); got != `{"f":1e+300,"i":7,"r":1e+300,"small":2.5}` {
		t.Errorf("got %s", got)
	}
	if err := recovered(func() { DataMode(map[string]any{"n": beyond}, Relaxed) }); err == nil || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("10^400 in relaxed mode: %v, want NUMERIC_OVERFLOW", err)
	}
	if err := recovered(func() { Data(map[string]any{"n": 1e300}) }); err == nil || err.Code != "NUMERIC_OVERFLOW" {
		t.Errorf("1e300 in strict mode: %v, want NUMERIC_OVERFLOW", err)
	}
}

func TestFormatFloatReadsBack(t *testing.T) {
	for f, want := range map[float64]string{1e300: "1e+300", -1e300: "-1e+300", 1.5e-7: "1.5e-7", 1e21: "1e+21", 123456789012345680000: "123456789012345680000", 1e-6: "0.000001", 4.9e-324: "5e-324"} {
		if got := formatFloat(f); got != want {
			t.Errorf("formatFloat(%v) = %q, want %q", f, got, want)
		}
		v, err := ParseJSONMode([]byte(formatFloat(f)), Relaxed)
		if err != nil {
			t.Errorf("%v: %v", f, err)
			continue
		}
		if g, ok := v.(float64); ok && g != f {
			t.Errorf("%v read back as %v", f, g)
		}
	}
}
