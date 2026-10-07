// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 4.2 (plan section 7): the `validate` list, `unique`, and the shared
// entry point that both the pipeline's `validate` phase and the `check`
// instruction call (core sections 7 and 8).
//
// ====================================================================
// Decisions where the spec is silent or ambiguous (V1..V7)
// ====================================================================
//
//	V1  `mode` is mandatory on every entry and instruction. The spec's
//	    examples always give it and no default is stated; requiring it
//	    keeps the choice explicit, and a default can be added later without
//	    breaking a package, whereas changing one could.
//	V2  `into` is meaningful only in report mode; with gate it is a
//	    SHAPE_ERROR. It must be a writable path (state or output): this is
//	    checked by running the core static checker over the `set`/`append`
//	    instruction the write is carried out with.
//	V3  Delivery. A report-mode `validate` entry with `into` SETS the whole
//	    report value ({"conforms", "violations"}) there. A report-mode
//	    `check` with `into` APPENDS each violation to the array there, which
//	    must already exist (as for the `append` instruction). Both are
//	    carried out by running a synthesised `set`/`append` through the
//	    Machine, so each is an ordinary program write: it costs what that
//	    instruction costs (one step per violation appended), and it is
//	    visible to the mutation log, so indices over the written path are
//	    rebuilt on next use. Entries with no `into` add their violations to
//	    the run's combined report (core section 7, last bullet).
//	V4  `unique` skips an element that is an object lacking `field`: it has
//	    no key, so it cannot repeat one. (An index build, by contrast,
//	    raises MISSING_PATH, as section 3 says.) Keys are compared with
//	    jaxson.Equal semantics through their canonical rendering, so 1 and
//	    1.0 are the same key and "1" is not.
//	V5  A `unique` violation has no shape, so its `shape` member is null,
//	    and no constraintPath (section 10's table). Its message defaults to
//	    "repeated value" when `message` is not given.
//	V6  `severity` and `message` on an entry are accepted only with
//	    `unique`; a shape carries its own. On a `shape` entry they are a
//	    SHAPE_ERROR rather than silently ignored.
//	V7  Indices an entry uses are refreshed (built, or confirmed reusable)
//	    before its targets resolve, which is where core section 3 puts the
//	    charge. "Uses" is found statically: every reference-kind shape
//	    reachable from the entry's shape, plus the index a `$indexed` target
//	    names. Freshness never depends on this analysis, only the moment of
//	    the charge does: a lookup the analysis missed still rebuilds a stale
//	    index when it happens.

import (
	"fmt"
	"sort"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// UniqueDecl is a parsed `unique` declaration (core section 7).
type UniqueDecl struct {
	Field    string // "" means the element itself
	ID       string
	Severity string
	Message  string
}

// Entry is one `validate` list entry, or the operands of a `check`
// instruction: a target, exactly one of Shape or Unique, and a mode.
type Entry struct {
	Target Target
	Shape  string
	Unique *UniqueDecl
	Mode   Mode
	Into   []any // nil when omitted
}

// ParseValidate statically checks the package's `validate` list. It
// returns the entries in declared order; an absent member is an empty list.
func ParseValidate(pkg map[string]any, r *Registries) (entries []Entry, err *jaxson.Err) {
	defer func() {
		if rec := recover(); rec != nil {
			e, ok := rec.(*jaxson.Err)
			if !ok {
				panic(rec)
			}
			entries, err = nil, e
		}
	}()
	raw, has := pkg["validate"]
	if !has {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		failLoad("validate must be an array")
	}
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			failLoad("validate[%d] must be an object", i)
		}
		entries = append(entries, parseEntry(fmt.Sprintf("validate[%d]", i), m, r))
	}
	return entries, nil
}

