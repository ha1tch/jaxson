# Jaxson + Shaxon and SHACL: Condensed Technical Assessment

## Executive Summary

Jaxson and Shaxon occupy a substantially different design space from SHACL, but there is a large practical overlap between them.

Jaxson is a deterministic JSON-native computation and mutation language. Shaxon builds a declarative validation and relational layer on top of that execution model. Together they provide structural validation, relational validation, business-rule computation, explicit references and relations, targets, reusable shapes, recursive structures, validation reports, and bounded deterministic execution.

SHACL, by contrast, is fundamentally an RDF graph constraint language. Its expressive model is built around RDF graphs, RDF terms, RDF property paths, shapes, targets, constraints, and—through its extensions—SPARQL-based graph querying, node expressions, inference, profiling, UI-related facilities, and other ecosystem capabilities.

The practical result is not that one system simply contains the other.

A useful approximation from the study is that **Jaxson + Shaxon v0.3.1 cover roughly 75% of the practically important SHACL workload, with an uncertainty of approximately ±10 percentage points**. The estimate is about practical validation work, not about the percentage of the SHACL specification reproduced feature-for-feature.

The remaining approximately 25% should also not be interpreted as 25% missing validation concepts. Much of it consists of RDF-specific representation and semantics, RDF graph-navigation machinery, and surrounding SHACL ecosystem functionality.

After those categories are separated out, the genuinely substantive expressive gap is estimated at approximately **4–6 percentage points of the total practical workload**. The largest part of that residual gap is SHACL's ability to use open-ended graph/query computation, particularly through SPARQL.

The resulting picture is therefore better described as:

> **Jaxson + Shaxon cover most of the ordinary practical validation workload addressed by SHACL, but do not currently reproduce SHACL's open-ended graph-query model or RDF-specific semantic model.**

---

# The Two Systems Solve Related but Different Problems

The conceptual distinction is important.

Jaxson provides a deterministic computation and mutation environment:

```text
data
  ↓
Jaxson computation
  ↓
deterministic result / mutation
```

Shaxon adds declarative validation and relational semantics:

```text
data
  ↓
Shaxon shapes / relations / targets
  ↓
Jaxson computation
  ↓
validation result
```

SHACL instead starts from the RDF graph model:

```text
RDF graph
  ↓
SHACL shapes / paths / constraints
  ↓
validation report
```

and can extend this through graph-query mechanisms:

```text
RDF graph
  ↓
SPARQL / node expression
  ↓
arbitrary graph computation
  ↓
constraint
```

This difference explains much of the comparison. Shaxon does not attempt to reproduce RDF merely because SHACL uses RDF. It establishes its own explicit representation of structural and relational information suitable for JSON-oriented data.

---

# Practical Coverage

The strongest overlap is in ordinary validation work: defining structures, requiring fields, checking cardinalities and types, validating nested objects, composing constraints, checking references and relationships, applying business rules, and producing validation results.

The estimated practical coverage is approximately:

| Area                                         | Estimated practical coverage by Jaxson + Shaxon v0.3.1 |
| -------------------------------------------- | -----------------------------------------------------: |
| Ordinary structural validation               |                                                ~80–90% |
| Relationship validation                      |                                                ~75–85% |
| Business-rule validation                     |                                                ~70–80% |
| Overall practically important SHACL workload |                                        **~75% ±10 pp** |

These are workload estimates, not measurements of formal specification coverage.

The high structural coverage follows from the fact that Shaxon already has the major mechanisms needed for ordinary shape validation: shapes, fields, required members, types, closed structures, nested and recursive shapes, logical composition, qualified constraints, enumerations, severity, messages, and reports.

Relational coverage is also significant. Shaxon explicitly models concepts such as indices, references, relations, inverses, targets, and cardinalities. This gives it a way to represent referential structure that JSON itself does not inherently provide.

Business rules are supported through Jaxson computation and checking. Conditions such as arithmetic comparisons, conditional rules, cross-field relationships, and domain-specific calculations can therefore be expressed without introducing a separate query language.

---

# Where the Architectures Differ

The major differences can be summarized compactly:

| Dimension               | Jaxson + Shaxon                                             | SHACL                                          |
| ----------------------- | ----------------------------------------------------------- | ---------------------------------------------- |
| Primary data model      | JSON-oriented data and explicit graph/relational structures | RDF graphs                                     |
| Computation             | Jaxson deterministic computation                            | SHACL constraints, with SPARQL extensions      |
| Relationships           | Explicit indices and relations                              | RDF predicates and property paths              |
| Traversal               | Explicit and bounded                                        | RDF property-path semantics                    |
| Mutation                | First-class Jaxson capability                               | Not the central SHACL operation                |
| Execution bounds        | Explicit design concern                                     | Not the same universal bounded-execution model |
| Exact computation       | Jaxson's computation semantics                              | Depends on RDF/SPARQL execution model          |
| Validation composition  | Shaxon shapes and constraint composition                    | SHACL shapes and constraint components         |
| Arbitrary graph queries | Not currently provided as a general abstraction             | SPARQL provides an open-ended query mechanism  |
| Target selection        | Explicit Shaxon target machinery                            | SHACL target mechanisms and node expressions   |
| Validation reports      | Specified Shaxon model                                      | Native SHACL validation reports                |
| RDF semantics           | Not reproduced                                              | Fundamental to SHACL                           |
| UI/profiling/ecosystem  | Outside core design                                         | Part of broader SHACL family                   |

The differences therefore concern both **representation** and **computational philosophy**.

---

# What the Remaining ~25% Represents

The uncovered surface is heterogeneous.

A substantial portion consists of things that exist because SHACL is an RDF technology:

```text
RDF terms and datatypes
RDF classes and rdf:type
language-tagged literals
blank nodes
RDF lists and reification
RDF property-path representation
RDF graph semantics
RDF-specific graph navigation
```

Another portion belongs to the broader SHACL ecosystem rather than to basic validation expressiveness:

```text
UI
profiling
alternative syntaxes
inference facilities
serialization/ecosystem conventions
```

These differences are real, but they should not be conflated with missing general-purpose validation concepts.

A useful approximate decomposition is:

| Part of the original ~25% gap                  | Approximate share of total workload |
| ---------------------------------------------- | ----------------------------------: |
| RDF semantic machinery                         |                             ~7–9 pp |
| RDF graph representation/navigation            |                             ~4–5 pp |
| RDF/SPARQL-specific substrate                  |                             ~2–3 pp |
| Ecosystem, UI, profiling, syntax, tooling      |                             ~2–4 pp |
| **Residual substantive expressive capability** |                         **~4–6 pp** |

The boundaries overlap, so these figures should be treated as an analytical decomposition rather than precise measurements.

---

# The Genuine Expressiveness Gap

The most important remaining difference is **open-ended graph/query computation**.

SHACL can delegate a constraint to an arbitrary graph query. SPARQL makes it possible to discover graph patterns, bind variables, combine matches, filter results, use optional or negative patterns, aggregate results, and construct expressions from the resulting bindings.

Conceptually:

```text
arbitrary graph pattern
        ↓
bindings
        ↓
expression
        ↓
constraint
```

Shaxon does not currently have an equivalent general-purpose graph-query abstraction.

It has something deliberately different:

```text
explicit relation
        ↓
bounded traversal
        ↓
Jaxson computation
        ↓
constraint
```

This distinction explains why Shaxon can express many sophisticated business rules while still failing to match the full expressive envelope of SHACL-SPARQL.

The missing capability is not simply arithmetic, conditionals, aggregation, or reusable computation. Jaxson already supplies much of that.

The missing capability is the ability for the **validation expression itself to discover arbitrary graph structure and produce a dynamic set of bindings**.

That distinction is central to the comparison.

---

# What the Residual Gap Looks Like in Practice

Consider a rule of the form:

```text
For every customer,

find all active contracts whose supplier belongs
to an organization operating in the customer's region,

aggregate their value,

and require the result to remain below the customer's limit.
```

A graph-query system can formulate this as one query involving:

```text
customer
  → supplier
  → organization
  → region
```

together with filters, joins, and aggregation.

Shaxon can represent the individual relationships and can perform computation over explicitly accessible data. What it does not currently provide is a general declarative operation meaning:

```text
search the graph for every binding satisfying this arbitrary pattern
```

The same limitation appears in more specialized forms:

* arbitrary multi-hop joins;
* dynamically derived target sets;
* existential or universal conditions over arbitrary graph patterns;
* negated graph patterns;
* optional graph matching;
* aggregation over query-derived bindings;
* reusable constraints backed by arbitrary graph queries.

These are not ten independent architectural omissions. They are mostly consequences of the same missing abstraction: **general graph-query computation**.

---

# What Shaxon Already Does Instead

The absence of arbitrary graph queries should not obscure the fact that Shaxon has a distinctive alternative.

Rather than making the validator discover all relational structure dynamically, it makes important relationships explicit:

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

This is particularly appropriate when the domain model already knows what constitutes a reference, relation, index, or target.

Combined with Jaxson, this gives a coherent execution model:

```text
explicit structure
       +
deterministic computation
       +
bounded traversal
       +
validation
       +
mutation
```

This is not merely a weaker version of SPARQL.

