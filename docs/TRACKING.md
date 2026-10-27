# Open work register

Version: 0.3.1-p000.r021
Last reviewed: 2026-10-27

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
| T-21 | Which `sh:sourceConstraint` reading follows the W3C text | shacl-comparison | P4 | ☐ | T-15, T-16 |
| T-22 | `setup.sh` install branches unexercised end to end | shacl-comparison | P4 | ◐ | T-15, T-16 |
| T-23 | The 21 phase 5 source mutants are not reproducible | tooling | P3 | ☐ | T-15, T-16 |
| T-24 | repoman provenance flag on `CHANGELOG.md` | tooling | P4 | ☐ | T-15, T-16 |
| T-33 | accessgraph: delegation and hop-bound decisions thinly covered | accessgraph-coverage | P4 | ☐ | — |
| T-34 | accessgraph: one shared reading of the prose rules | accessgraph-coverage | P4 | ☐ | — |
| T-35 | accessgraph: delegation strictness not reviewed by a domain expert | accessgraph-coverage | P4 | ☐ | — |
| T-40 | accessgraph: third-party expected answers not yet used | accessgraph-real-data | P4 | ☐ | — |
| T-42 | jaxrun does not report its step count | tooling | P4 | ☐ | — |
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

### T-21. Which `sh:sourceConstraint` reading follows the W3C text

Theme: shacl-comparison · Priority: P4 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** pyshacl names the constraint and Jena names the shape; the harness reads Jena's reasons from `sh:resultMessage`.

**Scope:** Settle the reading against the W3C text. Ordering only after T-15 and T-16.

### T-22. `setup.sh` install branches unexercised end to end

Theme: shacl-comparison · Priority: P4 · Status: ◐ · Blocks/after: T-15, T-16

**Trigger:** Venv creation and the pip and Maven fetches have not been run; `--check`, `run.sh` and `scale.py` ran against existing environments.

**Scope:** Run the install branches on a clean machine; this has a real network and disk cost. Ordering only after T-15 and T-16. Status on 2026-10-26: the default branch ran end to end in a scratch copy with no environment present (venv created, pyshacl 0.40.1 and rdflib 7.6.0 installed in 23 s, 39 MB, the versions of the recorded comparison), and the earlier baseline of 50 agree, 0 differ, 1 not applicable reproduced. `--check` ran, and the Jena branch exits with a clear message when Maven is missing. Found and fixed: `.venv` and `.jena` were not ignored by git. Outstanding: the Jena fetch through Maven (Maven is not installed here; installing it has a real cost and needs a decision) and the macOS script.

## tooling

### T-23. The 21 phase 5 source mutants are not reproducible

Theme: tooling · Priority: P3 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** The 21 mutants of `pkg/shaxon` source were run on a scratch copy and not kept as a script; the authorisation mutants and the engine fixture mutants are kept.

**Scope:** Recreate them as a script and register it as a dormant guard. Ordering only after T-15 and T-16.

### T-24. repoman provenance flag on `CHANGELOG.md`

Theme: tooling · Priority: P4 · Status: ☐ · Blocks/after: T-15, T-16

**Trigger:** Repoman's provenance check run from the repository root reports `CHANGELOG.md` as changed outside repoman after edits made through the `src/` root's journal.

**Scope:** The fix belongs upstream in gorepoman; until then each flag is sanctioned with that reason. Ordering only after T-15 and T-16.

### T-42. jaxrun does not report its step count

Theme: tooling · Priority: P4 · Status: ☐ · Wave: 11

**Trigger:** T-39 needed the exact step count of a run of 220 million steps. The stand-in gives it, at about 130,000 steps a second, which is 20 minutes or more here. `jaxrun` runs at about 4.7 million steps a second and its machine has the counter (`Machine.Steps()`), but `Profile.Run` does not return it and `jaxrun` does not print it. The measurement used a scratch build with one variable and one line added; its counts matched the stand-in on four known databases.

**Scope:** Expose the count: an exported way to read the step total of a run (a field on a result, or a `RunSteps` beside `Run`), and a `-steps` flag in `jaxrun` that prints it on stderr, as `shaxonrun -steps` does. Add a test that the count equals the stand-in's on a small package. This touches the core runtime API, so it is ordered after layer 3 and checked against T-16, which wants a fixture that pins `steps` exactly.

## accessgraph-coverage

### T-33. accessgraph: delegation and hop-bound decisions thinly covered

Theme: accessgraph-coverage · Priority: P4 · Status: ☐ · Wave: 9

**Trigger:** Review of the accessgraph example. Over 60 generated databases about 6% of decisions rest on a delegation (516 of 8,400) and about 0.4% depend on the hop bound (34 of 8,400). The hop-bound mutants (one hop short, one hop extra) were detected on only 10 of 24 databases, so delegation is the least exercised part of the program.

**Scope:** Tune chain and anchored-request generation in the generator, and add minimum-coverage assertions to the oracle tests (delegated decisions, hop-bound flips, deny without support) so a generator change cannot lower them unnoticed. Re-run the mutants and require each to be detected on a stated minimum share of databases.

### T-34. accessgraph: one shared reading of the prose rules

Theme: accessgraph-coverage · Priority: P4 · Status: ☐ · Wave: 9

**Trigger:** Review of the accessgraph example. The reference implementation and the Jaxson program were written by one author from the same prose rules. A shared misreading would pass every comparison. The 13 hand-built cases, derived from the prose, are the only independent safeguard.

**Scope:** Obtain a second, independent reading of the rules: either expectations derived by hand by another reader from the prose alone, or an encoding in a different engine (a Prolog or Datalog rendering, as the medical example has). Compare it with the oracle on the hand-built database and on generated ones. Record any disagreement as a spec question, not as a bug in either side. Third-party expected answers from the OpenFGA sample stores are a further independent check and are tracked as T-40.

### T-35. accessgraph: delegation strictness not reviewed by a domain expert

Theme: accessgraph-coverage · Priority: P4 · Status: ☐ · Wave: 9

**Trigger:** Review of the accessgraph example. Delegation is implemented literally: a delegator must hold the same role, not merely a role that inherits it, so holding `admin` does not allow delegating `editor`. The rule was implemented and tested but never confirmed by anyone who administers real authorisation systems; most such systems would allow it.

**Scope:** Put the rule in front of a domain reviewer and record the decision with its reason. If inheritance-aware delegation is chosen, the change touches the oracle, the naive fixpoint in the tests, the package's role comparison and the hand-built expectations, which are re-derived by hand, not copied from the oracle.

## accessgraph-real-data

### T-40. accessgraph: third-party expected answers not yet used

Theme: accessgraph-real-data · Priority: P4 · Status: ☐ · Wave: 10

**Trigger:** The OpenFGA sample stores (40 stores, 481 tuples, 556 check assertions and 143 list_objects assertions, pinned in the data provisioning script) carry expected answers written by a third party. Their model (relationships) differs from the example's rules, so each store needs a mapping and some will not map.

**Scope:** Map the stores whose models fit the example (nested groups, roles, custom roles) to accessgraph databases and run their assertions through the reference and the Jaxson program. Record which stores map and which do not, and why. A disagreement is a question about the rules, to be settled by a reviewer, not a bug in either side. Related to T-34, which asks for an independent reading of the prose rules.

