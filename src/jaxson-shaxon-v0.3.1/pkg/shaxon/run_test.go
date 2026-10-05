// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 5 tests: shaxon.Run, the version gate and the pipeline order of
// core section 9 (decisions R1..R6).

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

func runSrc(t *testing.T, src string) (Result, *jaxson.Err) {
	t.Helper()
	res, err := RunJSON([]byte(src))
	if err != nil && err.Cat == "PARSE_ERROR" {
		t.Fatalf("fixture is not valid: %v", err)
	}
	return res, err
}

func mustRun(t *testing.T, src string) Result {
	t.Helper()
	res, err := runSrc(t, src)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	return res
}

func wantErr(t *testing.T, name string, err *jaxson.Err, cat, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: succeeded, want %s/%s", name, cat, code)
	}
	if err.Cat != cat || err.Code != code {
		t.Fatalf("%s: got %s/%s (%s), want %s/%s", name, err.Cat, err.Code, err.Msg, cat, code)
	}
}

// ------------------------------------------------------------ the happy path

func TestRunMinimalPackage(t *testing.T) {
	res := mustRun(t, `{"shaxon":"3.1","input":{"n":4},
	  "program":[{"op":"set","path":["output"],"value":{"$path":["input","n"]}}]}`)
	if got := jaxson.Show(res.Output); got != "4" {
		t.Fatalf("output = %s, want 4", got)
	}
	if res.Report != nil {
		t.Fatalf("report present though no report-mode entry ran")
	}
	if res.Steps <= 0 {
		t.Fatalf("Steps = %d", res.Steps)
	}
}

func TestRunAcceptsAnAlreadyDecodedPackage(t *testing.T) {
	v, perr := jaxson.ParseJSON([]byte(`{"shaxon":"3.1","program":[{"op":"set","path":["output"],"value":1}]}`))
	if perr != nil {
		t.Fatal(perr)
	}
	res, err := Run(v.(map[string]any))
	if err != nil || jaxson.Show(res.Output) != "1" {
		t.Fatalf("got %v, %v", res.Output, err)
	}
}

func TestRunJSONRejectsDuplicateKeysAndNonObjects(t *testing.T) {
	_, err := RunJSON([]byte(`{"shaxon":"3.1","shaxon":"3.1","program":[]}`))
	wantErr(t, "duplicate key", err, "PARSE_ERROR", "")
	_, err = RunJSON([]byte(`{"shapes":{"A":{"kind":"string"},"A":{"kind":"number"}},"shaxon":"3.1","program":[]}`))
	wantErr(t, "duplicate shape name", err, "PARSE_ERROR", "")
	_, err = RunJSON([]byte(`[1,2]`))
	wantErr(t, "array", err, "PARSE_ERROR", "")
}

// ------------------------------------------------------------ version, limits

func TestVersionGate(t *testing.T) {
	for name, src := range map[string]string{
		"missing":          `{"program":[]}`,
		"jaxson key":       `{"jaxson":"1.0","program":[]}`,
		"retired 3.0":      `{"shaxon":"3.0","program":[]}`,
		"retired 1.0":      `{"shaxon":"1.0","program":[]}`,
		"future":           `{"shaxon":"3.2","program":[]}`,
		"not a string":     `{"shaxon":3.1,"program":[]}`,
		"bad plus parse":   `{"shaxon":"3.1","shapes":{"A":{"kind":"bogus"}},"program":[]}`, // would be SHAPE_ERROR if the gate let it by
		"unknown limit":    `{"shaxon":"3.1","limits":{"bogus":1},"program":[]}`,
		"unknown table":    `{"shaxon":"3.1","limits":{"costTable":"no-such"},"program":[]}`,
		"table not string": `{"shaxon":"3.1","limits":{"costTable":1},"program":[]}`,
		"steps too big":    `{"shaxon":"3.1","limits":{"steps":9007199254740993},"program":[]}`,
	} {
		_, err := runSrc(t, src)
		if name == "bad plus parse" {
			wantErr(t, name, err, CatShapeError, "")
			continue
		}
		wantErr(t, name, err, "VERSION_ERROR", "")
	}
}