// parseEntry statically checks the members shared by a `validate` entry and
// a `check` instruction (the instruction's "op" is not among them).
func parseEntry(ctx string, m map[string]any, r *Registries, locals ...string) Entry {
	for k := range m {
		switch k {
		case "target", "shape", "unique", "mode", "into", "severity", "message":
		default:
			failLoad("%s: unknown key %q", ctx, k)
		}
	}
	rawT, has := m["target"]
	if !has {
		failLoad("%s: needs a target", ctx)
	}
	e := Entry{Target: parseTarget(ctx+".target", rawT, r, locals...)}

	shapeRaw, hasShape := m["shape"]
	uniqRaw, hasUnique := m["unique"]
	switch {
	case hasShape && hasUnique:
		failLoad("%s: declares both shape and unique; an entry has exactly one", ctx)
	case !hasShape && !hasUnique:
		failLoad("%s: needs exactly one of shape or unique", ctx)
	case hasShape:
		name, ok := shapeRaw.(string)
		if !ok {
			failLoad("%s.shape must be a shape name", ctx)
		}
		if _, declared := r.Shapes[name]; !declared {
			failLoad("%s.shape names an undeclared shape %q", ctx, name)
		}
		if _, s := m["severity"]; s {
			failLoad("%s: severity is declared by the shape, not the entry (V6)", ctx)
		}
		if _, s := m["message"]; s {
			failLoad("%s: message is declared by the shape, not the entry (V6)", ctx)
		}
		e.Shape = name
	default:
		e.Unique = parseUnique(ctx, uniqRaw, m)
	}

	modeRaw, has := m["mode"]
	if !has {
		failLoad("%s: needs a mode (V1)", ctx)
	}
	switch modeRaw {
	case "gate":
		e.Mode = ModeGate
	case "report":
		e.Mode = ModeReport
	default:
		failLoad("%s.mode must be \"gate\" or \"report\"", ctx)
	}

	if into, has := m["into"]; has {
		if e.Mode == ModeGate {
			failLoad("%s: into is only meaningful in report mode (V2)", ctx)
		}
		p, ok := into.([]any)
		if !ok || len(p) == 0 {
			failLoad("%s.into must be a path array", ctx)
		}
		probe := []any{map[string]any{"op": "set", "path": p, "value": map[string]any{"$lit": nil}}}
		if err := jaxson.CheckProgram(probe, jaxson.CoreInstructions()); err != nil {
			failLoad("%s.into: %s", ctx, err.Msg)
		}
		e.Into = p
	}
	return e
}

func parseUnique(ctx string, raw any, entry map[string]any) *UniqueDecl {
	um, ok := raw.(map[string]any)
	if !ok {
		failLoad("%s.unique must be an object", ctx)
	}
	u := &UniqueDecl{Severity: SeverityViolation}
	for k, v := range um {
		s, isStr := v.(string)
		switch {
		case k == "field" && isStr && s != "":
			u.Field = s
		case k == "id" && isStr && s != "":
			u.ID = s
		default:
			failLoad("%s.unique: %q is not a recognised member or has a bad value (field and id are plain strings)", ctx, k)
		}
	}
	if v, has := entry["severity"]; has {
		s, ok := v.(string)
		if !ok || (s != SeverityViolation && s != SeverityWarning && s != SeverityInfo) {
			failLoad("%s.severity must be violation, warning or info", ctx)
		}
		u.Severity = s
	}
	if v, has := entry["message"]; has {
		s, ok := v.(string)
		if !ok {
			failLoad("%s.message must be a string", ctx)
		}
		u.Message = s
	}
	return u
}

// ---------------------------------------------------------------- runtime

// Runtime holds what one run's validation needs: the registries, the
// run's IndexSet, and the combined report that entries with no `into`
// contribute to. Create it with NewRuntime, bind it to the Machine once the
// Machine exists, and use Run for every `validate` entry and `check`
// instruction so that index builds are shared between them.
type Runtime struct {
	Reg           *Registries
	MaxShapeDepth int // limits.maxShapeDepth; 0 means none declared

	m        *jaxson.Machine
	idx      *IndexSet
	ambient  *Ambient
	combined []Violation
	reported bool // a report-mode entry without `into` has run (R3)
}

