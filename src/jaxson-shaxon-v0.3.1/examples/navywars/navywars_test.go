// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package navywars

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const reference = "../../../../examples/game/navywars.json"

func loadReference(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(reference)
	if err != nil {
		t.Skipf("reference package not present: %v", err)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatal(err)
	}
	return jaxson.Normalize(v).(map[string]any)
}

// The acceptance test for the builder: the Go port must yield exactly the
// document the Python generator produced.
func TestPortMatchesPythonGeneratedPackage(t *testing.T) {
	want := loadReference(t)
	got, err := Package().Map()
	if err != nil {
		t.Fatal(err)
	}
	if jaxson.Equal(want, got) {
		return
	}
	for _, k := range jaxson.SortedKeys(want) {
		if !jaxson.Equal(want[k], got[k]) {
			t.Errorf("member %q differs\n got: %.400s\nwant: %.400s", k, jaxson.Show(got[k]), jaxson.Show(want[k]))
		}
	}
	t.Fatal("port differs from navywars.json")
}

func TestPortPlaysASeedZeroGameToVictory(t *testing.T) {
	pkg, err := Package().Map()
	if err != nil {
		t.Fatal(err)
	}
	out, e := jaxson.Run(pkg)
	if e != nil {
		t.Fatalf("new game: %v", e)
	}
	g := out.(map[string]any)["game"]
	if g == nil {
		t.Fatal("no game in output")
	}
	if !jaxson.Equal(g.(map[string]any)["status"], "playing") {
		t.Fatalf("unexpected status %s", jaxson.Show(g))
	}
}
