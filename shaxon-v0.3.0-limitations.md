# Shaxon v0.3.0 — limitations relative to SHACL

This document describes patterns SHACL supports — natively, or by reaching for
its open extension points (SPARQL-based constraints, SHACL-AF rules and
functions) — that Shaxon v0.3.0's closed vocabulary does not yet reach, or
reaches only partially. For each pattern: what SHACL does, a short example,
what Shaxon v0.3.0 can and can't currently express, and what Shaxon would need
to add to close the gap. Patterns Shaxon deliberately never intends to match
(the RDF/triple data model, IRI/blank-node identity, unbounded property-path
closure) are out of scope here — see shaxon-v0.3.0-core.md section 11 for
those. This document is about gaps that are candidates for future work, not
non-goals.

## 1. Property paths as a composable algebra

**SHACL.** `sh:path` accepts arbitrary composition of sequence, alternative,
inverse, and repetition, with normal algebraic precedence:

```turtle
sh:path ( [ sh:inversePath :parent ] [ sh:alternativePath (:manager :lead) ] )
```

reads as "the inverse of `parent`, followed by either `manager` or `lead`" —
one path expression built out of smaller ones.

**Shaxon v0.3.0.** `$altPath`, `$path*`, `$path+`, and `$inverse` (core §6)
each stand alone; none can be nested inside another. There is no way to write
"the alternative of two sequences" or "the inverse of a bounded closure" as a
single path expression — each form has to be the outermost (and only) wrapper
around a plain segment array.

**What Shaxon would need.** A path-expression grammar in which `$altPath`,
`$path*`/`$path+`, and `$inverse` can each appear as `innerStep` or as an
alternative inside one another, plus a rule for how a bound like `maxDepth`
composes when a closure is nested inside another form (a closure inside a
closure needs two depth budgets, not one).

## 2. Targeting a population by structure rather than by a tag

**SHACL.** A target can be another shape's own success set: "every node
that already conforms to shape X" (`sh:targetShape`, or `sh:target` bound
to a shape-based target). No field on the data needs to say what kind of
thing it is; the shape itself is the classifier.

```turtle
sh:target [ a sh:Target ; sh:targetShape :DiscountLineShape ] .
```

**Shaxon v0.3.0.** `{"$discriminator": {"at": [...], "field": "kind",
"value": "discount"}}` (core §7) is the only "give me the population of
things that are Xs" target form, and it requires an explicit tag field to
match against. Given

```json
{"kind": "discount", "amount": -5}
```

`$discriminator` can find it. Given the same semantic object without a
`kind` field — identifiable only by "it has an `amount` field and no
`quantity` field" — there is no target form that selects it.

**What Shaxon would need.** A target form such as `{"$conforms": "ShapeName"}`
over a given population (`$each`, an index, etc.), meaning "every element of
this population that satisfies `ShapeName`" — itself a shape-check, reused as
a filter rather than as pass/fail on an already-fixed focus node. This is
already flagged as open question 2 in core §12.

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

**Shaxon v0.3.0.** `severity` is a single value on the shape (core §4):
`violation` | `warning` | `info` for the whole shape, not per field. To get
an `Order` where a missing `id` is a violation but a missing `legacyCode` is
only a warning, the fields have to be split into two shapes and combined —
typically with `and`, so both still have to pass for the combined shape to
conform, which is not the same thing as "collect both findings at their own
severity level."

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
mechanism `extends` already uses — the machinery for "this one member behaves
differently from the shape's default" already exists for inheritance; it
would need extending to severity.

## 4. Per-constraint diagnostic granularity in the report

**SHACL.** Every `sh:ValidationResult` names the exact failing constraint
component (`sh:sourceConstraintComponent`), the exact path (`sh:resultPath`),
and the exact offending value (`sh:value`) — so a single focus node with
three unrelated constraint failures produces three separate, individually
addressable results.

