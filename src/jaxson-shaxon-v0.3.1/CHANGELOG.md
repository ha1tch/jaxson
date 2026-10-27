# Changelog

## 0.3.1-p000.r021 - 2026-10-27

### Added
- `examples/accessgraph/FINDINGS.md`, two sections and two summary rows: the existing declarative
  languages (Soufflé, Starlark, CEL, Rego, Cedar, SpiceDB and OpenFGA, SQL) against the three kinds
  of test, rules, validation and contract, and Shaxon against SHACL and others for the validation
  role, from the authorization examples already in the tree. The finding: no existing language covers
  the rules, the validation and the contract together, and only the SHACL comparison is measured.

## 0.3.1-p000.r020 - 2026-10-27

### Added
- `examples/accessgraph/engines/starlark/` and `run_starlark.py`: the access-decision rules in
  Starlark, run through starlark-go. It agrees with the reference on generated databases and
  on both Amazon variants. `engines/cel/`: a probe of CEL's static cost estimate on a
  membership closure. `FINDINGS.md` has a new section 5 with both results: Starlark's step
  budget is enforced and deterministic but not derived before the run, and it does not refuse
  data that is too deep; CEL's estimate is a sound bound that grows 87 times for every round
  of a closure. Starlark is also in the timing sweep and the Amazon runner.

## 0.3.1-p000.r019 - 2026-10-27

### Changed
- The accessgraph cost model is rebuilt (`cost.py`, `cost_model.json`, `calibrate.py`): its
  features follow the program's loops (the held table over every user, each delegation hop
  over its delegator's held entries, each request's scan), fitted on 73 synthetic databases,
  18 of them stretching one dimension at a time because generated databases alone cannot
  identify the terms. Held out: 28 synthetic databases, including 5,000 users and one with
  the Amazon data's proportions, none over their limit. The two Amazon databases are covered:
  the program runs inside its derived limit on both, using 48%; the model over-predicts there
  by 1.68 times. They were looked at after each of four fits and decided between versions, so
  they are validation data, not an untouched test. `test_cost.py` checks them when present
  and `check_amazon.py` saves their step counts. Closes T-41 (docs/RESOLVED.md).

## 0.3.1-p000.r018 - 2026-10-27

### Added
- `examples/accessgraph/agogo/` (the Go ports of the reference, their tests and a cross-check
  against `oracle.py`) and `examples/accessgraph/engines/` (the Soufflé, CozoDB and retired Go
  Datalog encodings, the timing sweep, the fault-planting comparison and the runner for the
  Amazon variants), moved in from the loose archives so that the numbers in `FINDINGS.md` can
  be reproduced. About 2,000 lines. Paths are relative or set by environment variables, built
  binaries are ignored by git, and each directory has a README.

## 0.3.1-p000.r017 - 2026-10-27

### Added
- accessgraph on real-shaped data (`provision/amazon_to_db.py`, `provision/check_amazon.py`):
  the Amazon Access Samples as two databases of 36,063 users and 430,188 grants, flat and
  nested (270 covering edges, 10,134 implied memberships dropped, true group depth 5), with
  a record of everything added to the data. Seven evaluators agree with the reference on
  10,000 decisions on both variants; the integrity gate passes both. The results are in
  `FINDINGS.md`, section 5. Closes T-39 (docs/RESOLVED.md).

### Changed
- The derived step limit did not hold on this data: the program needed 2.21 times the
  prediction and stopped at its declared limit, as it should when exceeded. The README and
  `FINDINGS.md` now say so, and T-41 is filed for the model; T-42 for exposing the step
  count in `jaxrun`, which the measurement needed a scratch build for.

## 0.3.1-p000.r016 - 2026-10-27

