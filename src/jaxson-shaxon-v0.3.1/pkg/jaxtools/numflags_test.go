// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import (
	"errors"
	"flag"
	"io"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func parseNumeric(t *testing.T, args ...string) (jaxson.NumericMode, error) {
	t.Helper()
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	read := NumericFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return read()
}

func TestNumericFlags(t *testing.T) {
	for _, c := range []struct {
		name    string
		args    []string
		mode    jaxson.NumericMode
		wantErr error
		anyErr  bool
	}{
		{name: "default is strict", mode: jaxson.Strict},
		{name: "-strict", args: []string{"-strict"}, mode: jaxson.Strict},
		{name: "-relaxed overrides the default", args: []string{"-relaxed"}, mode: jaxson.Relaxed},
		{name: "both is an error", args: []string{"-strict", "-relaxed"}, anyErr: true},
		{name: "-strict=false names no mode", args: []string{"-strict=false"}, anyErr: true},
	} {
		mode, err := parseNumeric(t, c.args...)
		switch {
		case c.wantErr != nil && !errors.Is(err, c.wantErr):
			t.Errorf("%s: got %v, want %v", c.name, err, c.wantErr)
		case c.wantErr == nil && !c.anyErr && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.anyErr && err == nil:
			t.Errorf("%s: no error", c.name)
		}
		if !c.anyErr && mode != c.mode {
			t.Errorf("%s: mode %v, want %v", c.name, mode, c.mode)
		}
	}
}

func TestNumericModeStrings(t *testing.T) {
	if jaxson.Strict.String() != "strict" || jaxson.Relaxed.String() != "relaxed" {
		t.Error("mode names")
	}
}
