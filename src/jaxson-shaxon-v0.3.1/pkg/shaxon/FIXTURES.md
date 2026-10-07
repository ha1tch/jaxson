# Shaxon conformance fixtures

`shaxon-v0.3.1-fixtures.json` is a list of complete Shaxon packages, each with
the result the core specification (`shaxon-v0.3.1-core.md`) requires of it.
It is written to be read by any implementation, not only this one: a fixture
is a runnable package plus three bookkeeping members that are not part of the
package.

| Member | Meaning |
|---|---|
| `name` | unique; says what the fixture is about |
| `note` | why the fixture exists and what a wrong runtime would do differently |
| `expect` | the required result (below) |

`expect` is one of two shapes.

| Shape | Meaning |
|---|---|
| `{"output": V}` | the run succeeds and its output equals `V` |
| `{"error": {"category": C, "code": X}}` | the run fails with that category and, if `code` is given, that code |

A success may add `"report": R` and `"steps": N`.

- `report` is the combined report the run returns alongside its output
  (core section 7, last bullet). `"report": null` means none is returned.
  `conforms` must be equal and `violations` must have the same length, in the
  same order. An expected violation lists only the members the core fixes for
  that case, and each listed member must be equal. A message the core leaves
  to the runtime is simply not listed. `{"$absent": true}` as a value requires
  the member to be missing, as opposed to null.
- `steps` pins the exact step total. Only two fixtures use it: the core fixes
  only some charges (an index build, a closure hop), not a program's total, but
  it does say that `aggregate` costs exactly what its expansion costs, so one
  `aggregate` fixture and its hand-written fold are pinned to the same figure
  (18 under the `unit` table). Other cost rules are tested with a step limit: a
  limit that a runtime obeying the rule stays under (or exceeds) and a runtime
  ignoring it does not.

Identifiers are spelt as the core spells them. The ones Shaxon introduces carry
the prefix `SHAX_` (`SHAX_VALIDATION_ERROR`, `SHAX_SHAPE_MISMATCH`,
`SHAX_DANGLING_REFERENCE`, ...); the ones Jaxson already defines do not.

## Running

```
go test -count=1 -run TestFixtures ./pkg/shaxon/        # the fixtures
python3 pkg/shaxon/fixtures_mutants.py -v               # do the fixtures notice broken rules?
```

The first is part of the default test run. The second is not: it copies the
module, breaks one engine rule at a time and requires the fixtures to fail.
It is listed in the dormant guards table in `TRACKER.md`.

## How they were written

From the core specification, not from what the implementation does. For each
fixture the expected result was written first, and a failure was settled by
reading the specification, not by editing the expectation to fit. Where the
specification is silent, no fixture asserts a result; the cases are listed
under "Left out on purpose" and filed in `TRACKER.md`.

## Coverage

| Group (name prefix) | Fixtures | Core section |
|---|---|---|
| `granularity-*`, `severity-*` | 12 | 10 (the six-row granularity table, severity and `conforms`) |
| `minimal-*`, `extends-*`, `check-and-check-override-*`, `override-outside-*` | 31 | 4 (minimal combinator evaluation, the `extends` merge rule and the spelling of each override) |
| `gate-*`, `report-lists-*`, `report-orders-*` | 8 | 7, 10 (gate and report ordering) |
| `unique-*` | 7 | 7 |
| `qualified-*` | 8 | 4 (counting, inclusive bounds, the step charge) |
| `reference-*`, `index-*`, `relation-*`, `two-relations-*` | 21 | 3, 4a, 4c, 5 (including the index build charge and the reuse rule) |
| `indexed-*`, `inverse-*` | 4 | 6, 7 (visiting order) |
| `closure-*`, `altpath-*` | 13 | 6 (bounded closures, the horizon, `$altPath`) |
| `recursion-*` | 4 | 2, 10 (`maxShapeDepth`) |
| `closed-*`, `ignored-*`, `each-*`, `field-*`, `wrong-*`, `enum-*`, `required-ids-*` | 10 | 4, 10 |
| version, pipeline order and `output-*` | 10 | 2, 9 |
| `check-*`, `validate-*`, `only-*`, `report-entries-*` | 7 | 7, 8 (the `check` instruction, report delivery) |
| `aggregate-*` | 22 | 8a (the measures, `where`, the empty result, exact sums, cost equal to the expansion, malformed forms) |
| `attic-*` | 6 | mined from `attic/` (below) |
| the rulings of 2026-10-07 (see below) | 14 | 3, 4, 6, 7, 10 and the Jaxson template section; each note names its ruling |
| `extends-*` from the G11 layer 1 ruling (see below) | 19 | 4 (what `extends` carries and how a child narrows) |

The counts add up to the 196 fixtures in the file: the 163 of the groups above, the 14 of
the rulings of 2026-10-07 and the 19 of the `extends` narrowing group. The `extends-*` names of the
last two groups are counted there and not in the `extends-*` row. 78 of the fixtures expect an
error, the rest a result.

Two fixtures in the `gate-*` and `unique-*` rows pin that only a violation-severity finding
aborts a gate (`gate-does-not-abort-on-a-warning-or-info-finding`,
`unique-gate-does-not-abort-on-a-warning-severity-repeat`).

