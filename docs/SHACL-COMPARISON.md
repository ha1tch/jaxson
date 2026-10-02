# SHACL-Shaxon Comparison 
We treat **SHACL 1.1** as the 2017 Recommendation and **SHACL 1.2** as the current W3C 1.2 Core Working Draft (18 September 2026), rather than treating 1.2 as a finalized standard. [W3C](https://www.w3.org/TR/shacl/)

For “primitive function” we use it in the **semantic-operation** sense: what the validator actually does, rather than merely matching vocabulary names. That makes the comparison with Shaxon much more meaningful.

| Primitive function | How SHACL 1.1 does it | How SHACL 1.2 does it | How Shaxon v0.3.1 does it |
|---|---|---|---|
| **Primitive type / kind** | `sh:nodeKind` tests RDF node kind: IRI, blank node, literal, or combinations. | Same basic operation; 1.2 additionally allows lists of node kinds as a union. [W3C](https://www.w3.org/TR/shacl12-core/) | `kind`: `string`, `number`, `boolean`, `null`, `object`, `array`, or `node`; directly matches JSON types rather than RDF node categories. |
| **Datatype** | `sh:datatype` tests the RDF datatype of each value. | Same, with 1.2 allowing a list of datatypes representing alternatives. [W3C](https://www.w3.org/TR/shacl12-core/) | Folded into JSON `kind`; no RDF datatype layer. |
| **Class / shape membership** | `sh:class` requires RDF instances of a class; `sh:node` validates a value against another node shape. | Same, with expanded shape/node-expression machinery. [W3C](https://www.w3.org/TR/2026/WD-shacl12-core-20260917/) | Nested `fields`, named shapes, `reference`, and combinators provide the corresponding JSON-tree mechanisms. |
| **Required member / minimum cardinality** | `sh:minCount` requires at least N values reached through a property path. | Same. | `required` directly requires named JSON object members; absence produces one violation **per missing member**. |
| **Maximum cardinality** | `sh:maxCount` limits the number of values reached through a property path. | Same. | No general RDF-style `maxCount`; JSON object fields naturally have at most one value. Array cardinality is represented through the array-oriented shape machinery. |
| **String length** | `sh:minLength` / `sh:maxLength`. | Same. | `minLen` / `maxLen` on fields. |
| **Pattern matching** | `sh:pattern` plus optional `sh:flags`, using SPARQL regex semantics. | Same basic operation. | Regex constraint on a JSON string field, under Jaxson's deterministic execution model. |
| **Enumeration** | `sh:in` checks membership in an RDF list. | Same, with additional list-related Core machinery. [W3C](https://www.w3.org/TR/shacl12-core/) | `enum` is directly attached to scalar kinds and checks exact membership; deliberately restricted to `string`, `number`, `boolean`, and `null`. |
| **Exact required value** | `sh:hasValue` requires a particular RDF value. | Same. [W3C](https://www.w3.org/TR/shacl12-core/) | No separate `hasValue` primitive; `enum` on a scalar is the corresponding restricted JSON operation. |
| **Language constraint** | `sh:languageIn` restricts language tags; `sh:uniqueLang` requires unique language-tagged values. | Same, within RDF's language/datatype model. | No direct analogue because JSON strings do not carry RDF language tags. |
| **Numeric range** | `sh:minInclusive`, `sh:minExclusive`, `sh:maxInclusive`, `sh:maxExclusive`. | Same. | Numeric range constraints are expressed over JSON `number` values using Jaxson computations/field constraints rather than RDF comparison semantics. |
| **Equality of two paths** | `sh:equals` compares values reached through two RDF property paths. | Same, and 1.2 broadens the four property-pair constraints to support property paths. [W3C](https://www.w3.org/TR/shacl12-core/) | No direct `equals` constraint primitive; path/compute machinery can express the comparison while retaining deterministic execution. |
| **Disjointness of two paths** | `sh:disjoint`. | Same, with property-path expansion in 1.2. [W3C](https://www.w3.org/TR/shacl12-core/) | No direct primitive; relational/path/index mechanisms provide the corresponding reachability, with `$compute` available for custom predicates. |
| **Subset relationship** | No dedicated Core subset primitive in 1.1. | 1.2 adds the relevant list/property-path constraint machinery. | No dedicated primitive; can be expressed through indexed/path or `$compute` logic. |
| **Less-than comparison** | `sh:lessThan`. | `sh:lessThan` now also accepts property paths. [W3C](https://www.w3.org/TR/shacl12-core/) | No separate RDF-style comparison component; JSON values can be compared through Jaxson computation. |
| **Less-than-or-equal** | `sh:lessThanOrEquals`. | Same, with property-path support. [W3C](https://www.w3.org/TR/shacl12-core/) | Computation rather than a dedicated comparison vocabulary. |
| **Nested shape validation** | `sh:node` validates each value against another node shape. | Same. | Named/inline shapes and nested `fields`; bounded by `maxShapeDepth`. |
| **Property-shape composition** | `sh:property` attaches property shapes to a node shape. | Same. | `fields` is the native JSON equivalent: each named member carries its own constraints/shape. |
| **AND** | `sh:and` requires conformance to every shape in an RDF list. | Same. | `and: [...]`; evaluation is **ordered and mandatorily short-circuiting**. |
| **OR** | `sh:or` requires conformance to at least one shape. | Same. | `or: [...]`; stops at first successful alternative. |
| **Exactly-one** | `sh:xone` requires exactly one alternative to conform. | Same. | `xone: [...]`; stops once a second successful alternative is found. |
| **NOT** | `sh:not` requires non-conformance to a shape. | Same. | `not: shape`. |
| **Qualified count** | `sh:qualifiedValueShape` + `sh:qualifiedMinCount` / `sh:qualifiedMaxCount`, optionally `sh:qualifiedValueShapesDisjoint`. | Same, plus `sh:someValue` as a direct existential form. [W3C](https://www.w3.org/TR/shacl12-core/) | `qualified` provides the analogous qualified population constraint over JSON values. |
| **Existential “some value conforms”** | Not a named Core primitive in 1.1; typically expressed with qualified shape + minimum count. | `sh:someValue` explicitly provides it. [W3C](https://www.w3.org/TR/shacl12-core/) | Expressible through shape combinators/qualified checking rather than copying the RDF-specific primitive. |
| **Closed object/schema** | `sh:closed true` says only properties declared by the shape are permitted; `sh:ignoredProperties` exempts selected properties. | Same, plus `sh:ByTypes` as an additional closed-shape mode. [W3C](https://www.w3.org/TR/shacl12-core/) | `closed` + `ignoredProperties`; naturally applies to JSON object members rather than RDF predicates. |
| **Recursive shapes** | Recursive shapes are syntactically possible, but recursion semantics are deliberately left to processor implementations. [W3C](https://www.w3.org/TR/shacl12-core/) | Same general situation. | Explicit `maxShapeDepth`; exceeding it is a defined `SHAPE_DEPTH_EXCEEDED` execution error/violation. |
| **Property paths** | RDF property paths: predicate, inverse, sequence, alternative, zero-or-more, one-or-more, zero-or-one. | Same core path family, with the 1.2 path/node-expression extensions. [W3C](https://www.w3.org/TR/shacl12-core/) | JSON-native paths plus `$altPath`, `$path*`, `$path+`, and `$inverse`; closures require explicit `maxDepth` and produce ordered, duplicate-free paths. |
| **Inverse navigation** | `sh:inversePath` traverses an RDF edge backwards. | Same. | `$inverse` uses a **declared index/relation**, not a graph traversal/search. |
| **Relational join** | Emerges from RDF graph edges and SPARQL/property paths. | Same basic graph model. | Explicit `indices` and `relations`; relational reach is materialized as named maps rather than inferred from a graph. |
| **Indexing** | No SHACL Core primitive equivalent to a JSON index. RDF graph lookup is part of the underlying graph model. | Same. | First-class named `indices`; build once and reuse is mandatory. |
| **Reference integrity** | RDF IRIs naturally refer to graph nodes; SHACL can constrain whether referenced nodes conform, but has no JSON-style dangling-ID primitive. | Same. | `kind: "reference"` resolves a scalar against a named index/relation; missing target is explicitly `DANGLING_REFERENCE`. |
| **Target selection** | `sh:targetNode`, `sh:targetClass`, `sh:targetSubjectsOf`, `sh:targetObjectsOf`; 1.1 targets are RDF-graph derived. | Adds `targetWhere`, generalized node expressions, and other target machinery. [W3C](https://www.w3.org/TR/shacl12-core/) | Explicit `validate` targets: `$path`, `$each`, `$discriminator`, `$indexed`, `$path*`, `$path+`. |
| **Target every array/object member** | No JSON equivalent; RDF targets operate on graph nodes. | Node expressions make target selection more expressive, but still RDF-oriented. [W3C](https://www.w3.org/TR/shacl12-core/) | `$each` explicitly turns every member into an independent focus node. |
| **Target by discriminator** | No direct primitive. | `targetWhere` can express this indirectly by selecting nodes satisfying a shape. [W3C](https://www.w3.org/TR/shacl12-core/) | `$discriminator` directly selects elements whose named field equals a specified value. |
| **Target from an index** | No direct equivalent. | No direct equivalent. | `$indexed` turns the source elements underlying a named index into focus nodes. |
| **Uniqueness** | `sh:uniqueLang` is specifically about unique language tags; there is no general arbitrary-field uniqueness constraint. | 1.2 adds additional list/property facilities but does not turn SHACL Core into a general JSON-field uniqueness primitive. | `unique` is a first-class validation operation over a target population; repeated values produce individual violations. |
| **Custom computation** | SHACL-SPARQL provides arbitrary SPARQL-based constraints; JavaScript is provided by SHACL-JS rather than Core. | SPARQL-related features are moved out of Core into the SHACL 1.2 SPARQL specification. [W3C](https://www.w3.org/TR/shacl12-sparql/) | `$compute` is the controlled extension mechanism: deterministic, closed, step-bounded Jaxson computation. |
| **Inline executable check** | No equivalent Core instruction. | No equivalent Core instruction. | `check` is an actual Jaxson instruction: validation can occur **inside** a running program. |
| **Schema inheritance / extension** | No `extends` mechanism in Core. | No general Core `extends` mechanism. | `extends` merges shapes with explicit aggregation/override rules and bounded recursion. |
| **Constraint identity** | Results identify source shape/component, but arbitrary stable constraint IDs are not a central Core mechanism. | Same general report model. | `constraintId` is explicitly part of Shaxon reporting; `requiredIds`, `check`, and `unique` can assign stable IDs. |
| **Validation report** | RDF `sh:ValidationReport` containing `sh:ValidationResult` resources, with focus node, result path/value, source shape/component, severity, message, etc. [W3C](https://www.w3.org/TR/shacl/) | Same basic report model; 1.2 retains mandatory result properties and extends reporting-related capabilities elsewhere. [W3C](https://www.w3.org/TR/shacl12-core/) | JSON-native report with deterministic violation ordering/granularity, `focusPath`, `constraintPath`, `constraintId`, `kind`, severity, message, and `conforms`. |
| **Gate vs collect** | Validation produces a report; processor behavior is not a Jaxson-style pipeline gate with an execution continuation. | Same basic model. | Explicit `gate` vs `report`: gate aborts at the first violation; report collects all applicable violations and can write them to `into`. |
| **Deterministic execution bound** | SHACL validation semantics do not impose a universal execution-step budget. Recursive-shape behavior is implementation-dependent. [W3C](https://www.w3.org/TR/shacl12-core/) | Still not the central semantic invariant of SHACL 1.2. | Fundamental language invariant: every recursive shape/path operation has an explicit bound; Jaxson step accounting supplies the execution bound. |
| **Data mutation during validation** | SHACL is fundamentally a validation operation over RDF graphs; Core does not make validation a mutation phase of an executable program. | Same. | Validation is explicitly pure; `check` observes the current Jaxson state and can be called during execution. |
| **Validation + computation** | Separate conceptual systems: SHACL validates RDF; application logic acts on the result. | Same overall separation. | Deliberately unified: **Jaxson computes, Shaxon validates**, within one executable package and determinism model. |
| **Data model** | RDF graph: subject–predicate–object triples. | RDF 1.2 graph model. | JSON tree: `input`, `state`, `output`, `local`; paths and arrays/objects are native. |
| **Custom graph relations** | Native RDF edges; arbitrary graph structure does not require an auxiliary index declaration. | Same. | A JSON tree has no implicit graph edges, so relational edges must be explicitly declared through `indices`/`relations`. |
| **Reification** | RDF 1.1 reification mechanisms can be constrained through SHACL's RDF model, but no dedicated Core `reificationRequired` primitive. | Adds `sh:reifierShape` and `sh:reificationRequired`. [W3C](https://www.w3.org/TR/2026/WD-shacl12-core-20260917/) | No RDF reification analogue; JSON objects are already first-class data nodes. |

### The big structural difference

The table makes something important visible: **Shaxon isn't really trying to reproduce SHACL primitive-for-primitive.**

SHACL's primitive unit is roughly:

> **RDF focus node → RDF property path → set of RDF value nodes → constraint component**

Shaxon's is closer to:

> **JSON focus path → JSON value(s) → shape/field constraint → deterministic execution/report**

That difference explains several of the seemingly "missing" SHACL primitives. For example, `maxCount` is fundamental in RDF because one RDF subject can have arbitrarily many values for the same predicate. A JSON object member is intrinsically singular, so Shaxon doesn't need to reproduce that primitive for ordinary fields.

Conversely, Shaxon has primitives that **SHACL doesn't have at all**: `$indexed`, `$discriminator`, `$inverse` over declared indices, `reference`, `unique`, `check`, bounded JSON closures, and executable gate/report semantics.

And SHACL 1.2 is itself still a **Working Draft**, not a successor Recommendation to the 2017 SHACL 1.1 Recommendation. [W3C](https://www.w3.org/TR/shacl12-core/)


