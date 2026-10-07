// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Phase 3 tests (plan section 6). One test per row of core section 10's
// granularity table, plus the rules the spec makes mandatory (minimal
// combinator evaluation and its step counts, gate order, depth) and the
// numeric-exactness requirement.
//
// fakeResolver below isolates shape evaluation from index building: it is
// handed to the evaluator wherever a test is about shapes, not about
// indices. The real IndexSet is exercised in indices_test.go, and by the
// no-resolver case in TestReferenceRaisedErrors.

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ha1tch/jaxson/pkg/jaxson"
)

// fakeResolver maps an index/relation name to the set of keys it contains.
type fakeResolver map[string]map[string]bool

func (f fakeResolver) Lookup(t ReferenceTarget, key any) bool {
	name := t.Index
	if name == "" {
		name = t.Relation
	}
	switch k := key.(type) {
	case string:
		return f[name][k]
	case *big.Rat:
		return f[name][jaxson.FormatDecimal(k)]
	}
	return false
}

func parseReg(t *testing.T, pkgJSON string) *Registries {
	t.Helper()
	reg, err := ParseRegistries(decode(t, pkgJSON))
	if err != nil {
		t.Fatalf("package rejected by Phase 2: %v", err)
	}
	return reg
}

func evalAt(t *testing.T, reg *Registries, inputJSON string, limit int, opts Options, shape string, segs ...any) (Report, *jaxson.Err) {
	t.Helper()
	m := jaxson.NewMachine(decode(t, inputJSON), nil, nil, limit, nil, nil)
	return NewEvaluator(m, reg, opts).Check(shape, "input", segs)
}

// stepsUsed finds the exact step count by finding the smallest limit under
// which the evaluation does not fail with RESOURCE_ERROR/STEPS.
func stepsUsed(t *testing.T, reg *Registries, inputJSON string, opts Options, shape string, segs ...any) int {
	t.Helper()
	for n := 1; n < 500; n++ {
		_, err := evalAt(t, reg, inputJSON, n, opts, shape, segs...)
		if err == nil {
			return n
		}
		if err.Code != "STEPS" {
			t.Fatalf("unexpected error while measuring steps: %v", err)
		}
	}
	t.Fatalf("evaluation never fit in 499 steps")
	return 0
}

type wv struct {
	focus, cp []any
	shape     string
	id, sev   string
	code      string
	msg       string // exact when non-empty
}

func assertViolations(t *testing.T, rep Report, want []wv) {
	t.Helper()
	if len(rep.Violations) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(rep.Violations), len(want), rep.Violations)
	}
	for i, w := range want {
		g := rep.Violations[i]
		sev := w.sev
		if sev == "" {
			sev = SeverityViolation
		}
		if !reflect.DeepEqual(g.FocusPath, w.focus) || !reflect.DeepEqual(g.ConstraintPath, w.cp) ||
			g.Shape != w.shape || g.ConstraintID != w.id || g.Severity != sev || g.Code != w.code ||
			(w.msg != "" && g.Message != w.msg) {
			t.Errorf("violation %d:\n got  %+v\n want %+v", i, g, w)
		}
	}
}

func p(segs ...any) []any { return segs }

// ---------------------------------------------------------------- section 10 rows

const pkgDoc = `{
  "shapes": {
    "Doc": {
      "kind": "object",
      "fields": {
        "id":     {"kind": "string", "minLen": 1},
        "name":   {"kind": "string"},
        "status": {"kind": "string", "enum": ["draft", "placed"], "id": "STATUS_ENUM", "message": "status must be draft or placed"}
      },
      "required": ["id", "name"],
      "requiredIds": {"id": "DOC_NEEDS_ID"}
    }
  }
}`

// Row 1: one violation per absent required member, in listed order, with
// the requiredIds id where declared.
func TestRowRequiredAbsent(t *testing.T) {
	rep, err := evalAt(t, parseReg(t, pkgDoc), `{"doc": {}}`, 1000, Options{}, "Doc", "doc")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{
		{focus: p("input", "doc"), cp: p("id"), shape: "Doc", id: "DOC_NEEDS_ID"},
		{focus: p("input", "doc"), cp: p("name"), shape: "Doc"},
	})
}

