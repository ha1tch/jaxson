// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 4.2 (plan section 7): the `check` instruction (core section 8), a
// ninth instruction beside the eight of Jaxson, registered into the table a
// Shaxon run builds. It calls the same Runtime.Run the pipeline's
// `validate` phase uses, so the two cannot drift: section 8 states their
// mode semantics are identical.
//
//	{"op": "check", "target": {...}, "shape": "Name", "mode": "gate"}
//	{"op": "check", "target": {...}, "unique": {...}, "mode": "report", "into": [...]}
//
// Exec charges the instruction's own step through the machine's normal
// instruction accounting before it runs, like every instruction.

import (
	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// CheckInstruction returns the `check` instruction definition, bound to rt.
// Its static Check reads the package's Registries from Checker.Host, which a
// Shaxon run supplies through Hooks.Host.
func CheckInstruction(rt *Runtime) jaxson.InstructionDef {
	return jaxson.InstructionDef{
		Name: "check",
		Req:  []string{"target", "mode"},
		Opt:  []string{"shape", "unique", "into", "severity", "message"},
		Check: func(in map[string]any, c *jaxson.Checker) {
			reg, ok := c.Host.(*Registries)
			if !ok || reg == nil {
				jaxson.Fail("PROGRAM_ERROR", "", "check needs a Shaxon host: it cannot be used in a core Jaxson program")
			}
			locals := make([]string, 0, len(c.Locals))
			for name := range c.Locals {
				locals = append(locals, name)
			}
			parseEntry("check", entryMembers(in), reg, locals...)
		},
		Exec: func(m *jaxson.Machine, in map[string]any) {
			// Already checked statically; this only builds the Entry. Operand
			// segments are evaluated when the target resolves.
			e := parseEntry("check", entryMembers(in), rt.Reg, anyLocal)
			if _, err := rt.Run(e, true); err != nil {
				panic(err)
			}
		},
	}
}

// entryMembers is the instruction minus its "op".
func entryMembers(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if k != "op" {
			out[k] = v
		}
	}
	return out
}

// Instructions returns Jaxson's eight core instructions plus `check`, bound
// to rt. Build it fresh for each run.
func (rt *Runtime) Instructions() map[string]jaxson.InstructionDef {
	tbl := jaxson.CoreInstructions()
	tbl["check"] = CheckInstruction(rt)
	return tbl
}
