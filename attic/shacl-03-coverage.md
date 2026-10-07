# SHACL Coverage and Capability Comparison

## Scope and status

This document evaluates Jaxson + Shaxon as a real-world capability set relative to SHACL. The comparison is weighted toward practical validation workloads rather than treating every feature in the SHACL specification as equally significant.

One important date qualification applies: **SHACL 1.2 remains a set of W3C Working Drafts as of October 2026**. The latest SHACL Recommendation remains the 2017 SHACL 1.0 Recommendation. Accordingly, established SHACL practice and the newer SHACL 1.2 draft family are distinguished where relevant. ([W3C][1])

## Summary

The current estimate is:

> **Jaxson + Shaxon v0.3.1 specifies roughly 75% of the practically important SHACL functionality encountered in real-world validation, weighted by usage rather than by counting specification features.**

A reasonable uncertainty band is **±10 percentage points**.

The estimate does **not** mean that Shaxon implements or specifies 75% of all SHACL Core features. A mechanical count of every SHACL constraint component, path construct, target mechanism, report property, SPARQL facility, UI facility, inference facility, and related feature produces a substantially lower percentage.

Conversely, if the question is:

> How much of what people actually use SHACL to accomplish in ordinary RDF data-quality work could Shaxon v0.3.1 express once its evaluator is completed?

then **approximately 75% is the current estimate**.

The missing approximately 25% is not distributed uniformly. A substantial portion consists of specialized RDF and Semantic Web capabilities, while Shaxon deliberately provides capabilities addressing execution, mutation, boundedness, indexing, relational structure, and deterministic computation that SHACL does not itself define.

---

# 1. Practical SHACL usage

The strongest recent empirical source identified for practical usage is the 2024–2025 RDF Validation Community Survey, with **94 respondents** from academia and industry. The survey reports that SHACL is by far the most familiar shape language among respondents, with government data being a particularly prominent application domain and operational/industrial use often occurring daily or weekly. ([Sage Journals][2])

The broader SHACL ecosystem includes applications in:

* government and open-data validation,
* enterprise knowledge graphs,
* supply-chain data,
* geospatial data,
* cultural and heritage data,
* vocabulary and ontology quality,
* data integration,
* knowledge-graph pipelines,
* application and form generation.

The European Commission operates an SHACL validation service for RDF content and shapes. GS1's EPCIS project contains a substantial production-oriented SHACL file with constraints covering document structure, cardinality, datatypes, node kinds, patterns, forbidden properties, nested shapes, and related requirements. ([ITB][3])

The current SHACL specifications explicitly identify validation, data integration, UI generation, and code generation as uses of shapes. ([W3C][1])

## Practical validation workload

The center of gravity of ordinary validation work can be represented as:

```text
Identify things to validate
        ↓
Describe their structure
        ↓
Require/forbid properties
        ↓
Check cardinalities
        ↓
Check value types
        ↓
Check nested structures
        ↓
Check relationships
        ↓
Apply logical/business constraints
        ↓
Produce useful violations
```

This workload corresponds closely to the capabilities present in the Shaxon design.

---

# 2. SHACL functionality covered by Jaxson + Shaxon

