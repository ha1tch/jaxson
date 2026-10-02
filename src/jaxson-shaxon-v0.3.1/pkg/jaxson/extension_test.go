// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson_test

// Phase 1 extension-point tests. These are written from outside the
// package on purpose: they use only what a host package (Shaxon) can
// actually reach, so a missing export shows up here as a compile error
// rather than later as a surprise in Phase 2.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func dec(t testing.TB, s string) any {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader([]byte(s)))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		t.Fatalf("bad test JSON: %v", err)
	}
	return jaxson.Normalize(v)
}

// probeTable returns the core table plus one host instruction, "probe",
// whose Check reads Checker.Host and whose Exec evaluates its operand
// through the exported Machine.Eval and charges a step.
func probeTable(hostSeen *[]any, evalSeen *[]string) map[string]jaxson.InstructionDef {
	tbl := jaxson.CoreInstructions()
	tbl["probe"] = jaxson.InstructionDef{
		Name: "probe", Req: []string{"value"},
		Check: func(in map[string]any, c *jaxson.Checker) {
			*hostSeen = append(*hostSeen, c.Host)
		},
		Exec: func(m *jaxson.Machine, in map[string]any) {
			m.Step()
			*evalSeen = append(*evalSeen, jaxson.Show(m.Eval(in["value"])))
		},
	}
	return tbl
}

func TestHostInstructionIsCheckedAndExecuted(t *testing.T) {
	var hostSeen []any
	var evalSeen []string
	tbl := probeTable(&hostSeen, &evalSeen)
	prog := dec(t, `[
	  {"op":"set","path":["state","n"],"value":7},
	  {"op":"probe","value":{"$path":["state","n"]}}
	]`)

	if e := jaxson.CheckProgramHost(prog, tbl, "host-state"); e != nil {
		t.Fatalf("CheckProgramHost: %v", e)
	}
	if len(hostSeen) != 1 || hostSeen[0] != "host-state" {
		t.Fatalf("Checker.Host not delivered to the host Check: %v", hostSeen)
	}

	m := jaxson.NewMachine(nil, map[string]any{}, nil, 100, tbl, nil)
	m.RunProgram(prog.([]any))
	if !reflect.DeepEqual(evalSeen, []string{"7"}) {
		t.Fatalf("host Exec saw %v, want [7]", evalSeen)
	}
}

func TestHostInstructionGetsGenericFieldChecks(t *testing.T) {
	var h []any
	var e []string
	tbl := probeTable(&h, &e)
	cases := map[string]string{
		"missing required": `[{"op":"probe"}]`,
		"unknown field":    `[{"op":"probe","value":1,"extra":2}]`,
	}
	for name, src := range cases {
		if err := jaxson.CheckProgramHost(dec(t, src), tbl, nil); err == nil || err.Cat != "PROGRAM_ERROR" {
			t.Errorf("%s: got %v, want PROGRAM_ERROR", name, err)
		}
	}
}

func TestUnregisteredHostInstructionIsRejectedByCore(t *testing.T) {
	err := jaxson.CheckProgram(dec(t, `[{"op":"probe","value":1}]`), jaxson.CoreInstructions())
	if err == nil || err.Cat != "PROGRAM_ERROR" {
		t.Fatalf("core table accepted a host instruction: %v", err)
	}
}

func TestHostStepsShareTheMachineLimit(t *testing.T) {
	var h []any
	var e []string
	tbl := probeTable(&h, &e)
	prog := dec(t, `[{"op":"probe","value":1},{"op":"probe","value":2},{"op":"probe","value":3}]`)
	m := jaxson.NewMachine(nil, map[string]any{}, nil, 3, tbl, nil)
	var got *jaxson.Err
	func() {
		defer func() {
			if r := recover(); r != nil {
				got, _ = r.(*jaxson.Err)
			}
		}()
		m.RunProgram(prog.([]any))
	}()
	if got == nil || got.Cat != "RESOURCE_ERROR" {
		t.Fatalf("expected RESOURCE_ERROR from shared step counter, got %v", got)
	}
}

