# Jaxson showcase: report

Written 2026-09-28, after building `examples/showcase/`. Findings are labelled by how
they were established, because that differs:

- **[observed]** I ran it and saw the result (against the Python cross-check executor).
- **[source]** I read it in `pkg/jaxson` or the design doc and did not run it.
- **[untested]** Inference or reasoning only.

Nothing here has been run against the Go implementation. There was no Go toolchain.
The Python executor (`tools/jaxsonpy.py`) is a port I wrote from the Go source it
sits beside. It passes all 28 official fixtures, but it shares my reading of that
source, so a misreading on my part would be present in both the examples and the port.
Running `TestShowcaseFixtures` in Go is the check that could expose that.

---

## 1. Index of examples

Every package has a happy-path input in its own file; `showcase-fixtures.json` adds
17 variants (29 cases in total).

**General domain**

| ID | Package | Summary | Variants |
|----|---------|---------|----------|
| A1 | `a1-recipe-scaler` | Scales ingredient amounts from a base to a target serving count (round half-up to 2 dp) and builds a label such as "5 tbsp olive oil" | none |
| A2 | `a2-library-fine` | Overdue fine = days x daily rate, capped at a maximum; lost books cost the replacement price; on-time returns are waived | lost book; returned on time |
| A3 | `a3-word-frequency` | Counts words into a map with dynamic keys, deletes stopwords, emits `{word, count}` sorted by word | none |
| A4 | `a4-tictactoe-move` | Validates a move: in range, cell empty, not blocked by a variant rule; reports the reason | occupied cell; blocked cell |

**Real-world finance (no securities)**

| ID | Package | Summary | Variants |
|----|---------|---------|----------|
| B1 | `b1-loan-amortization` | Per-period interest/principal split with interest rounded to cents each period, running balance, an origination row inserted at index 0 | payment overshoots balance (fails) |
| B2 | `b2-invoice-discount-tax` | Line items with quantities as strings, volume-discount tiers (0/5/10%), per-line rounding, tax on the discounted subtotal | none |
| B3 | `b3-bill-split` | Splits a total in integer cents; leftover cents go to the first payers; asserts the shares sum to the total | people/names count mismatch (fails) |
| B4 | `b4-dti-eligibility` | Debt-to-income screening over a map of debts plus an optional extra field; approved / conditional / denied, with different output shapes | denied; approved with extra obligation |

**Cloud API control of data-science packages**

| ID | Package | Summary | Variants |
|----|---------|---------|----------|
| C1 | `c1-training-job-sizing` | Chooses a compute instance for a training job from model family and dataset size; approves or denies against an hourly budget | over-budget GPU; small linear model |
| C2 | `c2-hyperparam-grid` | Expands learning-rate x batch-size into job specs, pins a baseline first, enforces a job-count quota | quota exceeded (fails) |
| C3 | `c3-autoscaler` | Desired replica count from utilisation ratio, with deadband, rounding and min/max clamping; halts early inside the deadband | inside deadband; scale-down clamped to min; clamp equals current (no-op); inverted bounds (fails) |
| C4 | `c4-inference-quota-gate` | Allow/deny a batch-inference request against a window quota; admin override; remaining budget floored at zero | admin override; within limit |

Coverage (checked by script): all 8 instructions and all 30 compute operators appear at
least once, including every feature the core design doc listed as having no fixture.

---

## 2. What went well

- **The port validated first time it was allowed to.** It failed on one official fixture
  (loop-variable shadowing) only because I had skipped the static checker. With a ported
  checker added, all 28 official fixtures passed. That is what made the rest trustworthy
  enough to proceed. **[observed]**
- **All 12 packages ran on the first execution and matched my hand-predicted results**,
  including the six-period B1 ledger, the B3 cent sums and the B4 percentages. **[observed]**
- **Exact decimals earned their keep.** B1 rounds interest each period and carries the
  rounded figure forward, B3's shares provably sum to the total, and B2's per-line
  rounding leaves an exact subtotal. There is no drift to manage. **[observed]**
- **The static checker caught nothing in my programs**, which is weak evidence the
  grammar is learnable from the spec alone: arities, operand forms and path rules were
  followable without trial and error. **[observed]**
- **Mechanical coverage scanning paid off.** It showed I had planned `list` but never
  used it, and that `delete`, `halt`, `concat` and `ne` were also unexercised. All five
  were then added where they do real work. **[observed]**
- **Variants found a real bug in my own example.** In C3, when clamping left the replica
  count unchanged, the draft reported `scale_down`. It now reports `none`. **[observed]**
- **Dynamic path segments work as object keys, not just array indices** (A3 counts and
  deletes through `["state","counts",{"$path":["local","word"]}]`). **[observed]**
- **`anyOf` gave B4 and C1 honest distinct output shapes** (a denied result carries a
  `reason`; an approved one does not), enforced by the contract rather than by convention. **[observed]**
- **`assert` works as a cross-field validator** (B3 count mismatch, C3 inverted bounds)
  where the schema cannot. **[observed]**

---

## 3. Difficult or convoluted, but implemented anyway

