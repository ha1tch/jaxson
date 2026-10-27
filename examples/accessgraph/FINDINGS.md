# accessgraph findings

Updated: 2026-10-27

What the work has shown and what it has not, with the measurements behind each
statement. The README says what accessgraph is and how to run it; this document
holds the conclusions.

## Summary

The question tracked here is whether Jaxson does at least as much inference as the
incumbents surveyed in the same domain space, and whether it has better control. Each
claim below has its status as of checkpoint r021 and the basis for it. The sections
after it hold the measurements: 1 agreement, 2 real-shaped data, 3 declared limits and
what they cost, 4 speed, 5 Starlark and CEL, 6 other languages against the tests, and
7 Shaxon against its incumbents.

| Claim | Status | Basis |
|---|---|---|
| It does this inference in a total, JSON-native language under an enforced budget | Shown | Nine evaluators agree on every decision; nine planted faults caught; identical step counts on two runtimes up to 5,000 users |
| It does at least as much inference as the incumbents in the same domain space | Shown against the Datalog engines and Starlark, on this task; not tested against the policy engines | Soufflé, CozoDB, a Go Datalog library and Starlark did the same inference once the rules were encoded, and agreed with the reference on every decision (section 5 for Starlark). The domain space here is the access-decision task of this example: nested groups, role inheritance, a resource tree, allow and deny grants and bounded delegation, with the supporting grant named. That domain is defined by this example, not by the incumbents, and what they express beyond it is not covered. OPA, SpiceDB, OpenFGA, Cedar, CEL and Biscuit were surveyed or only partly tried, and none was run on this task; third-party expected answers (T-40) are the first test of the domain itself |
| It has better control: a declared limit, refusal of data beyond it, a cost derived before the run, identical counts across runtimes | Shown against the engines run, not against all; the cost derived before the run holds on the real-shaped data run so far, after rebuilds that the data itself helped select (T-41) | Soufflé, CozoDB and the Go library have none of these; Starlark has an enforced, deterministic step budget but no limit derived before the run and no refusal of data that is too deep, and CEL's static estimate is a sound bound that grows exponentially with the rounds of a closure (section 5) |
| No existing language covers the rules, the validation and the contract together | Shown for the languages surveyed; most are from documentation | Each has part: rules and validation natively in Rego, SQL and Datalog; a deterministic step budget in Starlark; a static cost estimate in CEL; a depth limit with an error in SpiceDB and OpenFGA. None has the runtime refusing, a deterministic count, a cost derived beforehand and identical counts across implementations together (section 6) |
| Shaxon validates with findings at exact JSON paths, under the same budget as the rules | Shown against SHACL on the five authorization examples; the accessgraph gate was not tried in SHACL or other languages | 50 of 50 comparable cases agree with SHACL-SPARQL. Shaxon is JSON-native and refuses over-budget and malformed input, which SHACL does not express, or expresses only with an added structure shape. SHACL is a well-understood standard and its shapes are shorter (section 7) |
| Programs written in Jaxson can run at the speed of optimized Go for the same task | The target is established; Jaxson has not reached it yet: the current runtime is slower, and the compiler (jaxpiler) designed to close the gap is the next stage | The optimized Go port decides the same requests in 271 ms end to end at 5,000 users, 32 ms of it computation; the program on the current runtime takes 3,172 ms. Against the other engines the current runtime is slower than compiled Soufflé and the plain implementations and faster than CozoDB (section 4) |

What would change the answer: third-party expected answers (T-40), a second
reading and a domain review of the rules (T-34, T-35), real-shaped data from a source
other than the Amazon data, and the same task on one policy engine.

## 1. What agreement shows

The reference, a naive fixpoint and seven other evaluators agree on every
decision they were given. That shows the implementations agree. All of them were
written from one reading of the rules, so it does not show that the rules are
right. A second, independent reading (T-34), third-party expected answers
(T-40) and a domain review of the delegation rule (T-35) are open.

## 2. Real-shaped data (T-39)

