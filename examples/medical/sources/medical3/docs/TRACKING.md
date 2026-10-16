# Tracking register

Version: 0.10.0
Last reviewed: 2026-10-15

Open actionable items only. Closed items are in `RESOLVED.md`; intentional limits are in
`KNOWN_ISSUES.md`. Status legend: ✓ done · ◐ partial · ☐ not started · ✗ dropped.

| ID | Summary | Theme | Priority | Status | Blocks |
|----|---------|-------|----------|--------|--------|
| T-14 | Schema guards are a closed vocabulary of four, not knowledge | Engine | P3 | ☐ | — |
| T-21 | No independent oracle for the solver's derivations | Verification | P2 | ☐ | — |

## Engine

### T-14 Schema guards are a closed vocabulary of four, not knowledge
Theme: Engine · Priority: P3 · Status: ☐ · Blocks/after: —

Trigger: reviewer, on declarative completeness. Schema bodies may contain `guard(G)` goals,
but `G` is limited to `distinct/2`, `before/2`, `is_product/1` and `entity_domain/2`,
implemented in `solve.pl` and listed in `vocab.pl`. A new kind of constraint needs an engine
edit. Done when guards are declared as knowledge, evaluated by a generic evaluator, and
validated (unknown guard, unbound variable, non-terminating guard) before use.

## Verification

### T-21 No independent oracle for the solver's derivations
Theme: Verification · Priority: P2 · Status: ☐ · Blocks/after: —

Trigger: reviewer, on independent verification of reasoning correctness. The conformance and
contract tests check the solver against hand-written expectations and properties, and the
impact analysis against its previous algorithm, but not the derivations themselves against a
second implementation. Done when a separate saturation-based derivation of the headline
conclusions is compared with the solver on the shipped knowledge and on the fixtures, using
the bound certificate to know where the solver may legitimately differ.

## Dormant guards

Verifications that do not run in the default suite (`swipl bench/bench.pl`).

| Guard | Gating condition | Hardware | Canonical invocation | Last exercised |
|-------|------------------|----------|----------------------|----------------|
| `bench/scale.pl` | Not in the default suite | None; inference counts are machine-independent, milliseconds are not | `swipl bench/scale.pl` | 2026-10-15, x86_64-linux, swipl 9.0.4, 0.10.0 |
| `compare.sh` | Not in the default suite | None | `sh compare.sh` | 2026-10-15, x86_64-linux, swipl 9.0.4, 0.10.0 |
