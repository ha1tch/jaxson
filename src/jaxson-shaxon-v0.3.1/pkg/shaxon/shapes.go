// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 3 (plan section 6): shape evaluation. Given a parsed Registries
// (Phase 2) and a jaxson.Machine, check a focus node against a named shape
// and produce the section 10 violation records.
//
// ====================================================================
// Indices (was a temporary stopgap, closed in Phase 4.1)
// ====================================================================
//
// `reference`-kind shapes (core section 5) are served through the
// IndexResolver interface. Phase 3 shipped it with only a test fake behind
// it; Phase 4.1 supplies the real implementation, IndexSet (indices.go),
// which builds `indices` and `relations` and reuses builds via
// Machine.OnMutate. With Options.Resolver == nil the Evaluator creates an
// IndexSet on first use. The interface stays as the seam: the Phase 3 tests
// still drive the evaluator with a fake, so they test shape evaluation
// alone. The SHAX_NO_INDEX_RESOLVER code of the stopgap no longer exists.
//
// ====================================================================
// Places the spec is silent or ambiguous, and what this file does
// ====================================================================
//
// Each is a decision, not a discovery of intent; each is logged in
// TRACKER.md as G1..G11 and is cheap to change, because it lives in one
// place:
//
//	G1  The "ordinary shape-evaluation step cost" is never defined. Here:
//	    one step per shape *activation* (shape() below) — a named shape,
//	    a combinator alternative, a qualified element, an item, or an
//	    inline field that has structure. Primitive checks on a leaf are
//	    free. A `check` island additionally costs what its expressions
//	    cost, charged by jaxson itself. See chargeShape.
//	G2  Closed-object violations are not in the section 10 table: one
//	    per unexpected member, constraintPath = [member].
//	G3  A failed `qualified` count is not in the table: one violation at
//	    the collection's focus, no constraintPath.
//	G4  A kind/keyword failure of the focus node itself (top-level target
//	    or a named-shape field) is one violation at that focus, no
//	    constraintPath; for an inline field it is the table's row 2
//	    (focus = the parent object, constraintPath = [field]).
//	G5  Rows 2 and 3 both cover "a fields member": an inline field's own
//	    kind/keywords are row 2 (reported on the parent); any structure it
//	    declares, and every {"shape": ...} field, is row 3 (its own focus).
//	G6  Evaluation order: kind, keywords, required, fields, items, closed,
//	    qualified, and/or/xone/not, check. Section 10 fixes only required
//	    before fields before check. Plan section 6 listed fields before
//	    required; the spec governs.
//	G7  A probe (combinator alternative, qualified element) stops at its
//	    first violation-severity finding; warning/info never fail one.
//	G8  Depth counts *named* shapes only: the root is depth 1 and each
//	    {"shape": N} followed is one deeper (an inline structured field is
//	    not a descent). Entering a named shape at depth > maxShapeDepth is a
//	    structural finding, not a raised error — except in gate mode, where
//	    it raises EXECUTION_ERROR/SHAX_SHAPE_DEPTH_EXCEEDED.
//	G9  Messages: the most specific authored `message` (field shape, then
//	    enclosing shape) wins; otherwise the text generated here, which is
//	    descriptive and NOT normative.
//	G10 An `"id"` member on a shape is accepted and becomes constraintId
//	    (section 10 names it; the Phase 2 parser rejected it).
//	G11 RULED 2026-10-07: a child may only narrow what it extends.
//	    mergeShapes carries kind, reference target, items, qualified and the
//	    primitive keywords (stricter bound wins, enums intersect); a
//	    conflicting kind, target, items or qualified is a SHAPE_ERROR. Layer
//	    1 is done; severity/message inheritance (layer 2) is open.

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// ReferenceTarget names what a reference-kind shape points at: exactly one
// of Index, Relation, or the inline Of+By shorthand (core sections 4a, 5).
type ReferenceTarget struct {
	Index, Relation string
	Of, By          []any
}

// IndexResolver answers "is this scalar present as a key in the target?".
//
// IndexSet (indices.go) is the implementation a run uses; the interface
// remains so the evaluator can be tested apart from index building.
//
// key is a string or a *big.Rat (never an object or array — the evaluator
// raises TYPE_ERROR before calling). An implementation may raise a
// documented error with jaxson.Fail; the evaluator lets it propagate.
type IndexResolver interface {
	Lookup(target ReferenceTarget, key any) (found bool)
}

