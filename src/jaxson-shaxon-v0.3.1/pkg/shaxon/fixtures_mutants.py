#!/usr/bin/env python3
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
"""Mutation check for the conformance fixtures (shaxon-v0.3.1-fixtures.json).

Breaks one rule of the engine at a time in a scratch copy of the module and
requires TestFixtures to fail. A mutant that survives is either a rule no
fixture exercises (add a fixture) or a rule the core leaves open (then the
mutant belongs in LEFT_OPEN with the reason, so the gap stays visible).

    python3 pkg/shaxon/fixtures_mutants.py            # all mutants
    python3 pkg/shaxon/fixtures_mutants.py -v         # also name a fixture that killed each

Each mutant is (file, text, replacement, occurrences, what is broken). The text
must occur exactly that many times in the file, so an anchor that no longer
matches fails loudly instead of testing nothing. A baseline run of the
untouched copy must pass first. Exit status 0 if every mutant is killed, 1 if
one survives, 2 on a bad anchor, a failing baseline or a missing Go toolchain.

It runs `go test` once per mutant; it is not part of the default test run.
"""
import os, re, shutil, subprocess, sys, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
MODULE = os.path.abspath(os.path.join(HERE, "..", ".."))
P = "pkg/shaxon/"

M = [
    (P + "shapes.go", "\t\t\t\tany1 = true\n\t\t\t\tbreak\n", "\t\t\t\tany1 = true\n", 1,
     "or no longer stops at the first passing alternative"),
    (P + "shapes.go", "does not satisfy every alternative of and\"))\n\t\t\t\tbreak\n", "does not satisfy every alternative of and\"))\n", 1,
     "and no longer stops at the first failing alternative"),
    (P + "shapes.go", "\t\t\t\tif n >= 2 {\n\t\t\t\t\tbreak\n\t\t\t\t}\n", "", 1,
     "xone no longer stops at the second match"),
    (P + "shapes.go", "if n != 1 {", "if n < 1 {", 1, "xone accepts two or more matches"),
    (P + "shapes.go", "int64(count) < *q.Min", "int64(count) <= *q.Min", 1, "qualified minimum is exclusive"),
    (P + "shapes.go", "int64(count) > *q.Max", "int64(count) >= *q.Max", 1, "qualified maximum is exclusive"),
    (P + "shapes.go", "func (e *Evaluator) chargeShape() { e.m.ChargeEvent(EventShapeActivation, 0) }",
     "func (e *Evaluator) chargeShape() {}", 1, "shape evaluation (qualified elements included) is free"),
    (P + "shapes.go", "e.depth > e.opts.MaxShapeDepth {", "e.depth > e.opts.MaxShapeDepth+100 {", 1,
     "maxShapeDepth is not enforced"),
    (P + "shapes.go", "\tfor _, name := range sd.Required {", "\tfor ri := len(sd.Required) - 1; ri >= 0; ri-- {\n\t\tname := sd.Required[ri]", 1,
     "required members are reported in reverse order"),
    (P + "shapes.go", "\tsort.Strings(out)\n\treturn out\n}\n\n", "\tsort.Sort(sort.Reverse(sort.StringSlice(out)))\n\treturn out\n}\n\n", 1,
     "fields are visited in descending order (sortedFieldNames)"),
    (P + "shapes.go", "\tsort.Strings(out)\n\treturn out\n}\n\n",
     "\tsort.Slice(out, func(i, j int) bool {\n\t\ta, b := out[i], out[j]\n\t\tif a != \"\" && b != \"\" && (a[0]|0x20) != (b[0]|0x20) {\n\t\t\treturn (a[0] | 0x20) < (b[0] | 0x20)\n\t\t}\n\t\treturn a < b\n\t})\n\treturn out\n}\n\n", 1,
     "fields are visited case-insensitively instead of by code point"),
    (P + "indices.go", "s.m.ChargeEvent(EventIndexElement, 0)", "_ = s", 1, "index builds are free"),
    (P + "indices.go", "if u.built && !s.stale(u) {", "if u.built {", 1, "an index is never rebuilt, even after its source was written"),
    (P + "indices.go", "if u.built && !s.stale(u) {", "if false {", 1, "an index is rebuilt for every use (reuse removed)"),
    (P + "indices.go", "return strings.Compare(a.flag.text, b.flag.text)", "return -strings.Compare(a.flag.text, b.flag.text)", 1,
     "$indexed visits string keys in descending order"),
    (P + "indices.go", "return compareKeys(a.val(), b.val())", "return -compareKeys(a.val(), b.val())", 1,
     "$indexed visits mixed-type or numeric keys in descending order"),
    (P + "report.go", "\t\tif v.Severity == SeverityViolation {\n\t\t\treturn false", "\t\tif v.Severity != \"\" {\n\t\t\treturn false", 1,
     "a warning or info finding flips conforms"),
    (P + "parse_shapes.go", "out.Or = append(append([]FieldDecl{}, parent.Or...), child.Or...)",
     "out.Or = append(append([]FieldDecl{}, child.Or...), parent.Or...)", 1, "extends pools the child's or-list before the parent's"),
    (P + "parse_shapes.go", "out.Xone = append(append([]FieldDecl{}, parent.Xone...), child.Xone...)",
     "out.Xone = child.Xone", 1, "extends does not pool xone lists"),
    (P + "parse_shapes.go", "\ts.Closed = true\n", "\ts.Closed = false\n", 1, "objects are open by default"),
    (P + "paths.go", "if hop > maxDepth {", "if hop > maxDepth+1 {", 1, "a closure may run one hop past maxDepth"),
    (P + "paths.go", "\tif !plus {\n\t\tout = append(out, Focus", "\tif true {\n\t\tout = append(out, Focus", 1, "$path+ includes its base"),
    (P + "validate.go", "\t\tif !seen[k] {", "\t\tif true {", 1, "unique never flags a repeat"),
    (P + "parse_shapes.go", "out.Closed = parent.Closed || child.Closed", "out.Closed = parent.Closed && child.Closed", 1,
     "extends closes a shape by the most permissive of parent and child (S8)"),
    (P + "parse_shapes.go", "\tcase child.ClosedOverride:\n", "\tcase false && child.ClosedOverride:\n", 1, "closedOverride is ignored"),
    (P + "parse_shapes.go", "if child.AndOverride {", "if false {", 1, "andOverride is ignored"),
    (P + "parse_shapes.go", "if child.OrOverride {", "if false {", 1, "orOverride is ignored"),
    (P + "parse_shapes.go", "if child.XoneOverride {", "if false {", 1, "xoneOverride is ignored"),
    (P + "parse_shapes.go", "if child.NotOverride {", "if false {", 1, "notOverride is ignored"),
    (P + "parse_shapes.go", "if child.CheckOverride != nil {", "if false {", 1, "checkOverride is ignored"),
    (P + "parse_shapes.go", "parent.RequiredIds[k]; collide && !v.Override {", "parent.RequiredIds[k]; collide && false {", 1,
     "a requiredIds collision is silently accepted"),
    (P + "targets.go", "for _, k := range c.SortedKeys() {",
     "for _, k := range func() []string { s := c.SortedKeys(); for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 { s[i], s[j] = s[j], s[i] }; return s }() {", 1,
     "$each over an object visits keys in descending order"),
    (P + "validate.go", "rt.combined = append(rt.combined, rep.Violations...)",
     "rt.combined = append(append([]Violation{}, rep.Violations...), rt.combined...)", 1,
     "the combined report lists later entries first"),
    (P + "shapes.go", "if _, declared := sd.Fields[k]; !declared && !ignored[k] {", "if _, declared := sd.Fields[k]; !declared {", 1,
     "ignoredProperties is ignored"),
    (P + "shapes.go", "\tcase *jaxson.Object, []any:\n\t\tjaxson.Fail(\"EXECUTION_ERROR\", \"TYPE_ERROR\", \"%s: a reference must be a scalar",
     "\tcase *jaxson.Object:\n\t\tjaxson.Fail(\"EXECUTION_ERROR\", \"TYPE_ERROR\", \"%s: a reference must be a scalar", 1,
     "an array is accepted as a reference value"),
    (P + "aggregate.go", 'if w, has := in["where"]; has {', 'if w, has := in["where"]; has && false {', 1, "aggregate ignores where"),
    (P + "aggregate.go", 'init[name] = nil', 'init[name] = zero()', 1, "aggregate min and max start at 0 instead of null"),
    (P + "aggregate.go", '[]any{kind, aggVar("x")},', 'aggVar("x"),', 1, "aggregate min and max take the first element without checking it is a number"),
    (P + "aggregate.go", '[]any{kind, aggVar("cur"), aggVar("x")}})', '[]any{"max", aggVar("cur"), aggVar("x")}})', 1, "aggregate min keeps the greater value"),
    (P + "aggregate.go", 'aggFail("measure %q: count takes true", name)\n\t\t\t}\n\t\t\tinit[name] = zero()',
     'aggFail("measure %q: count takes true", name)\n\t\t\t}\n\t\t\tinit[name] = big.NewRat(1, 1)', 1, "aggregate count starts at 1"),
    (P + "aggregate.go", 'if arg != true {', 'if false {', 1, "aggregate count accepts any argument"),
    (P + "aggregate.go", 'aggFail("unknown field %q", k)', '_ = k', 1, "aggregate accepts unknown members"),
    (P + "aggregate.go", '!ok || len(measures) == 0 {', '!ok {', 1, "aggregate accepts empty measures"),
    (P + "aggregate.go", 'return []any{\n\t\tmap[string]any{"op": "set", "path": jaxson.Clone(into),',
     'return []any{\n\t\tmap[string]any{"op": "set", "path": jaxson.Clone(into), "value": map[string]any{"$lit": init}},\n\t\tmap[string]any{"op": "set", "path": jaxson.Clone(into),', 1,
     "aggregate expansion has one extra instruction (cost differs from the hand-written fold)"),
    # The rulings of 2026-10-07 (TRACKER P1, R1, S9).
    (P + "indices.go", "\td := dep[1:]\n\tn := len(segs)\n", "\td := dep[1:]\n\tif len(segs) > len(d) {\n\t\treturn false\n\t}\n\tn := len(segs)\n", 1,
     "index reuse ignores a write below the source (the literal 'prefix of the source' reading)"),
    (P + "run.go", "if e.Target.root(reg) == \"input\" {", "if true {", 1,
     "every validate entry runs before the program"),
    ("pkg/jaxson/compile.go", "\t\tsort.Strings(names)\n\t\tfns := make([]func(*Machine) any, len(names))", "\t\tsort.Sort(sort.Reverse(sort.StringSlice(names)))\n\t\tfns := make([]func(*Machine) any, len(names))", 1,
     "template members are evaluated in reverse key order"),
    # Ruling G11, layer 1: what extends carries and how it narrows.
    (P + "parse_shapes.go", 'out.MinLen = maxInt(p.MinLen, c.MinLen)', 'out.MinLen = c.MinLen', 1,
     "extends drops the parent's minLen"),
    (P + "parse_shapes.go", 'out.MaxLen = minInt(p.MaxLen, c.MaxLen)', 'out.MaxLen = c.MaxLen', 1,
     "extends drops the parent's maxLen"),
    (P + "parse_shapes.go", 'out.MinItems = maxInt(p.MinItems, c.MinItems)', 'out.MinItems = c.MinItems', 1,
     "extends drops the parent's minItems"),
    (P + "parse_shapes.go", 'out.MaxItems = minInt(p.MaxItems, c.MaxItems)', 'out.MaxItems = c.MaxItems', 1,
     "extends drops the parent's maxItems"),
    (P + "parse_shapes.go", '(b != nil && *b >= *a)', '(b != nil && *b <= *a)', 1,
     'extends keeps the smaller of two lower bounds (minLen, minItems)'),
    # Severity and the gate: only a violation-severity finding aborts.
    (P + "shapes.go", "if c.stop && v.Severity == SeverityViolation {", "if c.stop {", 1,
     "a gate aborts on a warning or info finding of a shape"),
    (P + "validate.go", "if e.Mode == ModeGate && u.Severity == SeverityViolation {", "if e.Mode == ModeGate {", 1,
     "a gate aborts on a warning or info repeat of a unique entry"),
    (P + "parse_shapes.go", "both := make([]any, 0, len(p.Enum))", "var both []any", 1,
     "extends turns an empty enum intersection into no enum (accepts everything)"),
    (P + "parse_shapes.go", '(b != nil && *b <= *a)', '(b != nil && *b >= *a)', 1,
     'extends keeps the larger of two upper bounds (maxLen, maxItems)'),
    (P + "parse_shapes.go", 'out.WantInt = p.WantInt || c.WantInt', 'out.WantInt = c.WantInt', 1,
     "extends drops the parent's int"),
    (P + "parse_shapes.go", 'case c.MinRat == nil || p.MinRat.Cmp(c.MinRat) > 0:', 'case c.MinRat == nil || p.MinRat.Cmp(c.MinRat) < 0:', 1,
     'extends keeps the smaller of two min bounds'),
    (P + "parse_shapes.go", 'case c.MaxRat == nil || p.MaxRat.Cmp(c.MaxRat) < 0:', 'case c.MaxRat == nil || p.MaxRat.Cmp(c.MaxRat) > 0:', 1,
     'extends keeps the larger of two max bounds'),
    (P + "parse_shapes.go", '\t\tout.Enum = both\n', '\t\tout.Enum = c.Enum\n', 1,
     "extends lets the child's enum replace the parent's"),
    (P + "parse_shapes.go", '\tcase c.Enum == nil:\n\t\tout.Enum = p.Enum\n', '\tcase c.Enum == nil:\n', 1,
     "extends drops the parent's enum"),
    (P + "parse_shapes.go", '\tcase child.Kind != parent.Kind:\n', '\tcase false:\n', 1,
     'extends accepts a conflicting kind'),
    (P + "parse_shapes.go", '\tcase child.Kind == "":\n\t\treturn parent.Kind\n', '\tcase child.Kind == "":\n\t\treturn ""\n', 1,
     "extends drops the parent's kind when the child states none"),
    (P + "parse_shapes.go", '\tif !hasReference(parent) {\n\t\treturn\n\t}\n', '\tif true {\n\t\treturn\n\t}\n', 1,
     "extends drops the parent's reference target"),
    (P + "parse_shapes.go", '\tif child.RefIndex != parent.RefIndex || child.RefRelation != parent.RefRelation ||\n\t\t!reflect.DeepEqual(child.RefOf, parent.RefOf) || !reflect.DeepEqual(child.RefBy, parent.RefBy) {', '\tif false {', 1,
     'extends accepts a different reference target'),
    (P + "parse_shapes.go", '\tcase parent.Items != nil && !reflect.DeepEqual(parent.Items, child.Items):\n', '\tcase false:\n', 1,
     'extends accepts different items'),
    (P + "parse_shapes.go", '\tcase parent.Items != nil && !reflect.DeepEqual(parent.Items, child.Items):\n', '\tcase parent.Items != nil:\n', 1,
     'extends rejects items restated identically'),
    (P + "parse_shapes.go", '\tcase child.Items == nil:\n\t\tout.Items = parent.Items\n', '\tcase child.Items == nil:\n', 1,
     "extends drops the parent's items"),
    (P + "parse_shapes.go", '\tcase !reflect.DeepEqual(parent.Qualified, child.Qualified):\n', '\tcase false:\n', 1,
     'extends accepts a different qualified rule'),
    (P + "parse_shapes.go", '\tcase !reflect.DeepEqual(parent.Qualified, child.Qualified):\n', '\tcase true:\n', 1,
     'extends rejects a qualified rule restated identically'),
    (P + "parse_shapes.go", '\tcase child.Qualified == nil:\n\t\tout.Qualified = parent.Qualified\n', '\tcase child.Qualified == nil:\n', 1,
     "extends drops the parent's qualified rule"),
]

