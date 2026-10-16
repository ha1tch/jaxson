// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxtools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func writeTemp(t *testing.T, name, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// The loaders read with the language's own reader, so the numeric mode decides
// what a number beyond the Real limits is (Jaxson core design, section 2).
func TestLoadPackageMode(t *testing.T) {
	p := writeTemp(t, "p.json", `{"jaxson":"1.0","input":{"n":1e300},"program":[]}`)
	if _, err := LoadPackage(p); err == nil || !strings.Contains(err.Error(), "number out of range") {
		t.Errorf("strict: %v, want number out of range", err)
	}
	pkg, err := LoadPackageMode(p, jaxson.Relaxed)
	if err != nil {
		t.Fatal(err)
	}
	if got := jaxson.Show(pkg["input"]); got != `{"n":1e+300}` {
		t.Errorf("relaxed input: %s", got)
	}
	dup := writeTemp(t, "dup.json", `{"jaxson":"1.0","jaxson":"1.0"}`)
	if _, err := LoadPackage(dup); err == nil {
		t.Error("a duplicate key must be an error")
	}
	arr := writeTemp(t, "arr.json", `[1]`)
	if _, err := LoadPackage(arr); err == nil {
		t.Error("a package must be an object")
	}
}

func TestLoadTurnsMode(t *testing.T) {
	p := writeTemp(t, "t.json", `[{"a":1e300},{"a":2}]`)
	if _, err := LoadTurns(p); err == nil {
		t.Error("strict must refuse 1e300")
	}
	turns, err := LoadTurnsMode(p, jaxson.Relaxed)
	if err != nil || len(turns) != 2 || jaxson.Show(turns[0]["a"]) != "1e+300" {
		t.Errorf("relaxed: %v, %v", turns, err)
	}
	for name, text := range map[string]string{"not an array": `{"a":1}`, "not objects": `[1]`} {
		if _, err := LoadTurns(writeTemp(t, "x.json", text)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
