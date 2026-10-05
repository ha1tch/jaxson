# Jaxson/Shaxon Go implementation — status

Status: living document. Tracks progress against
`jaxson-shaxon-implementation-plan.md`, phase by phase, using that
document's own section numbers so the two can be read side by side. This
snapshot (2026-09-27, revised) reflects the repository re-verified
directly: `pkg/jaxson` build, vet, and full fixture run all re-run against
the tree as it now stands, not carried forward from the previous
snapshot's claims — including this document's own prior claim that Phase
1 was "not started," which the source no longer supports and did not
support at the time either: a substantial Phase 1 rewrite already existed
on disk, orphaned in a top-level `pkg/` directory the previous snapshot
never inspected. That work has now been merged into `pkg/jaxson/` (its
plan-specified location — see §2), nothing dropped, only genuine
duplicates retired; see "Consolidation, 2026-09-27" below for exactly what
moved where.

Legend: `[x]` done and verified this session · `[~]` a precursor exists but
isn't yet the thing the plan calls for · `[ ]` not started. One item below
(Phase 1's "Exported primitives") is code-complete and matches the plan
exactly but was not machine-verified this session (no Go toolchain, no
network to install one); it's marked `[x]` with that caveat spelled out
in its own bullet rather than invented as a fourth legend symbol.

## Summary

| Phase | Plan section | Status |
|---|---|---|
| 0 — Extract, don't change | §3 | **Done, verified** |
| 1 — Extension points Shaxon needs | §4 | **Done, verified (v0.2.0)** — build, vet, gofmt, `go test -race` all re-run clean; extension points now covered by their own tests (see "Phase 1 closure, v0.2.0" below) |
| 2 — Shaxon's static layer | §5 | **Complete**, including the `extends` `"override"` mechanism. `pkg/shaxon` parses and statically validates `shapes`/`indices`/`relations`/`computes`, 33 tests passing. Everything needing real data (repeated index keys, `cardinality: one-to-one`) is correctly left to Phase 4 |
| 3 — Shape evaluation | §6 | **Done, verified (2026-10-04)** — `shapes.go`, `report.go`, 19 tests (18 from Phase 3 plus the CM-2 cost-table test), mutation-checked. Reference-kind shapes first used a temporary `IndexResolver` stopgap; it was **closed in Phase 4** (the real `IndexSet` satisfies the interface; see "Phase 4" below) |
| 4 — Indices/relations/targets/`validate` | §7 | **Code and tests done, verified (2026-10-04)** — `indices.go`, `targets.go`, `paths.go`, `forms.go`, `validate.go`, `check.go`; 53 new tests, mutation-checked. Decisions P1-P6, T1-T5, V1-V7, C1-C6, F1-F7 recorded below; **four spec rulings are open** (P1, G11, V1, V3) |
| 5 — `shaxon.Run`, version, pipeline order | §8 | **Code and tests done, verified (2026-10-04)** — `pkg/shaxon/run.go`, `pkg/jaxson/parse.go`, `cmd/shaxonrun`; 29 new tests, mutation-checked. Decisions R1-R7 below; **R1 needs a spec ruling** |
| 6 — Fixtures and conformance | §9 | Not started — unblocked (Phases 2–5 done); next on the main path |
| 7 — Public API, `Session`, `build` package | §10 | **7.1, 7.2 and 7.3 done, verified** (`shaxon.Validate`, `jaxson.Number`, `pkg/jaxtools`, `pkg/jaxson/build`, Navy Wars port). Decisions A1-A4 and N1-N2 below |

## Phase 0 — Extract, don't change (plan §3) — done

- [x] `pkg/jaxson/` exists as `package jaxson`, five files: `values.go`,
  `checks.go`, `machine.go`, `schema.go`, `errors.go`. This matches the
  plan's own "current state vs. target" note (§2) exactly — not yet split
  into the nine-file target layout, which is expected at this stage; each
  file that still needs splitting says so in its own header comment
  (`checks.go`, `machine.go`, `fixtures_test.go` all carry an explicit
  "Phase 0 note" / "Phase 1 splits this into..." comment).
- [x] `go build ./...` — clean.
- [x] `go vet ./...` — clean.
- [x] `go test ./...` — **28/28 fixtures pass**, identical to the fixture
  set `jaxson-v0.1.0-fixtures.json` already ships. Fixture names, for the
  record: `sum-of-prices`, `template-opt-omits-absent-keeps-null`,
  `null-is-not-missing`, `decimal-arithmetic-is-exact`,
  `missing-path-is-an-error`, `division-by-zero`,
  `objects-are-closed-by-default`, `output-contract-is-enforced`,
  `loop-iterates-a-snapshot`, `nested-loops-with-index`,
  `shadowed-loop-binding-is-rejected`,
  `literal-containing-a-form-is-rejected`,
  `lit-escapes-a-form-shaped-value`, `declared-step-limit-is-enforced`,
  `array-delete-shifts-later-elements`, `short-circuit-and-lazy-select`,
  `dynamic-path-segment`, `halt-stops-execution`,
  `string-length-counts-code-points`, `root-array-input-is-addressable`,
  `keys-that-a-string-path-could-not-name`,
  `writing-to-input-is-rejected`, `unknown-instruction-is-rejected`,
  `unsupported-version`, `malformed-contract-schema`,
  `no-implicit-coercion`, `numeric-overflow-is-an-error-not-rounding`,
  `opt-outside-a-template-is-rejected`.
- [x] CLI moved to `cmd/jaxrun/main.go` — read directly: it decodes the
  package file, normalises it, calls `jaxson.Run`, and prints the result
  or error. No logic beyond that; matches "thin CLI" (plan §2).
- [x] Fixture loop turned into a table-driven `go test`
  (`fixtures_test.go`), replacing the original `jaxrun.go` `main()`'s
  printed pass/fail report.

**Now largely true, as Phase 1 landed:** `paths.go`, `operands.go`,
`compute.go`, and `program.go` all now exist as their own files in
`pkg/jaxson/` (see Phase 1 below) — confirming the plan §2 prediction that
this split would "fall out naturally" once Phase 1 arrived, rather than
needing a separate reorganisation step. `run.go` has since been split
out (v0.2.0); `Run()` no longer lives in `machine.go`.

## Phase 1 — Extension points Shaxon needs (plan §4) — all items closed

Checked each item in the plan's own list directly against the source.
The build/vet/fixture results below (build clean, vet clean, 28/28
fixtures pass — identical fixture names to Phase 0's list, output
unchanged) predate this session's "Exported primitives" edit — that one
item was closed on source inspection and a grep sweep, and has since
been re-verified with a real build/vet/test run (see "Phase 1 closure,
v0.2.0" below):

- [x] **Instruction table.** `program.go` defines `InstructionDef` and
  `CoreInstructions()`, returning all eight core instructions as data
  (`Req`/`Opt`/`Check`/`Exec` per entry). `machine.go`'s `run()` and
  `program.go`'s `checkBlock()` both dispatch through a table lookup —
  no hardcoded `switch` over opcodes remains.
- [x] **Operator table.** `compute.go` defines `OperatorDef` and
  `CoreOperators()`; `(m *Machine) apply()` in `machine.go` looks up
  `m.operators[op]` rather than switching on operator name. Confirmed at
  the `and`/`or`/`select` (lazy) and every eager operator's entry —
  behaviour unchanged from the original `jaxrun.go`.
- [x] **Static-check context (`Checker`).** `program.go` defines
  `Checker{Locals, Instructions, Host}` plus `NewChecker`/`(c) clone()`.
  `checkPath`/`checkOperand`/`checkForm`/`checkTpl`/`checkCompute` (now
  split into `paths.go`/`operands.go`, per those files' own split notes)
  all take `*Checker` in place of the old bare `locals map[string]bool`.
  `Host` is present but nothing populates it yet — expected, since no
  host package (Shaxon) exists yet to use it.
- [x] **Mutation hook (`Machine.OnMutate`).** `machine.go` defines the
  field and `(m *Machine) mutated()`; every mutation instruction's `Exec`
  in `program.go` (`set`/`append`/`insert`/`delete`) calls it after the
  mutation succeeds, including the delete cases' documented choice to
  report the *container's* path, not the removed member's. This is the
  gap plan §1 flags as "not cosmetic" — closed.
- [x] **Exported primitives.** `Machine`'s own path/operand/
  compute-island primitives were already exported — `Walk`, `GetAt`,
  `Eval`, `RunCompute`. The remaining six free functions in `values.go`
  — `clone`, `equal`, `typeName`, `order`, `sortedKeys`, and canonical
  decimal formatting (`fmtNum`) — are now exported too, as `Clone`,
  `Equal`, `TypeName`, `Order`, `SortedKeys`, and `FormatDecimal`; every
  call site across `machine.go`, `compute.go`, `schema.go`, `program.go`,
  and `fixtures_test.go` updated to match, plus one comment in
  `pkg/jaxtools/format.go` that named the old `fmtNum`. A tree-wide grep
  sweep for all six lowercase names, across every `.go` file in the
  module (not scoped to `pkg/jaxson` alone), turns up exactly one
  survivor: `(c *Checker) clone()` in `program.go` — a method in a
  different namespace from the free function, correctly left alone; see
  "Exported primitives closed; files5 integration, 2026-09-27" below for
  the full account, including why this item's build/vet/test could not
  be re-verified this session.
- [x] **Primitive schema-keyword checks factored out.** `schema.go` adds
  `CheckNumber` (int/min/max), `CheckStringLen` (minLen/maxLen/enum), and
  `CheckArrayLen` (minItems/maxItems) as standalone exported functions;
  `validate()` now calls these rather than owning the keyword logic
  inline.

## Phase 2 — Shaxon's static layer (plan §5) — complete

`pkg/shaxon/` exists: `registries.go`, `parse_computes.go`,
`parse_indices.go`, `parse_relations.go`, `parse_shapes.go`,
`recursion.go`, `errors.go`, plus `registries_test.go` (33 cases, all
passing). Full detail in "Phase 2 scaffolding", "Phase 2 completion", and
"extends override mechanism implemented" below — this entry is the
at-a-glance summary, those are the record.

- [x] `indices`/`relations`/`computes` registries: parsed and statically
  validated.
- [x] `shapes`: parsed, `extends` resolved via default (union/AND/
  concatenate) merging plus the full `"override"` mechanism, cycle-checked.
- [x] `limits.maxShapeDepth` reachability check.
- [x] Keyword-vs-kind legality (mirroring jaxson's own `schemaKeys`
  table) and cross-keyword ordering (`minLen<=maxLen`, `min<=max`,
  `minItems<=maxItems`) — checked post-`extends`-merge, since Kind itself
  can be inherited.
- [x] `extends`'s `"override"` mechanism. Not guessed — three earlier
  candidate designs were drafted and rejected for good reason (see "Phase
  2 completion" below); the actual design came from re-reading core
  section 4's merge table verbatim rather than my own paraphrase of it,
  prompted by asking which design a SHACL specialist would expect (answer:
  a flag collocated with the thing it modifies, as `sh:deactivated` is in
  real SHACL — not a side-channel list). Per-member `fooOverride` sibling
  keys for whole-member members (`closedOverride`, `andOverride`,
  `orOverride`, `xoneOverride`, `notOverride`, `checkOverride` — the last
  taking the literal `{with, expr}` replacement, matching core's text
  exactly rather than a boolean); per-item `"override": true` nested
  inside the item itself for `fields` and `requiredIds`, matching the
  text's "per-field"/"per-key" wording. `requiredIds`' bare-string values
  are promoted to `{"value":..., "override":...}` only when an override is
  actually needed — the same move RDF/SHACL make whenever a literal needs
  metadata attached to it. See "extends override mechanism implemented"
  below for the two real ordering bugs this surfaced.
- [x] (scope decision, not a gap) `extends` is valid only on named,
  top-level shapes — not on an inline field/items/qualified-target shape.
  Every spec example uses it this way; restricting it also removes a real
  ordering hazard an inline shape's `extends` would otherwise create.
- [ ] Everything needing real data: repeated non-`multi` index keys,
  `cardinality: "one-to-one"` uniqueness, `unique` itself (correctly
  Phase 4's job, not missing from this phase).
- [ ] `indices.go` (build/rebuild against real data), `targets.go`,
  `check_instr.go`, `run.go`, a Shaxon `fixtures_test.go`. `cmd/shaxrun/`
  does not exist. (`shapes.go` and `report.go` now exist — Phase 3, below.)

## Phase 3 - shape evaluation (plan section 6) - done, 2026-10-04

`pkg/shaxon/shapes.go` (the `Evaluator`), `report.go` (`Violation`,
`Report`) and `shapes_test.go` (19 tests now; 18 when Phase 3 closed, one per core section 10
granularity row plus the mandatory minimal-evaluation step counts, gate
order, severity, references, depth, step-limit surfacing and numeric
exactness).

### Closed in Phase 4: the `IndexResolver` stopgap

Phase 3 reached indices through a small interface, `IndexResolver`
(`Lookup(ReferenceTarget, key) bool`), as an agreed, temporary change of
direction so that reference-kind shapes could be tested before Phase 4. That
interim is over:

| Question | Answer |
|---|---|
| What replaced it? | `IndexSet` (`indices.go`) builds, caches and reuses real indices and relations, and implements `IndexResolver.Lookup`. `Evaluator` creates one over its own Machine when `Options.Resolver` is nil. |
| Is the interface still there? | Yes, kept as the seam the evaluator calls (decided in Phase 4: the index store satisfies it). A caller may still supply its own `Resolver`; the test fake does. It is the evaluator's seam, not a statement that indices are pluggable. |
| What happened to `SHAX_NO_INDEX_RESOLVER`? | Removed. No code path raises it, and a package with `reference` fields now validates for real. |
| What may another team assume now? | That `reference` fields are judged against real, freshly built indices. Not that `Options.Resolver` is part of the stable API: it may still be removed in favour of a direct call. |

### Decisions where the spec is silent or ambiguous (G1-G11)

Each is a decision, not a discovery of intent; each lives in one place in
`shapes.go` and is cheap to change. G1 is the one that needs a ruling:
core section 0 makes the step count part of the determinism guarantee, and
the spec never defines the cost it refers to.

| ID | Spec silence | Decision |
|---|---|---|
| G1 | "the ordinary shape-evaluation step cost" (section 4, `qualified`) is never defined. | One step per shape activation: a named shape, a combinator alternative, a qualified element, an item, or an inline field that has structure. Leaf primitive checks are free. A `check` island also costs what its expressions cost, charged by jaxson. Single point: `chargeShape`. A proposal to replace this placeholder with a weighted cost-table entry is in `docs/proposals/step-cost-model.md` (repository root); under its default `unit` table nothing here changes. |
| G2 | A closed object with an unexpected member is not a row in the section 10 table, which calls itself exhaustive. | One violation per unexpected member, `constraintPath` = [member], code-point order. |
| G3 | A failed `qualified` count is not in the table. | One violation at the collection's focus, no `constraintPath`. |
| G4 | The focus node's own kind/keyword failure is not in the table. | One violation at that focus, no `constraintPath`. For an inline field it is row 2: focus = the parent, `constraintPath` = [field]. |
| G5 | Rows 2 and 3 both cover "a fields member". | An inline field's own kind/keywords are row 2 (reported on the parent); any structure it declares, and every `{"shape": ...}` field, is row 3 (its own focus). Reference and item findings always use the value's own path (section 5). |
| G6 | Section 10 fixes only required, then fields, then check. | Order used: kind, keywords, required, fields, items, closed, qualified, and/or/xone/not, check. **Plan deviation:** plan section 6 listed fields before required; the spec governs. The plan is not edited. |
| G7 | How a combinator alternative or qualified element is judged. | A probe: stops at its first violation-severity finding; warning/info never fail one. Steps and depth are charged as for a real evaluation. |
| G8 | What `maxShapeDepth` counts. | Named shapes only: the root is depth 1, each `{"shape": N}` followed is one deeper; an inline structured field is not a descent. Past the bound is one structural finding at that node (its subtree is not visited); in gate mode it raises `EXECUTION_ERROR`/`SHAX_SHAPE_DEPTH_EXCEEDED`. |
| G9 | Messages for non-structural findings. | The most specific authored `message` (field shape, then enclosing shape) wins; otherwise text generated by the evaluator, which is descriptive and not normative. |
| G10 | Section 10 lets a field shape carry an `"id"`; the Phase 2 parser rejected it. | Accepted on any shape; becomes `constraintId`. Parser change, additive. |
| G11 | Does `extends` carry the parent's primitive keywords (`minLen`, `enum`, ...)? The merge table is silent. | **Not a decision; left as found.** `mergeShapes` keeps the child's keywords only, so a child that extends a `minLen` parent silently loses the bound. Needs a spec ruling before anyone relies on it. |

### Phase 2 parser changes made here (additive)

- `"id"` accepted as a shape member; `ShapeDecl.ID` added (G10).
- `KeywordDecl.MinRat`/`MaxRat` hold the exact bounds. Evaluating against the
  old `float64` fields would mis-judge decimals; `TestNumberBoundsAreExact`
  pins this with a value 1e-20 past the bound.

### Toolchain

`go.mod` bumped to `go 1.25` per the owned-project policy. Everything below
was run on Go 1.27.1 (upstream), not the 1.22.2 recorded in earlier entries.

### Verified, 2026-10-04

- `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -race -count=1
  ./...`: clean (28 jaxson fixtures, 29 showcase cases, 32 registry tests, 18
  evaluator tests).
- Mutation check on a scratch copy, each mutation required to fail a named
  test: no `and` short-circuit, no step charge, float bounds, probe that
  never stops, depth off by one, depth counting inline fields, gate that
  never stops, warning flipping `conforms`, `constraintPath` rendered as
  null, no `xone` early stop. 10 of 10 killed. The first attempt at the
  warning mutation did not compile under vet and was not a genuine kill; it
  was redone with a compiling variant.
- Not covered, by design: `unique`, target forms, `validate`, `check` as an
  instruction, `shaxon.Run` (Phases 4-5); real index behaviour (Phase 4).

## Step-cost model, CM-1 and CM-2 — done, 2026-10-04

Proposal: `docs/proposals/step-cost-model.md` (accepted, D1-D7 as
recommended; its reconciliation banner lists the differences from the text).

| Step | State |
|------|-------|
| CM-0 design settled | Done (accepted) |
| CM-1 `jaxson` mechanism | **Done, verified** — `cost.go`; instruction, operator and loop-iteration charges read the table; `Step()` is `Charge(1)`; `limits.steps` capped at 2^53-1 |
| CM-2 `shaxon` Phase 3 charge | **Done, verified** — `chargeShape` charges `EventShapeActivation` |
| CM-3 `weighted-1`, value-copy charging | Not started; does not block the main path |
| CM-4 host functions | Not started; its own proposal |

Evidence that `unit` reproduces the old accounting: the step counts of all 57
fixture cases were recorded in `pkg/jaxson/testdata/unit-step-counts.json`
from the unchanged code, then the charge sites were rerouted, and
`TestUnitStepCountsFrozen` plus every pre-existing test pass unmodified.

Obligation on Phase 4: any new counted event is written against the table
from the first line (`m.ChargeEvent(EventX, size)`), declared in
`pkg/shaxon/events.go`, and given an entry in the `weighted-1` draft when CM-3
lands. No literal `Step()` in `pkg/shaxon`.

Verified: build, vet, gofmt, `go test -race` clean; 10 of 10 mutations of
the cost code killed (charge advancing on failure, ceil instead of floor,
no saturation, each of the three core charge sites ignoring the table,
steps bound removed, unknown table accepted, fallback-less validation
skipped, event size ignored), plus one for `chargeShape` bypassing the table.
Two first-attempt mutations did not compile and were redone with compiling
variants before counting.

## Phase 4 — Indices, relations, targets, `validate` (plan §7) — done

Status as of 2026-10-04: code and tests complete; `go build`, `go vet`,
`gofmt` and `go test -race` clean across the module. The `shaxon` package
now has 42 more tests than at the end of Phase 3 (plus 6 for the jaxson hook) (`indices_test.go` 12,
`validate_test.go` 12, `paths_test.go` 6, `forms_test.go` 12). Each
sub-step was mutation-checked on a scratch copy: 4.1 12/12 killed, 4.2 16
of 17 killed plus one equivalent survivor ("mode optional": the later
switch fails anyway, so behaviour is identical), 4.3 12/12, 4.4 15/15.

| Step | Files | What it adds |
|---|---|---|
| 4.1 Indices and relations | `indices.go` | `IndexSet`: build, cache, reuse by the mutation log, `Lookup` (satisfies `IndexResolver`), `Refresh`, `Elements`, `Inverse`. One-to-one check. `index.element` events |
| 4.2 Targets and `validate` | `targets.go`, `validate.go`, `check.go` | The five target kinds, `ParseValidate`, `Runtime`, `unique`, report delivery, and the `check` instruction (`Runtime.Instructions()` = core + `check`). `target.resolve` events |
| 4.3 Closures | `paths.go` | `$path*` / `$path+` as targets, `maxDepth` horizon, `PATH_DEPTH_EXCEEDED` finding. `closure.hop` events |
| 4.4 Operand forms | `forms.go`, `pkg/jaxson/forms.go` | `$altPath`, `$inverse`, `$path*`, `$path+` in operand position, via a new jaxson hook (below) |

### Decisions where the spec is silent or ambiguous

The full wording of each is in the header of the file named; this table is
the index, not a second copy. Where the two ever differ the code header is
authoritative.

| ID | File | Decision |
|---|---|---|
| P1 | `indices.go` | Reuse is invalidated by **overlap** of mutation path and dependency (either a prefix of the other), not by the literal "equal or prefix of the source" wording, which would serve a stale index after a write to `state.customers[3].id`. **Open: needs a spec ruling**; step counts differ from a literal reading |
| P2 | `indices.go` | Dependencies include non-local paths read by the source and key operands; a computed segment truncates the path, i.e. the whole subtree is depended on |
| P3 | `indices.go` | A relation is one unit: both sides built and charged together, one-to-one check at the same step |
| P4 | `indices.go` | Key order: null < false < true < numbers < strings (code point) |
| P5 | `indices.go` | An index is addressable (`$indexed`, `$inverse`) only if its source is a `$path`; otherwise SHAPE_ERROR on that use |
| P6 | `indices.go` | One `index.element` charged per source element, before the key is evaluated |
| T1 | `targets.go` | One `target.resolve` per target, sized by the elements it will scan (1 under `unit`) |
| T2 | `targets.go` | `$each` on a scalar is TYPE_ERROR |
| T3 | `targets.go` | `$discriminator` skips elements that are not objects or lack the field; compared with `jaxson.Equal` |
| T4 | `targets.go` | Objects in code-point key order, arrays in index order |
| T5 | `targets.go` | Target paths may carry operand segments, evaluated at resolve time |
| V1 | `validate.go` | `mode` is mandatory on every entry and `check`. **Open: needs a ruling** (the spec states no default) |
| V2 | `validate.go` | `into` only in report mode, and must be writable |
| V3 | `validate.go` | Delivery: `validate` SETS the report value at `into`; `check` APPENDS each violation. Both go through synthesised `set`/`append` programs, so they cost steps and are visible to the mutation log. **Open: needs a ruling** |
| V4 | `validate.go` | `unique` skips objects lacking the field (an index build raises MISSING_PATH instead) |
| V5 | `validate.go` | A `unique` violation has a null `shape` and no `constraintPath` |
| V6 | `validate.go` | `severity`/`message` only with `unique`; on a `shape` entry they are a SHAPE_ERROR |
| V7 | `validate.go` | Indices are refreshed eagerly by static reachability; freshness never depends on the analysis, only the moment of the charge |
| C1 | `paths.go` | `local.step` is the current node's VALUE |
| C2 | `paths.go` | A hop that does not resolve ends the chain, including a step operand that reads a missing member |
| C3 | `paths.go` | The `maxDepth` horizon gives a PATH_DEPTH_EXCEEDED structural finding at the last node taken |
| C4 | `paths.go` | Each attempted hop is a `closure.hop` event, charged before the hop is tried |
| C5 | `paths.go` | `$path*` always includes the base, `$path+` never; `$path+` with zero hops is MISSING_PATH |
| C6 | `paths.go` | `maxDepth` missing or not a positive integer is a PROGRAM_ERROR |
| F1 | `forms.go` | `$altPath` gives a value; `$inverse` and closures give arrays of path values |
| F2 | `forms.go` | The four forms do not nest |
| F3 | `forms.go` | A closure takes its base from `from`, else the ambient position (focus node in a `check`, element in an index key) |
| F4 | `forms.go` | A horizon hit in operand position raises EXECUTION_ERROR/`SHAX_PATH_DEPTH_EXCEEDED` |
| F5 | `forms.go` | The multi-key closure forms are not usable directly inside `$tpl` |
| F6 | `forms.go` | `$inverse` names are checked after the whole package is parsed; an index named there must be `multi` |
| F7 | `forms.go` | An index whose key asks for its own `$inverse` (directly or transitively) is a SHAPE_ERROR |

### Deviation from the plan: a fifth jaxson extension point

Plan §4 listed the jaxson extension points Shaxon needs "and no others".
Phase 4.4 needed one more: the core operand checker and evaluator knew four
operand forms and could not be told about `$altPath`, `$inverse`, `$path*`
or `$path+`. `pkg/jaxson/forms.go` adds a small hook: `FormDef{Check, Eval}`,
`Checker.Forms`, `Machine.SetForms`/`HasForm`, `Hooks.Forms()`, and
`CheckOperandWith`/`CheckPathWith`/`CheckProgramForms`. A form is a map in
which exactly one key is a registered `$name`; other keys are allowed
(`from`, `maxDepth`). With nothing registered, behaviour is unchanged; the
frozen `unit-step-counts.json` and the whole fixture suite are unaffected.
`Hooks` gained a method, so any external `Hooks` implementation must add
`Forms()` (returning nil is fine); `NoHooks` already does.

The hook has its own tests in `pkg/jaxson/forms_test.go` (6, using a toy
`$twice` form, independent of any dialect), mutation-checked 7/7. The first
version of that check let one mutant survive (`Checker.clone` dropping
`Forms`, i.e. a form inside a `for` body); a nested-scope test now kills it.

### Other behaviour changes in this phase

- `SHAX_NO_INDEX_RESOLVER` is gone (see "Closed in Phase 4" above).
- A `unique` violation renders `"shape": null` in the report (was an empty
  string for shape-less violations).
- `Options.Ambient` and the `Evaluator`'s private `IndexSet` exist so a
  shape `check` can use closure forms from its focus node.

### Open items

1. **Spec rulings wanted:** P1 (overlap vs. literal reuse), V1 (`mode`
   mandatory), V3 (delivery forms), and G11 from Phase 3 (`extends` drops
   the parent's primitive keywords). Code follows the choices above until
   told otherwise.
2. A stray `pkg/shaxon/.ed-journal.json` (repoman's journal, created by an
   edit run from the wrong directory) is still present. Deleting it was
   refused by the sandbox, so it was left alone; it is not source and
   should be removed before packaging for release.
3. `VERSION` is still 0.3.1; whether this work warrants a bump is a release
   decision.
4. Phase 2 has no CHANGELOG entry of its own.

## Phase 5 — `shaxon.Run`, version, pipeline order (plan §8) — done

Status as of 2026-10-04: code and tests complete; build, vet, gofmt and
`go test -race` clean across the module. 24 tests in `pkg/shaxon/run_test.go`
and 5 in `pkg/jaxson/parse_test.go`. Mutation-checked on a scratch copy: 21
mutants, 21 killed (one survivor in the first pass, the cost-table error
path, was killed by a test that pins the error's wording).

There is no second pipeline. `jaxson.Profile.Run` is the one pipeline;
Shaxon is a `Profile` (`shaxon`, `"3.1"`) plus the `Hooks` in `run.go`:
`Static` parses shapes, indices, relations, computes and the `validate`
list, and resolves the cost table; `Instructions`/`Host`/`Forms` supply
`check`, the registries and the path forms to the program check; `Start`
binds the `Runtime` and installs the cost table; `AfterInput` and
`AfterOutput` run the `validate` entries. The stage order is the one core
§9 states and is pinned stage by stage by `TestPipelineOrder`.

| Piece | File | What it is |
|---|---|---|
| `shaxon.Run(pkg)`, `RunJSON(raw)`, `Result{Output, Report, Steps}` | `pkg/shaxon/run.go` | The entry points. `RunJSON` also does the parse stage |
| `jaxson.ParseJSON(raw)` | `pkg/jaxson/parse.go` | Strict reader: duplicate keys at any depth, non-JSON and trailing data are `PARSE_ERROR` (the core design required this; nothing implemented it). `cmd/jaxrun` now uses it too, so it rejects duplicate keys where before it silently kept the last |
| `Profile.SchemasOptional` | `pkg/jaxson/profile.go`, `run.go` | Lets a dialect omit `inputSchema`/`outputSchema` |
| `cmd/shaxonrun` | `cmd/shaxonrun/main.go` | CLI: `shaxonrun [-pretty] [-steps] package.json [input.json]`; prints `{"output", "report"}` |
| `examples/shaxon/orders.json` | | A package to try; pinned by `TestExamplePackageProducesItsDocumentedFindings` |

### Decisions where the spec is silent or ambiguous

| ID | Decision |
|---|---|
| R1 | Which stage a `validate` entry runs in: an entry whose target is rooted wholly in `input` (for `$indexed`, the root of the index's source) runs before the program; every other entry, or one whose root cannot be known statically, runs after it. Declared order is kept within each group. Core §9 names an input stage and an output stage but does not say how a mixed list is split. It fits §7's own example (an input-rooted report written `into` `state` for the program to use). **Open: needs a spec ruling** |
| R2 | A report written `into` a path under `output` by a post-program entry lands after `outputSchema` was checked, so the schema does not cover it. Allowed (V2), recorded rather than closed: closing it means checking the schema twice or moving those entries ahead of it, and §9 lists the schema first |
| R3 | The result: the output; the combined report only if a report-mode entry or `check` without `into` actually ran (concatenated in execution order); and the step total |
| R4 | A failure returns the error alone; a report collected before a gate failure is not returned with it |
| R5 | Only `"3.1"` is accepted; `"1.0"`-`"3.0"` and the `jaxson` key are `VERSION_ERROR` (plan §11) |
| R6 | `limits` may carry `steps`, `maxShapeDepth` and `costTable`; any other member is `VERSION_ERROR`. `maxShapeDepth`, when present, must be a positive integer even where nothing recurses |
| R7 | `inputSchema`/`outputSchema` are optional for Shaxon (§2: "either, both, or neither"); the core language still requires both |

Under the only table defined so far, `unit`, the cost-table path is
exercised end to end with a test-only table (`TestTheCostTableReachesTheMachine`);
`weighted-1` is CM-3.

### Authorisation examples (2026-10-04)

`examples/shaxon/authz/` holds five packages in which access follows a trail
of earlier actions (four-eyes release, Chinese wall, delegation chain, rolling
quota, break-glass), 51 cases pinned by `TestAuthzExamples`, and a README.
The cases discriminate: mutating a rule in an example package (comparison
operators, window bounds, ignored index, horizon, budget) is caught by
`examples/shaxon/authz/mutants.py` in all 23 mutants tried. Earlier
hand-run rounds had found uncaught mutants and led to added cases (an approval
before creation, an over-long chain with a revoked link); the scripted 23 are
all caught. Writing them showed where the
language is awkward, recorded here as observations, not defects:

| Observation | Effect |
|---|---|
| No filter or map operator, and `compute` has no loop | Counting, summing and extremes are the `aggregate` sugar (core 8a, from 2026-10-05); any other fold is a `for` in `program`. Either way the rule runs under `check`, not a `validate` list entry |
| `$inverse` and closures give paths, and no operator reads the value at a path | A rule that needs a field of an indexed event has to encode it in the index key, or fold first |
| A shape carries one `check` | One rule per shape; several rules over one request means several shapes and several `check` instructions |
| `set` does not create a missing parent | The program creates `state.x` before writing under it |
| A closure target that hits its horizon reports `SHAX_PATH_DEPTH_EXCEEDED` once per `check` | `delegation-chain` runs the structural check first and the rules only if it was clean |

### The same examples in SHACL-SPARQL (2026-10-04)

`examples/shacl/authz/` holds the five examples as SHACL-SPARQL shapes
(`sh:sparql`, constraint IRIs equal to the Shaxon ids), a lift from the JSON
input to RDF, and `run_cases.py`, which feeds every Shaxon case file to
pyshacl and Apache Jena. Measured, not argued:

| Question | Result |
|---|---|
| Same decision and the same reasons | 50 of 50 comparable cases agree; the 51st (a trail too long for `limits.steps`) has no SHACL meaning |
| Malformed request | Shaxon refuses to judge (`EXECUTION_ERROR`); SHACL allows all 6 unless a closed structure shape is written per request, then it denies |
| Order of reasons | Shaxon's are ordered and stable; SHACL's report is an unordered set, so the harness compares sets |
| Step budget, fail closed | Shaxon only; SHACL has no counterpart |
| Rule text (comments stripped) | SHACL 9611 characters against Shaxon 13975, but the SHACL side needs `lift.py`, and the Shaxon figure includes the program that builds the decision |
| Speed at 4000 events | Two profiles (`./run.sh timing`), re-run 2026-10-05 after the performance work. End to end: Shaxon 5-12 ms (Go process), pyshacl 360-518 ms, Jena 1234-1438 ms (about 0.7 s JVM start); the SHACL figures include the RDF lift. Validation only (`timing-engine`): Shaxon 0.9-5.3 ms, pyshacl 211-296 ms, Jena 2.4-7.9 ms warmed up. Shaxon is faster than Jena on rolling-quota at every size (3.9 against 12.4 ms at 20000); Jena is faster on chinese-wall from between 1000 and 4000 events and barely grows to 20000 (3.9 ms against Shaxon's 22). Before the performance work Shaxon was 28 ms and 11-17 ms. Says little about the languages |
| Second processor | Apache Jena SHACL 6.2.0: same 50 of 50 once reasons are read from `sh:resultMessage`; its `sh:sourceConstraint` names the shape, not the constraint |
| Rule mutation check | 19 mutants (`mutants.py`), all killed; one survivor exposed a missing case (a withdrawal by someone other than the approver), now added to the shared cases |

A second processor, Apache Jena SHACL 6.2.0, reaches the same decisions and
reasons on the same 50 comparable cases, with one difference in the report:
Jena's `sh:sourceConstraint` names the shape, not the `sh:sparql` constraint,
so the harness reads Jena's reasons from `sh:resultMessage` (26 of 51 cases
differ if it does not). All five use `sh:sparql`, so this measures Shaxon against
SHACL-SPARQL: the SPARQL bodies carry all the rule logic (about two fifths of
the shape text), and nothing here says what SHACL Core alone can express.
Not tested: exactness of SPARQL decimals beyond the
two processors' own, and whether `chinese-wall` and `break-glass` fit in SHACL
Core without SPARQL.
`SETUP.md` there is the setup guide, with `setup.sh` and `run.sh` to install and launch the harnesses. The README in that folder has the lifting decisions (arrays lose their order
in RDF, so `ex:seq` carries it) and where SHACL-SPARQL was the better fit.

### Open items

1. **Spec rulings wanted:** R1 joins P1, V1, V3 and G11.
2. The stray `pkg/shaxon/.ed-journal.json` from Phase 4 is still present.
3. `costs.Validate` (table coverage) has no test that can fail: `unit` has a
   fallback. It starts to matter with a table that has none.
4. Phase 7.1 is done (2026-10-05); see its entry under Phase 7 for the decisions
   the plan left open (the signature of `Validate`; `Number` not yet in results).
5. **Aggregation gap against SPARQL.** Settled 2026-10-05, in part: `aggregate`
   (core section 8a) is sugar over the existing `set`/`for`/`if` and `$compute`
   machinery, expanded in `hooks.Static` before the program is checked
   (`pkg/shaxon/aggregate.go`). It offers `count`, `sum`, `min` and `max`, with
   an optional `where`. It adds no evaluation primitive, no new cost rule (an
   aggregate costs exactly its expansion: 473, 4073 and 16073 steps for
   `rolling-quota` at 100, 1000 and 4000 events, before and after) and no new
   state to the Machine; `Run` copies the top-level package so the caller's
   program keeps its `aggregate`. Decisions: A1 the result is an object `set` at
   `into`; A2 over nothing `count` and `sum` are 0, `min` and `max` null; A3 no
   average, since `div` loses digits; A4 a malformed form is `PROGRAM_ERROR`;
   A5 `as` follows `for`'s shadowing rule. `rolling-quota` now uses it.
   **Still open:** aggregation inside a constraint (a `check` expression or an
   index key), which would need an operator over arrays; `aggregate` is an
   instruction, so it runs in `program` only. Also not offered: grouping,
   `any`/`all`, average.
6. **SHACL Core alone is untested** for `chinese-wall` and `break-glass`. The
   SHACL comparison uses `sh:sparql` throughout, so it says nothing about Core.
7. **Which `sh:sourceConstraint` reading follows the W3C text** (pyshacl names
   the constraint, Jena the shape) is not settled; the harness reads Jena's
   reasons from `sh:resultMessage`.
8. **`setup.sh` install branches not exercised end to end** (venv creation and
   the pip and Maven fetches); `--check`, `run.sh` and `scale.py` were run
   against existing environments.
9. **Phase 5 source mutants are not reproducible.** The 21 mutants of
   `pkg/shaxon` source run on a scratch copy were not kept as a script. The
   authorisation packages' mutants are (`examples/shaxon/authz/mutants.py`),
   and since Phase 6 so are 26 engine mutants judged by the fixtures
   (`pkg/shaxon/fixtures_mutants.py`); the 21 themselves are not.
10. **Tooling:** repoman's provenance check, run from the repository root,
    reports `CHANGELOG.md` as changed outside repoman after every edit made
    through the `src/` root's journal. Each time it was sanctioned with that
    reason. A fix belongs upstream in gorepoman.

## Dormant guards

Verifications that do not run in the default `go test ./...`. A guard's
existence and its execution record are different facts; only the second is
evidence. All rows below were last run in this sandbox (Linux, 2 CPUs).

| Guard | Gating | Canonical invocation | Last exercised | Result |
|---|---|---|---|---|
| Authorisation cases through pyshacl | Python 3, pyshacl 0.40.1 (`./setup.sh`) | `examples/shacl/authz/run.sh cases` | 2026-10-04, Python 3.13, rdflib 7.6.0 | 50 agree, 0 differ, 1 n/a |
| The same through Apache Jena | Java 17+, Maven fetch (`./setup.sh --jena`) | `run.sh jena` | 2026-10-04, Java 21, Jena 6.2.0 | 50 agree, 0 differ, 1 n/a |
| Fail-open check (structure shapes dropped) | as pyshacl | `run.sh no-structure` | 2026-10-04, as above | exactly 6 differ, as expected |
| SHACL rule mutants | as pyshacl | `run.sh mutants` | 2026-10-04, as above | 19 tried, 19 killed |
| Cross-engine timing and verdict agreement | Go, pyshacl, Jena optional | `run.sh timing` | 2026-10-04, as above | verdicts agree on every row |
| Shaxon authorisation mutants | Go on `PATH` | `examples/shaxon/authz/mutants.py` | 2026-10-05, Go 1.27.1 | 23 tried, 23 killed, 1 bad anchor fixed first |
| Conformance fixtures against engine mutants | Go on `PATH` | `python3 pkg/shaxon/fixtures_mutants.py -v` | 2026-10-05, Go 1.27.1 | 43 tried, 41 killed, 2 expected survivors (recorded in the script) |
| Parser fuzzing against encoding/json | Go on `PATH` | `go test ./pkg/jaxson -run '^$' -fuzz FuzzParseJSON -fuzztime 40s` | 2026-10-05, Go 1.27.1, 2 CPUs | 384329 inputs, 0 disagreements |
| Concurrency tests under `-race` | true multi-core parallelism | `go test -race -count=1 ./pkg/shaxon` | 2026-10-04, a 2-CPU sandbox | pass; a 2-CPU pass is weak evidence for races and should be repeated on a many-core machine |

`run.sh all` runs the first four in one go. Every run above is the author's;
none has been run on another machine yet.

## Phase 6 — Fixtures and conformance (plan §9) — mostly done, 2026-10-05

`pkg/shaxon/shaxon-v0.3.1-fixtures.json` holds 161 fixtures, all passing under
`TestFixtures` (`fixtures_test.go`); `pkg/shaxon/FIXTURES.md` describes the
format, the coverage by core section and what was left out. They were written
from the core specification, with the expected result fixed before the run;
every failure while writing them was a fault in the fixture (a wrong assert
form, an output that starts as null, a premise the 3.1 text had dropped), a
difference in spelling (S1), or, once, a defect in the engine (S8, `closed`
under `extends`). 70 expect an error, the rest a result.

| Item | Status |
|---|---|
| Runner, with spec-fixed-only matching of report violations | done |
| Fixtures for the plan's priorities (granularity split, `qualified` charge, `unique` ordering and gate rule; the purity priority became "no `into`, nothing is written" once S3 was ruled) | done |
| The six attic fixtures, rewritten for 3.1 | done (three premises had changed; recorded in each note) |
| Mutation check of the fixtures against the engine (`pkg/shaxon/fixtures_mutants.py`) | done: 43 mutants, 41 killed, 2 expected survivors with reasons |
| Navy Wars replay as an integration test (plan §9, optional) | not started |
| Fixtures run by a second, independent runtime | not started; none exists |

**Not independent evidence.** The fixtures and the engine share an author and
a reading of the specification. They show that the engine does what the text
says as one reader understood it; they do not show the reading is the only
one. The spec-gap list below is where the readings diverge.

### Spec findings from writing the fixtures

These add to P1, V1, V3, G11 and R1. IDs of decisions already logged above are
cited, not repeated.

| ID | Finding | State |
|---|---|---|
| S1 | Identifier prefix. The implementation prefixed Shaxon's own identifiers with `SHA_`, which the core did not. Settled 2026-10-05: the prefix is `SHAX_` (`SHA_` reads as the SHA family of hash functions) and the core now specifies it (§9, "Identifier prefix"). Code, tests, the authorisation cases, the fixtures, TRACKER, CHANGELOG and the v0.3.2 walk proposal follow; the runner's mapping is gone. | settled, 2026-10-05 |
| S2 | The merge table wrote `"override": true` on the child's `and`/`or`/`xone`/`not`/`check`, which a JSON array or a `check` object cannot carry. Settled 2026-10-05: the core now uses the implementation's spellings (`closedOverride`, `andOverride`, `orOverride`, `xoneOverride`, `notOverride`, `checkOverride`; per-item `override` for `fields` and the `{value, override}` form for `requiredIds`), states the `requiredIds` collision rule and that `check` with `checkOverride` is an error. 16 fixtures cover it. | settled, 2026-10-05 |
| S3 | Section 0 and section 1 said `check` and `validate` never write to any root; sections 7 and 8 give them an `into` that writes to `state` or `output`. Ruled 2026-10-05: the core makes no claim about it. The invariant (now "the three invariants") and the section 1 paragraph are removed, with nothing in their place; the editorial, limitations and v0.3.2 walk texts that cited purity follow. Revisit when there are users; analysis kept below. | settled, 2026-10-05: deliberately unspecified |
| S4 | The section 10 table is called exhaustive but has no row for a failed `qualified` count or an extra member of a closed object (G2, G3). The fixtures test both only through gate mode. | open, spec edit |
| S5 | Section 2 said a 3.1 runtime must also execute packages declaring `"1.0"` to `"3.0"`; R5 accepts only `"3.1"`. Settled 2026-10-05: the core now says a runtime accepts exactly the versions it implements and rejects any other with `VERSION_ERROR`; it need not execute earlier versions. R5 stands. | settled, 2026-10-05 |
| S6 | Not stated by the core: what counts as one level of `maxShapeDepth` (G8); what a plain shape activation costs (G1); whether `unique` skips an element lacking its field (V4); where a `PATH_DEPTH_EXCEEDED` finding is attached (C3). The fixtures assert none of them. | open, spec edit |
| S7 | Section 6 stops a chain at the first hop that does not resolve. A member holding `null` resolves, so a null-terminated list overruns a horizon one hop earlier than it looks. The text supports this; it is easy to miss, and one attic fixture was written the other way. Now `closure-through-a-null-link-still-resolves-and-can-overrun`. | recorded; no change needed |
| S8 | `closed` under `extends`. The merge table says AND, "most restrictive of parent/child wins". The implementation took AND to mean a boolean AND of the two `closed` flags, so the most permissive won: a closed parent extended by a child writing `closed: false`, or an open parent extended by a child writing `closed: true`, gave an open shape. Found by a fixture written from the spec. 2026-10-05: changed to the spec's reading (closed if either stated value is closed, unless `closedOverride`). A child that does not write `closed` still inherits the parent's posture; the core is silent on that and it stays an implementation decision. Revert is one `switch` in `mergeShapes`. | fixed, 2026-10-05; accepted by the user the same day |

| S9 | The core does not say in what order the members of a `$tpl` object are evaluated. The first interpreter ranged over a Go map, so with a failing member the error and the step total at the failure varied between runs (found 2026-10-05 by the oracle test, which compared the same program twice). The implementation now evaluates in sorted key order, as `$compute` bindings are. Proposed ruling: say so in the core (section on `$tpl`), and add a fixture. | open, spec edit |

### S3 in depth: is validation pure?

**Ruling, 2026-10-05: the core says nothing about purity.** The text below is the analysis that preceded it, kept as a record. It speaks of "four invariants" and of options A to C; none was taken. The invariant and the section 1 paragraph were removed instead, and the implementation behaves as the "What the implementation does" table describes.

**What the texts say.** Section 0 lists "validation purity" as one of the four
invariants: `check` and `validate` never write to `state`, `output` or any
root, "under any circumstance". Section 1 repeats it and adds the reason:
anything a validation needs is derived by an ordinary program step before the
`check`, which "keeps program state transition and predicate evaluation from
blurring". Section 7 then gives a report-mode `validate` an `into` ("the report
is written to the path named by `into`") and its own example writes
`state.lineWarnings`; section 8 says a report-mode `check` appends into its
`into` so "a program can accumulate a running log of non-fatal findings turn
over turn". The core forbids and requires the same write.

**What the implementation does** (all run, 2026-10-05):

| Case | Behaviour |
|---|---|
| No `into` | Nothing is written. Findings go to the combined report returned beside `output` (fixture `check-in-report-mode-without-into-never-aborts-and-leaves-state-alone`). |
| `into` under `state` or `output` | Allowed. The write is a synthesised `set` (`validate`) or `append` (`check`) run through the machine (V3), so it is an ordinary program write: one step per violation appended (22 steps against 15 for the same program without `into`), recorded in the mutation log, and indices over the written path rebuild. |
| `into` under `input` | `SHAX_SHAPE_ERROR`, "cannot write to root input". |
| `into` in gate mode | `SHAX_SHAPE_ERROR` (V2). |

**What purity is for.** Three things could go wrong if validation wrote:
(a) it changes the data other validation sees, so a result depends on the order
things were evaluated in; (b) the write escapes step accounting; (c) the write
is invisible to index reuse and leaves an index stale. The implementation
closes (b) and (c) by making the write an ordinary charged, logged program
write. It does not close (a).

**The hazard that remains.** Three report-mode `check`s of `state` with `into:
["state","log"]`, where `log` items must be strings, grew the log to 1, then 3,
then 7 entries: each pass validated the previous passes' reports as data. It is
deterministic and charged, so it breaks no invariant the core states other than
purity itself; but the second report is not a function of the data the author
thought they were validating. The same program without `into` gives three
identical reports.

**Verdict.** Neither text is right as it stands. The core's sentence is false
of its own sections 7 and 8, and the use it forbids is one the authorisation
examples depend on (all five deliver a report into `state` with `into` and read it
in their program). The implementation is the more defensible, because the write is
the program's own, made on validation's behalf, and is accounted for. What the
core should say is narrower than "never writes": validation never mutates the
data it validates, its only effects are the report and the step charge, and
delivering the report to `into` is an ordinary write. The feedback hazard is
real and not covered by that wording.

| Option | What changes | For | Against |
|---|---|---|---|
| A. Strict purity | Remove `into`. Add a pure expression operand (say `$validate`) that yields the report as a value; the program writes it with ordinary `set`/`append`. | The invariant is true as written; no feedback by construction; V2, V3 and R2 disappear; a report is usable mid-program as a value. | New operand form (v0.4.0 surface); the authorisation examples and the section 7 example are rewritten; the pre-program entries lose their direct delivery to `state`. |
| B. Narrow the invariant, keep `into` (recommended for 3.1 text) | Reword sections 0 and 1 as above; settle V3 (replace for `validate`, append for `check`); add a static rule that a literal `into` may not equal, contain or lie under a literal target path. | Matches what runs and what the examples use; one sentence and one load-time check. | The static rule cannot see dynamic targets (`$each` over a computed path), so feedback stays possible there and has to be documented. |
| C. B plus a run-time overlap check | As B, and raise an error when a delivery path overlaps any focus node evaluated in the same entry. | Closes (a) fully for one entry. | Costs a path comparison per delivery; does not stop feedback across two entries or two `check`s, as in the experiment. |

**Recommendation.** B for the 3.1 text now, and A on the v0.4.0 list as the
route to making the invariant true without a qualifier. The user ruled
for none of them (see the ruling above); the A route stays available for v0.4.0.

### Phase 6 open items

1. Navy Wars replay as an integration test (plan §9).
2. `fixtures_mutants.py` is a dormant guard (see the table above); its first
   record is 2026-10-05.
3. A fixture that pins `steps` exactly is possible only where the core fixes the
   whole total; none does yet, so `steps` is supported by the runner and
   unused.

## Performance, 2026-10-05

Six steps, none changing a result or a step count (the 161 fixtures and the 51
authorisation cases pin both); see CHANGELOG for the detail. Benchmarks
(`go test -run '^$' -bench Engine -benchmem ./pkg/shaxon/`, 2 CPUs, in ms):

| Benchmark | Original | Copy avoidance | Compiler | Maps, tuned | Objects |
|---|---|---|---|---|---|
| rolling-quota, 4000 events | 16.5 | 10.6 | 1.5 | 1.0 | 0.7 |
| rolling-quota, 20000 events | 82 | 46 | 7.3 | 4.6 | 2.9 |
| chinese-wall, 4000 events | 22.3 | 17.0 | 9.3 | 6.0 | 5.2 |
| chinese-wall, 20000 events | 91 | 78 | 37 | 23.5 | 22 |

The last column leaves the compile (the first run) out of the timing and holds
the input as Objects; the compile is under the run-to-run noise (the first run
takes about as long as a later one). The same benchmark with a map input, where
`Run` converts it each time, gives 1.5, 6.3, 6.6 and 24 ms.

Other measurements: 20000 `append`s 2100 ms to 4 ms; `ParseJSON` of a 920 KB trail
38 ms to 13.7 ms, `ParseData` of it 12.7 ms. After this, reading and building the value dominates:
at 20000 events `ParseJSON` (about 14 ms) now costs more than running the
rolling-quota package (3 ms). Objects were done in step 6 and gained 35-40% on
rolling-quota but only 6% on chinese-wall, which spends its time building and
sorting two indices. What is left is building `*big.Rat` values and, in the
parser, allocating them; getting past that needs a small-decimal number type,
which touches every operator and was not attempted. Tried and rejected: a global
string-to-integer dictionary and a packed `[4]uint64` key (no faster than the
string map in a lookup benchmark, and a dictionary of data-dependent keys never
shrinks). Not done, and why: specialised fixed-arity
operator calls (the argument slice is already scratch); a tokenizer-only
`ParseJSON` in the style of queryfy's superjsonic, which validates without building
values and so cannot feed the interpreter, which needs the tree.

### Open items

| ID | Item | State |
|---|---|---|
| PF1 | Delete the tree interpreter oracle when the compiler has been in use long enough: `pkg/jaxson/tree_oracle.go`, the `tree` switch and `UseTreeInterpreter`, the derived `OperatorDef.Apply`, the `Exec` members of the eight core instructions, `compile_oracle_test.go`, `shaxon/oracle_test.go` and the `SHAXON_BENCH_TREE` switch. The header of `tree_oracle.go` has the full list. | open |
| PF2 | `FuzzParseJSON` as a dormant guard: only its seed corpus runs in the default test run. | recorded in the table above |
| PF3 | **Wall time is quadratic in the trail on `four-eyes-release` and `break-glass`** (steps stay linear). Measured 2026-10-05 with all five authorisation examples (`examples/shacl/authz/results/REPORT.md`): four-eyes at 4000 events takes 2.8 s, at 20000 80 s; break-glass 4.6 s and 148 s; warmed-up Jena takes 8 and 144 ms at 4000 on the same two. Cause in four-eyes: the replay binds the whole `state.pay` map in a `$compute` `with` (`{"$path":["state","pay"]}`) only to ask `has` of it, and a composite read of `state` is cloned, so every event copies the growing map (profile: `Clone` and GC, 56%). Cause in break-glass: a review event has no `id`, so the key of its `reviews` binding is `""`, which matches every break-glass event in the index; `with` bindings are evaluated eagerly, so the unused value is built anyway (profile: `evalInverse`/`pathValue`, 52%). Candidate fixes: borrow, not clone, a `with` binding whose uses are all in operators that cannot keep or return it (`has`, `len`, `type_of`, `keys`, comparisons), and cloning only a composite result of `get`/`get_or`; and, in the package, a key that cannot collide for events that are not break-glass uses. Neither is done; the second changes an example package. | open |

## Phase 7 — Public API and tooling (plan §10) — partial

- [x] **7.1 Core API surface** (`shaxon.Validate`, the `jaxson.Number`
  boundary type, immutable-after-construction tables) — done and verified
  2026-10-05. `pkg/shaxon/api.go`, `pkg/jaxson/number.go`, `RunJSON` in
  `pkg/jaxson/profile.go`; tests `api_test.go`, `number_test.go`,
  `runjson_test.go`. Decisions where the plan is loose (**the signature and the
  Number scope need your confirmation**):
  - **A1** `Validate(pkg, shape, doc) (Report, *jaxson.Err)`, not the plan's
    `(pkg, target)`: `target` is a shape name and the document is a separate
    argument, so one package can validate many documents. `ValidateJSON` takes
    raw bytes for both.
  - **A2** What runs: the static checks, then one report-mode validation of
    `doc` placed at `input`. Not run: `program`, the package's own `validate`
    entries, `inputSchema`, `outputSchema`. Computes and `check` islands inside
    shapes do run. Indices and relations are built over `doc`, so their
    `source` paths start at `input`.
  - **A3** The result is always a Report (never a gate); errors are those Run
    would give.
  - **A4** Step limit is `limits.steps` or Run's default 100000; neither
    argument is changed.
  - **N1** `jaxson.Number`: an immutable exact decimal (`ParseNumber`,
    `NumberFromInt64`, `NumberFromRat`, `AsNumber`; `String`, `Rat` (a copy),
    `Sign`, `IsInt`, `Int64`, `Float64` with exactness, `Cmp`, `MarshalJSON`).
  - **N2** Results still hold `*big.Rat`; `AsNumber` converts. Changing the
    element type of every result would break each existing caller, and the
    machine shares Rats on purpose. A caller who keeps a Rat from a result must
    not change it; with `Number` they cannot. If you want results to hold
    `Number`, that is a separate, breaking step.
  - `jaxson.RunJSON` added to match `shaxon.RunJSON` (the plan asks for both).
    `Err` is documented (categories and the Cat/Code/Msg contract) and its text
    no longer has a stray `/` when Code is empty (`PARSE_ERROR: msg`).
  - Immutable tables: `CoreOperators` and the instruction tables are built per
    call and the compile cache belongs to a Machine; pinned by
    `TestConcurrentRunAndValidateShareAPackage` (8 goroutines, `Run` and
    `Validate` on one package, under `-race`).
- [x] **7.2 Session helper.** `src/jaxson-v0.1.0/play.go` (145 lines, in
  the pre-split `package main` tree, not `pkg/jaxson`) already implements
  the input/output-threading pattern the plan describes — clone the
  package, splice in carried-forward state, call `Run`, pull state back
  out. The finding behind this plan item is already proven correct; the
  promotion into a reusable `Session` *type inside the library* still
  hasn't happened. What has changed (2026-09-27): the driver itself is no
  longer stranded — `cmd/jaxplay/main.go` is `play.go` ported to call
  `pkg/jaxson`'s exported `Run`/`Normalize`/`Show` instead of its old
  sibling `jaxrun.go`'s unexported equivalents, as its own binary rather
  than a `jaxrun play` subcommand reached through a shared `init()` hack.
  Game logic and every printed message are unchanged — verified by
  diffing a scripted `-seed 0` session against
  `examples/game/example-game-seed0.txt`: identical on every real
  game-state turn (the recorded transcript's few illegal-move demo lines
  aren't in the scripted session, hence not compared).

  Update (2026-09-27): the reusable `Session` type this bullet said was
  still missing now exists — `pkg/jaxtools.Session` (`Step`, `Last`,
  `LastErr`, `Turn`), plus `Carry` for the state-splice half of the
  pattern, `LoadPackage`/`LoadTurns`, and a `Render` function for three
  package-agnostic output formats (`text`/`markdown`/`json`). This is a
  deliberate departure from §10's own layout, which sited `session.go`
  inside `pkg/shaxon` — and it is a departure made for a load-bearing
  reason, not convenience: §10 blocks `session.go` on Shaxon's own
  unstarted phases (2–6), and nothing about turn-threading actually
  depends on any of that — it only needs `pkg/jaxson`'s exported `Run`.
  Siting it in `pkg/jaxtools` instead means any application that only
  needs `jaxson` gets exactly the same runtime tooling — `Session`,
  `Carry`, `LoadPackage`/`LoadTurns`, `Render`, and now `jaxplay` — that
  Shaxon's own embedders will use once Phases 2–6 land, on the same
  timeline as `jaxson`-only users rather than after Shaxon ships. Once
  `pkg/shaxon` exists, it depends on `pkg/jaxtools` exactly as any other
  embedder would, rather than owning a second copy of this layer — so
  `jaxson`-only and Shaxon-based applications end up equally capable of
  driving multi-turn programs with the same tooling, not two tiers of
  support. `cmd/jaxplay/main.go` was rewritten from scratch on top of it
  as a genuinely package-agnostic REPL/CLI (`-format`, `-carry`,
  `-turns`) rather than the Navy-Wars-specific `play.go` port described
  above, which is now superseded in-tree (the original
  `src/jaxson-v0.1.0/play.go` + `jaxrun.go` are untouched and still the
  thing to run for the old, game-specific UX and as the verification
  oracle). `examples/game/turns-seed0.json` was added: the same seed-0
  transcript's real moves as a `-turns` file, so
  `jaxplay -carry game -turns examples/game/turns-seed0.json -format
  json examples/game/navywars.json` (flags before the package path —
  see "jaxtools verification, 2026-09-27" below for why the order
  matters) is *intended* as the way to cross-check the new library
  against the old one on Navy Wars — though that same verification pass
  found it doesn't currently complete past turn 1; see below.
  **Verified this session**: a Go toolchain (Ubuntu LTS `golang-go`,
  1.22.2) was installed and `go build ./...`, `go vet ./...`, and
  `go test ./...` all run clean across the whole module — 28/28
  `pkg/jaxson` fixtures pass (same names as Phase 0/1's list) and all
  eight `pkg/jaxtools` unit tests in `jaxtools_test.go` pass
  (`TestSessionStepAdvancesOnSuccess`, `TestSessionStepHoldsStateOnError`,
  `TestSessionBaseIsNotMutated`, `TestCarry`,
  `TestRenderJSONIsValidAndIndented`,
  `TestRenderTextAndMarkdownAreDeterministic`, `TestRenderUnknownFormat`,
  `TestNormalizeSanity`); `gofmt -l .` is clean. This closes the specific
  gap the previous snapshot left open — but this verification pass also
  surfaced two real issues while exercising `cmd/jaxplay` against Navy
  Wars, neither of which existed before this session's verification
  found them; see "jaxtools verification, 2026-09-27" below for what
  they were, and "cmd/jaxplay fixes, 2026-09-27" for how they were
  closed and re-verified, later the same session. With both fixed,
  build/vet/test clean, and the Navy Wars cross-check running end to end
  through two independent host loops (a shell-script launcher each,
  `examples/game/run-play-go.sh` and `examples/game/run-jaxplay.sh` —
  see that section), this item moves to `[x]`.
- [x] **7.3 Builder/compiler package.** `pkg/jaxson/build` (v0.3.0) is a
  fluent, chained, type-checked API rather than a direct port of the
  Python combinators: sealed `Operand`/`Expr`/`Instr`/`Schema` types,
  typestate `If(...).Then(...).Else(...)` and
  `For(...).As(...).Index(...).Do(...)`, one typed constructor per
  operator, `Package.Check/Map/JSON/Run`. Proven on a real-size program in
  v0.3.1: `examples/navywars` reproduces `examples/game/navywars.json`
  exactly (JSON-equal), and `jaxplay` on both packages produces
  byte-identical 8-turn transcripts ending in `won`. `build_navywars.py`
  is retained as the oracle, not as the source of truth.

## jaxtools verification, 2026-09-27

Build/vet/test results are recorded in the 7.2 bullet above. Beyond that,
this pass also ran `cmd/jaxplay` for real against
`examples/game/navywars.json` and `examples/game/turns-seed0.json` — the
exact command the 7.2 bullet's previous snapshot names as "the intended
way to cross-check the new library against the old one on Navy Wars".
Two issues turned up. Both are flagged here, not fixed, since fixing
either changes documented, public-facing behaviour rather than
correcting a typo:

- **The documented flag order doesn't parse.** The command as written
  everywhere it appears — `cmd/jaxplay/main.go`'s package doc (three
  separate example lines), its own `-h`/usage string, and the 7.2 bullet
  above — puts the positional `<package.json>` argument *before* the
  flags: `jaxplay examples/game/navywars.json -carry game -turns
  examples/game/turns-seed0.json -format json`. Go's standard `flag`
  package stops parsing at the first non-flag argument, so with the
  package path first, none of `-carry`/`-turns`/`-format` are recognised
  as flags at all — `fs.NArg()` comes back as 4, not 1, and the command
  prints its own usage message and exits 2. The flags have to come
  *before* the positional argument for this to work:
  `jaxplay -carry game -turns examples/game/turns-seed0.json -format
  json examples/game/navywars.json`. Confirmed both ways this session.
  This is a documentation defect repeated in three places, not three
  independent mistakes — worth checking for the same pattern anywhere
  else a `jaxplay` invocation is written down.
- **`-carry` splices the wrong thing for Navy Wars.** `step()`'s
  implementation matches its doc comment exactly — "splice the previous
  turn's *whole* output into this turn's input under key" — but Navy
  Wars' own `outputSchema` has three top-level fields (`game`, `view`,
  `messages`), and its `inputSchema.game` expects the *shape of
  `output.game` alone* (with a `seed` field, board arrays, and so on),
  not `{game: {...}, view: [...], messages: [...]}` nested one level
  too deep. Run for real (both via `cmd/jaxplay` and directly against
  `pkg/jaxtools`), turn 1 succeeds, then turn 2 fails immediately with
  `INPUT_ERROR/: $.game: missing required "seed"` — because
  `input.game.seed` doesn't exist; `input.game.game.seed` does. Confirmed
  the diagnosis by hand-carrying only `output["game"]` (rather than the
  whole output map) under the next turn's `"game"` key instead of using
  `jaxtools.Carry` as `cmd/jaxplay` calls it: with that one change, all 8
  scripted turns run cleanly and the game reaches `status: "won"` on
  turn 8, matching what `example-game-seed0.txt`'s real transcript
  expects for seed 0. So `Session`/`Carry`/`jaxson.Run` are not at
  fault — the mismatch is specifically in what `cmd/jaxplay`'s `-carry`
  flag chooses to splice. This means the 7.2 bullet's own recommended
  cross-check command does not currently work as written for Navy Wars,
  and needs a decision, not a patch: either `-carry key` should splice
  `previousOutput[key]` (the same-named sub-field) rather than the whole
  output — which would match Navy Wars' convention but silently assumes
  every stateful package names its input and output state fields
  identically — or the flag's contract stays as documented and Navy
  Wars simply isn't a package `-carry` can drive un-aided (in which case
  the 7.2 bullet's cross-check recommendation needs to say so, or be
  withdrawn).

Neither issue touches `pkg/jaxson` or `pkg/jaxtools`'s own unit tests,
which is why `go test ./...` above is genuinely clean — both bugs are in
`cmd/jaxplay`'s glue code and its documentation, not in the libraries it
calls.

## cmd/jaxplay fixes, 2026-09-27

Before fixing anything, the question this raised: does the mismatch mean
Navy Wars needs forking — a second copy of the example for whichever
tool doesn't currently run it? No. Both bugs above are in `cmd/jaxplay`
itself (an argument-order documentation error, and what `-carry`
chooses to splice), not in `examples/game/navywars.json`, not in
`pkg/jaxson`, and not in the Jaxson package format. The package already
runs identically old vs. new: `pkg/jaxtools.Session` driving
`navywars.json` directly (bypassing `cmd/jaxplay` entirely) completes
all 8 seed-0 turns to `status: "won"`, matching
`example-game-seed0.txt` — confirmed again below, this time through
`cmd/jaxplay` itself once its bug was fixed. One example, two host
loops, was already the right shape; it just needed the newer host loop
fixed, and something to paper over its one remaining rough edge (flag
order) for a person typing commands by hand.

Two changes, both in `cmd/jaxplay/main.go`, `pkg/jaxtools` untouched:

- **`-carry`'s default now splices the sub-field, not the whole
  output.** `step()` now reads `sess.Last().(map[string]any)[carryKey]`
  (when `Last()` is a map) and splices *that* under the next turn's
  `input[carryKey]`, rather than splicing all of `Last()` there. This is
  the fix option recorded as "1" in the previous discussion: it matches
  what Navy Wars' `outputSchema` (`game`/`view`/`messages` as siblings)
  actually needs, at the cost of assuming a package names its carried
  state the same thing on both sides of the boundary — true for Navy
  Wars, not guaranteed for every conceivable package.
- **`-carry-whole` added as the explicit opt-out.** Fix option "2":
  rather than silently dropping the old behaviour, it's now reachable
  on request — `-carry key -carry-whole` splices the entire previous
  output under `key`, exactly as `cmd/jaxplay` always used to. If
  `Last()` isn't a `map[string]any` (nothing to pick a sub-field out
  of), `step()` falls back to whole-output splicing regardless of the
  flag, since there's nothing else it could do. Both modes are
  documented in the package doc comment and in `-h`'s flag
  descriptions; the doc comment's example commands are corrected to put
  flags before the package path, matching what Go's `flag` package
  actually requires (see the previous section) — the same ordering bug
  doesn't need fixing twice, once in code and once in prose, if the
  prose is fixed at the same time as the code it describes.

Re-verified after the change: `go build ./...`, `go vet ./...` clean,
`gofmt -l .` clean, `go test ./...` still 28/28 fixtures + all 8
`pkg/jaxtools` unit tests, unchanged (this fix only touches
`cmd/jaxplay`, which has no test file of its own). Functionally, run
twice against the same seed-0 turns:

- Default mode — `jaxplay -carry game -turns turns-seed0.json -format
  json navywars.json` — now completes all 8 turns, reaching
  `status: "won"` on turn 8. This is the fix working.
- `-carry-whole` mode — same command plus `-carry-whole` — reproduces
  the *original* failure on purpose: turn 1 succeeds, turn 2 fails with
  `INPUT_ERROR/: $.game: missing required "seed"`, identical to what the
  unfixed default used to do. This confirms the flag actually switches
  behaviour rather than being cosmetic.

**Two launcher scripts added**, `examples/game/run-play-go.sh` and
`examples/game/run-jaxplay.sh` — not a fork, a convenience layer over
the one example, one script per host loop:

- `run-play-go.sh` runs the original `play.go`/`jaxrun.go` driver
  (`src/jaxson-v0.1.0`, no `go.mod`, so it's invoked with
  `GO111MODULE=off`), passing through `-seed N` and defaulting to
  `navywars.json` in the same directory as the script.
- `run-jaxplay.sh` runs the new `cmd/jaxplay` (from
  `src/jaxson-shaxon-v0.3.1`, its own module) against the same
  `navywars.json`, defaulting `-carry game` behaviour to the fixed,
  sub-field splice. It builds the binary from inside that module (where
  `go.mod` resolves) but *runs* it from wherever the script was invoked
  from, specifically so a relative path you type — `-turns
  turns-seed0.json` from inside `examples/game`, say — resolves against
  your own shell rather than against the script's internals; an earlier
  draft of this script `cd`'d into the module directory for the whole
  invocation and broke exactly that case, caught by actually running it
  with `-turns turns-seed0.json` before considering it done, not by
  inspection. It also appends the package path after any flags you
  pass, so the flag-before-positional-argument ordering `cmd/jaxplay`
  requires doesn't have to be remembered at the call site.

Both scripts were run for real this session, not just read: interactive
mode (piping `q` in immediately) and scripted mode (the full seed-0
`-turns` run) for each, plus the `-carry`/`-carry-whole` comparison
above through `run-jaxplay.sh` directly.

## Exported primitives closed; files5 integration, 2026-09-27

Two pieces of work, same session, kept separate below since they touch
different parts of the tree and carry different verification status.

**files5 integration.** The `cmd/jaxplay` fixes and the two launcher
scripts described above (the `-carry`/`-carry-whole` split and
`examples/game/run-play-go.sh` / `run-jaxplay.sh`) arrived this session
as a separate delivered archive and were copied into place byte-for-byte
(diff-confirmed against the delivered copies before and after). No new
edits were made to any of it here — the account in "jaxtools
verification, 2026-09-27" and "cmd/jaxplay fixes, 2026-09-27" above,
including their build/vet/test results, already covers this content and
is not re-claimed as newly verified in this pass.

**Exported primitives, closed.** The one item Phase 1 still owed —
`clone`/`equal`/`typeName`/`order`/`sortedKeys`/`fmtNum` in `values.go`,
still lowercase as of the previous snapshot — is now closed: all six
capitalised (`Clone`, `Equal`, `TypeName`, `Order`, `SortedKeys`,
`FormatDecimal`), every call site across the package updated to match.
While in `values.go`, one adjacent documentation defect was fixed at the
same time, on the "fix silently, mechanical" test (§4.3-equivalent):
`Normalize`'s own doc comment had been separated from `func Normalize`
by the `Order` function and its doc comment landing directly in between
(a pre-existing structural issue, not introduced by this pass) — `godoc`
would have attached the "Normalize converts a value..." text to `Order`
instead, and left `Normalize` undocumented. Reordered so each function's
doc comment sits directly above it again; no behavioural change, nothing
public added or removed.

Verification performed, and its limits:

- A tree-wide grep sweep for all six lowercase names (`clone(`,
  `equal(`, `typeName(`, `sortedKeys(`, `order(`, `fmtNum(`), across
  every `.go` file in the module — not scoped to `pkg/jaxson` — finds
  exactly one remaining match, `(c *Checker) clone()` in `program.go`
  plus its one call site `c.clone()`. That's a method in a different
  namespace from the free function this item renames, and was
  deliberately left alone.
- A brace/paren balance check across every edited file (a rough proxy
  for gross syntax damage, not a substitute for a real compiler) is
  clean in all seven touched files.
- **Not verified with `go build`/`go vet`/`go test`.** No Go toolchain
  is installed in the environment that made this edit (`go: command not
  found`), and this environment's network access is disabled, so neither
  installing Go directly nor following this document's own §6.3
  offline-proxy procedure was possible here. This is a real gap, not a
  formality: the previous two verification passes recorded in this
  document (in "jaxtools verification, 2026-09-27" and "cmd/jaxplay
  fixes, 2026-09-27") both used a real `go build`/`go vet`/`go test` run
  and both found something the grep-and-inspection-only pass before them
  had missed. Treat this closure the same way — plausible, mechanically
  swept, but not yet the thing this document elsewhere calls "verified."
  Run `go build ./... && go vet ./... && go test ./...` locally before
  relying on it, and if anything fails, `values.go`'s six renames and
  their call sites in `machine.go`/`compute.go`/`schema.go`/`program.go`/
  `fixtures_test.go` are the first place to look.

## Phase 2 scaffolding: pkg/shaxon, 2026-09-30

First Shaxon code in this repository — `pkg/shaxon/{errors,registries,
parse_computes,parse_indices,parse_relations,parse_shapes,recursion}.go`.
Parses and statically validates `shapes`/`indices`/`relations`/`computes`
(core sections 2–4d), independent of any input, per plan section 5.
`ParseRegistries(pkg) (*Registries, *jaxson.Err)` is the entry point —
shaped to be called from a future `Hooks.Static`, not yet wired to one.

**Settled, on request, rather than left open:**
- An `extends` cycle is unconditionally `SHAX_SHAPE_ERROR`, never valid
  under any `maxShapeDepth` value — static merging has no focus node to
  descend through, so no depth bound makes a literal cycle terminate.
- `validate`/`check` shape-xor-`unique` structural checks belong in
  `targets.go` (Phase 4), not here — this package's `registries.go` stays
  scoped to exactly the four registries the plan's file layout names.

**Error identifiers: every one Shaxon introduces is prefixed `SHAX_`**
(`SHAX_SHAPE_ERROR`, `SHAX_VALIDATION_ERROR`/`SHAX_SHAPE_MISMATCH`,
`SHAX_DANGLING_REFERENCE`, `SHAX_SHAPE_DEPTH_EXCEEDED`,
`SHAX_PATH_DEPTH_EXCEEDED`). `EXECUTION_ERROR` itself keeps Jaxson's own
spelling (core section 9: "the existing Jaxson category, extended," not a
new one), and `TYPE_ERROR`/`MISSING_PATH` keep Jaxson's spelling where
reused unchanged — only Shaxon's own new codes within that category are
prefixed.

**One upstream change to `pkg/jaxson`, additive and verified**:
`OperatorDef` gained `MinArity, MaxArity int`, filled from the existing
private `arity` table in `operands.go` (one source of truth, not two).
Full repo build/vet/gofmt/`test -race` re-run clean afterward. Originally
made because Shaxon's `computes` registry seemed to need arity bounds
exposed to validate expressions without a `Machine`. Correction, found
while actually writing `parse_computes.go`: it didn't — `jaxson.CheckOperand`
already delegates a compute body's full structural check (arity included)
to Jaxson's own internal checker. The `OperatorDef` change is kept anyway
— it's genuine, additive, harmless, and plausibly useful for future
tooling — but the justification that motivated it turned out to have a
better alternative, found a few steps later than it should have been.

**Known, explicitly out of scope for this batch, not overlooked:**
- The `"override": true` / `{"override": {...}}` extends mechanism has no
  implementation — core section 4 never shows its JSON syntax anywhere,
  only its semantics in prose. Default (union/AND/concatenate) merging is
  complete and tested; override is a confirmed gap, not a guess.
- Primitive keywords (`minLen`/`maxLen`/`enum`/`min`/`max`/`int`/
  `minItems`/`maxItems`) are type-checked and carried through in
  `KeywordDecl`, with the one rule core section 4 states explicitly
  enforced (`enum` restricted to string/number/boolean/null). Cross-
  keyword sanity (e.g. `minLen <= maxLen`) is not yet enforced.
- Everything core section 9 marks as needing real data — a repeated
  non-`multi` index key, `cardinality: "one-to-one"` uniqueness, a
  `unique` declaration's own checks — is correctly left to Phase 4.

**Verified**: `go build`/`go vet`/`gofmt -l .` clean; `go test -race ./...`
clean across every package including the new `pkg/shaxon`, which has its
own 18-case test file covering a full valid package, the extends merge
(including the `closed`-default bug below), and one test per `SHAX_SHAPE_
ERROR` condition implemented so far.

**Self-check note**: the first draft of the `closed` member's default got
it backwards — Go's zero-value `false` instead of core section 4's stated
default ("closed objects are the default posture" means `true` when
undeclared). Caught by `TestParseRegistries_ExtendsMerge` failing, not by
inspection. Worth double-checking stated defaults against Go zero values
specifically in any future section-4-adjacent work — this is exactly the
class of bug Go's zero values make easy to introduce silently.

## Phase 2 completion, 2026-09-30

Closed the two gaps the scaffolding entry above flagged as remaining
within Phase 2's own scope (the data-dependent and `targets.go`/`unique`
items were never Phase 2's to close):

- **Keyword-vs-kind legality**, mirroring jaxson's own `schemaKeys`
  table: `minLen`/`maxLen` only on `string`, `min`/`max`/`int` only on
  `number`, `minItems`/`maxItems` only on `array`. Plus cross-keyword
  ordering (`minLen<=maxLen`, `min<=max`, `minItems<=maxItems`).
- **A real ordering bug caught while wiring the above in**, not after:
  keyword-vs-kind legality has to run *after* `extends` resolves `Kind`
  inheritance, not on each shape's pre-merge body — a child extending a
  `kind: "string"` parent and adding `minLen` without repeating `kind`
  would otherwise be wrongly rejected (pre-merge `Kind` is still `""`).
  Fixed by moving the check to run post-merge for named shapes, and by
  restricting `extends` to named shapes only (every spec example already
  uses it this way), which removes the same hazard for inline shapes by
  construction rather than by special-casing it.
- **A real gap found in the same pass**: an inline field/items/qualified-
  target shape's own `"extends"` key was silently ignored — neither
  honoured nor rejected. Now explicitly `SHAX_SHAPE_ERROR`
  ("extends is only valid on a named, top-level shape"), consistent with
  the restriction above.
- **`parseQualified`'s inline branch bypassed `parseField` entirely**,
  so it had neither the new extends-rejection nor the keyword/kind
  validation call. Same fix applied there directly.

**The `extends` `"override"` mechanism remains deliberately
unimplemented**, as of this entry. Three candidate JSON syntaxes were
drafted against core section 4's prose and rejected: (A) a flag inside the
item itself (fits `fields`, has no slot for `closed`/`and`/`or`/`xone`/
`not`/`check`); (B) a shape-level `"override": [...]` list of member names
to replace outright (fits `closed` and the combinator lists, doesn't
address per-field or per-`requiredIds`-key granularity, doesn't fit
`check`'s documented `{"override": {with, expr}}` replacement-value
form); (C) a single `override` object keyed by member name whose
value-shape varies per key — internally inconsistent enough at the time to
suggest reverse-engineering the wrong design rather than the real one.
**Superseded a few hours later**, same day — see "extends override
mechanism implemented" below: re-reading core section 4's merge table
verbatim (rather than the paraphrase this entry was working from) showed
the per-member value genuinely does vary, which is exactly what candidate
C guessed and I wrongly treated as a red flag against it. Left standing
here as the honest record of what going in circles on a spec gap actually
looks like, not retouched to read as if the right answer were obvious
from the start.

**Verified**: `go build`/`go vet`/`gofmt -l .` clean; `go test -race
./...` clean across every package. `pkg/shaxon`'s test count: 18 → 24.

## Imported from a parallel checkpoint, 2026-10-04

A separate drafting team delivered a checkpoint that branched from this
repository and went its own direction rather than continuing Phase 2.
Diffed in full first; imported only what was genuinely new, nothing that
would regress anything here:

**Imported:**
- `shaxon-v0.3.2-proposal-walk.md` (+ `attic/`'s rev1/rev2 drafts) — a new
  proposal: a "Walk profile," modeling a bounded session as a walk over a
  reference graph, for checking access behaviour against a normative
  model. Targets core 3.1.
- `examples/walk/` — the proposal's first executable evidence: a hop
  kernel as a pure Jaxson package, a Python differential test, and cost
  measurements. **Re-run here, not just trusted from their report**:
  `test_w3.py` — 600 slices, 0 mismatches against an independent
  reference, 0 differences under replay or member-order shuffling,
  matching their stated numbers exactly. `measure.py` also re-run
  cleanly. Both are Python-only — no Shaxon evaluation layer exists yet
  to validate against, by their own account and ours.
- `examples/showcase/SHOWCASE-REPORT.md` — a written report on the
  showcase examples, self-labelled by evidence type; explicitly notes
  nothing in it was run against Go, since they had no Go toolchain.
- `jaxson-intro.md`, and root-level `LICENSE`/`README.md` copies (content
  identical to the existing nested `src/jaxson-shaxon-v0.3.1/` copies —
  confirmed by diff before copying, not assumed).
- One bugfix to `examples/showcase/tools/jaxsonpy.py`: a zero-iteration
  `for` loop crashed the Python executor (`del` on an unbound key);
  changed to `.pop(key, None)`, matching Go's `delete()`, which is always
  a safe no-op. Applied as a targeted patch to the existing file, not an
  overwrite. **Re-verified here**: `run_showcase.py` still 29/29 after
  the patch.

**Deliberately not imported:**
- Their `pkg/shaxon` (`registries.go`/`parse_shapes.go`/
  `registries_test.go`, 18 tests). Diffed and confirmed: it is this
  repository's *original* Phase 2 scaffolding checkpoint, predating both
  "Phase 2 completion" and the `extends` override mechanism above.
  Importing it would have been a straight regression. Their `TRACKER.md`
  still carries the original `## Phases 2–6 — not started` header despite
  their own `pkg/shaxon/` containing Phase 2 scaffolding code — the same
  stale-summary mistake caught and fixed in this document earlier, left
  uncaught in theirs.
- `jaxson-v0.1.0-core-design.md`'s rewrite: purely editorial (an added
  Introduction section, depersonalized voice, reworded headers), no
  technical content changed. Not an unambiguous improvement over the
  existing copy, so left as-is rather than overwritten for a style
  preference.

**Verified**: Go side untouched by this import — `go build`/`go vet`/
`gofmt -l .`/`go test -race ./...` all re-run clean. Python side: both
newly-imported scripts actually executed here, not trusted from their own
report; `run_showcase.py` re-run after the `jaxsonpy.py` patch, still
29/29.

## extends override mechanism implemented, 2026-09-30

Prompted by asking which of the three rejected candidates above would
match a SHACL specialist's expectations. Answering that properly meant
re-pulling core section 4's merge table verbatim instead of trusting my
own earlier paraphrase of it — the verbatim text settled the question
outright rather than just favoring one candidate:

```
| closed      | AND         | "override": true makes the child's own value win outright        |
| fields      | union, collision = error | per-field "override": true replaces the parent's field |
| requiredIds | union by key | per-key "override": true replaces the parent's id for that key   |
| and         | concatenate | "override": true on the child's and replaces the parent's list    |
| not         | both apply  | "override": true on the child's not replaces the parent's         |
| check       | ANDed       | "override": {"with":..., "expr":...} on the child replaces it     |
```

Implemented exactly this: whole-member `fooOverride` sibling keys
(`closedOverride`, `andOverride`, `orOverride`, `xoneOverride`,
`notOverride` — all booleans — and `checkOverride`, taking the literal
`{with, expr}` replacement per the table's own distinct form for that
row); per-item `"override": true` nested inside the item itself for
`fields` (matching "per-field") and `requiredIds` (matching "per-key",
via `requiredIds`' bare-string values now optionally promoted to
`{"value":..., "override":...}` — resolving the one blocker none of the
three earlier candidates could, by applying the same literal-to-object
promotion RDF/SHACL use whenever a bare value needs metadata attached).
`"override"` is rejected as an unknown key anywhere other than `fields`
(items/qualified-target/and/or/xone/not list positions), rather than
silently accepted and ignored — checked by
`TestParseRegistries_OverrideRejectedOutsideFields`.

**Two real ordering bugs surfaced while wiring this in, both the same
class as the `Kind`-inheritance bug from the previous entry, caught by
tests failing, not by inspection:**

- `requiredIds`' "must name a member in `required`" check was validated
  against each shape's own *pre-merge* `required` list. A child adding
  `requiredIds` for a name only its *parent* requires was wrongly
  rejected. Fixed by moving the check to run post-merge
  (`validateRequiredIds`), parallel to `validateKeywordsAgainstKind`.
- Also added, previously missing entirely: a `requiredIds` key colliding
  between parent and child without `"override": true` now correctly
  raises `SHAX_SHAPE_ERROR`, matching `fields`' existing collision
  behaviour. Before this batch the child silently won on every
  `requiredIds` collision regardless of any flag — not wrong relative to
  anything tested before now, but inconsistent with `fields`' parallel
  construct and with what the override table implies the *un-flagged*
  default should be.

**Verified**: `go build`/`go vet`/`gofmt -l .` clean; `go test -race
./...` clean across every package. `pkg/shaxon`'s test count: 24 → 33 (9
new cases: per-field override success and its still-an-error-without-the-
flag counterpart, override rejected outside `fields`, `closedOverride`,
`andOverride` replacing rather than concatenating, `checkOverride`
replacing rather than ANDing, `check`/`checkOverride` mutual exclusivity,
`requiredIds` per-key override, and `requiredIds` collision-without-
override).

## Showcase examples merged into this branch, 2026-09-28

`examples/showcase/` and `pkg/jaxson/showcase_test.go` were authored on a
separate branch (no Go toolchain in that environment) taken from this
repository *before* the "Phase 1 closure, v0.2.0" work below — that
branch still had `Run()` in `machine.go`, no `run.go`, no extension
mechanism (`host.go`/`profile.go`), no `CheckProgramHost`/`CheckSchema`/
`Validate`, and no packaging files. Its own tracker entry is otherwise
reproduced as written; only its "Unverified against Go" caveat is
superseded here, since that verification has now actually happened:

`examples/showcase/` adds 12 example packages (4 general, 4 finance, 4
cloud/data-science control-plane) and 29 fixture cases, covering all 8
instructions and all 30 compute operators. See its README.

Merged onto this branch (not the branch that authored it) rather than
merging this branch's own newer library work backward into that older
state. Only two files were added — `examples/showcase/` and
`pkg/jaxson/showcase_test.go` — nothing already on this branch was
changed to accommodate them.

**Verified, on this merge, with a real Go toolchain**: `go build ./...`,
`go vet ./...`, `gofmt -l .` all clean; `go test -race ./...` clean
across every package (`pkg/jaxson`, `pkg/jaxson/build`, `pkg/jaxtools`,
`examples/navywars`). `TestShowcaseFixtures` passes all 29 cases against
this branch's implementation, expected values taken from the independent
Python port (`examples/showcase/tools/jaxsonpy.py`) noted in that test's
own comment — a passing run here confirms agreement between the two, not
that either is independently proven correct.

## Present, but out of this plan's scope (plan §12)

Listed for orientation only — not tracked as progress against this plan,
since the plan explicitly excludes them:

- `analysis-WIP/shaxon-v0.3.1-review01.md` through `review07.md` — a
  further review series, exploring what its own framing calls a "bounded
  compositional expressions" layer.
- `shaxon-delta/` — Delta-G graph-transformation notes and two figures.
- `attic/` — superseded drafts kept for history: Jaxson v0.0.1b (the
  original 6-part draft), Shaxon v0.0.1 / v0.1.1 / v0.1.2 (with their own
  small fixture sets — plan §9 flags these as worth mining for test cases,
  rewritten to declare `"shaxon": "3.1"`, rather than running as-is) /
  v0.2.0, and `shaxon-v0.3.1-proposal-alpha.md`, superseded by the merged
  `shaxon-v0.3.1-proposal.md`/`shaxon-v0.3.1-core.md`.

## Consolidation, 2026-09-27

Prior to this snapshot, the Phase 1 rewrite existed as four files
(`program.go`, `machine.go`, `compute.go`, `schema.go`) sitting directly
under `pkg/`, one level above the plan's `pkg/jaxson/` target — orphaned:
nothing imported that location (`cmd/jaxrun/main.go` still pointed at
`pkg/jaxson`), and it didn't build (20 undefined symbols — the primitive
helper layer had never been ported over). Reconciled as follows, merging
into `pkg/jaxson/` and dropping nothing:

- `errors.go`, unchanged.
- `values.go`: kept, plus `order()` (moved from the old `machine.go`,
  body copied verbatim rather than reconstructed) and `numRe` (moved from
  the old `checks.go`) — both homeless after the split, both belonging
  here per plan §2's own file description.
- `paths.go`, `operands.go`: new files, old `checks.go` redistributed into
  them exactly as that file's own header comment already specified,
  signatures updated from bare `locals map[string]bool` to `*Checker`.
  Old `checks.go` deleted — every line in it moved to one of these two
  files or to `values.go` (`numRe`); nothing was dropped.
- `schema.go`, `machine.go`, `program.go`, `compute.go`: the versions from
  the orphaned `pkg/` promoted in — `schema.go` and `machine.go` are
  each a strict superset of their old `pkg/jaxson/` equivalents (confirmed
  by diff before deleting the old copies), `program.go` and `compute.go`
  have no old equivalent at all.

Verified after merging: `go build ./...` and `go vet ./...` clean,
28/28 fixtures pass, `cmd/jaxrun` builds. `gofmt -l` flagged four files for
trailing-newline/alignment only (fixed); one comment in `compute.go` still
named the now-deleted `checks.go` as where `arity` lives (corrected to
`operands.go`).

## Phase 1 closure, v0.2.0 (2026-09-27)

Re-verified from a fresh unpack of the checkpoint, Go 1.22.2 (Ubuntu LTS
`golang-go`): `go build ./...`, `go vet ./...`, `gofmt -l .` clean;
`go test -race -count=3 ./...` clean. This supersedes the "not verified"
caveat in the exported-primitives section above: the six renames build and
pass. A grep sweep again finds only the unrelated `(c *Checker) clone()`.

Work done in this pass:

- **Extension points now have tests.** `pkg/jaxson/extension_test.go`
  (external package, so it only sees what a host can reach): a host
  instruction registered into the table is statically checked and
  executed; `Checker.Host` reaches its `Check`; it inherits the generic
  missing/unknown-field checks; its `Step()` calls share the machine's
  limit; `OnMutate` fires once per `set`/`append`/`insert`/`delete`
  (deletes report the container's path) and not on a failed mutation;
  the core tables are fresh per call; concurrent `Run` is race-clean.
- **Two gaps the tests exposed, closed additively.** No exported path could
  set `Checker.Host` (added `CheckProgramHost`), and a host could not repeat
  the schema stages of the pipeline (added `CheckSchema` and `Validate`).
  No existing signature changed.
- `Run` split out of `machine.go` into `run.go`.
- LICENSE, README, VERSION, `pkg/version`, CHANGELOG added.

Open, deliberately not changed: `Run` takes no instruction table, so a host
must reimplement the pipeline order using the pieces above; and jaxson's
comments still name Shaxon in places, against plan §4's "no Shaxon
vocabulary anywhere in `jaxson`" (code identifiers are clean).

## Keeping this current

Each time this is updated: re-run `go build ./...`, `go vet ./...`, and
`go test ./...` rather than copying forward the previous result, and
re-grep the source for the Phase 1 markers (`OnMutate`, `CoreInstructions`,
`type Checker`, `type InstructionDef`, `CoreOperators`) and for
`pkg/shaxon`'s existence, rather than trusting this document's own last
snapshot — this snapshot exists precisely because the previous one claimed
Phase 1 was "not started" while a build of the actual source tree said
otherwise. The plan's "current state vs. target" note in its own §2 is
exactly the kind of claim this document exists to keep independently
re-verified, not the kind to take on faith once and reuse. When checking
the "exported primitives" item specifically, grep the whole module (not
just `values.go`) for lowercase `clone(`/`equal(`/`typeName(`/`order(`/
`sortedKeys(`/`fmtNum(` — as of this snapshot the only survivor is the
unrelated `(c *Checker) clone()` method in `program.go`, and that's
expected, not a regression to chase. This item is closed and was re-verified with a real
build/vet/test run in v0.2.0 (see "Phase 1 closure, v0.2.0").

The two `cmd/jaxplay` issues logged under "jaxtools verification,
2026-09-27" were fixed the same session — see "cmd/jaxplay fixes,
2026-09-27" — so don't re-flag them from that first section without
reading the second one too; re-run `./examples/game/run-jaxplay.sh
-carry game -turns turns-seed0.json -format json` (or the equivalent
direct `cmd/jaxplay` invocation, flags before the package path) to
confirm they're still fixed before trusting this account of them,
rather than assuming a fix recorded once stays fixed forever.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0