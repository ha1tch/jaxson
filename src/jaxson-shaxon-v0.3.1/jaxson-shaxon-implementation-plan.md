# Jaxson/Shaxon Go implementation plan

Status: proposal, unimplemented. Planning artifact only — no code has been
written against this plan yet.

## 0. Scope and goals

Two outcomes, pursued together rather than sequentially:

1. Turn the existing `jaxrun.go` (a 1,300-line, single-file, `package main`
   specification check) into a reusable `jaxson` library, without changing
   its behaviour.
2. Implement Shaxon (targeting the `shaxon-v0.3.1-core.md` package
   format, into which `shaxon-v0.3.1-proposal.md` has now been merged) as a
   second library, `shaxon`, that depends on `jaxson` and adds no forked
   copy of anything `jaxson` already does correctly.

"Reuse as much as possible" is treated literally: every place this plan
calls for new Shaxon-only code, it says so explicitly, and everywhere else
the intent is that `shaxon` calls into exported `jaxson` functions rather
than re-implementing them. Both packages are designed to be imported by
applications other than the two reference CLIs.

## 1. Why `jaxrun.go` cannot be reused as-is

`jaxrun.go` hardcodes the instruction set and the compute-operator set as
two Go `switch` statements (one in the static checker, one in the
executor), backed by a few package-level `var` tables. Shaxon adds a ninth
instruction (`check`) that must run interleaved with the existing eight.
A `switch` statement in another package cannot be extended with a new
`case`. Everything else needed for reuse — path walking, operand
evaluation, the value model, the primitive schema-keyword checks — is
already correct, just unexported.

One gap is not cosmetic. Shaxon's index-reuse rule
(`shaxon-v0.3.1-core.md`, section 3) states that rebuild avoidance is
"decidable from the mutation-path log already required for the
no-aliasing guarantee." `jaxrun.go`'s no-aliasing guarantee is implemented
by deep-copying on every read (`clone()`), not by logging mutations —
there is no mutation-path log in the existing code. This plan adds the
minimal mechanism needed to give Shaxon one, described in section 4.

## 2. Package layout

```
pkg/jaxson/                  package jaxson — the core language only
  values.go     decimal number model; clone/equal/typeName/order/show
  paths.go      path segments; walk/getAt/putAt
  operands.go   $path / $lit / $compute / $tpl / $opt
  compute.go    operator table + apply()
  schema.go     inputSchema/outputSchema + shared primitive-keyword checks
  program.go    instruction table + the 8 core instructions
  machine.go    Machine type, Step(), OnMutate hook
  errors.go     Err type, categories/codes
  run.go        Package type, Run(), CheckProgram()
  build/        optional: Go combinator package (section 7.3)
  fixtures_test.go   the existing 28 fixtures, as a go test

pkg/shaxon/                  package shaxon, imports jaxson, changes nothing in it
  registries.go  indices/relations/shapes/computes: parse + load-time checks
  indices.go     build/rebuild, reuse via jaxson's mutation hook
  shapes.go      kind/fields/closed/extends/combinators/qualified/check
  targets.go     $path / $each / $discriminator / $indexed / $path* / $path+
  report.go      violation records
  check_instr.go registers "check" against jaxson's instruction table
  run.go         shaxon.Run(): version gate, pipeline order
  session.go     optional: multi-turn Session helper (section 7.2)
  fixtures_test.go

cmd/jaxrun/       thin CLI, calls jaxson.Run
cmd/shaxonrun/    thin CLI, calls shaxon.Run
examples/game/    Navy Wars, importing jaxson as a library instead of
                  living inside package main
```

`internal/` is not an option for `pkg/jaxson` or `pkg/shaxon`: Go's
compiler restricts an `internal/` package to importers inside the same
module tree, which would contradict section 0's requirement that both
packages be importable by applications other than the two reference CLIs.
`pkg/` itself is a naming convention, not a Go toolchain requirement —
adopted here on explicit instruction, not because the plan depends on it.

