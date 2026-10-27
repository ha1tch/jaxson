# accessgraph

accessgraph answers one question about a standing structure of permissions:
may this user perform this action on this resource, and why? The structure has
nested groups (cycles allowed), a role hierarchy, a resource tree, allow and
deny grants, and delegation chains of bounded length. Every answer names the
grant or chain that justifies it, chosen by a fixed rule, so any implementation
must give the same one.

The Shaxon authorization examples judge a trail of events against a rule. This
one derives what a structure entails, which is what Jaxson is for.

## Why it matters

- **It is inference, not checking.** A decision depends on closure through
  groups, role inheritance, resource ancestry and delegation, on a condition at
  every delegation hop, and on choosing the best supporting derivation. SHACL
  Core can follow transitive paths and can express sequence rules (two of the
  Shaxon authorization examples were written in it), but it judges and does not
  derive: it cannot return the supporting chain or compute the minimum over
  derivations that picks it. We have not tried to write the verdict itself in
  SHACL, so whether Core suffices for that part is open.
- **The whole thing is JSON, under a contract.** The database, the program, the
  data-integrity gate and the answers are JSON documents run by one small
  runtime with no dependencies. Evaluation is total and has an enforced step
  limit, derived from the data before the run; the program is the same for every database and refuses data deeper
  than its declared limits; and on two separately written runtimes it gave the
  same decisions and exactly the same step count on every database compared, up
  to 5,000 users. The other tools we tried needed the data translated first:
  SHACL an RDF graph, and the three Datalog engines (Soufflé, CozoDB, a Go
  library) flat tuples, with a different encoding of the same rules each. A cost
  limit on its own is not new: CEL computes a static worst-case cost and
  enforces a runtime limit, and Starlark limits execution steps (Starlark was run on the
  same task and CEL probed; see `FINDINGS.md`, section 5).
- **It is checked from several directions.** Decisions are compared with a
  top-down reference, a naive fixpoint and seven other evaluators (two Go
  ports of the reference, Soufflé interpreted and compiled, CozoDB and two
  Jaxson runtimes); a Go Datalog library agreed at four-digit ids and is
  retired at the current width. Nine planted faults are caught on the real
  runner, and the integrity gate is checked against 732 planted defects.

What the work has and has not shown, with the measurements behind it, is in
`FINDINGS.md`.

## Status

| Part | State |
|------|-------|
| Generator (`synth.py`), presets `tiny` to `xlarge` | done; stays within the policy limits and refuses a count beyond the fixed id width |
| Reference oracle (`oracle.py`) | done; agrees with a naive fixpoint (200 databases, 28,000 decisions) |
| Generated Jaxson package (fixed rounds) | done; the same program for every database; agrees with the oracle on the stand-in and on `jaxrun` (150 databases, 31,320 decisions, 0 mismatches) |
| Checker with planted faults | done; nine mutants detected on `jaxrun`, round mutants both silent (answers must differ) and asserted (the program must refuse) |
| Policy limits (`policy.py`) | done: declared once, with the grant and delegation id width; the reference, generator and program enforce the depth limits, and the reference, generator and gate the id width (see Limits are policy) |
| Declared step limit (`cost.py`) | done: derived from the data before the run, fitted on 73 synthetic databases, held out at 5,000 users and checked on two real-shaped databases; exceeding it stops both runtimes with `RESOURCE_ERROR/STEPS` (see Scale) |
| Real-shaped data run (`provision/amazon_to_db.py`) | done for the Amazon data: seven evaluators agree on 10,000 decisions on both variants; the nested variant puts true group depth at 5; the program fits its derived step limit (48% used) |
| Scale record | done to 5,000 users (see Scale) |
| Comparison with other engines | done (see Compared with other engines) |
| Data-integrity gate (`integrity.py`, a Shaxon package) | references, uniqueness, enum and id-width checks done and tested against 732 planted defects of 24 kinds; depth checks work when limits are passed in, and the program asserts its own limits, so the gate's depth check stays opt-in (T-32, decided) |
| Policy layer (separation of duty) | not yet written |
| Real-shaped data, third-party expected answers | not started |

