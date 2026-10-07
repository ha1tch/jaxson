# Jaxson

**Jaxson** language 0.1.0 · **Shaxon** specification 0.3.1 · Go implementation 0.3.1

Jaxson is a small, closed language for writing **deterministic logic as JSON**. A package bundles an input, a program, and contracts for the input and the output. Any conforming runtime gives it the same result, or the same error, using exact decimals, no coercion, guaranteed termination and no I/O. The step count is part of the result: a run that exceeds its declared limit does so at the same point on every runtime.

```json
{
  "jaxson": "1.0",
  "input": {"a": 0.1, "b": 0.2},
  "inputSchema": {"type": "object", "fields": {"a": {"type": "number"}, "b": {"type": "number"}}, "required": ["a", "b"]},
  "program": [
    {"op": "set", "path": ["output"], "value": {"$compute": {
      "with": {"a": {"$path": ["input", "a"]}, "b": {"$path": ["input", "b"]}},
      "expr": ["add", {"$v": "a"}, {"$v": "b"}]
    }}}
  ],
  "outputSchema": {"type": "number"}
}
```

Saved as `sum.json`, `go run ./cmd/jaxrun sum.json` (from the Go directory below) prints `0.3`. Calculation happens in `$compute` islands, small pure expressions whose operands are bound up front, so they cannot read or change state.

## How Jaxson and Shaxon relate

Jaxson computes; Shaxon validates. Shaxon is a **dialect** mounted on Jaxson, not a second language and not a change to it. The Jaxson core stays closed and unchanged. Shaxon plugs in through the core's extension mechanism (`Dialect`): host instructions (`validate`, `check`), host operators, extra operand forms (`$altPath`, `$path*`, `$path+`, `$inverse`), and hooks that observe mutations. A Shaxon package is therefore a Jaxson package that also carries declarations, namely shapes, indices and relations, and runs through the same machine.

Because of that, validation and computation are phases of one execution model. They share one step budget, one error model and one definition of a path or operand, so the guarantee Jaxson gives (same package and input, same result and same step count, on every conforming runtime) holds for a validation report as well as for a computed value. Shaxon's constraint layer is derived from SHACL's model of node shapes, property constraints, severities, qualified counts and logical combinators, redesigned around a JSON tree instead of an RDF graph. Constraints that SHACL would send to SPARQL or a JavaScript extension are written in the same total, step-bounded sublanguage that runs the computation. Shaxon is not SHACL written in JSON.

Where each stands: the Jaxson core is specified in `jaxson-v0.1.0-core-design.md` and checked by 28 golden fixtures. The current Shaxon specification is v0.3.1 (`shaxon-v0.3.1-core.md`), checked by 175 conformance fixtures. Both documents are labelled proposals, and the Go implementation runs both end to end. A later Walk profile (`shaxon-v0.3.2-walk-proposal.md`) is proposed and not implemented. The points the specification left open were settled by writing the implementation's behaviour into it and pinning each with a fixture (tracker rows P1, V1, V3, G11, R1, S4, S6, S9); known limits are in `shaxon-v0.3.1-limitations.md`.

## Try it

The Go implementation lives in [`src/jaxson-shaxon-v0.3.1`](src/jaxson-shaxon-v0.3.1) and needs Go 1.25 or later.

```
cd src/jaxson-shaxon-v0.3.1
go test ./...
go run ./cmd/shaxonrun -pretty examples/shaxon/orders.json
```

`cmd/jaxrun` runs a plain Jaxson package, `cmd/jaxplay` drives a multi-turn one, and `cmd/shaxonrun` runs a Shaxon package. The `orders.json` run validates three orders against shapes and an index and prints a report with its violations (a dangling customer reference, a quantity below the minimum) beside the program's output.

## How it is built

- **Execution.** Programs are compiled once to closures over a scratch stack, with a borrow analysis that reads a state value without copying it when an island only inspects it. The original tree interpreter is kept as a differential oracle until it is retired (tracker item PF1).
- **Values.** One value model (`Object`, exact `Number`) from parse to result; the public edges still take `map[string]any`.
- **Indices and relations.** Built on first use, charged by element, and reused until a mutation overlaps a path they depend on, so index reuse is a rule of the specification and cannot vary between runtimes. A string key that is a `concat` of bound values is built into a byte slab with a parallel-array hash table.
- **Cost model.** Every instruction, operator and index element is charged against a cost table, so a step limit is a bound on work as well as on length.
- **Embedding.** `shaxon.Validate` and `ValidateJSON` check a document against declared shapes and are safe for concurrent use; the host primitives (`Walk`, `GetAt`, `Eval`, `RunCompute`, `Clone`, `Equal`, `Order`) are exported for dialects of your own; `pkg/jaxson/build` is a fluent, type-checked builder for writing packages in Go.

