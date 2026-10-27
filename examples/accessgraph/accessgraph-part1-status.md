# accessgraph, part 1: status report

Updated: 2026-10-26

Status: report. It records what has been built for the accessgraph example, how
it was checked, how it compared with every other engine and system we touched,
what the research found, and what is still open. A second report will follow
when the remaining features (section 10) are done.

Snapshot of part 1 as of checkpoint r007. Since then T-20, T-31, T-32, T-36, T-37 and T-38 are closed
and T-22 is partial; see
`docs/TRACKING.md` and `docs/RESOLVED.md` for the current state, and the
related-work proposal for the reconciliation of its survey. The sections below
are not updated.

Evidence level: every measurement comes from one single-CPU sandbox with 4 GB of
memory, one run per figure unless stated, and repeated runs of the same
experiment varied by up to a factor of three. Claims about systems we did not
run rest on their abstracts and documentation, not on a full reading, and are
marked as such.

## 1. Summary

accessgraph is an inference example for Jaxson. Given a standing structure of
users, nested groups, roles with inheritance, a resource tree, allow and deny
grants, and bounded delegation chains, it decides whether a user may perform an
action on a resource and names the grant or chain that justifies the answer.
The Shaxon authorization examples judge a trail of events; this one derives
what a structure entails.

What part 1 produced:

- A complete example (version 0.0.4): a seeded generator, a top-down reference
  implementation, a generated Jaxson program, a stand-in interpreter, a checker
  that plants faults, and a scale script.
- Nine independent evaluators of the same rules, plus a naive fixpoint used in
  tests. All nine produced identical decisions on every database they were given.
- A Go version of the whole reference pipeline, in two forms, the second about
  10 times faster than the first.
- A survey of comparable systems, two external datasets, and ten new register
  items (T-31 to T-40) in three waves.

What the results say:

- The Jaxson program is correct against every other evaluator, and its cost is
  deterministic: the stand-in and the real runner agree on the step count.
- It scales roughly linearly, taking 2.4 s at 5,000 users on the real runner.
  It is not fast. Compiled Soufflé is about 8 times quicker end to end, and an
  optimized Go version of the reference about 30 times quicker in compute.
- The differentiator is a contract, not speed or expressive power: total
  evaluation, a declared and enforced step limit, and the same answer and cost
  on independent runtimes. The engines we tried do not offer that contract.
- Three limits surfaced when the data got realistic: ids are fixed width, the
  declared step limit does not scale, and the program's round counts come from
  the reference. All three are filed.
- Agreement so far is agreement of implementations. Every evaluator was written
  from the same reading of the rules, so it does not show the rules are right.

## 2. The example

### 2.1 What it decides

| Element | Rule |
|---|---|
| Subjects | A user's subjects are the user plus every group reachable through membership edges. Groups may nest, and cycles and self-membership are allowed. |
| Roles | A role holds actions and may inherit from other roles. |
| Resources | A tree. A grant on a resource covers everything beneath it. |
| Grants | Allow or deny, to a user or a group, for a role, on a scope. A deny beats an allow on the same resource and action. |
| Delegation | One user hands a role on a scope to another. The chain is at most 3 links. The delegator must hold the same role (not merely a role that inherits it) on the scope or an ancestor. A deny is not transitive through delegation and does not stop a delegator from delegating. |
| Output | For each request: allow or deny, the supporting grant or chain (fewest hops, then the smallest chain string), and every matching deny. |

### 2.2 How it is computed

| Method | Where | Direction |
|---|---|---|
| Reference implementation | `oracle.py` | Top-down: start from one request and search backwards for the derivations that justify it, memoized per user and delegation budget. |
| Generated Jaxson program | `gen.py` emits it as data | Bottom-up: build every user's subjects, every role's permissions, every resource's ancestors and every user's holdings through up to three delegation rounds, then look each request up. Each round is written out, because the language has no unbounded loop. |
| Naive fixpoint | `test_oracle.py` | Bottom-up: apply the rules to everything until nothing changes. Used only as a second check on the reference. |

