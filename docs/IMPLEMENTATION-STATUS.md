# Jaxson and Shaxon — Implementation Status Matrix

**Snapshot:** current `jaxson-shaxon-v0.3.1` package
**Status categories:**

* **RUNNABLE** — implemented in the current Go package and exercised by tests/examples.
* **PARTIALLY IMPLEMENTED** — real implementation exists, but the complete feature/pipeline is not yet available end-to-end.
* **SPECIFIED ONLY** — defined by the Shaxon/Jaxson design documents but not yet implemented in the current runtime.

The distinction is deliberately conservative: the existence of parsing or static-validation code does not count as a runnable language feature unless a user can actually execute the corresponding behavior through the runtime.

---

## 1. Jaxson

| Feature                               | Status       | Current evidence / qualification                                                                                                 |
| ------------------------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| JSON package loading / normalization  | **RUNNABLE** | Implemented in `pkg/jaxson` and exercised by fixtures and tooling.                                                               |
| Core program execution                | **RUNNABLE** | `jaxson.Run` is implemented and tested.                                                                                          |
| Paths                                 | **RUNNABLE** | Explicit path traversal and addressing are implemented.                                                                          |
| Reading/writing state                 | **RUNNABLE** | Core mutation instructions are implemented and tested.                                                                           |
| `set`                                 | **RUNNABLE** | Tested, including mutation hooks.                                                                                                |
| `append`                              | **RUNNABLE** | Tested.                                                                                                                          |
| `insert`                              | **RUNNABLE** | Tested.                                                                                                                          |
| `delete`                              | **RUNNABLE** | Tested, including array-shift semantics and mutation reporting.                                                                  |
| `$compute`                            | **RUNNABLE** | Full compute machinery is implemented.                                                                                           |
| Core compute operators                | **RUNNABLE** | Showcase tests cover all 30 current compute operators.                                                                           |
| Lazy `and` / `or` / `select` behavior | **RUNNABLE** | Explicitly covered by fixtures.                                                                                                  |
| Exact decimal arithmetic              | **RUNNABLE** | Exact arithmetic is covered by fixtures.                                                                                         |
| No implicit numeric coercion          | **RUNNABLE** | Tested.                                                                                                                          |
| Numeric overflow behavior             | **RUNNABLE** | Tested as an error rather than silent rounding.                                                                                  |
| `null` vs missing                     | **RUNNABLE** | Explicitly tested.                                                                                                               |
| Templates / optional values           | **RUNNABLE** | Current fixture suite covers template behavior.                                                                                  |
| Conditionals / control flow           | **RUNNABLE** | Current instruction/expression machinery supports the implemented control constructs.                                            |
| Loops                                 | **RUNNABLE** | Snapshot iteration, nested loops, indices, binding rules, and step limits are tested.                                            |
| Dynamic path segments                 | **RUNNABLE** | Tested.                                                                                                                          |
| `halt`                                | **RUNNABLE** | Tested.                                                                                                                          |
| Static program checking               | **RUNNABLE** | `Checker` and program/schema checking are implemented.                                                                           |
| Output contracts / schema validation  | **RUNNABLE** | Implemented and covered by fixtures.                                                                                             |
| Closed objects                        | **RUNNABLE** | Current Jaxson schema behavior is tested.                                                                                        |
| Execution limits / step accounting    | **RUNNABLE** | Declared step limits and host-instruction step sharing are tested.                                                               |
| Deterministic ordering                | **RUNNABLE** | Ordering utilities and deterministic traversal are implemented.                                                                  |
| Explicit mutation notification        | **RUNNABLE** | `Machine.OnMutate` is implemented and tested.                                                                                    |
| Host instruction extension mechanism  | **RUNNABLE** | Instruction tables, host checking/execution, and extension tests exist.                                                          |
| Host/operator extension architecture  | **RUNNABLE** | Core instruction/operator tables and host hooks are implemented.                                                                 |
| Exported Jaxson primitives for hosts  | **RUNNABLE** | `Walk`, `GetAt`, `Eval`, `RunCompute`, `Clone`, `Equal`, `TypeName`, `Order`, `SortedKeys`, `FormatDecimal`, etc. are available. |
| `jaxtools.Session`                    | **RUNNABLE** | Implemented and tested.                                                                                                          |
| `jaxplay`                             | **RUNNABLE** | Ported to the package API and verified.                                                                                          |
| `jaxrun`                              | **RUNNABLE** | Thin CLI over the Jaxson runtime.                                                                                                |
| Showcase examples                     | **RUNNABLE** | 12 example packages / 29 fixture cases; current repository tests them.                                                           |
| Navy Wars example                     | **RUNNABLE** | Ported and tested.                                                                                                               |