`jx.py` is a strict stand-in for `jaxrun`: 21 of the 28 conformance fixtures
pass on it and the other 7 use features it does not implement. For the
generated program, the real runner accepts a step limit equal to the stand-in's
count and rejects one less on all nine databases tried, from 12 to 5,000 users
(10,533 to 10,229,845 steps), so the two counts are identical. Run
`check.py --jaxrun` for the agreement, `mutants.py --jaxrun` for the planted
faults, `test_policy.py --jaxrun` for the limits and `test_cost.py --jaxrun` for the
declared step limit.

## Semantics

- `subjects(u)`: the user, plus every group reachable through membership
  edges. Nesting, two-cycles and self-membership are allowed.
- A user holds a role on a scope through an allow grant to any of their
  subjects (hops 0), or through a delegation (hops k+1): a delegation is valid
  only if the delegator holds the same role, at k hops, on that scope or an
  ancestor of it. Holding a role that inherits the delegated role does not
  entitle the holder to delegate it. Chains are bounded by `MAX_HOPS` (3).
- `can(u, action, resource)` holds when some held (role, scope) covers the
  resource (scope is the resource or an ancestor) and the role, including
  inherited roles, grants the action, and no deny grant to any of the user's
  subjects covers the same resource and action. Denies are not transitive
  through delegation and do not stop a delegator from delegating.
- Provenance (`via`) is the support with the fewest hops, then the smallest
  chain string, for example `G000003>D000002`. Grant and delegation ids are
  fixed width (six digits), so string order is numeric order.

## Limits are policy

Jaxson has no unbounded loop, so a closure is written as a fixed number of
rounds. `policy.py` declares them once: 3 delegation hops, 16 group rounds, 8
role rounds and resource depth 6, chosen from measured depths (12 group rounds
at 5,000 users, 5 role rounds, 6 resource levels by construction). The
generated program is therefore the same for every database. The reference and
the generator work within the limits, and the program asserts, with one extra
round per closure, that its input fits: data deeper than a limit fails with
`ASSERTION_FAILED` on both runtimes and is never answered from a truncated
closure. At each limit exactly, a database is accepted and answered like the
reference; one past it, everything refuses (`test_policy.py`). Built without
the assertion, the same database is answered wrongly, which is what it
prevents.

The price is that every database pays for the full rounds: 97% more steps at 12
users, 22% more at 5,000. A larger limit costs steps for everyone; a smaller
one rejects more data.

Grant and delegation ids are policy too: `ID_WIDTH` digits after the letter
(`G000001`, `D000001`), so up to 999,999 of each. They appear in chain strings
and in sorted lists, where a mixed width would sort in an order nobody intended
and an engine that encodes a chain as numbers would overflow. A database with a
grant or delegation id of another width is refused by the reference, the
generator and the integrity gate (`test_policy.py`, `test_integrity.py`); the
other ids are opaque. With four digits the generator stopped at 9,999 grants;
`test_policy.py` now answers a database of 10,068 grants identically on the
reference and both runtimes.

## Data

The data most checks use is synthetic: `synth.py` generates every database from a
seed and a preset, so each one can be reproduced.

Two external datasets are provisioned. The Amazon data has been run through the
program (T-39; see `FINDINGS.md`, section 2); the OpenFGA stores have only been
inventoried. `provision/fetch.sh` (with `provision/amazon_sample.py`) downloads,
verifies and reduces them; both are pinned, and `data/` is ignored by git, so no
external data is committed. `provision/amazon_to_db.py` turns the Amazon data into
two accessgraph databases, flat and nested, keeping its people, memberships and
direct grants and adding roles, a resource tree, group grants, denies, delegations
and requests from the generator's distributions, with a record of every addition;
`provision/check_amazon.py` checks them.

| Dataset | Origin | Terms | Pinned by |
|---|---|---|---|
| Amazon Access Samples | anonymized access data from one company, published through the UCI Machine Learning Repository (dataset 216, 2011) | CC BY 4.0 | SHA-256 `d493e337...b8228d224` of the zip |
| OpenFGA sample stores | authorization models with expected answers, from the OpenFGA project | Apache-2.0 | commit `6802b4d8...2893a0` |

