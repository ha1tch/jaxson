// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 7.1: Validate, the entry point for the common case of checking one
// document against a shape without running a program.
//
// Decisions (the plan names the function and its purpose but not its
// signature):
//
//	A1  Validate(pkg, shape, doc). pkg declares the vocabulary: shapes, and
//	    whatever indices, relations and computes those shapes refer to. doc is
//	    the document, in the value model Run takes (maps, arrays, *big.Rat,
//	    strings, booleans, nil) or already as Objects. shape is the name of a
//	    declared shape.
//	A2  What runs: the package's static checks (the same ones Run makes), then
//	    one report-mode validation of doc, which sits at the root `input`.
//	    What does not: the package's `program`, its `validate` entries, and
//	    its schemas. Computes and `check` islands inside shapes do run, since
//	    shape evaluation calls them. Indices and relations are built from doc,
//	    so a package's `source` paths must start at `input`, as they do in Run.
//	A3  The result is a Report, never a gate: every finding is collected, and
//	    Conforms says whether there were none of violation severity. An error
//	    is returned only for what Run would also return as an error: a package
//	    that is not valid, an unknown shape, a step limit exceeded, a failure
//	    while evaluating (EXECUTION_ERROR).
//	A4  The step limit is the package's limits.steps, or Run's default. Neither
//	    pkg nor doc is changed.

import "github.com/ha1tch/jaxson/pkg/jaxson"

// Validate checks doc against the shape named shape, which pkg declares, and
// returns the report of every finding. See the decisions A1 to A4 above.
func Validate(pkg map[string]any, shape string, doc any) (Report, *jaxson.Err) {
	run := make(map[string]any, len(pkg)+3)
	for k, v := range pkg {
		switch k {
		case "program", "validate", "inputSchema", "outputSchema", "input":
		default:
			run[k] = v
		}
	}
	run["input"] = doc
	run["program"] = []any{}
	run["validate"] = []any{map[string]any{
		"target": map[string]any{"$path": []any{"input"}},
		"shape":  shape,
		"mode":   "report",
	}}
	res, err := Run(run)
	if err != nil {
		return Report{}, err
	}
	if res.Report == nil {
		return Report{}, nil // not reached: a report-mode entry always reports
	}
	return *res.Report, nil
}

// ValidateJSON is Validate on raw JSON for both the package and the
// document. Either is a PARSE_ERROR if it is not exactly one JSON value
// (duplicate keys included), and the package must be an object.
func ValidateJSON(pkg []byte, shape string, doc []byte) (Report, *jaxson.Err) {
	pv, err := jaxson.ParseJSON(pkg)
	if err != nil {
		return Report{}, err
	}
	p, ok := pv.(map[string]any)
	if !ok {
		return Report{}, &jaxson.Err{Cat: "PARSE_ERROR", Msg: "a package must be a JSON object"}
	}
	dv, err := jaxson.ParseData(doc)
	if err != nil {
		return Report{}, err
	}
	return Validate(p, shape, dv)
}