It represents a different set of priorities: explicitness, determinism, bounded execution, predictable computation, and integration with mutation.

---

# The Importance of Boundedness

One of the strongest architectural distinctions is the treatment of computation limits.

Shaxon explicitly incorporates concepts such as shape-depth limits and bounded traversal. Jaxson provides execution accounting and limits as part of its computation model.

A general query language provides considerably greater freedom to discover and combine graph structure. That freedom is precisely what gives SPARQL much of its expressive power, but it also means that reproducing the full model would introduce a substantially different computational subsystem.

Consequently, the most coherent route toward closing the remaining gap would not necessarily be to implement SPARQL.

A more consistent direction would be a **bounded, deterministic graph-expression facility** that provides useful subsets of:

```text
MATCH
JOIN
FILTER
EXISTS
NOT EXISTS
OPTIONAL
GROUP
AGGREGATE
```

while maintaining explicit limits on traversal, bindings, intermediate result size, iteration, and total computation.

Such a facility could extend the existing Shaxon model rather than replacing it.

---

# What Shaxon Does Not Need to Reproduce

The comparison also indicates several areas where feature-for-feature SHACL compatibility would not necessarily be a useful objective.

Reproducing RDF's complete term model would largely reproduce the data model of another technology rather than address a missing Shaxon validation capability.

Likewise, reproducing SHACL UI, profiling, compact syntaxes, or other ecosystem facilities would expand Shaxon into adjacent domains without necessarily improving its core validation model.

Inference rules are somewhat different: they are substantial functionality, but they perform derivation rather than ordinary validation. Jaxson's general mutation and execution model already occupies a different position in that space.

The relevant question is therefore not:

> How can Shaxon implement every SHACL feature?

but rather:

> Which SHACL capabilities represent useful validation semantics that the Jaxson + Shaxon model does not currently provide?

On that question, the residual gap is much smaller.

---

# The Current Implementation Position

There is an important distinction between the architecture and the current implementation.

Jaxson v0.3.1 has crossed into implemented, tested software. Its core execution model, paths, mutation, computation, deterministic behavior, exact decimal handling, execution limits, checking infrastructure, and extension architecture are implemented and tested.

Shaxon v0.3.1 has a credible static/declarative implementation foundation, including registries, parsing and static validation, shape composition, `extends` handling, recursive analysis, depth checking, deterministic ordering, and delegation to Jaxson checking.

The complete Shaxon runtime evaluator remains unfinished. In particular, full shape evaluation, nested and qualified runtime evaluation, runtime index construction/use, relation validation, runtime targets, end-to-end validation instructions, gate/report execution, and the complete validation pipeline remain to be implemented.

Therefore the comparison should not be read as claiming that all of the approximately 75% practical coverage is already production-ready Shaxon runtime functionality.

A more precise characterization is:

> **The Jaxson foundation is implemented and credible; the Shaxon specification and static implementation foundation are substantial, but the full Shaxon evaluator remains the major implementation milestone.**

---

# Overall Assessment

The study does not support characterizing Shaxon as simply a replacement for SHACL.

It also does not support characterizing it as merely a reduced JSON version of SHACL.

The more accurate description is that Jaxson + Shaxon establish a different validation architecture that overlaps strongly with the practical center of SHACL while making different choices about data representation, relational structure, computation, execution bounds, and mutation.

The approximately 75% practical-workload coverage is significant because it is concentrated in the parts of validation that recur in ordinary applications: structure, cardinality, types, nested data, logical constraints, relationships, references, and business rules.

The approximately 25% outside that coverage is less homogeneous than the number suggests. Most of it is explained by RDF itself or by the broader SHACL ecosystem. The genuinely substantive gap is closer to **4–6%**, and is dominated by one architectural capability:

> **open-ended graph/query computation.**

This is the principal area in which SHACL currently has expressive power that Jaxson + Shaxon v0.3.1 do not match.

At the same time, that gap is closely connected to a deliberate Shaxon design choice. Shaxon exchanges some of the unrestricted graph-query freedom of SHACL/SPARQL for explicit relationships, deterministic computation, bounded traversal, and integration with Jaxson.

The resulting design is therefore best understood not as an attempt to reproduce SHACL feature-for-feature, but as a **deterministic, bounded data-validation and computation architecture whose practical validation territory substantially overlaps SHACL while avoiding dependence on RDF's semantic and representation model**.

The principal technical question for future development is correspondingly narrow: whether Shaxon can acquire enough bounded graph-query expressiveness to cover the remaining substantive workload without sacrificing the determinism, explicitness, and resource-bounded execution model that distinguish it.

That is a considerably more focused problem than reproducing SHACL as a whole.