- **Every operand must be pre-bound.** `expr` can only reference `with` bindings, never
  paths, so even `a + b` needs two bindings. Programs are 3-4x longer than the logic
  they express. **[observed]** Workaround: none needed, just verbosity.
- **`expr` cannot build objects, and can build only flat arrays via `list`.** Any
  structured value must be assembled with `$tpl` around individual computes. **[source]**
  Workaround: `$tpl` with a `$compute` per field.
- **No accumulate-in-place.** Each running total is read, computed and set back
  (B2 subtotal, B3 check-sum, B4 debt total). **[observed]**
- **No `switch` or `elif`.** C1's decision table is four levels of nested `if`/`else`,
  and B4/B2 need two. It works but reads badly. **[observed]**
- **Iteration count is bounded by an existing array.** There is no range and no `while`.
  B1 supplies a `payment_schedule` array only so the loop has something to walk. The
  README originally said this was because the standard annuity formula is impossible;
  **that was wrong.** A test program computes (1+r)^n by looping over an n-element array
  and yields the correct 1695.95 for the sample loan. The payment could have been
  derived in B1, at the cost of a second loop; I did not change B1. **[observed]**
- **Cartesian products of arbitrary dimension** look possible by index arithmetic with
  `mod`/`div` over a supplied array of length (product of sizes), but that needs the
  caller to supply that array, so C2 is a fixed two-parameter grid instead. **[untested]**
- **Early exit duplicates the output template.** `halt` skips the tail of the program,
  so C3 builds its output twice (deadband branch and normal path). **[observed]**
- **Optional inputs.** An absent optional field makes a plain `$path` fail. B4 reads it
  with `get_or` on the whole `input` root, which works but is not obvious. **[observed]**
- **Scratch space pollutes `state`.** Locals exist only as loop variables, so
  intermediates live in `state` alongside real results. **[observed]**
- **Object iteration.** `for` walks arrays only; iterating an object means `keys` first
  and `get` per key (A3, B4). **[observed]**
- **Verifying without Go.** I wrote a second implementation to be able to execute
  anything at all. That worked, but it costs an unverified test file
  (`showcase_test.go`), an unverified path assumption for the `jgo/` symlink layout,
  and correlated-error risk described above. **[observed]**

---

## 4. Limitations I could not work around

Judged against Jaxson 1.0 as specified. "Cannot" means no route in the language, not
merely no convenient one.

**Language**

1. **No data-driven iteration count.** No `while`, no recursion, no range. Any
   algorithm whose step count depends on the data (Newton iteration, gcd, binary
   search, tree walk) cannot be written unless the caller supplies an array whose
   length bounds it. **[source]**
2. **No error handling.** A failed `assert` or `TYPE_ERROR` ends the run. A program
   cannot catch it and shape a graceful result. B1's overshoot is a hard
   `ASSERTION_FAILED`, where a real servicer would adjust the final payment.
   Programs can only guard with `if` before failing. **[observed]**
3. **No number formatting.** `to_string` emits the canonical form only, so `4.00`
   becomes `"4"`. Currency display strings with trailing zeros, padding or thousands
   separators cannot be produced. **[observed]**
4. **Almost no string operations.** Only `concat`, `len`, `to_string`, `to_number`.
   No split, substring, case, trim or contains. A3 has to take pre-tokenised words;
   any text-processing task must be done by the caller. **[source]**
5. **No date or time operations.** B1 and A2 take periods and days as pre-computed
   integers. Day-count conventions (actual/360, 30/360) and due-date arithmetic
   cannot be done in-language. This is a large gap for real finance. **[source]**
6. **No aggregate, search or sort primitives.** No sum, no index_of/contains on
   arrays, no sort, no dedupe. Each is a loop, and dedupe and sort are the
   bounded-loop kind that becomes very long. C2's duplicate baseline stays. **[source]**
7. **No square root, log or exponential.** Standard deviation, normalisation and
   compound growth for non-integer exponents cannot be computed; this rules out most
   statistics a data-science control plane might want to do itself. **[source]**

**Schema**

8. **No map-of-T.** An object with arbitrary keys can be declared only with
   `extra: "allow"`, and then the values are not validated at all. In B4 a debt value
   of `"abc"` passes the input contract and fails later as `TYPE_ERROR` at runtime
   rather than as an input error. **[observed]**
9. **No cross-field, conditional or exclusive-bound constraints.** "max >= min",
   "required only when status is X", "strictly greater than 0" are inexpressible.
   `assert` covers them but reports as `EXECUTION_ERROR`, not `INPUT_ERROR`, so the
   failure category tells the caller the wrong thing. **[observed]**
10. **No defaults, patterns or formats.** An optional field has no schema-level default
    and strings cannot be constrained beyond length and enum. **[source]**

**Process**

11. **Not verified against the Go implementation.** Stated once more because it
    bounds everything above: items 1, 4-7 and 10 are from reading source and the spec,
    not from running Go.

---

## Corrections made to the earlier deliverable

- README claimed Jaxson "cannot compute the standard annuity formula". Wrong; see
  section 3. The README has been corrected.
- These limitations were previously only described in the README's prose and in chat,
  not recorded as findings. This report and a pointer in `TRACKER.md` now do that.