The Amazon Access Samples, adapted by `provision/amazon_to_db.py` (seed 39), run
through the program and seven other evaluators. From the data: 36,063 people,
6,048 groups with members, 185,318 memberships and 410,188 direct grants over
21,578 resources. Added from the generator's distributions, because the data has
none: 16 roles, a resource tree of 601 folders (depth 3), 12,000 group grants,
8,000 denies, 28,304 delegations and 10,000 requests. The people, memberships and
direct grants are the only part not made by the generator.

There are two variants. Flat is the data as it is. Nested uses the 270 covering
edges of the 371 containments inferred between groups of at least 20 members, and
drops the 10,134 memberships those edges imply. Adding the 371 edges alone changes
nothing, because the data's memberships are already closed (a user in a small
group is listed in every larger group that contains it), so the true depth would
stay 1. In nested, every user's reach is identical to flat's, the true group depth
is 5 against the limit of 16, and the reference gives identical decisions on both:
10,000 of 10,000 (4,335 allowed, 5,665 denied, 1,320 through a delegation chain).

End to end, seconds, one run each on a single-CPU sandbox (`engines/amazon_one.py`):

| Evaluator | Flat | Nested | Agrees with the reference |
|---|---|---|---|
| Reference (`oracle.py`) | 9.4 | 8.7 | |
| Go port, optimized | 7.5 | 8.3 | yes |
| Go port, line for line | 10.2 | 10.3 | yes |
| Soufflé, compiled | 4.5 | 4.7 | yes |
| Soufflé, interpreter | 6.1 | 5.0 | yes |
| Jaxson program on `jaxrun` | 127.7 | 122.6 | yes |
| CozoDB | 156 | 185 | yes |

Not run: the Python stand-in, too slow at this size, and the Go Datalog library,
retired. The Jaxson times used a raised limit and include building and writing a
package of about 150 MB; the runner itself took 47.0 s and 47.7 s. Against compiled
Soufflé, Jaxson is about 28 times slower end to end at this size, and about 14 times
slower than the reference. CozoDB is 1.2 to 1.5 times slower than Jaxson, where it
was about 5 times slower at 1,000 and 5,000 users.

The first cost model (section 3) derived a declared limit of 119,612,662 steps for
flat (119,745,205 for nested) before any run, and the program stopped at it with
`RESOURCE_ERROR/STEPS`. Its exact need, from a scratch build of `jaxrun` that prints
the step counter and agrees with the stand-in on four known databases, is
220,106,734 steps for flat and 220,150,664 for nested: 2.21 times the prediction and
1.84 times the declared limit. The nested closure costs 43,930 steps more than flat
(0.02%). The limit failed loudly, as designed, but it was wrong: the model counted
requesters where the program builds its held table for every user, and its
delegation term, requests times delegations, only imitated the real cost on
synthetic data, where requests are about twice the users. T-41 rebuilt the features
from the program's loops and fitted them on synthetic data, with the Amazon databases
held out of every fit. The first rebuild fitted tightly and covered the Amazon data
(69% of the limit used), but `test_policy.py` then failed on a contrived database at
the resource depth limit, because the fit had given the resources term a coefficient
of zero: generated databases grow every dimension together, so a fit on them alone
cannot tell the terms apart. Adding 18 databases that stretch one dimension at a time
fixed that, and widened the margin to 1.25. A further refinement, counting distinct
role-and-scope pairs in the held table as the program keys it, fitted the synthetic
databases more tightly and failed on the Amazon data (2.17 times under, limit
exceeded), so it was reverted. The final model predicts 370,299,033 steps for flat,
1.68 times the measured, and declares 462,873,792 (463,002,897 for nested): the
program runs inside its declared limit on both, using 48% of it, with answers equal
to the reference. The held-out synthetic databases, one of them with the Amazon
data's proportions, are predicted between 17% over and 7% under. The Amazon
databases were looked at after each of four fits and decided between versions, so
they are validation data, not an untouched test, and the model is checked against
that one real-shaped source.

The integrity gate passes both databases with 0 findings: 2.9 million steps without
the depth check (4 s) and 33.9 million with it (7 s), about 1% and 15% of the
program's steps. That is below the 30 to 50% measured on synthetic data, so the depth
check is not prohibitive here and stays opt-in (T-32). Step-count equality between
the stand-in and the real runner is checked to 5,000 users (10.2 million steps) and
not at this size.