func TestCostTableErrorsNameTheirCause(t *testing.T) {
	_, err := runSrc(t, `{"shaxon":"3.1","limits":{"costTable":"no-such"},"program":[]}`)
	if err == nil || !strings.Contains(err.Msg, `unsupported costTable "no-such"`) {
		t.Fatalf("unknown table: %v", err)
	}
	_, err = runSrc(t, `{"shaxon":"3.1","limits":{"costTable":1},"program":[]}`)
	if err == nil || !strings.Contains(err.Msg, "costTable must be a string") {
		t.Fatalf("non-string table: %v", err)
	}
}

func TestLimitsAreAccepted(t *testing.T) {
	mustRun(t, `{"shaxon":"3.1","limits":{"steps":1000,"maxShapeDepth":4,"costTable":"unit"},"program":[]}`)
}

func TestStepLimitStopsTheRun(t *testing.T) {
	_, err := runSrc(t, `{"shaxon":"3.1","limits":{"steps":3},"program":[
	  {"op":"set","path":["state","a"],"value":1},{"op":"set","path":["state","b"],"value":2},
	  {"op":"set","path":["state","c"],"value":3},{"op":"set","path":["state","d"],"value":4}]}`)
	wantErr(t, "step limit", err, "RESOURCE_ERROR", "STEPS")
}

func TestMaxShapeDepthMustBeAPositiveIntegerWhenGiven(t *testing.T) {
	for _, v := range []string{`0`, `-1`, `1.5`, `"8"`} {
		_, err := runSrc(t, `{"shaxon":"3.1","limits":{"maxShapeDepth":`+v+`},"program":[]}`)
		wantErr(t, "maxShapeDepth "+v, err, CatShapeError, "")
	}
}

func TestMissingMaxShapeDepthWhereShapesRecurse(t *testing.T) {
	_, err := runSrc(t, `{"shaxon":"3.1","shapes":{"Node":{"kind":"object","closed":false,"fields":{"next":{"shape":"Node"}}}},"program":[]}`)
	wantErr(t, "recursion without a bound", err, CatShapeError, "")
}

// ------------------------------------------------------------ pipeline order

// Each case has two defects in different stages; the earlier stage's error
// must be the one reported, and the later stage must not have run.
func TestPipelineOrder(t *testing.T) {
	shapeOK := `"shapes":{"Pos":{"kind":"number","min":1}}`
	cases := []struct {
		name, src, cat, code string
	}{
		{"version before schema",
			`{"shaxon":"9","inputSchema":{"kind":"bogus"},"program":[]}`, "VERSION_ERROR", ""},
		{"schema before shapes",
			`{"shaxon":"3.1","inputSchema":{"type":"bogus"},"shapes":{"A":{"kind":"bogus"}},"program":[]}`, "SCHEMA_ERROR", ""},
		{"shapes before program",
			`{"shaxon":"3.1","shapes":{"A":{"kind":"bogus"}},"program":[{"op":"nope"}]}`, CatShapeError, ""},
		{"validate list before program",
			`{"shaxon":"3.1",` + shapeOK + `,"validate":[{"target":{"$path":["input"]},"shape":"Missing","mode":"gate"}],"program":[{"op":"nope"}]}`, CatShapeError, ""},
		{"program before input",
			`{"shaxon":"3.1","input":"x","inputSchema":{"type":"number"},"program":[{"op":"nope"}]}`, "PROGRAM_ERROR", ""},
		{"inputSchema before input gates",
			`{"shaxon":"3.1","input":"x","inputSchema":{"type":"number"},` + shapeOK + `,"validate":[{"target":{"$path":["input"]},"shape":"Pos","mode":"gate"}],"program":[]}`, "INPUT_ERROR", ""},
		{"input gate before execution",
			`{"shaxon":"3.1","input":0,` + shapeOK + `,"validate":[{"target":{"$path":["input"]},"shape":"Pos","mode":"gate"}],"program":[{"op":"assert","that":false,"msg":"ran"}]}`, CatValidationError, CodeShapeMismatch},
		{"execution before output schema",
			`{"shaxon":"3.1","outputSchema":{"type":"string"},"program":[{"op":"assert","that":false,"msg":"ran"}]}`, "EXECUTION_ERROR", "ASSERTION_FAILED"},
		{"output schema before output gates",
			`{"shaxon":"3.1","outputSchema":{"type":"string"},` + shapeOK + `,"validate":[{"target":{"$path":["output"]},"shape":"Pos","mode":"gate"}],"program":[{"op":"set","path":["output"],"value":0}]}`, "OUTPUT_ERROR", ""},
	}
	for _, c := range cases {
		_, err := runSrc(t, c.src)
		wantErr(t, c.name, err, c.cat, c.code)
	}
}