The program is generated per database, and its round counts come from the
reference's own measurement of that database. That is the weakness filed as T-31.

### 2.3 What was delivered

| Artifact | Contents | State |
|---|---|---|
| `examples/accessgraph/` | The example: `synth.py`, `oracle.py`, `gen.py`, `jx.py`, `check.py`, `mutants.py`, `scale.py`, `test_oracle.py`, README and this report. Presets from `tiny` to `xlarge` (5,000 users). | In the tree since checkpoint r007, version 0.0.4 |
| `agogo.zip` | A Go-only pipeline: generator, line-for-line port of the reference, optimized oracle, tests, benchmark | Delivered |
| `examples/accessgraph/provision/` | A script that fetches and verifies the two external datasets, reading the large one from its compressed stream | In the tree since checkpoint r007 |
| `accessgraph-engines-trial.zip` | The rules written for Soufflé, CozoDB and the Go Datalog engine, with a mutation harness | Delivered, trial quality |
| `jaxson-0.3.1-p000.r007.zip` | The repository with the example and the provisioning tools in the tree, the new register items, waves 8 to 10 and a related-work proposal | Delivered |

## 3. How it was checked

| Check | Scope | Result |
|---|---|---|
| Reference against naive fixpoint | 200 databases, 28,000 decisions (earlier session); 60 databases, 8,400 decisions (re-run) | 0 mismatches |
| Conformance fixtures on the stand-in | 28 fixtures | 21 pass, 0 fail, 7 not implemented by the stand-in |
| Reference, stand-in and real runner | 150 databases of mixed size, 31,320 decisions | 0 mismatches |
| Planted faults, on the real runner | 24 databases, six mutants: one hop short, one hop extra, group round short, role round short, ancestor walk short, denies ignored | All six detected (on 10, 10, 15, 13, 17 and 24 databases); unmutated control clean |
| Step count, stand-in against real runner | Three `medium` databases, found by bisecting the real runner's step limit | Identical: 313,354, 212,944 and 262,733 steps |
| All sizes, seed 0 | `scale.py`, `tiny` to `xlarge` | Reference, stand-in and real runner agree on every decision |
| Datalog engines | Soufflé, CozoDB and the Go Datalog engine, 40 databases, 8,400 decisions each | 0 mismatches each |
| Planted faults, Datalog engines | 24 databases, five mutants each | Hop mutants caught on 10 of 24; denies, group nesting and ancestor mutants on all 24 |
| Go oracles against each other | Both versions, 60 databases at five hop limits, 63,000 decisions | 0 mismatches |
| Go-generated databases | 39 databases, 21,680 decisions, against the Python reference, both Go oracles and Soufflé (not at `xlarge`) | 0 mismatches |
| Nine evaluators at five sizes | Every evaluator asserted equal to the reference on every timed database | No assertion failed |

The weakest spot in the mutation results is delegation. Delegated decisions are
about 6% of all decisions and the hop bound decides about 0.4%, so the hop
mutants are caught on fewer databases than the others (T-33).

## 4. The engines compared

### 4.1 Everything we touched