## 3. Declared limits and what they cost

Every database pays for the full rounds declared in `policy.py`. Compared with
sizing the rounds to each database, the program takes 97% more steps at 12 users
(5,336 against 10,533) and 22% more at 5,000 users (8,377,283 against
10,229,845). A larger limit costs steps for everyone and a smaller one rejects
more data. A fixed limit does not fit: 5,000 users exceed a default of 5,000,000 steps by
2.05 times, so the declared limit is derived from the data before the run (T-38,
`cost.py`). At the fitted sizes it is 1.2 to 1.3 times the measured steps; at
5,000 users, five times larger than anything the model was fitted on, it is
13,587,319 for 10,229,845 measured (1.33 times). On real-shaped data at 36,063 users
a first model failed, 2.21 times under; the rebuilt model covers it, using 48% of the
limit (section 2, T-41).

## 4. Speed

Time to decide the same requests, by evaluator, on synthetic data to 5,000 users (the
table below) and on real-shaped data at 36,063 users (section 2). The Jaxson program on
`jaxrun` is slower than compiled Soufflé (about 5 times at 12 users, growing to about 10
times at 5,000) and than the Python and Go implementations of the same rules (about 5
times slower than the Python reference at 5,000 users, and more at smaller sizes). It is
faster than CozoDB from 60 users upward: about 2 times at 60 users, 3 times at 200 and 5
times at 1,000 and 5,000, and 1.2 to 1.5 times at 36,063 users on real-shaped data. At 12
users CozoDB is faster.

What this measures. The Jaxson times are those of the current Go runtime, which runs
the generated program directly and counts its steps. They measure that runtime, not the
language. The performance achievable in Go for this task is established: the optimized
Go port decides the same requests in 271 ms end to end at 5,000 users, 32 ms of it
computation, against 3,172 ms for the program on `jaxrun`. The compiler from Jaxson to
Go now being designed (tentatively jaxpiler, the next stage after this one) aims for
type-safe generated code that reproduces that performance. Among interpreters,
starlark-go decides the same requests 1.3 to 2.8 times faster than the Jaxson runtime end
to end (section 5).

End to end, milliseconds per database, called from Python with the database in
memory, current program (fixed rounds, six-digit ids), from `engines/sweep.py`. One run; the first three
sizes are the mean over 6 databases, the last two are one database each.

| Evaluator | 12 users | 60 users | 200 users | 1,000 users | 5,000 users |
|---|---|---|---|---|---|
| `oracle.py` (the reference) | 0.4 | 3.9 | 10.7 | 97.6 | 618 |
| Go port, optimized | 3.8 | 6.5 | 19.0 | 63.7 | 271 |
| Go port, line for line | 4.0 | 10.5 | 24.2 | 112 | 486 |
| Soufflé 2.5, compiled | 3.3 | 7.3 | 14.3 | 67.8 | 326 |
| Soufflé 2.5, interpreter | 206 (first call) | 18.3 | 26.4 | 114 | 361 |
| Jaxson program on `jaxrun` | 16.5 | 42.8 | 97.5 | 559 | 3,172 |
| CozoDB 0.7.6 | 11.8 | 90.7 | 318 | 2,959 | 15,442 |

The figures come from a single-CPU sandbox and vary between runs, by up to a
factor of three between sessions, so the ordering is the finding and the digits
are not. The ordering was the same in an earlier run at 5,000 users, where the
fast engines were repeated four times and Jaxson took 2.4 s. The Soufflé
interpreter figure at 12 users includes a one-off cold start. The `jaxrun` row
includes building the package, and the end-to-end rows include data export and
process start, which dominate at the small sizes; computation alone takes 32 ms
in the optimized Go port at 5,000 users. The Go Datalog library (22 s at 5,000
users on four-digit ids, since retired) and the Python stand-in `jx.py` (not
meant to be fast) are left out.

## 5. Starlark and CEL

The two incumbents surveyed that have cost limits of their own, run on the same task.
Starlark (starlark-go) got the whole task. CEL is an expression language with no
procedures, so it got a probe of its cost estimate on the one part it can express, the
group-membership closure.