**Shaxon v0.3.0.** The validation report (core §10) carries `focusPath`,
`shape`, `severity`, `message`, and `code` — one entry per focus node per
target, naming the shape that was checked, not which field or keyword inside
it failed, and not the offending value itself. The one place granularity is
explicitly specified is a `reference` inside an array (core §5), where each
dangling element does get its own violation — because that case is
data-shaped (an array) rather than field-shaped. For an ordinary object shape
with several independently failing fields (`id` missing *and* `status` not
in its `enum`), the spec does not currently say whether that is one
violation or several, and there is no `value` field to say what `status` was.

**What Shaxon would need.** Either a normative rule that field-level
constraint failures within `fields` are individually reported (each with its
own sub-`focusPath` down to the failing field) the same way the `reference`-
in-array case already is, or an explicit statement that only the first field
failure per focus node is reported — plus, either way, a `value` member on
the violation record.

## 5. Regular-expression / structured-format string constraints

**SHACL.** `sh:pattern` validates a string against a regular expression,
commonly used for anything with a textual format not otherwise typed —
postal codes, SKUs, and (absent a native SHACL date type on a JSON-derived
system) ISO-8601-shaped date strings.

```turtle
sh:property [ sh:path :sku ; sh:pattern "^[A-Z]{3}-[0-9]{4}$" ] .
```

**Shaxon v0.3.0.** Inherited unchanged from Jaxson (core §1, and Jaxson's own
contract vocabulary): `pattern` is deliberately absent, on the grounds that
regex engines disagree at the edges and a total, portable language can't
depend on one that isn't specified as part of it. `{"kind": "string"}` has
`minLen`, `maxLen`, `enum` — nothing that inspects the string's internal
shape. A `$compute` `check` can hand-roll a specific check (e.g. `len` plus a
few character comparisons for a fixed-width code) but there is no general
pattern facility, and nothing that could validate an arbitrary date format
without one.

**What Shaxon would need.** Jaxson's own design already names the shape of
the fix: "If it is added, it is a named profile with a fixed grammar" — a
closed, deterministic pattern-matching operator (or a small closed set of
common ones: fixed-width alphanumeric, a specific date format) added to the
`$compute` operator table (core §1 / Jaxson §7), not open regex.

## 6. Value-membership constraints on non-string types

**SHACL.** `sh:in` restricts a value to an explicit list, for any RDF term
type — numbers, dates, IRIs, booleans:

```turtle
sh:property [ sh:path :priority ; sh:in ( 1 2 3 ) ] .
```

**Shaxon v0.3.0.** `enum` exists only on `{"kind": "string"}` (core §4). A
number, boolean, or object field has no equivalent "must be one of these
exact values" constraint short of a `check` that hand-writes an `or` of `eq`
comparisons.

**What Shaxon would need.** `enum` generalised to `number`, `boolean`, and
`null` kinds (skipping `object`/`array`, where deep-equality-against-a-list
starts to look like a different feature) rather than left as a `check`
work-around for a constraint this common.

## 7. Report-mode uniqueness / functional-dependency constraints

**SHACL.** A "no two siblings share this value" constraint is typically
written as a `sh:sparql`-based constraint with its own severity, so a
duplicate can be a `warning` and still show up in a full conformance report
next to everything else.

**Shaxon v0.3.0.** The only place uniqueness is enforced at all is a
non-`multi` index's key: "a repeated key is a `SHAPE_ERROR` at build time"
(core §3). That gives Shaxon a uniqueness check "for free" wherever an
index already exists over the field in question — but `SHAPE_ERROR` is not
a `report`-mode finding; it aborts the whole package the moment two elements
collide, with no `severity`, no `message` of the author's choosing, and no
way to keep validating the rest of the target's population to see what else
is wrong. There is no way to say "flag every duplicate `sku` as a `warning`
and still tell me about the other, unrelated violations in the same report."

**What Shaxon would need.** A `qualified`- or `check`-level construct that
can compare a focus node against the rest of its own population (not just
against itself, which is all `check` currently sees via `local.focus`) and
report a `severity`-bearing finding rather than aborting — most naturally, a
target-level "unique" declaration alongside `qualified`, keyed by a field or
index, that participates in `report` mode like anything else in section 10.

## 8. Reusable, named computations shared across shapes

**SHACL** (via SHACL-AF). `sh:function` declares a named, reusable
computation once; any number of constraints across any number of shapes can
call it by name.