## Files

- `synth.py`: seeded generator. `python3 synth.py SEED [tiny|small|medium|large|xlarge]`
- `oracle.py`: reference implementation (top-down enumeration of derivations)
  and `bounds()`, the round counts a bounded bottom-up evaluator needs
- `test_oracle.py`: hand-built adversarial cases, agreement with a naive
  fixpoint on generated databases, determinism and coverage checks
- `FINDINGS.md`: what the work has and has not shown, with the measurements behind
  each statement
- `policy.py`: the limits (delegation hops, group and role rounds, resource
  depth, id width), declared once
- `gen.py`: builds the Jaxson package from a database (closures unrolled to the
  rounds of `policy.py`, with an assertion that the data fits)
- `jx.py`: stand-in interpreter for the subset the package uses
- `check.py`: oracle vs stand-in, and vs `jaxrun` with `--jaxrun`
- `mutants.py`: plants nine faults in the generated program and requires the real
  runner to expose each
- `test_policy.py`: one program for every database, and each limit accepted at
  its value and refused one past it
- `integrity.py`, `integrity.json`: the data-integrity gate as a Shaxon package, and
  the generated package; `test_integrity.py` checks it against planted defects
- `scale.py`: steps, declared limit and wall time per preset on the reference and both
  runtimes
- `cost.py`, `cost_model.json`: the declared step limit, derived from the data before
  the run, and its fitted coefficients; `calibrate.py` fits them on measured steps
  (needs numpy) and `test_cost.py` checks them on data the fit never saw
- `agogo/`: the reference and its optimized version as a Go program, with a generator,
  tests and a cross-check against `oracle.py`; see its README
- `engines/`: the same rules for Soufflé and CozoDB (and a retired Go Datalog library),
  the timing sweep and the fault-planting comparison behind `FINDINGS.md`; see its
  README
- `provision/`: fetches and verifies two external datasets, a real anonymized
  access matrix and third-party authorization models, and adapts and checks the
  first as accessgraph databases; see its README
- `accessgraph-part1-status.md`: status report for part 1: what was built, how
  it was checked, how other engines compared, what is open

## Running the checks

    python3 test_oracle.py [N_SEEDS]
    python3 check.py [N_SEEDS]
    python3 check.py [N_SEEDS] --jaxrun /path/to/jaxrun
    python3 mutants.py --jaxrun /path/to/jaxrun [N_SEEDS]
    python3 test_policy.py --jaxrun /path/to/jaxrun
    python3 test_cost.py --jaxrun /path/to/jaxrun
    python3 calibrate.py [--write]       (needs numpy; about 8 minutes from scratch)
    SHAXONRUN=/path/to/shaxonrun python3 test_integrity.py [N_TRIALS]

Requires Python 3 only, except `calibrate.py`, which also needs numpy, and the
comparison in `engines/`, which needs the engines named in its README. Build `jaxrun` with
`go build -o jaxrun ./cmd/jaxrun` in `src/jaxson-shaxon-v0.3.1`.
`python3 gen.py SEED [size] > pkg.json` writes a package for `jaxrun pkg.json`.

## Scale

`python3 scale.py --jaxrun PATH` records steps and wall time at each preset on
the reference, the stand-in and the real runner, and requires all three to give
identical decisions. One database per size, seed 0:

| Size | Users | Requests | Steps | Declared | Used | Stand-in | `jaxrun` | Reference |
|---|---|---|---|---|---|---|---|---|
| tiny | 12 | 40 | 10,533 | 12,819 | 82% | 0.02 s | under 0.01 s | under 1 ms |
| small | 60 | 200 | 99,451 | 117,719 | 84% | 0.30 s | 0.01 s | 2 ms |
| medium | 200 | 400 | 341,942 | 431,419 | 79% | 1.3 s | 0.05 s | 8 ms |
| large | 1,000 | 2,000 | 2,217,569 | 2,909,292 | 76% | 10.3 s | 0.31 s | 66 ms |
| xlarge | 5,000 | 10,000 | 10,229,845 | 13,587,319 | 75% | 44.5 s | 1.5 s | 353 ms |

