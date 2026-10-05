# The same five examples in SHACL-SPARQL

`examples/shaxon/authz` has five authorisation examples in which access follows
a trail of earlier actions. Here they are again in SHACL-SPARQL, so the two can be
compared on the same cases. Nothing is shared except the data: this folder
reads the Shaxon folder's `<name>.json` (for the default input) and
`<name>.cases.json` (51 cases), so the inputs and the expected decisions are
identical.

SHACL Core has no aggregation and cannot compare values across different
nodes, so all five use SHACL-SPARQL (`sh:sparql`). Whether the two that look
simplest, `chinese-wall` and `break-glass`, could be written in Core alone was
not tried.

SPARQL is not SHACL. `sh:sparql` is the separate SHACL-SPARQL extension, and
the SPARQL query bodies are about two fifths of the shape text (4577 of 11101
characters, comments removed) and all of the rule logic: the aggregates, the
cross-node comparisons, `FILTER NOT EXISTS` and the path expressions. What is
compared with Shaxon below is therefore Shaxon against SHACL-SPARQL, and no
result here says what SHACL Core alone can or cannot express. Where a result
is a property of SHACL itself (the RDF data model, the unordered report, no
step budget) the text says so; where it is a property of SPARQL (aggregation,
joins, negation) it says that.

## Run

```
./setup.sh                 # .venv with pyshacl
./setup.sh --jena          # also Apache Jena's jars (needs Java and Maven)
./run.sh cases             # 51 cases through pyshacl; exit 0 if all comparable ones agree
./run.sh jena              # the same cases through Apache Jena SHACL
./run.sh no-structure      # the rules alone: the 6 malformed requests get through
./run.sh mutants           # break each rule in turn; every mutant must be caught
./run.sh timing            # Shaxon, pyshacl and Jena at 100, 1000 and 4000 events, end-to-end and engine-only
./run.sh all               # everything that is installed
```

`SETUP.md` is the setup guide for each harness: what it needs, how to install
and launch it, what a good run looks like, and a troubleshooting table.
Apache Jena SHACL is the second processor (Java 17 or later; the runs here used
Java 21); the scripts fetch its jars with Maven from `pom.xml`.

Tested with pyshacl 0.40.1, Apache Jena SHACL 6.2.0 and rdflib 7.6.0. The Jena
run starts a JVM per case (about 0.7 s each) and needs about 17 MB of jars.

| File | Role |
|---|---|
| `<name>.shapes.ttl` | The SHACL shapes for one example; each rule is a named `sh:SPARQLConstraint` whose IRI is the Shaxon `constraintId` |
| `lift.py` | Lifts a case's JSON input into the RDF graph SHACL validates |
| `run_cases.py` | Runs every Shaxon case through pyshacl (or Jena, with `--processor jena`) and compares |
| `mutants.py` | Breaks one rule at a time and checks that `run_cases.py` notices (pyshacl) |
| `scale.py` | Times Shaxon, pyshacl and Jena on long trails and checks that their verdicts agree |
| `setup.sh`, `run.sh` | Install the harnesses and launch them; `SETUP.md` explains both |
| `pom.xml` | Names the Jena jars for Maven to fetch; nothing is built |

## Lifting the JSON

SHACL validates an RDF graph, so the trail has to be lifted first (`lift.py`).
An object becomes a blank node and each member a triple; an array becomes one
triple per element; strings, booleans and numbers become typed literals. The
decisions that mattered:

- **Order is lost.** RDF has no array order. Only `four-eyes-release` depends
  on it, and its events already carry `seq`; the SHACL rules compare `seq`
  where the Shaxon package replays the array in order.
- **Exact decimals need care.** A plain JSON-LD lift turns `0.1` into an
  `xsd:double`, which cannot hold it. `lift.py` keeps the digits as written
  and types a number with a fraction as `xsd:decimal`.
- **Data in member names has no RDF form.** `chinese-wall`'s `classOf` map
  (company name to class) is rewritten as a list of `{id, class}` records.

## Results

