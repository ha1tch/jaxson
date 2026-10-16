# medical

Two Prolog medical knowledge systems re-implemented on Jaxson and Shaxon, and compared with the
originals' own output. The medical facts are illustrative; nothing here is clinical advice.

## Layout

| Path | Role |
|------|------|
| `sources/medical/` | The first coordinator, as supplied |
| `sources/medical3/` | medical3 0.10.0 (checkpoint 10), as supplied |
| `medical/spike.py` | Converts the 46 `tosid/5` facts to JSON and generates the Jaxson program for the recommendation pipeline |
| `medical/compare.py`, `medical/expected/` | Compares that program with the original system's real output, case by case |
| `medical/shaxon_kb.py` | Shaxon shapes, indices and uniqueness checks over the knowledge base |
| `medical3/export.pl` | Dumps medical3's ingested facts, evidence, lineage, active vocabulary and its own `assess/3`, `assessment/2` and `recommend_tiers/2` outputs |
| `medical3/gen.py` | Compiles the vocabulary (schemas, axioms, composition rules) into a bottom-up Jaxson evaluator, bounded to the solver's depth |
| `medical3/check.py` | Compares every part of every assessment, and every tier row, with the engine's |
| `medical3/overlays.py` | Five knowledge-only overlays applied in Prolog and compared again, plus contract properties C4, C5, C6 |
| `medical3/cyclecheck.pl` | Reproduces the cycle defect described below |
| `medical/medical_pkg.json`, `medical/kb_shaxon.json`, `medical3/data/` | Generated packages and exported data, kept so the Jaxson side runs without Prolog |

## Running

From `src/jaxson-shaxon-v0.3.1`, build the runners, then run the example:

    go build -o /tmp/jaxrun ./cmd/jaxrun && go build -o /tmp/shaxonrun ./cmd/shaxonrun
    cd ../../examples/medical && JAXRUN=/tmp/jaxrun SHAXONRUN=/tmp/shaxonrun ./run_all.sh

Python 3 is needed throughout and SWI-Prolog 9 for the medical3 steps. Without Prolog, run
`jaxrun medical3/data/m3_pkg.json` on the included package.

## Results

| Check | Result |
|-------|--------|
| First coordinator, five patient cases | Identical to the Prolog output (recommendations in order, interactions, cost), within 80,000 steps |
| First coordinator, Shaxon over its knowledge base | 24 findings: 7 dangling references, 17 missing required fields |
| medical3, shipped knowledge base | 94 of 94 assessments identical in every part, 58 of 58 tier rows identical, proof counts equal at both bounds |
| Overlays: defeaters, a new relation and schema declared as knowledge, multiple sources with lineage, a chain longer than the bound | Identical |
| Overlay: a cycle in a transitive relation | One conclusion differs, see below |
| Contract C4, C5, C6 | Hold on the Jaxson evaluator |

The evaluator derives, in six bottom-up rounds that each see only the previous round's facts, the
conclusions, their strength, independent support, proof count and completeness. It is a different
algorithm from medical3's top-down solver and agrees with it on every derivation count.

## A defect in medical3 found by the comparison (present in 0.10.0)

With a cycle in a transitive relation, an open query (object unbound) finds a reflexive proof, because the
guard `S \== O` is tested before `O` is bound. With both ends given there is no proof. The headline list
therefore contains a phantom conclusion ("a Tryptophan tablet supplies_precursor_of Tryptophan", strength
`none`). The evaluator does not report it.

## Limits

The vocabulary is compiled when the package is generated, so a knowledge-only change means regenerating it.
Completeness is decided by comparing proof counts at two bounds, not by the engine's probe. A defeater with
proofs of several strengths is reported at its best one. Not built: corroboration lift, proof terms and
`explain`, impact analysis, the vocabulary validator, ingest (name normalisation cannot be written in
Jaxson), and the scale suite.

Copyright (c) 2026 haitch <h@ual.li>. Licensed under Apache-2.0 (https://www.apache.org/licenses/LICENSE-2.0).