// Row 2: a field failing its own primitive check is one violation on the
// parent, with constraintPath = [field], the field's own id and message.
func TestRowFieldPrimitive(t *testing.T) {
	rep, err := evalAt(t, parseReg(t, pkgDoc), `{"doc": {"id": "a", "name": "n", "status": "shipped"}}`, 1000, Options{}, "Doc", "doc")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{
		{focus: p("input", "doc"), cp: p("status"), shape: "Doc", id: "STATUS_ENUM", msg: "status must be draft or placed"},
	})
	rep, _ = evalAt(t, parseReg(t, pkgDoc), `{"doc": {"id": "", "name": "n"}}`, 1000, Options{}, "Doc", "doc")
	assertViolations(t, rep, []wv{{focus: p("input", "doc"), cp: p("id"), shape: "Doc", msg: "too short"}})
}

const pkgNested = `{
  "shapes": {
    "Order": {"kind": "object", "fields": {"line": {"shape": "Line"}}, "required": ["line"]},
    "Line":  {"kind": "object", "fields": {"qty": {"kind": "number", "int": true, "min": 1}}, "required": ["qty"]}
  }
}`

// Row 3: a nested shape is its own target — own focusPath, and NO
// constraintPath finding on the parent for the field that holds it.
func TestRowNestedShapeIsItsOwnTarget(t *testing.T) {
	reg := parseReg(t, pkgNested)
	rep, err := evalAt(t, reg, `{"order": {"line": {}}}`, 1000, Options{}, "Order", "order")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{{focus: p("input", "order", "line"), cp: p("qty"), shape: "Line"}})

	rep, _ = evalAt(t, reg, `{"order": {"line": {"qty": 0}}}`, 1000, Options{}, "Order", "order")
	assertViolations(t, rep, []wv{{focus: p("input", "order", "line"), cp: p("qty"), shape: "Line", msg: "below minimum"}})
}

const pkgCheck = `{
  "computes": {"nonNeg": {"with": {"x": {"$path": ["local", "focus", "total"]}}, "expr": ["ge", {"$v": "x"}, 0]}},
  "shapes": {
    "Order": {
      "kind": "object",
      "fields": {"total": {"kind": "number"}},
      "required": ["total"],
      "check": {"with": {"total": {"$path": ["local", "focus", "total"]}}, "expr": ["ge", {"$v": "total"}, 0], "id": "TOTAL_NONNEG"},
      "message": "total must not be negative"
    },
    "ViaCompute": {
      "kind": "object",
      "fields": {"total": {"kind": "number"}},
      "check": {"compute": "nonNeg"}
    }
  }
}`

// Row 4: a failing check is one violation at the shape's own focus, no
// constraintPath (absent from the rendered value, not null).
func TestRowCheck(t *testing.T) {
	reg := parseReg(t, pkgCheck)
	rep, err := evalAt(t, reg, `{"order": {"total": -1}}`, 1000, Options{}, "Order", "order")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{{focus: p("input", "order"), shape: "Order", id: "TOTAL_NONNEG", msg: "total must not be negative"}})

	v := rep.Violations[0].Value()
	if _, has := v["constraintPath"]; has {
		t.Errorf("constraintPath must be omitted, not null: %v", v)
	}
	if v["code"] != nil || v["kind"] != "constraint" || v["constraintId"] != "TOTAL_NONNEG" || v["severity"] != "violation" {
		t.Errorf("bad rendered violation: %v", v)
	}
	if fp := v["focusPath"].([]any); len(fp) != 2 || fp[0] != "input" || fp[1] != "order" {
		t.Errorf("bad focusPath: %v", fp)
	}

	if rep, _ = evalAt(t, reg, `{"order": {"total": 5}}`, 1000, Options{}, "Order", "order"); !rep.Conforms() || len(rep.Violations) != 0 {
		t.Errorf("a passing check must produce nothing: %+v", rep.Violations)
	}
	// A check naming a computes entry behaves identically.
	rep, _ = evalAt(t, reg, `{"order": {"total": -1}}`, 1000, Options{}, "ViaCompute", "order")
	assertViolations(t, rep, []wv{{focus: p("input", "order"), shape: "ViaCompute"}})
}

