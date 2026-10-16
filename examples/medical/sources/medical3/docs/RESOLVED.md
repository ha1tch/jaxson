# Resolution record

Closed items, full text as at closure, newest first. Append-only. Versions follow the
checkpoint numbers: 0.N.0 is checkpoint N. All dates are 2026-10-15.

## Closed in 0.10.0

### T-18 No release-hygiene script — closed 0.10.0, 2026-10-15
Nothing mechanical checked that the register held no done items, that `RESOLVED.md` only
grew, that headers matched `VERSION`, or that the dormant guards had been exercised.
Resolved: `tools/release_check.pl` checks those, and runs the default suite, `scale.pl` and
`compare.sh`; `--record` stamps the documents, the guard record and the baseline kept in
`docs/snapshot/`. It does not compare the README's generated tables with fresh output,
because they include milliseconds.

### T-22 No semantic contract — closed 0.10.0, 2026-10-15
The meaning of the outputs was implicit. Resolved: `docs/CONTRACT.md` states eleven
properties (C1–C11) and what a consumer may and may not conclude from each kind of output.
They are checked as properties over every shipped conclusion (`k1`–`k7`) plus the
certificate and tier tests. Three deliberate engine breakages (a composed proof taking its
strongest step, absence turning into refutation, the certificate probe disabled) each fail at
least one of these tests. The first version of the ceiling property missed the first
breakage; it now covers every relation and a fixture with mixed-strength steps.

### T-23 Conflicted candidates mixed into the recommendation list — closed 0.10.0, 2026-10-15
`recommend/2` returned supported, conditional and conflicted candidates together. Resolved:
`recommend_tiers/2` sorts rows into `supported`, `needs_review` and `withheld`, and lists
the defeating proofs for withheld rows; every candidate row is in exactly one tier and
`supported` plus `needs_review` equals `recommend/2`. Evidence: `rc1`–`rc7`.

### T-24 Proof strength, evidence support and conflict collapsed into one status — closed 0.10.0, 2026-10-15
Resolved: `assessment/2` reports them as separate parts beside the unchanged `assess/3`
status; a defeater cannot contest a conclusion with no proof. Evidence: `as1`–`as8`.

### T-25 The depth bound was invisible to consumers — closed 0.10.0, 2026-10-15
A failed search could not be told from a truncated one. Resolved: `solve_certified/3` returns
the proofs with `exhaustive` or `bound_reached`; `assessment/2` carries it. `exhaustive` is
checked against a deeper search on every relation of the shipped knowledge and the fixtures.
Evidence: `ce1`–`ce7`.

## Closed in 0.9.0

### T-07 Evidence independence — closed 0.9.0, 2026-10-15
A fact's status ignored how many independent sources stood behind it, so repetition and
corroboration were indistinguishable. Resolved: `engine/evidence.pl` maps each evidence entry
to an origin (declared lineage group, assertion-level `lineage(G)` taking precedence, else the
source label) and counts distinct origins per fact and per proof (weakest leaf). A declared,
validated `corroboration_lift/3` rule can raise a status; none is shipped, so no status
changes. A label-only policy was built and compared, overcounted shared-origin sources, and
was removed. Evidence: tests `in1`–`in7`, `al1`–`al4`, `lf0`–`lf13`.

### T-16 Impact analysis cost — closed 0.9.0, 2026-10-15
`retract_impact_set/2` enumerated every headline conclusion and then solved each one again,
making it quadratic in fan-out (33.7 million inferences, 1.9 s at 1000 spokes). Resolved:
one enumeration grouped by conclusion (`proof_table/1`). The same case now costs 0.63 million
inferences, 40 ms. Evidence: `im1`–`im3` compare against the previous algorithm, kept as a
test oracle, on every single retraction and on retraction pairs.

### T-17 No tracking register — closed 0.9.0, 2026-10-15
The repository had no register, resolution record, limits document or dormant-guard table.
Resolved: `docs/TRACKING.md`, `docs/RESOLVED.md`, `docs/KNOWN_ISSUES.md` and `VERSION`.