| | Shaxon | SHACL-SPARQL |
|---|---|---|
| Cases agreeing on decision and reasons | 51 of 51 (its own expectations) | 50 of 50 comparable |
| Cases with no SHACL meaning | | 1: a trail too long for `limits.steps` fails closed. SHACL has no step budget, so there is nothing to fail |
| Mutation check (a rule deliberately broken) | 23 mutants tried (`../../shaxon/authz/mutants.py`); all killed | 19 mutants tried (`mutants.py`); all killed, after one case was added for the one that survived, see below |

Mutants that survive point at gaps in the shared case file, not in either
implementation. An earlier, hand-run round found that no case had an approval
that predates its payment and none had an over-long chain that also had a
revoked link; a case for each was added. The scripted round (`mutants.py`,
19 mutants over all five rule sets) then found that no case had a withdrawal
by someone other than the approver, and a case for that was added; all 19 are
now killed. The earlier round's tally was not kept, so only the scripted one is
quoted. One mutant is left out of the script because it is equivalent: dropping
the horizon test from the owner rule in `delegation-chain` changes nothing, as
a link within the horizon always has a parent, so the rule cannot fire.

## A second processor: Apache Jena

The same shapes, the same lift and the same 51 cases were run through Apache
Jena SHACL 6.2.0 (`--processor jena`).

| Question | pyshacl 0.40.1 | Jena 6.2.0 |
|---|---|---|
| Decision and reasons on the 50 comparable cases | 50 agree | 50 agree, once reasons are read as described below |
| `sh:sourceConstraint` of a `sh:sparql` result | The constraint (`ex:CREATOR_CANNOT_RELEASE`) | The shape that holds it (`ex:RequestShape`) |
| Malformed requests, rules only (`--no-structure`) | Allows all 6 | Allows all 6 |
| Malformed requests, with `ex:RequestStructure` | Denies all 6 | Denies all 6 |
| Time at 4000 events, whole call | 408 to 524 ms | 1169 to 1566 ms, of which about 0.7 s is starting the JVM |

The one difference is in the report, not the verdict. Jena fills
`sh:sourceConstraint` with the shape that holds the `sh:sparql` link, so a
shape with two constraints cannot be told apart by that property; this folder
reads the reasons from pyshacl's `sh:sourceConstraint`, and for Jena from
`sh:resultMessage`, which works only because every constraint here has its own
`sh:message`. A harness that relied on `sh:sourceConstraint` would have
reported 26 of the 51 cases as different under Jena, every one of them a
reason mismatch with the decision unchanged. I did not find which of the two
reports follows the W3C text, so I do not say either is wrong. Each processor
is tied to a different way of naming the rule that fired, and in Shaxon the
`constraintId` is part of the language.

The timings are in the Speed section below, from `./run.sh timing`.

## What differed, measured

| Question | Shaxon | SHACL-SPARQL |
|---|---|---|
| A malformed request (missing member, string for a number): 6 cases | Refuses to judge: `EXECUTION_ERROR` | Allows all 6 when only the rules are written (`--no-structure`). With a closed structure shape per request (`ex:RequestStructure`) it denies all 6, as a violation, not an error |
| A trail too long for the budget | `RESOURCE_ERROR/STEPS`, a refusal | Not expressible. A processor may time out, which is not defined by the standard |
| The order of reasons | Fixed: the same list in the same order, pinned by the test | A set. The report is unordered, so `run_cases.py` compares sets |
| Staging (judge the rules only if the chain is within the horizon) | One `if` in the program | Repeated by hand as `FILTER NOT EXISTS` in each rule |
| A depth horizon | `maxDepth: 3` on the closure | Written out: three optional hops, and a fixed four-hop path for "goes further" |
| The decision itself | `{"decision", "findings"}` is produced by the same package | The report says conforms or not. Mapping that to allow or deny, and each result to a reason, is the caller's |

## Size

Rule text with comments stripped, in characters (not tokens):