Module path is left open — set it to match wherever this repository
actually lives; nothing below depends on the choice.

**Current state vs. this target (as of the Phase 0 work already done):**
the code at `pkg/jaxson/` today is five files — `values.go`, `checks.go`
(covers this diagram's `paths.go`+`operands.go`+part of `program.go`),
`machine.go` (covers this diagram's `machine.go`+`run.go`), `schema.go`,
`errors.go` — not yet split into the nine above. Confirmed working:
`go build`/`go vet` clean, all 28 fixtures pass, output identical to the
original `jaxrun.go` run against the same fixture file. Phase 1 proceeds
on this five-file layout rather than reorganising first; `program.go`
(the instruction table) is expected to fall out as its own file
naturally once Phase 1 adds it, without a separate reorganisation step.

## 3. Phase 0 — Extract, don't change

Split `jaxrun.go` along the file boundaries in section 2, mechanically.
Same logic, same behaviour, identifiers exported only where Shaxon or an
external caller needs them. Turn `main()`'s fixture loop into a
table-driven `go test` against `fixtures.json`. Move the CLI into
`cmd/jaxrun/main.go`, calling `jaxson.Run`.

This phase changes no behaviour and produces the regression net every
later phase is checked against: if the 28 existing fixtures do not pass
identically after the split, something leaked.

## 4. Phase 1 — The extension points Shaxon needs, and no others

- **Instruction table.** Replace both hardcoded `switch` statements with a
  table: `{name, requiredFields, optionalFields, checkFn, execFn}` per
  instruction. `jaxson.CoreInstructions()` returns the existing eight,
  unchanged in behaviour. A runtime is core-table-plus-extra-entries;
  Shaxon's `check` instruction is one more entry, registered from the
  `shaxon` package.
- **Operator table.** Same treatment for the compute operators (`add`,
  `eq`, `concat`, ...), made data instead of an implicit `switch`. Not
  required for v0.3.1 (no new operators), but `jaxson-v0.1.0-core-design.md`
  section 7 already anticipates a later named-profile operator, and the
  cost of doing this now, while the file is already being restructured,
  is close to zero.
- **Static-check context.** Extend the context passed to instruction
  checkers from bare `locals map[string]bool` to a small `Checker` value
  that also carries an open slot for host-specific state, so Shaxon's
  `check` instruction can confirm `shape` names something declared in
  `shapes` without `jaxson` knowing shapes exist.
- **Mutation hook.** Add one optional field to `Machine`:
  `OnMutate(root string, segs []any)`, called after each successful
  `set`/`delete`/`append`/`insert`. Cost to Jaxson-only callers is one nil
  check per mutation. This is what closes the mutation-log gap from
  section 1 — Shaxon sets the hook to build its own log and checks it
  before every index rebuild, instead of duplicating the four mutation
  instructions.
- **Exported primitives.** `clone`, `equal`, `typeName`, `order`,
  `sortedKeys`, canonical decimal formatting, and `Machine` methods for
  path walking, operand evaluation, and running a compute island. Also
  factor the primitive schema-keyword checks (`minLen`/`maxLen`/`enum` on
  strings, `min`/`max`/`int` on numbers, `minItems`/`maxItems` on arrays)
  out of `validate()` into standalone functions — a shape's `fields` reuse
  the identical keywords and should call the same checks, not
  re-implement them.

Deliverable: same library, same 28 fixtures passing, now with exactly the
extension points Shaxon needs and no Shaxon vocabulary anywhere in
`jaxson`.

## 5. Phase 2 — Shaxon's static layer

Parse `indices`, `relations`, `shapes`, `computes` once at load time,
independent of input — same spirit as today's `checkSchema`, larger
vocabulary. Implement every `SHAPE_ERROR` condition from `shaxon-v0.3.1-core.md`
section 9's error table: malformed shape, index, `relations` or
`computes` entries, a missing `maxShapeDepth` where recursion is
reachable, a repeated key in a non-`multi` index, a `requiredIds` key
absent from `required`, `enum` declared on a `kind` other than
`string`/`number`/`boolean`/`null`, a `validate`/`check` entry declaring
both `shape` and `unique`, a malformed `unique` declaration, a redundant
relation/shorthand pair, and so on.

The `extends` merge table (closed = AND, fields = union with
collision check, required = union, requiredIds = union by key,
and/or/xone = concatenate then apply, check = AND, each with its own
`"override"` behaviour) is implemented as one general "merge shape"
function driven by a small per-member table, not nine hand-written
special cases — which also happens to be a cleaner version of what
`shaxon-v0.3.1-core.md` section 4 itself specifies. `extends` takes a
single parent name by design (core resolves the open question the
v0.3.0 draft had left open here: composing without field-level merging
is `and: [...]`; `extends` is for when flattening into one shape is the
actual goal) — no multi-parent resolution logic is needed.

## 6. Phase 3 — Shape evaluation

Evaluation order per shape, against a focus node:
kind check, then closed/ignoredProperties, then fields (primitive
check, or recurse as the field's own focus node, per the granularity
table in `shaxon-v0.3.1-core.md` section 10 exactly), then
required/requiredIds, then qualified
counting, then combinators (mandatory short-circuit per Core section 4),
then check (a `$compute` island with `local.focus` bound to the focus
node).

A `shapeDepth` counter runs alongside `Machine.Step()` the same way
`steps` does, charged on every recursive descent (through `extends` or a
self-referencing field), raising `SHAPE_DEPTH_EXCEEDED` against
`limits.maxShapeDepth`.

Violation records are built exactly per `shaxon-v0.3.1-core.md` section
10's granularity table: one violation per absent required member (using
`requiredIds` for `constraintId` when declared), per failing field, per
failing `check`, per failing combinator (never one per failing or
unevaluated alternative inside a combinator), and per repeating element
under a `unique` declaration; a nested shape gets its own focus node and
its own violation, never a `constraintPath` on the parent. Every record
carries the mechanically-derived `kind` (`"structural"` iff `code` is
non-`null`), never author-set, and omits `constraintPath` entirely
(not `null`) where a finding isn't attributable to a specific member —
there is no `value` member; the offending value is always derivable from
`focusPath`/`constraintPath` against the original input.

## 7. Phase 4 — Indices, relations, targets, the `validate` pipeline

Index build/rebuild reuses the same snapshot-iteration primitive `for`
already has (one step per element visited), and checks the mutation log
from Phase 1 before rebuilding, skipping the charge when nothing touched
the index's declared `source` or a prefix of it. `relations` compiles to
one or two `indices` entries plus the section 4c redundancy check.
`reference` and `$inverse` are lookups against the built indices;
`$inverse` yields paths, so its result composes with any other
path-expression form. `local.step` inside a bounded closure must resolve
to a single scalar segment (string or non-negative integer) — anything
else is `TYPE_ERROR` — and counts as exactly one hop toward `maxDepth`.

Target forms (`$path`, `$each`, `$discriminator`, `$indexed`,
`$path*`/`$path+`) each resolve to an ordered list of focus paths; the
bounded-closure forms share the same depth-counter mechanism as shape
recursion. `$indexed` puts each source element's own path in that role —
never the index's key or value — visited in ascending key order for a
plain index, or grouped by key in ascending key order and then by array
index within each group for a `multi` index.

A `validate`/`check` entry carries exactly one of `shape` or `unique`
(both present is `SHAPE_ERROR`, checked in Phase 2). `unique` is a
distinct evaluation path from shape checking, not a shape variant: it
walks the target's population in visiting order comparing a declared
`field` (or, for non-object elements, the element itself), flags every
repeat of an already-seen key at the declared `severity`, and — in
`gate` mode — aborts on the first repeating element in visiting order,
the same "first offender" rule `reference`'s gate mode already uses
(section 5). It shares `check`'s optional `"id"` member for
`constraintId` but not `check`'s `$compute` machinery.

The `validate` list and the `check` instruction share one evaluation
entry point for the shape-checking path, since Core section 8 states
their mode semantics are identical — `check` is exactly the
`InstructionDef` from Phase 1, and calls into the same function the
pipeline phase uses.

## 8. Phase 5 — `shaxon.Run`, version handling, error pipeline order

Pipeline order exactly per Core section 9: parse, version, schemas and
shapes and indices and relations and computes (static — a malformed
`computes` entry, or a `check` referencing an undeclared compute name, is
`SHAPE_ERROR` at this stage, not at first use), program validation, input
schema/gate-shapes, execution (including any `check` instructions),
output schema/gate-shapes. Validation purity (Core section 0/1: `check`
and `validate` never write to `state`, `output`, or any root) needs no
enforcement code of its own — it holds by construction, since a `check`
or `unique` evaluation only ever calls into the read-only shape/target
machinery and the closed `$compute` operand, never the mutation
instructions — but it is worth a dedicated Phase 6 fixture asserting it,
rather than leaving it an implicit property nobody tests.

**Explicit scope decision.** The spec requires a `"3.1"` runtime to also
correctly execute packages declaring `"1.0"` through `"3.0"` — five
retired `shaxon` version strings. This plan implements `"3.1"` only for a
first version (it is additive over `"3.0"` by `shaxon-v0.3.1-core.md`'s own
framing) and returns `VERSION_ERROR` for the older strings. No package anywhere
declares them yet. Revisit if that changes.

## 9. Phase 6 — Fixtures and conformance

The regression net stays the 28 Jaxson fixtures, run against the
refactored library unchanged. A new `shaxon-v0.3.1-fixtures.json` is
written from scratch — there are currently zero fixtures at v0.3.0 or
v0.3.1 — prioritised exactly where `shaxon-v0.3.1-core.md` itself flags
the risk: the granularity split (required vs. field vs. nested-shape vs.
combinator vs. `unique`-repeat, section 10), `qualified`'s step charge,
`unique`'s population-ordering and gate/report rule, and the validation-
purity property noted in Phase 5.

The existing `attic/shaxon-v0.1.1-fixtures.json` and
`attic/shaxon-v0.1.2-fixtures.json` (three fixtures each) target version
strings out of scope per Phase 5 — mine them for test cases rewritten to
declare `"shaxon": "3.1"`, rather than trying to run them as-is.

The Navy Wars example, replayed at a fixed seed, is a deterministic
multi-turn transcript already captured in `example-game-seed0.txt`. Once
the library split lands, this is a natural integration-level regression
test, distinct from and complementary to the unit-level fixtures.

## 10. Phase 7 — Public API and supporting tooling

### 7.1 Core API surface

`jaxson.Run` and `shaxon.Run` accept either a decoded package or raw
bytes; both return a documented `Err` type. A `shaxon.Validate(pkg,
target)` convenience entry point covers the common case of validating a
document without touching `program` or compute. Numbers are exposed at
the API boundary as a small `jaxson.Number` type (`String()`, and
whatever else callers need), even though the internal representation
stays `*big.Rat`. Since both libraries are meant for reuse by other
applications, the instruction and operator tables must be
immutable-after-construction (or built fresh per call), so concurrent
`Run` calls are race-safe — a constraint worth holding from Phase 1
onward rather than retrofitting later.

### 7.2 Session helper

Finding from the Navy Wars example, not from the spec: a host that wants
a multi-turn Jaxson program has to thread state through `input`/`output`
by hand — clone the package, splice in the carried-forward state, call
`Run`, pull the new state back out. `play.go` does this once, correctly,
inside a closure. It is a mechanical pattern that every stateful embedder
will otherwise reinvent slightly differently. A small `Session` type
(wraps a package and a current state value; exposes `Session.Step(input)
(output, error)`) belongs in the library. This does not change the
execution model — Jaxson's no-ambient-state guarantee is exactly why the
host has to do this threading at all — it only stops every caller from
rewriting the same dozen lines. Worth documenting alongside it: the Navy
Wars pattern of re-validating the entire carried-forward state against
`inputSchema` on every turn is the right general idiom for stateful
embedding (a tampered or corrupted session is caught as `INPUT_ERROR`
before the program runs), and should be written up as the recommended
approach, not left implicit in one example.

### 7.3 Builder/compiler package

Second finding from the Navy Wars example: `build_navywars.py` is not
game logic. Its helper functions (`P` for a path, `V` for a `$v`
reference, `C` for a compute island, `SET`/`APPEND`/`IF`/`FOR`/`ASSERT`
for instructions) are general-purpose combinators for constructing
Jaxson programs, used once, in one script, because writing instruction
JSON by hand at any real size is impractical — the core design document's
own section 11 anticipated exactly this need under the heading "a
non-normative surface syntax." This plan promotes that DSL to a real,
tested part of the system: a `jaxson/build` Go package exposing the same
primitives as constructors, so future examples, fixtures, and
Shaxon package authors do not each reinvent it. This is packaging
existing, already-proven work, not new design.

## 11. Decisions this plan makes explicitly

Recorded here so they are visible choices, not silent ones:

| Decision | Where | Reason |
|---|---|---|
| Implement only `shaxon: "3.1"`, not the full 1.0-3.0 version matrix | Phase 5 | No packages exist yet at any version; implementing five never-used dialects first is the kind of premature completeness the project's own reviews warned against |
| Add a mutation-path log via an `OnMutate` hook rather than assuming one exists | Phase 1 | The index-reuse rule's precondition is not actually met by the current interpreter; this is the minimal fix |
| Instruction/operator tables must be immutable-after-construction | Phase 1 | Reuse by other applications implies concurrent use; cheaper to guarantee now than retrofit |
| Add a `build` combinator package and a `Session` helper | Phase 7 | Both are formalisations of code that already had to be written once, ad hoc, in the Navy Wars example; the alternative is every future embedder rewriting them |

## 12. Explicitly out of scope for this plan

- The `shaxon-v0.3.0-orientation.md` category-2 items the v0.3.1 proposal
  itself defers (per-field severity, composable path expressions,
  `$conforms` structural targeting, a named-profile pattern operator,
  cross-package imports) are not addressed here. This plan implements
  v0.3.1 as specified; it does not extend the spec.
- A further review series' proposed "bounded compositional
  expressions" layer, and a graph-transformation formalism (Delta-G), are separate, larger design questions and are not part of
  this implementation plan. Building the v0.3.1 reference interpreter is
  a prerequisite for evaluating either seriously, not a step toward
  either.

## 13. Sequencing summary

| Phase | Depends on | Produces |
|---|---|---|
| 0 | — | `jaxson` as an importable library, behaviour-identical, 28 fixtures as a `go test` |
| 1 | 0 | Instruction/operator tables, mutation hook, exported primitives |
| 2 | 1 | Shaxon registries and load-time `SHAPE_ERROR` checks |
| 3 | 2 | Shape evaluation engine |
| 4 | 1, 3 | Indices, relations, targets, the `validate` pipeline |
| 5 | 2-4 | `shaxon.Run`, full pipeline, version gate |
| 6 | 5 | New v0.3.1 fixture set; Navy Wars as an integration test |
| 7 | 0 (7.2, 7.3 can start once Phase 0 lands) | Public API polish, `Session`, `build` package |

Copyright (c) 2026 haitch. Licensed under the GNU General Public License, version 3: https://www.gnu.org/licenses/gpl-3.0.html
