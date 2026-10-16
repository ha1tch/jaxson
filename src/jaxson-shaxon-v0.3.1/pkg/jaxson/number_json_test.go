// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import (
	"encoding/json"
	"testing"
)

func TestNumberUnmarshalRoundTrip(t *testing.T) {
	type doc struct {
		N Number `json:"n"`
		M Number `json:"m"`
	}
	in := doc{N: mustNum(t, "0.30"), M: mustNum(t, "123456789012345678901234567890.5")}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out doc
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
	if out.N.Cmp(in.N) != 0 || out.M.Cmp(in.M) != 0 || out.N.String() != "0.3" {
		t.Errorf("round trip: %s gave %v and %v", raw, out.N, out.M)
	}
}

func TestNumberUnmarshalRejectsWhatIsNotAReal(t *testing.T) {
	for _, s := range []string{`"1.5"`, `true`, `[1]`, `1e999999`, `-1e-999999`, `123456789012345678901234567890123456789`} {
		var n Number
		if err := json.Unmarshal([]byte(s), &n); err == nil {
			t.Errorf("Unmarshal(%s) accepted %v", s, n)
		}
	}
	n := mustNum(t, "5")
	if err := json.Unmarshal([]byte(`null`), &n); err != nil || n.String() != "5" {
		t.Errorf("null should leave the value alone: %v, %v", n, err)
	}
}

func TestNumberAsMapKeyRoundTrips(t *testing.T) {
	raw, err := json.Marshal(map[Number]int{mustNum(t, "2.50"): 1})
	if err != nil || string(raw) != `{"2.5":1}` {
		t.Fatalf("%s, %v", raw, err)
	}
	var back map[Number]int
	if err := json.Unmarshal(raw, &back); err != nil || len(back) != 1 {
		t.Fatalf("%v, %v", back, err)
	}
	for k, v := range back {
		if k.Cmp(mustNum(t, "2.5")) != 0 || v != 1 {
			t.Errorf("got %v: %d", k, v)
		}
	}
}
