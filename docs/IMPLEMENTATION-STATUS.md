# Jaxson and Shaxon: implementation status

Updated: 2026-10-07. Snapshot of `src/jaxson-shaxon-v0.3.1` (Go, module `github.com/ha1tch/jaxson`, version 0.3.1).

This page is a summary. The phase-by-phase record, with every decision taken where a specification was silent, is `src/jaxson-shaxon-v0.3.1/TRACKER.md`; what shipped is in `CHANGELOG.md` beside it. Where this page and the tracker disagree, the tracker is right.

**Status categories**

- **Runnable**: implemented in the Go package and exercised by tests or examples.
- **Partial**: implemented, with a stated gap.
- **Specified only**: in a design document, not implemented.

The distinction is conservative: parsing or static checking alone does not count as a runnable feature.

## Summary

| Layer | Status |
|-------|--------|
| Jaxson language and runtime | Runnable |
| Jaxson extension mechanism (`Dialect`, host instructions and operators) | Runnable |
| Jaxson tooling and examples | Runnable |
| Shaxon declarations, registries, static checks | Runnable |
| Shaxon shape evaluation, indices, relations, targets, `validate`, `check`, `aggregate` | Runnable |
| Shaxon reports and the `shaxon.Run` pipeline | Runnable |
| `shaxon.Validate` public API | Runnable |
| Shaxon conformance fixtures | Runnable; 196 fixtures, plus 71 engine mutants judged by them (69 killed, 2 recorded survivors) |
| Walk profile (v0.3.2 proposal) | Specified only |
| Delta and graph-transformation research | Not in this repository |

## 1. Jaxson

| Feature | Status | Evidence |
|---------|--------|----------|
| Package loading and normalisation | Runnable | `pkg/jaxson`, 28 golden fixtures |
| Core execution: `jaxson.Run`, `RunJSON`, `Profile` | Runnable | Fixtures and unit tests |
| Paths, dynamic path segments | Runnable | Tested |
| `set`, `append`, `insert`, `delete`, `halt` and the other core instructions | Runnable | Tested, including array-shift semantics and mutation reporting |
| `$compute` and its operators | Runnable | The showcase covers all 30 operators |
| Lazy `and`, `or`, `select` | Runnable | Fixtures |
| Exact decimal arithmetic, no implicit coercion, overflow as an error | Runnable | Fixtures; `jaxson.Number` for the API boundary |
| `null` against missing; templates and optional values | Runnable | Fixtures |
| Loops over a snapshot, nested loops, step limits | Runnable | Tested |
| Static checking, input and output contracts, closed objects | Runnable | `Checker`, schema fixtures |
| Mutation notification (`Machine.OnMutate`) | Runnable | Tested |
| Host instructions, host operators, shared step accounting | Runnable | Extension tests |
| Exported host primitives (`Walk`, `GetAt`, `Eval`, `RunCompute`, `Clone`, `Equal`, `TypeName`, `Order`, `SortedKeys`, `FormatDecimal`) | Runnable | Tested |
| Closure compiler | Runnable | Checked against the tree interpreter, which is kept as a test oracle until tracker item PF1 removes it |
| `Object` value type, `ParseData`, `ParsePackage` | Runnable | Parse differential tests; public edges still use `map[string]any` |
| `jaxtools.Session`, `jaxplay`, `jaxrun` | Runnable | Tested |
| Fluent `build` package | Runnable | Navy Wars port |
| Showcase examples | Runnable | 12 packages, 29 fixture cases |

Jaxson is a tested language and runtime with an extension surface, tooling and examples.

## 2. Shaxon

| Feature | Status | Evidence |
|---------|--------|----------|
| Shape, index, relation and compute declarations; error categories | Runnable | Parsing and static checks, `pkg/shaxon` |
| `extends` merging, the `override` mechanism, cycle detection | Runnable | Tested |
| Recursive shape analysis and `maxShapeDepth` | Runnable | Static analysis and evaluation both tested |
| Shape evaluation: kinds, fields, `closed`, combinators, `qualified`, nested and recursive shapes | Runnable | Fixtures |
| Indices, uniqueness, relations, cardinality | Runnable | Fixtures |
| Targets and path expressions (`$altPath`, `$path*`, `$path+`, `$inverse`) | Runnable | Fixtures |
| `validate` and `check` instructions; gate and report modes | Runnable | Fixtures |
| `aggregate` (`count`, `sum`, `min`, `max`) | Runnable | Sugar expanded before checking |
| Validation purity (no writes during `check` or `validate`) | Runnable | Enforced and tested |
| Step-cost model | Runnable | Cost-table tests |
| `shaxon.Run` pipeline and `cmd/shaxonrun` | Runnable | Run tests, `examples/shaxon` |
| `shaxon.Validate`, `ValidateJSON` | Runnable | API tests, including a concurrent-use test |
| Conformance fixtures | Runnable | 196 fixtures in `pkg/shaxon/shaxon-v0.3.1-fixtures.json` |
| Access-by-trail examples | Runnable | Five packages with 51 cases, in `examples/shaxon/authz` |

### Not implemented

| Capability | Status |
|------------|--------|
| Composable property-path algebra, structural targeting (`$conforms`), per-constraint severity, regular-expression string constraints, cross-document shape imports | Specified as gaps in `shaxon-v0.3.1-limitations.md` sections 1 to 5; not implemented |
| `weighted-1` cost table, value-copy charging (CM-3), declared host functions (CM-4) | Not started; tracker "Step-cost model" |
| Entailment, arbitrary undeclared relational reach | Stated design boundary (limitations sections 6 and 7), not planned |

### Open points

| Item | State |
|------|-------|
| Spec rulings P1, V1, V3, G11, R1, S4, S6, S9 | Settled 2026-10-07: the implementation's behaviour written into the specification, pinned by 14 new fixtures; see the tracker. G11 was then reversed (a child may only narrow what it extends): layer 1 is done and pinned by 19 more; layers 2 and 3 are open and designed in `docs/proposals/extends-narrowing.md` |
| Removal of the tree-interpreter oracle (PF1) | Open |
| `FuzzParseJSON` as a dormant guard (PF2) | Recorded in the tracker's dormant-guards table |

## 3. Integration

| Capability | Status |
|------------|--------|
| Jaxson as the execution substrate for Shaxon | Runnable |
| Shaxon packages calling Jaxson operand and path checking | Runnable |
| Shared mutation hooks and step accounting | Runnable |
| Complete Jaxson to Shaxon pipeline with a report | Runnable (`shaxon.Run`) |
| Walk profile (bounded sessions over a reference graph) | Specified only; `shaxon-v0.3.2-walk-proposal.md` |

## 4. Comparison with SHACL engines

The five authorisation examples are also written as SHACL-SPARQL shapes and run through pyshacl and Apache Jena (`src/jaxson-shaxon-v0.3.1/examples/shacl/authz`). The results are in its `results/REPORT.md`; the summary is in the repository README. At 4000 events Shaxon is ahead of warmed-up Jena engine-only on all five, and at 20000 on three, level on one and 1.4x behind on one (`chinese-wall`; tracker item PF4). These are measurements on one host with the harness's own generated trails, and the harness is reproducible on Linux and macOS with `bench.sh`.

## 5. Summary

Jaxson is a runnable, tested language and runtime. Shaxon v0.3.1 is implemented end to end in Go and checked against its conformance fixtures; the remaining work is the Walk profile proposal, the retirement of the tree-interpreter oracle (PF1) and the gaps listed in the limitations document.