| Real-world SHACL job        | Jaxson + Shaxon v0.3.1                          | Assessment                                    |
| --------------------------- | ----------------------------------------------- | --------------------------------------------- |
| Define shapes               | `shapes`                                        | **Covered**                                   |
| Validate object structure   | `kind: object`, `fields`                        | **Covered**                                   |
| Required properties         | `required`                                      | **Covered**                                   |
| Maximum/minimum presence    | required + structural/cardinality mechanisms    | **Mostly covered**                            |
| Datatypes                   | `kind`                                          | **Covered**                                   |
| Enumerations                | `enum`                                          | **Covered**                                   |
| Closed objects              | `closed`, `ignoredProperties`                   | **Covered**                                   |
| Nested shapes               | named/inline shapes                             | **Covered**                                   |
| Recursive shapes            | shape references + `maxShapeDepth`              | **Covered, with stronger bounding semantics** |
| Logical AND                 | `and`                                           | **Covered**                                   |
| Logical OR                  | `or`                                            | **Covered**                                   |
| XOR                         | `xone`                                          | **Covered**                                   |
| NOT                         | `not`                                           | **Covered**                                   |
| Qualified constraints       | `qualified`                                     | **Covered**                                   |
| Custom value/business rules | `$compute`/`check`                              | **Covered in a different model**              |
| Targets                     | `validate` targets                              | **Covered**                                   |
| Target subsets              | `$each`, `$discriminator`, `$indexed`, closures | **Covered, differently**                      |
| References                  | `reference` + indices                           | **Covered**                                   |
| Inverse relationships       | `$inverse` / relations                          | **Covered**                                   |
| Relational cardinality      | `relations`                                     | **Covered**                                   |
| Uniqueness                  | `unique`, index uniqueness                      | **Covered**                                   |
| Validation reports          | explicit Shaxon report                          | **Covered**                                   |
| Severity                    | violation/warning/info                          | **Covered, but `conforms` differs**: only `violation` makes it false, where in SHACL 1.1 any result does (limitations section 8) |
| Stable constraint IDs       | `constraintId`                                  | **Covered**                                   |
| Gate/conformance validation | `mode: gate`                                    | **Covered**                                   |
| Non-fatal reporting         | `mode: report`                                  | **Covered**                                   |
| Path traversal              | `$path*`, `$path+`, `$altPath`, etc.            | **Covered, with explicit bounds**             |
| Reusable computations       | `computes`                                      | **Covered**                                   |

These capabilities account for the relatively high practical-coverage estimate. The v0.3.1 design is aimed at the **shape-validation workload**, rather than being limited to JSON Schema-style typing.

Several of these features correspond directly to constructs found in real SHACL files. GS1's EPCIS shapes, for example, use `sh:minCount`, `sh:maxCount`, `sh:datatype`, `sh:nodeKind`, `sh:pattern`, nested shapes, and forbidden properties. These represent the same general structural-validation workload addressed by Shaxon. ([GitHub][4])

---

# 3. Basis of the practical-coverage estimate

The following percentages are reasoned estimates rather than measurements.

## 3.1 Ordinary structural validation

Estimated coverage: **~80–90%**.

Shaxon provides:

* shape definitions,
* fields,
* required members,
* types,
* closed structures,
* nested shapes,
* logical composition,
* qualified constraints,
* recursive structures,
* enumerations,
* severity,
* messages,
* reports.

These capabilities cover a substantial portion of the ordinary structural validation workload associated with SHACL.

---

## 3.2 Relationship validation

Estimated coverage: **~75–85%** of common relationship-validation workloads.

The v0.3.1 design provides:

```text
index
   ↓
reference
   ↓
relation
   ↓
inverse
   ↓
target
```

together with explicit cardinality.

SHACL obtains much of its relational power from RDF's native graph structure, property paths, and constraints. Because JSON does not intrinsically provide RDF-style graph edges, Shaxon makes referential structure explicit through its `indices` and `relations` system.

This is not merely an implementation workaround. It constitutes an explicit representation of referential structure in the language.

---

## 3.3 Business rules

Estimated coverage: **~70–80%** of ordinary business-rule workloads.

SHACL use frequently extends beyond datatype and cardinality constraints. The 2025 survey reports that users turn to SHACL-SPARQL when additional expressiveness is required. ([Sage Journals][2])

Shaxon provides a different mechanism:

```text
shape
  ↓
check
  ↓
$compute
```

rather than:

```text
shape
  ↓
SPARQL query
```

This permits constraints such as:

```text
total >= 0
age >= 18
x < y
a == b
conditional rule
```

to be represented naturally.

The difference becomes significant for constraints requiring the general expressive power of SPARQL, because the current Jaxson computation language is deliberately closed.

---

# 4. Major SHACL capabilities not currently matched by Shaxon

The largest remaining gap is not basic validation. It is **open-ended RDF/SPARQL semantics**.

## 4.1 SPARQL-based constraints

SHACL-SPARQL permits arbitrary SPARQL-based constraints and user-defined constraint components. SHACL 1.2 continues this model. ([W3C][5])

Conceptually, a constraint can be expressed as:

> Execute an arbitrary graph query; if it returns a row, report a violation.

This provides a substantially open-ended source of expressive power.

Shaxon deliberately takes a different approach:

> Use the closed `$compute` language.