const pkgComb = `{
  "shapes": {
    "Pass":     {"kind": "string"},
    "Fail":     {"kind": "string", "minLen": 5},
    "WarnOnly": {"kind": "string", "minLen": 5, "severity": "warning"},
    "AndFirstFails": {"and": [{"shape": "Fail"}, {"shape": "Pass"}, {"shape": "Pass"}]},
    "AndAllPass":    {"and": [{"shape": "Pass"}, {"shape": "Pass"}, {"shape": "Pass"}]},
    "OrFirstPasses": {"or":  [{"shape": "Pass"}, {"shape": "Fail"}, {"shape": "Fail"}]},
    "OrNonePass":    {"or":  [{"shape": "Fail"}, {"shape": "Fail"}]},
    "XoneTwoEarly":  {"xone": [{"shape": "Pass"}, {"shape": "Pass"}, {"shape": "Pass"}]},
    "XoneOne":       {"xone": [{"shape": "Fail"}, {"shape": "Pass"}, {"shape": "Fail"}]},
    "NotPass":       {"not": {"shape": "Pass"}},
    "AndWarn":       {"and": [{"shape": "WarnOnly"}]}
  }
}`

// Row 5 and core section 4's mandatory minimal evaluation: exactly one
// violation per failing combinator, and the number of alternatives
// evaluated (hence steps charged, G1) is fixed by the short-circuit rule.
func TestRowCombinatorsAndMinimalEvaluation(t *testing.T) {
	reg := parseReg(t, pkgComb)
	in := `{"v": "abc"}`
	cases := []struct {
		shape string
		viol  int
		steps int
	}{
		{"AndFirstFails", 1, 2}, // root + Fail; stops at the first failure
		{"AndAllPass", 0, 4},    // root + all three
		{"OrFirstPasses", 0, 2}, // root + Pass; stops at the first pass
		{"OrNonePass", 1, 3},    // root + both
		{"XoneTwoEarly", 1, 3},  // root + two; stops at the second match
		{"XoneOne", 0, 4},       // root + all three; never reaches two
		{"NotPass", 1, 2},       // root + the one inner shape
		{"AndWarn", 0, 2},       // a warning never fails an alternative (G7)
	}
	for _, c := range cases {
		t.Run(c.shape, func(t *testing.T) {
			rep, err := evalAt(t, reg, in, 1000, Options{}, c.shape, "v")
			if err != nil {
				t.Fatal(err)
			}
			if len(rep.Violations) != c.viol {
				t.Fatalf("got %d violations, want %d: %+v", len(rep.Violations), c.viol, rep.Violations)
			}
			if c.viol == 1 {
				assertViolations(t, rep, []wv{{focus: p("input", "v"), shape: c.shape}})
			}
			if got := stepsUsed(t, reg, in, Options{}, c.shape, "v"); got != c.steps {
				t.Errorf("steps = %d, want %d", got, c.steps)
			}
		})
	}
}

const pkgQual = `{
  "shapes": {
    "Doc": {"kind": "object", "fields": {"tags": {
      "kind": "array",
      "items": {"kind": "string"},
      "qualified": {"shape": {"kind": "string", "enum": ["x"]}, "min": 1, "max": 2}
    }}}
  }
}`