// NewRuntime returns an unbound Runtime.
func NewRuntime(reg *Registries, maxShapeDepth int) *Runtime {
	return &Runtime{Reg: reg, MaxShapeDepth: maxShapeDepth, ambient: &Ambient{}}
}

// Forms returns the path-expression operand forms (forms.go), checked and
// evaluated against this Runtime. A pipeline returns it from Hooks.Forms.
func (rt *Runtime) Forms() map[string]jaxson.FormDef {
	return formEnv{reg: rt.Reg, idx: func() *IndexSet { return rt.idx }, ambient: rt.ambient}.defs()
}

// Bind attaches the Runtime to its Machine and creates the run's IndexSet,
// which installs itself on the Machine's mutation hook. Call it before
// anything runs.
func (rt *Runtime) Bind(m *jaxson.Machine) {
	rt.m = m
	rt.idx = NewIndexSet(m, rt.Reg)
	rt.idx.ambient = rt.ambient
	m.SetForms(rt.Forms())
}

// Indices returns the run's IndexSet.
func (rt *Runtime) Indices() *IndexSet { return rt.idx }

// Combined returns the report built from every report-mode entry that had
// no `into`, in the order the entries ran (core section 7).
func (rt *Runtime) Combined() Report {
	return Report{Violations: append([]Violation(nil), rt.combined...)}
}

// Run executes one entry: refresh its indices, resolve its target, judge
// every focus node, and deliver the result. appendInto selects `check`'s
// append-into delivery over `validate`'s set (V3).
//
// In gate mode the first violation-severity finding is returned as the
// error (VALIDATION_ERROR/SHAX_SHAPE_MISMATCH, or the structural finding's
// own EXECUTION_ERROR code) and nothing is delivered. Any other failure the
// machinery raises is returned as the error too.
func (rt *Runtime) Run(e Entry, appendInto bool) (rep Report, err *jaxson.Err) {
	defer func() {
		if r := recover(); r != nil {
			x, ok := r.(*jaxson.Err)
			if !ok {
				panic(r)
			}
			err = x
		}
	}()
	for _, t := range rt.usedRefs(e) {
		rt.idx.Refresh(t)
	}
	foci, depthAt := e.Target.resolve(rt.m, rt.idx)
	if depthAt != nil { // a closure hit its horizon (C3): a structural finding
		v := structural(CodePathDepthExceeded, msgPathDepthExceeded, e.Shape, depthAt)
		rep.Violations = append(rep.Violations, v)
		if e.Mode == ModeGate {
			return rep, gateError(v)
		}
	}

	if e.Shape != "" {
		ev := NewEvaluator(rt.m, rt.Reg, Options{Mode: e.Mode, MaxShapeDepth: rt.MaxShapeDepth, Resolver: rt.idx, Ambient: rt.ambient})
		for _, f := range foci {
			r, ferr := ev.CheckValue(e.Shape, f.Path, f.Value)
			rep.Violations = append(rep.Violations, r.Violations...)
			if ferr != nil { // a gate failure or anything raised: either way, stop
				return rep, ferr
			}
		}
	} else {
		if uerr := rt.unique(e, foci, &rep); uerr != nil {
			return rep, uerr
		}
	}
	rt.deliver(e, rep, appendInto)
	return rep, nil
}

// unique walks foci in visiting order, flagging each element whose key was
// already seen (V4, V5). In gate mode it stops at the first violation-
// severity repeat and returns its error.
func (rt *Runtime) unique(e Entry, foci []Focus, rep *Report) *jaxson.Err {
	u := e.Unique
	seen := map[string]bool{}
	for _, f := range foci {
		key := f.Value
		if u.Field != "" {
			obj, ok := f.Value.(*jaxson.Object)
			if !ok {
				continue
			}
			v, has := obj.Get(u.Field)
			if !has {
				continue
			}
			key = v
		}
		k := jaxson.Show(key)
		if !seen[k] {
			seen[k] = true
			continue
		}
		msg := u.Message
		if msg == "" {
			msg = "repeated value"
		}
		v := Violation{FocusPath: copyPath(f.Path), ConstraintID: u.ID, Severity: u.Severity, Message: msg}
		rep.Violations = append(rep.Violations, v)
		if e.Mode == ModeGate && u.Severity == SeverityViolation {
			return gateError(v)
		}
	}
	return nil
}

