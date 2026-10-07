# Access decided by a trail of actions

Five Shaxon packages in which whether a request is allowed does not depend on
who is asking, but on what has already happened: a log of earlier actions.
Each package takes the trail and a request as `input` and returns

```json
{ "decision": "allow" | "deny", "findings": [ { "constraintId": "...", "focusPath": [...], ... } ] }
```

Every finding is a core section 10 violation: a stable id, and a `focusPath`
that points back into the input, so a reason can be traced to the event or the
request member behind it.

Run one, with its default input or with another input file:

```
go run ./cmd/shaxonrun -pretty -steps examples/shaxon/authz/chinese-wall.json
```

`<name>.cases.json` lists the inputs that `TestAuthzExamples`
(`pkg/shaxon/authz_examples_test.go`) pins: 51 cases, each with its decision
and its reasons. A run that fails, for example by running out of steps, is a
refusal, not an allow.

| Package | What the trail decides | Shaxon features used |
|---|---|---|
| `four-eyes-release` | A payment can be released only if someone other than its creator and the releaser approved it after its last amendment, and has not withdrawn the approval | A program replays the log into `state`; `check` judges the request against it; one stable id per rule |
| `chinese-wall` | A person who has opened one company in a conflict class cannot open another in the same class, but may return to the first | Two `multi` indices with computed keys (actor and class, actor and company); `$inverse` counts; no loops |
| `delegation-chain` | A delegated capability works only if every link in its provenance chain is unrevoked, unexpired, within its allowed uses, and the chain ends at the resource owner, within a depth horizon | `$path*` closure as a `check` target; indices over revocations and uses; `PATH_DEPTH_EXCEEDED` as a finding; exact decimal expiry times |
| `rolling-quota` | An export is allowed only if this person's exports in the trailing 24 hours plus this one stay within a volume limit and a count limit | `aggregate` (a count and an exact decimal sum) over the trail (0.1 + 0.2 + 0.3 + 0.3 is exactly 0.9, not 0.9000000000000001); `limits.steps` as fuel, so a trail too long to judge fails closed |
| `break-glass` | Emergency access is refused while any earlier emergency use by the same person has no review by someone else, and after two uses | `$each` over the trail with `$inverse` inside the shape, so findings land on the offending earlier events |

## What this buys, and what it does not

A policy written as a Shaxon package is one document with a fixed meaning: the
same trail and request give the same decision, the same reasons and the same
step count on any conforming runtime, in exact decimal arithmetic, with the
cost capped by `limits.steps`. Role- and attribute-based
engines usually decide from the facts about one request; the sequence rules
here need a derived state, which normally lives in application code outside
the policy. Here the replay, the rules and the reasons are in the package.

Two limits shape these examples. The language has no filter or map
operator. `rolling-quota` counts and sums the trail with the `aggregate`
instruction (core section 8a), which is sugar for a `for` fold; the
replay in `four-eyes-release` is a `for` loop written out in `program`, and
the rest are purely declarative. And the trail is input: the package judges it, it does not
establish that the events are genuine or in order, nor read a clock.

## Against SHACL-SPARQL

The same five examples are written in SHACL-SPARQL in `examples/shacl/authz`
and run over these case files with two processors, pyshacl 0.40.1 and Apache
Jena SHACL 6.2.0. That folder's README has the full comparison and its `SETUP.md` how to rerun it (`./setup.sh`, then `./run.sh all`); this is how
the three stand on the points that were measured.

| Question | pyshacl 0.40.1 | Jena 6.2.0 | Shaxon |
|---|---|---|---|
| Decision and reasons, 50 comparable cases | 50 agree | 50 agree, with reasons read from `sh:resultMessage` | 50 agree. The expectations are Shaxon's own, held by `TestAuthzExamples`, which also passes the 51st case |
| What names the rule that fired | `sh:sourceConstraint` is the constraint, e.g. `ex:CREATOR_CANNOT_RELEASE` | `sh:sourceConstraint` is the shape that holds it, e.g. `ex:RequestShape` | Each finding carries `constraintId` (e.g. `CREATOR_CANNOT_RELEASE`), plus `shape`, `focusPath` and `message`, as a field of the language |
| Malformed requests, rules only | Allows all 6 | Allows all 6 | Not run without the request shape, which is part of each package |
| Malformed requests, with structure shapes | Denies all 6, as violations | Denies all 6, as violations | Refuses all 6 with `EXECUTION_ERROR` and produces no decision |
| Time at 4000 events, end to end | 360-518 ms | 1234-1438 ms, of which about 0.7 s is JVM start | 8-18 ms for `rolling-quota`, `chinese-wall`, `four-eyes-release` and `delegation-chain` (16073, 16011, 60339 and 8081 steps); 80 ms for `break-glass` (77617 steps) |
| Time at 4000 events, validation only (lift, process start and parsing left out) | 211-296 ms | 2.4-7.9 ms (warmed-up JVM) | 0.8-6.9 ms for the first four examples; 67 ms for `break-glass` |

All five SHACL versions use `sh:sparql`, the SHACL-SPARQL extension, which
carries the rule logic, so the SHACL columns are SHACL-SPARQL, not SHACL Core.
The timings come from `./run.sh timing` in `examples/shacl/authz` (`scale.py`:
fixed-seed inputs, best of three). Shaxon is the whole `shaxonrun` process;
pyshacl and Jena include the RDF lift, and Jena the JVM start. The
validation-only row comes from `./run.sh timing-engine`, which leaves the lift out
and favours the SHACL engines; it is the fairer reading when the data already is
RDF. Both are indicative only. The shipped `rolling-quota`
package has `limits.steps` of 600 and refuses 1000 or more events with
`RESOURCE_ERROR/STEPS`; for the timing the limit was raised in a copy. SHACL
has no equivalent of that ceiling.

The two failure modes differ. SHACL turns a malformed request into a
violation, so the answer is deny. Shaxon returns an error and no decision, so
the caller has to handle it and an allow cannot be mistaken for a pass.