The 14 of the rulings row pin the rulings that settled the open points of the tracker
(P1, V1, V3, R1, G11, S4, S6, S9) by writing the implementation's behaviour into the
specification: `extends-carries-the-parents-primitive-keywords` (rewritten by the layer 1 ruling below),
`tpl-members-are-evaluated-in-sorted-key-order`,
`max-shape-depth-counts-only-named-shape-descents`,
`max-shape-depth-does-not-count-inline-structure`,
`a-shape-activation-costs-one-step`, `unique-skips-an-element-lacking-the-field`,
`path-depth-exceeded-is-reported-at-the-last-node-taken`,
`a-validate-entry-without-a-mode-is-a-shape-error`,
`input-rooted-entries-run-before-the-program-and-the-rest-after`,
`closed-object-reports-one-violation-per-extra-member`,
`failed-qualified-count-is-one-violation-at-the-collection`,
`index-reuse-is-invalidated-by-a-write-below-the-source`,
`index-reuse-survives-a-write-that-overlaps-nothing-it-reads` and
`validate-report-with-into-sets-the-whole-report`.

The 19 of the `extends` narrowing group pin layer 1 of ruling G11, that a child can only narrow what it
extends: the stricter numeric, length and item-count bound wins from either side
(`extends-takes-the-stricter-numeric-bound-from-either-side` and its two siblings), `int` from either
side, enum intersection, a chain of three, an inherited reference target, `items` and `qualified`,
the same `items` or `qualified` restated, and a `SHAX_SHAPE_ERROR` each for a different reference target,
kind, `items`, `qualified`, and a merged minimum above its maximum. Two more fix what "the same" means (a named shape and its inline copy differ) and that disjoint enums accept nothing. A parent of kind `node` may be
narrowed to a kind (`extends-lets-a-node-parent-be-narrowed-to-a-kind`).

### Mined from the attic

The six fixtures in `attic/shaxon-v0.1.1-fixtures.json` and
`attic/shaxon-v0.1.2-fixtures.json` were rewritten for `"3.1"` and carry an
`attic-` prefix. Each note says what changed. Three rewrites are worth
knowing about, because each shows a premise that no longer holds.

- An object is closed by default, which the 0.1 fixture did not assume. Its
  closure fixture had to declare `closed: false` to keep its meaning.
- `$inverse` yields paths, not elements.
- A member holding `null` still resolves along a closure. The old list ended
  in a `null` link, which under section 6 is one hop more than it looks. The
  finding became its own fixture, `closure-through-a-null-link-still-resolves-and-can-overrun`.

## Left out on purpose

These are cases where the specification is silent or contradicts the
implementation, so a fixture would assert the implementation, not the
specification. Each has a row in `TRACKER.md`, under the ID in brackets (G, V, R, C and P are
implementation decisions logged in earlier phases; S1 to S8 are the findings of
Phase 6; S1, S2, S5 and S8 are settled and have fixtures, S3 is settled as "not specified", S4 and S6 are open).

| Case | Why there is no fixture |
|---|---|
| Report-mode `check ... into` (V3) | Whether `check` appends and `validate` replaces (V3) awaits a ruling. The core makes no claim about validation and the data it validates (S3: deliberately unspecified), so nothing fixes what a second `check` sees after a first has delivered into the data. The fixtures use the combined returned report instead. |
| A mixed `validate` list split between input and output stages (R1) | R1: the core does not say how entries are assigned to a stage. The pipeline fixtures use only entries whose stage is obvious. |
| A child that does not write `closed` at all (S8) | The core says how two stated values combine, not what an unstated one means. The implementation inherits the parent's posture. |
| A failed `qualified` count, and an extra member of a closed object, in a report (G2, G3, S4) | The section 10 table is called exhaustive and has neither row. The fixtures test them through gate mode, which needs no violation shape. |
| What counts as one level of `maxShapeDepth` (G8) | The core does not define it. The recursion fixtures use a wide margin (a bound of 2 against 10 levels). |
| A `unique` element lacking its field (V4) | V4. The core does not say whether it is skipped or an error. |
| Where a `SHAX_PATH_DEPTH_EXCEEDED` finding is attached (C3) | The core fixes its message, code and severity, not its `focusPath`. |
| The cost of a plain shape activation (G1) | The core fixes the cost of an element checked through `qualified` as "the ordinary shape-evaluation cost" without stating it. The cost fixtures rest on the `check` inside the shape. |

## What the mutation check shows

`fixtures_mutants.py` tries 43 deliberate breaks of the engine: a combinator
that no longer short-circuits, a qualified bound made exclusive, an index that
is never rebuilt or never reused, a closure allowed one hop too far, fields
visited in the wrong order, a warning that flips `conforms`, an override
spelling ignored, an `aggregate` that ignores its `where`, and so on. 41 are
killed by the fixtures. The other two are recorded in the script with the
reason: a cost the core does not fix, and an equivalent mutant (a second check
elsewhere enforces the same rule). A mutant that survives for any other reason
is a missing fixture.

This is a sample of the engine's rules, not a proof that the fixtures cover
all of them.
