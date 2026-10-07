// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import (
	"encoding/json"
	"testing"
)

func TestBorrowOnly(t *testing.T) {
	parse := func(s string) any {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := []struct {
		expr string
		want bool
	}{
		{`["has", {"$v":"x"}, "k"]`, true},
		{`["len", {"$v":"x"}]`, true},
		{`["eq", {"$v":"x"}, {"$v":"y"}]`, true},
		{`["select", ["has", {"$v":"x"}, "k"], 1, 2]`, true},
		{`1`, true},                         // unused
		{`{"$v":"x"}`, false},               // returned as the result
		{`["get", {"$v":"x"}, "k"]`, false}, // returns a member, which may be composite
		{`["get_or", {"$v":"x"}, "k", 0]`, false},
		{`["list", {"$v":"x"}]`, false}, // keeps it in a new list
		{`["and", ["has", {"$v":"x"}, "k"], ["list", {"$v":"x"}]]`, false},
		{`["and", ["has", {"$v":"x"}, "a"], ["has", {"$v":"x"}, "b"]]`, true}, // several safe uses
		{`["and", ["has", {"$v":"x"}, "a"], ["eq", ["get", {"$v":"x"}, "a"], 1]]`, true},
		{`["len", ["keys", ["get_or", ["get_or", {"$v":"x"}, "p", {}], "q", {}]]]`, true},
		{`["gt", ["len", ["get", {"$v":"x"}, "a"]], 0]`, true},
		{`["select", true, ["get", {"$v":"x"}, "a"], 0]`, false},
		{`["list", ["get", {"$v":"x"}, "a"]]`, false},
		{`["eq", ["select", true, {"$v":"x"}, null], 1]`, true},
		{`["get_or", {"$v":"y"}, "k", {"$v":"x"}]`, false},
		{`["select", ["has", {"$v":"x"}, "a"], {"$v":"x"}, null]`, false}, // select can return a branch as it is
	}
	for _, c := range cases {
		if got := borrowOnly(parse(c.expr), "x"); got != c.want {
			t.Errorf("borrowOnly(%s) = %v, want %v", c.expr, got, c.want)
		}
	}
}

// A binding over a state map that grows every iteration and is read only by
// `has`, so it is borrowed: the answers must be those a copy would give, as
// the map grows between reads. (Case shape from the parallel PF3 draft.)
func TestBorrowedHasOnAGrowingStateMap(t *testing.T) {
	const pkg = `{"jaxson":"1.0","inputSchema":{"type":"any"},"outputSchema":{"type":"any"},"input":{},
	  "program":[
	    {"op":"set","path":["state","m"],"value":{"$lit":{}}},
	    {"op":"set","path":["state","seen"],"value":{"$lit":[]}},
	    {"op":"for","in":{"$lit":["a","b","a","c","b"]},"as":"k","do":[
	      {"op":"append","path":["state","seen"],"value":{"$compute":{
	        "with":{"m":{"$path":["state","m"]},"k":{"$path":["local","k"]}},
	        "expr":["has",{"$v":"m"},{"$v":"k"}]}}},
	      {"op":"set","path":["state","m",{"$path":["local","k"]}],"value":{"$lit":true}}]},
	    {"op":"set","path":["output"],"value":{"$path":["state","seen"]}}]}`
	out, err := RunJSON([]byte(pkg))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Show(out), `[false,false,true,false,true]`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
