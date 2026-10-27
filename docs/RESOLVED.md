# Resolved work

Closed items, newest first, with the full detail text as it stood at closure.
Append-only: an entry is never edited after it is written.

## [0.3.1-p000.r019] T-41 — accessgraph: the cost model under-predicts on real-shaped data (v0.3.1-p000.r019, 2026-10-27)

Theme: accessgraph-architecture · closed 0.3.1-p000.r019 · 2026-10-27
**Trigger:** T-39 ran the program on real-shaped data. The cost model of T-38 derived a declared limit of 119,612,662 steps before the run; the program needed 220,106,734, which is 2.21 times the prediction, and stopped with `RESOURCE_ERROR/STEPS`. The model was fitted on synthetic data up to 1,000 users and checked on synthetic data up to 5,000. The program builds its `held` table for every user, while the model's allow feature counts requesters, and the delegation term, requests times delegations, stood in for the cost of scanning each delegator's held entries. Both coincide on synthetic data, where requests are about twice the users. The README and FINDINGS.md say the limit is derived before the run; they are qualified until this item closes.

**Scope:** Replace the features with ones that follow the program: the held table over all users (the grants reachable through each user's subjects, with the role closure of each), the delegation hops over each delegation's holder (hops times the delegator's held entries), and each request's scan of the held entries and of the denies. Recalibrate on the synthetic set. The two Amazon databases, never in the fit, must then be predicted within the margin: measured 220,106,734 and 220,150,664 steps. Keep the held-out synthetic check including 5,000 users, and add a synthetic database with the Amazon proportions of users, requests and delegations. Record the margin the fit needs. If real-shaped data cannot be predicted within a stated margin, say that the limit cannot be derived before the run for data of another shape, and state what the model does cover. Resolved 2026-10-27, in four fits. The features of `cost.py` follow the program's loops: the closures (group and role, one coefficient), the held table over every user, each delegation hop over the delegator's held entries, and each request's scan of the held entries and denies. First fit, on the 55 generated databases: measured over predicted 0.952 to 1.045, margin 1.10. The Amazon databases, never in a fit, were predicted at 0.76 and the program fit its limit (69% used); but `test_policy.py` then failed on a contrived database at the resource depth limit, because the fit had given the resources term a coefficient of zero (generated databases grow every dimension together, so a fit on them alone cannot identify the terms). Second fit: 18 controlled databases that stretch one dimension at a time were added to the calibration set (73 in all): measured over predicted 0.878 to 1.189, margin 1.25 (the worst ratio plus 5%, rounded up to the next 0.05). Third fit: counting distinct role-and-scope pairs in the held table fitted the synthetic databases more tightly (0.93 to 1.10, margin 1.15) and failed on the Amazon databases (2.17 times under, limit exceeded), so it was reverted. The final model is the second fit. Held out: 28 synthetic databases, among them 5,000 users and one with the Amazon data's proportions (11,891,052 steps), measured over predicted 0.83 to 1.07, none over their limit. The two Amazon databases are predicted at 370,299,033 and 370,402,317 steps against 220,106,734 and 220,150,664 measured (0.59), declared 462,873,792 and 463,002,897 (48% used); the program ran with answers equal to the reference on both at the first fit's lower limit of about 318 million, so it fits this one. The model over-predicts there, so its limit is 2.1 times the measured steps. The Amazon databases were looked at after each fit and decided between versions: they are validation data, not an untouched test, and the model is checked against that one real-shaped source. `test_cost.py` checks the two databases when they are present and `check_amazon.py --steps-jaxrun` records their step counts.

Cross-ref: CHANGELOG 0.3.1-p000.r019.

## [0.3.1-p000.r017] T-39 — accessgraph: never run on real-shaped data (v0.3.1-p000.r017, 2026-10-27)

Theme: accessgraph-real-data · closed 0.3.1-p000.r017 · 2026-10-27
**Trigger:** External data was provisioned for the accessgraph example: the Amazon Access Samples (36,063 people, 595,506 edges; permission groups of up to 3,882 members; up to 6,978 direct grants for one person) and the OpenFGA sample stores. Neither has been mapped to the example. The generator's distributions are far more uniform than the real ones, so agreement between engines on generated data says little about skew.

**Scope:** Write an adapter from the Amazon data to an accessgraph database: people become users, permission groups become groups with memberships, direct resource grants stay direct. The data has no roles, resource tree, group grants, denies or delegations; complete those synthetically from the generator's own distributions and record exactly what was added. Build two variants: flat membership as in the data, and membership with the inferred group containments (371 strict containments among groups of at least 20 members, longest chain 5, 146 of them involving the two near-universal groups). Run every engine that still takes part and record the outcome, including disagreements. The second variant is the first test of T-31's round counts against nesting that came from data. After T-36, T-37 and T-38, because the data needs the wider ids, the larger step limit and the larger presets. Added 2026-10-27, from T-36: run the Amazon-scale case (36,063 users) here, on every engine that still takes part, since the generator stops at 9,999 users and this adapter is the only source of that data; record steps and wall time, and the declared limit against the measured steps (the cost model is fitted on synthetic data up to 1,000 users and has not seen data of another shape). Added, from T-32: measure the integrity gate with and without `--limits policy` at that scale. T-36, T-37 and T-38 are closed. Resolved 2026-10-27: `provision/amazon_to_db.py` writes the Amazon data as two accessgraph databases of 36,063 users, 6,048 groups, 21,578 resource leaves and 430,188 grants. The people, 185,318 memberships and 410,188 direct grants come from the data; added, and counted in `record.json`: 16 roles, 601 folders (tree depth 3), 12,000 group grants, 8,000 denies, 28,304 delegations and 10,000 requests. The nested variant uses the 270 covering edges of the 371 inferred containments and drops the 10,134 memberships they imply: every user's reach is identical to flat's, the true group depth is 5 against the limit of 16, and the reference gives identical decisions on both (10,000 of 10,000). The containment edges alone change nothing, because the data's memberships are already closed (true depth 1). Seven evaluators agree with the reference on all 10,000 decisions on both variants: both Go ports, Soufflé compiled and interpreted, CozoDB and the Jaxson program on `jaxrun`; the stand-in is too slow at this size and the Go Datalog library is retired. End to end, flat and nested, seconds: reference 9.4 and 8.7, Go ports 10.2 and 10.3 (line for line) and 7.5 and 8.3 (optimized), Soufflé compiled 4.5 and 4.7, interpreted 6.1 and 5.0, Jaxson 127.7 and 122.6 (the runner itself 47.0 and 47.7; the rest builds and writes a package of about 150 MB), CozoDB 156 and 185. The declared limit derived before the run (119,612,662 steps, flat) was exceeded: the program stopped with `RESOURCE_ERROR/STEPS`. The exact need, from a scratch build of `jaxrun` that prints its step counter (identical to the stand-in on four known databases), is 220,106,734 (flat) and 220,150,664 (nested): 2.21 times the prediction. The model counts requesters where the program builds its held table for every user, and its requests-times-delegations term only imitated the real cost on synthetic data; the fix is T-41. The integrity gate passes both databases with 0 findings, at 2.9 million steps without the depth check and 33.9 million with it (4 s and 7 s), about 1% and 15% of the program's steps, so the depth check is not prohibitive and the decision under T-32 stands. Step-count equality between the stand-in and the real runner is not checked at this size.

Cross-ref: CHANGELOG 0.3.1-p000.r017.

## [0.3.1-p000.r016] T-32 — accessgraph: Shaxon data-integrity layer is missing and is load-bearing (v0.3.1-p000.r016, 2026-10-27)

Theme: accessgraph-architecture · closed 0.3.1-p000.r016 · 2026-10-27
**Trigger:** Review of the accessgraph example. The Jaxson program reads keyed maps and fails with `MISSING_PATH` on a dangling reference, and the reference implementation is defined only on referentially clean databases (`check_refs` raises). No Shaxon layer exists yet, although the medical example has one.

**Scope:** Write the data-integrity layer: dangling members, unknown roles or scopes in grants, delegations from non-users, missing resource parents, and the limits fixed in T-31. Add `inject_defects(db, rng, n)` to the generator returning the mutated database and a ground-truth defect list; the layer's findings must equal that list exactly, including zero findings on a clean database. Mutate the layer (drop a rule) and confirm the check notices. After T-31 because the limits it enforces are defined there. Status on 2026-10-26: stage 1 is done in `examples/accessgraph/integrity.py` (generated package `integrity.json`, tests in `test_integrity.py`): references, enum and uniqueness checks of 22 kinds, each reported at the exact path of the offending element. 200 databases with 769 planted defects give findings equal to the plan, with none missed and none extra; removing any of its 22 rules makes the test fail; clean generated databases pass; the gate costs 2 to 4% of the inference program's steps. A depth check is also expressible, as the same fixed closure rounds the program unrolls plus one detection round, and it equals independent ground truth on 27 databases at and below the true bounds. With limits it costs 30% (1,000 users) to 50% (5,000 users) of the inference program's steps, because it recomputes the closure; stage 2 should share that closure with the program. Outstanding: the constants to enforce (T-31) and the id width (T-37). Update after T-31: the inference program now asserts its own depth limits with one extra round per closure, so a separate depth gate repeats that work; stage 2 can shrink to the id width and to giving the program's refusal the gate's finding paths, or drop its depth part. Update after T-37: the id width is policy and the gate checks it (`Grant.id`, `Deleg.id`), so stage 2 is now only giving the program's refusal the gate's finding paths, or dropping the gate's depth part. Decided 2026-10-27, stage 2 not built: the program asserts its own depth limits (T-31), so the gate's depth check is an opt-in extra (`--limits policy`) that gives located findings and costs 30 to 50% of the program's steps at 1,000 to 5,000 users; it stays that, and the program's refusal keeps its own message without the gate's finding paths. Sharing the closure with the program (about a day) is the alternative; it is reconsidered as a new item only if T-39 shows the gate's depth cost is prohibitive at 36,063 users, which T-39 now measures.

Cross-ref: CHANGELOG 0.3.1-p000.r016.

## [0.3.1-p000.r016] T-36 — accessgraph: scale untested beyond the medium preset (v0.3.1-p000.r016, 2026-10-27)

Theme: accessgraph-coverage · closed 0.3.1-p000.r016 · 2026-10-27
**Trigger:** Review of the accessgraph example. Nothing has been run beyond the `medium` preset, at roughly 200,000 to 400,000 steps against a declared limit of 5,000,000. The unrolled loops grow faster than the data, and where this stops being a demonstration is unknown.

**Scope:** Add a `large` preset (for example 1,000 users) and record steps and wall time on both the stand-in and `jaxrun` before any statement about scale is made. Ordering only after T-31, because fixed policy constants change the cost curve. Widened on 2026-10-26: the large preset in the example is joined by xlarge (5,000 users) and by the Amazon-scale case (36,063 users). The id width is T-37, the declared step limit is T-38, and the run on real-shaped data is T-39; this item stays open until those sizes run on every engine that still takes part. Status on 2026-10-26: the large and xlarge presets, an id-width guard in the generator and scale.py are written and verified on the reference, the stand-in and jaxrun (steps and wall time recorded in the example README; accessgraph 0.0.3, since placed under `examples/accessgraph` as 0.0.4). Outstanding: the Amazon-scale case, which needs T-37 and T-38; and the other engines at the new presets once T-31 fixes the round counts. Resolved 2026-10-27, with a scope change: the Amazon-scale case moves to T-39, because the generator stops at 9,999 users and that data can only come from T-39's adapter; this item had said it stays open until that size runs, and that run is T-39's. The other engines at the new presets ran on 2026-10-27: the reference, both Go ports, Soufflé (interpreted and compiled), CozoDB and the Jaxson program on `jaxrun` on the `large` and `xlarge` presets, seed 0, each required by the harness to equal the reference on every decision, with the stand-in checked the same way by `scale.py`; the Go Datalog library is retired (T-37). Steps and wall time are recorded in the example README and FINDINGS.md, and measured against declared steps for every preset (T-38).

Cross-ref: CHANGELOG 0.3.1-p000.r016.

## [0.3.1-p000.r015] T-38 — accessgraph: declared step limit does not scale with the data (v0.3.1-p000.r015, 2026-10-27)

Theme: accessgraph-architecture · closed 0.3.1-p000.r015 · 2026-10-27
**Trigger:** Review of scale timings for the accessgraph example. The generated program declares a step limit of 5,000,000. Measured on the stand-in, it used 1,751,108 steps at 1,000 users and 8,377,283 at 5,000, so the declared limit is already too small at 5,000 users. Growth is roughly linear (4.8 times the steps for 5 times the users). At 36,063 users the extrapolation is about 60 million steps, which is unmeasured.

**Scope:** Derive the declared limit from the data-size bounds and the policy constants, with a stated margin, instead of a fixed number. Calibrate the formula on measured counts across sizes, then check it on a held-out size. Record measured against declared steps for every preset, and make an exceeded limit fail loudly (`RESOURCE_ERROR/STEPS`) instead of being worked around. After T-31. Status on 2026-10-26: with T-31 done the fixed rounds raise the program's steps to 10,533 at 12 users, 341,942 at 200 and 10,229,845 at 5,000, so 5,000 users exceed the 5,000,000 default by 2.05 times. Prior art, from documentation and not run here: CEL computes a static worst-case cost before evaluation from declared size bounds and also enforces a runtime cost limit, and Kubernetes uses it for custom resource validation (generally available since 1.29); Starlark counts execution steps and can limit them. A cost declared before the run and enforced is therefore not new. What this item can still show is that property in a runtime that also gives identical step counts across implementations, exact decimals and one budget for validation and computation. Resolved 2026-10-27: the declared limit is derived from the data before the run by `cost.py`: features from counts and closure sizes, multiplied by the policy rounds, with the group and role closures sharing one coefficient because they are the same code. `calibrate.py` fits the coefficients on 55 databases from 12 to 1,000 users by non-negative least squares of relative error (measured over predicted 0.913 to 1.116) and sets the margin at 1.20, the worst ratio plus 5% rounded up to the next 0.05. On 27 databases the fit never saw, including 5,000 users, five times the largest size fitted, none exceeded its limit (measured over predicted 0.621 to 1.106); at 5,000 users the model over-predicts, so the limit is 19,782,499 for 10,229,845 measured steps (52% used). Measured against declared steps for the five presets: 85%, 80%, 87%, 80% and 52% used. A limit exceeded fails loudly: both runtimes stop with `RESOURCE_ERROR/STEPS` (`test_cost.py`), and `scale.py` and `check.py` no longer raise it. The model stores a fingerprint of the generated program and refuses to run when the program has changed. A first fit gave the role closure a coefficient of zero and a deep role chain in `test_policy.py` exceeded its limit; tying it to the group closure fixed that. The model is fitted on synthetic data, so the real-shaped data of T-39 has to check it.

Cross-ref: CHANGELOG 0.3.1-p000.r015.

## [0.3.1-p000.r013] T-37 — accessgraph: fixed-width ids cap grants and delegations below 10,000 (v0.3.1-p000.r013, 2026-10-27)

Theme: accessgraph-architecture · closed 0.3.1-p000.r013 · 2026-10-27
**Trigger:** Review of external data for the accessgraph example. Grant and delegation ids are fixed width, four digits (`G0001`, `D0001`), which caps both below 10,000. The Amazon access dataset has 406,661 direct grants and 185,318 permission-group memberships. The chain tie-break (fewest hops, then smallest chain string) compares strings, which equals a numeric comparison only while the width is fixed. The Go Datalog engine encodes a chain as base-10,000 segments in a 64-bit integer; four segments of six-digit ids need 24 digits and overflow, so that engine cannot take wider ids.

**Scope:** Make the id width a stated policy constant, with six digits as the working value, used by the generator, the reference, the Jaxson program and the Go oracle. Check fixed width in the data-integrity layer (T-32) so a mixed-width database is rejected, not mis-ordered. Retire the Go Datalog engine encoding at the wider widths and record why: it overflows, and it already scales quadratically. After T-31, as one of the policy constants. Resolved 2026-10-27: `policy.ID_WIDTH` is 6, so grant and delegation ids are `G000001` and `D000001`, up to 999,999 of each. The generator, the reference (`LimitExceeded` naming `id_width`), the builder, the integrity gate (`Grant.id` and `Deleg.id`; a half-converted database gives exactly its non-conforming findings) and both Go oracles use or enforce it; the other ids stay opaque. `test_policy.py` answers a database of 10,068 grants, past the old four-digit cap, identically on the reference, the stand-in and the real runner; `test_integrity.py` plants 732 defects of 24 kinds in 200 databases with findings equal to the plan, and removing any of its 24 rules is caught. Soufflé and CozoDB agree with the reference at six digits (4,200 decisions each; the mutants are caught as before). The Go Datalog engine's encoding does not: it gave wrong answers on 6 of 6 databases, and four six-digit segments need 24 digits, which does not fit 64 bits, so it is retired and its trial script now refuses. The two Go oracles agree on 42,000 decisions and with the Python reference on 21,680 decisions over 39 Go-generated databases.

Cross-ref: CHANGELOG 0.3.1-p000.r013.

## [0.3.1-p000.r010] T-31 — accessgraph: round counts come from the reference, not from policy (v0.3.1-p000.r010, 2026-10-26)

Theme: accessgraph-architecture · closed 0.3.1-p000.r010 · 2026-10-26
**Trigger:** Review of the accessgraph example (version 0.0.2, since placed under `examples/accessgraph`). The generated package unrolls each closure to a round count that the reference implementation's `bounds()` computes from the database being run: group rounds, role rounds and resource depth. The package is therefore tailored to each dataset: it is regenerated for every database, sized by a measurement made by Python code, and a program run on data deeper than it was sized for would answer with confidence from a truncated closure. (Corrected on 2026-10-26: the first wording said the reference's correctness leaks into the program; it does not, because the reference evaluates the full closure and never uses `bounds()`, so a wrong measurement is caught by comparison on tested data. The hazard is data outside what was measured, where nothing compares.)

**Scope:** Replace the per-database counts with fixed policy constants (maximum group nesting, role inheritance depth, resource depth), the way `MAX_HOPS` is already fixed. The oracle evaluates under the same constants. A database that exceeds a constant is rejected, never silently truncated. This is the cost of totality in Jaxson shown on a real domain and should be written up as such in the example's README. Resolved 2026-10-26: `policy.py` declares `GROUP_ROUNDS` 16, `ROLE_ROUNDS` 8, `RES_DEPTH` 6 and `MAX_HOPS` 3, chosen from measured depths (12 group rounds at 5,000 users, 5 role rounds, 6 resource levels by construction). The generated program is the same for every database (one distinct program over 24 databases of three sizes). The reference refuses a deeper database (`LimitExceeded`), the generator produces only databases within the limits (none of 174 outside; no seed needed a redraw), the builder refuses one, and the program itself asserts with one extra round per closure, failing with `ASSERTION_FAILED` on the stand-in and the real runner. At each limit exactly, the database is accepted and answered like the reference; one past it, all four refuse; built without the assertion it is answered wrongly, which is what the assertion prevents. Cost: every database pays for the full rounds, raising steps from 5,336 to 10,533 at 12 users (+97%) and from 8,377,283 to 10,229,845 at 5,000 users (+22%). Fault planting: round mutants left silent are caught on 15, 13 and 17 of 24 databases; the asserted forms are refused on all 24. The README write-up is deferred, by decision, to the final README update.

Cross-ref: CHANGELOG 0.3.1-p000.r010.

## [0.3.1-p000.r008] T-20 — SHACL Core alone untested for `chinese-wall` and `break-glass` (v0.3.1-p000.r008, 2026-10-26)

Theme: shacl-comparison · closed 0.3.1-p000.r008 · 2026-10-26
**Trigger:** The SHACL comparison uses `sh:sparql` throughout, so it says nothing about Core.

**Scope:** Express both packages in SHACL Core and compare; needs the pyshacl environment. Ordering only after T-15 and T-16. Resolved 2026-10-26: both are expressible in SHACL Core alone (`chinese-wall.core.shapes.ttl`, `break-glass.core.shapes.ttl`, run by `core_cases.py` through pyshacl 0.40.1 with advanced features off, after an audit that no `sh:` term is outside Core). They agree with Shaxon on verdict and reasons on the 15 case files and on 4,100 random trails per package, and with the SHACL-SPARQL versions on 800 each. The one difference is that Shaxon refuses a request for an unknown company (`EXECUTION_ERROR`) and both SHACL versions allow it. Recorded in `examples/shacl/authz/README.md`, Core only, and in the dormant guards table.

Cross-ref: CHANGELOG 0.3.1-p000.r008.

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