## How well it is checked

| What | Evidence |
|------|----------|
| Jaxson language | 28 golden fixtures plus 29 showcase cases, each with an expected result |
| Shaxon specification | 196 conformance fixtures, each with a note saying what a wrong runtime would do differently; written to be read by any implementation (`pkg/shaxon/FIXTURES.md`) |
| The fixtures themselves | 71 deliberately broken engine variants (mutants) judged by the fixtures: 69 rejected, 2 recorded with the reason they survive |
| The compiler | Compiled and tree-interpreted runs compared over every fixture and example, including at tightened step limits, for result, error and step count |
| The parser | Differential tests of `ParseData` against a reference parser; a fuzz target recorded as a dormant guard |
| Concurrency and hygiene | About 460 tests and subtests, run under the race detector, with `go vet` clean |
| Access-by-trail examples | Five packages with 51 cases; the same rules written as SHACL-SPARQL and run through pyshacl and Apache Jena, which reach the same verdicts; 19 mutants of the rules, all caught by the shared cases |
| Speed | A reproducible harness against pyshacl and Jena on trails up to 20000 events, below |

Decisions taken where a specification was silent, and what is still open, are in `src/jaxson-shaxon-v0.3.1/TRACKER.md`; every change is in its `CHANGELOG.md`.

## Examples and harnesses

- `examples/showcase/`: twelve worked Jaxson packages across everyday logic, finance and machine-learning operations, with 29 fixture cases and a coverage report.
- `examples/game/` and `src/jaxson-shaxon-v0.3.1/examples/navywars`: a game written first as hand-built packages and then ported to the Go builder.
- `src/jaxson-shaxon-v0.3.1/examples/shaxon/`: Shaxon packages to run, including five access-control examples (rolling quota, chinese wall, delegation chain, four-eyes release, break glass), each with a case file.
- `src/jaxson-shaxon-v0.3.1/examples/shacl/authz/`: the SHACL side of those five, the lift from JSON to RDF, a rule-mutation check, and the timing harness (`bench.sh`, `scale.py`, `report.py`) with set-up scripts for Linux and macOS and the recorded results.

## Repository layout

| Path | Contents |
|------|----------|
| `jaxson-intro.md`, `jaxson-v0.1.0-core-design.md`, `jaxson-v0.1.0-fixtures.json` | The Jaxson introduction, core design and its 28 golden fixtures |
| `shaxon-v0.3.1-*.md` | The current Shaxon specification: core, limitations, editorial, and the proposal it followed |
| `shaxon-v0.3.2-walk-proposal.md` | Proposal for a later version (the Walk profile); not implemented |
| `docs/` | [Implementation status](docs/IMPLEMENTATION-STATUS.md), [precedents](docs/PRECEDENTS.md), and the six-part SHACL comparison (`shacl-01` to `shacl-06`) |
| `examples/showcase/` | Twelve worked Jaxson packages with fixtures |
| `examples/game/` | The earliest Navy Wars game as Jaxson packages, with the Python and shell scaffolding that built and ran it |
| `src/jaxson-shaxon-v0.3.1/` | The current Go implementation: `pkg/jaxson`, `pkg/shaxon`, `pkg/jaxtools`, three commands, and its own examples; its [README](src/jaxson-shaxon-v0.3.1/README.md) has the package layout, [TRACKER](src/jaxson-shaxon-v0.3.1/TRACKER.md) the status and decisions, [CHANGELOG](src/jaxson-shaxon-v0.3.1/CHANGELOG.md) what changed |
| `src/jaxson-v0.1.0/`, `attic/`, `shaxon-v0.3.0-*.md` | The first Go implementation, earlier drafts and the previous Shaxon specification, kept for history |

## Reading

- [An Introduction to Jaxson](jaxson-intro.md) and the [Jaxson core design](jaxson-v0.1.0-core-design.md)
- [Shaxon v0.3.1 core](shaxon-v0.3.1-core.md), its [limitations](shaxon-v0.3.1-limitations.md) and its [editorial](shaxon-v0.3.1-editorial.md)
- [Implementation status](docs/IMPLEMENTATION-STATUS.md), feature by feature
- [Precedents](docs/PRECEDENTS.md): comparable systems, and how Jaxson differs
- [Showcase](examples/showcase/README.md): worked examples across a range of uses

## Shaxon and SHACL

