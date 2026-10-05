// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// A Profile is a dialect of the language: the same pipeline as Run, with
// a different version member, extra instructions, extra limit members and
// extra stages. It exists so that a dialect reuses the one pipeline
// instead of keeping a second copy that can drift from it.
//
// The zero Profile is the core language. Run is CoreProfile().Run.

// Profile parameterises Run.
type Profile struct {
	// Key is the package member that carries the version ("jaxson" when
	// empty). Versions are the values accepted for it (["1.0"] when nil).
	// A package with a missing or unlisted version fails with
	// VERSION_ERROR before anything else runs.
	Key      string
	Versions []string

	// Limits names the members of `limits` the dialect owns, in addition
	// to "steps". Any other member is a VERSION_ERROR. Profile.Run does not
	// interpret them; the dialect reads them from the package in its own
	// hooks.
	Limits []string

	// SchemasOptional lets a package omit inputSchema and/or outputSchema,
	// in which case that edge is not schema-checked. The core language
	// requires both (the zero value); Shaxon's package, section 2, may use
	// "either, both, or neither".
	SchemasOptional bool

	// Begin is called once per run, after the version gate, and returns
	// the Hooks for that run. Because it is called per run, a dialect keeps
	// per-run state in the value it returns, and every table is built
	// fresh, so concurrent runs share nothing. The package may be
	// malformed in any way beyond the version gate: Begin should capture,
	// not judge. Nil means the core, as does a nil return.
	Begin func(p map[string]any) Hooks
}

// CoreProfile is the core Jaxson language.
func CoreProfile() Profile { return Profile{Key: "jaxson", Versions: []string{"1.0"}} }

// Hooks are a dialect's extra pipeline stages for one run. Embed NoHooks
// and override only the methods needed. Any hook may raise a failure with
// Fail. The order, with the core stages between, is:
//
//	version gate; Begin
//	inputSchema and outputSchema checked
//	Static
//	program checked against Instructions, with Host as Checker.Host
//	limits read
//	input validated against inputSchema
//	Machine built; Start
//	AfterInput
//	program run
//	output validated against outputSchema
//	AfterOutput
//
// An earlier failure means later stages do not run.
type Hooks interface {
	// Instructions returns the instruction table for this run: the core
	// table plus the dialect's own entries. Build it fresh; it may close
	// over per-run state.
	Instructions() map[string]InstructionDef
	// Static checks the package independently of any input.
	Static(p map[string]any)
	// Host is delivered to every registered instruction's Check as
	// Checker.Host.
	Host() any
	// Forms returns the operand forms the dialect adds to the core's, for
	// the static check of the program and for evaluation. Nil means none.
	Forms() map[string]FormDef
	// Start receives the machine before anything runs: set OnMutate here.
	Start(m *Machine)
	// AfterInput runs once the input has passed inputSchema.
	AfterInput(m *Machine)
	// AfterOutput runs once the output has passed outputSchema.
	AfterOutput(m *Machine)
}

// NoHooks is the core's behaviour at every stage. Embed it.
type NoHooks struct{}

// Instructions returns the core table.
func (NoHooks) Instructions() map[string]InstructionDef { return CoreInstructions() }

// Static does nothing.
func (NoHooks) Static(map[string]any) {}

// Host returns nil.
func (NoHooks) Host() any { return nil }

// Forms returns no extra operand forms.
func (NoHooks) Forms() map[string]FormDef { return nil }

// Start does nothing.
func (NoHooks) Start(*Machine) {}

// AfterInput does nothing.
func (NoHooks) AfterInput(*Machine) {}

// AfterOutput does nothing.
func (NoHooks) AfterOutput(*Machine) {}

func (pr Profile) key() string {
	if pr.Key == "" {
		return "jaxson"
	}
	return pr.Key
}

func (pr Profile) versions() []string {
	if pr.Versions == nil {
		return []string{"1.0"}
	}
	return pr.Versions
}

func (pr Profile) accepts(version string) bool {
	for _, v := range pr.versions() {
		if v == version {
			return true
		}
	}
	return false
}

func (pr Profile) ownsLimit(name string) bool {
	if name == "steps" {
		return true
	}
	for _, l := range pr.Limits {
		if l == name {
			return true
		}
	}
	return false
}

// hasSchema reports whether the named schema member is to be checked: always
// under the core's rules, and only when present under SchemasOptional.
func (pr Profile) hasSchema(p map[string]any, name string) bool {
	if !pr.SchemasOptional {
		return true
	}
	_, has := p[name]
	return has
}

func (pr Profile) begin(p map[string]any) Hooks {
	if pr.Begin != nil {
		if h := pr.Begin(p); h != nil {
			return h
		}
	}
	return NoHooks{}
}

// Run executes a core Jaxson package and returns either its output or a
// failure.
func Run(p map[string]any) (out any, err *Err) { return CoreProfile().Run(p) }

// RunJSON is Run on raw JSON: the bytes must be exactly one JSON object, with
// no duplicate key at any depth, or the failure is a PARSE_ERROR. The package's
// input is read straight into the machine's own representation.
func RunJSON(raw []byte) (out any, err *Err) { return CoreProfile().RunJSON(raw) }

// RunJSON is Run on raw JSON; see the package-level RunJSON.
func (pr Profile) RunJSON(raw []byte) (out any, err *Err) {
	v, err := ParsePackage(raw)
	if err != nil {
		return nil, err
	}
	p, ok := v.(map[string]any)
	if !ok {
		return nil, &Err{Cat: "PARSE_ERROR", Msg: "a package must be a JSON object"}
	}
	return pr.Run(p)
}