// deliver hands a report-mode result on (V3). Gate mode delivers nothing:
// reaching here means it conformed.
func (rt *Runtime) deliver(e Entry, rep Report, appendInto bool) {
	if e.Mode != ModeReport {
		return
	}
	switch {
	case e.Into == nil:
		rt.reported = true
		rt.combined = append(rt.combined, rep.Violations...)
	case appendInto:
		for _, v := range rep.Violations {
			rt.m.RunProgram([]any{map[string]any{"op": "append", "path": e.Into, "value": map[string]any{"$lit": v.Value()}}})
		}
	default:
		rt.m.RunProgram([]any{map[string]any{"op": "set", "path": e.Into, "value": map[string]any{"$lit": rep.Value()}}})
	}
}

// ---------------------------------------------------------------- used indices

// usedRefs lists, in a fixed order, the index-backed references an entry
// depends on (V7).
func (rt *Runtime) usedRefs(e Entry) []ReferenceTarget {
	found := map[string]ReferenceTarget{}
	if e.Target.Kind == TargetIndexed {
		t := ReferenceTarget{Index: e.Target.Index}
		found[refID(t)] = t
	}
	if e.Shape != "" {
		seen := map[string]bool{}
		rt.collectShapeRefs(e.Shape, seen, found)
	}
	ids := make([]string, 0, len(found))
	for id := range found {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]ReferenceTarget, 0, len(ids))
	for _, id := range ids {
		out = append(out, found[id])
	}
	return out
}

func (rt *Runtime) collectShapeRefs(name string, seen map[string]bool, out map[string]ReferenceTarget) {
	if seen[name] {
		return
	}
	seen[name] = true
	s, ok := rt.Reg.Shapes[name]
	if !ok {
		return
	}
	rt.collectDeclRefs(s, seen, out)
}

func (rt *Runtime) collectDeclRefs(s ShapeDecl, seen map[string]bool, out map[string]ReferenceTarget) {
	if s.Kind == "reference" {
		t := ReferenceTarget{Index: s.RefIndex, Relation: s.RefRelation, Of: s.RefOf, By: s.RefBy}
		out[refID(t)] = t
	}
	for _, ck := range s.Check { // an $inverse in a check uses its index too
		with, expr := ck.With, ck.Expr
		if ck.ComputeRef != "" {
			cd := rt.Reg.Computes[ck.ComputeRef]
			with, expr = cd.With, cd.Expr
		}
		for _, x := range []any{with, expr} {
			scanInverse(x, func(spec map[string]any) {
				t := ReferenceTarget{}
				if name, ok := spec["index"].(string); ok {
					t.Index = name
				} else {
					t.Relation, _ = spec["relation"].(string)
				}
				out[refID(t)] = t
			})
		}
	}
	visit := func(fd FieldDecl) {
		if fd.ShapeRef != "" {
			rt.collectShapeRefs(fd.ShapeRef, seen, out)
		} else if fd.Inline != nil {
			rt.collectDeclRefs(*fd.Inline, seen, out)
		}
	}
	for _, fd := range s.Fields {
		visit(fd)
	}
	if s.Items != nil {
		visit(*s.Items)
	}
	for _, q := range s.Qualified {
		visit(q.Target)
	}
	for _, group := range [][]FieldDecl{s.And, s.Or, s.Xone, s.Not} {
		for _, fd := range group {
			visit(fd)
		}
	}
}