Shaxon is not SHACL written in JSON; its model differences are set out in the SHACL comparison:
[technical primer](docs/shacl-01-technical-primer.md),
[primitive-by-primitive comparison](docs/shacl-02-comparison.md),
[coverage](docs/shacl-03-coverage.md),
[what is not covered](docs/shacl-04-not-covered.md),
[real-world graph databases](docs/shacl-05-real-world-graphs.md) and
[integration with Xolu](docs/shacl-06-xolu-integration.md).
Work continues towards the expressiveness of the SHACL 1.2 draft where the model differences allow.

### Features and idioms compared

What each can express and how. "Yes" means a native construct; otherwise the cell names the route or the gap. The Shaxon column follows `shaxon-v0.3.1-limitations.md`; the SHACL columns are SHACL 1.0 Core and the SHACL-SPARQL extension (`sh:sparql`), not the 1.2 drafts.

| Capability or idiom | Jaxson | Shaxon | SHACL Core | SHACL-SPARQL |
|---|---|---|---|---|
| Data model | JSON tree (`input`, `state`, `output`, `local`) | JSON tree | RDF graph | RDF graph |
| General computation (arithmetic, branching, loops) | Yes: 30 operators, `if`, loops over a snapshot | Yes, inherited; a `$compute` can be a constraint | No | Expressions and aggregates inside a query; no loops |
| Exact decimal arithmetic, no coercion | Yes | Yes | Datatypes compared, not computed | `xsd:decimal` arithmetic, with promotion in mixed expressions |
| Programs that change state | Yes (`set`, `append`, `insert`, `delete`) | Yes; validation itself is pure | No | No |
| Guaranteed termination and a step budget | Yes | Yes, validation included | No; recursive shapes are processor-defined | No |
| Same result and same step count on every runtime | Yes | Yes | Verdict defined; report unordered; no step count | As Core, and engine-defined query evaluation |
| Input and output contracts | Yes, closed by default | Yes (shapes) | Yes (shapes) | Yes |
| Scalar kind, numeric range, enum, string length | Yes (schema keywords) | Yes (`kind`, `min`, `max`, `enum`, `minLen`, `maxLen`) | Yes (`sh:datatype`, `sh:min*`, `sh:in`, `sh:minLength`) | Yes |
| Regular-expression string constraint | No | No (limitations section 4) | Yes (`sh:pattern`) | Yes (`REGEX`) |
| Closed shapes | Yes (`extra`) | Yes (`closed`) | Yes (`sh:closed`) | Yes |
| And, or, not, exactly-one | As compute operators | Yes (`and`, `or`, `xone`, `not`), ordered and short-circuiting | Yes | Yes |
| Qualified counts | By hand in a loop | Yes (`qualified`) | Yes (`sh:qualifiedValueShape`) | Yes (`COUNT`) |
| Nested and recursive shapes | Not applicable | Yes, bounded by `maxShapeDepth` | Nested yes; recursion left to the processor | As Core |
| Shape reuse by extension | Not applicable | Yes (`extends`, with override rules) | No | No |
| Property paths | Fixed or computed path segments | `$altPath`, `$path*`, `$path+`, `$inverse`; no composable path algebra (limitations section 1) | Full property paths | Full SPARQL property paths |
| Reference integrity (no dangling ids) | By hand with `has` | Yes (`kind: reference` over a declared index) | Class and node conformance only | Yes (`FILTER NOT EXISTS`) |
| Uniqueness of a field across a collection | By hand | Yes (`unique`) | No (only `sh:uniqueLang`) | Yes (`GROUP BY ... HAVING`, or a self-join) |
| Comparison across different nodes (joins) | By hand | Through declared indices and relations; no undeclared reach (limitations section 7) | Between paths of one focus node | Yes, arbitrary joins |
| Aggregation over a collection | Loops | Yes (`aggregate`: count, sum, min, max) | No | Yes |
| Selecting what to validate | Not applicable | `$each`, `$discriminator`, `$indexed`; structural targeting not implemented (limitations section 2) | `sh:targetNode`, `Class`, `SubjectsOf`, `ObjectsOf` | As Core, plus query-selected nodes |
| Validation inside a running program | Not applicable | Yes (`check`); `gate` or `report` mode | No | No |
| Severity per constraint | Not applicable | No (limitations section 3) | Yes (`sh:severity`) | Yes |
| Entailment and inferred data | No | No (limitations section 6) | Processor-dependent | Processor-dependent |
| Shapes shared across documents | No | No (limitations section 5) | Yes (graph imports, processor-dependent) | As Core |
| Constraint ids in reports | Not applicable | Yes (`constraintId`) | Source shape and component | As Core |

### Compared with SHACL engines

Five access-control examples (rolling quota, chinese wall, delegation chain, four-eyes release, break glass) are written both as Shaxon packages and as SHACL-SPARQL shapes, and timed on trails of growing length against pyshacl and Apache Jena.

