# Changelog

## Unreleased

### Added
- SHACL comparison harness, all five examples and in batches: `scale.py` now
  generates trails for `four-eyes-release`, `delegation-chain` and `break-glass`
  as well as `rolling-quota` and `chinese-wall`, takes `--example`, `--budget`,
  `--json`, `--repeat` and `--engine-runs`, raises `limits.steps` in the copy of
  every package it runs, and exits 2 (not 1) on a harness failure, 1 only when
  engines disagree. A per-cell time budget projects the next size from the
  growth seen so far and skips it, and every larger one, if it would be over;
  slow cells get fewer repeats. New: `bench.sh` (the run as ten resumable
  batches under `results/`), `report.py` (merges the JSON lines into
  `results/REPORT.md`, with Shaxon's speed relative to each engine),
  `setup-mac.sh` and `env-mac.sh` (Homebrew: go, openjdk, maven; asks before
  installing; not tested on a Mac, only against a stub `brew`),
  `enginetime -cpuprofile`. `pom.xml` had `--` inside an XML comment, which
  current Maven rejects; reworded. Finding: Shaxon is quadratic in wall time on
  `four-eyes-release` and `break-glass` (TRACKER PF3).
- Phase 7.1, the core API surface: `shaxon.Validate(pkg, shape, doc)` and
  `ValidateJSON` check one document against a declared shape without running
  the package's program; `jaxson.Number`, an immutable exact decimal for the API
  boundary (results still hold `*big.Rat`, `AsNumber` converts); `jaxson.RunJSON`
  and `Profile.RunJSON`; the `Err` type documented, and its text no longer shows
  an empty code as `CAT/: msg`. A race test runs `Run` and `Validate` on one
  package from eight goroutines. The signature of `Validate` and the scope of
  `Number` are decisions of ours, recorded as A1-A4 and N1-N2 in TRACKER.
- SHACL comparison harness: a second timing profile, engine-only, beside the
  end-to-end one (`examples/shacl/authz`: `scale.py --profile`, `./run.sh
  timing-engine` and `timing-end-to-end`, `enginetime/main.go`,
  `JenaTime.java`). It leaves the RDF lift, process start and parsing out of
  the count and prints the lift as its own column. Result: with the lift out,
  warmed-up Jena validated 4000-event trails faster than Shaxon (3.5-5.2 ms
  against 11-17 ms) and barely grew to 20000 events; pyshacl stayed about
  20-26 times slower than Shaxon. READMEs and TRACKER state both profiles.
- The `aggregate` instruction (core section 8a): `count`, `sum`, `min` and
  `max` over an array, with an optional `where`, written to an object at
  `into`. It is sugar, expanded into `set`, `for`, `if` and `$compute` before
  the program is checked (`pkg/shaxon/aggregate.go`), so it adds no evaluation
  rule and costs exactly its expansion. Malformed forms are `PROGRAM_ERROR`.
  Tests in `aggregate_test.go`; 22 fixtures (`aggregate-*`), two of which pin
  the same step total as the hand-written fold; 9 more mutants in
  `fixtures_mutants.py`. `Run` now works on a shallow copy of the package, so
  the caller's `program` is not rewritten.
- Phase 6, conformance fixtures. `pkg/shaxon/shaxon-v0.3.1-fixtures.json`
  (161 fixtures, 70 of them expecting an error) with its runner
  `fixtures_test.go` (`TestFixtures`) and `pkg/shaxon/FIXTURES.md` (format,
  coverage by core section, cases left out and why). Written from the core
  specification: the violation-granularity table, minimal combinator
  evaluation, gate and report ordering, `unique`, `qualified` (bounds and
  step charge), references, indices (build charge, reuse, rebuild), relations,
  `$indexed` and `$inverse` order, path closures and their horizon,
  `maxShapeDepth`, the `extends` merge rule, version and pipeline order, the
  `check` instruction and report delivery. Six fixtures are mined from
  `attic/` and rewritten for 3.1. The file spells Shaxon's identifiers with the
  `SHAX_` prefix, as the core now does.
- `pkg/shaxon/fixtures_mutants.py`: breaks 43 engine rules one at a time in a
  scratch copy and requires the fixtures to fail. 41 are killed; the two
  survivors (a cost the core does not fix, and an equivalent mutant) are
  recorded in the script. Not part of the default test run.
- TRACKER: Phase 6 section, spec findings S1 to S8 (S3, whether validation is
  pure, is analysed there; ruled: the core says nothing about it), and a
  dormant-guards row for the mutation check.

- `pkg/shaxon`: shape evaluation (plan phase 3). `Evaluator` checks a focus
  node against a named shape and returns a `Report` of `Violation` records
  built per core section 10's granularity table, in report mode or gate
  mode. Covers kind and primitive keywords, `required`/`requiredIds`,
  `fields`, `items`, `closed`/`ignoredProperties`, `qualified`,
  `and`/`or`/`xone`/`not` with the mandatory minimal evaluation, `check`
  (inline or by `computes` name), severity, `limits.maxShapeDepth`, and step
  charging through the shared `Machine.Step()`. Files: `shapes.go`,
  `report.go`, `shapes_test.go` (19 tests).
- `shapes` entries accept an `"id"` member, used as the finding's
  `constraintId`. Core section 10 names it; the parser rejected it.
- `KeywordDecl.MinRat`/`MaxRat`: the exact `*big.Rat` number bounds. The
  existing `float64` fields are inexact for decimals and are no longer
  consulted when judging data.

- `pkg/jaxson/cost.go`: the step-cost model's mechanism (design note
  `docs/proposals/step-cost-model.md`, steps CM-1 and CM-2). `CostTable`,
  `Cost` (`base + floor(per*size/div)`, saturating), `UnitTable()`,
  `Machine.Charge`/`Steps`/`SetCostTable`/`ChargeEvent`/`EventCost`, and
  `ResolveCostTable` for a dialect that owns `limits.costTable`. The only
  table is `unit`, under which every event costs one step, so **no step count
  changes**; this is pinned by `testdata/unit-step-counts.json` (57 cases,
  generated before the change) and `TestUnitStepCountsFrozen`.
