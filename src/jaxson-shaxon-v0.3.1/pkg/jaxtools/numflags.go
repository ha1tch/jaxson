// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

// The numeric mode switches of the command-line tools (Jaxson core design,
// section 2): -strict, the default, and -relaxed, which overrides it.

import (
	"flag"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// NumericFlags adds -strict and -relaxed to fs and returns a function that,
// called after fs.Parse, gives the mode they ask for. It returns an error for
// a contradictory pair of switches and for a mode this runtime does not
// implement, so a tool has one check to make before it runs anything.
func NumericFlags(fs *flag.FlagSet) func() (jaxson.NumericMode, error) {
	strict := fs.Bool("strict", true, "strict numeric mode, the default: every number is a Real (at most 38 digits and 200 after the point)")
	relaxed := fs.Bool("relaxed", false, "relaxed numeric mode, overriding -strict: a number beyond the Real limits is an IEEE float64; the standard does not guarantee determinism across architectures, though this runtime's results are reproducible")
	return func() (jaxson.NumericMode, error) {
		strictSet := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "strict" {
				strictSet = true
			}
		})
		mode, err := jaxson.ModeFromFlags(*strict, *relaxed, strictSet)
		return mode, err
	}
}