func TestOnMutateReportsEveryMutationInstruction(t *testing.T) {
	prog := dec(t, `[
	  {"op":"set","path":["state","xs"],"value":[1,2]},
	  {"op":"append","path":["state","xs"],"value":3},
	  {"op":"insert","path":["state","xs"],"at":0,"value":0},
	  {"op":"delete","path":["state","xs",0]},
	  {"op":"set","path":["state","o"],"value":{"k":1}},
	  {"op":"delete","path":["state","o","k"]},
	  {"op":"set","path":["output"],"value":1}
	]`)
	var log []string
	m := jaxson.NewMachine(nil, map[string]any{}, nil, 1000, nil, nil)
	m.OnMutate = func(root string, segs []any) {
		log = append(log, fmt.Sprintf("%s %v", root, segs))
	}
	m.RunProgram(prog.([]any))
	want := []string{
		"state [xs]", // set
		"state [xs]", // append
		"state [xs]", // insert
		"state [xs]", // delete of an element reports the container
		"state [o]",  // set
		"state [o]",  // delete of a member reports the container
		"output []",  // set of a whole root
	}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("mutation log\n got: %q\nwant: %q", log, want)
	}
}

func TestOnMutateNotCalledOnFailedMutation(t *testing.T) {
	prog := dec(t, `[{"op":"append","path":["state","x"],"value":1}]`)
	n := 0
	m := jaxson.NewMachine(nil, map[string]any{"x": "not an array"}, nil, 100, nil, nil)
	m.OnMutate = func(string, []any) { n++ }
	func() {
		defer func() { _ = recover() }()
		m.RunProgram(prog.([]any))
	}()
	if n != 0 {
		t.Fatalf("OnMutate fired %d times for a failed append", n)
	}
}

func TestCoreTablesAreFreshPerCall(t *testing.T) {
	a, b := jaxson.CoreInstructions(), jaxson.CoreInstructions()
	delete(a, "set")
	a["extra"] = jaxson.InstructionDef{Name: "extra"}
	if _, ok := b["set"]; !ok || len(b) != 8 {
		t.Fatalf("CoreInstructions() shares state between calls (len=%d)", len(b))
	}
	x, y := jaxson.CoreOperators(), jaxson.CoreOperators()
	for k := range x {
		delete(x, k)
		break
	}
	if len(y) == len(x) {
		t.Fatalf("CoreOperators() shares state between calls")
	}
}

func TestSchemaPipelineStagesAreReachable(t *testing.T) {
	if e := jaxson.CheckSchema(dec(t, `{"type":"nonsense"}`)); e == nil {
		t.Fatal("CheckSchema accepted a malformed schema")
	}
	s := dec(t, `{"type":"number","int":true,"min":0}`)
	if e := jaxson.CheckSchema(s); e != nil {
		t.Fatalf("CheckSchema rejected a good schema: %v", e)
	}
	if msg := jaxson.Validate(s, dec(t, `3`)); msg != "" {
		t.Fatalf("Validate rejected 3: %s", msg)
	}
	if msg := jaxson.Validate(s, dec(t, `-1`)); msg == "" {
		t.Fatal("Validate accepted -1")
	}
}

const concurrentPkg = `{
  "jaxson":"1.0","input":{"xs":[1,2,3,4]},
  "inputSchema":{"type":"object","fields":{"xs":{"type":"array","items":{"type":"number"}}},"required":["xs"]},
  "program":[
    {"op":"set","path":["state","t"],"value":0},
    {"op":"for","in":{"$path":["input","xs"]},"as":"x","do":[
      {"op":"set","path":["state","t"],"value":{"$compute":{"with":{"a":{"$path":["state","t"]},"b":{"$path":["local","x"]}},"expr":["add",{"$v":"a"},{"$v":"b"}]}}}]},
    {"op":"set","path":["output"],"value":{"$tpl":{"total":{"$path":["state","t"]}}}}
  ],
  "outputSchema":{"type":"object","fields":{"total":{"type":"number"}},"required":["total"]}
}`

func TestConcurrentRunIsRaceFree(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				p := dec(t, concurrentPkg).(map[string]any)
				out, e := jaxson.Run(p)
				if e != nil {
					errs <- e.Error()
					return
				}
				if jaxson.Show(out) != `{"total":10}` && jaxson.Show(out) != "{\"total\": 10}" {
					errs <- "unexpected output " + jaxson.Show(out)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}