| Example | Shaxon package | SHACL-SPARQL shapes |
|---|---|---|
| `four-eyes-release` | 4158 | 1937 |
| `chinese-wall` | 1676 | 1068 |
| `delegation-chain` | 3816 | 3397 |
| `rolling-quota` | 1959 | 1855 |
| `break-glass` | 2366 | 1354 |
| Total | 13975 | 9611 |

The Shaxon figure includes the program that replays the trail and assembles
the decision, which SHACL has no counterpart for, and JSON's punctuation. The
SHACL side needs `lift.py` (2.6 KB) on top.

## Speed

`./run.sh timing` (`scale.py`) runs the two examples that scale with the trail
at 100, 1000 and 4000 events, on fixed-seed inputs, in two profiles, and checks
that all engines reach the same verdict. The two answer different questions, and
the first one flatters Shaxon.

**End-to-end** is what a caller holding the JSON input pays for a verdict. SHACL
validates an RDF graph, so the SHACL figures include lifting the input to RDF
(`lift.py`); Jena also writes Turtle files and starts a JVM (about 0.7 s of every
Jena figure). Shaxon reads the JSON as it is and has no such step. Best of three,
in milliseconds:

| Example | Events | Shaxon steps | Shaxon (Go process) | pyshacl (Python) | Jena (JVM process) |
|---|---|---|---|---|---|
| `rolling-quota` | 100 | 473 | 2 | 49 | 725 |
| `rolling-quota` | 1000 | 4073 | 3 | 151 | 1004 |
| `rolling-quota` | 4000 | 16073 | 5 | 518 | 1438 |
| `chinese-wall` | 100 | 411 | 3 | 27 | 729 |
| `chinese-wall` | 1000 | 4011 | 4 | 111 | 942 |
| `chinese-wall` | 4000 | 16011 | 12 | 360 | 1234 |

**Engine-only** leaves the lift, process start and parsing out, and times the
validation alone on data already in each engine's own form: `shaxon.Run` on a
decoded package (`enginetime/main.go`), `pyshacl.validate` on a lifted graph, and
Jena's validator in one warmed-up JVM (`JenaTime.java`, compiled on the fly with
`javac`). The lift is printed beside them. Best of ten after a warm-up, in
milliseconds (`./run.sh timing-engine`):

| Example | Events | Shaxon | pyshacl | Jena | The lift (not counted) |
|---|---|---|---|---|---|
| `rolling-quota` | 100 | 0.14 | 40 | 4.8 | 7 |
| `rolling-quota` | 1000 | 0.33 | 99 | 5.9 | 81 |
| `rolling-quota` | 4000 | 0.58 | 296 | 7.9 | 223 |
| `rolling-quota` | 20000 | 2.4 | 1587 | 12.4 | 1428 |
| `chinese-wall` | 100 | 0.17 | 22 | 1.8 | 11 |
| `chinese-wall` | 1000 | 1.1 | 62 | 2.3 | 41 |
| `chinese-wall` | 4000 | 5.0 | 211 | 2.4 | 129 |
| `chinese-wall` | 20000 | 22 | 1056 | 3.9 | 902 |

(These figures are from after the engine's performance work: a compiled program,
a hand-written JSON reader, cheaper index builds, and JSON objects held as
short member lists with canonical keys instead of Go maps. An earlier run put
Shaxon at 11 to 17 ms for 4000 events, engine-only, and Jena ahead on both
examples. The Shaxon column was re-measured after the last of these changes,
with the input decoded into the engine's own form and the program compiled in
the warm-up run, so the compile is not in the figure; the pyshacl, Jena and lift
columns are from the run before it and do not depend on Shaxon.)