### Changed
- Register: T-36 is closed with a scope change (the Amazon-scale case moves to T-39, since
  only T-39's adapter can produce that data; the other engines at the new presets are
  recorded as run) and T-32 is closed on a decision (stage 2 is not built; the program's
  own assertion is the always-on backstop and the gate's depth check stays opt-in). T-39
  takes the Amazon-scale run and the measurement of the gate's depth cost at that scale.
  Both closures are in docs/RESOLVED.md. README and the two reconciliation banners follow.

## 0.3.1-p000.r015 - 2026-10-27

### Added
- The accessgraph declared step limit is derived from the data before the run
  (`cost.py`, `cost_model.json`, `calibrate.py`, `test_cost.py`): features from counts and
  closure sizes, fitted on 55 databases up to 1,000 users with a margin of 1.20, and
  checked on 27 databases the fit never saw, including 5,000 users, none over their limit.
  A limit exceeded stops both runtimes with `RESOURCE_ERROR/STEPS`; the model refuses to
  run when the generated program has changed. `scale.py` and `check.py` no longer raise
  the limit and report measured against declared steps. Closes T-38 (docs/RESOLVED.md).

### Changed
- `gen.build` derives the limit by default (it was a fixed 5,000,000, which 5,000 users
  exceed). `calibrate.py` needs numpy; nothing else does.

## 0.3.1-p000.r014 - 2026-10-27

### Added
- `examples/accessgraph/FINDINGS.md`: what the work has and has not shown. Speed is stated
  as not the fastest (slower than compiled Soufflé, about 5 times at 12 users growing to
  about 10 times at 5,000, and slower than plain Python and Go implementations; faster
  than CozoDB from 60 users up, about 2 to 5 times), with a table measured at five sizes
  on the current program; the cost of the fixed limits; and what agreement between
  evaluators shows.

### Changed
- The README no longer carries its "What it does not show" paragraph; it points to
  `FINDINGS.md`.

## 0.3.1-p000.r013 - 2026-10-27

### Added
- The id width is policy (`policy.ID_WIDTH`, six digits): grant and delegation ids are
  `G000001` and `D000001`, so a database can hold up to 999,999 of each, where four
  digits stopped at 9,999. A database with a grant or delegation id of another width is
  refused by the reference, the generator, the builder and the integrity gate, so a
  mixed-width database is rejected and not mis-ordered; the Go oracles enforce it too.
  `test_policy.py` answers a database of 10,068 grants identically on the reference and
  both runtimes. The integrity gate has two more rules (24), checked against 732 planted
  defects. Closes T-37 (docs/RESOLVED.md).

### Changed
- The Go Datalog engine's encoding is retired at six-digit ids: it gave wrong answers, and
  four six-digit segments do not fit 64 bits. Soufflé and CozoDB still agree with the
  reference. The accessgraph and root READMEs say so.

## 0.3.1-p000.r012 - 2026-10-27

### Changed
- The accessgraph README is brought up to date: what it is and why it matters (the data,
  the program and the gate are JSON under one contract; a cost limit on its own is not new,
  since CEL and Starlark have one), the declared limits, nine mutants, new step and time
  tables (10,533 to 10,229,845 steps), a fresh comparison of all nine evaluators at 5,000
  users, and the known limits after T-31. The root README lists the declared limits, the
  integrity gate and the SHACL Core result.
- Checked again on the current program before the README repeated them: identical step
  counts on the stand-in and the real runner for nine databases from 12 to 5,000 users;
  the reference against a naive fixpoint on 200 databases (28,000 decisions); the
  reference, stand-in and real runner on 150 databases (31,320 decisions, 0 mismatches).

## 0.3.1-p000.r011 - 2026-10-27

### Changed
- Documentation only. The register no longer names the closed T-31 as a blocker of
  T-32, T-36, T-37 and T-38; T-38 records CEL and Starlark as prior art for a cost
  declared before the run and enforced. The related-work proposal and the accessgraph part 1
  status report carry dated reconciliation banners; their content is unchanged.
- The accessgraph and root READMEs are deliberately not updated; they wait for the final
  README update.

## 0.3.1-p000.r010 - 2026-10-26

### Added
- `examples/accessgraph/policy.py` and `test_policy.py`: the example's limits (3 delegation
  hops, 16 group rounds, 8 role rounds, resource depth 6) are declared once, not measured per
  database. The generated program is the same for every database. The reference and the
  generator work within the limits, and the program asserts with one extra round per closure
  that its input fits, failing with `ASSERTION_FAILED` on both runtimes instead of answering
  from a truncated closure. Each limit is accepted exactly at its value and refused one past
  it. Closes T-31 (docs/RESOLVED.md).

