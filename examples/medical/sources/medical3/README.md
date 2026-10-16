# medical3

A symbolic inference engine over a small medical knowledge base, built as an experiment in
separating three things: the knowledge being reasoned about, the rules governing valid
inference, and the evidence behind a conclusion. The medical facts are illustrative. The
tests check inference validity, not clinical truth.

## Layout

| Path | Role |
|------|------|
| `kb/*.pl` | Domain knowledge (`tosid/5`), curated bridges, extension vocabulary |
| `engine/vocab.pl` | Relation types, composition rules, derivation schemas, status ranks |
| `engine/validate.pl` | Static validation of the vocabulary; errors disable the offending rule |
| `engine/ingest.pl` | `tosid/5` and bridges into `rel(S,P,O,Evidence)`; names resolved only here |
| `engine/evidence.pl` | Independence of evidence: origins per fact and per proof, by declared lineage |
| `kb/lineage.pl` | Declared source lineage |
| `engine/solve.pl` | Proof-producing solver, `assess/3`, `retract_impact_set/2`, `recommend/2` |
| `docs/` | Tracking register, resolution record, known limits, semantic contract |
| `tools/release_check.pl` | Release hygiene: documents, register, guards and every executable check |
| `bench/` | Positive, negative, adversarial, Experiment 1 and conformance tests |
| `legacy/` | The original coordinator, kept as a baseline |

## Declarative language

- Relations: `rel_pred/2`, `kind(base|derived)`, `polarity(...)`.
- Axioms: `symmetric(P)`, `transitive(P)`, `inverse(P,Q)`, `subrel(P,Q)`, `defeats(D,T)`.
- Composition: `compose_rule(P1,P2,P3)`; nothing composes unless declared.
- Schemas: `schema(Name, Head, Body, StatusCeiling)` with a closed guard vocabulary.
- Evidence: each stored fact carries one `ev(Source,Strength,Detail)` per asserting source.
  Sources are merged, never dropped; identical assertions are not double-counted.
- Defeat: `assess/3` weighs the best proof against declared defeaters and returns
  `supported`, `conditional`, `conflicted`, `refuted` or `unknown`.

## Running

    swipl bench/bench.pl     # full suite, exit 0 only if everything passes
    swipl demo.pl            # walkthrough
    sh compare.sh            # Experiment 1: legacy coordinator against this engine
    swipl bench/scale.pl     # cost on synthetic rings, chains and fan-outs
    swipl tools/release_check.pl   # release hygiene (add --record to stamp a release)

## Conformance suite (`bench/conformance.pl`)

Every fixture is knowledge only; no engine file is edited by any of them.

