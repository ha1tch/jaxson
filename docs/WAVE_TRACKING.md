# Wave tracking

## 1. Progress at a glance

```
Wave 0  Extract, don't change       ████████████████████   100%  (6/6 items)
Wave 1  Extension points Shaxon needs  ████████████████████   100%  (6/6 items)
Wave 2  Shaxon's static layer       ████████████████████   100%  (9/9 items)
Wave 3  Shape evaluation            ████████░░░░░░░░░░░░    38%  (5/13 items)
Wave 4  Indices, relations, targets, validate  ████████████████████   100%  (7/7 items)
Wave 5  shaxon.Run, version, pipeline order  █████████░░░░░░░░░░░    47%  (8/17 items)
Wave 6  Fixtures and conformance    ███████████░░░░░░░░░    57%  (4/7 items)
Wave 7  Public API, Session, build package  ████████████████████   100%  (5/5 items)
```

Overall by item count: 50 of 70 items ≈ **71%**

### Wave 0 — Extract, don't change (6 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 1 | `pkg/jaxson` extracted as its own package, five files | ✓ | not yet filed |
| 2 | `go build ./...` clean | ✓ | not yet filed |
| 3 | `go vet ./...` clean | ✓ | not yet filed |
| 4 | 28 of 28 Jaxson fixtures pass | ✓ | not yet filed |
| 5 | CLI moved to `cmd/jaxrun` | ✓ | not yet filed |
| 6 | Fixture loop turned into a table-driven `go test` | ✓ | not yet filed |

**Wave 0: 6/6, done.**

### Wave 1 — Extension points Shaxon needs (6 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 7 | Instruction table (`InstructionDef`, `CoreInstructions`) | ✓ | not yet filed |
| 8 | Operator table (`OperatorDef`, `CoreOperators`) | ✓ | not yet filed |
| 9 | Static-check context (`Checker`) | ✓ | not yet filed |
| 10 | Mutation hook (`Machine.OnMutate`) | ✓ | not yet filed |
| 11 | Six primitives exported (`Clone`, `Equal`, `TypeName`, `Order`, `SortedKeys`, `FormatDecimal`) | ✓ | not yet filed |
| 12 | Schema-keyword checks factored out (`CheckNumber`, `CheckStringLen`, `CheckArrayLen`) | ✓ | not yet filed |

**Wave 1: 6/6, done.**

### Wave 2 — Shaxon's static layer (9 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 13 | `indices`, `relations` and `computes` registries parsed and statically validated | ✓ | not yet filed |
| 14 | `shapes` parsed, `extends` resolved by default merging | ✓ | not yet filed |
| 15 | `extends` override mechanism | ✓ | not yet filed |
| 16 | `limits.maxShapeDepth` reachability check | ✓ | not yet filed |
| 17 | Keyword-vs-kind legality and cross-keyword ordering | ✓ | not yet filed |
| 18 | `extends` restricted to named top-level shapes | ✓ | not yet filed |
| 19 | Checks needing real data (repeated index keys, one-to-one, `unique`), delivered in phase 4 | ✓ | not yet filed |
| 20 | `indices.go`, `targets.go`, `check.go`, `run.go`, Shaxon fixtures and `cmd/shaxonrun`, delivered in phases 4 to 6 | ✓ | not yet filed |
| 21 | CHANGELOG entry for phase 2 | ✓ | not yet filed |

**Wave 2: 9/9, done.**

### Wave 3 — Shape evaluation (13 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 22 | Evaluator (`shapes.go`) and `report.go` | ✓ | not yet filed |
| 23 | `IndexResolver` stopgap closed in phase 4 | ✓ | not yet filed |
| 24 | G11 layer 1: a child shape may only narrow | ✓ | not yet filed |
| 25 | Step-cost model CM-1, the `jaxson` charging mechanism | ✓ | not yet filed |
| 26 | Step-cost model CM-2, the `shaxon` shape-activation charge | ✓ | not yet filed |
| 27 | G11-2a: inherit `severity` and `message` when the child states none | ☐ | not yet filed |
| 28 | G11-2b: rewrite the core section 4 merge table around narrowing | ☐ | not yet filed |
| 29 | G11-2c: property test that a child accepts only what its parent accepts | ☐ | not yet filed |
| 30 | G11-2d: counts, CHANGELOG, `FIXTURES.md` and mutants for layer 2 | ☐ | not yet filed |
| 31 | G11-3a: loosening flags for kind, items, `qualified` and keywords | ☐ | not yet filed |
| 32 | G11-3b: more than one `qualified` rule per shape | ☐ | not yet filed |
| 33 | G11-3c: inheriting a shape `id` | ☐ | not yet filed |
| 34 | G11-3d: a named shape and its inline copy compared as equal | ☐ | not yet filed |