func TestOutputGateRunsAfterTheProgram(t *testing.T) {
	src := func(v string) string {
		return `{"shaxon":"3.1","shapes":{"Pos":{"kind":"number","min":1}},
		  "validate":[{"target":{"$path":["output"]},"shape":"Pos","mode":"gate"}],
		  "program":[{"op":"set","path":["output"],"value":` + v + `}]}`
	}
	mustRun(t, src("5"))
	_, err := runSrc(t, src("0"))
	wantErr(t, "output gate", err, CatValidationError, CodeShapeMismatch)
}

func TestStateRootedEntriesRunAfterTheProgram(t *testing.T) {
	// R1: the entry reads state, which only the program fills.
	src := func(v string) string {
		return `{"shaxon":"3.1","shapes":{"Pos":{"kind":"number","min":1}},
		  "validate":[{"target":{"$path":["state","n"]},"shape":"Pos","mode":"gate"}],
		  "program":[{"op":"set","path":["state","n"],"value":` + v + `}]}`
	}
	mustRun(t, src("2"))
	_, err := runSrc(t, src("0"))
	wantErr(t, "state gate", err, CatValidationError, CodeShapeMismatch)
}

// ------------------------------------------------------------ report flow

const orderPkg = `{"shaxon":"3.1",
  "input":{"lines":[{"kind":"discount","amt":-1},{"kind":"item","amt":5},{"kind":"discount","amt":-9}]},
  "shapes":{
    "Discount":{"kind":"object","closed":false,"fields":{"amt":{"kind":"number","max":-5}}},
    "AnyLine":{"kind":"object","closed":false,"fields":{"amt":{"kind":"number"}}}
  },
  "validate":[
    {"target":{"$discriminator":{"at":["input","lines"],"field":"kind","value":"discount"}},
     "shape":"Discount","mode":"report","into":["state","lineWarnings"]}
  ],
  "program":[
    {"op":"set","path":["output"],"value":{"$path":["state","lineWarnings"]}}
  ]}`

func TestInputReportIntoStateIsVisibleToTheProgram(t *testing.T) {
	res := mustRun(t, orderPkg)
	out, ok := res.Output.(map[string]any)
	if !ok {
		t.Fatalf("output = %v", jaxson.Show(res.Output))
	}
	if out["conforms"] != false {
		t.Fatalf("conforms = %v, want false (the first discount breaks max)", out["conforms"])
	}
	vs := out["violations"].([]any)
	if len(vs) != 1 {
		t.Fatalf("violations = %d, want 1: %s", len(vs), jaxson.Show(out))
	}
	if res.Report != nil {
		t.Fatalf("an entry with `into` must not feed the combined report")
	}
}

func TestCombinedReportIsInExecutionOrder(t *testing.T) {
	// pre-program entry, then a `check` inside the program, then a
	// post-program entry: R1 and section 7's last bullet.
	res := mustRun(t, `{"shaxon":"3.1","input":{"a":0},
	  "shapes":{"Pos":{"kind":"number","min":1}},
	  "validate":[
	    {"target":{"$path":["state","s"]},"shape":"Pos","mode":"report"},
	    {"target":{"$path":["input","a"]},"shape":"Pos","mode":"report"}
	  ],
	  "program":[
	    {"op":"set","path":["state","s"],"value":0},
	    {"op":"check","target":{"$path":["state","s"]},"shape":"Pos","mode":"report"}
	  ]}`)
	if res.Report == nil {
		t.Fatal("no combined report")
	}
	var focus []string
	for _, v := range res.Report.Violations {
		focus = append(focus, pathString(v.FocusPath))
	}
	// declared order is [state.s, input.a]; execution order is input.a (pre),
	// then the program's check on state.s, then the post entry on state.s.
	if got := strings.Join(focus, ","); got != "input.a,state.s,state.s" {
		t.Fatalf("combined report order = %s", got)
	}
	if res.Report.Conforms() {
		t.Fatal("conforms must be false")
	}
}