**Starlark.** The rules were ported within the language as shipped
(`engines/starlark/rules.star`): no sets, no recursion, no `while`. Sets are dicts,
delegation chains are built level by level, and graph loops are bounded with `range`.
The port agrees with the reference on 24 generated databases (5,040 decisions) and on
both Amazon variants (10,000 decisions each, 24.4 s and 18.4 s end to end).

What differs from Jaxson:

- The closure loops to a fixed point and stops early, with `range(1000000)` as the bound,
  so it needs no declared number of rounds and does not refuse data that is too deep. The
  policy limits and the assertion that the data fits are Jaxson's; Starlark has no
  counterpart.
- It has a step budget that the interpreter enforces, and a deterministic count: repeated
  runs give the same number. A run past the budget stops with an error. A budget equal to
  the exact count still stops, so it needs one more.
- It has no limit derived before the run and no input validation.
- Its unit differs, so counts compare only within one implementation. Identical counts
  across implementations (it has others, in Java and Rust) were not tested.

| Data | Users | Starlark steps | Jaxson steps |
|---|---|---|---|
| `tiny` | 12 | 43,010 | 10,533 |
| `small` | 60 | 339,420 | 99,451 |
| `medium` | 200 | 1,149,150 | 341,942 |
| `large` | 1,000 | 7,111,270 | 2,217,569 |
| `xlarge` | 5,000 | 30,194,300 | 10,229,845 |
| Amazon, flat | 36,063 | 188,232,060 | 220,106,734 |
| Amazon, nested | 36,063 | 187,987,570 | 220,150,664 |

End to end, milliseconds per database, one run with the three evaluators together (the
same conditions as section 4):

| Evaluator | 12 users | 60 users | 200 users | 1,000 users | 5,000 users |
|---|---|---|---|---|---|
| `oracle.py` (the reference) | 0.3 | 4.1 | 10.1 | 77.3 | 510 |
| Jaxson program on `jaxrun` | 12.7 | 31.2 | 98.4 | 449 | 3,383 |
| Starlark (starlark-go) | 4.5 | 18.3 | 59.0 | 355 | 2,373 |

Starlark is 1.3 to 2.8 times faster than the Jaxson runtime end to end, and 4.5 to 15
times slower than the Python reference. On the Amazon variants it took 24.4 s and 18.4 s
end to end, against 47.0 s and 47.7 s for the Jaxson runner alone (122.6 s and 127.7 s
with the package build, section 2). The two are in the same class of interpreter.

**CEL.** The probe (`engines/cel/`, cel-go 0.30.0) unrolls the closure k rounds on the
nested Amazon database, for its heaviest user (true reach 100). The bounds it declares
are the data's maxima, the tightest honest declaration: lists of at most 87 elements, a
map of 31,516 entries, strings of at most 8 characters.

| Rounds | Core macros: estimated cost, upper bound | Counted | Materialized: estimated | Counted |
|---|---|---|---|---|
| 1 | 619 | 618 | unbounded | 16,433 |
| 2 | 54,211 | 941 | unbounded | 89,073 |
| 3 | 4,716,715 | 1,035 | unbounded | 173,859 |
| 4 | 410,354,563 | 1,175 | unbounded | 261,177 |
| 6 | 3.1 x 10^12 | 1,399 | unbounded | 443,589 |
| 8 | 2.4 x 10^16 | 1,399 | unbounded | 626,889 |

With the core macros only, reachability asked as k nested `exists`, the estimate is a sound
bound and exact for one round, then 87 times larger for every round, while the counted
cost is flat once the closure has converged. Materializing the reach set needs extension
functions: `flatten()` is estimated, but `distinct()` is not (about 1.5 x 10^19 for a
list of two), so that encoding has no usable static estimate at any number of rounds. The
policy needs 16 rounds. Jaxson's derived limit is a calibrated prediction instead: 1.2 to
1.3 times the measured steps at the fitted sizes and 2.1 times on the Amazon data
(section 3). It is not a proven bound and it stops loudly when exceeded. CEL's estimate is
a proven bound under declared sizes, and it is exponential in the rounds of a closure.
The factor per round is the declared list bound: 87 on the Amazon data, and about 3 on a
generated 5,000-user database, where the estimate at eight rounds is 91,843 against 749
counted.