| Engine or system | Kind | What we did | Notes |
|---|---|---|---|
| `oracle.py` | Reference, Python, top-down | Ran | The reference all others were checked against |
| Naive fixpoint in `test_oracle.py` | Test oracle, Python, bottom-up | Ran | A different algorithm, used only to check the reference |
| Go oracle, line-for-line port | Go, top-down | Ran | Same algorithm and data structures as the Python reference |
| Go oracle, optimized | Go, top-down | Ran | Integer ids, permission bitmasks, Euler-tour ancestor test, flat adjacency arrays, reused buffers |
| Jaxson program on `jaxrun` | The real Jaxson runner, Go | Ran | Built from the repository at 0.3.1-p000.r003 |
| Jaxson program on `jx.py` | A strict stand-in interpreter, Python | Ran | Implements the subset the program uses |
| Soufflé 2.5, interpreter | Datalog, C++, free (UPL) | Ran | Installed from a Linux package (4.8 MB), 20 MB installed |
| Soufflé 2.5, compiled | The same rules compiled to native code | Ran | 34 s to compile once, 1.1 MB executable |
| Go Datalog engine (`swdunlop.dev/pkg`) | Datalog library in Go, MIT | Ran | About 27,700 lines including a web workbench, an MCP server and an agent integration; one maintainer; no release tags, a snapshot dated 2026-09-12 |
| CozoDB 0.7.6 | Embeddable database queried in Datalog | Ran | A native Python wheel, 11 MB |
| pyshacl 0.40.1 | SHACL processor | Ran | Installed with `setup.sh` (23 s, 39 MB); the earlier baseline of 50 agree, 0 differ, 1 not applicable reproduced; the Core versions ran through it |
| Apache Jena SHACL 6.2.0 | SHACL processor | Read only | From the earlier comparison; Maven is not installed here, so the Jena branch of `setup.sh` was not run |
| SHACL Core | A W3C standard | Ran, for two of the five examples | `chinese-wall` and `break-glass` were written in Core alone and agree with Shaxon (T-20, section 6); Core has counts but no arithmetic |
| SWI-Prolog 9.0.4 | Prolog | Considered, not run | Available through the package manager; would probably need tabling to survive the cyclic memberships |
| Biscuit | Datalog authorization tokens | Surveyed, rejected as an engine | Tied to its token format |
| LogicBlox | Commercial Datalog dialect | Surveyed, skipped | An academic-use port only |
| clingo, Soufflé through `apt` | Other engines | Looked for, not available | Neither was in the package list |
| Cedar, Zanzibar and SpiceDB, OpenFGA, Delegation Logic and RT | Authorization systems and theory | Surveyed; OpenFGA's sample stores downloaded | See section 5 |

### 4.2 What it took to write the same rules in each

The rules are about 20 in all: subject closure, role closure, permission sets,
ancestor closure, derivations of "holds role on scope" with a delegation hop
counter, the covering and deny conditions, and a minimum over hops and chain
strings.

| Engine | Chain string and tie-break | Notable constraint |
|---|---|---|
| Soufflé | Built in the engine with string concatenation; string comparison and `min` aggregate used directly | Its extended features allow unbounded recursion over numbers, so termination is no longer guaranteed beyond the pure subset |
| CozoDB | Built in the engine with a concatenation function | Straightforward |
| Go Datalog engine | No string concatenation, and `min` is numeric only, so the chain was encoded as a base-10,000 integer and decoded afterwards | Four six-digit segments would need 24 digits and overflow 64 bits, so it cannot take wider ids (T-37) |
| Jaxson | Chain strings produced by the program and compared byte for byte with the reference | Rounds are written out per database |

The same rules therefore needed three different encodings across three Datalog
engines, which is evidence of dialect fragmentation. The Jaxson program ran on
two runtimes without change.

### 4.3 Timings

All figures are milliseconds per database, end to end as called from Python with
the database in memory. That includes exporting the data, starting a process
where one is needed, and reading the answer back. Sorted by the 5,000-user
column. Small sizes are the mean of 6 databases, the 1,000- and 5,000-user rows
are one database each, best of up to three calls.

| Engine | tiny | small | medium | 1,000 users | 5,000 users |
|---|---|---|---|---|---|
| Go oracle, optimized | 4.9 | 9.7 | 21.3 | 94 | 381 |
| Soufflé, compiled | 4.9 | 8.4 | 17.2 | 83 | 424 |
| Soufflé, interpreter | 20.6 | 24.6 | 35.5 | 122 | 480 |
| Go oracle, line-for-line port | 4.4 | 11.8 | 30.1 | 163 | 706 |
| `oracle.py` | 0.4 | 3.4 | 14.6 | 142 | 782 |
| Jaxson program on `jaxrun` | 14.3 | 31.9 | 112 | 648 | 3,345 |
| CozoDB | 12.1 | 117 | 474 | 3,493 | 17,050 |
| Jaxson program on `jx.py` | 31.2 | 502 | 2,773 | 19,231 | 84,470 |
| Go Datalog engine | 5.7 | 19.7 | 82.0 | 1,733 | 85,992 |