// Mode selects core section 7's two result modes.
type Mode int

const (
	// ModeReport collects every independently failing finding.
	ModeReport Mode = iota
	// ModeGate stops at the first violation-severity finding and raises.
	ModeGate
)

// Options configures an Evaluator. The zero value is report mode, no
// shape-depth bound, no resolver.
type Options struct {
	Mode Mode
	// MaxShapeDepth is limits.maxShapeDepth. 0 means none was declared
	// (the parser requires a positive value whenever one is, so 0 is never
	// a real bound); the bound is then not enforced.
	MaxShapeDepth int
	// Resolver serves reference-kind shapes. Nil means the Evaluator makes
	// an IndexSet on first use; a pipeline passes the run's own IndexSet so
	// builds are shared across entries.
	Resolver IndexResolver
	// Ambient publishes the focus path to the closure forms while a `check`
	// runs (forms.go, F3). Nil means the Evaluator keeps its own.
	Ambient *Ambient
}

// Evaluator checks focus nodes against the shapes of one Registries.
//
// It is single-use per call and not safe for concurrent use: it shares the
// Machine's step counter (as plan section 6 requires — "alongside
// Machine.Step() the same way steps does"), and the Machine is not
// concurrent either. Build one Evaluator per run.
type Evaluator struct {
	m       *jaxson.Machine
	reg     *Registries
	opts    Options
	depth   int
	ambient *Ambient
	own     *IndexSet // made on demand when Options.Resolver is not an *IndexSet
}

// NewEvaluator returns an Evaluator over m and reg.
func NewEvaluator(m *jaxson.Machine, reg *Registries, opts Options) *Evaluator {
	e := &Evaluator{m: m, reg: reg, opts: opts, ambient: opts.Ambient}
	if e.ambient == nil {
		e.ambient = &Ambient{}
	}
	if !m.HasForm("$inverse") { // a bare Machine: install the path-expression forms
		m.SetForms(formEnv{reg: reg, idx: e.indices, ambient: e.ambient}.defs())
	}
	return e
}

// indices returns the IndexSet this Evaluator reads, making one on first
// need. When Options.Resolver is an *IndexSet that is the one; when it is
// nil the new set also becomes the resolver; when it is some other
// IndexResolver (a test double) the set is a private one for the forms.
func (e *Evaluator) indices() *IndexSet {
	if s, ok := e.opts.Resolver.(*IndexSet); ok {
		return s
	}
	if e.own == nil {
		e.own = NewIndexSet(e.m, e.reg)
		e.own.ambient = e.ambient
		if e.opts.Resolver == nil {
			e.opts.Resolver = e.own
		}
	}
	return e.own
}

// Check evaluates the named shape against the value at root+segs (root is
// "input", "state" or "output"). A path that does not resolve is returned
// as the Machine's own MISSING_PATH/TYPE_ERROR.
func (e *Evaluator) Check(shape, root string, segs []any) (Report, *jaxson.Err) {
	val, werr := e.m.Walk(root, segs)
	if werr != nil {
		return Report{}, werr
	}
	return e.CheckValue(shape, appendPathAll([]any{root}, segs), val)
}

// CheckValue evaluates the named shape against val, reporting focusPath as
// val's address. In ModeGate the first violation-severity finding is
// returned as the error — VALIDATION_ERROR/SHAX_SHAPE_MISMATCH, or for a
// structural finding its EXECUTION_ERROR code — and the Report holds what
// was collected up to and including it. Any other raised failure (step
// limit, TYPE_ERROR, a missing resolver) is returned as the error too.
func (e *Evaluator) CheckValue(shape string, focusPath []any, val any) (rep Report, err *jaxson.Err) {
	sd, ok := e.reg.Shapes[shape]
	if !ok {
		return Report{}, &jaxson.Err{Cat: CatShapeError, Msg: fmt.Sprintf("unknown shape %q", shape)}
	}
	c := &collector{stop: e.opts.Mode == ModeGate}
	e.depth = 0
	defer func() {
		rep = Report{Violations: c.list}
		switch x := recover().(type) {
		case nil:
		case stopSignal:
			err = gateError(c.list[len(c.list)-1])
		case *jaxson.Err:
			err = x
		default:
			panic(x)
		}
	}()
	e.shape(c, &sd, shape, focusPath, val, false)
	return
}