Steps grow roughly in line with the data and include the full policy rounds (see
Limits are policy). The declared limit is derived from the data before the run
(`cost.py`): features that follow the program's loops (the closure scans, the held
table, each delegation hop over its delegator's held entries, each request's scan),
multiplied by the policy rounds and weighted by coefficients fitted on measured
steps (`calibrate.py`, 73 synthetic databases up to 1,000 users, 18 of them
stretching one dimension at a time), with a margin of 1.25. It was checked on 28
synthetic databases the fit never saw, including 5,000 users and one with the
Amazon data's proportions, and none exceeded it. The two real-shaped Amazon
databases were never in a fit, and the program used 48% of its declared limit on
them: the model over-predicts there by 1.68 times. A first model, whose features
counted requesters where the program builds its held table for every user, had
under-predicted by 2.21 times (T-41). A run that exceeds its limit stops with `RESOURCE_ERROR/STEPS` on both
runtimes, and nothing raises it (`test_cost.py`). The model carries a fingerprint
of the generated program and refuses to run after the program changes, until it
is calibrated again. Ids are fixed width, so the generator refuses a preset with a
million or more grants or delegations. Times come from one run on a single-CPU
sandbox and vary by up to a factor of two between runs.

## Compared with other engines

The same rules were written for each engine below and every one agreed with the
reference on every decision it was given. End to end from Python at 5,000 users,
one database, including data export and process start, fastest first:

| Evaluator | Time | Note |
|---|---|---|
| Go port of the reference, optimized | 0.27 s | 28 ms of that is computation |
| Soufflé 2.5, compiled | 0.30 s | |
| Soufflé 2.5, interpreter | 0.37 s | |
| Go port of the reference, line for line | 0.47 s | |
| `oracle.py` | 0.49 s | the reference |
| Jaxson program on `jaxrun` | 2.4 s | includes building the package |
| CozoDB 0.7.6 | 12.5 s | |
| Go Datalog library | 22 s | measured on four-digit ids; chains were encoded as numbers, which does not fit six-digit ids, so it is retired |
| Jaxson program on `jx.py` | 67 s | a stand-in, not meant to be fast |

The first six rows are the median of four runs (the Soufflé interpreter once
took 1.5 s, a cold start the median leaves out); the last three are one run
each, and the last two traded places between runs. Times vary by up to a factor
of three between runs on this single-CPU sandbox, so read the order, not the
digits. The Go ports are in `agogo/` and the Datalog encodings in `engines/`; the
status report describes them and the other systems surveyed.

## Known limits

- The data-integrity gate enforces the depth limits only when they are passed in
  (`--limits policy`), and it is a separate step. The program asserts its own
  limits, so the gate's depth part repeats that work at 30 to 50% of the
  program's steps (measured to 5,000 users, not yet at 36,063); it stays an
  opt-in extra that gives located findings (decided under T-32).
- The cost model is fitted on synthetic data up to 1,000 users and checked on two
  real-shaped databases from one source, the Amazon data. It over-predicts on that
  data by 1.68 times, so its limit is 2.1 times the measured steps there. Those
  databases were looked at after every fit and decided between model versions, so
  they are validation data, not an untouched test; data of a shape neither the
  generator nor that source has is untested.
- Fixed limits cost steps: 22% to 97% more than sizing the rounds to each
  database, and a larger limit costs everyone.
- Delegation decisions are the least exercised part of the checks (T-33), and
  the rule that a delegator must hold the same role, not an inheriting one, has
  not been reviewed by a domain expert (T-35).
- One reading of the prose rules produced every evaluator (T-34).
- The only data not made by the generator is the Amazon data's people,
  memberships and direct grants; roles, the resource tree, group grants, denies,
  delegations and requests were added (FINDINGS.md, section 2). Third-party
  expected answers are open (T-40).

## License

Apache License 2.0: https://www.apache.org/licenses/LICENSE-2.0

Copyright (c) 2026 haitch <h@ual.li>
https://oldbytes.space/@haitchfive