# Mutants the core leaves open: a survivor here is expected and is not a failure.
LEFT_OPEN = {
    "$indexed visits mixed-type or numeric keys in descending order":
        "the core names only the sorted-key convention and leaves the order of numeric and mixed-type keys to decision P4 (indices.go); "
        "it is pinned by TestIndexedOrderAndGrouping, which is outside this fixtures-only run",
    "an array is accepted as a reference value":
        "equivalent through the real IndexSet, whose Lookup (indices.go) enforces the same scalar rule; the shape-side check is pinned by "
        "TestReferenceNonScalarIsTypeErrorWhateverTheResolver, which uses a stub resolver and is outside this fixtures-only run",
}


def run_tests(work):
    return subprocess.run(["go", "test", "-count=1", "-run", "TestFixtures", "./pkg/shaxon/"],
                          cwd=work, capture_output=True, text=True)


def main():
    verbose = "-v" in sys.argv
    if shutil.which("go") is None:
        sys.exit("fixtures_mutants.py: go not found on PATH")
    survived, bad = [], 0
    with tempfile.TemporaryDirectory() as tmp:
        work = os.path.join(tmp, "module")
        shutil.copytree(MODULE, work, ignore=shutil.ignore_patterns(".git", ".ed-journal.json", "__pycache__", ".venv", ".jena"))
        base = run_tests(work)
        if base.returncode != 0:
            sys.exit("fixtures_mutants.py: the unmodified copy fails TestFixtures, so no mutant can be judged:\n"
                     + (base.stdout + base.stderr)[-800:])
        pristine = {}
        for i, (rel, old, new, n, what) in enumerate(M, 1):
            path = os.path.join(work, rel)
            if rel not in pristine:
                with open(path) as f:
                    pristine[rel] = f.read()
            text = pristine[rel]
            if text.count(old) != n:
                print(f"{i:2} {rel}: ANCHOR FOUND {text.count(old)} TIMES, EXPECTED {n}: {old[:60]!r}")
                bad += 1
                continue
            with open(path, "w") as f:
                f.write(text.replace(old, new))
            r = run_tests(work)
            with open(path, "w") as f:
                f.write(text)
            out = r.stdout + r.stderr
            if r.returncode != 0 and "[build failed]" in out:
                print(f"{i:2} {os.path.basename(rel):16} BUILD FAILED (the mutant does not compile) {what}")
                bad += 1
                continue
            killed = r.returncode != 0
            first = ""
            if killed and verbose:
                names = re.findall(r"--- FAIL: TestFixtures/(\S+)", out)
                first = "  first: " + (names[0] if names else "?")
            print(f"{i:2} {os.path.basename(rel):16} {'killed  ' if killed else 'SURVIVED'} {what}{first}", flush=True)
            if not killed:
                survived.append((i, what))
    unexpected = [s for s in survived if s[1] not in LEFT_OPEN]
    print(f"\ntried {len(M)}, killed {len(M) - len(survived) - bad}, survived {len(survived)}"
          f" ({len(survived) - len(unexpected)} expected, see LEFT_OPEN), bad {bad}")
    sys.exit(2 if bad else (1 if unexpected else 0))


if __name__ == "__main__":
    main()