**Wave 3: 5/13, in progress.**

### Wave 4 — Indices, relations, targets, validate (7 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 35 | 4.1 Indices and relations (`indices.go`) | ✓ | not yet filed |
| 36 | 4.2 Targets, `validate`, `unique` and the `check` instruction | ✓ | not yet filed |
| 37 | 4.3 Closures `$path*` and `$path+` as targets | ✓ | not yet filed |
| 38 | 4.4 Operand forms `$altPath`, `$inverse` and closures in operand position (`forms.go`) | ✓ | not yet filed |
| 39 | Operand-form hook in `pkg/jaxson/forms.go`, with its own tests | ✓ | not yet filed |
| 40 | Remove the stray `.ed-journal.json` from the author's tree before packaging | ✓ | not yet filed |
| 41 | Decide whether the work warrants a `VERSION` bump past 0.3.1 | ✓ | not yet filed |

**Wave 4: 7/7, done.**

### Wave 5 — shaxon.Run, version, pipeline order (17 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 42 | `shaxon.Run`, `RunJSON` and `Result` | ✓ | not yet filed |
| 43 | Strict `jaxson.ParseJSON` (duplicate keys, trailing data) | ✓ | not yet filed |
| 44 | `Profile.SchemasOptional` | ✓ | not yet filed |
| 45 | `cmd/shaxonrun` | ✓ | not yet filed |
| 46 | Example package `examples/shaxon/orders.json` | ✓ | not yet filed |
| 47 | `aggregate` instruction (count, sum, min, max) | ✓ | not yet filed |
| 48 | Authorisation examples: five packages, 51 cases | ✓ | not yet filed |
| 49 | SHACL-SPARQL comparison of the authorisation examples | ✓ | not yet filed |
| 50 | `costs.Validate` has no test that can fail | ☐ | not yet filed |
| 51 | R2: a report written into `output` lands after the schema check | ☐ | not yet filed |
| 52 | Aggregation inside a constraint (check expression or index key) | ☐ | not yet filed |
| 53 | Grouping, `any`/`all` and average not offered | ☐ | not yet filed |
| 54 | SHACL Core alone untested for `chinese-wall` and `break-glass` | ☐ | not yet filed |
| 55 | Which `sh:sourceConstraint` reading follows the W3C text | ☐ | not yet filed |
| 56 | `setup.sh` install branches unexercised end to end | ☐ | not yet filed |
| 57 | The 21 phase 5 source mutants are not reproducible | ☐ | not yet filed |
| 58 | repoman provenance flag on `CHANGELOG.md`; the fix belongs upstream in gorepoman | ☐ | not yet filed |

**Wave 5: 8/17, in progress.**

### Wave 6 — Fixtures and conformance (7 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 59 | Fixture runner with spec-fixed-only matching of violations | ✓ | not yet filed |
| 60 | Fixtures for the plan's priorities (196 fixtures pass) | ✓ | not yet filed |
| 61 | Six attic fixtures rewritten for 3.1 | ✓ | not yet filed |
| 62 | Mutation check of the fixtures (71 mutants, 2 expected survivors) | ✓ | not yet filed |
| 63 | Navy Wars replay as an integration test | ☐ | not yet filed |
| 64 | Fixtures run by a second, independent runtime | ☐ | not yet filed |
| 65 | A fixture that pins `steps` exactly (needs the core to fix a total) | ☐ | not yet filed |

**Wave 6: 4/7, in progress.**

### Wave 7 — Public API, Session, build package (5 items, ideal 0.0d, added 2026-10-14)

| # | Summary | Status | Register item |
|---|---|---|---|
| 66 | 7.1 Core API: `shaxon.Validate`, `jaxson.Number`, `RunJSON` | ✓ | not yet filed |
| 67 | 7.2 Session helper (`pkg/jaxtools`) and `jaxplay` | ✓ | not yet filed |
| 68 | 7.3 Builder package (`pkg/jaxson/build`) and the Navy Wars port | ✓ | not yet filed |
| 69 | Confirm the A1 `Validate` signature | ✓ | not yet filed |
| 70 | Confirm the N1 and N2 `Number` scope | ✓ | not yet filed |

**Wave 7: 5/5, done.**

---

