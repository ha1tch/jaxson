# Shaxon v1: SHACL's expressiveness on Jaxson's total, deterministic core

Status: proposal, unimplemented — no reference interpreter or fixtures yet (see section 14).

## 0. Stance

Jaxson computes; SHACL validates. Neither does the other's job well, and gluing them
together at the API boundary (validate with one tool, hand the result to a different
runtime) throws away the one property both halves separately worked hard for:
determinism. Shaxon is a single closed spec where the constraint layer is exactly as
expressive as SHACL, but every constraint — including the ones SHACL would reach for
SPARQL or a JS extension to express — is written in the same total, step-bounded
sublanguage that runs the computation. One conformance claim covers both halves.

Where SHACL specifically comes short for JSON-native systems, and what Shaxon does
about it:

| Gap in SHACL for JSON | Why it's a gap | What Shaxon does |
|---|---|---|
| Data model is RDF triples | JSON must be lifted into a graph (JSON-LD or similar) before SHACL can see it, and document order/plain nesting is lost in the lift | Shaxon validates the JSON tree directly — Jaxson's roots and paths, unchanged |
| No computation | A conformance report is a dead end; you validate, then hand off to a *different* engine to act on the result, with no shared determinism guarantee | Validation and computation are phases of one package; a `check` instruction lets a program call into shapes mid-run |
| Custom constraints are open (SPARQL / JS) | `sh:sparql` and `sh:js` constraints can be arbitrarily expensive or fail to terminate; SHACL's own totality guarantee stops at the vocabulary's edge | Custom constraints are `$compute` islands — closed, step-bounded, no exception to the termination proof |
| Relational reach depends on real graph edges | SHACL's inverse paths and joins work because RDF has edges; a JSON tree has none | Shaxon adds declared, named **indices** — explicit, built-once maps that stand in for the edges a tree doesn't have |

Everything not called out below carries over from Jaxson v1 unchanged; this document
only specifies the delta.

## 1. What carries over from Jaxson unchanged

- The four roots (`input`, `state`, `output`, `local`) and plain segment-array paths.
- Operand forms: `$lit`, `$path`, `$compute`, `$tpl`, and `$opt` inside templates.
- The eight core instructions (`set`, `delete`, `append`, `insert`, `for`, `if`,
  `assert`, `halt`), plus one new instruction added in section 8.
- The compute sublanguage: closed operator set, `with`-bound operands, no reach into
  state.
- Exact-decimal numbers, the 38-digit magnitude bound, and `div`/`round` as the only
  digit-losing operations.
- Step-counted termination as the semantic resource bound.
- The error-category discipline: stable categories, stable codes within
  `EXECUTION_ERROR`, fixed pipeline order.

A Shaxon runtime that also implements Jaxson v1 can run a Jaxson package unmodified —
`shapes`, `validate`, and `indices` are all optional package members with an empty
default.

## 2. The package

```json
{
  "shaxon": "1.0",
  "limits": { "steps": 100000, "maxShapeDepth": 8 },
  "input": {},
  "inputSchema": {},
  "indices": {},
  "shapes": {},
  "validate": [],
  "program": [],
  "outputSchema": {}
}
```

- `shaxon` is mandatory, same failure discipline as Jaxson's `jaxson` key
  (`VERSION_ERROR` before anything else runs).
- `limits.maxShapeDepth` is mandatory whenever any shape in the package uses a bounded
  closure path (section 6) or `extends` (section 4); its absence in that situation is
  a `SHAPE_ERROR`, not a silent default — recursion is never accidentally unbounded.
- `inputSchema`/`outputSchema` keep Jaxson's original narrow contract vocabulary
  (section 8 of the Jaxson spec) for the common case where a package only needs
  type-and-shape gating at the two edges. `shapes`/`validate` are for everything a
  plain schema can't say. A package may use either, both, or neither.
- `indices` and `shapes` are named registries, read before `validate` runs.
- Duplicate keys anywhere, including inside `shapes` or `indices`, are a `PARSE_ERROR`
  — unchanged from Jaxson.