## 6. Other languages against the tests

Which existing declarative, deterministic, inspectable languages could express the tests
reported here? The tests are of three kinds:

- the rules: closure over nested groups, role inheritance and the resource tree; deny
  overriding allow; delegation chains of at most 3 hops, with the fewest-hops provenance;
- the validation of the input: references, uniqueness, enums, id width and depth, with
  findings located exactly;
- the contract: refusal of data past a declared depth, a deterministic step budget, a
  cost derived before the run, identical counts across implementations.

This section is about languages that exist. Writing the rules in a language by hand shows
that the author can encode them, not that the language has them, so the table separates
what a language provides (native) from what an encoding supplies. Rego, Cedar, SpiceDB and
OpenFGA are from their documentation and were not run here.

| Language | Rules and closure | Delegation chains with provenance | Validation | Contract | Basis |
|---|---|---|---|---|---|
| Soufflé (Datalog) | native | by encoding | by encoding: rules that derive violations | none | run |
| Starlark | by encoding, bounded loops | by encoding | by encoding | step budget enforced and deterministic; no depth refusal; none derived before the run | run |
| CEL | closure only by unrolling | no | native (its main use in Kubernetes) | static estimate plus runtime limit, sound but exponential per round | probed (section 5) |
| Rego (OPA) | native: `graph.reachable`; recursion is forbidden so that evaluation terminates | `graph.reachable_paths` returns paths; provenance by encoding | rules over JSON input | no depth refusal or deterministic budget found | documentation |
| Cedar | native: `in` hierarchy, `forbid` overrides `permit`, deny by default | no | schema checks, not our uniqueness or width | termination guaranteed, deterministic; no budget exposed | documentation |
| SpiceDB, OpenFGA | native: nested groups, parents, exclusion | no: boolean checks | schema-level | depth limit with an error (SpiceDB max depth; OpenFGA 25, being replaced by dynamic load management) | documentation |
| SQL, recursive queries | native | by encoding | native: foreign keys, unique, check | no declared depth; SQLite can count and cap its steps | knowledge; a probe port, not kept, agreed on the inference tests |

The rules and the validation can be expressed natively, or nearly so, in several of these:
Rego, SQL and Datalog most naturally. The refusal condition is expressible as a rule in any
of them; what no language provides is the runtime itself refusing to answer, with a
deterministic count, a cost derived beforehand and identical counts across implementations.
Each has one piece: Starlark the budget, CEL the estimate, SpiceDB the depth limit. For
rules plus validation over JSON in one language, Rego is the closest, from its documentation
only. Also considered and not checked here: answer set programming (clingo), Alloy, Flix,
CUE and Dhall; none is known to have the contract.

## 7. Shaxon against its incumbents

Shaxon expresses the validation tests: the integrity gate (references, uniqueness, enums,
id width and depth; 24 kinds of planted defect, 732 planted, found at their exact JSON
paths) and the five authorization examples over event trails. The measured comparison is
with SHACL, in the authz README
(`../../src/jaxson-shaxon-v0.3.1/examples/shacl/authz/README.md`): the same five examples
written in SHACL-SPARQL and run through pyshacl and Apache Jena, and two of them in SHACL Core
alone. Only the SHACL rows below are measured; the others are from documentation.

| Language | JSON-native input | Cross-element rules | Refuses past a declared budget | Evidence |
|---|---|---|---|---|
| Shaxon | yes | native: references, uniqueness, counts, sequence rules | yes, `RESOURCE_ERROR/STEPS`, the same budget as Jaxson | The gate, five examples (51 cases), 23 mutants killed |
| SHACL-SPARQL | no: lifted to RDF, array order lost, decimals need care | yes, through SPARQL, in all five examples | no: a budget is not expressible, and processor timeouts are undefined by the standard | 50 of 50 comparable cases agree with Shaxon; 19 mutants killed |
| SHACL Core | no, same lift | 2 of 5 examples; no arithmetic, so `rolling-quota` is not expected to fit | no | Same verdicts on 4,100 random trails per package; at 4,000 events 8 to 21 times faster than SHACL-SPARQL |
| Rego | yes | rules over JSON, closure by `graph.reachable` | no deterministic budget found | Documentation |
| CEL | typed, JSON-like | native validation rules (Kubernetes) | static estimate plus runtime limit | Probed (section 5) |
| CUE | yes | validation-oriented, terminates by design | no budget found | Documentation |
| JSON Schema | yes | no references between values (general knowledge, not checked here) | no | Not tried |