The `jaxrun` and `jx.py` rows include building the Jaxson package from the
database. Earlier tables left that out.

Compute only, inside the process, for the three oracle implementations. The
others have no internal timer.

| Engine | tiny | small | medium | 1,000 users | 5,000 users |
|---|---|---|---|---|---|
| Go oracle, optimized | 0.04 | 0.53 | 1.4 | 7.4 | 33.8 |
| Go oracle, line-for-line port | 0.27 | 1.8 | 9.9 | 66 | 381 |
| `oracle.py` | 0.70 | 4.3 | 20.7 | 147 | 1,038 |

The Go-only pipeline, with no hand-over to Python, gives the cleanest
comparison of the two Go versions on its own generated databases:

| Size | Reference | Optimized | Speedup | Allocated, reference | Allocated, optimized |
|---|---|---|---|---|---|
| tiny | 118 µs | 42 µs | 2.8x | 0.1 MB | 0.0 MB |
| small | 817 µs | 174 µs | 4.7x | 0.8 MB | 0.2 MB |
| medium | 6.6 ms | 0.65 ms | 10x | 4.9 MB | 0.7 MB |
| large | 67 ms | 5.8 ms | 11x | 28.7 MB | 4.2 MB |
| xlarge | 379 ms | 36 ms | 10x | 123.7 MB | 17.6 MB |

The Jaxson program's steps on the example's own presets, seed 0:

| Size | Users | Requests | Steps | Stand-in | `jaxrun` | Reference |
|---|---|---|---|---|---|---|
| tiny | 12 | 40 | 5,336 | 0.02 s | 0.38 s | under 1 ms |
| small | 60 | 200 | 66,098 | 0.41 s | 0.02 s | 4 ms |
| medium | 200 | 400 | 255,453 | 2.0 s | 0.06 s | 13 ms |
| large | 1,000 | 2,000 | 1,751,108 | 16.8 s | 0.55 s | 114 ms |
| xlarge | 5,000 | 10,000 | 8,377,283 | 75.6 s | 2.4 s | 553 ms |

### 4.4 Reading the results

- The fastest evaluators are the optimized Go oracle and compiled Soufflé. They
  are close from Python, and at 5,000 users Go is slightly ahead. In compute
  alone the Go oracle takes 34 ms at 5,000 users.
- Soufflé's time is mostly start-up at these sizes, so these runs say little
  about how it compares at larger scale. Compiling cut its fixed cost most, from
  20.6 ms to 4.9 ms at `tiny`.
- The Go Datalog engine collapses as the data grows: competitive at small sizes,
  86 seconds at 5,000 users, roughly quadratic.
- `jaxrun` grows about 6 times for every 5-fold increase in users, which is
  roughly linear. It is slower than the Datalog engines that scale well, and
  faster than CozoDB and the Go Datalog engine at 5,000 users.
- The declared step limit of 5,000,000 is exceeded at 5,000 users (1.68 times).
  The scale runs raise it to 2,000,000,000. Growth is roughly linear in the
  data, about 4.8 times the steps for 5 times the users.
- The optimized Go oracle is slower end to end from Python than its compute time
  suggests, because most of that time is the hand-over of data. A pipeline
  entirely in Go avoids it.
- Run-to-run noise is large. An earlier run gave 29 s for the Go Datalog engine
  at 5,000 users, 0.25 s for Soufflé's interpreter, 10 s for CozoDB, 2.6 s for
  `jaxrun` and 48 s for the stand-in. The order held apart from near ties, but
  figures from different runs should not be mixed.

## 5. The survey of comparable systems

