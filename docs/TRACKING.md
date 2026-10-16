# Open work register

Version: 0.3.1-p000.r003
Last reviewed: 2026-10-15

Open items only. A closed item moves to `RESOLVED.md` and leaves no row here.
Status legend: ✓ done · ◐ partial · ☐ not started · ✗ dropped.

Items were filed on 2026-10-14 from the pending rows of waves 3 to 7 in
`WAVE_TRACKING.md`. `Blocks` lists the items that must close first; where an
entry is ordering only and not a technical dependency, its detail says so.

## Status

| ID | Summary | Theme | Priority | Status | Blocks |
|---|---|---|---|---|---|

| T-05 | G11-2a: inherit `severity` and `message` when the child states none | extends-narrowing | P2 | ☐ | T-04 |
| T-06 | G11-2b: rewrite the core section 4 merge table around narrowing | extends-narrowing | P2 | ☐ | T-05 |
| T-07 | G11-2c: property test that a child accepts only what its parent accepts | extends-narrowing | P2 | ☐ | T-06 |
| T-08 | G11-2d: counts, CHANGELOG, `FIXTURES.md` and mutants for layer 2 | extends-narrowing | P2 | ☐ | T-07 |
| T-09 | Navy Wars replay as an integration test | conformance | P2 | ☐ | T-08 |
| T-10 | `costs.Validate` has no test that can fail | engine | P2 | ☐ | T-08 |
| T-11 | G11-3a: loosening flags for kind, items, `qualified` and keywords | extends-narrowing | P4 | ☐ | T-08, T-09, T-10 |
| T-12 | G11-3b: more than one `qualified` rule per shape | extends-narrowing | P4 | ☐ | T-08, T-09, T-10 |
| T-13 | G11-3c: inheriting a shape `id` | extends-narrowing | P4 | ☐ | T-08, T-09, T-10 |
| T-14 | G11-3d: a named shape and its inline copy compared as equal | extends-narrowing | P4 | ☐ | T-08, T-09, T-10 |
| T-15 | Fixtures run by a second, independent runtime | conformance | P3 | ☐ | T-11, T-12, T-13, T-14 |
| T-16 | A fixture that pins `steps` exactly | conformance | P4 | ☐ | T-11, T-12, T-13, T-14 |
| T-17 | R2: a report written into `output` lands after the schema check | engine | P3 | ☐ | T-15, T-16 |
| T-18 | Aggregation inside a constraint (check expression or index key) | engine | P3 | ☐ | T-15, T-16 |
| T-19 | Grouping, `any`/`all` and average not offered | engine | P4 | ☐ | T-15, T-16 |
| T-20 | SHACL Core alone untested for `chinese-wall` and `break-glass` | shacl-comparison | P3 | ☐ | T-15, T-16 |
| T-21 | Which `sh:sourceConstraint` reading follows the W3C text | shacl-comparison | P4 | ☐ | T-15, T-16 |
| T-22 | `setup.sh` install branches unexercised end to end | shacl-comparison | P4 | ☐ | T-15, T-16 |
| T-23 | The 21 phase 5 source mutants are not reproducible | tooling | P3 | ☐ | T-15, T-16 |
| T-24 | repoman provenance flag on `CHANGELOG.md` | tooling | P4 | ☐ | T-15, T-16 |
## Detail

## extends-narrowing

### T-05. G11-2a: inherit `severity` and `message` when the child states none

Theme: extends-narrowing · Priority: P2 · Status: ☐ · Blocks/after: T-04

**Trigger:** G11 layer 1 (a child shape may only narrow) is done; severity and message are still child-only.