// qualified (G3): one violation at the collection's focus when the count of
// matching elements is out of range; counting evaluates every element.
func TestQualified(t *testing.T) {
	reg := parseReg(t, pkgQual)
	focus := p("input", "doc", "tags")
	for _, c := range []struct {
		in   string
		want []wv
	}{
		{`["x", "y"]`, nil},
		{`["y"]`, []wv{{focus: focus, shape: "Doc"}}},
		{`["x", "x", "x"]`, []wv{{focus: focus, shape: "Doc"}}},
		// An item of the wrong kind: its own finding first (items), then the
		// qualified count (the element cannot match) — G6 order.
		{`[1]`, []wv{{focus: p("input", "doc", "tags", 0), shape: "Doc", msg: "expected string, got number"}, {focus: focus, shape: "Doc"}}},
	} {
		rep, err := evalAt(t, reg, `{"doc": {"tags": `+c.in+`}}`, 1000, Options{}, "Doc", "doc")
		if err != nil {
			t.Fatal(err)
		}
		assertViolations(t, rep, c.want)
	}
	// Doc activation + the structured `tags` field; inline primitive probes are free (G1).
	if got := stepsUsed(t, reg, `{"doc": {"tags": ["x", "y"]}}`, Options{}, "Doc", "doc"); got != 2 {
		t.Errorf("steps = %d, want 2", got)
	}
}

// closed (G2): one violation per unexpected member, in code-point order;
// ignoredProperties are exempt.
func TestClosedAndIgnoredProperties(t *testing.T) {
	reg := parseReg(t, `{"shapes": {"Doc": {"kind": "object", "ignoredProperties": ["$meta"], "fields": {"a": {"kind": "string"}}}}}`)
	rep, err := evalAt(t, reg, `{"doc": {"a": "x", "$meta": 1, "zzz": 1, "b": 2}}`, 1000, Options{}, "Doc", "doc")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{
		{focus: p("input", "doc"), cp: p("b"), shape: "Doc"},
		{focus: p("input", "doc"), cp: p("zzz"), shape: "Doc"},
	})
	reg = parseReg(t, `{"shapes": {"Doc": {"kind": "object", "closed": false, "fields": {"a": {"kind": "string"}}}}}`)
	rep, _ = evalAt(t, reg, `{"doc": {"a": "x", "extra": 1}}`, 1000, Options{}, "Doc", "doc")
	assertViolations(t, rep, nil)
}

// The focus node's own kind (G4): one violation at the focus, no constraintPath.
func TestFocusKindMismatch(t *testing.T) {
	reg := parseReg(t, `{"shapes": {"S": {"kind": "string"}}}`)
	rep, err := evalAt(t, reg, `{"v": 3}`, 1000, Options{}, "S", "v")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{{focus: p("input", "v"), shape: "S", msg: "expected string, got number"}})
}

// ---------------------------------------------------------------- severity, modes

const pkgGate = `{
  "shapes": {
    "G": {
      "kind": "object",
      "fields": {"n": {"kind": "number", "min": 5}, "r": {"kind": "number"}, "s": {"kind": "number"}},
      "required": ["r", "s"],
      "check": {"with": {"n": {"$path": ["local", "focus", "n"]}}, "expr": ["ge", {"$v": "n"}, 100], "id": "BIG"}
    },
    "Soft": {
      "kind": "object",
      "fields": {"n": {"kind": "number", "min": 5, "severity": "warning"}}
    }
  }
}`

// Report mode collects everything, in the order required, fields, check.
func TestReportModeCollectsAllInOrder(t *testing.T) {
	rep, err := evalAt(t, parseReg(t, pkgGate), `{"g": {"n": 1}}`, 1000, Options{}, "G", "g")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{
		{focus: p("input", "g"), cp: p("r"), shape: "G"},
		{focus: p("input", "g"), cp: p("s"), shape: "G"},
		{focus: p("input", "g"), cp: p("n"), shape: "G"},
		{focus: p("input", "g"), shape: "G", id: "BIG"},
	})
}