## Closed in 0.8.0

(No items closed.)

## Closed in 0.6.0

### T-08 Alternative proofs preserved — closed 0.6.0, 2026-10-15
Alternative derivations, including ones of different strength, are all found and the strongest
is assessed. Evidence: `alt1`–`alt4`, `m1`, `m2`.

### T-09 Impact analysis tests — closed 0.6.0, 2026-10-15
Impact analysis tested with alternative routes, a cycle with two exits, and one and several
simultaneous retractions. Evidence: `alt2`–`alt4`, `y1`–`y6`, `c6`, `pt8`.

### T-10 New relation class without a solver edit — closed 0.6.0, 2026-10-15
Two relations (`cf_chelates`, `cf_synergises`) and a new derived relation with its schema and
defeat axiom (`cf_may_aggravate`) are declared as knowledge only and participate in existing
reasoning. Evidence: `nr1`–`nr5`, `pt1`–`pt8`.

### T-11 Termination and scale quantified — closed 0.6.0, 2026-10-15
Cycles terminate; `bench/scale.pl` measures ring, chain and fan-out graphs at 12 to 1000
entities. Search cost is set by the depth bound, linear in fan-out.

### T-12 Adversarial axiom combinations — closed 0.6.0, 2026-10-15
Subrel with transitive, symmetric with transitive, defeat through composition and through a
derived defeater. Evidence: `cl1`–`cl8`, `df1`, `d3`.

### T-19 Closure ignored subrel and symmetric edges, and manufactured reflexive links — closed 0.6.0, 2026-10-15
Transitive closure followed only stored facts, so X inside Y inside-subrel-of part_of did not
close, a symmetric and transitive relation did not behave as an equivalence, and a cycle
produced "K linked K". Resolved: closure takes one-step edges from stored facts, stored
converses of symmetric relations and stored facts of declared subrelations, and refuses a
reflexive conclusion. Evidence: `cl1`, `cl4`, `cl7` fail on the previous solver.

### T-20 Quadratic cycle detection at ingest — closed 0.6.0, 2026-10-15
Cycle detection ran a search from every node (2.1 s at 1000 entities). Resolved: Tarjan's
strongly connected components, one linear pass per transitive relation (about 0.09 s).

## Closed in 0.5.0

### T-03 Dedicated tests for each axiom — closed 0.5.0, 2026-10-15
Symmetry in both directions, transitive closure, inverse, subrel, composition over new
relations, cycles, and behaviour at the depth bound. Evidence: `s1`–`s5`, `t1`–`t5`,
`i1`–`i3`, `c1`–`c5`, `b1`–`b5`.

### T-04 Reject invalid declarations — closed 0.5.0, 2026-10-15
Every validator error class and warning is exercised; a rejected declaration is disabled whole
and isolated from other rules. Evidence: `v0`–`v23`, `w1`–`w7`.

### T-13 Comparison tooling — closed 0.5.0, 2026-10-15
`compare.sh` labels corrected and its measured result recorded: legacy coordinator derives 1
of 3 intended conclusions plus a wrong and a spurious one; this engine derives 3 of 3.

## Closed in 0.4.0

### T-01 Run the benchmarks on a real runtime — closed 0.4.0, 2026-10-15
SWI-Prolog 9.0.4 installed in the sandbox; the full suite runs.

### T-02 Keep the earlier safeguards while adding mechanisms — closed 0.4.0, 2026-10-15
The 20 original tests pass unchanged.

### T-05 Keep evidence from multiple sources — closed 0.4.0, 2026-10-15
Sources are merged per fact, the strongest source sets the status, identical assertions are
not counted twice. Evidence: `ev1`–`ev4`.

### T-06 Adversarial tests for defeaters — closed 0.4.0, 2026-10-15
Conflicted and refuted outcomes, product exclusion, unrelated conclusions unchanged. Evidence:
`d1`–`d6`.
