// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 5 (plan section 8): shaxon.Run, the version gate, and the pipeline
// in the order of core section 9.
//
// Shaxon does not keep a pipeline of its own. jaxson.Profile.Run is the one
// pipeline; this file is a Profile ("shaxon", version "3.1") plus the Hooks
// that fill in what Shaxon adds. The order, with the hook that owns each
// Shaxon step in brackets, is core section 9's:
//
//	parse (ParseJSON, for RunJSON)
//	version
//	inputSchema, outputSchema checked
//	shapes, indices, relations, computes, validate checked   [Static]
//	program checked, `check` included                        [Instructions, Host, Forms]
//	limits read; input checked against inputSchema
//	input-rooted validate entries                            [AfterInput]
//	program run, `check` instructions included
//	output checked against outputSchema
//	remaining validate entries                               [AfterOutput]
//
// An earlier failure means later stages do not run.
//
// ====================================================================
// Decisions where the spec is silent or ambiguous (R1..R7)
// ====================================================================
//
//	R1  Which stage a `validate` entry belongs to. Core section 9 puts
//	    "input schema/gate-shapes" before execution and "output schema/
//	    gate-shapes" after it, and section 7 says entries run in declared
//	    order, but nothing says how a list that mixes the two is split. The
//	    rule here: an entry whose target is rooted wholly in `input` runs
//	    before the program, every other entry (rooted in `state` or `output`,
//	    or whose root cannot be known statically) after it. Each group keeps
//	    declared order. This fits section 7's own example, whose report-mode
//	    entry reads `input` and writes `into` ["state", "lineWarnings"] so
//	    that the program can use it. Needs a spec ruling.
//	R2  A report written `into` a path under `output` by a post-program
//	    entry lands after outputSchema was checked, so outputSchema does not
//	    cover it. Writing there is allowed (V2); the gap is recorded, not
//	    closed, because closing it means either checking the schema twice or
//	    moving the post-program entries ahead of the schema, and section 9
//	    lists the schema first.
//	R3  The result. Run returns the output, and, "alongside output" (section
//	    7), the combined report: present only when at least one report-mode
//	    entry or `check` without `into` actually ran. Its violations are
//	    concatenated in execution order. Steps is the total the run charged;
//	    the count is semantic, so a caller can pin it.
//	R4  A failure returns the error alone. A report collected before a gate
//	    failure is not returned with it: the gate's error is the finding.
//	R5  Only "3.1" is accepted (plan section 11); the retired strings
//	    "1.0".."3.0" are a VERSION_ERROR.
//	R6  `limits` may carry steps, maxShapeDepth and costTable; any other
//	    member is VERSION_ERROR (Profile.Run's rule). maxShapeDepth, when
//	    present, must be a positive integer even in a package where nothing
//	    recurses (section 2 only makes it mandatory when something can).
//	R7  inputSchema and outputSchema are optional (section 2: "either,
//	    both, or neither"). The core language requires both, so this is a
//	    jaxson.Profile option, SchemasOptional; an absent schema means that
//	    edge is simply not schema-checked.