### Changed
- Fixed rounds cost steps: 10,533 at 12 users (was 5,336) and 10,229,845 at 5,000 (was
  8,377,283). `mutants.py` now has nine mutants: round mutants silent (the answers must
  differ) and asserted (the program must refuse). `integrity.py` accepts `--limits policy`.
- The example README is not updated for this on purpose; it waits for the final README
  update.

## 0.3.1-p000.r009 - 2026-10-26

### Added
- `examples/accessgraph/integrity.py`, `integrity.json` and `test_integrity.py`: a
  data-integrity gate for accessgraph databases, written as a Shaxon package (T-32, stage 1).
  It reports dangling references, bad effects and repeated ids at the exact path of the
  offending element. 200 databases with 769 planted defects of 22 kinds give findings equal
  to the plan, removing any of its 22 rules makes the test fail, and it costs 2 to 4% of the
  inference program's steps. With depth limits passed in it also reports every user, role
  and resource deeper than the limit, equal to independently computed truth, at 30 to 50%
  of the program's steps.

### Changed
- T-32 is partial: the limits to enforce wait on T-31 and T-37. The accessgraph README
  and status report say so.

## 0.3.1-p000.r008 - 2026-10-26

### Added
- `examples/shacl/authz/chinese-wall.core.shapes.ttl`, `break-glass.core.shapes.ttl`,
  `core_cases.py` and `./run.sh core`: the two examples written in SHACL Core alone, with
  an audit that every `sh:` term is Core. Both are expressible. They agree with Shaxon on
  verdict and reasons on the 15 case files and on 4,100 random trails per package, and
  with the SHACL-SPARQL versions on 800 each. The one difference is that Shaxon refuses a
  request for an unknown company and both SHACL versions allow it. In pyshacl, Core
  validated the same trails faster than SHACL-SPARQL (break-glass at 4,000 events: 5.6 s
  against 45 s). Closes T-20 (docs/RESOLVED.md).

### Changed
- The authz README no longer says SHACL Core cannot compare values across nodes; it
  records the Core result. The accessgraph README and status report carry the same
  correction.
- `.venv` and `.jena` under `examples/shacl/authz` are ignored by git.
- T-22: the default `setup.sh` branch ran end to end and is partial, not closed.

## 0.3.1-p000.r007 - 2026-10-26

### Added
- `examples/accessgraph/`: an inference example in the authorization domain. It decides
  whether a user may act on a resource from nested groups, role inheritance, a resource
  tree, allow and deny grants and delegation chains of at most three links, and names
  the supporting grant. It has a seeded generator with five presets (to 5,000 users),
  a top-down reference, a naive fixpoint, the generated Jaxson program, a stand-in
  interpreter, a checker, a script that plants six faults, a scale record and a
  part 1 status report. The reference, the stand-in and the real runner agree on 150
  databases (31,320 decisions); all six faults are caught; the step count is 5,336
  at 12 users and 8,377,283 at 5,000, which exceeds the declared default of 5,000,000
  (T-38). Eight other evaluators agree with the reference, among them Soufflé,
  CozoDB and a Go Datalog library.
- `examples/accessgraph/provision/`: fetches and verifies two external datasets, the
  UCI Amazon Access Samples and the OpenFGA sample stores at a pinned commit, and reads
  the 4.8 GB access matrix from its compressed stream instead of extracting it. The
  data directory is ignored by git.
