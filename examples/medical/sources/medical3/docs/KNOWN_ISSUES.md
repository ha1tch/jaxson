# Known issues and intentional limits

Version: 0.10.0
Last reviewed: 2026-10-15

Recorded decisions and boundaries. These are not open work; open work is in `TRACKING.md`.

| Limit | Detail | Why it stands |
|-------|--------|---------------|
| Depth bound | `solve/2` searches to depth 6; a failed search proves nothing past it (`solve/3` takes a deeper bound). Failure at the bound is reported `unknown`, never `refuted`. `solve_certified/3` reports whether the bound cut any search (`exhaustive` or `bound_reached`) and `assessment/2` carries it. Tested in `b1`–`b5`, `ce1`–`ce7`. | Termination is guaranteed without loop detection. Removing the bound is a solver redesign. |
| Status model | Weakest-step ordering over five statuses, with proof strength, independent support and conflict reported separately by `assessment/2`; not a calibrated evidence calculus, and sources of different reliability are not weighted. | Calibration needs data the knowledge base does not have. |
| No lift rule shipped | `corroboration_lift/3` is implemented and validated, but every shipped fact has one source (102 facts: 94 `kb`, 8 `curated`), so no rule has anything to act on. | A lift is a policy choice; declare one when corroborated facts exist. |
| Declared shared origin | `kb/lineage.pl` declares `kb` and `curated` one origin, because the curated bridges were authored from the legacy entities. | It is an authoring decision of the tech team; correct it there if wrong. |
| Impact analysis baseline | About 0.55 million inferences per call, set by enumerating the shipped headline conclusions. | It is proportional to the knowledge base, not to the retracted fact. |
| Search cost | Linear in fan-out, in step with the number of derivable conclusions. | Inherent to enumerating conclusions. |
| Validation is structural | A declared rule can still be semantically wrong; validation checks form, not meaning. | The engine cannot judge whether declared semantics are right. |
| Clinical validity | Out of scope for the engine. The facts are illustrative; the tests check inference validity, not medical truth. | A validated engine can still be populated with incorrect facts. |
| Conservative certificate | `bound_reached` is reported whenever a depth-consuming rule could make progress at a cut branch, including when the cut changed nothing. `exhaustive` is checked against a deeper search. | A false `bound_reached` costs a review; a false `exhaustive` would be a wrong absence. |
| `recommend/2` kept | It still lists `needs_review` products beside `supported` ones, with statuses. `recommend_tiers/2` is the three-way policy of contract statement C9. | Kept for compatibility. |
