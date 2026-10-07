// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Pins the five authorisation examples in examples/shaxon/authz. Each
// <name>.json is a package whose input is a trail of events plus a request;
// <name>.cases.json lists inputs (merged over the package's own, member by
// member) with the expected decision and the reasons behind it, or the
// error the run must fail closed with.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func readValue(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	v, perr := jaxson.ParseJSON(raw)
	if perr != nil {
		t.Fatalf("%s: %v", path, perr)
	}
	return v
}

// decide runs the package on the case's input and returns the decision, the
// reasons in order, or the error as "CATEGORY/CODE".
func decide(t *testing.T, pkgPath string, override map[string]any) (decision string, reasons []string, errText string, shown string) {
	t.Helper()
	pkg := readValue(t, pkgPath).(map[string]any)
	in := pkg["input"].(map[string]any)
	for k, v := range override {
		in[k] = v
	}
	res, err := Run(pkg)
	if err != nil {
		return "", nil, fmt.Sprintf("%s/%s", err.Cat, err.Code), ""
	}
	out := res.Output.(map[string]any)
	for _, f := range out["findings"].([]any) {
		fm := f.(map[string]any)
		switch {
		case fm["constraintId"] != nil:
			reasons = append(reasons, fm["constraintId"].(string))
		case fm["code"] != nil:
			reasons = append(reasons, fm["code"].(string))
		default:
			reasons = append(reasons, "?")
		}
	}
	return out["decision"].(string), reasons, "", jaxson.Show(res.Output)
}

func TestAuthzExamples(t *testing.T) {
	files, err := filepath.Glob("../../examples/shaxon/authz/*.cases.json")
	if err != nil || len(files) != 5 {
		t.Fatalf("want the five example case files, got %v (%v)", files, err)
	}
	for _, cf := range files {
		pkgPath := strings.TrimSuffix(cf, ".cases.json") + ".json"
		name := filepath.Base(strings.TrimSuffix(cf, ".cases.json"))
		cases := readValue(t, cf).([]any)
		var sawAllow, sawDeny bool
		for _, c := range cases {
			cm := c.(map[string]any)
			cname := cm["name"].(string)
			expect := cm["expect"].(map[string]any)
			override, _ := cm["input"].(map[string]any)

			decision, reasons, errText, shown := decide(t, pkgPath, override)
			label := name + ": " + cname
			if want, ok := expect["error"].(string); ok {
				if errText != want {
					t.Errorf("%s: error %q, want %q", label, errText, want)
				}
				sawDeny = true
				continue
			}
			if errText != "" {
				t.Errorf("%s: failed with %s", label, errText)
				continue
			}
			var wantReasons []string
			for _, r := range expect["reasons"].([]any) {
				wantReasons = append(wantReasons, r.(string))
			}
			if decision != expect["decision"].(string) || strings.Join(reasons, ",") != strings.Join(wantReasons, ",") {
				t.Errorf("%s: got %s %v, want %s %v", label, decision, reasons, expect["decision"], wantReasons)
			}
			sawAllow = sawAllow || decision == "allow"
			sawDeny = sawDeny || decision == "deny"

			// Deterministic: a second run gives the identical output.
			if _, _, _, again := decide(t, pkgPath, override); again != shown {
				t.Errorf("%s: two runs differ", label)
			}
		}
		if !sawAllow || !sawDeny {
			t.Errorf("%s: cases should include both an allow and a deny", name)
		}
	}
}