func appendPathAll(p, more []any) []any {
	out := make([]any, 0, len(p)+len(more))
	out = append(out, p...)
	return append(out, more...)
}

func gateError(v Violation) *jaxson.Err {
	at := v.FocusPath
	if v.ConstraintPath != nil {
		at = appendPathAll(at, v.ConstraintPath)
	}
	msg := fmt.Sprintf("%s: %s", pathString(at), v.Message)
	if v.Code != "" {
		return &jaxson.Err{Cat: "EXECUTION_ERROR", Code: v.Code, Msg: msg}
	}
	return &jaxson.Err{Cat: CatValidationError, Code: CodeShapeMismatch, Msg: msg}
}

// ---------------------------------------------------------------- collection

// stopSignal unwinds an evaluation that has already determined its answer:
// gate mode on its first failure, a probe on its first failing finding.
type stopSignal struct{}

type collector struct {
	list []Violation
	stop bool
}

func (c *collector) emit(v Violation) {
	c.list = append(c.list, v)
	if c.stop && v.Severity == SeverityViolation {
		panic(stopSignal{})
	}
}

// finding builds a constraint finding for sd, with sd's severity, id and
// message (G9), falling back to generated text.
func (e *Evaluator) finding(sd *ShapeDecl, named string, fp, cp []any, generated string) Violation {
	msg := sd.Message
	if msg == "" {
		msg = generated
	}
	sev := sd.Severity
	if sev == "" { // a ShapeDecl built by hand rather than parsed
		sev = SeverityViolation
	}
	return Violation{
		FocusPath: copyPath(fp), ConstraintPath: copyPath(cp),
		Shape: named, ConstraintID: sd.ID, Severity: sev, Message: msg,
	}
}

func structural(code, msg, named string, fp []any) Violation {
	return Violation{
		FocusPath: copyPath(fp), Shape: named,
		Severity: SeverityViolation, Message: msg, Code: code,
	}
}

// ---------------------------------------------------------------- evaluation

// chargeShape is G1's single point of definition: one step per activation.
func (e *Evaluator) chargeShape() { e.m.ChargeEvent(EventShapeActivation, 0) }

// shape evaluates one shape activation against val at fp. primitiveDone is
// true when the caller already judged sd's own kind and keywords (an inline
// field — G5).
func (e *Evaluator) shape(c *collector, sd *ShapeDecl, named string, fp []any, val any, primitiveDone bool) {
	e.chargeShape()
	if !primitiveDone { // a named shape: one level deeper (G8)
		e.depth++
		defer func() { e.depth-- }()
	}
	if !primitiveDone && e.opts.MaxShapeDepth > 0 && e.depth > e.opts.MaxShapeDepth {
		c.emit(structural(CodeShapeDepthExceeded, msgShapeDepthExceeded, named, fp))
		return
	}
	if !primitiveDone && !e.primitive(c, sd, named, fp, val, fp, nil) {
		return
	}
	mp, _ := val.(*jaxson.Object)

	for _, name := range sd.Required {
		if mp == nil || !mp.Has(name) {
			v := e.finding(sd, named, fp, []any{name}, fmt.Sprintf("required member %q is absent", name))
			if id, ok := sd.RequiredIds[name]; ok {
				v.ConstraintID = id.ID
			}
			c.emit(v)
		}
	}
	for _, name := range sortedFieldNames(sd.Fields) {
		if v, has := mp.Get(name); has { // a nil *Object has no members
			e.member(c, sd.Fields[name], named, appendPath(fp, name), v, fp, []any{name})
		}
	}
	if arr, ok := val.([]any); ok && sd.Items != nil {
		for i, el := range arr {
			p := appendPath(fp, i)
			e.member(c, *sd.Items, named, p, el, p, nil)
		}
	}
	if sd.Kind == "object" && sd.Closed && mp != nil {
		ignored := map[string]bool{}
		for _, n := range sd.IgnoredProperties {
			ignored[n] = true
		}
		for _, k := range mp.SortedKeys() {
			if _, declared := sd.Fields[k]; !declared && !ignored[k] {
				c.emit(e.finding(sd, named, fp, []any{k}, fmt.Sprintf("member %q is not allowed", k)))
			}
		}
	}
	for _, qk := range sortedQualifiedKeys(sd.Qualified) {
		q := sd.Qualified[qk]
		count := 0
		e.each(val, fp, func(p []any, el any) {
			if e.probe(q.Target, named, p, el) {
				count++
			}
		})
		if (q.Min != nil && int64(count) < *q.Min) || (q.Max != nil && int64(count) > *q.Max) {
			c.emit(e.finding(sd, named, fp, nil, qualifiedMessage(q, count)))
		}
	}
	e.combinators(c, sd, named, fp, val)
	for _, ck := range sd.Check {
		if !e.runCheck(ck, fp, val) {
			v := e.finding(sd, named, fp, nil, "check failed")
			v.ConstraintID = ck.ID
			c.emit(v)
		}
	}
}