// Gate mode aborts on the first failure in the fixed order (core section
// 10): required, then fields, then check.
func TestGateModeFixedOrder(t *testing.T) {
	reg := parseReg(t, pkgGate)
	gate := Options{Mode: ModeGate}
	for _, c := range []struct{ in, sub string }{
		{`{"g": {"n": 1}}`, "input.g.r"},                    // required beats field and check
		{`{"g": {"n": 1, "r": 1, "s": 1}}`, "input.g.n"},    // field beats check
		{`{"g": {"n": 6, "r": 1, "s": 1}}`, "check failed"}, // check last
	} {
		_, err := evalAt(t, reg, c.in, 1000, gate, "G", "g")
		if err == nil || err.Cat != CatValidationError || err.Code != CodeShapeMismatch || !strings.Contains(err.Msg, c.sub) {
			t.Errorf("input %s: err = %v, want %s/%s containing %q", c.in, err, CatValidationError, CodeShapeMismatch, c.sub)
		}
	}
	if _, err := evalAt(t, reg, `{"g": {"n": 100, "r": 1, "s": 1}}`, 1000, gate, "G", "g"); err != nil {
		t.Errorf("conforming input must not raise: %v", err)
	}
}

// Only violation-severity findings gate or flip conforms (core section 10).
func TestWarningNeverGatesOrFlipsConforms(t *testing.T) {
	reg := parseReg(t, pkgGate)
	for _, mode := range []Mode{ModeReport, ModeGate} {
		rep, err := evalAt(t, reg, `{"s": {"n": 1}}`, 1000, Options{Mode: mode}, "Soft", "s")
		if err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		assertViolations(t, rep, []wv{{focus: p("input", "s"), cp: p("n"), shape: "Soft", sev: SeverityWarning}})
		if !rep.Conforms() {
			t.Errorf("mode %d: a warning must not flip conforms", mode)
		}
		if rep.Value()["conforms"] != true {
			t.Errorf("mode %d: rendered conforms must be true", mode)
		}
	}
}

// ---------------------------------------------------------------- references (fake resolver)

const pkgRef = `{
  "indices": {"tagsById": {"source": {"$path": ["input", "tags"]}, "key": {"$path": ["local", "item", "id"]}}},
  "shapes": {
    "Doc": {"kind": "object", "fields": {
      "owner":  {"kind": "reference", "index": "tagsById"},
      "tagIds": {"kind": "array", "items": {"kind": "reference", "index": "tagsById"}}
    }}
  }
}`

// The evaluator itself refuses a non-scalar reference value; it does not
// leave that to whatever resolver it was given.
func TestReferenceNonScalarIsTypeErrorWhateverTheResolver(t *testing.T) {
	reg := parseReg(t, pkgRef)
	opts := Options{Resolver: fakeResolver{"tagsById": {"a": true}}}
	for _, owner := range []string{`["a"]`, `{"id": "a"}`} {
		_, err := evalAt(t, reg, `{"doc": {"owner": `+owner+`, "tagIds": []}}`, 1000, opts, "Doc", "doc")
		if err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != "TYPE_ERROR" {
			t.Errorf("owner %s: got %v, want EXECUTION_ERROR/TYPE_ERROR", owner, err)
		}
	}
}

func TestReferencesDanglingEachOwnViolation(t *testing.T) {
	reg := parseReg(t, pkgRef)
	opts := Options{Resolver: fakeResolver{"tagsById": {"a": true, "b": true}}}
	in := `{"doc": {"owner": "a", "tagIds": ["a", "zz", "b", "qq", "yy"]}}`

	rep, err := evalAt(t, reg, in, 1000, opts, "Doc", "doc")
	if err != nil {
		t.Fatal(err)
	}
	d := func(i int) wv {
		return wv{focus: p("input", "doc", "tagIds", i), shape: "Doc", code: CodeDanglingReference, msg: msgDanglingReference}
	}
	assertViolations(t, rep, []wv{d(1), d(3), d(4)})
	if rep.Violations[0].Kind() != "structural" || rep.Conforms() {
		t.Errorf("a dangling reference is a structural violation-severity finding")
	}
	if v := rep.Violations[0].Value(); v["code"] != CodeDanglingReference || v["kind"] != "structural" {
		t.Errorf("bad rendering: %v", v)
	}

	// A plain reference field reports on its own path, not on the parent.
	rep, _ = evalAt(t, reg, `{"doc": {"owner": "nope", "tagIds": []}}`, 1000, opts, "Doc", "doc")
	assertViolations(t, rep, []wv{{focus: p("input", "doc", "owner"), shape: "Doc", code: CodeDanglingReference}})

	// Gate: the first dangling element in array order raises.
	_, err = evalAt(t, reg, in, 1000, Options{Mode: ModeGate, Resolver: opts.Resolver}, "Doc", "doc")
	if err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != CodeDanglingReference || !strings.Contains(err.Msg, "tagIds[1]") {
		t.Errorf("gate err = %v", err)
	}
}