| Kind | What it establishes |
|------|---------------------|
| `cf_symmetric` | Symmetry and subrelation work, combine, and do not leak to undeclared relations |
| `cf_transitive` | Closure over chains; no backward closure; cycles terminate, yield no reflexive proof and are reported at ingest |
| `cf_inverse` | Inverse correctness, and composition with transitive closure |
| `cf_compose` | Composition over stored and derived facts; existing schemas see new relations |
| `cf_validate` | All validator error classes (relations, axioms, composition, schemas, guards, property maps) are rejected, disabled whole, and isolated from other rules |
| `cf_evidence` | Multi-source merge, strongest-source status, no double counting |
| `cf_defeat` | Conflicted and refuted outcomes; a defeater that is itself derived through subrel; refuted products are not recommended; unrelated conclusions unchanged |
| `cf_alternatives` | Alternative proofs preserved, including ones of different strength (strongest assessed); impact analysis with one, then several simultaneous retractions |
| `cf_depth` | Seven hops derive at the default bound; eight do not; a deeper explicit bound recovers them; failure at the bound is `unknown`, never `refuted` |
| `cf_warnings` | The five validator warnings fire, and a warning never disables its rule |
| `cf_closure` | Transitive closure follows subrelation and symmetric edges; symmetric plus transitive acts as an equivalence; no reflexive links |
| `cf_cycle_impact` | A cycle is reported at ingest; impact analysis over a cycle with alternative exits, one and several retractions |
| `cf_pattern` | A new derived relation, its schema and a defeat axiom declared as knowledge only: derived, ceilinged, explained, defeats a competing claim, retractable |
| `cf_independence` | Shared lineage counted once, undeclared sources independent, one source asserting twice, proof support as weakest leaf, assertion-level lineage overriding source-level, contradictory lineage reported, rejected declarations disabled whole |
| `cf_lift` | A declared lift raises a fact and the proofs through it; thresholds; shared lineage and repetition do not lift; no rule means no change; every invalid rule class is rejected and disabled |
| `cf_impact` | Impact analysis equals the previous algorithm, kept as an oracle, on every single and paired retraction |
| `cf_certificate` | `exhaustive` is sound against a deeper search; a cut search reports `bound_reached`; a deeper explicit bound resolves it |
| `cf_assessment` | The record's parts for supported, contested, defeated and absent conclusions; absence is never refutation |
| `cf_recommend` | Tier membership, withheld products carry their defeating proofs, tiers partition the candidates and match `recommend/2` |
| `contract` | The properties of `docs/CONTRACT.md` over every shipped conclusion: absence, refutation, ceilings, defeat and retraction monotonicity, determinism, bounded independence |
| `cf_new_relation` | New relation classes (`cf_chelates`, `cf_synergises`) participate in existing interference and synergy reasoning through declarations alone |

## Experiment 1 (measured, `sh compare.sh`)

The same knowledge-only additions are given to the legacy coordinator and to this engine.

| Case | Legacy coordinator | medical3 |
|------|--------------------|----------|
| Iron Fumarate addresses fatigue | derived | derived |
| Zinc addresses hair_loss | missing | derived |
| Folic Acid addresses glossitis | missing | derived |
| B12 product for glossitis (must be absent) | present (wrong) | absent |
| Vitamin K, no product exists | suggested with confidence high | conditional candidate, no product |

Intended conclusions derived: 1 of 3 for the legacy coordinator, 3 of 3 for medical3.

## Assessment, certificate and recommendation tiers

`assessment/2` reports an outcome as separate parts: `status` (as `assess/3`, which is
unchanged), `proof_strength` (supported, conditional or none), `independent_support`,
`conflict` (none, contested, defeated), `defeaters` and `completeness`. A defeater cannot
contest a conclusion that has no proof: absence is not refutation.

`solve_certified/3` returns the proofs with a certificate. `exhaustive` means no search branch
was cut by the depth bound where a rule could have continued, so a deeper bound finds the same
proofs; `bound_reached` means proofs beyond the bound may exist. The probe is conservative:
`exhaustive` is checked against a deeper search, while `bound_reached` may be reported when
the cut changed nothing.

`recommend_tiers/2` sorts candidates into `supported`, `needs_review` and `withheld`, and
lists the defeating proofs for withheld ones. `recommend/2` is unchanged. What each output
entitles a consumer to conclude is in `docs/CONTRACT.md`.

## Independence of evidence

Several sources agreeing is not the same as one claim repeated, and two sources that
derive from one origin are one piece of evidence. The engine counts the independent
origins behind a fact, and behind a proof (its least-corroborated leaf).

An evidence entry maps to an origin: a `lineage(Group)` entry in the assertion's own detail,
else the group declared by `source_lineage(Source, Group)` in `kb/lineage.pl`, else the source
label. Independence is the number of distinct origins. `independent_support/2` gives the best
support over a goal's proofs; `explain_support/1` prints the origins behind each leaf.

A status changes only through a declared rule `corroboration_lift(MinOrigins, From, To)`: a
fact at status `From` with at least `MinOrigins` independent origins among entries at least as
strong as `From` is read as `To`. The rule is applied once, never chained, and proofs inherit
it through weakest-step propagation. No rule is shipped, so no status changes. Validation
rejects a source with two lineage groups, contradictory assertion-level lineage (reported at
ingest), and a lift with a threshold below two, an unknown status, a non-positive `From`, or
a target that is not stronger; a rejected declaration is disabled whole.