func (e *Evaluator) combinators(c *collector, sd *ShapeDecl, named string, fp []any, val any) {
	if len(sd.And) > 0 {
		for _, fd := range sd.And {
			if !e.probe(fd, named, fp, val) {
				c.emit(e.finding(sd, named, fp, nil, "does not satisfy every alternative of and"))
				break
			}
		}
	}
	if len(sd.Or) > 0 {
		any1 := false
		for _, fd := range sd.Or {
			if e.probe(fd, named, fp, val) {
				any1 = true
				break
			}
		}
		if !any1 {
			c.emit(e.finding(sd, named, fp, nil, "does not satisfy any alternative of or"))
		}
	}
	if len(sd.Xone) > 0 {
		n := 0
		for _, fd := range sd.Xone {
			if e.probe(fd, named, fp, val) {
				n++
				if n >= 2 {
					break
				}
			}
		}
		if n != 1 {
			c.emit(e.finding(sd, named, fp, nil, "does not satisfy exactly one alternative of xone"))
		}
	}
	for _, fd := range sd.Not {
		if e.probe(fd, named, fp, val) {
			c.emit(e.finding(sd, named, fp, nil, "satisfies a shape it must not satisfy (not)"))
			break
		}
	}
}

// member evaluates one field, item, or alternative. path is the value's own
// address; attachFP/attachCP is where an inline shape's own kind/keyword
// findings attach (G4/G5).
func (e *Evaluator) member(c *collector, fd FieldDecl, named string, path []any, v any, attachFP, attachCP []any) {
	if fd.ShapeRef != "" {
		ref := e.namedShape(fd.ShapeRef)
		e.shape(c, &ref, fd.ShapeRef, path, v, false)
		return
	}
	in := fd.Inline
	if !e.primitive(c, in, named, path, v, attachFP, attachCP) {
		return
	}
	if hasStructure(in) {
		e.shape(c, in, named, path, v, true)
	}
}

func (e *Evaluator) namedShape(name string) ShapeDecl {
	sd, ok := e.reg.Shapes[name]
	if !ok {
		jaxson.Fail(CatShapeError, "", "unknown shape %q", name)
	}
	return sd
}

// primitive judges sd's own kind and primitive keywords against val. It
// returns false only when the kind itself is wrong (or a reference
// dangles), meaning nothing structural can sensibly follow; a failed
// keyword is reported but evaluation continues.
func (e *Evaluator) primitive(c *collector, sd *ShapeDecl, named string, path []any, val any, attachFP, attachCP []any) bool {
	switch sd.Kind {
	case "", "node", "any":
		return true
	case "reference":
		return e.reference(c, sd, named, path, val)
	}
	if got := jaxson.TypeName(val); got != sd.Kind {
		c.emit(e.finding(sd, named, attachFP, attachCP, fmt.Sprintf("expected %s, got %s", sd.Kind, got)))
		return false
	}
	if why := keywordFailure(sd, val); why != "" {
		c.emit(e.finding(sd, named, attachFP, attachCP, why))
	}
	return true
}

// reference resolves a reference-kind value through the IndexResolver. Findings attach to the value's own path: section 5's
// reporting rule, "every dangling element gets its own violation with its
// own focusPath", applied to fields and items alike.
func (e *Evaluator) reference(c *collector, sd *ShapeDecl, named string, path []any, val any) bool {
	switch val.(type) {
	case *jaxson.Object, []any:
		jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "%s: a reference must be a scalar, got %s", pathString(path), jaxson.TypeName(val))
	}
	if e.opts.Resolver == nil {
		e.indices()
	}
	t := ReferenceTarget{Index: sd.RefIndex, Relation: sd.RefRelation, Of: sd.RefOf, By: sd.RefBy}
	if !e.opts.Resolver.Lookup(t, val) {
		c.emit(structural(CodeDanglingReference, msgDanglingReference, named, path))
		return false
	}
	return true
}