**Scope:** Inherit them when the child states none (needs stated flags as `ClosedSet`); a child lowering severity needs an explicit override (option C recommended; A: the child's value wins; B: reject). Design in `docs/proposals/extends-narrowing.md`. After T-04 because that decision fixes the release content.

### T-06. G11-2b: rewrite the core section 4 merge table around narrowing

Theme: extends-narrowing · Priority: P2 · Status: ☐ · Blocks/after: T-05

**Trigger:** The merge table still describes the pre-narrowing rules.

**Scope:** Rewrite it around the narrowing principle with the loosenings listed in one place. Technical dependency on T-05.

### T-07. G11-2c: property test that a child accepts only what its parent accepts

Theme: extends-narrowing · Priority: P2 · Status: ☐ · Blocks/after: T-06

**Trigger:** No test states the narrowing principle as a property.

**Scope:** A child accepts only what its parent accepts, apart from the loosenings; a larger run behind a tag, registered in the dormant-guards table. Technical dependency on T-06, which lists the loosenings.

### T-08. G11-2d: counts, CHANGELOG, `FIXTURES.md` and mutants for layer 2

Theme: extends-narrowing · Priority: P2 · Status: ☐ · Blocks/after: T-07

**Trigger:** Each layer 2 change moves counts and needs a changelog entry, fixture documentation and mutants.

**Scope:** Do it as each of T-05 to T-07 lands; closes last. Technical dependency on T-07.

### T-11. G11-3a: loosening flags for kind, items, `qualified` and keywords

Theme: extends-narrowing · Priority: P4 · Status: ☐ · Blocks/after: T-08, T-09, T-10

**Trigger:** Layer 3 is postponed in the tracker; the flag spelling is undecided.

**Scope:** Design and add the flags. Technical dependency on T-08; ordering only after T-09 and T-10.

### T-12. G11-3b: more than one `qualified` rule per shape

Theme: extends-narrowing · Priority: P4 · Status: ☐ · Blocks/after: T-08, T-09, T-10

**Trigger:** Layer 3 is postponed in the tracker.

**Scope:** Name the rules and merge by name. Technical dependency on T-08; ordering only after T-09 and T-10.

### T-13. G11-3c: inheriting a shape `id`

Theme: extends-narrowing · Priority: P4 · Status: ☐ · Blocks/after: T-08, T-09, T-10

**Trigger:** Layer 3 is postponed in the tracker.

**Scope:** Decide and implement inheritance of a shape's `id` (constraintId). Technical dependency on T-08; ordering only after T-09 and T-10.

### T-14. G11-3d: a named shape and its inline copy compared as equal

Theme: extends-narrowing · Priority: P4 · Status: ☐ · Blocks/after: T-08, T-09, T-10

**Trigger:** Layer 3 is postponed in the tracker.

**Scope:** Compare a named shape with its inline copy as equal. Technical dependency on T-08; ordering only after T-09 and T-10.

## conformance

### T-09. Navy Wars replay as an integration test

Theme: conformance · Priority: P2 · Status: ☐ · Blocks/after: T-08

**Trigger:** Plan section 9 asks for a Navy Wars replay as an integration test.

**Scope:** The builder port (`pkg/jaxson/build`) and `examples/game/turns-seed0.json` exist; turn the replay into a test. Ordering only after T-08.

### T-15. Fixtures run by a second, independent runtime

Theme: conformance · Priority: P3 · Status: ☐ · Blocks/after: T-11, T-12, T-13, T-14

**Trigger:** Plan section 9 asks for the fixtures to be run by a second runtime; none exists.

**Scope:** A separate implementation of the Shaxon runtime judged by the same fixtures. Ordering only after layer 3.

### T-16. A fixture that pins `steps` exactly

Theme: conformance · Priority: P4 · Status: ☐ · Blocks/after: T-11, T-12, T-13, T-14

**Trigger:** The runner supports `steps` but no fixture uses it, because the core fixes no whole total.

**Scope:** Blocked on a core-specification ruling that fixes a total, not on any register item. Ordering only after layer 3.

## engine

### T-10. `costs.Validate` has no test that can fail

Theme: engine · Priority: P2 · Status: ☐ · Blocks/after: T-08

**Trigger:** `unit` has a fallback, so table coverage can never be reported missing.

**Scope:** Add a test with a cost table that has no fallback. Ordering only after T-08.

### T-17. R2: a report written into `output` lands after the schema check

Theme: engine · Priority: P3 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** A post-program entry can write a report into an `output` path after `outputSchema` has been checked.

**Scope:** Recorded in the tracker, not closed; decide the rule and close it. Ordering only after T-15 and T-16.

### T-18. Aggregation inside a constraint (check expression or index key)

Theme: engine · Priority: P3 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** `aggregate` is an instruction and runs in `program` only.

**Scope:** Needs an operator over arrays so a `check` expression or an index key can aggregate. Ordering only after T-15 and T-16.

### T-19. Grouping, `any`/`all` and average not offered

Theme: engine · Priority: P4 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** `aggregate` offers count, sum, min and max only; average was left out because `div` loses digits.

**Scope:** Decide whether to offer grouping, `any`/`all` and average. Ordering only after T-15 and T-16.

## shacl-comparison

### T-20. SHACL Core alone untested for `chinese-wall` and `break-glass`

Theme: shacl-comparison · Priority: P3 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** The SHACL comparison uses `sh:sparql` throughout, so it says nothing about Core.

**Scope:** Express both packages in SHACL Core and compare; needs the pyshacl environment. Ordering only after T-15 and T-16.

### T-21. Which `sh:sourceConstraint` reading follows the W3C text

Theme: shacl-comparison · Priority: P4 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** pyshacl names the constraint and Jena names the shape; the harness reads Jena's reasons from `sh:resultMessage`.

**Scope:** Settle the reading against the W3C text. Ordering only after T-15 and T-16.

### T-22. `setup.sh` install branches unexercised end to end

Theme: shacl-comparison · Priority: P4 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** Venv creation and the pip and Maven fetches have not been run; `--check`, `run.sh` and `scale.py` ran against existing environments.

**Scope:** Run the install branches on a clean machine; this has a real network and disk cost. Ordering only after T-15 and T-16.

## tooling

### T-23. The 21 phase 5 source mutants are not reproducible

Theme: tooling · Priority: P3 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** The 21 mutants of `pkg/shaxon` source were run on a scratch copy and not kept as a script; the authorisation mutants and the engine fixture mutants are kept.

**Scope:** Recreate them as a script and register it as a dormant guard. Ordering only after T-15 and T-16.

### T-24. repoman provenance flag on `CHANGELOG.md`

Theme: tooling · Priority: P4 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** Repoman's provenance check run from the repository root reports `CHANGELOG.md` as changed outside repoman after edits made through the `src/` root's journal.

**Scope:** The fix belongs upstream in gorepoman; until then each flag is sanctioned with that reason. Ordering only after T-15 and T-16.