func TestReferenceRaisedErrors(t *testing.T) {
	reg := parseReg(t, pkgRef)
	opts := Options{Resolver: fakeResolver{"tagsById": {"a": true}}}
	if _, err := evalAt(t, reg, `{"doc": {"owner": {"x": 1}}}`, 1000, opts, "Doc", "doc"); err == nil || err.Code != "TYPE_ERROR" {
		t.Errorf("non-scalar reference: err = %v, want TYPE_ERROR", err)
	}
	// With no resolver configured the evaluator builds the declared index
	// from the data itself.
	in := `{"tags": [{"id": "a"}], "doc": {"owner": "a", "tagIds": ["a", "zz"]}}`
	rep, err := evalAt(t, reg, in, 1000, Options{}, "Doc", "doc")
	if err != nil {
		t.Fatalf("real index set: %v", err)
	}
	assertViolations(t, rep, []wv{{focus: p("input", "doc", "tagIds", 1), shape: "Doc", code: CodeDanglingReference}})
}

// ---------------------------------------------------------------- depth, steps, exactness

const pkgTree = `{
  "limits": {"maxShapeDepth": 3},
  "shapes": {
    "Cat": {"kind": "object", "required": ["name"], "fields": {
      "name": {"kind": "string"},
      "children": {"kind": "array", "items": {"shape": "Cat"}}
    }}
  }
}`

// maxShapeDepth counts named-shape descents (G8): the root is depth 1, so
// a fourth nested Cat exceeds a bound of 3. The overrun is one structural
// finding at the node that would have been entered; its subtree is not
// visited; a shallow sibling is unaffected (depth is restored).
func TestShapeDepthExceeded(t *testing.T) {
	reg := parseReg(t, pkgTree)
	opts := Options{MaxShapeDepth: 3}
	in := `{"t": {"name": "a", "children": [
	  {"name": "b", "children": [{"name": "c", "children": [{"name": "d", "children": []}]}]},
	  {"name": "sibling", "children": []}
	]}}`
	rep, err := evalAt(t, reg, in, 1000, opts, "Cat", "t")
	if err != nil {
		t.Fatal(err)
	}
	assertViolations(t, rep, []wv{{
		focus: p("input", "t", "children", 0, "children", 0, "children", 0),
		shape: "Cat", code: CodeShapeDepthExceeded, msg: msgShapeDepthExceeded,
	}})

	opts.MaxShapeDepth = 4
	if rep, _ = evalAt(t, reg, in, 1000, opts, "Cat", "t"); len(rep.Violations) != 0 {
		t.Errorf("depth 4 admits this tree: %+v", rep.Violations)
	}
	gate := Options{Mode: ModeGate, MaxShapeDepth: 3}
	if _, err = evalAt(t, reg, in, 1000, gate, "Cat", "t"); err == nil || err.Cat != "EXECUTION_ERROR" || err.Code != CodeShapeDepthExceeded {
		t.Errorf("gate err = %v", err)
	}
}

func TestStepLimitSurfaces(t *testing.T) {
	_, err := evalAt(t, parseReg(t, pkgComb), `{"v": "abc"}`, 1, Options{}, "AndAllPass", "v")
	if err == nil || err.Cat != "RESOURCE_ERROR" || err.Code != "STEPS" {
		t.Errorf("err = %v, want RESOURCE_ERROR/STEPS", err)
	}
}