- End to end (process start, parsing and the RDF lift included), Shaxon is the fastest of the three on all five examples at every size measured.
- Engine only, with the RDF lift and start-up left out, Shaxon is ahead of a warmed-up Jena on all five examples at 4000 events (by 23x to 1.2x). At 20000 events it is ahead on `rolling-quota` (18x), `break-glass` (2.9x) and `four-eyes-release` (1.2x), level on `delegation-chain`, and behind on `chinese-wall` (Jena 1.4x faster). pyshacl is far slower than both.
- The five were re-run together on one host. Jena's own figures move by up to a third between runs, so the smaller ratios are not firm.

### Timings, all five examples

Milliseconds, on trails of 100 to 20000 events. **End to end** is the JSON input to a verdict: Shaxon is the whole Go process; pyshacl and Jena include the RDF lift, and Jena the JVM start. **Engine only** is validation alone on data already in each engine's form (a warmed-up JVM for Jena). End-to-end figures are the best of up to 3 runs, engine-only the best of up to 10 after a warm-up. "skipped" means the harness's 120 s budget projected the cell too slow to run. Every cell where more than one engine ran returned the same allow/deny verdict.

| Example | Events | Shaxon steps | E2E Shaxon | E2E pyshacl | E2E Jena | Engine Shaxon | Engine pyshacl | Engine Jena |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `rolling-quota` | 100 | 473 | 4.2 | 73 | 1,366 | 0.21 | 66 | 7.5 |
| `rolling-quota` | 1,000 | 4,073 | 5.7 | 226 | 1,708 | 0.37 | 138 | 16 |
| `rolling-quota` | 4,000 | 16,073 | 8.5 | 813 | 2,814 | 0.82 | 447 | 19 |
| `rolling-quota` | 20,000 | 80,073 | 49 | 5,446 | 7,427 | 3.6 | 2,523 | 63 |
| `chinese-wall` | 100 | 411 | 4.3 | 44 | 1,219 | 0.20 | 27 | 3.1 |
| `chinese-wall` | 1,000 | 4,011 | 5.5 | 148 | 1,524 | 0.82 | 85 | 4.3 |
| `chinese-wall` | 4,000 | 16,011 | 10 | 523 | 2,025 | 3.3 | 303 | 3.8 |
| `chinese-wall` | 20,000 | 80,011 | 42 | 3,548 | 5,668 | 10 | 1,934 | 7.6 |
| `four-eyes-release` | 100 | 1,546 | 4.7 | 145 | 1,149 | 0.32 | 71 | 5.1 |
| `four-eyes-release` | 1,000 | 15,101 | 6.4 | 2,622 | 1,593 | 1.4 | 2,312 | 5.6 |
| `four-eyes-release` | 4,000 | 60,339 | 18 | 42,964 | 2,437 | 6.9 | 38,740 | 12 |
| `four-eyes-release` | 20,000 | 301,568 | 92 | skipped | 7,291 | 31 | skipped | 36 |
| `delegation-chain` | 100 | 281 | 4.3 | 122 | 1,084 | 0.49 | 87 | 12 |
| `delegation-chain` | 1,000 | 2,081 | 5.1 | 408 | 1,374 | 0.88 | 326 | 10.0 |
| `delegation-chain` | 4,000 | 8,081 | 13 | 1,461 | 2,057 | 3.2 | 1,224 | 8.5 |
| `delegation-chain` | 20,000 | 40,081 | 49 | 8,297 | 6,663 | 19 | 6,223 | 20 |
| `break-glass` | 100 | 1,969 | 7.7 | 1,717 | 1,595 | 1.2 | 1,220 | 26 |
| `break-glass` | 1,000 | 19,398 | 22 | 14,148 | 2,541 | 14 | 13,200 | 87 |
| `break-glass` | 4,000 | 77,617 | 80 | 63,156 | 4,375 | 67 | 56,231 | 275 |
| `break-glass` | 20,000 | 387,551 | 468 | skipped | 12,180 | 334 | skipped | 967 |

Measured on 2026-10-07 on one 2-CPU Xeon VM (Go 1.27, Java 21, pyshacl 0.40.1, rdflib 7.6.0). Jena's own figures move by up to a third between runs, so ratios under about 1.5x are not firm. The full report, with the ratios, is `results/REPORT.md`.

The harness, the numbers and the set-up scripts for Linux and macOS are in [`examples/shacl/authz`](src/jaxson-shaxon-v0.3.1/examples/shacl/authz), with the generated [report](src/jaxson-shaxon-v0.3.1/examples/shacl/authz/results/REPORT.md). The Shaxon side is in [`examples/shaxon/authz`](src/jaxson-shaxon-v0.3.1/examples/shaxon/authz).