- Five accessgraph rows in the dormant guards table of the tracker.

### Changed
- The root README lists the example; the related-work proposal and two register
  statements no longer say the example is outside the tree.

## 0.3.1-p000.r006 - 2026-10-26

### Added
- Register items T-37 to T-40 (fixed-width ids, the declared step limit, a run on
  real-shaped data, third-party expected answers) and wave 10.

### Changed
- T-36 is widened to cover 5,000 users and the Amazon-scale case, and is partial.

## 0.3.1-p000.r005 - 2026-10-25

### Added
- `docs/proposals/accessgraph-related-work-and-roadmap.md`: a survey of comparable
  systems, where the accessgraph example is stronger and weaker, and the work ahead.

## 0.3.1-p000.r004 - 2026-10-25

### Added
- Register items T-31 to T-36 for the accessgraph example, grouped as waves 8 and 9.

## 0.3.1-p000.r003 - 2026-10-15

### Added
- `examples/medical/`: two Prolog medical knowledge systems re-implemented on Jaxson and
  Shaxon and compared with the originals' own output. The first coordinator's five cases
  are identical; a Shaxon check of its knowledge base finds 24 problems. For medical3
  0.10.0 a bottom-up Jaxson evaluator compiled from its vocabulary matches all 94
  assessments and 58 recommendation-tier rows, five knowledge-only overlays and the
  contract properties C4, C5 and C6, except one phantom conclusion that comes from a
  cycle defect in medical3 itself. The generated packages are included, so the Jaxson
  side runs without Prolog.

## 0.3.1-p000.r002 - 2026-10-15

### Changed
- Wave 7 confirmations (T-01, T-02): `Validate(pkg, shape, doc)` stands, and
  `Number` stays the Real-only view; a relaxed result is a `*big.Rat` or a
  `float64`, and `AsNumber` reports false for the latter. Recorded in the
  tracker, Phase 7.
- `AsNumber`'s documentation says a float64, which a relaxed run can return, is
  not a `Number`, and a test pins it.

## 0.3.1-p000.r001 - 2026-10-15

### Specification (severity)
- Core section 7 now says a `warning` or `info` finding, of a shape or of a
  `unique` entry, never aborts a `gate` (the implementation already behaved so;
  the text said "first violation"). Two fixtures (196 in all) and two mutants
  (71, 69 killed) pin it. `shaxon-v0.3.1-limitations.md` has a new section 8 on
  how `conforms` and severity differ from SHACL, and the SHACL coverage and
  comparison tables no longer call severity plainly "covered". The Layer 2 and
  3 proposal (`docs/proposals/extends-narrowing.md`) compares the two in full.

### Specification (numbers)
- The Jaxson core design (section 2) now defines a **Real** (at most 38 digits
  and at most 200 digits after the point) and two numeric modes. Strict, the
  engine default, rejects a JSON number that is not a Real as `PARSE_ERROR`
  without building a value for it. Relaxed is an optional engine switch: a
  number that does not fit is an IEEE 754 float64, and determinism across
  architectures is not guaranteed. `jaxson-intro.md`, `README.md`,
  `shaxon-v0.3.1-core.md` and the Walk proposal now say so (T-29). The
  implementation does not enforce the strict rule at parse time yet (T-25).

### Changed
- Strict numeric mode in the runtime and in the Python port (core design,
  section 2; T-25). A JSON number that is not a Real (more than 38 digits, or
  more than 200 digits after the point) is now a `PARSE_ERROR`, "number out of
  range", found from the literal's text before any value is built. A 4 MB
  document of `1e999999` is refused in under a millisecond (about 3.5 hours
  before, by extrapolation), and a 1 MB literal costs 2.6 ms and 1 MB, not
  1.3 s and 2.2 GB. The same rule covers `Normalize`, `ParseNumber` and the
  `to_number` operator (`NUMERIC_OVERFLOW`); a `*big.Rat` is tested by size in
  constant time. Behaviour change: a document with a number of more than 38
  digits, such as `1e66`, used to parse and fail only when arithmetic touched
  it; it now fails at parse. The scale limit is 200 digits after the point (the
  runtime accepted 201 through a loop quirk; the Python port always stopped at
  200).