- `pkg/shaxon/events.go`: `EventShapeActivation`; `chargeShape` now goes
  through the cost table.
- `cost_test.go` (11 tests) and `TestShapeActivationFollowsTheCostTable`.
- `pkg/shaxon`: indices, relations, targets, the `validate` pipeline and the
  `check` instruction (plan phase 4, core sections 3-9).
  - `indices.go`: `IndexSet`, building and reusing indices and relations
    under the mutation log; `index.element` cost events; the one-to-one
    check; `Lookup`, `Refresh`, `Elements`, `Inverse`.
  - `targets.go`: `$path`, `$each`, `$discriminator`, `$indexed` and closure
    targets; `target.resolve` cost events.
  - `paths.go`: bounded closures `$path*` / `$path+`, with the `maxDepth`
    horizon and `PATH_DEPTH_EXCEEDED`; `closure.hop` cost events.
  - `validate.go`, `check.go`: `ParseValidate`, `Runtime`, `unique`, report
    delivery, and the `check` instruction (`Runtime.Instructions()` is the
    core table plus `check`).
  - `forms.go`: the operand forms `$altPath`, `$inverse`, `$path*`, `$path+`.
  - 42 tests across `indices_test.go`, `validate_test.go`, `paths_test.go`
    and `forms_test.go`.
  - Spec gaps are decided in the file headers and indexed in TRACKER.md as
    P1-P6, T1-T5, V1-V7, C1-C6 and F1-F7. Four need a ruling: P1, V1, V3 and
    (from phase 3) G11.
- `shaxon.Run`, `shaxon.RunJSON` and `shaxon.Result` (plan phase 5): the
  Shaxon pipeline in the order of core section 9, as a `jaxson.Profile`
  (`shaxon`, `"3.1"`) plus hooks, not a second pipeline. Returns the output,
  the combined validation report (only when a report-mode entry or `check`
  without `into` ran) and the step total. `limits` may carry `steps`,
  `maxShapeDepth` and `costTable`; the cost table now reaches the machine.
  `validate` entries rooted wholly in `input` run before the program, the
  rest after it (decision R1, needs a ruling; R1-R7 in TRACKER.md). 24 tests.
