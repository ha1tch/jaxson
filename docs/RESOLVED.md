# Resolved work

Closed items, newest first, with the full detail text as it stood at closure.
Append-only: an entry is never edited after it is written.

## [0.3.1-p000.r002] T-02 — Confirm the N1 and N2 `Number` scope (v0.3.1-p000.r002, 2026-10-15)

Theme: public-api · closed 0.3.1-p000.r002 · 2026-10-15
**Trigger:** Phase 7 asks for confirmation that `jaxson.Number` exists as an opt-in type while results still hold `*big.Rat` (decisions N1 and N2 in TRACKER.md).

**Scope:** Confirm that scope, or decide on the separate, breaking step of making results hold `Number`.

Cross-ref: CHANGELOG 0.3.1-p000.r002.

## [0.3.1-p000.r002] T-01 — Confirm the A1 `Validate` signature (v0.3.1-p000.r002, 2026-10-15)

Theme: public-api · closed 0.3.1-p000.r002 · 2026-10-15
**Trigger:** Phase 7 asks for confirmation of `Validate(pkg, shape, doc) (Report, *jaxson.Err)`, which differs from the plan's `(pkg, target)` (decision A1 in TRACKER.md).

**Scope:** Confirm the signature or change it; `ValidateJSON` follows. First on the board; nothing is blocked on it technically.

Cross-ref: CHANGELOG 0.3.1-p000.r002.

## [0.3.1-p000.r001] T-04 — Decide whether the work warrants a `VERSION` bump past 0.3.1 (v0.3.1-p000.r001, 2026-10-15)

Theme: release · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** `VERSION` is 0.3.1; the Unreleased changelog section holds phases 3 to 7 and the tracker calls the bump a release decision.

**Scope:** Decide the bump and whether the release includes G11 layer 2 (T-05 to T-08). About a dozen files carry 0.3.1, and some mean the spec version, so each occurrence needs classifying before any substitution. Ordering only after T-01 and T-02; T-05 waits for this decision. The strict number rule (T-25) is a behaviour change that belongs in the same release: a document with a number of more than 38 digits parsed before and fails at parse after it.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-03 — Remove the stray `.ed-journal.json` from the author's tree before packaging (v0.3.1-p000.r001, 2026-10-15)

Theme: release · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** A stray `pkg/shaxon/.ed-journal.json` was present in the author's working tree; it is not in the published repository.

**Scope:** Check the working tree and delete the file before packaging. Ordering only: it follows T-01 and T-02 because of the order chosen for this board.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-30 — Implement relaxed numeric mode (IEEE floats beyond the Real limits) (v0.3.1-p000.r001, 2026-10-15)

Theme: numeric-limits · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** the Jaxson core design (section 2) defines relaxed numeric mode: a number that does not fit the strict range is an IEEE 754 binary64 float. `jaxrun` and `shaxonrun` had a `-relaxed` switch that refused it.

**Scope:** floats as `float64` in the machine, carried by `Profile.Numbers`; `Parse*Mode`, `DataMode`, `NewMachineMode`, `shaxon.RunMode`, `RunJSONMode` and `ValidateMode`; the operators, comparison, schema checks, index keys and rendering; both command-line switches; tests in `relaxed_test.go` (jaxson and shaxon) and `format_float_test.go`. The Python port stays strict only, as relaxed mode is optional.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-28 — `Validate` reports a document value outside the value model as `expected number, got unknown` (v0.3.1-p000.r001, 2026-10-15)

Theme: public-api · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** A Go `int`, `float64` or `jaxson.Number` in a document gives a violation with that message, not an error.

**Scope:** Decide whether such values are an error (`TYPE_ERROR`) or are converted, and say which in decision A1. Ordering only after T-08.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-27 — `Number`: `==` and map keys compare pointers, and there is no unmarshal side (v0.3.1-p000.r001, 2026-10-15)

Theme: public-api · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** Two equal Numbers are `!=` and distinct map keys because `Number` holds a pointer; only `Cmp` compares by value. `json.Unmarshal` into a `Number` fails with Go's own error, although `MarshalJSON` and `MarshalText` exist.

**Scope:** Document use of `Cmp`, or make `Number` comparable by value; decide whether `UnmarshalJSON` is wanted. Ordering only after T-08.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-26 — `NumberFromRat(nil)` and `AsNumber((*big.Rat)(nil))` panic (v0.3.1-p000.r001, 2026-10-15)

Theme: public-api · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** Both dereference a nil `*big.Rat`; `AsNumber` is documented to report false for any other value.

**Scope:** Return false for a nil Rat in both, and add a test. Ordering only after T-08.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-25 — Huge-exponent numbers cost CPU and memory at parse and are accepted by the runtime (v0.3.1-p000.r001, 2026-10-15)

Theme: numeric-limits · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** `ParseJSON` and `ParseData` build the full rational for a literal such as `1e999999` (about 28 ms and 0.4 MB each; 100 of them in 901 bytes took 2.8 s). The runtime then accepts the value: it can be copied to `output` (a 1,000,000-byte result from a 9-byte input), compared, and validated as a conforming `number`. Only arithmetic rejects it (`NUMERIC_OVERFLOW`), after 65 ms for `1e999999` plus 1 and 515 ms for `1e-999999` plus 1. `ParseNumber` rejects the same literal.

**Scope:** Bound the cost before the value is built, from the literal's digit count and exponent, and enforce the 38-digit and 200-fractional-digit limits where a value enters the runtime, not only on arithmetic results. The parser may still accept the literal as grammar-valid (it is differential-tested against `encoding/json`), so the decision is how: a marker value the runtime rejects on entry, or a rejection with an exception recorded in `parse_diff_test.go`. An adversarial package of a few megabytes can otherwise exhaust CPU and memory.

Cross-ref: CHANGELOG 0.3.1-p000.r001.

## [0.3.1-p000.r001] T-29 — Documents restate the number rules without the strict and relaxed modes (v0.3.1-p000.r001, 2026-10-15)

Theme: docs · closed 0.3.1-p000.r001 · 2026-10-15
**Trigger:** the Jaxson core design now defines Reals and the strict and relaxed numeric modes (section 2). `shaxon-v0.3.1-core.md` (the list of what Shaxon inherits) still says "the 38-digit magnitude bound"; `jaxson-intro.md` says Jaxson has no floating point, and `README.md` says any conforming runtime gives the same result, both without saying that this holds in strict mode only.

**Scope:** point the Shaxon core bullet at Jaxson core section 2 instead of restating the bound, and qualify the intro and README statements with the strict default and the relaxed switch. Ordering only after T-08.

Cross-ref: CHANGELOG 0.3.1-p000.r001.
