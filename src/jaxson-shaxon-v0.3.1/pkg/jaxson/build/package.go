// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// Package assembles a complete Jaxson package: the contract schemas, an
// optional sample input, an optional step limit, and the program. It is
// immutable; each method returns a modified copy.
type Package struct {
	in, out  Schema
	input    any
	hasInput bool
	steps    int
	prog     []Instr
}

// NewPackage begins an empty package.
func NewPackage() Package { return Package{} }

// InputSchema sets the schema the input must satisfy.
func (p Package) InputSchema(s Schema) Package { p.in = s; return p }

// OutputSchema sets the schema the output must satisfy.
func (p Package) OutputSchema(s Schema) Package { p.out = s; return p }

// Input sets the package's input document (the "input" member). It is
// validated against the input schema on assembly.
func (p Package) Input(v any) Package { p.input, p.hasInput = v, true; return p }

// Steps sets limits.steps, the execution step budget.
func (p Package) Steps(n int) Package { p.steps = n; return p }

// Program appends instructions to the program.
func (p Package) Program(is ...Instr) Package {
	p.prog = append(append([]Instr(nil), p.prog...), is...)
	return p
}

func (p Package) tree() (map[string]any, error) {
	if p.in == nil {
		return nil, errors.New("build: no input schema (call InputSchema)")
	}
	if p.out == nil {
		return nil, errors.New("build: no output schema (call OutputSchema)")
	}
	t := map[string]any{
		"jaxson":       "1.0",
		"inputSchema":  p.in.schemaNode(),
		"outputSchema": p.out.schemaNode(),
		"program":      block(p.prog),
	}
	if p.steps > 0 {
		t["limits"] = map[string]any{"steps": p.steps}
	}
	if p.hasInput {
		t["input"] = p.input
	}
	return t, nil
}

// compile builds the tree, round-trips it through JSON (which is what
// makes numbers exact), and runs the runtime's own static checks, so the
// builder can never accept what the runtime would refuse.
func (p Package) compile() (raw []byte, norm map[string]any, err error) {
	t, err := p.tree()
	if err != nil {
		return nil, nil, err
	}
	raw, err = json.Marshal(t)
	if err != nil {
		return nil, nil, fmt.Errorf("build: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err = dec.Decode(&v); err != nil {
		return nil, nil, fmt.Errorf("build: %w", err)
	}
	norm = jaxson.Normalize(v).(map[string]any)
	if e := jaxson.CheckSchema(norm["inputSchema"]); e != nil {
		return nil, nil, fmt.Errorf("build: input schema: %w", e)
	}
	if e := jaxson.CheckSchema(norm["outputSchema"]); e != nil {
		return nil, nil, fmt.Errorf("build: output schema: %w", e)
	}
	if e := jaxson.CheckProgram(norm["program"], jaxson.CoreInstructions()); e != nil {
		return nil, nil, fmt.Errorf("build: program: %w", e)
	}
	if p.hasInput {
		if msg := jaxson.Validate(norm["inputSchema"], norm["input"]); msg != "" {
			return nil, nil, fmt.Errorf("build: sample input violates its schema: %s", msg)
		}
	}
	return raw, norm, nil
}

// Check assembles the package and reports the first static error, or nil.
func (p Package) Check() error { _, _, err := p.compile(); return err }

// Map returns the package as the normalised document jaxson.Run accepts.
func (p Package) Map() (map[string]any, error) { _, n, err := p.compile(); return n, err }

// JSON returns the package as indented JSON, ready to write to a file for
// jaxrun or jaxplay. Object members are emitted in sorted order, so the
// output is deterministic.
func (p Package) JSON() ([]byte, error) {
	raw, _, err := p.compile()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", " "); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// Run assembles and executes the package against its own Input.
func (p Package) Run() (any, error) {
	m, err := p.Map()
	if err != nil {
		return nil, err
	}
	out, e := jaxson.Run(m)
	if e != nil {
		return nil, e
	}
	return out, nil
}
