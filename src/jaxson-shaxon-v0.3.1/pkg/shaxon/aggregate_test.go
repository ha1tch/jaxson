// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

import (
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

const aggInput = `{"events":[
  {"actor":"a","mb":10},{"actor":"b","mb":7},{"actor":"a","mb":"2.5"},{"actor":"a","mb":4}]}`

// agg builds a package whose program is the given instructions followed by
// "output := state.r".
func agg(input, prog string) string {
	return `{"shaxon":"3.1","input":` + input + `,"program":[` + prog + `,
	  {"op":"set","path":["output"],"value":{"$path":["state","r"]}}]}`
}

func TestAggregateAllMeasures(t *testing.T) {
	// "mb" is a string in one event; filter to numbers by actor first, with
	// the string one excluded through where.
	res := mustRun(t, agg(`{"events":[
	  {"actor":"a","mb":10},{"actor":"b","mb":7},{"actor":"a","mb":2.5},{"actor":"a","mb":4}]}`,
		`{"op":"aggregate","in":{"$path":["input","events"]},"as":"e","into":["state","r"],
		  "where":{"$compute":{"with":{"a":{"$path":["local","e","actor"]}},"expr":["eq",{"$v":"a"},"a"]}},
		  "measures":{"n":{"count":true},
		    "total":{"sum":{"$path":["local","e","mb"]}},
		    "lo":{"min":{"$path":["local","e","mb"]}},
		    "hi":{"max":{"$path":["local","e","mb"]}}}}`))
	if got, want := jaxson.Show(res.Output), `{"hi":10,"lo":2.5,"n":3,"total":16.5}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestAggregateOverNothing(t *testing.T) {
	res := mustRun(t, agg(`{"events":[]}`,
		`{"op":"aggregate","in":{"$path":["input","events"]},"as":"e","into":["state","r"],
		  "measures":{"n":{"count":true},"s":{"sum":{"$path":["local","e","mb"]}},
		              "lo":{"min":{"$path":["local","e","mb"]}},"hi":{"max":{"$path":["local","e","mb"]}}}}`))
	if got, want := jaxson.Show(res.Output), `{"hi":null,"lo":null,"n":0,"s":0}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestAggregateWhereExcludesEverything(t *testing.T) {
	res := mustRun(t, agg(`{"xs":[1,2,3]}`,
		`{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],
		  "where":{"$compute":{"with":{},"expr":["eq",1,2]}},
		  "measures":{"n":{"count":true},"hi":{"max":{"$path":["local","x"]}}}}`))
	if got, want := jaxson.Show(res.Output), `{"hi":null,"n":0}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

// The sugar is only sugar: the same fold written out by hand gives the same
// output and the same step total.
func TestAggregateEqualsTheHandWrittenFold(t *testing.T) {
	const in = `{"xs":[3,9,1,7]}`
	sugar := mustRun(t, agg(in,
		`{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],
		  "measures":{"n":{"count":true},"s":{"sum":{"$path":["local","x"]}}}}`))
	hand := mustRun(t, agg(in,
		`{"op":"set","path":["state","r"],"value":{"$lit":{"n":0,"s":0}}},
		 {"op":"for","in":{"$path":["input","xs"]},"as":"x","do":[
		   {"op":"set","path":["state","r","n"],"value":{"$compute":{"with":{"c":{"$path":["state","r","n"]}},"expr":["add",{"$v":"c"},1]}}},
		   {"op":"set","path":["state","r","s"],"value":{"$compute":{"with":{"s":{"$path":["state","r","s"]},"x":{"$path":["local","x"]}},"expr":["add",{"$v":"s"},{"$v":"x"}]}}}]}`))
	if jaxson.Show(sugar.Output) != jaxson.Show(hand.Output) || jaxson.Show(sugar.Output) != `{"n":4,"s":20}` {
		t.Fatalf("sugar %s, hand %s", jaxson.Show(sugar.Output), jaxson.Show(hand.Output))
	}
	if sugar.Steps != hand.Steps {
		t.Fatalf("sugar cost %d steps, hand-written fold %d", sugar.Steps, hand.Steps)
	}
}

func TestAggregateInsideIfAndFor(t *testing.T) {
	res := mustRun(t, `{"shaxon":"3.1","input":{"rows":[[1,2],[3,4,5]]},"program":[
	  {"op":"set","path":["state","r"],"value":{"$lit":[]}},
	  {"op":"for","in":{"$path":["input","rows"]},"as":"row","do":[
	    {"op":"if","cond":{"$compute":{"with":{},"expr":["eq",1,1]}},"then":[
	      {"op":"aggregate","in":{"$path":["local","row"]},"as":"x","into":["state","t"],
	       "measures":{"s":{"sum":{"$path":["local","x"]}}}},
	      {"op":"append","path":["state","r"],"value":{"$path":["state","t","s"]}}]}]},
	  {"op":"set","path":["output"],"value":{"$path":["state","r"]}}]}`)
	if got, want := jaxson.Show(res.Output), `[3,12]`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestAggregateDoesNotChangeTheCallersPackage(t *testing.T) {
	pkg, perr := jaxson.ParseJSON([]byte(agg(`{"xs":[1]}`,
		`{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],"measures":{"n":{"count":true}}}`)))
	if perr != nil {
		t.Fatal(perr)
	}
	before := jaxson.Show(pkg)
	for i := 0; i < 2; i++ {
		if _, err := Run(pkg.(map[string]any)); err != nil {
			t.Fatal(err)
		}
	}
	if after := jaxson.Show(pkg); after != before {
		t.Fatalf("Run changed the package:\n%s\n%s", before, after)
	}
	if !strings.Contains(before, `"aggregate"`) {
		t.Fatal("test package lost its aggregate")
	}
}

func TestAggregateStepLimitStillBinds(t *testing.T) {
	_, err := runSrc(t, `{"shaxon":"3.1","input":{"xs":[1,2,3,4,5,6,7,8]},"limits":{"steps":10},"program":[
	  {"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],
	   "measures":{"n":{"count":true},"s":{"sum":{"$path":["local","x"]}}}}]}`)
	wantErr(t, "step limit", err, "RESOURCE_ERROR", "STEPS")
}

func TestAggregateSumOfNonNumbersIsATypeError(t *testing.T) {
	_, err := runSrc(t, agg(`{"xs":["a"]}`,
		`{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],
		  "measures":{"s":{"sum":{"$path":["local","x"]}}}}`))
	if err == nil || err.Cat != "EXECUTION_ERROR" {
		t.Fatalf("got %v, want an EXECUTION_ERROR", err)
	}
	_, err = runSrc(t, agg(`{"xs":["a"]}`,
		`{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],
		  "measures":{"m":{"min":{"$path":["local","x"]}}}}`))
	if err == nil || err.Cat != "EXECUTION_ERROR" {
		t.Fatalf("min of a string: got %v, want an EXECUTION_ERROR", err)
	}
}

func TestAggregateMalformedForms(t *testing.T) {
	base := `"in":{"$path":["input","xs"]},"as":"x","into":["state","r"]`
	cases := map[string]string{
		"missing in":         `{"op":"aggregate","as":"x","into":["state","r"],"measures":{"n":{"count":true}}}`,
		"missing measures":   `{"op":"aggregate",` + base + `}`,
		"empty measures":     `{"op":"aggregate",` + base + `,"measures":{}}`,
		"unknown field":      `{"op":"aggregate",` + base + `,"measures":{"n":{"count":true}},"group":1}`,
		"unknown aggregate":  `{"op":"aggregate",` + base + `,"measures":{"n":{"avg":1}}}`,
		"two aggregates":     `{"op":"aggregate",` + base + `,"measures":{"n":{"count":true,"sum":1}}}`,
		"count takes true":   `{"op":"aggregate",` + base + `,"measures":{"n":{"count":false}}}`,
		"measure not object": `{"op":"aggregate",` + base + `,"measures":{"n":true}}`,
		"empty into":         `{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":[],"measures":{"n":{"count":true}}}`,
		"as not a string":    `{"op":"aggregate","in":{"$path":["input","xs"]},"as":1,"into":["state","r"],"measures":{"n":{"count":true}}}`,
	}
	for name, ins := range cases {
		_, err := runSrc(t, agg(`{"xs":[1]}`, ins))
		wantErr(t, name, err, "PROGRAM_ERROR", "")
	}
}

func TestAggregateIntoInputAndShadowedBindingAreRefused(t *testing.T) {
	_, err := runSrc(t, agg(`{"xs":[1]}`,
		`{"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["input","r"],"measures":{"n":{"count":true}}}`))
	wantErr(t, "into input", err, "PROGRAM_ERROR", "")
	_, err = runSrc(t, agg(`{"xs":[1]}`,
		`{"op":"for","in":{"$path":["input","xs"]},"as":"x","do":[
		   {"op":"aggregate","in":{"$path":["input","xs"]},"as":"x","into":["state","r"],"measures":{"n":{"count":true}}}]}`))
	wantErr(t, "shadowed binding", err, "PROGRAM_ERROR", "")
}