Consequently, there are constraints expressible in SHACL-SPARQL that cannot currently be expressed in Shaxon v0.3.1.

Therefore, Shaxon should not be characterized as having all of the expressive power of SHACL-SPARQL.

The difference is architectural rather than merely syntactic.

---

## 4.2 Arbitrary RDF property-path semantics

SHACL 1.2 Core provides path constructs including:

* sequence paths,
* alternative paths,
* inverse paths,
* zero-or-more,
* one-or-more,
* zero-or-one,
* increasingly rich expression mechanisms. ([W3C][1])

Shaxon provides corresponding machinery including:

```text
$altPath
$path*
$path+
$inverse
```

The principal difference is that Shaxon requires explicit traversal bounds.

SHACL does not impose Shaxon's universal `maxDepth` / `maxShapeDepth` discipline.

Consequently, Shaxon covers a substantial portion of useful path-validation workloads while deliberately avoiding unrestricted path semantics.

---

## 4.3 RDF class semantics

SHACL can require that a value be an instance of a particular RDF class. This interacts with RDF/RDFS typing, subclass relationships, and entailment regimes.

SHACL 1.2 explicitly defines SHACL types and subclass relationships and permits entailment regimes. ([W3C][1])

Shaxon instead provides mechanisms such as:

```text
JSON kind
shape
reference
index
relation
```

These mechanisms serve analogous validation purposes in a different data ontology.

This is therefore not simply a missing feature. It reflects a different underlying data model.

---

## 4.4 Language-tagged RDF literals

SHACL includes constraints such as:

* `languageIn`,
* `uniqueLang`.

These are meaningful because RDF literals can carry language tags. ([W3C][1])

Plain JSON has no equivalent primitive.

This therefore represents a genuine domain that Shaxon does not natively cover.

---

## 4.5 RDF lists and RDF reification

SHACL 1.2 includes list constraints and newer reification-related constraints. ([W3C][1])

These address structures that arise specifically from RDF's representation of lists and reified statements.

Shaxon does not reproduce these mechanisms because arrays and ordinary JSON objects represent such structures directly.

These capabilities can therefore be classified as:

**SHACL capabilities absent from Shaxon primarily because the corresponding underlying problem is represented differently or disappears in Shaxon's data model.**

---

# 5. Capabilities specified by Shaxon that SHACL does not specify

The comparison is not one-directional. Shaxon introduces semantics that are outside SHACL's problem definition.

## 5.1 Jaxson as an execution language

Jaxson specifies an execution model involving:

```text
input
state
local
output
   ↓
instructions
   ↓
mutation
   ↓
deterministic state transition
```

The language includes operations such as:

* `set`,
* `delete`,
* `append`,
* `insert`,
* `for`,
* `if`,
* `assert`,
* `halt`.

SHACL explicitly treats the data and shapes graphs as immutable during validation. ([W3C][6])

Consequently, Jaxson's execution model is not something that SHACL itself specifies.

Jaxson can be implemented alongside SHACL, and SHACL results can control another program, but SHACL itself does not define the Jaxson-style execution model.

This constitutes an additional capability of the Jaxson/Shaxon system.

---

# 6. Deterministic execution as a language property

Jaxson specifies properties including:

* deterministic ordering,
* explicit instruction sequencing,
* explicit step counting,
* explicit failure categories,
* bounded execution,
* mandatory short-circuit behavior,
* deterministic index rebuild/reuse.

SHACL's semantic objective is primarily to produce the specified validation results. It does not define a general deterministic execution-cost model analogous to Jaxson's.

Shaxon extends this principle to a semantic requirement that whether a package succeeds within its declared step budget must not depend on which conformant runtime executes it.

This is not a property specified by SHACL.

---

# 7. Explicit computational resource bounds

Shaxon makes resource limits explicit through mechanisms including:

```text
maxShapeDepth
maxDepth
steps
```

Their semantics form part of the language.

SHACL 1.2 addresses recursive shapes and contains extensive treatment of recursion, but does not make Shaxon's universal explicit resource horizon part of the semantic contract of validation. ([W3C][1])

Similarly, Shaxon's path closure can be expressed as:

```text
$path+
maxDepth: 10
```

rather than as an unbounded closure.

This is a **language-design property**, rather than merely an implementation optimization.

---