### Jaxson overall status

**RUNNABLE.**

Jaxson is no longer merely a design with a reference interpreter. The current package contains a tested runtime, extension surface, tooling, examples, fixtures, and application-level usage.

---

# 2. Shaxon

The Shaxon status requires more care because the latest package has now crossed into actual implementation, but only the **static/declarative layer** is currently present.

## 2.1 Static Shaxon layer

| Feature                          | Status                    | Current evidence / qualification                                                                                                    |
| -------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `pkg/shaxon` package             | **PARTIALLY IMPLEMENTED** | Real package now exists.                                                                                                            |
| Shaxon error categories          | **PARTIALLY IMPLEMENTED** | `SHA_*` errors are defined for implemented static checks.                                                                           |
| Registry model                   | **PARTIALLY IMPLEMENTED** | `Registries` exists for shapes, indices, relations, and computes.                                                                   |
| Shape declarations               | **PARTIALLY IMPLEMENTED** | Parsing and static checking exist.                                                                                                  |
| Index declarations               | **PARTIALLY IMPLEMENTED** | Parsing/static validation exists; no runtime index execution yet.                                                                   |
| Relation declarations            | **PARTIALLY IMPLEMENTED** | Parsing/static validation exists; data-dependent relation validation is later.                                                      |
| Compute declarations             | **PARTIALLY IMPLEMENTED** | Parsing and delegation to Jaxson's compute checker exist.                                                                           |
| `extends` merging                | **PARTIALLY IMPLEMENTED** | Default union/AND/concatenation semantics are implemented and tested.                                                               |
| `extends` cycle detection        | **PARTIALLY IMPLEMENTED** | Static cycle detection is implemented.                                                                                              |
| Recursive shape analysis         | **PARTIALLY IMPLEMENTED** | Recursive references and static depth-related checks exist.                                                                         |
| `maxShapeDepth` analysis         | **PARTIALLY IMPLEMENTED** | Static infrastructure exists; actual recursive evaluation is not yet implemented.                                                   |
| Primitive shape keywords         | **PARTIALLY IMPLEMENTED** | Keywords are parsed/type-checked/carried through, but complete data-dependent enforcement belongs to shape evaluation.              |
| Keyword cross-consistency checks | **SPECIFIED ONLY / GAP**  | For example, relationships such as `minLen <= maxLen` are not yet enforced.                                                         |
| `extends` override syntax        | **SPECIFIED ONLY / GAP**  | Semantics are described, but the concrete JSON syntax is insufficiently specified; implementation deliberately does not invent one. |

---

## 2.2 Shaxon validation/evaluation

| Feature                                    | Status             | Current evidence / qualification                                                                        |
| ------------------------------------------ | ------------------ | ------------------------------------------------------------------------------------------------------- |
| Shape evaluation against actual input data | **SPECIFIED ONLY** | The current implementation has not yet reached the shape-evaluation phase.                              |
| Field constraint evaluation                | **SPECIFIED ONLY** | Declared in the Shaxon design; no complete evaluator yet.                                               |
| Nested shape evaluation                    | **SPECIFIED ONLY** | Depends on the evaluator.                                                                               |
| Qualified constraints                      | **SPECIFIED ONLY** | Defined by the language design; full evaluation remains ahead.                                          |
| Recursive shape evaluation                 | **SPECIFIED ONLY** | Static recursion analysis exists, but runtime recursive validation does not.                            |
| Shape-depth enforcement during validation  | **SPECIFIED ONLY** | The design specifies the behavior; evaluator implementation remains.                                    |
| Data-dependent index construction/use      | **SPECIFIED ONLY** | Static index declarations exist; actual graph/data indexing belongs to later phases.                    |
| Index uniqueness checking                  | **SPECIFIED ONLY** | Requires actual input data.                                                                             |
| Relation validation                        | **SPECIFIED ONLY** | Relation declarations are parsed; runtime relation checking is not yet implemented.                     |
| Cardinality validation                     | **SPECIFIED ONLY** | Especially `one-to-one` and other data-dependent constraints.                                           |
| `unique` validation                        | **SPECIFIED ONLY** | The registry knows the declaration, but its data-dependent checks belong to the later validation phase. |
| Target resolution                          | **SPECIFIED ONLY** | Target semantics are defined but the target evaluator is not yet implemented.                           |
| Explicit graph reach                       | **SPECIFIED ONLY** | Architectural rule is defined; runtime traversal machinery remains.                                     |
| `check` instruction                        | **SPECIFIED ONLY** | Planned Shaxon instruction; not yet part of a complete runtime pipeline.                                |
| `validate` instruction                     | **SPECIFIED ONLY** | Same distinction: specified, but not yet executable end-to-end.                                         |
| Gate validation                            | **SPECIFIED ONLY** | Semantics defined; evaluator/report pipeline remains.                                                   |
| Diagnostic/report validation               | **SPECIFIED ONLY** | Reporting semantics are defined but current package has no complete Shaxon reporting runtime.           |
| Shaxon validation pipeline                 | **SPECIFIED ONLY** | The planned `shaxon.Run` pipeline has not yet been implemented.                                         |
| `shaxon.Validate` public API               | **SPECIFIED ONLY** | Explicitly identified as a later phase.                                                                 |
| Shaxon conformance fixtures                | **SPECIFIED ONLY** | The current Shaxon fixture/conformance phase has not yet been completed.                                |
| `cmd/shaxrun`                              | **SPECIFIED ONLY** | Not present in the current package.                                                                     |