// probe reports whether v conforms to fd without recording findings. It
// stops at the first violation-severity finding (G7); steps and depth are
// charged exactly as for a real evaluation.
func (e *Evaluator) probe(fd FieldDecl, named string, path []any, v any) (ok bool) {
	pc := &collector{stop: true}
	defer func() {
		if r := recover(); r != nil {
			if _, is := r.(stopSignal); is {
				ok = false
				return
			}
			panic(r)
		}
	}()
	e.member(pc, fd, named, path, v, path, nil)
	return true
}

// each visits a collection's elements: an array in index order, an object's
// member values in code-point key order; anything else has none.
func (e *Evaluator) each(val any, fp []any, fn func(path []any, el any)) {
	switch t := val.(type) {
	case []any:
		for i, el := range t {
			fn(appendPath(fp, i), el)
		}
	case *jaxson.Object:
		for _, k := range t.SortedKeys() {
			v, _ := t.Get(k)
			fn(appendPath(fp, k), v)
		}
	}
}

// runCheck runs a shape's `check` island with local.focus bound to val.
func (e *Evaluator) runCheck(ck CombinatorCheck, fp []any, val any) bool {
	with, expr := ck.With, ck.Expr
	if ck.ComputeRef != "" {
		cd := e.reg.Computes[ck.ComputeRef]
		with, expr = cd.With, cd.Expr
	}
	body := map[string]any{"expr": expr}
	if with != nil {
		body["with"] = with
	}
	var res any
	e.ambient.with(fp, func() {
		e.m.WithLocal("focus", val, func() { res = e.m.RunCompute(body) })
	})
	b, ok := res.(bool)
	if !ok {
		jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", "a check must reduce to a boolean, got %s", jaxson.TypeName(res))
	}
	return b
}

// ---------------------------------------------------------------- helpers

// hasStructure: does an inline shape declare anything beyond its own
// kind/keywords that must be judged as a nested focus (G5)? An object kind
// always does, since closed is the default posture.
func hasStructure(s *ShapeDecl) bool {
	return s.Kind == "object" || len(s.Fields) > 0 || len(s.Required) > 0 ||
		s.Items != nil || len(s.Qualified) > 0 ||
		len(s.And)+len(s.Or)+len(s.Xone)+len(s.Not)+len(s.Check) > 0
}

func keywordFailure(sd *ShapeDecl, val any) string {
	k := sd.Keywords
	switch sd.Kind {
	case "string":
		return jaxson.CheckStringLen(val.(string), ratOfCount(k.MinLen), ratOfCount(k.MaxLen), k.Enum)
	case "number":
		if why := jaxson.CheckNumber(val.(*big.Rat), k.WantInt, k.MinRat, k.MaxRat); why != "" {
			return why
		}
		return enumFailure(k.Enum, val)
	case "array":
		return jaxson.CheckArrayLen(len(val.([]any)), ratOfCount(k.MinItems), ratOfCount(k.MaxItems))
	case "boolean", "null":
		return enumFailure(k.Enum, val)
	}
	return ""
}

func enumFailure(enum []any, val any) string {
	if enum == nil {
		return ""
	}
	for _, e := range enum {
		if jaxson.Equal(e, val) {
			return ""
		}
	}
	return "not in enum"
}

func ratOfCount(p *int64) *big.Rat {
	if p == nil {
		return nil
	}
	return big.NewRat(*p, 1)
}

func qualifiedMessage(q QualifiedDecl, count int) string {
	lo, hi := "0", "no limit"
	if q.Min != nil {
		lo = fmt.Sprint(*q.Min)
	}
	if q.Max != nil {
		hi = fmt.Sprint(*q.Max)
	}
	return fmt.Sprintf("%d element(s) match the qualified shape; between %s and %s are required", count, lo, hi)
}

func sortedFieldNames(m map[string]FieldDecl) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedQualifiedKeys(m map[string]QualifiedDecl) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
