# Semantic contract

Version: 0.10.0
Last reviewed: 2026-10-15

What each kind of output entitles a consumer to conclude, and what it does not. Each
statement is checked by tests written from the statement as properties over every shipped
conclusion, not as examples. The facts are illustrative and clinically unvalidated; the
contract is about inference, not medicine.

## Statements

| # | Statement | Verified by |
|---|-----------|-------------|
| C1 | A conclusion with no proof within the bound has status `unknown`. `unknown` never means refuted, even when a defeater exists. | `k1`, `as4`, `as6` |
| C2 | `refuted` requires a declared defeater of conditional or higher strength. It is never produced by absence of proof. | `k1`, `d3`, `as3` |
| C3 | No proof is stronger than its weakest step: each schema ceiling, and the strongest evidence entry of each leaf fact. A declared `corroboration_lift` rule is the only exception; none is shipped. | `k2`, `ev1`–`ev4`, `lf0`–`lf13` |
| C4 | Adding a defeater never improves the status of a conclusion. | `k3` |
| C5 | Removing a fact never creates a conclusion. Schemas contain no negation. | `k4` |
| C6 | Ingestion is deterministic: running it again gives identical facts and identical issues. | `k5` |
| C7 | Independence is bounded: every fact has between one origin and as many origins as evidence entries, and a proof is no better corroborated than its least-corroborated leaf. | `k6`, `in1`–`in7`, `al1`–`al4` |
| C8 | Completeness: `exhaustive` means no search branch was cut by the depth bound where a rule could have continued, so a deeper bound finds the same proofs. `bound_reached` means proofs beyond the bound may exist; it does not mean they do. | `ce1`–`ce7` |
| C9 | Recommendation tiers: `supported` needs proof strength supported and no defeater; `needs_review` is conditional or contested; `withheld` is defeated and lists the defeating proofs. Each candidate row is in exactly one tier, and `supported` plus `needs_review` equals `recommend/2`. | `rc1`–`rc7`, `d4`, `d5` |
| C10 | `may_address` is a candidate relation, not a diagnosis. Membership of any tier is not an endorsement. | `k7`, `rc1`–`rc7` |
| C11 | Impact analysis is relative to the proofs the solver can discover within the bound. | `im1`–`im3`, C8 |

## What a consumer may conclude

| Output | May conclude | May not conclude |
|--------|--------------|------------------|
| `may_address` in tier `supported` | The knowledge base lists the product for the concept and nothing declared opposes it | That it is appropriate for a person, or medically validated |
| `may_address` in tier `needs_review` | A candidate exists, supported only by deficiency reasoning, or contested by a defeater of equal strength | That it is recommended |
| `may_address` in tier `withheld` | A declared defeater outweighs the support; the defeating proofs are listed | That the product is unsafe, only that the knowledge base opposes it here |
| `unknown` with `exhaustive` | Nothing derives this under the declared rules | That it is false |
| `unknown` with `bound_reached` | Nothing was found within the bound | That nothing exists beyond it |
| `refuted` | A declared defeater outweighs the best proof | That the conclusion is false in the world |
| `duplicates`, `interferes_with`, `boosts` | The declared relations imply the stated interaction | A clinical effect size or advice to act |
| Independent support of N | N distinct origins stand behind the weakest leaf of the best proof | That N sources agree on the conclusion; agreement is not tested |
