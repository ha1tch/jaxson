# Jaxson showcase examples

Twelve self-contained Jaxson packages, four each in three domains, plus 17 variant
inputs that push each one down its other branches and failure paths.

| # | Package | What it does |
|---|---------|--------------|
| A1 | `a1-recipe-scaler` | Scales ingredient amounts to a target serving count; builds readable labels |
| A2 | `a2-library-fine` | Overdue fine with a cap; lost-book and returned-on-time branches |
| A3 | `a3-word-frequency` | Word counts in a map with dynamic keys, stopword removal, sorted output |
| A4 | `a4-tictactoe-move` | Move validation: range, occupancy, variant-blocked cells |
| B1 | `b1-loan-amortization` | Per-period interest/principal split with cent rounding, opening balance row |
| B2 | `b2-invoice-discount-tax` | Volume-discount tiers and tax; quantities arrive as strings from a legacy system |
| B3 | `b3-bill-split` | Splitting a bill in exact cents; leftover cents go to the first payers |
| B4 | `b4-dti-eligibility` | Debt-to-income screening; approved/conditional/denied with distinct output shapes |
| C1 | `c1-training-job-sizing` | Picks a compute instance for a training job; approved/denied against budget |
| C2 | `c2-hyperparam-grid` | Learning-rate x batch-size grid, baseline pinned first, job-count quota |
| C3 | `c3-autoscaler` | Replica count for an inference endpoint: deadband, ratio, clamping |
| C4 | `c4-inference-quota-gate` | Allow/deny a batch-inference request against a window quota, admin override |

Files: one `<name>.json` per package (runnable, `input` filled in), and
`showcase-fixtures.json` (29 cases: the 12 base packages plus 17 variants) in
the same fixture format as `pkg/jaxson/jaxson-v0.1.0-fixtures.json`, each with an `expect`.

## Coverage

All 8 instructions and all 30 compute operators are used at least once
(checked mechanically, not by hand). That includes everything the core design
doc lists as "implemented but no fixture yet": `insert`, `assert`, `if`, `sub`,
`neg`, `abs`, `min`, `max`, `mod`, `lt/le/gt/ge`, `not`, `to_number`, `keys`,
`get`, `type_of`, `list`, and the schema keywords `anyOf`, `enum`, `min`, `max`,
`minLen`, `maxLen`, `minItems`, `maxItems` and `extra: "allow"`.

Also exercised: dynamic path segments used as object keys (A3) and array
indices (A4), iterating `keys` of an object (A3, B4), `get_or` on a member that
is genuinely absent (B4), nested `for` (C2), `halt` (C3), `anyOf` outputs (B4, C1).

## How the expected values were produced -- read this

There was no Go toolchain available when these were written. The `expect`
blocks were recorded from `tools/jaxsonpy.py`, an independent Python port of
`pkg/jaxson`'s semantics (exact decimals via `fractions.Fraction`). Before
being trusted it was run against the project's own 28 official fixtures and
passes all 28. Every result was also eyeballed against hand arithmetic.

That makes the expectations a **cross-check, not ground truth**. The Go code
has not run any of this. If `TestShowcaseFixtures` fails, either the Go
implementation or the Python port is wrong; find out which before regenerating
anything.

    go test ./pkg/jaxson/ -run TestShowcaseFixtures -v
    # override the location if the default relative path doesn't resolve:
    JAXSON_SHOWCASE=/path/to/showcase-fixtures.json go test ./pkg/jaxson/ -run TestShowcaseFixtures

    python3 tools/run_showcase.py      # Python cross-check on its own

## Things these examples deliberately don't hide

- **No range, no recursion, no exponent.** B1 takes the payment schedule as an
  input array (its length is the term) and a fixed payment, because Jaxson
  cannot generate `1..n` or compute the standard annuity formula. C2 is a fixed
  two-parameter grid for the same reason (loop depth is fixed in the program).
- **Numbers are canonical.** `4.00` serialises as `4`, `40.0` as `40`. Not a bug.
- **`extra: "allow"` skips value checks.** The debts map in B4 and `metadata`
  in C1 accept any values; the closed-schema language has no "map of numbers".
- **C2 keeps a duplicate.** The baseline (0.01/32) also appears in the grid; no
  dedupe was written.
- **C3 was corrected while writing.** An early draft reported `scale_down` when
  clamping left the replica count unchanged; the `clamped-to-current-is-no-op`
  variant pins the fix.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0.