# 8. Mutation-aware validation

Shaxon specifies a model in which validation can occur within an executing program:

```text
program
   ↓
state mutation
   ↓
check
   ↓
more mutation
   ↓
check
```

The `check` instruction can validate an intermediate state.

The v0.3.1 design specifies `check` as a callable validation operation within `program`.

SHACL does not define this execution model. Its data graph remains immutable during validation. ([W3C][6])

An application can certainly implement repeated validation externally:

```text
application
   ↓
mutate
   ↓
SHACL
   ↓
mutate
   ↓
SHACL
```

However, that orchestration is outside the SHACL language itself.

Shaxon incorporates the validation operation into the execution model.

---

# 9. First-class named indices

Shaxon explicitly specifies:

```text
indices
```

as named, deterministic, rebuildable data structures.

An index has properties including:

* source paths,
* key expressions,
* uniqueness semantics,
* `multi`,
* rebuild rules,
* mutation invalidation rules,
* reuse rules,
* step costs.

SHACL has no corresponding first-class language construct specifying:

> Construct this named index over this population and provide later validation operations with deterministic access to it.

RDF stores naturally maintain indexes internally, and SHACL implementations can use those indexes for performance. That implementation-level use is distinct from exposing the index as a semantic object in the language.

**SHACL does not specify the index as a semantic object available to the language.**

---

# 10. First-class relations

Shaxon provides relations as first-class semantic objects, for example:

```json
"relations": {
  "orderCustomer": {
    "from": ...,
    "to": ...,
    "cardinality": "one-to-many"
  }
}
```

SHACL can describe resulting relational constraints through combinations of:

* property paths,
* cardinalities,
* node constraints,
* SPARQL.

However, SHACL does not define `relation` as a named, bidirectional, indexed semantic object with prescribed construction and reuse semantics.

Shaxon does.

---

# 11. Explicit graph identity by path

Shaxon defines:

> `FocusPath` is the canonical identity of a finding.

This follows directly from its JSON-native design.

A violation can identify a location such as:

```text
["input", "orders", 4]
```

with the offending value obtained from the path.

SHACL instead operates with RDF nodes and validation-result vocabulary, including:

* focus node,
* result path,
* value,
* source shape,
* source constraint component,
* details,
* and related properties. ([W3C][1])

Neither identity model is inherently superior; they correspond to different data models.

Path identity as the fundamental identity mechanism is, however, a specific Shaxon design choice.

---

# 12. Exact decimal computation

Jaxson specifies exact decimal arithmetic with:

* exact numeric representation,
* bounded magnitude,
* explicit rounding,
* `div` and `round` as the digit-losing operations.

SHACL does not specify a JSON-oriented exact-decimal computation language with these semantics.

SPARQL provides arithmetic, and RDF/XSD provides numeric datatypes, so this does not mean that arithmetic cannot be performed in an SHACL environment.

The distinction is that:

> **SHACL does not define Jaxson's exact, closed, deterministic computation semantics.**

---

# 13. Conceptual relationship between the systems

The clearest description of the relationship is:

## SHACL

**A graph constraint language.**

```text
RDF graph
   +
SHACL shapes
   ↓
validation report
```

## Jaxson

**A deterministic JSON program language.**

```text
JSON
 +
program
 ↓
new JSON/state
```

## Shaxon

**A bounded JSON graph-validation language operating on the Jaxson execution model.**

```text
JSON
  +
indices
  +
relations
  +
shapes
  +
targets
  +
bounded paths
  ↓
deterministic validation
```

Jaxson + Shaxon is therefore not simply an attempt to reproduce SHACL for JSON.

Its scope is broader in a different direction:

> **Deterministic data transformation combined with deterministic structural and relational validation within one execution model.**

---

# 14. Coverage estimates by comparison scope

Three different coverage numbers are useful because a single percentage obscures important distinctions.

| Comparison                                                                                        | Estimated coverage |
| ------------------------------------------------------------------------------------------------- | -----------------: |
| **Common real-world SHACL validation workload**                                                   |           **~75%** |
| **SHACL Core 1.2 expressive feature surface**                                                     |        **~55–65%** |
| **Full current SHACL 1.2 family, including SPARQL, node expressions, rules, UI, profiling, etc.** |        **~35–45%** |