// Number keywords use the exact decimal, never a float conversion: a value
// 1e-20 above the bound must fail.
func TestNumberBoundsAreExact(t *testing.T) {
	reg := parseReg(t, `{"shapes": {"N": {"kind": "number", "min": 0.1, "max": 0.3}}}`)
	for _, c := range []struct {
		v    string
		bad  bool
		want string
	}{
		{"0.3", false, ""},
		{"0.1", false, ""},
		{"0.30000000000000000001", true, "above maximum"},
		{"0.09999999999999999999", true, "below minimum"},
	} {
		rep, err := evalAt(t, reg, `{"v": `+c.v+`}`, 1000, Options{}, "N", "v")
		if err != nil {
			t.Fatal(err)
		}
		if c.bad {
			assertViolations(t, rep, []wv{{focus: p("input", "v"), shape: "N", msg: c.want}})
		} else if len(rep.Violations) != 0 {
			t.Errorf("%s should conform: %+v", c.v, rep.Violations)
		}
	}
}

func TestCheckErrors(t *testing.T) {
	reg := parseReg(t, pkgDoc)
	if _, err := evalAt(t, reg, `{"doc": {}}`, 1000, Options{}, "Nope", "doc"); err == nil || err.Cat != CatShapeError {
		t.Errorf("unknown shape: err = %v", err)
	}
	if _, err := evalAt(t, reg, `{"doc": {}}`, 1000, Options{}, "Doc", "missing"); err == nil || err.Code != "MISSING_PATH" {
		t.Errorf("missing path: err = %v", err)
	}
}

// The Phase 2 parser now accepts "id" and keeps exact numeric bounds.
func TestParserCarriesIDAndExactBounds(t *testing.T) {
	reg := parseReg(t, `{"shapes": {"N": {"kind": "number", "id": "NUM", "min": 0.1}}}`)
	n := reg.Shapes["N"]
	if n.ID != "NUM" {
		t.Errorf("ID = %q", n.ID)
	}
	if n.Keywords.MinRat == nil || n.Keywords.MinRat.Cmp(big.NewRat(1, 10)) != 0 {
		t.Errorf("MinRat = %v, want exactly 1/10", n.Keywords.MinRat)
	}
	if _, err := ParseRegistries(decode(t, `{"shapes": {"N": {"kind": "number", "id": 5}}}`)); err == nil {
		t.Errorf("a non-string id must be a SHAPE_ERROR")
	}
}

// The shape-activation charge is priced by the machine's cost table
// (docs/proposals/step-cost-model.md, CM-2): the same evaluation costs
// activations x the table's price, and under `unit` the price is one, which
// is what every step-count test above relies on.
func TestShapeActivationFollowsTheCostTable(t *testing.T) {
	reg := parseReg(t, pkgComb)
	run := func(tbl jaxson.CostTable, limit int, shape string) (int64, *jaxson.Err) {
		m := jaxson.NewMachine(decode(t, `{"v": "abc"}`), nil, nil, limit, nil, nil)
		if e := m.SetCostTable(tbl); e != nil {
			t.Fatal(e)
		}
		_, err := NewEvaluator(m, reg, Options{}).Check(shape, "input", p("v"))
		return m.Steps(), err
	}
	five := jaxson.UnitTable()
	five.Event[EventShapeActivation] = jaxson.Cost{Base: 5}

	for _, c := range []struct {
		shape string
		acts  int64
	}{{"AndAllPass", 4}, {"OrNonePass", 3}, {"NotPass", 2}} {
		if got, err := run(jaxson.UnitTable(), 1000, c.shape); err != nil || got != c.acts {
			t.Errorf("%s under unit: %d steps, err %v, want %d", c.shape, got, err, c.acts)
		}
		if got, err := run(five, 1000, c.shape); err != nil || got != 5*c.acts {
			t.Errorf("%s at 5 per activation: %d steps, err %v, want %d", c.shape, got, err, 5*c.acts)
		}
	}
	// Charge before effect: a limit of 12 admits two activations at 5 and
	// refuses the third, leaving the total at 10.
	got, err := run(five, 12, "AndAllPass")
	if err == nil || err.Cat != "RESOURCE_ERROR" || err.Code != "STEPS" || got != 10 {
		t.Errorf("limit 12: total %d, err %v; want 10 and RESOURCE_ERROR/STEPS", got, err)
	}
}