func TestWarningsDoNotFlipConforms(t *testing.T) {
	res := mustRun(t, `{"shaxon":"3.1","input":{"lines":[{"sku":"a"},{"sku":"a"}]},
	  "validate":[{"target":{"$each":["input","lines"]},"unique":{"field":"sku","id":"DUP"},
	    "severity":"warning","message":"dup","mode":"report"}],
	  "program":[]}`)
	if res.Report == nil || len(res.Report.Violations) != 1 || !res.Report.Conforms() {
		t.Fatalf("report = %+v", res.Report)
	}
}

func TestEmptyReportWhenEntriesRanAndFoundNothing(t *testing.T) {
	res := mustRun(t, `{"shaxon":"3.1","input":5,"shapes":{"N":{"kind":"number"}},
	  "validate":[{"target":{"$path":["input"]},"shape":"N","mode":"report"}],"program":[]}`)
	if res.Report == nil || len(res.Report.Violations) != 0 || !res.Report.Conforms() {
		t.Fatalf("report = %+v, want present, empty, conforming", res.Report)
	}
}

func TestGateFailureReturnsTheErrorAlone(t *testing.T) { // R4
	res, err := runSrc(t, `{"shaxon":"3.1","input":{"a":0,"b":0},"shapes":{"Pos":{"kind":"number","min":1}},
	  "validate":[
	    {"target":{"$path":["input","a"]},"shape":"Pos","mode":"report"},
	    {"target":{"$path":["input","b"]},"shape":"Pos","mode":"gate"}],"program":[]}`)
	wantErr(t, "gate", err, CatValidationError, CodeShapeMismatch)
	if res.Report != nil || res.Output != nil {
		t.Fatalf("a failed run returned %+v", res)
	}
}

// ------------------------------------------------------------ no into, no write

func TestValidationWithoutIntoLeavesRootsAlone(t *testing.T) { // plan section 8
	res := mustRun(t, `{"shaxon":"3.1","input":{"xs":[{"sku":"a"},{"sku":"a"}]},
	  "shapes":{"Row":{"kind":"object","required":["sku","zz"],"closed":false}},
	  "validate":[
	    {"target":{"$each":["input","xs"]},"shape":"Row","mode":"report"},
	    {"target":{"$each":["input","xs"]},"unique":{"field":"sku"},"mode":"report"}],
	  "program":[
	    {"op":"check","target":{"$each":["input","xs"]},"shape":"Row","mode":"report"},
	    {"op":"set","path":["output"],"value":{"$lit":{}}},
	    {"op":"set","path":["output","state"],"value":{"$path":["state"]}},
	    {"op":"set","path":["output","input"],"value":{"$path":["input"]}}]}`)
	got := jaxson.Show(res.Output)
	want := `{"input":{"xs":[{"sku":"a"},{"sku":"a"}]},"state":{}}`
	if got != want {
		t.Fatalf("validation changed a root:\n got %s\nwant %s", got, want)
	}
	if res.Report == nil || len(res.Report.Violations) != 5 {
		t.Fatalf("report = %+v, want 5 violations (2+1 from validate, 2 from check)", res.Report)
	}
}

// ------------------------------------------------------------ cost table

const costPkg = `{"shaxon":"3.1","input":{"n":[1,2,3]},
  "shapes":{"N":{"kind":"number"}},
  "validate":[{"target":{"$each":["input","n"]},"shape":"N","mode":"gate"}],
  "program":[{"op":"for","in":{"$path":["input","n"]},"as":"x","do":[]}]}`

func TestUnitIsTheDefaultAndNamingItChangesNothing(t *testing.T) {
	a := mustRun(t, costPkg)
	b := mustRun(t, strings.Replace(costPkg, `"input"`, `"limits":{"costTable":"unit"},"input"`, 1))
	if a.Steps != b.Steps || a.Steps == 0 {
		t.Fatalf("steps differ: default %d, named unit %d", a.Steps, b.Steps)
	}
}