What Shaxon changes:

- One package produces the decision and the findings. SHACL gives a conforms-or-not report,
  and mapping it to allow or deny, and each result to a reason, is the caller's.
- A malformed request, a missing member or a string for a number, is refused by Shaxon
  (`EXECUTION_ERROR`). SHACL allowed all 6 such cases unless a closed structure shape was
  added for each request.
- A trail too long for the budget is a refusal in Shaxon and not expressible in SHACL.
- An alternative needs at least two systems, for example a rules language plus SHACL or CUE
  for validation, and the shared contract is what the combination loses.

Where it does not favour Shaxon:

- SHACL is a W3C standard with mature tools; Shaxon is a new language to learn, which counts
  against it on being well understood.
- The Shaxon packages are longer than the SHACL shapes, 13,975 characters against 9,611 for
  the five examples, although the Shaxon figure includes the program that replays the trail,
  which SHACL has no counterpart for.
- The accessgraph gate was never written in SHACL or any other language, so that comparison
  is open; the measured one covers the authorization examples.

## Update log

| Date | Checkpoint | Change |
|---|---|---|
| 2026-10-27 | r016 | First entry. |
| 2026-10-27 | r017 | T-39 ran on real-shaped data. Agreement holds: seven engines, 10,000 decisions, two variants. The program refused loudly when its derived limit was exceeded, but the limit was wrong by a factor of 2.21, so the claim of a cost derived before the run is withdrawn for data of another shape until T-41. The speed gap to CozoDB narrows to 1.2 to 1.5 times at this size. |
| 2026-10-27 | r019 | T-41 rebuilt the cost model from the program's loops. The two real-shaped databases are covered: the program runs inside its derived limit, using 48%, and the model over-predicts there by 1.68 times. The claim of a cost derived before the run is restored for the data run so far, with the attempts counted: the first model failed; the rebuilt one covered the data but failed a policy test on a contrived database, so controlled databases were added to the fit; a refinement that fitted synthetic data more tightly failed on the real-shaped data and was reverted. The real-shaped databases decided between versions, so they are validation data and not an untouched test. |
| 2026-10-27 | after r019 | The speed assessment is qualified: it measures the current runtime, not the language, and the planned Jaxson-to-Go compiler (jaxpiler) is outside it. |
| 2026-10-27 | after r019 | The document is reordered so that the summary comes first and speed is the last measured section; its content is unchanged except for the opening of the speed section, which now states the measured ordering without a verdict. Sections are renumbered: the old 1 (speed), 2 (limits), 3 (agreement) and 5 (real-shaped data) are now 4, 3, 1 and 2, and the old 4 is the summary. |
| 2026-10-27 | after r019 | The claim about inference is reframed, at the project's request: from doing more inference than the incumbents to doing at least as much in the same domain space, with the domain space stated. Status: shown against the Datalog engines run on this task, not tested against the policy engines. |
| 2026-10-27 | after r019 | The speed claim is reframed, at the project's request: from being better in speed or scale to being able to run at the speed of the optimized Go port, which the compiler being designed (jaxpiler) is meant to reproduce. The compiler is no longer described as outside the assessment. |
| 2026-10-27 | after r019 | Starlark was run on the task and CEL probed (section 5). Starlark agrees with the reference on generated and real-shaped data; its step budget is enforced and deterministic, with no limit derived before the run. CEL's static estimate is sound and exponential in the rounds of a closure. |
| 2026-10-27 | after r020 | Two investigations are integrated: the existing declarative languages against the three kinds of test (section 6), and Shaxon against SHACL and others for the validation role (section 7), with two summary rows. The finding is that no existing language covers the rules, the validation and the contract together, and that only the SHACL comparison is measured. |