One round of searches on 2026-10-25. The claims below rest on abstracts,
documentation pages and an advisory. The full table and its caveats are in
`docs/proposals/accessgraph-related-work-and-roadmap.md`.

| Work | Relevant to us because |
|---|---|
| Cedar (AWS) | The closest in method: an executable model checked by proofs, then compared with the production engine by differential random testing. Policies of bounded size are documented to terminate, and an SMT-based analysis checks properties such as policy equivalence. |
| Biscuit and an agent identity protocol built on it | Authorization in a Datalog variant, with delegation by attenuating blocks. The protocol bounds evaluation by forbidding recursion in one profile and capping iterations in another. |
| Zanzibar and SpiceDB | Permissions derived from a relationship graph with nested groups, and explanations of a check. A published advisory describes a wrong denial for caveated groups, the kind of fault an independent reference exists to catch. |
| OpenFGA | Sample authorization models with expected answers written by a third party. Downloaded; not yet mapped. |
| Delegation Logic and RT (Li, Mitchell, Winsborough) | The theory behind the delegation rules: explicit delegation depth, a Datalog translation with tractable semantics, and separation of duty through manifold roles. |

We found no work that combines all of: a general-purpose total language, bounds
that are structural, a program checked against an independently written
reference, and a validation layer that gates the input. That is the result of one
round of searching, not proof that none exists.

Where accessgraph is weaker than this work: no formal proof of any property, no
policy analysis such as equivalence or "can this ever grant X", no authoring
surface for policies (the rules are fixed in the generator), synthetic data until
now, a single reading of the prose rules, and no attribute conditions.

## 6. The earlier SHACL comparison

The Shaxon authorization examples (five of them, with 51 hand-written cases) were
compared earlier with pyshacl 0.40.1 and Apache Jena SHACL 6.2.0, and were timed
at 100, 1,000 and 4,000 events. We reran the pyshacl baseline in this work (50 agree, 0
differ, 1 not applicable) and did not rerun Jena. Two points matter here:

- The five examples use `sh:sparql` throughout. The two that look simplest,
  `chinese-wall` and `break-glass`, were rewritten in SHACL Core alone (T-20)
  and both are expressible: they agree with Shaxon on verdict and reasons on the
  15 case files and on 4,034 and 4,100 random trails, and with the SPARQL
  versions on 800 trails each. Core has counts but no arithmetic, so
  `rolling-quota` is not expected to have a Core form; the other two examples
  were not tried. The one difference is that Shaxon refuses a request for an
  unknown company, which both SHACL versions allow.
- The two SHACL processors agree on every verdict; they differ only in how a
  result is reported (T-21). That means there is no disagreement among
  incumbents to exploit.

accessgraph answers a different question from those examples (what a structure
entails, not whether a trail breaks a rule). SHACL Core can follow transitive
paths, and the two examples above show it can express sequence rules, but it
judges and does not derive: it cannot return the supporting chain or compute the
minimum over derivations that picks it. We have not tried to write the allow and
deny verdict itself in SHACL, so whether Core suffices for that part is open.

## 7. External data

Two datasets of different origin, so that agreement is not an artifact of one
source. The provisioning script pins both.

| | Amazon Access Samples | OpenFGA sample stores |
|---|---|---|
| Origin | Anonymized access data from one company, UCI Machine Learning Repository, 2011 | Models written by a vendor to teach and test a Zanzibar-style engine |
| Terms | CC BY 4.0 | Apache-2.0 |
| Pinned by | SHA-256 of the zip, 12,268,663 bytes | Commit `6802b4d8...2893a0`, 2026-10-09 |
| Content | 36,063 people; 595,506 grants and memberships; a history of 716,063 events | 40 stores with 481 tuples, 556 `check` and 143 `list_objects` assertions |
| State | Profiled, not yet mapped | Inventoried, not yet mapped |

### 7.1 The Amazon data