- Numbers handed in as Go values (T-28). `Data`, and so `Run` and
  `shaxon.Validate`, convert the `int`, `uint` and `float` types, `json.Number`
  and `jaxson.Number` to the machine's numbers, a float through its shortest
  decimal, and fail `NUMERIC_OVERFLOW` for NaN, infinity, or a value that is not
  a Real. Before, such a value was reported as "expected number, got unknown".
  `TestDataLegacyRoundTrip` used 1/3, which is not a decimal and so not a number
  of the language; it now uses 1/4.
- Relaxed numeric mode (core design, section 2; T-30). A number that does not
  fit the strict range is an IEEE 754 float64 instead of an error: a literal
  beyond the limits, a string read by `to_number`, a number handed in as a Go
  value, or the result of an operation. A float and a Real compare and order by
  exact value; an operation with a float is done exactly and rounded once to the
  nearest float64, and a float that is itself a Real is that Real. A value no
  float64 can hold is still `PARSE_ERROR` or `NUMERIC_OVERFLOW`. Strict stays
  the default and is unchanged. `jaxrun` and `shaxonrun` take `-strict`, the
  default, and `-relaxed`, which overrides it; `-strict -relaxed` and
  `-strict=false` are errors. The mode is carried by `Profile.Numbers`, not a
  global. New API: `jaxson.NumericMode` (`Strict`, `Relaxed`, `ModeFromFlags`),
  `ParseJSONMode`, `ParseDataMode`, `ParsePackageMode`, `DataMode`,
  `NewMachineMode`, `Machine.SetNumericMode`, `AsRat`, `shaxon.RunMode`,
  `RunJSONMode`, `ValidateMode` and `jaxtools.NumericFlags`. Tests:
  `relaxed_test.go` in both packages and `format_float_test.go`. `jaxplay` takes
  the switches too. The Python port is strict only. This runtime computes
  relaxed results exactly and rounds once, so they are reproducible across
  architectures, though the standard does not require it.
- Versioning (T-04): a version is `X.Y.Z-pNNN.rMMM`, where `X.Y.Z` is the Shaxon specification
  version implemented and the suffix counts releases of the implementation at that
  level, in zero-padded counters so that it sorts as text as well as as numbers;
  this is `0.3.1-p000.r001`. `VERSION`, `pkg/version` and the register header are kept in
  step by `repoman syncver`, configured in `.repoman.json`.
- `.gitignore` ignores repoman's `.ed-journal.json`, so an edit journal can no longer
  be left in the tree as an untracked file (T-03).
- `jaxtools.LoadPackage` and `LoadTurns` read with the language's own reader
  (`jaxson.ParseJSON`), with `LoadPackageMode` and `LoadTurnsMode` for a mode. A
  duplicate key is now an error, and a number that is not a Real is an error
  instead of a panic from `Normalize`.
- Range errors name the limit and the way out: `number out of range: a Real has
  at most 38 digits and 200 after the point; relaxed mode reads a larger number
  as a float64` for a literal, and `not a Real: ...` (`NUMERIC_OVERFLOW`) for a
  result; in relaxed mode a value no float64 can hold says so. The Python port's
  messages match.
- `jaxson.Number` (T-26, T-27): `NumberFromRat(nil)` and `AsNumber` of a nil
  `*big.Rat` return false instead of panicking. `Number` unmarshals from JSON
  under the same rule (`UnmarshalJSON`, `UnmarshalText`), and its documentation
  says to compare with `Cmp`, not `==`.
