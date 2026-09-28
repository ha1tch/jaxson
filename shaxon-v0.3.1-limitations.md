# Shaxon v0.3.1 — limitations relative to SHACL

This document describes patterns SHACL supports — natively, or by reaching for
its open extension points (SPARQL-based constraints, SHACL-AF rules and
functions) — that Shaxon v0.3.1's closed vocabulary does not yet reach, or
reaches only partially. For each pattern: what SHACL does, a short example,
what Shaxon v0.3.1 can and can't currently express, and what Shaxon would need
to add to close the gap. Patterns Shaxon deliberately never intends to match
(the RDF/triple data model, IRI/blank-node identity, unbounded property-path
closure) are out of scope here — see shaxon-v0.3.1-core.md section 11 for
those. This document is about gaps that are candidates for future work, not
non-goals.

This is a new, parallel document for this revision — it does not restate or
supersede shaxon-v0.3.0-limitations.md. Four items from that document are
absent here because v0.3.1 closed them: per-constraint diagnostic granularity,
non-string `enum`, report-mode uniqueness, and named compute reuse. See
shaxon-v0.3.1-editorial.md for exactly what closed each one and where.

## 1. Property paths as a composable algebra

**SHACL.** `sh:path` accepts arbitrary composition of sequence, alternative,
inverse, and repetition, with normal algebraic precedence:

```turtle
sh:path ( [ sh:inversePath :parent ] [ sh:alternativePath (:manager :lead) ] )
```

reads as "the inverse of `parent`, followed by either `manager` or `lead`" —
one path expression built out of smaller ones.

**Shaxon v0.3.1.** `$altPath`, `$path*`, `$path+`, and `$inverse` (core §6)
each stand alone; none can be nested inside another. There is no way to write
"the alternative of two sequences" or "the inverse of a bounded closure" as a
single path expression — each form has to be the outermost (and only) wrapper
around a plain segment array. Unchanged from v0.3.0.

**What Shaxon would need.** A path-expression grammar in which `$altPath`,
`$path*`/`$path+`, and `$inverse` can each appear as `innerStep` or as an
alternative inside one another, plus a rule for how a bound like `maxDepth`
composes when a closure is nested inside another form (a closure inside a
closure needs two depth budgets, not one). No accounting rule for this exists
yet; proposing syntax ahead of one would bake in an arbitrary choice.

## 2. Targeting a population by structure rather than by a tag

**SHACL.** A target can be another shape's own success set: "every node
that already conforms to shape X" (`sh:targetShape`, or `sh:target` bound
to a shape-based target). No field on the data needs to say what kind of
thing it is; the shape itself is the classifier.

```turtle
sh:target [ a sh:Target ; sh:targetShape :DiscountLineShape ] .
```

**Shaxon v0.3.1.** `{"$discriminator": {"at": [...], "field": "kind",
"value": "discount"}}` (core §7) is the only "give me the population of
things that are Xs" target form, and it requires an explicit tag field to
match against. Given

```json
{"kind": "discount", "amount": -5}
```

`$discriminator` can find it. Given the same semantic object without a
`kind` field — identifiable only by "it has an `amount` field and no
`quantity` field" — there is no target form that selects it. This is the one
item carried over unresolved from v0.3.0's original six open questions (core
§12); every other original open question closed this revision.

**What Shaxon would need.** A target form such as `{"$conforms": "ShapeName"}`
over a given population (`$each`, an index, etc.), meaning "every element of
this population that satisfies `ShapeName`" — itself a shape-check, reused as
a filter rather than as pass/fail on an already-fixed focus node. The specific
obstacle, not merely an unwritten feature: a structural target needs shape
evaluation to run as part of target resolution itself, which risks
`target → shape → target → shape` recursion — a direct threat to the
boundedness invariant (core §0) the rest of the design treats as
non-negotiable. A proposal here needs an answer to that recursion before it
needs syntax.

## 3. Different severities for different constraints on the same shape

**SHACL.** `sh:severity` can be attached to an individual constraint
component inside a node shape, not only to the shape as a whole:

```turtle
:OrderShape a sh:NodeShape ;
  sh:property [ sh:path :id ; sh:minCount 1 ] ;                       # violation (default)
  sh:property [ sh:path :legacyCode ; sh:datatype xsd:string ;
                sh:severity sh:Warning ] .                             # warning
```

One shape, two different severities, depending on which constraint failed.

**Shaxon v0.3.1.** `severity` is still a single value on the shape (core §4):
`violation` | `warning` | `info` for the whole shape, not per field. To get
an `Order` where a missing `id` is a violation but a missing `legacyCode` is
only a warning, the fields have to be split into two shapes and combined —
typically with `and`, so both still have to pass for the combined shape to
conform, which is not the same thing as "collect both findings at their own
severity level." Unlike in v0.3.0, this gap now sits behind a resolved
prerequisite rather than beside an unresolved one: v0.3.1 fully specifies
violation granularity (core §10), which is what makes it possible to say
precisely what a per-field severity would even attach to.

```json
"OrderCore":   { "kind": "object", "fields": {"id": {...}}, "required": ["id"], "severity": "violation" },
"OrderLegacy": { "kind": "object", "fields": {"legacyCode": {...}}, "severity": "warning" }
```

— and then there is no single shape whose `report`-mode output naturally
carries both a `violation` finding and a `warning` finding for the same
`Order` focus node in one pass, only two separate `validate` entries against
the same target.

**What Shaxon would need.** A per-field (or per-combinator-branch) `severity`
override inside `fields`, alongside the existing per-field `"override": true`
mechanism `extends` already uses. This is deliberately sequenced after
granularity, not merged alongside it: severity of an `and`, of an inherited
field, of `qualified`, and of a nested shape failure all need their own
answers first, now that "what is one finding" has a fixed definition to
answer them against.