Decision record: a second policy that took the source label as the origin was built and
compared against this one, behind a switch. It overcounted sources that share an origin:

| Case | Origins under `label` | Origins under `lineage` |
|------|-----------------------|-------------------------|
| Two sources of one lineage (fact) | 2 | 1 |
| Two undeclared sources (fact) | 2 | 2 |
| One source asserting twice (fact) | 1 | 1 |
| Proof over a shared-lineage leaf | 2 | 1 |
| Shipped knowledge base (102 facts, all single-source) | no difference | no difference |

Lineage was kept and the label policy and its switch were removed.

## Scale (measured, `swipl bench/scale.pl`)

Synthetic graphs over `precursor_of`, one product containing the first entity. Cost is in
inferences (machine-independent). Fan is one hub with N spokes.

| Shape | Entities | Ingest ms (of which cycle check) | Failing query (inf) | Open query: proofs / inf | Product query: proofs / inf | Impact: lost / inf (ms) |
|-------|----------|---------------------------------|---------------------|--------------------------|-----------------------------|-------------------------|
| ring | 12 | 1 (cycle check 0) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (39 ms) |
| ring | 50 | 4 (cycle check 1) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (55 ms) |
| ring | 200 | 17 (cycle check 6) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (38 ms) |
| ring | 1000 | 92 (cycle check 68) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (31 ms) |
| chain | 12 | 1 (cycle check 0) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (29 ms) |
| chain | 50 | 2 (cycle check 0) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (30 ms) |
| chain | 200 | 5 (cycle check 2) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (29 ms) |
| chain | 1000 | 69 (cycle check 55) | 217 | 7 / 234 | 6 / 253 | 6 / 549122 (30 ms) |
| fan | 12 | 1 (cycle check 0) | 443 | 12 / 470 | 12 / 524 | 1 / 549343 (28 ms) |
| fan | 50 | 2 (cycle check 0) | 1697 | 50 / 1800 | 50 / 1854 | 1 / 552383 (29 ms) |
| fan | 200 | 5 (cycle check 2) | 6647 | 200 / 7050 | 200 / 7104 | 1 / 564383 (31 ms) |
| fan | 1000 | 18 (cycle check 7) | 33047 | 1000 / 35050 | 1000 / 35104 | 1 / 628383 (40 ms) |

- Search cost is set by the depth bound, not the graph: ring and chain queries cost the same
  at 12 and at 1000 entities.
- Search cost grows linearly with fan-out, in step with the number of derivable conclusions.
- Impact analysis enumerates the headline conclusions once and groups the proofs, so it
  carries a fixed baseline (about 0.55 million inferences on the shipped knowledge base) and
  grows slowly with fan-out: 0.63 million inferences, 40 ms, at 1000 spokes. Solving each
  conclusion separately cost 33.7 million inferences, 1.9 s, for the same case.
- Ingest is near-linear. Cycle detection is Tarjan's algorithm; an earlier per-node search
  made 1000-entity ingest cost about 2 s.

## Known limits

The solver depth bound (6) means a failed search is not a proof of absence at greater depth.
`solve_certified/3` reports whether the bound cut any search, `solve/3` accepts a deeper
bound, and the suite documents the behaviour (`cf_depth`, `cf_certificate`). Status
propagation is a weakest-step ordering, not a calibrated evidence calculus, and independent
corroboration raises a status only through a declared lift rule, of which none is shipped. A
declared rule can still be semantically wrong; validation checks structure, not meaning. The
full list is in `docs/KNOWN_ISSUES.md`.

Copyright (c) 2026 haitch <h@ual.li>. Licensed under Apache-2.0
(https://www.apache.org/licenses/LICENSE-2.0).
