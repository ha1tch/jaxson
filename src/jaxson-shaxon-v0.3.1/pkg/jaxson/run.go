// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// The package-level entry points: Run, plus the two pipeline stages a
// host package's own Run must repeat (contract-schema checking and
// value validation) because it cannot reach the unexported originals.

import "math/big"

// CheckSchema statically checks a contract schema (an inputSchema or
// outputSchema document) and returns a documented Err instead of
// panicking. Exported for host packages whose own Run repeats this
// stage of the pipeline.
func CheckSchema(s any) (err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	checkSchema(s)
	return nil
}

// Validate returns "" when v satisfies the (already checked) contract
// schema s, otherwise the first failure message, with paths rooted at
// "$". The schema must have passed CheckSchema first.
func Validate(s, v any) string { return validate(s, v, "$") }

// Run executes a package under the profile and returns either its output
// or a failure.
func (pr Profile) Run(p map[string]any) (out any, err *Err) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Err)
			if !ok {
				panic(r)
			}
			out, err = nil, e
		}
	}()
	if v, _ := p[pr.key()].(string); !pr.accepts(v) {
		fail("VERSION_ERROR", "", "unsupported or missing %s version", pr.key())
	}
	h := pr.begin(p)
	checkSchema(p["inputSchema"])
	checkSchema(p["outputSchema"])
	h.Static(p)
	instructions := h.Instructions()
	c := NewChecker(instructions)
	c.Host = h.Host()
	checkBlock(p["program"], c)
	limit := 100000
	if l, ok := p["limits"].(map[string]any); ok {
		for k := range l {
			if !pr.ownsLimit(k) {
				fail("VERSION_ERROR", "", "unsupported limit %q", k)
			}
		}
		if s, ok := l["steps"].(*big.Rat); ok && s.IsInt() && s.Sign() > 0 {
			limit = int(s.Num().Int64())
		} else if _, has := l["steps"]; has {
			fail("VERSION_ERROR", "", "steps must be a positive integer")
		}
	}
	if msg := validate(p["inputSchema"], p["input"], "$"); msg != "" {
		fail("INPUT_ERROR", "", "%s", msg)
	}
	m := NewMachine(Clone(p["input"]), map[string]any{}, nil, limit, instructions, CoreOperators())
	h.Start(m)
	h.AfterInput(m)
	m.RunProgram(p["program"].([]any))
	if msg := validate(p["outputSchema"], m.output, "$"); msg != "" {
		fail("OUTPUT_ERROR", "", "%s", msg)
	}
	h.AfterOutput(m)
	return m.output, nil
}