func TestTheCostTableReachesTheMachine(t *testing.T) {
	one := jaxson.Cost{Base: 1}
	heavy := jaxson.CostTable{
		Name:     "test-heavy",
		Instr:    map[string]jaxson.Cost{"for": {Base: 10}},
		Op:       map[string]jaxson.Cost{},
		Event:    map[string]jaxson.Cost{EventShapeActivation: {Base: 7}, EventTargetResolve: {Base: 5}},
		LoopIter: jaxson.Cost{Base: 2},
		Fallback: &one,
	}
	steps := func(table *jaxson.CostTable) int64 {
		v, perr := jaxson.ParseJSON([]byte(costPkg))
		if perr != nil {
			t.Fatal(perr)
		}
		h := &hooks{}
		if table != nil {
			h.resolveCosts = func(map[string]any) (jaxson.CostTable, *jaxson.Err) { return *table, nil }
		}
		if _, err := profile(h).Run(v.(map[string]any)); err != nil {
			t.Fatalf("run: %v", err)
		}
		return h.m.Steps()
	}
	unit := steps(nil)
	got := steps(&heavy)
	// Under heavy: target.resolve 5 + 3 shape activations at 7 each (one per
	// element) + for 10 + 3 loop iterations at 2 = 42. Under unit the same
	// events cost 1 each; the difference is exactly what the table adds.
	if got <= unit {
		t.Fatalf("steps under the heavy table (%d) not above unit (%d): the table was not applied", got, unit)
	}
	if got != 42 {
		t.Fatalf("steps under the heavy table = %d, want 42", got)
	}
}

// ------------------------------------------------------------ concurrency

func TestConcurrentRunsShareNothing(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				res, err := RunJSON([]byte(orderPkg))
				if err != nil || res.Steps == 0 {
					t.Errorf("run: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// ------------------------------------------------------------ indexed targets and the example

func TestIndexedTargetOverAnInputIndexRunsBeforeTheProgram(t *testing.T) {
	// R1 for $indexed: the root comes from the index's source.
	_, err := runSrc(t, `{"shaxon":"3.1","input":{"xs":[{"id":0},{"id":2}]},
	  "indices":{"byId":{"source":{"$path":["input","xs"]},"key":{"$path":["local","item","id"]}}},
	  "shapes":{"Pos":{"kind":"object","closed":false,"fields":{"id":{"kind":"number","min":1}}}},
	  "validate":[{"target":{"$indexed":"byId"},"shape":"Pos","mode":"gate"}],
	  "program":[{"op":"assert","that":false,"msg":"ran"}]}`)
	wantErr(t, "indexed gate", err, CatValidationError, CodeShapeMismatch)
}

func TestPathFormsWorkInProgramOperandsThroughRun(t *testing.T) {
	// The program is checked with the dialect's forms and evaluated on a
	// machine that has them installed.
	res := mustRun(t, `{"shaxon":"3.1","limits":{"maxShapeDepth":4},
	  "input":{"x":7,"head":{"v":1,"next":{"v":2,"next":{"v":3}}}},
	  "program":[
	    {"op":"set","path":["state","alt"],"value":{"$altPath":[["input","missing"],["input","x"]]}},
	    {"op":"set","path":["state","chain"],"value":{"$path*":["next"],"from":["input","head"],"maxDepth":5}},
	    {"op":"set","path":["output"],"value":{"$lit":{}}},
	    {"op":"set","path":["output","alt"],"value":{"$path":["state","alt"]}},
	    {"op":"set","path":["output","hops"],"value":{"$compute":{"with":{"c":{"$path":["state","chain"]}},"expr":["len",{"$v":"c"}]}}}
	  ]}`)
	if got := jaxson.Show(res.Output); got != `{"alt":7,"hops":3}` {
		t.Fatalf("output = %s", got)
	}
}

func TestExamplePackageProducesItsDocumentedFindings(t *testing.T) {
	raw, rerr := os.ReadFile("../../examples/shaxon/orders.json")
	if rerr != nil {
		t.Fatal(rerr)
	}
	res, err := RunJSON(raw)
	if err != nil {
		t.Fatalf("example failed: %v", err)
	}
	if jaxson.Show(res.Output) != `"done"` || res.Report == nil || res.Report.Conforms() {
		t.Fatalf("output %s, report %+v", jaxson.Show(res.Output), res.Report)
	}
	var got []string
	for _, v := range res.Report.Violations {
		got = append(got, pathString(v.FocusPath)+" "+v.Code+v.Message)
	}
	want := []string{
		"input.orders[1].customer SHAX_DANGLING_REFERENCEreferenced value is not present in the named index",
		"input.orders[2].lines[0] below minimum",
		"input.orders[2] not in enum",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("findings:\n got %q\nwant %q", got, want)
	}
}