- `examples/shaxon/authz/`: five authorisation packages where access follows
  a trail of earlier actions (`four-eyes-release`, `chinese-wall`,
  `delegation-chain`, `rolling-quota`, `break-glass`), their 51 cases, and a
  README; pinned by `TestAuthzExamples`.
- `examples/shaxon/authz/mutants.py`: breaks one rule at a time in a scratch copy
  of the module (23 mutants over the five packages) and requires
  `TestAuthzExamples` to fail; it first checks the untouched copy passes.
- `examples/shacl/authz/`: the five authorisation examples again as SHACL-SPARQL
  shapes, with a JSON-to-RDF lift, a harness that runs the Shaxon case files
  through pyshacl and Apache Jena (50 of 50 comparable cases agree on both; Jena
  names the shape, not the constraint, in `sh:sourceConstraint`), a 19-mutant
  check, `scale.py` (timing across Shaxon, pyshacl and Jena), `setup.sh`, `run.sh`
  and `pom.xml` to install and launch them, `SETUP.md` as the setup guide, and a
  README comparing the two. Outside the Go module; needs Python and pyshacl.
- `cmd/shaxonrun`: runs a Shaxon package from the command line
  (`shaxonrun [-pretty] [-steps] package.json [input.json]`), and
  `examples/shaxon/orders.json` to try it on.
- `jaxson.ParseJSON`: a strict reader. Duplicate object keys at any depth,
  non-JSON and trailing data are `PARSE_ERROR`, as the core design always
  required. 5 tests.
- `jaxson.Profile.SchemasOptional`: a dialect may omit `inputSchema` and
  `outputSchema` (Shaxon does, per core section 2); the core still requires
  both.
- `pkg/jaxson/forms.go`: an operand-form hook, a fifth extension point beyond
  the four plan section 4 listed. `FormDef{Check, Eval}`, `Checker.Forms`,
  `Machine.SetForms`/`HasForm`, `Hooks.Forms()`, and
  `CheckOperandWith`/`CheckPathWith`/`CheckProgramForms`. Nothing changes
  when no form is registered. Tested directly in `forms_test.go` (6 tests).

### Changed
- Performance, step 1 of 3 (no change to results or step counts). The machine no
  longer copies what cannot change under it: `input` is not cloned at the start
  of a run, a `$path` read of `input` or `local` is shared, `$lit`, `$compute`
  and plain literals are not cloned on evaluation, and `*big.Rat` values are
  shared by `Clone` (none is ever modified in place). `set`, `append` and
  `insert` copy what they store, so nothing in `state` or `output` aliases a
  shared value; a composite read from `state` or `output` is still copied, so a
  loop over it iterates a snapshot. `Machine.Eval` results are now read-only for
  a host (documented). Engine benchmarks (`bench_test.go`, 2 CPUs): rolling-quota
  at 20000 events 82 ms to 46 ms and half the allocations; chinese-wall 91 ms to
  78 ms. `aliasing_test.go` pins the contract (writes through a copy never reach
  the input, a loop variable or a literal).
- Performance, step 2 of 3 (no change to results or step counts): the program
  tree is compiled to closures the first time it runs (`pkg/jaxson/compile.go`),
  after the threaded-code approach in ual's `iual`. `$compute` bindings become
  slots and `$v` an index, operator costs are looked up once, static path
  segments are converted once, and operand arguments live in a scratch stack
  instead of a fresh slice per call. Eager operators now carry `OperatorDef.Fn`
  (over evaluated arguments; `Apply` is derived from it); core instructions
  carry an optional `InstructionDef.Compile`; `Compiler` is exported with
  `Operand`, `Path` and `Block`. Host forms and host instructions (Shaxon's
  `check`) are not compiled: they keep their own `Eval`/`Exec`, and an unknown
  operator falls back to the tree path. The old tree interpreter stays as an
  oracle only: `pkg/jaxson/tree_oracle.go`, header "SCHEDULED FOR DELETION" with
  the list of what goes with it, selectable with `jaxson.UseTreeInterpreter`.
  Oracle tests (`compile_oracle_test.go`, `shaxon/oracle_test.go`, in the default
  test run) compare compiled and tree runs on output, report, step total and
  error for every core and Shaxon fixture, every authorisation case, those
  re-run under tighter step limits (so the order of charging is compared, not
  only the total), and 12000 seeded random programs over every operator and
  core instruction. Breaking the compiler on purpose (path before value in
  `set`, `for` charged once, `with` order reversed, a changed charge) is caught
  each time.