- **Layout.** The archive is 12 MB, but its main file expands to 4.8 GB, a wide
  matrix of 33,252 binary columns that is almost entirely zeros. It has one row
  per person in strictly increasing person id (0 to 78,181), each row holding
  the person's attributes and all of their access. It is not segmented into
  setup and access blocks. The statistics of its three thirds agree.
- **Columns.** 21,921 system resources, 9,117 permission groups, 2,214 hosts.
  The grants split into 406,661 system, 185,318 permission group and 3,527 host.
- **Skew.** Of the permission groups 6,048 are used. The largest has 3,882
  members, the median group 7, and 12.6% have one person. The top ten groups
  hold 14.8% of group grants. System resources are flat by comparison: 19,845
  used, largest 500, top ten hold 0.7%. One person holds 6,978 direct grants,
  and 12.8% to 13.5% of people have no group.
- **Structure.** Manager ids match no person anywhere in the file (3,207 distinct
  managers), so the hierarchy is unusable. The second org grouping refines both
  others with 5 exceptions among 111 values, a three-level structure with two
  parents. Departments do not nest under it cleanly. Among 1,209 groups of at
  least 20 members, 371 strictly contain another, the longest chain has five
  groups, 146 involve the two groups of about 3,880 members, and 46 pairs have
  identical members.
- **History.** 705,152 `add_access` and 10,911 `remove_access` events from 2005
  to 2010. All 17,612 logins in it are people in the matrix, covering 48.8% of
  them. Its targets match 57.5% of permission-group columns and 54.6% of host
  columns, but none of the system-resource columns, which hold 68% of the
  grants.
- **What it can test.** Scale and skew at realistic shape, and agreement among
  engines on such data. It has no roles, resource tree, group grants, denies or
  delegations, and no expected answers.

### 7.2 What the first look got wrong

A 2,000-person sample suggested a largest group of 225 people, a median of 1 and
51.5% single-person groups. The whole file shows 3,882, 7 and 12.6%. The
provisioning script now reads the whole file by default, in one pass of one to
three minutes, writing two files (23 MB) and nothing else.

## 8. What the results show and do not show

Shown:

- The Jaxson program computes the right answer: it matches independent
  evaluators on every decision, and every planted fault was caught.
- Its cost is deterministic and the same on the stand-in and the real runner,
  and it grows about linearly with the data to 5,000 users.
- The same rules could not be moved between Datalog engines without rewriting,
  while the same Jaxson package ran on two runtimes.

Not shown:

- That the rules are right. One reading of the prose produced every evaluator.
  The 13 hand-built cases are the only check that is independent of it, and the
  OpenFGA stores are not yet mapped (T-34, T-40).
- Speed. The program is slower than the best engines. The case for Jaxson is the
  contract, not performance.
- That totality is affordable on real data. The round counts have to become
  policy, and real data may need deeper nesting than a policy would allow. The
  nested Amazon variant (T-39) is the first real test and its outcome is unknown.
- Behavior at 36,063 users. Nothing has been run at that size on the Jaxson
  program, and the stand-in cannot reach it in reasonable time.
- Step equality beyond `medium`. The step counts of the two runtimes were
  compared exactly on three `medium` databases. At the larger sizes the real
  runner completed under a raised limit, but its own count was not compared.

## 9. Process notes

These are recorded because they cost time or could have cost data.

- A 4.8 GB extraction was started before looking at the archive's shape. It was
  removed once stopped, and the archive was kept.
- A first conversion of that file did not strip the quotes from its cells,
  wrote every cell, and filled the disk with a 5.1 GB file. The broken output was
  deleted at once and the conversion redone.
- A join between the two files read the wrong column and gave a false zero. It
  was redone with the right column. A README also carried a wrong abbreviation of
  a commit hash; both were caught on review.
- The register tool does not record several of its own writes (`register add`,
  `syncver`, `addwave`) in its provenance record, so the record had to be cleared
  four times. Two of those clearances were exact hash matches, one rested on file
  modification times, and one on the sequence of commands, and each reason says
  which. `addwave` writing the register was not in the known-bug list. It should
  be fixed in the tool.