## 4. Regular-expression / structured-format string constraints

**SHACL.** `sh:pattern` validates a string against a regular expression,
commonly used for anything with a textual format not otherwise typed —
postal codes, SKUs, and (absent a native SHACL date type on a JSON-derived
system) ISO-8601-shaped date strings.

```turtle
sh:property [ sh:path :sku ; sh:pattern "^[A-Z]{3}-[0-9]{4}$" ] .
```

**Shaxon v0.3.1.** Inherited unchanged from Jaxson (core §1, and Jaxson's own
contract vocabulary): `pattern` is deliberately absent, on the grounds that
regex engines disagree at the edges and a total, portable language can't
depend on one that isn't specified as part of it. `{"kind": "string"}` has
`minLen`, `maxLen`, and (as of this revision) `enum` — nothing that inspects
the string's internal shape. A `$compute` `check`, optionally now a named
one (core §4d), can hand-roll a specific check (e.g. `len` plus a few
character comparisons for a fixed-width code) but there is no general
pattern facility, and nothing that could validate an arbitrary date format
without one.

**What Shaxon would need.** Jaxson's own design already names the shape of
the fix: "If it is added, it is a named profile with a fixed grammar" — a
closed, deterministic pattern-matching operator (or a small closed set of
common ones: fixed-width alphanumeric, a specific date format) added to the
`$compute` operator table (core §1 / Jaxson §7), not open regex. Simply out
of scope for this revision, which was scoped to closing the report-model and
internal-ambiguity items rather than adding vocabulary.

## 5. Importing shapes across separately-authored documents

**SHACL.** A shapes graph can import another shapes graph (`owl:imports`,
or a SHACL processor's own multi-graph loading), so an organisation can
maintain one library of common shapes (`:PersonShape`, `:AddressShape`) and
reuse it across many separate validation documents without copying it.

**Shaxon v0.3.1.** `shapes`, `indices`, `relations`, and (as of this
revision) `computes` are all registries declared inside one package (core
§2); there is no import or module mechanism. A shared `Address` shape used
across ten packages currently has to be copy-pasted into all ten, with the
attendant risk of the copies drifting apart.

**What Shaxon would need.** Some notion of a package reference — at minimum,
a way to name an external package and pull named `shapes`/`indices`/
`relations`/`computes` members from it into the current one's registries
before `validate` runs, with the same duplicate-key discipline (core §2)
applied across the import boundary as within one package. Deliberately not
attempted before a reference interpreter exists for the base language:
versioning, name collision, and dependency cycles are a module-system
problem in their own right, and designing one against a language that has
not yet been implemented risks solving the wrong problem.

## 6. Entailment / inferred data as part of validation

**SHACL** (via SHACL-AF `sh:rule`). A shapes graph can declare rules that add
inferred triples to the data graph before (or as part of) validation — e.g.,
"if a node has a `:manager`, infer that the manager `:supervises` this
node" — so a downstream constraint can validate against data that was never
in the original input.

**Shaxon v0.3.1.** Nothing in the validation pipeline generates new data.
Jaxson's `program` can compute and `set` derived values into `state` before
`check` runs against them (core §1, §8), which covers the case where a
program author is willing to write that derivation explicitly as an ordinary
mutation step — but there's no declarative "entailment rule" that runs
automatically as a phase of validation itself, the way SHACL-AF rules do.
This revision promotes what was previously an implicit consequence to a
stated invariant: "validation is a pure predicate" is now normative core
text (core §0, §1), not only an inference from how the report model happens
to be described.

**What Shaxon would need.** This remains less a missing feature than a
deliberate design boundary. Introducing validation-time writes would blur
the pure-predicate invariant the entire report model (core §10) now
explicitly depends on. If this gap is ever closed, it should be closed by
convention (an explicit `program` step before the `check` that needs the
derived data) rather than by adding write access to validation itself.

## 7. Arbitrary, undeclared relational reach

**SHACL** (via `sh:sparql`). A SPARQL-based constraint can walk the entire
graph on demand — joins, aggregates, and filters over any nodes and edges,
without those nodes and edges being declared as a target or an index ahead
of time.

**Shaxon v0.3.1.** All relational reach goes through a declared `indices` or
`relations` entry (core §3, §4a), built from a named `source` path. A
`check` can only look up what an index already enumerates; it cannot express
"sum the `amount` field over every element anywhere in `input` whose
`category` matches mine," unless an index or relation over that exact
population was declared in advance. This revision's `unique` declaration
(core §7) closes the narrower "does anything repeat" case as a report-mode
finding, but does not touch this broader one: a `unique` still operates over
a single declared target's population, not an arbitrary ad hoc join. This is
a deliberate consequence of totality (an undeclared, unbounded graph walk
has no step-accounting story), not an oversight.

**What Shaxon would need.** Nothing that preserves totality — this is the
one item on this list where the honest answer is "SHACL's version of this
pattern is unbounded by construction, and Shaxon's bounded-computation
premise means the two are not reconcilable without either giving up
determinism or requiring the index/relation to be declared." Any narrowing
of this gap should come from making index/relation declaration cheaper or
more automatic, not from adding an escape hatch that reintroduces
open-ended traversal.

## A note on scope

A considerably deeper comparison against the current W3C SHACL 1.2 draft
(node expressions, dynamic targeting, list constraints, derived properties,
and a proposed unifying "compositional expression layer" underlying most of
what remains) exists as external research material, not reflected in this
document. That material has not been distilled into settled items the way
the eleven items across this document and its v0.3.0 predecessor have been,
and mixing raw research findings into a document meant to track considered,
stable gaps would make this document's own status harder to read. It will
earn its own entry here once — and if — a specific mechanism is proposed and
checked the way each item above was.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
