// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import "testing"

// A float64 (a number beyond the strict range in relaxed mode, Jaxson core
// design section 2) renders as a scalar, written as a JSON number.
func TestFloatsRenderAsScalars(t *testing.T) {
	if !isScalar(1e300) {
		t.Error("a float is a scalar")
	}
	if got := scalarText(1e300); got != "1e+300" {
		t.Errorf("scalarText(1e300) = %q, want 1e+300", got)
	}
}
