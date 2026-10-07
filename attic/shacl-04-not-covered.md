# SHACL Functionality Not Covered by Shaxon

## Scope

The current comparison estimates that Jaxson + Shaxon v0.3.1 covers approximately 75% of the practically important SHACL validation workload.

The remaining approximately 25% should not be interpreted as a homogeneous set of missing validation capabilities. A substantial portion is attributable to RDF-specific semantics and representation, RDF-oriented graph construction and access mechanisms, or functionality surrounding SHACL itself.

The current estimate is:

> **Approximately 75–85% of the remaining ~25% gap is attributable to RDF-specific semantics/representation, RDF-oriented construction and access machinery, or external/ecosystem functionality.**

This corresponds to approximately:

> **19–21 percentage points of the total SHACL functionality** that Shaxon does not cover primarily because the functionality belongs to the RDF/SPARQL ecosystem or to facilities surrounding SHACL, rather than because Shaxon lacks an equivalent general validation capability.

The remaining **~4–6 percentage points** represent the more substantive residual: constraint and expression capabilities that provide genuinely greater expressive power and are not primarily consequences of RDF representation.

These figures are estimates rather than measurements. An uncertainty of approximately **±5 percentage points** applies to the first four categories collectively.

## Estimated decomposition

| Portion of the ~25% gap                                                    | Approx. share of gap | Approx. points of total functionality |
| -------------------------------------------------------------------------- | -------------------: | ------------------------------------: |
| **RDF-specific data model / semantics**                                    |              ~30–35% |                             ~7.5–9 pp |
| **RDF-specific graph construction / traversal / representation machinery** |              ~15–20% |                               ~4–5 pp |
| **SPARQL / RDF query-expression machinery**                                |              ~15–20% |                               ~4–5 pp |
| **SHACL ecosystem / tooling / UI / profiling / serialization**             |              ~10–15% |                          ~2.5–3.75 pp |
| **Genuinely additional constraint/expression capability**                  |              ~15–20% |                            ~3.75–5 pp |
| **Total**                                                                  |             **100%** |                            **~25 pp** |

The categories are analytical rather than mutually exclusive measurements. In particular, SPARQL combines genuinely additional computational expressiveness with functionality derived specifically from the RDF/SPARQL graph-query model.

---

# 1. RDF-specific functionality

RDF-specific functionality constitutes the largest component of the uncovered surface.

SHACL 1.2 Core is explicitly a language for **RDF graphs**, and the shapes graph is itself an RDF graph. Its semantics are defined in terms of RDF terms, triples, predicates, IRIs, literals, blank nodes, and RDF graph operations. ([W3C][1])

Consequently, some functionality exposed by SHACL is not a general validation capability independent of the underlying data model. It exists because SHACL operates over RDF.

Examples include:

* RDF `IRI`s,
* blank nodes,
* RDF literals,
* datatype IRIs,
* language-tagged literals,
* RDF classes,
* `rdf:type`,
* RDFS class relationships,
* RDF lists,
* RDF reification,
* RDF property paths,
* RDF graph identity,
* entailment regimes.

These concepts have no direct counterpart in Shaxon because JSON does not require the same representation machinery.

For example, SHACL list constraints must operate on RDF's representation of a list as a linked graph structure. A JSON array directly represents the collection as an array.

Similarly, an RDF language-tagged literal is a primitive RDF data-model concept. A normal JSON string does not have an equivalent primitive representation.

Consequently, the absence of mechanisms such as:

```text
RDF list traversal
rdf:first
rdf:rest
rdf:type
IRI identity
blank-node identity
language-tagged literals
```

should not generally be interpreted as evidence that Shaxon lacks an equivalent general validation capability. These mechanisms are largely consequences of the RDF substrate.

This category accounts for approximately **7.5–9 percentage points** of the total functionality estimate.

---

# 2. RDF graph construction, traversal, and representation machinery

A second significant portion of the gap results from the machinery required to construct and navigate RDF graphs.

For example, a relationship such as:

```text
Alice knows Bob
```

is represented in RDF as:

```text
<alice> <knows> <bob>
```

