// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

import (
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// Reads of `input` and `local` are shared, not copied, and so are literals
// (see Machine.read). These tests pin the other half of that contract: what
// is stored in state or output is private, so no write can reach the input,
// a loop variable, or the text of the program.

func runAlias(t *testing.T, src string) (map[string]any, any) {
	t.Helper()
	v, perr := jaxson.ParseJSON([]byte(src))
	if perr != nil {
		t.Fatal(perr)
	}
	pkg := v.(map[string]any)
	res, err := Run(pkg)
	out := res.Output
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return pkg, out
}

func TestWritingACopyOfTheInputDoesNotChangeTheInput(t *testing.T) {
	pkg, out := runAlias(t, `{"shaxon":"3.1","input":{"xs":[{"a":1},{"a":2}]},"program":[
	  {"op":"set","path":["state","c"],"value":{"$path":["input","xs"]}},
	  {"op":"set","path":["state","c",0,"a"],"value":{"$lit":99}},
	  {"op":"set","path":["output"],"value":{"$lit":{}}},
	  {"op":"set","path":["output","c"],"value":{"$path":["state","c"]}},
	  {"op":"set","path":["output","orig"],"value":{"$path":["input","xs"]}}]}`)
	if got, want := jaxson.Show(out), `{"c":[{"a":99},{"a":2}],"orig":[{"a":1},{"a":2}]}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got, want := jaxson.Show(pkg["input"]), `{"xs":[{"a":1},{"a":2}]}`; got != want {
		t.Fatalf("the caller's input changed: %s", got)
	}
}

func TestALoopVariableCopiedToStateIsPrivate(t *testing.T) {
	pkg, out := runAlias(t, `{"shaxon":"3.1","input":{"xs":[{"a":1},{"a":2}]},"program":[
	  {"op":"set","path":["state","seen"],"value":{"$lit":[]}},
	  {"op":"for","in":{"$path":["input","xs"]},"as":"e","do":[
	    {"op":"append","path":["state","seen"],"value":{"$path":["local","e"]}},
	    {"op":"set","path":["state","seen",0,"a"],"value":{"$lit":7}}]},
	  {"op":"set","path":["output"],"value":{"$path":["state","seen"]}}]}`)
	if got, want := jaxson.Show(out), `[{"a":7},{"a":2}]`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if got, want := jaxson.Show(pkg["input"]), `{"xs":[{"a":1},{"a":2}]}`; got != want {
		t.Fatalf("the caller's input changed: %s", got)
	}
}

func TestALiteralIsNotChangedByWritingToACopyOfIt(t *testing.T) {
	// The second pass of the loop must see the literal as written, not as the
	// first pass left it.
	_, out := runAlias(t, `{"shaxon":"3.1","input":{"n":[1,2]},"program":[
	  {"op":"set","path":["state","all"],"value":{"$lit":[]}},
	  {"op":"for","in":{"$path":["input","n"]},"as":"i","do":[
	    {"op":"set","path":["state","o"],"value":{"$lit":{"l":[]}}},
	    {"op":"append","path":["state","o","l"],"value":{"$path":["local","i"]}},
	    {"op":"append","path":["state","all"],"value":{"$path":["state","o"]}}]},
	  {"op":"set","path":["output"],"value":{"$path":["state","all"]}}]}`)
	if got, want := jaxson.Show(out), `[{"l":[1]},{"l":[2]}]`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestALoopOverStateIteratesASnapshot(t *testing.T) {
	_, out := runAlias(t, `{"shaxon":"3.1","input":{},"program":[
	  {"op":"set","path":["state","xs"],"value":{"$lit":[{"a":1},{"a":2}]}},
	  {"op":"set","path":["state","log"],"value":{"$lit":[]}},
	  {"op":"for","in":{"$path":["state","xs"]},"as":"e","do":[
	    {"op":"set","path":["state","xs",1,"a"],"value":{"$lit":50}},
	    {"op":"append","path":["state","log"],"value":{"$path":["local","e","a"]}}]},
	  {"op":"set","path":["output"],"value":{"$path":["state","log"]}}]}`)
	if got, want := jaxson.Show(out), `[1,2]`; got != want {
		t.Fatalf("a loop over state saw its own writes: got %s, want %s", got, want)
	}
}

func TestTheSamePackageRunTwiceGivesTheSameResult(t *testing.T) {
	v, _ := jaxson.ParseJSON([]byte(`{"shaxon":"3.1","input":{"xs":[{"a":1}]},"program":[
	  {"op":"set","path":["state","o"],"value":{"$lit":{"k":[1,2]}}},
	  {"op":"append","path":["state","o","k"],"value":{"$path":["input","xs"]}},
	  {"op":"set","path":["output"],"value":{"$path":["state","o"]}}]}`))
	pkg := v.(map[string]any)
	first, err := Run(pkg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if jaxson.Show(first.Output) != jaxson.Show(second.Output) || jaxson.Show(first.Output) != `{"k":[1,2,[{"a":1}]]}` {
		t.Fatalf("first %s, second %s", jaxson.Show(first.Output), jaxson.Show(second.Output))
	}
}