- New tests: `strict_numbers_test.go`, `host_numbers_test.go` (both packages),
  `number_json_test.go`, `examples/showcase/tools/test_strict_numbers.py`, and 11
  conformance fixtures (`jaxson-v0.1.0-strict-numbers-fixtures.json`, run by
  `strict_fixtures_test.go` and `examples/showcase/tools/run_strict_fixtures.py`).
  Relaxed mode has 14 more fixtures (`jaxson-v0.1.0-relaxed-numbers-fixtures.json`,
  run by `relaxed_fixtures_test.go`).
  The differential tests against `encoding/json` accept the one disagreement
  strict mode creates, a number that is not a Real.
- A shape that extends another can now only narrow it (ruling G11, layer 1).
  `extends` carries the parent's `kind`, reference target (`index`,
  `relation`, `of`/`by`), primitive keywords, `items` and `qualified`. Where
  both shapes state a keyword the stricter wins: the larger of two minimums,
  the smaller of two maximums, `int` if either asks, the intersection of two
  enums. A different kind, reference target, `items` or `qualified` from the
  child's is a load-time `SHAX_SHAPE_ERROR`; restating the parent's is
  accepted. Before this a child lost all of these unless it restated them, so
  extending a `minLen` shape silently dropped the bound, and extending a
  `reference` shape panicked at load. This reverses the specification entry
  below, which had settled on "the child's own keywords only". Core section 4's
  merge table has a row for each member. `severity` and `message` are still
  the child's own only (layer 2, open). The error message for a merge failure
  now names the extending shape (it was empty). 19 conformance fixtures (194 at
  that point) and 22 new mutants of the engine (69 then, 67 killed) pin the change.

### Specification
- The eight points the specification left open were settled by writing the
  implementation's behaviour into it, since that behaviour passes every
  fixture and example and is the fastest of the options. Shaxon core section 3
  (index reuse is invalidated by any overlap with what the build reads, P1),
  section 4 (`extends` carries kind, fields, required, the combinators, `check`
  and `qualified`, not the parent's primitive keywords, G11; one step per shape
  activation, G1), section 2 (`maxShapeDepth` counts named-shape descents
  only, G8), section 6 (a depth overrun is attached at the last node taken,
  C3), section 7 (`mode` is required, V1; a `validate` entry sets its report
  and a `check` appends to it, V3; entries rooted wholly in `input` run before
  the program and the rest after, R1; `unique` skips an element lacking the
  field, V4) and section 10 (rows for a closed object's extra members and an
  unmet `qualified` count, S4); and Jaxson core design section 5 (the members
  of a `$tpl` object are evaluated in sorted key order, S9). Fourteen new
  conformance fixtures pin them (175 in all), and three new mutants of the
  engine (the literal reuse reading, one stage for every entry, reversed
  template order) are killed by them; a shape activation made free, which no
  fixture used to catch, is now killed too. The mutant check's anchor for the
  `$indexed` sort, broken by the earlier sort change, is repaired and split in
  two, and the numeric-key order it exposed as unpinned by fixtures is
  recorded in its list of known survivors.

### Changed
- An index whose key is a `concat` of string literals and bound values is built
  into a byte slab: `Machine.ConcatKey` (new, `pkg/jaxson/concatkey.go`) appends
  each key to a caller's buffer with the island's bindings, charges and TYPE_ERROR,
  and declines when the host has replaced `concat` or the `$compute` form;
  `pkg/shaxon/slabindex.go` keeps the keys in an open-addressing table over
  parallel arrays and makes entries only for distinct keys, in three allocations
  (one string, one entry block, one positions array). `Lookup` and `Inverse` read
  that table. No result or step count changes (the 161 fixtures and 51
  authorisation cases pin both, compared against the tree interpreter, which
  never takes this path; new `slabindex_test.go`). `chinese-wall` at 20000 events,
  engine only, same host: about 18 ms to 7.5 ms (30-35 to 11-13 ms on a slower
  host). The five examples were re-run together on one host; see TRACKER PF4 for
  the figures against Jena.
