// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Numbers handed to Validate as Go values (Jaxson core design, section 2).

import (
	"math"
	"math/big"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const hostPkg = `{"shaxon":"3.1","shapes":{"P":{"kind":"object","required":["n"],"fields":{"n":{"kind":"number","min":0,"max":10}}}}}`

func TestValidateAcceptsHostNumbers(t *testing.T) {
	pkg := decode(t, hostPkg)
	for name, n := range map[string]any{"int": 3, "float64": 2.5, "Number": jaxson.NumberFromInt64(4), "uint8": uint8(9)} {
		rep, err := Validate(pkg, "P", map[string]any{"n": n})
		if err != nil || !rep.Conforms() {
			t.Errorf("%s: %v, %v", name, rep.Value(), err)
		}
	}
	rep, err := Validate(pkg, "P", map[string]any{"n": 11})
	if err != nil || rep.Conforms() {
		t.Errorf("11 is above the maximum: %v, %v", rep.Value(), err)
	}
}

func TestValidateRejectsHostNonReals(t *testing.T) {
	pkg := decode(t, hostPkg)
	huge := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(999999), nil))
	for name, n := range map[string]any{"Rat 10^999999": huge, "+Inf": math.Inf(1), "1e300": 1e300} {
		if _, err := Validate(pkg, "P", map[string]any{"n": n}); err == nil || err.Code != "NUMERIC_OVERFLOW" {
			t.Errorf("%s: got %v, want NUMERIC_OVERFLOW", name, err)
		}
	}
}