- Performance, step 3 (no change to results or step counts, except as noted):
  `ParseJSON` is a hand-written scanner in place of the `encoding/json` token
  reader (22 to 56 MB/s on a 920 KB trail, a third of the allocations). Numbers of
  up to 18 significant digits and no exponent become a `*big.Rat` directly, in
  lowest terms without a gcd; any other number goes through `big.Rat.SetString` as
  before, so decimals stay exact. Object keys are interned per document. Nesting
  beyond 10000 levels is a `PARSE_ERROR` (encoding/json's limit). It accepts and
  rejects exactly what the old reader did and builds the same values:
  `parse_reference_test.go` keeps the old reader, and `parse_diff_test.go`
  compares the two on the fixtures, edge cases, 80000 generated and mutated
  documents, and `FuzzParseJSON`.
- Performance, step 4: `append` grows the array in place (it copied the whole
  array each time: 20000 appends took 2.1 s and 4 GB, now 4 ms); safe because no
  array in `state` or `output` is shared (`TestAppendDoesNotAlias`). Path
  segments built at run time use the scratch stack and are not re-boxed;
  `concat` of two strings allocates once; index builds skip the per-element
  focus-path slice (it is built only if a closure form asks, `Ambient.Path`),
  key maps use a struct key instead of a prefixed string, and entries sort with
  `slices.SortFunc`. Engine benchmarks, same binary, 2 CPUs, original baseline in
  brackets: rolling-quota 4000 events 1.1 ms (16.5), 20000 events 5.2 ms (82);
  chinese-wall 4000 events 6.2 ms (22.3), 20000 events 28 ms (91). On the tree
  oracle in the same binary (which has the copy-avoidance and the other changes
  above): 13 and 45 ms, 17 and 75 ms. `SHAXON_BENCH_TREE=1` runs the benchmarks on it.
- Performance, step 5: `local.*` bindings are a small stack scanned from the top
  instead of a map, a static `local` read compiles to a direct lookup, and a loop
  index uses shared small numbers (no allocation). Rolling-quota at 20000 events
  5.2 to 4.6 ms, chinese-wall 28 to 23.5 ms. `RunProgram`, `Eval` and
  `RunCompute` pop any bindings left by a failed evaluation, as they already did
  for the scratch stack.
- Performance, step 6: a JSON object is held as a `*jaxson.Object` instead of a
  `map[string]any`: a short list of (key, value) pairs, searched linearly, with a
  hash index only above 12 members. Keys are `unique.Handle[string]` values
  (`jaxson.Key`), so comparing two keys is one pointer comparison, and a path
  that the program fixes in advance is turned into keys once, at compile time.
  The public edges still use maps: `Run` takes its input as maps (or as Objects)
  and returns its output as maps (`Legacy`), `ParseJSON` returns maps, and
  `Data`/`Legacy` convert. New: `ParseData` (a document straight into Objects)
  and `ParsePackage` (a package as maps, its `input` member as Objects), which
  `shaxon.RunJSON`, `shaxonrun` and the benchmarks use, so that the conversion
  Run would otherwise make is not paid. A `nil *Object` reads as empty. Measured
  with the program compiled in a warm-up run and left out of the timing:
  rolling-quota 4000 events 1.0 to 0.7 ms and 20000 events 4.6 to 2.9 ms (best
  of runs 2.4); chinese-wall 4000 events about 6 to 5.2 ms and 20000 events 23.5
  to 22 ms (index building, not member access, dominates there); `ParseData` of
  the 920 KB trail 12.7 ms against 13.7 for `ParseJSON`, with a third less
  memory. Handing `Run` an already-parsed map input now costs a conversion (about
  60 ns per new key): the same benchmark with map input takes 6.3 ms, not 2.9, at
  20000 rolling-quota events. Creating members under keys the program only
  learns at run time is slower than with a Go map (about 160 ns per new member in
  `BenchmarkObjectSetDynamic`, growth included, of which `unique.Make` is about
  60; a map insert measured about 10 ns in the earlier microbenchmark); reading
  a fixed member is 3 ns. No result or step count
  changed: the oracle comparison, the 161 fixtures and the authorisation cases
  pass unchanged. The parser's output and `ParseData`'s agree on every input the
  differential tests use (`checkDataModes`).
- The SHACL comparison was re-run after these changes (scratch pyshacl and Jena
  installs, `./run.sh timing-end-to-end` and `timing-engine`, with 20000-event
  rows added to the second): Shaxon at 4000 events takes 5-12 ms end to end
  (was 28) and 0.9-5.3 ms validating (was 11-17). Rolling-quota is now faster
  than warmed-up Jena at every size; Jena still wins chinese-wall from between
  1000 and 4000 events. The READMEs in `examples/shacl/authz` and
  `examples/shaxon/authz` and TRACKER carry the new tables.
- Behaviour that was undefined is now fixed: the members of a `$tpl` object are
  evaluated in sorted key order. The old interpreter ranged over the Go map, so
  when one member failed, which error came out and how many steps had been
  charged by then changed from run to run. A run that succeeds is unaffected.
  Recorded as S9 in TRACKER (the core does not say).
- Core specification: the "validation purity" invariant (section 0, now "the
  three invariants") and the section 1 paragraph "Validation is a pure
  predicate" are removed, with nothing in their place: sections 7 and 8 give
  `validate` and `check` an `into` that writes, and the core now makes no claim
  about validation and the data it validates. The editorial, limitations and
  v0.3.2 walk texts that cited purity follow. One fixture was renamed
  (`check-in-report-mode-without-into-never-aborts-and-leaves-state-alone`) and
  one test (`TestValidationWithoutIntoLeavesRootsAlone`).
- `examples/shaxon/authz/rolling-quota.json` uses `aggregate` in place of its
  hand-written fold: same decisions, same step counts, 318 fewer characters
  (the SHACL comparison's size table is updated).
- Error identifiers that Shaxon introduces are now prefixed `SHAX_`, not `SHA_`
  (`SHAX_SHAPE_ERROR`, `SHAX_VALIDATION_ERROR`, `SHAX_SHAPE_MISMATCH`,
  `SHAX_DANGLING_REFERENCE`, `SHAX_SHAPE_DEPTH_EXCEEDED`,
  `SHAX_PATH_DEPTH_EXCEEDED`), because `SHA_` reads as the SHA hash family. The
  core specification now specifies the prefix. Code, tests, the authorisation
  cases and the notes follow. **A package or host that matched the old spelling
  must change.**
- `extends`: `closed` now combines as the core says, "most restrictive of
  parent/child wins" (closed if either stated value is closed, unless
  `closedOverride`). Before, a boolean AND of the two flags let the most
  permissive win. A child that does not write `closed` inherits the parent's.
  **A shape extending a closed parent with `closed: false`, or an open parent
  with `closed: true`, now behaves differently.**
- Core specification (`shaxon-v0.3.1-core.md`): the identifier prefix (section
  9); the `extends` merge table uses the implementation's override spellings
  (`closedOverride`, `andOverride`, ..., per-item `override`) and states the
  `requiredIds` collision rule; section 2 no longer requires a 3.1 runtime to
  execute packages declaring earlier versions.
- `go.mod`: `go 1.25` (was `go 1.22`). Verified on Go 1.27.1.
- `limits.steps` must now be no larger than 2^53-1 (was: anything that fit in
  an `int64`); a larger value is `VERSION_ERROR`, with the message "steps must
  be a positive integer no larger than 2^53-1". Step counters are `int64`.
- `Machine.Charge` does not advance the total when a charge fails; the old
  `Step()` incremented first. No program can observe this under `unit`.
- `jaxson.Hooks` has a new method, `Forms()`. An implementation outside this
  module must add it (nil is fine); `NoHooks` has it.
- A report renders a shape-less violation (a `unique` violation) with
  `"shape": null` instead of an empty string.

- `cmd/jaxrun` reads packages with `jaxson.ParseJSON`, so a package with a
  duplicate key is now a `PARSE_ERROR` instead of silently keeping the last
  value. Not a change to `jaxson.Run`, which takes an already-decoded value.

### Notes
- The Phase 3 `IndexResolver` stopgap is closed (see Phase 4 above and
  TRACKER.md, "Closed in Phase 4"). `reference`-kind shapes now resolve
  against real indices; `SHAX_NO_INDEX_RESOLVER` no longer exists.
- The spec leaves several points open (notably the shape-evaluation step
  cost); eleven decisions are recorded as G1-G11 in TRACKER.md and in the
  `shapes.go` header.
- `docs/proposals/step-cost-model.md` (accepted 2026-10-04): the weighted
  step-cost design. CM-1 and CM-2 are in; the `weighted-1` table, value-copy
  charging (CM-3) and host functions (CM-4) are not started and nothing
  depends on them.
- Phase 2 (`pkg/shaxon` registries and parser) has no entry in this file;
  TRACKER.md is its record.

## 0.3.1 - 2026-09-27

### Added
- `examples/navywars`: Navy Wars written with `jaxson/build`, the Go port of
  `examples/game/build_navywars.py`, plus `examples/navywars/cmd` to write
  it out as JSON (`go run ./examples/navywars/cmd navywars.json`).

### Verified
- The Go-built package is JSON-equal to `examples/game/navywars.json`
  (the acceptance test for the builder, `TestPortMatchesPythonGeneratedPackage`).
- `jaxplay -carry game -turns turns-seed0.json -format json` on the Go-built
  package and on the original produces byte-identical transcripts
  (1480 lines, reaching `status: "won"` on turn 8).
- The first attempt differed in one place only: a compute variable named
  `l` where the original used `L`. Fixed.
- `go build`, `go vet`, `gofmt -l`, `go test -race`: clean.

### Notes
- `build_navywars.py` is kept as the independent oracle the port is checked
  against; it is no longer the source of truth for new work.

## 0.3.0 - 2026-09-27

### Added
- `pkg/jaxson/build`: a fluent, chained, type-checked builder for Jaxson
  packages (plan 7.3), replacing the Python script as the way to write
  programs. Import as `jb`. Operands, expressions, instructions and schemas
  are separate sealed types; `If` without `Then`, `For` without `As`/`Do`,
  an operator below its minimum arity, a path inside an expression, and an
  unknown operator or rounding mode are compile errors. An unbound `Var`,
  a write to `input`, a shadowed loop name, a `$`-form inside `Data`, and a
  sample input that violates its schema are reported on assembly
  (`Package.Check`/`Map`/`JSON`/`Run`) by the runtime's own static checker.
  Builder values are immutable. `Template.Opt` takes a `Path`, matching the
  runtime's `$opt`.
- Tests: the `sum-of-prices` fixture rebuilt with the builder and compared to
  the fixture document; every operator constructor accepted by the runtime;
  assembly rejections; immutability; a runnable `Example`.

### Not yet done
- Port of Navy Wars to the builder, as the acceptance test against
  `examples/game/navywars.json`. `build_navywars.py` remains the source of
  truth until then.

## 0.2.0 - 2026-09-27

Phase 1 closed and verified. Implementation version; not the language
version (jaxson "1.0"; Shaxon v0.3.1 is specified but not yet implemented).

### Added
- `jaxson.CheckSchema` and `jaxson.Validate`: exported wrappers for the
  contract-schema check and value validation stages, which a host package's
  own `Run` must repeat.
- `jaxson.CheckProgramHost`: `CheckProgram` with the `Checker.Host` slot
  populated. Previously no exported path could set `Host`.
- `pkg/jaxson/extension_test.go`: nine tests, written from outside the
  package, covering host instruction registration, `Checker.Host`
  delivery, shared step accounting, `OnMutate` for all four mutation
  instructions, fresh-per-call tables, and concurrent `Run` under `-race`.
- `LICENSE`, `README.md`, `VERSION`, `pkg/version`.

### Changed
- `Run` moved from `machine.go` to its own `run.go`. No behaviour change.
- `TRACKER.md` brought up to date; implementation plan filename corrected
  in comments and the tracker.

### Verified
- `go build`, `go vet`, `gofmt -l`, `go test -race`: clean on Go 1.22.2.
- 28/28 Jaxson fixtures; 8/8 `jaxtools` tests; 9/9 extension tests.
- Navy Wars, seed 0, all 8 turns through `jaxplay`, reaches `won`.