**Shaxon v0.3.0.** `$compute` (core §1, inherited from Jaxson) is always
written inline, once per `check`. Two shapes that both need "is this total
non-negative after rounding to two decimal places" each write out their own
`with`/`expr` in full; there is no way to declare that computation once and
reference it from both. `extends` (core §4) reuses a whole `check` clause
from a parent shape, which covers the case where the reuse is already
structured as inheritance, but not the case of two otherwise-unrelated
shapes wanting the same small computation.

**What Shaxon would need.** A named, package-level compute-island registry
(parallel to `indices`, `relations`, and `shapes`), invocable from any
shape's `check` by name with its own `with` bindings — kept closed and
step-bounded the same way an inline `$compute` already is, so this is an
authoring convenience, not a new capability.

## 9. Importing shapes across separately-authored documents

**SHACL.** A shapes graph can import another shapes graph (`owl:imports`,
or a SHACL processor's own multi-graph loading), so an organisation can
maintain one library of common shapes (`:PersonShape`, `:AddressShape`) and
reuse it across many separate validation documents without copying it.

**Shaxon v0.3.0.** `shapes`, `indices`, and `relations` are all registries
declared inside one package (core §2); there is no import or module
mechanism. A shared `Address` shape used across ten packages currently has
to be copy-pasted into all ten, with the attendant risk of the copies
drifting apart.

**What Shaxon would need.** Some notion of a package reference — at minimum,
a way to name an external package and pull named `shapes`/`indices`/
`relations` members from it into the current one's registries before
`validate` runs, with the same duplicate-key discipline (core §2) applied
across the import boundary as within one package.

## 10. Entailment / inferred data as part of validation

**SHACL** (via SHACL-AF `sh:rule`). A shapes graph can declare rules that add
inferred triples to the data graph before (or as part of) validation — e.g.,
"if a node has a `:manager`, infer that the manager `:supervises` this
node" — so a downstream constraint can validate against data that was never
in the original input.

**Shaxon v0.3.0.** Nothing in the validation pipeline generates new data.
Jaxson's `program` can compute and `set` derived values into `state` before
`check` runs against them (core §1, §8), which covers the case where a
program author is willing to write that derivation explicitly as an ordinary
mutation step — but there's no declarative "entailment rule" that runs
automatically as a phase of validation itself, the way SHACL-AF rules do.

**What Shaxon would need.** This is less a missing feature than a
deliberate design boundary worth stating plainly rather than closing:
Shaxon's `check`/`validate` are read-only over `state` by construction (no
validation-phase instruction writes), and introducing one would blur the
"validation is a pure predicate" property the rest of the report model
(core §10) depends on. If this gap is ever closed, it should be closed by
convention (an explicit `program` step before the `check` that needs the
derived data) rather than by adding write access to validation itself.

## 11. Arbitrary, undeclared relational reach

**SHACL** (via `sh:sparql`). A SPARQL-based constraint can walk the entire
graph on demand — joins, aggregates, and filters over any nodes and edges,
without those nodes and edges being declared as a target or an index ahead
of time.

**Shaxon v0.3.0.** All relational reach goes through a declared `indices` or
`relations` entry (core §3, §4a), built from a named `source` path. A
`check` can only look up what an index already enumerates; it cannot express
"sum the `amount` field over every element anywhere in `input` whose
`category` matches mine," unless an index or relation over that exact
population was declared in advance. This is a deliberate consequence of
totality (an undeclared, unbounded graph walk has no step-accounting story),
not an oversight — but it means some SPARQL-based constraints that were easy
to write ad hoc in SHACL require upfront index design in Shaxon, and a
constraint whose population isn't known until the data is loaded (as
opposed to known from the package structure) has no expression at all.

**What Shaxon would need.** Nothing that preserves totality — this is the
one item on this list where the honest answer is "SHACL's version of this
pattern is unbounded by construction, and Shaxon's bounded-computation
premise means the two are not reconcilable without either giving up
determinism or requiring the index/relation to be declared." Any narrowing
of this gap should come from making index/relation declaration cheaper or
more automatic, not from adding an escape hatch that reintroduces
open-ended traversal.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