## 3. Indices: the relational-reach mechanism

A tree has no edges, so anything SHACL gets for free from RDF — "does this ID exist
elsewhere," "what points at me" — has to be declared explicitly. An index is a named,
deterministic map built once per validation pass from a source path and a key
expression.

```json
"indices": {
  "customersById": {
    "source": { "$path": ["input", "customers"] },
    "key": { "$path": ["local", "item", "id"] }
  },
  "ordersByCustomer": {
    "source": { "$path": ["input", "orders"] },
    "key": { "$path": ["local", "item", "customerId"] },
    "multi": true
  }
}
```

- `source` must resolve to an array; each element becomes `local.item` while `key` is
  evaluated against it, exactly like a `for` binding — snapshot iteration, no
  aliasing, same as everywhere else in Jaxson.
- Without `"multi": true`, a repeated key is a `SHAPE_ERROR` at build time (an index is
  a function unless you say otherwise). With `"multi": true`, the index maps a key to
  the *set* of matching elements — this is how an inverse relationship ("every order
  for this customer") is recovered without a graph edge.
- Indices are rebuilt from their declared source immediately before every use — by any
  `validate` entry and by every `check` instruction (section 8) — so a `program` that
  mutates `state` between checks is always validated against current data, never a
  stale snapshot. This follows directly from Jaxson's copy-and-no-aliasing discipline;
  it would be a surprising exception to cache it.
- A `reference` node kind (section 5) and an `$inverse` operand (section 6) are the
  two places indices get consumed.

## 4. Shapes

A shape is a named predicate over one JSON value (the *focus node*). The vocabulary is
deliberately at SHACL's level of expressiveness, not JSON Schema's.

```json
"shapes": {
  "Order": {
    "kind": "object",
    "closed": true,
    "ignoredProperties": ["$meta"],
    "fields": {
      "id":         { "kind": "string", "minLen": 1 },
      "customerId": { "kind": "reference", "index": "customersById" },
      "status":     { "kind": "string", "enum": ["draft", "placed", "shipped"] },
      "lines": {
        "kind": "array",
        "items": { "shape": "LineItem" },
        "qualified": { "shape": "DiscountLine", "min": 0, "max": 1 }
      }
    },
    "required": ["id", "customerId", "lines"],
    "check": {
      "with": { "total": { "$path": ["local", "focus", "total"] } },
      "expr": ["ge", { "$v": "total" }, 0]
    },
    "severity": "violation",
    "message": "an order must reference a real customer and have a non-negative total"
  }
}
```

- `kind` extends Jaxson's `type`: `object` `array` `string` `number` `boolean` `null`
  `any` — plus **`reference`** (new; must name an `index`) and **`node`** (new; a
  shape that constrains structure without asserting JSON's primitive type, used for
  combinator-only shapes — see below).
- `closed` + `ignoredProperties` extends Jaxson's `extra: "reject"|"allow"` with a
  named exclusion list, matching `sh:closed` + `sh:ignoredProperties` exactly: closed
  objects are the default posture (per Jaxson's own stance), and `ignoredProperties`
  is the escape hatch for metadata fields no field rule should have to name.
- `fields` values are shapes inline or `{"shape": "Name"}` by reference — this is how
  recursion enters (a `Category` shape whose `children` field references `Category`
  itself; bounded by `maxDepth`, section 6).
- `qualified` on an array or object field is `sh:qualifiedValueShape` +
  `qualifiedMinCount`/`qualifiedMaxCount` folded into one field: "between `min` and
  `max` of this collection's elements match `shape`." Counting runs over the existing
  finite snapshot, so it costs nothing new in termination terms.
- `check` is the constraint SHACL would reach for `sh:sparql` or `sh:js` to express.
  It is a `$compute` island: `with` binds named operands (the focus node is always
  available as `local.focus`, without needing to be listed), `expr` must reduce to a
  boolean. Because it's `$compute`, it is closed and step-bounded by construction —
  there is no analogue of an unbounded SPARQL query here.
- `severity` is `violation` (default) | `warning` | `info`. Only `violation` affects
  `conforms` in the validation report (section 7); `warning`/`info` are always
  collected, never gate.
- Logical combinators, sibling to `fields`:

| Combinator | Meaning |
|---|---|
| `and: [shape, ...]` | focus node must satisfy every listed shape |
| `or: [shape, ...]` | at least one |
| `xone: [shape, ...]` | exactly one |
| `not: shape` | must not satisfy it |

  A shape using only combinators and no `kind`/`fields` is a `node`-kind shape — pure
  composition, the way SHACL lets a shape be nothing but `sh:and`/`sh:or` over other
  shapes.
- `extends: "Name"` (singular — see section 12 on why multiple inheritance is
  deliberately excluded from v1) merges another shape's `fields`/`required`/`and` etc.
  into this one. Merge rule: `closed` is AND'd (most restrictive wins) unless this
  shape sets `"override": true`; a field name collision between the two is a
  `SHAPE_ERROR` unless this shape's field is explicitly marked `"override": true`.
  `extends` participates in the same `maxShapeDepth` budget as recursive `fields`
  references — an `extends` chain has to terminate too.

## 5. The `reference` node kind

```json
"customerId": { "kind": "reference", "index": "customersById" }
```

The value must be a scalar found as a key in the named index; if not,
`DANGLING_REFERENCE` (a new `EXECUTION_ERROR` code, section 9). This is the direct,
JSON-native replacement for the referential integrity SHACL gets for free from RDF
identity: instead of the runtime knowing what an IRI is, Shaxon requires you to say,
once, which array is the universe of valid values for this field — then every use of
that field is checked against it.

## 6. Path expressions

Jaxson's plain segment array (`["input", "items", 0, "price"]`) is still the base
case and still the only form allowed where Jaxson itself uses paths (`set`, `delete`,
etc.). Shapes and indices may additionally use:

| Form | Meaning |
|---|---|
| `{"$altPath": [pathA, pathB, ...]}` | the first alternative that resolves; `MISSING_PATH` only if none do |
| `{"$path*": innerStep, "maxDepth": n}` | zero or more repetitions of `innerStep`, bounded |
| `{"$path+": innerStep, "maxDepth": n}` | one or more repetitions, bounded |
| `{"$inverse": {"index": name, "key": operand}}` | every element the named `multi` index maps that key to |

- `maxDepth` is mandatory on both closure forms — its absence is a static
  `PROGRAM_ERROR`, exactly parallel to Jaxson's undeclared-`local` and
  undeclared-`$v` checks. It is not merely documentation: **exceeding it is itself a
  violation** (`SHAPE_DEPTH_EXCEEDED`, section 9), not a silent stop. A `Category`
  tree deeper than its declared bound fails to conform; it doesn't just stop being
  checked partway down. This is the deliberate design choice that lets recursive
  shapes exist at all inside a total language — SHACL's own zero-or-more paths have
  no such bound and can, in principle, walk an infinite or cyclic graph forever.
- `$inverse` is how "what points at me" is expressed without a graph edge: it reads
  the `multi` index built in section 3 rather than walking anything. It is only valid
  where an index build has already made the answer a lookup, never a search.

## 7. Targets and the `validate` list

```json
"validate": [
  { "target": { "$path": ["input"] }, "shape": "Order", "mode": "gate" },
  { "target": { "$each": ["input", "customers"] }, "shape": "Customer", "mode": "gate" },
  { "target": { "$discriminator": { "at": ["input", "lines"], "field": "kind", "value": "discount" } },
    "shape": "DiscountLine", "mode": "report", "into": ["state", "lineWarnings"] }
]
```

- `{"$path": [...]}` — one focus node.
- `{"$each": [...]}` — every element of the array/object at that path is its own
  focus node (SHACL's `sh:targetObjectsOf` analogue for a tree: there's no property
  to target objects *of*, so this targets a collection directly).
- `{"$discriminator": {...}}` — every element under `at` whose `field` equals `value`
  (the tree-native stand-in for `sh:targetClass`, since a JSON element has no
  `rdf:type`, only whatever discriminator field the schema author chose).
- `{"$indexed": name}` — every entry of a named index (useful when the interesting
  population is exactly what an index already enumerates).
- `mode: "gate"` — first violation aborts the pipeline with `VALIDATION_ERROR`
  (section 9); `program` does not run. `["input"]`/`["output"]` gate targets are the
  direct generalisation of Jaxson's `INPUT_ERROR`/`OUTPUT_ERROR`.
- `mode: "report"` — every violation for that target is collected, never aborts; the
  full report is written to the path named by `into`, or returned alongside `output`
  if `into` is omitted.
- Validation entries run in declared order; each entry's index rebuilds (section 3)
  happen first; focus nodes within a target are visited in array-index or
  code-point-sorted-key order — deterministic, matching Jaxson's `keys` rule.

## 8. The `check` instruction

A ninth instruction, sibling to `assert`, usable anywhere in `program`:

```json
{"op": "check", "target": {"$path": ["state", "board"]}, "shape": "BoardInvariant", "mode": "gate"}
```

This is what lets validation be a *callable operation* rather than only a pipeline
phase — the SHACL-flavoured half of the language, invoked mid-computation. A
turn-based program (a game, a workflow with intermediate states) can assert a shape
against `state` after every step, the same way `assert` checks a boolean today. Mode
semantics are identical to section 7; a `report`-mode `check` appends into the named
`into` path rather than replacing it, so a program can accumulate a running log of
non-fatal findings turn over turn.

## 9. Errors (extends Jaxson section 10)

| Category | Raised when | New codes |
|---|---|---|
| `SHAPE_ERROR` | malformed shape, index, `extends` cycle without a depth bound, missing `maxShapeDepth` when required | |
| `VALIDATION_ERROR` | a `gate`-mode target fails to conform | `SHAPE_MISMATCH` |
| `EXECUTION_ERROR` | (existing category, new codes) | `DANGLING_REFERENCE`, `SHAPE_DEPTH_EXCEEDED` |

`VALIDATION_ERROR` on `["input"]` or `["output"]` targets subsumes Jaxson's
`INPUT_ERROR`/`OUTPUT_ERROR` — a Shaxon runtime reports those under the new category
name; a compatibility shim can re-emit the old names for a package that declares
`jaxson` rather than `shaxon` as its version key.

Pipeline order: parse, version, schemas and shapes and indices (all static), program
validation, `input` schema/gate-shapes, execution (including any `check`
instructions), `output` schema/gate-shapes. Same "earlier stage wins" discipline as
Jaxson.

## 10. The validation report

For `report`-mode targets, the collected value has this shape:

```json
{
  "conforms": true,
  "violations": [
    {
      "focusPath": ["input", "lines", 2],
      "shape": "DiscountLine",
      "severity": "warning",
      "message": "a discount line should not exceed the order subtotal",
      "code": null
    }
  ]
}
```

- `conforms` reflects `violation`-severity findings only; `warning`/`info` entries
  never flip it, matching SHACL's own severity convention.
- `focusPath` is a plain Jaxson path — always resolvable, always addressable, same
  guarantee as every other path in the language.
- `code` is present (one of the `EXECUTION_ERROR` codes above) for structural findings
  like a dangling reference or a depth overrun, and `null` for an author-written
  `check`/combinator failure, whose only identity is its `message`.

## 11. What Shaxon deliberately does not take from SHACL

- **The RDF/triple data model**, blank nodes, and IRI identity — a Shaxon focus node
  is a JSON value at a path; identity is "same index key," not "same IRI."
- **Open-ended custom constraints** (`sh:sparql`, `sh:js`) — replaced entirely by
  `$compute`. SHACL itself already validates under a closed-world assumption, same as
  Jaxson; that convergence needed no work. What needed replacing was only the escape
  hatch SHACL leaves for arbitrary code.
- **Property paths with unbounded closure** — SHACL's `sh:path` zero-or-more has no
  required depth bound; Shaxon's does, unconditionally (section 6).

## 12. Where I am least sure

1. **Single inheritance only (`extends` takes one name, not a list).** Diamond
   conflicts under multiple inheritance need a resolution rule I don't have yet;
   restricting v1 to single inheritance sidesteps the question rather than answering
   it, and should be revisited once real shapes exist to test against.
2. **`$discriminator` as the sole stand-in for `sh:targetClass`.** It assumes every
   population worth targeting carries an explicit tag field. That's usually true for
   hand-authored JSON but not guaranteed, and there's no fallback target form for a
   population identified only by structural shape rather than a tag.
3. **Rebuilding every index before every `check` instruction** is the safe default,
   but if a hot loop calls `check` every iteration against a large `input.customers`
   array, that's a full index rebuild per iteration. Whether that needs a "reuse if
   the source hasn't changed since last build" optimisation, and whether that
   optimisation is even observable-safe under the no-aliasing rule, is open.
4. **`SHAPE_DEPTH_EXCEEDED` as a violation versus a hard `RESOURCE_ERROR`.** Treating
   it as a reportable violation (so `report`-mode targets keep going) is more useful
   than aborting the whole pipeline, but it means a badly-bounded `maxDepth` produces
   a very large violation list rather than a fast failure — might want a secondary
   cap on violation count per target.
5. **Message templating.** Static strings are the safe default; whether `message`
   should accept a `$tpl` for interpolating the focus value in is tempting for
   readability and untested for how much it complicates the no-side-effect story.

## 13. Lineage table

| From | Idea kept | Changed for Shaxon |
|---|---|---|
| Jaxson | roots, paths, operand forms, instructions, compute, exact decimals, step accounting | paths gain alternative/closure forms; one new instruction (`check`) |
| Jaxson | closed contract schemas (`type`, `fields`, `required`, `extra`) | kept verbatim as `inputSchema`/`outputSchema` for the simple case; superseded by `shapes` wherever SHACL-level expressiveness is needed |
| SHACL | node shapes, property constraints, `sh:closed`/`sh:ignoredProperties` | ported to JSON `kind`/`fields`/`closed`/`ignoredProperties` |
| SHACL | `sh:and`/`sh:or`/`sh:not`/`sh:xone` | ported verbatim as shape combinators |
| SHACL | `sh:qualifiedValueShape` + min/max count | folded into a field-level `qualified` clause |
| SHACL | severity levels, validation report shape | ported as `severity` + the `{conforms, violations}` report |
| SHACL | inverse/alternative property paths, referential reach via graph edges | recovered without graph edges via named, declared, rebuildable **indices** and `$inverse` |
| SHACL | `sh:sparql`/`sh:js` custom constraints | replaced by `$compute` — same expressive slot, closed and step-bounded instead of open |

## 14. What would need to exist before this is real

Jaxson earned its "checked, not just proposed" status from `jaxrun.go` plus 28
fixtures. Shaxon has neither yet. A reference interpreter would need, on top of
everything `jaxrun.go` already does:

- An index builder (section 3) run before validation and before every `check`.
- A shape evaluator: `kind`/`fields`/`closed`/`ignoredProperties`/combinators/
  `qualified`/`check`, producing either a single pass/fail (`gate`) or an accumulated
  violation list (`report`).
- A target resolver for `$path`/`$each`/`$discriminator`/`$indexed`.
- The path-expression evaluator for `$altPath`/`$path*`/`$path+`/`$inverse`, with the
  depth-overrun check wired to produce `SHAPE_DEPTH_EXCEEDED` rather than looping.
- A fixture format extended with an `expect` that can hold `{"report": {...}}` in
  addition to Jaxson's existing `{"output": ...}`/`{"error": {...}}`, so a
  `report`-mode fixture can assert the exact violation list, not just conformance.

Until that exists, this document is a proposal in the same sense Jaxson's was before
section 14 of that spec was written — worth building fixtures against before treating
any of the above as settled, particularly the four items in section 12.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0