import (
	"math/big"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// Version is the one Shaxon version this runtime implements (R5).
const Version = "3.1"

// Result is what a successful run returns (R3).
type Result struct {
	Output any
	// Report is the combined report of every report-mode entry and `check`
	// that had no `into`; nil if none ran.
	Report *Report
	// Steps is the total charged to the step counter by the run.
	Steps int64
}

// Profile is the Shaxon dialect as a jaxson.Profile. The returned hooks
// value is per run; Run keeps hold of it to read the report back.
func profile(capture *hooks) jaxson.Profile {
	return jaxson.Profile{
		Key:      "shaxon",
		Versions: []string{Version},
		Limits:   []string{"maxShapeDepth", "costTable"},
		// Core section 2: a package may use either schema, both, or neither.
		SchemasOptional: true,
		Begin: func(p map[string]any) jaxson.Hooks {
			capture.pkg = p
			return capture
		},
	}
}

// Run executes a decoded Shaxon package (values as jaxson.ParseJSON or
// jaxson.Normalize make them).
func Run(pkg map[string]any) (res Result, err *jaxson.Err) {
	h := &hooks{}
	// Static replaces the package's program with its expansion (aggregate
	// sugar); do that on a copy, not on the caller's package.
	cp := make(map[string]any, len(pkg))
	for k, v := range pkg {
		cp[k] = v
	}
	out, err := profile(h).Run(cp)
	if err != nil {
		return Result{}, err
	}
	res = Result{Output: out}
	if h.m != nil {
		res.Steps = h.m.Steps()
	}
	if h.rt != nil && h.rt.reported {
		rep := h.rt.Combined()
		res.Report = &rep
	}
	return res, nil
}

// RunJSON is Run on raw JSON: duplicate keys and anything that is not one
// JSON object are a PARSE_ERROR.
func RunJSON(raw []byte) (Result, *jaxson.Err) {
	v, err := jaxson.ParsePackage(raw) // the input member is parsed straight into Objects
	if err != nil {
		return Result{}, err
	}
	pkg, ok := v.(map[string]any)
	if !ok {
		return Result{}, &jaxson.Err{Cat: "PARSE_ERROR", Msg: "a package must be a JSON object"}
	}
	return Run(pkg)
}

// hooks is the per-run state of one Shaxon run.
type hooks struct {
	jaxson.NoHooks
	pkg   map[string]any
	rt    *Runtime
	pre   []Entry // R1: rooted wholly in input
	post  []Entry
	costs jaxson.CostTable
	m     *jaxson.Machine

	// resolveCosts reads the cost table from the package; nil means
	// jaxson.ResolveCostTable. Only the tests set it, to run the pipeline
	// under a table other than `unit` before the language defines one.
	resolveCosts func(map[string]any) (jaxson.CostTable, *jaxson.Err)
}

func (h *hooks) Static(p map[string]any) {
	// Sugar first: after this the program holds only the core's instructions.
	if prog, has := p["program"]; has {
		p["program"] = expandAggregates(prog)
	}
	reg, err := ParseRegistries(p)
	if err != nil {
		panic(err)
	}
	entries, err := ParseValidate(p, reg)
	if err != nil {
		panic(err)
	}
	h.rt = NewRuntime(reg, readMaxShapeDepth(p))
	for _, e := range entries {
		if e.Target.root(reg) == "input" {
			h.pre = append(h.pre, e)
		} else {
			h.post = append(h.post, e)
		}
	}
	resolve := h.resolveCosts
	if resolve == nil {
		resolve = jaxson.ResolveCostTable
	}
	costs, cerr := resolve(p)
	if cerr != nil {
		panic(cerr)
	}
	if verr := costs.Validate(h.rt.Instructions(), jaxson.CoreOperators()); verr != nil {
		jaxson.Fail("VERSION_ERROR", "", "costTable: %v", verr)
	}
	h.costs = costs
}

func (h *hooks) Instructions() map[string]jaxson.InstructionDef { return h.rt.Instructions() }
func (h *hooks) Host() any                                      { return h.rt.Reg }
func (h *hooks) Forms() map[string]jaxson.FormDef               { return h.rt.Forms() }

func (h *hooks) Start(m *jaxson.Machine) {
	h.m = m
	h.rt.Bind(m)
	m.SetCostTable(h.costs)
}

func (h *hooks) AfterInput(*jaxson.Machine)  { h.runAll(h.pre) }
func (h *hooks) AfterOutput(*jaxson.Machine) { h.runAll(h.post) }

func (h *hooks) runAll(entries []Entry) {
	for _, e := range entries {
		if _, err := h.rt.Run(e, false); err != nil {
			panic(err)
		}
	}
}

// readMaxShapeDepth reads limits.maxShapeDepth (R6). 0 means none declared.
func readMaxShapeDepth(p map[string]any) int {
	limits, _ := p["limits"].(map[string]any)
	v, has := limits["maxShapeDepth"]
	if !has {
		return 0
	}
	n, ok := v.(*big.Rat)
	if !ok || !n.IsInt() || n.Sign() <= 0 || !n.Num().IsInt64() || n.Num().Int64() > 1<<31 {
		failLoad("limits.maxShapeDepth must be a positive integer")
	}
	return int(n.Num().Int64())
}

// root returns the root ("input", "state" or "output") a target reads from,
// or "" when it cannot be known statically (R1).
func (t Target) root(r *Registries) string {
	path := t.Path
	if t.Kind == TargetIndexed {
		src, _ := r.Indices[t.Index].Source.(map[string]any)
		path, _ = src["$path"].([]any)
	}
	if len(path) == 0 {
		return ""
	}
	root, _ := path[0].(string)
	return root
}