- The data lives in a sandbox that resets, so anything not delivered as a file
  is lost.

## 10. Open work

Ten register items were filed in this work. T-36 is partial because the larger
sizes have run on the reference and both runtimes but not yet on every engine.

| Item | What | Priority | State |
|---|---|---|---|
| T-31 | Round counts come from the reference, not from policy | P3 | Open |
| T-32 | The Shaxon data-integrity layer is missing and load-bearing | P3 | Partial: stage 1 done (references, uniqueness); stage 2 (enforcing the policy constants) after T-31 and T-37 |
| T-33 | Delegation and hop-bound decisions thinly covered | P4 | Open |
| T-34 | One shared reading of the prose rules | P4 | Open |
| T-35 | Delegation strictness not reviewed by a domain expert | P4 | Open |
| T-36 | Scale untested beyond the `medium` preset | P4 | Partial: presets and records done; Amazon scale outstanding |
| T-37 | Fixed-width ids cap grants and delegations below 10,000 | P3 | Open, after T-31 |
| T-38 | The declared step limit does not scale with the data | P3 | Open, after T-31 |
| T-39 | Never run on real-shaped data | P3 | Open, after T-36, T-37, T-38 |
| T-40 | Third-party expected answers not yet used | P4 | Open |

Waves: 8 (policy limits and data integrity: T-31, T-32), 9 (coverage,
independence and scale: T-33 to T-36), 10 (real-shaped data and scale: T-37 to
T-40). The wave estimates (2, 3 and 4 ideal days) were quick guesses and are too
low.

An estimate of the work with the highest conceptual value, in ideal days of
focused work:

| Package | Items | What it would show | Days |
|---|---|---|---|
| A. SHACL Core boundary | T-20, T-22 | Done for two examples (T-20): both are expressible in Core; the other three were not tried | 2 to 3 |
| B. Declared contract, gated input | T-31, T-32, T-33, T-37, T-38 | Validate, then infer, under policy limits and a step limit derived from the data | 6 to 9 |
| C. Real-shaped data | T-39, rest of T-36 | Predicted against measured cost at 36,063 users, flat and nested | 3.5 to 5 |
| D. Second runtime | T-15, T-16, for Jaxson first | The same fixtures, answers and steps from independent code | 4.5 to 6.5 |
| E. Third-party semantics | T-40, T-34 | The rules checked against someone else's answers | 4 to 7 |

A, B and C together come to about 11.5 to 17 days, centered on 14. D needs a
ruling that fixes a total (T-16) before it can start. A second runtime written
from the same specification would be independent code but not an independent
reading, so package D is strongest with a different author.

## 11. Reproducing the results

| To reproduce | Run |
|---|---|
| Reference against naive fixpoint | `python3 test_oracle.py 60` in the example |
| Reference, stand-in and real runner | `python3 check.py 24 --jaxrun PATH` |
| Planted faults | `python3 mutants.py --jaxrun PATH` |
| Steps and wall time per preset | `python3 scale.py --jaxrun PATH --sizes tiny,small,medium,large,xlarge` |
| Go pipeline | `go build -o agogo . && ./agogo check -n 60 && ./agogo bench` |
| Go against Python | `ACCESSGRAPH_DIR=... AGOGO=./agogo python3 xcheck.py` |
| External data | `examples/accessgraph/provision/fetch.sh [DATA_DIR]` (about 13 MB downloaded, 70 MB kept) |

### Environment

Single-CPU sandbox, 4 GB of memory, Linux x86-64. Go 1.27.3, Python 3. The real
runner built from the repository at 0.3.1-p000.r003. Soufflé 2.5. CozoDB 0.7.6.
Go Datalog engine `swdunlop.dev/pkg` at pseudo-version
`v0.0.0-20260912202005-5fb120d54af7`. pyshacl 0.40.1, Apache Jena SHACL 6.2.0 and
rdflib 7.6.0 are as recorded in the earlier authorization comparison.