The first number represents the practical validation workload rather than the complete specification surface.

The second is lower because SHACL Core contains substantial specialized RDF functionality that Shaxon intentionally does not reproduce. SHACL 1.2 Core currently includes a broader constraint vocabulary than Shaxon's v0.3.1 vocabulary, including value ranges, string constraints, property-pair constraints, list constraints, reification, `hasValue`, `in`, `uniqueValuesFor`, richer targets, and other metadata. ([W3C][1])

The third estimate is substantially lower because the SHACL 1.2 family is considerably broader than Core. It includes SPARQL extensions, node expressions, inference rules, UI, profiling, and other facilities. ([W3C][5])

The **35–45%** figure should not be interpreted as a negative assessment of Shaxon. A substantial part of the excluded functionality lies deliberately outside Shaxon's problem definition.

---

# 15. Distribution of the remaining practical gap

The practical gap between Shaxon and the broader SHACL capability set is not primarily a collection of missing ordinary structural-validation primitives.

The missing functionality clusters around:

```text
RDF semantics
SPARQL
language-tagged literals
RDF lists/reification
ontology/class entailment
advanced RDF-specific paths
SHACL ecosystem/UI/inference features
```

Shaxon instead places significant emphasis on:

```text
JSON structure
explicit references
named indices
relations
bounded traversal
deterministic targets
deterministic evaluation
exact computation
mutation
execution budgets
intermediate-state checking
```

This represents a coherent difference in scope and execution model.

The 2025 RDF validation community survey provides an additional external signal: respondents identified expressiveness and performance as significant challenges, while frequent industrial SHACL users were substantially more likely to use advanced or non-standard features. SPARQL-based constraints were used primarily to obtain additional expressiveness. ([Sage Journals][2])

This places the principal remaining expressive gap in open-ended computation and graph querying rather than in basic structural validation.

---

# 16. Overall characterization

The current comparison supports the following characterization:

> **Jaxson + Shaxon v0.3.1 specifies approximately 75% of the practical SHACL validation workload, with substantially lower coverage of the total SHACL specification surface. At the same time, it specifies additional semantics for execution, mutation, boundedness, indexing, relational structure, and deterministic computation that SHACL does not itself define.**

The practical-coverage estimate should therefore be interpreted together with the scope distinction:

* **~75%** of common real-world validation work,
* **~55–65%** of the SHACL Core 1.2 expressive surface,
* **~35–45%** of the complete current SHACL 1.2 family.

The largest remaining expressive gap is the open-ended computation and graph-query capability provided by SPARQL and related SHACL facilities. Other significant gaps are largely consequences of RDF-specific semantics and representation mechanisms, including RDF classes, language-tagged literals, RDF lists, reification, and entailment.

Conversely, Shaxon introduces a unified execution model in which transformation, mutation, validation, resource bounds, indexing, relational structure, and deterministic computation are explicit language-level concepts.

The comparison therefore describes two overlapping but non-identical language models:

```text
SHACL
RDF graph + shapes
        ↓
graph validation
```

versus:

```text
Jaxson + Shaxon
JSON/state + program + graph structures + shapes
        ↓
bounded deterministic transformation + validation
```

The practical significance of Shaxon's design lies not in reproducing the complete SHACL specification, but in covering a substantial portion of the validation workload through a smaller, bounded, deterministic execution model while providing capabilities outside SHACL's own semantic scope.

---

# References

[1]: https://www.w3.org/TR/2026/WD-shacl12-core-20260917/ "SHACL 1.2 Core"
[2]: https://journals.sagepub.com/doi/full/10.3233/SSW250011 "Bridging Gaps in RDF Validation: Insights and Innovation Opportunities in RDF Validation Practices, 2025"
[3]: https://www.itb.ec.europa.eu/shacl/any/upload "SHACL Validator"
[4]: https://github.com/gs1/EPCIS/blob/master/Ontology/EPCIS-SHACL.ttl "EPCIS/Ontology/EPCIS-SHACL.ttl at master · gs1/EPCIS · GitHub"
[5]: https://www.w3.org/TR/shacl12-sparql/ "SHACL 1.2 SPARQL-Related Features"
[6]: https://www.w3.org/TR/2026/WD-shacl12-core-20260917/ "SHACL 1.2 Core"