Shapes themselves are also RDF structures.

Consequently, SHACL requires mechanisms for expressing and accessing concepts such as:

* property paths,
* inverse paths,
* alternative paths,
* sequences,
* RDF collections,
* graph-node relationships,
* focus nodes,
* value nodes,
* RDF terms,
* shape nodes,
* property shapes,
* shape references.

The SHACL 1.2 Core specification defines property shapes in terms of following RDF properties or RDF property paths, and targets in terms of producing RDF terms from the data graph. ([W3C][1])

Shaxon can often represent the corresponding concepts more directly in its own data model:

```text
path
index
relation
target
```

The distinction can therefore be characterized as follows:

```text
SHACL:
construct an abstract mechanism for navigating RDF graphs

Shaxon:
represent the path, index, relation, or target directly
```

A portion of the SHACL surface therefore consists of machinery required to expose RDF graph structure to the validation language rather than additional validation concepts in the abstract.

This category accounts for approximately **4–5 percentage points** of the total functionality estimate.

---

# 3. SPARQL is a substantive expressive gap

The SPARQL-related portion of the gap requires separate treatment. It should not be classified entirely as RDF scaffolding.

SHACL-SPARQL provides mechanisms for defining constraints and constraint components using SPARQL. SHACL 1.2 also provides SPARQL-based node expressions. ([W3C][2])

SHACL 1.2 Node Expressions additionally provide access to SPARQL functions. ([W3C][3])

The resulting model provides a broad computational escape mechanism:

```text
arbitrary graph query
        ↓
arbitrary expression
        ↓
constraint
```

This represents genuine additional computational expressiveness.

At the same time, a substantial portion of this capability is specifically RDF/SPARQL capability. SPARQL operates on:

* RDF triples,
* graph patterns,
* RDF terms,
* graph matching,
* `OPTIONAL`,
* `UNION`,
* `FILTER`,
* aggregation,
* graph functions,
* RDF-specific operators.

The approximately **4–5 percentage points** attributed to SPARQL-related functionality can therefore be divided conceptually into:

* approximately **2–3 percentage points** of genuinely additional expressive power;
* approximately **2 percentage points** of power obtained specifically through the RDF/SPARQL graph-query substrate.

The complete SPARQL gap should therefore not be described simply as missing validation functionality.

The architectural distinction is:

> **SHACL provides an open-ended RDF query language as an escape mechanism; Shaxon provides a closed, bounded computation language with deterministic semantics.**

This is a genuine capability difference.

---

# 4. SHACL ecosystem and surrounding specifications

The distinction becomes more significant when the complete SHACL 1.2 family is considered rather than SHACL Core alone.

The current SHACL 1.2 family includes specifications for:

* Core,
* SPARQL-related features,
* Node Expressions,
* Inference Rules,
* Profiling,
* UI,
* Compact Syntax. ([W3C][1])

Not all of these represent additional core validation functionality.

## 4.1 SHACL UI

SHACL UI functionality concerns the use of shapes to help generate user interfaces.

This is separate from core validation expressiveness. Reproducing it would require Shaxon to adopt UI generation as part of its problem definition.

## 4.2 SHACL Profiling

Profiling concerns describing and profiling SHACL and RDF data.

This represents ecosystem infrastructure rather than a core validation capability.

## 4.3 Compact Syntax

Compact Syntax provides another syntax for expressing SHACL concepts.

A different serialization or syntax does not constitute an additional validation capability.

## 4.4 Inference Rules

Inference Rules represent a more substantive capability, but they perform a different operation:

```text
existing RDF
    ↓
inference rules
    ↓
new RDF triples
```

rather than:

```text
data
    ↓
validation
    ↓
report
```

The current SHACL 1.2 inference-rules draft explicitly describes a framework for deriving inferred RDF triples. ([W3C][4])

Jaxson provides a different and more general mutation/execution model for state transformation.

Approximately **2.5–3.75 percentage points** of the overall 25-point gap can therefore be attributed to this broader ecosystem, scaffolding, UI, profiling, syntax, and related functionality.

---

# 5. The substantive residual

Starting from the overall practical-functionality estimate:

```text
SHACL practical functionality                 100%
Jaxson + Shaxon v0.3.1                        ~75%
                                            ----
uncovered                                      ~25%
```

the majority of the remaining surface can be associated primarily with:

```text
RDF representation
RDF semantics
RDF graph construction
RDF-specific navigation
SHACL serialization
UI/profiling/tooling
```

After those categories are separated out, approximately:

> **4–6 percentage points of the total practical functionality remain as genuinely substantive additional capability.**

The largest component of this residual is **open-ended graph/query computation**.

SHACL can express:

> An arbitrary SPARQL expression or query defines the constraint.

Shaxon v0.3.1 instead provides:

> A closed Jaxson computation language with explicit bounds and deterministic semantics.

There are therefore real constraints expressible in SHACL that cannot currently be expressed in Shaxon.

This distinction should remain explicit in any capability comparison.

---

# 6. Revised interpretation of the 75% estimate

The practical-coverage estimate is better represented as:

```text
SHACL practical capability
──────────────────────────────────────── 100%

≈75%  ─── covered by Jaxson + Shaxon

≈7–9% ── RDF semantic machinery
≈4–5% ── RDF graph representation/navigation machinery
≈2–3% ── RDF/SPARQL-specific expression machinery
≈2–4% ── ecosystem / UI / profiling / syntax / tooling
≈4–6% ── genuinely additional expressive capability
```

The category boundaries overlap to some extent. The values should therefore not be interpreted as laboratory measurements or summed as independently measured quantities.

The qualitative result is nevertheless relatively stable:

* most of the uncovered surface is associated with RDF-specific representation or semantics;
* another portion is associated with the broader SHACL ecosystem;
* a smaller residual represents genuinely additional expressive capability;
* open-ended SPARQL-style computation is the most significant component of that residual.

---

# 7. Practical implications

The distinction between these categories is important when evaluating Shaxon's coverage.

The statement:

> **"Shaxon covers 75% of SHACL."**

can incorrectly suggest that approximately 25% of the general validation concepts available in SHACL are simply absent from Shaxon.

A more precise characterization is:

> **Shaxon v0.3.1 appears to cover roughly three quarters of the practical SHACL workload, while most of the remaining surface is tied to RDF's representation and semantic model or to the larger SHACL ecosystem. The genuinely nontrivial expressive gap is probably closer to 4–6 percentage points of the total practical workload, with open-ended SPARQL-style computation being the most important component.**

This distinction separates:

1. capabilities that Shaxon does not reproduce because its underlying data model does not require them;
2. capabilities that belong to RDF-specific graph-query machinery;
3. ecosystem and tooling capabilities outside core validation;
4. capabilities that represent genuinely greater expressive power.

The fourth category is the principal target for any future analysis of Shaxon's remaining expressive gap.

---

# 8. Implication for future Shaxon expressiveness

A potential future direction is a carefully bounded and deterministic analogue of the genuinely useful subset of SPARQL expressions.

Such an extension could reduce the remaining practical validation gap without reproducing:

* RDF itself,
* the complete SPARQL language,
* RDF-specific representation mechanisms,
* the broader SHACL ecosystem,
* UI generation,
* profiling infrastructure,
* alternative serialization syntaxes.

The relevant design objective would therefore not be feature-for-feature SHACL compatibility.

A more targeted objective would be:

> **Provide the subset of open-ended computational expressiveness that is materially useful for validation while preserving Shaxon's boundedness and deterministic execution model.**

Such a capability would address the most significant remaining substantive expressive gap while retaining the architectural distinction between Shaxon and the RDF/SPARQL execution model.

---

# References

[1]: https://www.w3.org/TR/2026/WD-shacl12-core-20260917/ "SHACL 1.2 Core"
[2]: https://www.w3.org/TR/shacl12-sparql/ "SHACL 1.2 SPARQL-Related Features"
[3]: https://www.w3.org/TR/2026/WD-shacl12-node-expr-20260108/ "SHACL 1.2 Node Expressions"
[4]: https://www.w3.org/TR/shacl-inference-rules/ "SHACL 1.2 Inference Rules"
