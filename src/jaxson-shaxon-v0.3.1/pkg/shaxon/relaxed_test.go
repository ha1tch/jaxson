// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Relaxed numeric mode through Shaxon (Jaxson core design, section 2): a number
// that does not fit the strict range is a float64, and shapes, index keys and
// Validate treat it as a number.

import (
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func relaxedPkg(t *testing.T, s string) map[string]any {
	t.Helper()
	v, err := jaxson.ParseJSONMode([]byte(s), jaxson.Relaxed)
	if err != nil {
		t.Fatal(err)
	}
	return v.(map[string]any)
}

func relaxedDoc(t *testing.T, s string) any {
	t.Helper()
	v, err := jaxson.ParseDataMode([]byte(s), jaxson.Relaxed)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

const relaxedShapes = `{"shaxon":"3.1","shapes":{"P":{"kind":"object","required":["n"],"fields":{"n":{"kind":"number","min":0,"max":1e301}}}}}`

func TestRelaxedValidateFloats(t *testing.T) {
	pkg := relaxedPkg(t, relaxedShapes)
	for name, c := range map[string]struct {
		doc  string
		want bool
	}{
		"a float within the bounds":  {`{"n":1e300}`, true},
		"a Real within the bounds":   {`{"n":12.5}`, true},
		"a float above the maximum":  {`{"n":1e302}`, false},
		"a negative float below min": {`{"n":-1e300}`, false},
	} {
		rep, err := ValidateMode(pkg, "P", relaxedDoc(t, c.doc), jaxson.Relaxed)
		if err != nil || rep.Conforms() != c.want {
			t.Errorf("%s: %v, %v; want conforms=%v", name, rep.Value(), err, c.want)
		}
	}
}

func TestRelaxedRunJSON(t *testing.T) {
	pkg := `{"shaxon":"3.1","input":{"a":1e300,"b":2e300},"program":[{"op":"set","path":["output"],"value":{"$compute":{"with":{"a":{"$path":["input","a"]},"b":{"$path":["input","b"]}},"expr":["add",{"$v":"a"},{"$v":"b"}]}}}]}`
	res, err := RunJSONMode([]byte(pkg), jaxson.Relaxed)
	if err != nil || jaxson.Show(res.Output) != "3e+300" {
		t.Fatalf("got %s, %v", jaxson.Show(res.Output), err)
	}
	if _, err := RunJSON([]byte(pkg)); err == nil || err.Cat != "PARSE_ERROR" {
		t.Errorf("strict mode: %v, want PARSE_ERROR", err)
	}
}

func TestFloatKeysAreKeys(t *testing.T) {
	a, okA := keyFlag(1e300)
	b, okB := keyFlag(1e300)
	c, _ := keyFlag(1e301)
	if !okA || !okB || a != b || a == c {
		t.Errorf("float keys: %v %v %v", a, b, c)
	}
	five, _ := jaxson.ParseJSONMode([]byte(`5`), jaxson.Relaxed)
	if compareKeys(1e300, five) <= 0 || compareKeys(five, 1e300) >= 0 || compareKeys(1e300, 1e300) != 0 {
		t.Error("a float and a Real do not order as numbers")
	}
}
