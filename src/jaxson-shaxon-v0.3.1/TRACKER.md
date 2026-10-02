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
| 2 — Shaxon's static layer | §5 | Not started (blocked on 1) |
| 3 — Shape evaluation | §6 | Not started (blocked on 1) |
| 4 — Indices/relations/targets/`validate` | §7 | Not started (blocked on 1, 3) |
| 5 — `shaxon.Run`, version, pipeline order | §8 | Not started (blocked on 2–4) |
| 6 — Fixtures and conformance | §9 | Not started (blocked on 5) |
| 7 — Public API, `Session`, `build` package | §10 | **7.2 and 7.3 done, verified** (`pkg/jaxtools`, `pkg/jaxson/build`, Navy Wars port). 7.1 (`shaxon.Validate`, `jaxson.Number`) remains and depends on Phases 2–5 |

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

## Phases 2–6 — Shaxon (plan §5–§9) — not started

No `pkg/shaxon/` directory exists in the delivered tree. None of
`registries.go`, `indices.go`, `shapes.go`, `targets.go`, `report.go`,
`check_instr.go`, `run.go`, or a Shaxon `fixtures_test.go` /
`shaxon-v0.3.1-fixtures.json` exist yet. `cmd/shaxrun/` does not exist.

This is expected, not a gap: every one of these phases depends on Phase 1's
extension points by the plan's own dependency table (§13), and Phase 1
hasn't started. Nothing here should be read as "behind schedule" — it's
"not yet reachable."

## Phase 7 — Public API and tooling (plan §10) — partial

- [ ] **7.1 Core API surface** (`shaxon.Validate`, the `jaxson.Number`
  boundary type, immutable-after-construction tables) — not started;
  depends on Phases 1–5 existing first.
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
- An `extends` cycle is unconditionally `SHA_SHAPE_ERROR`, never valid
  under any `maxShapeDepth` value — static merging has no focus node to
  descend through, so no depth bound makes a literal cycle terminate.
- `validate`/`check` shape-xor-`unique` structural checks belong in
  `targets.go` (Phase 4), not here — this package's `registries.go` stays
  scoped to exactly the four registries the plan's file layout names.

**Error identifiers: every one Shaxon introduces is prefixed `SHA_`**
(`SHA_SHAPE_ERROR`, `SHA_VALIDATION_ERROR`/`SHA_SHAPE_MISMATCH`,
`SHA_DANGLING_REFERENCE`, `SHA_SHAPE_DEPTH_EXCEEDED`,
`SHA_PATH_DEPTH_EXCEEDED`). `EXECUTION_ERROR` itself keeps Jaxson's own
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
(including the `closed`-default bug below), and one test per `SHA_SHAPE_
ERROR` condition implemented so far.

**Self-check note**: the first draft of the `closed` member's default got
it backwards — Go's zero-value `false` instead of core section 4's stated
default ("closed objects are the default posture" means `true` when
undeclared). Caught by `TestParseRegistries_ExtendsMerge` failing, not by
inspection. Worth double-checking stated defaults against Go zero values
specifically in any future section-4-adjacent work — this is exactly the
class of bug Go's zero values make easy to introduce silently.

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