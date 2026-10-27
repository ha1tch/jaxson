# Wave plan

**Wave 0 — Extract, don't change (≈ 0.0d, added 2026-10-14).** Phase 0 of the implementation plan (section 3): extract the existing Jaxson code into `pkg/jaxson` without changing behaviour. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 1 — Extension points Shaxon needs (≈ 0.0d, added 2026-10-14).** Phase 1 of the implementation plan (section 4): add the extension points Shaxon needs to `pkg/jaxson`, and no others. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 2 — Shaxon's static layer (≈ 0.0d, added 2026-10-14).** Phase 2 of the implementation plan (section 5): parse and statically validate shapes, indices, relations and computes, independent of any input. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 3 — Shape evaluation (≈ 0.0d, added 2026-10-14).** Phase 3 of the implementation plan (section 6): evaluate shapes against a document and report violations; carries the `extends` narrowing work (G11) and the step-cost model. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 4 — Indices, relations, targets, validate (≈ 0.0d, added 2026-10-14).** Phase 4 of the implementation plan (section 7): build indices and relations over real data, resolve the five target kinds, and deliver `validate` and `check`. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 5 — shaxon.Run, version, pipeline order (≈ 0.0d, added 2026-10-14).** Phase 5 of the implementation plan (section 8): make Shaxon a profile over the one jaxson pipeline, with `shaxon.Run`, version handling and the stage order the core states. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 6 — Fixtures and conformance (≈ 0.0d, added 2026-10-14).** Phase 6 of the implementation plan (section 9): write the conformance fixtures from the core specification and judge the engine with them. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 7 — Public API, Session, build package (≈ 0.0d, added 2026-10-14).** Phase 7 of the implementation plan (section 10): expose the public API, the `Session` helper and the builder package. Converted from TRACKER.md on 2026-10-14; item statuses are as verified that day, and no item has a register ticket yet.


**Wave 8 — accessgraph: policy limits and data integrity (≈ 2.0d, added 2026-10-25).** Architectural weakness found in review of the accessgraph example: the generated package takes its round counts from the reference implementation, per database, so the program under test inherits the checker's correctness. The remedy is to fix the limits as policy and let a Shaxon layer reject data that exceeds them, which makes the data-integrity layer a precondition of the inference and not an optional extra. T-31 first; T-32 depends on the constants it defines. Waves 0 to 7 are the implementation phases and are unaffected.


**Wave 9 — accessgraph: coverage, independence and scale (≈ 3.0d, added 2026-10-25).** Thin spots found in review of the accessgraph example: delegation is the least exercised part of the program, the reference and the package share one reading of the rules, the literal delegation rule has had no domain review, and nothing has run beyond the medium preset. None of these blocks the others. T-36 follows T-31 because fixed policy constants change the cost curve. Wave 8 comes first as the structural fix; this wave is the verification debt that remains once it lands.


**Wave 10 — accessgraph: real-shaped data and scale (≈ 4.0d, added 2026-10-26).** Found while provisioning external data for the accessgraph example. The real data (36,063 people, 595,506 edges, groups of up to 3,882 members) breaks two assumptions the example makes at every size: fixed four-digit ids that cap grants and delegations below 10,000, and a declared step limit that 5,000 users already exceed (8.4 million steps against 5 million). T-37 and T-38 fix those as policy constants, after T-31 defines the rest. T-39 is the run on real-shaped data and follows T-36, which stays in wave 9 because an item cannot move once attached. T-40 is independent of the others and relates to T-34. Wave 8 comes first; this wave is what turns the example from a generated demonstration into one checked against data it was not written for.


**Wave 11 — accessgraph: findings from real-shaped data (≈ 1.5d, added 2026-10-27).** Found by T-39, the first run of the example on data it did not generate. The derived step limit was wrong by a factor of 2.21 (T-41): the model counted requesters where the program builds its held table for every user, and it must be rebuilt on features that follow the program and checked on the Amazon databases as data it never saw. Measuring that needed a scratch build of jaxrun that prints its step counter (T-42), which is a small change to the core runtime API and waits for the same layer as T-16.