---

# 3. Jaxson/Shaxon integration

| Capability                                   | Status                         | Meaning                                                                                              |
| -------------------------------------------- | ------------------------------ | ---------------------------------------------------------------------------------------------------- |
| Jaxson as the execution substrate for Shaxon | **PARTIALLY IMPLEMENTED**      | The Jaxson extension points needed by Shaxon are implemented and tested.                             |
| Shaxon calling Jaxson operand/path checking  | **PARTIALLY IMPLEMENTED**      | Static Shaxon compute parsing already delegates appropriately to Jaxson.                             |
| Shared mutation hooks                        | **RUNNABLE infrastructure**    | Jaxson exposes the mutation hook Shaxon will need.                                                   |
| Shared execution accounting                  | **RUNNABLE infrastructure**    | Jaxson host instructions share the machine's step limit.                                             |
| Shaxon static registries                     | **PARTIALLY IMPLEMENTED**      | Implemented independently of input data.                                                             |
| Complete Jaxson → Shaxon execution pipeline  | **SPECIFIED ONLY**             | The complete integrated runtime is still a future phase.                                             |
| Shaxon validation of data produced by Jaxson | **SPECIFIED ONLY**             | Architecturally intended, but not yet an end-to-end executable capability.                           |
| Mutation/delta integration                   | **SPECIFIED / RESEARCH LAYER** | Delta/graph-transformation material exists separately and is not part of the current Shaxon runtime. |

---

# 4. The simplest way to present the status to a newcomer

The entire project can currently be summarized as:

| Layer                                        | Current status                          |
| -------------------------------------------- | --------------------------------------- |
| **Jaxson language**                          | 🟢 **Runnable**                         |
| **Jaxson runtime**                           | 🟢 **Runnable**                         |
| **Jaxson extension mechanism**               | 🟢 **Runnable**                         |
| **Jaxson tooling/examples**                  | 🟢 **Runnable**                         |
| **Shaxon declarations / registries**         | 🟡 **Partially implemented**            |
| **Shaxon static analysis**                   | 🟡 **Partially implemented**            |
| **Shaxon shape evaluator**                   | 🔵 **Specified, not yet implemented**   |
| **Shaxon graph/index/relation evaluator**    | 🔵 **Specified, not yet implemented**   |
| **Shaxon targets / validation instructions** | 🔵 **Specified, not yet implemented**   |
| **Shaxon reporting**                         | 🔵 **Specified, not yet implemented**   |
| **Complete Shaxon runtime**                  | 🔵 **Not yet implemented**              |
| **Jaxson + Shaxon end-to-end**               | 🔵 **Not yet implemented**              |
| **Delta / graph-transformation layer**       | ⚪ **Separate research/design material** |

---

> **Jaxson is currently a runnable, tested language and runtime. Shaxon is in active implementation: its static declaration and registry layer is implemented and tested, while its data-dependent validation engine, graph traversal, relation/index evaluation, targets, reporting, and end-to-end execution pipeline remain specified but not yet implemented.**