Two things stand out once the lift is out. pyshacl stays far behind (30 to 100
times slower end to end at 4000 events, 40 to 300 times engine-only), because its
validation alone costs about as much as its lift. And Jena, warmed up, behaves
differently on the two examples. On `rolling-quota`, a single fold written with
`aggregate`, Shaxon is faster at every size, 2.4 ms against 12.4 ms at 20000 events.
On `chinese-wall` Jena is faster from somewhere between 1000 and 4000 events and
grows much more slowly (3.9 ms at 20000 against Shaxon's 22): Shaxon builds two
indices with a computed key over every event, so its cost grows with the trail,
while Jena's figure moves little between 100 and 20000 events. That is consistent
with its SPARQL engine reaching one actor's events through an index; this was not
checked inside Jena. On `chinese-wall` the end-to-end advantage Shaxon shows over
Jena is mostly process start and the lift, not validation speed. The engine-only
comparison also favours Jena in another way: its figure is a warmed-up JVM, which
a caller starting a process per verdict does not get.

All five examples were later timed the same way (`./bench.sh run`, then
`./bench.sh report`, which writes `results/REPORT.md`). Shaxon is much the fastest
end to end on `rolling-quota`, `chinese-wall` and `delegation-chain` at every
size, and engine-only on `rolling-quota`. On `four-eyes-release` and
`break-glass` it is not: its wall time grows quadratically with the trail
(steps grow linearly), so at 4000 events it takes 2.8 s and 4.8 s, against about
8 and 144 ms for warmed-up Jena, and at 20000 events 80 s and 148 s; the cause
is in the engine and the package, not in the language (TRACKER PF3). pyshacl is
slower still on both (35 s and 51 s at 4000 events engine-only). Delegation
chain, engine-only, goes from Shaxon 4.1 ms (Jena 16) at 4000 events to Shaxon
28 ms (Jena 12) at 20000.

The rolling-quota package is a temporary copy with `limits.steps` raised, since
the shipped one refuses long trails on purpose. The figures compare a Go
binary, a Python library and a JVM, so they say little about the languages
themselves.

## Footprint

What each engine needs to run these examples, as measured here:

| | Shaxon | pyshacl 0.40.1 | Jena 6.2.0 |
|---|---|---|---|
| Runtime | One Go binary, 5.3 MB (`shaxonrun`) | Python and 9 packages (rdflib, owlrl and others), 36 MB installed | A JVM and 38 jars, 17 MB, fetched with Maven (about 42 MB cached) |
| Dependencies of the module | None (`go.mod` names only the Go version) | rdflib, owlrl, html5rdf, pyparsing, prettytable, wcwidth, packaging | Jena's own tree |
| Needs a lift from the JSON input | No | Yes (`lift.py`, 2.6 KB) | Yes, plus Turtle files on disk |
| Needs a step mapping the report to a decision | No, the package returns `{decision, findings}` | Yes, in the caller | Yes, and a different one: the rule is read from `sh:resultMessage` |
| Whole call at 4000 events | 27 to 29 ms | 408 to 524 ms | 1169 to 1566 ms, about 0.7 s of it JVM start |

None of this is large in absolute terms. It is heavy relative to a single
static binary, and it is a stack to keep patched and to adapt per processor,
which an RDF shop already running one may not mind. `sh:sparql` constraints
run in pyshacl with or without its `advanced` flag (checked on four of the five
examples), so that flag is not a hidden switch here.

## Where SHACL-SPARQL was the better fit

- This one is SPARQL's doing, not SHACL's. `rolling-quota` and
  `four-eyes-release` are single declarative constraints (SPARQL aggregates, or
  `FILTER NOT EXISTS`), where the Shaxon versions first compute a derived
  state in `program`: an `aggregate` for the quota, a `for` loop that replays
  the trail for `four-eyes-release`. About half the rule text for
  `four-eyes-release` (1937 against 4158 characters); for the quota the gap is
  small (1855 against 1959) since the `aggregate` instruction replaced the
  hand-written fold.
- Warm and in-process, with the lift out of the count, Jena validated the larger trails faster than Shaxon (Speed, engine-only): its figure barely moved between 100 and 20000 events, where Shaxon's fold grows with the trail.
- It is an existing standard with several processors, tooling, and a shared
  vocabulary for the report (`sh:focusNode`, `sh:sourceConstraint`).
- Rules over a graph that already is RDF need no lift at all.