- Engine speed-ups in index building and compute-binding borrowing, none changing
  a result or a step count (the 161 fixtures and 51 authorisation cases pin
  both). An index is sorted only when its order is first needed, and its key map
  is presized when it is not `multi`; the loop variable is bound once per build
  (`Machine.WithLocalSet`) and not once per element; the mutation log keeps only
  mutations that overlap a built index's dependencies, and path segments compare
  without printing; and a `$compute` binding is now borrowed through `get`,
  `get_or` and the branches of `select` as long as it only reaches `has`, `len`,
  `type_of`, `keys`, `eq`, `ne` (or `and`/`or`) and never leaves the island. At
  20000 events, engine only: `chinese-wall` 22 to 18 ms, `four-eyes-release` 28 to
  18 ms, `delegation-chain` 19 to 14 ms. Five further micro-changes (a presized
  map for `multi` indices, a hoisted cost lookup, a per-build compiled key,
  exact-size position lists, cheaper ambient updates) measured no gain and were
  not kept.
- Wall time on `four-eyes-release` and `break-glass` was quadratic in the trail
  (TRACKER PF3). Two causes, both fixed, with no change to any result or step
  count (the 161 fixtures and the 51 authorisation cases pin both). In the
  compiler, a `$compute` binding of the form `{"$path": [...]}` whose every use
  in the expression is an argument of `has`, `len`, `type_of`, `keys`, `eq` or `ne`
  (or that is not used) is now read without cloning a composite from state or
  output (`borrowOnly`, `pathReadMode` in `compile.go`), where it used to copy
  the whole `state.pay` map for every event. In `break-glass.json` the two
  lookup keys that default a missing `id` now default to `(no id)`, not `""`,
  which every break-glass event's index key also was. At 4000 events
  `four-eyes-release` takes 7.5 ms (was 2.8 s) and `break-glass` 40 ms (was 4.6
  s); at 20000, 33 ms and 176 ms (were 80 s and 148 s). Both example batches were
  re-run and `results/REPORT.md` regenerated. Tests for the borrow analysis in
  `pkg/jaxson/borrow_test.go`, including cases taken from a parallel draft of the
  same fix.
- Documentation brought into line with the published repository layout: the
  top-level `README.md` rewritten (layout table, quick start, links fixed,
  benchmark findings); `docs/IMPLEMENTATION-STATUS.md`
  refreshed, as it still described Shaxon as having no evaluator; the core
  specification's status line and section 13 now say an implementation exists;
  this directory's README and TRACKER point at the root specifications and
  at the renamed Walk proposal.

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
  current Maven rejects; reworded. The first full run found Shaxon quadratic in
  wall time on `four-eyes-release` and `break-glass`; fixed, see Changed.
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

- `pkg/shaxon`: the static layer (plan phase 2, core sections 2 to 4d).
  `ParseRegistries(pkg)` parses and statically validates `shapes`, `indices`,
  `relations` and `computes` without any input. `extends` is resolved by
  default merging plus the override mechanism (`closedOverride`,
  `andOverride`, `orOverride`, `xoneOverride`, `notOverride`, `checkOverride`,
  and a per-item `override` in `fields` and `requiredIds`); an `extends`
  cycle is always `SHAX_SHAPE_ERROR`. Also checked: `limits.maxShapeDepth`
  reachability, and keyword-versus-kind legality and cross-keyword ordering
  after the merge. `extends` is valid only on named top-level shapes.
  `jaxson.OperatorDef` gained `MinArity` and `MaxArity`. 32 tests in
  `registries_test.go`; the full record is in TRACKER.md, "Phase 2".

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
- Phase 2 (`pkg/shaxon` registries and parser) is summarised in one entry under Added;
  TRACKER.md holds the full record.

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
