# Shaxon v3.0 core: a deterministic, JSON-native constraint and validation layer over Jaxson

Status: proposal, unimplemented.

## 0. Stance

Shaxon is deterministic end to end. The same package, given the same input,
produces the same result — success, failure, and the exact number of steps
charged against `limits.steps` — on every conformant runtime, across
computation and validation alike. This is a single guarantee, not two: Shaxon
treats validation and computation as phases of one execution model rather
than as two systems bridged at an API boundary, and every rule in this
document that could otherwise leave a runtime a choice (index reuse, section
3; combinator evaluation, section 4) is written to remove that choice, so
that determinism holds by construction rather than by convention.

Jaxson computes; SHACL validates. Shaxon's constraint layer is derived from
SHACL's model — node shapes, property constraints, severity levels, qualified
counts, logical combinators — redesigned around a JSON tree instead of an RDF
graph, with every constraint, including the ones SHACL would reach for SPARQL
or a JS extension to express, written in the same total, step-bounded
sublanguage that runs the computation.

| Gap in SHACL for JSON | Why it's a gap | What Shaxon does |
|---|---|---|
| Data model is RDF triples | JSON must be lifted into a graph (JSON-LD or similar) before SHACL can see it, and document order/plain nesting is lost in the lift | Shaxon validates the JSON tree directly — Jaxson's roots and paths, unchanged |
| No computation | A conformance report only becomes useful once acted on, and acting on it means handing off to a separate engine with its own, separate guarantees | Validation and computation are phases of one package under one determinism guarantee; a `check` instruction lets a program call into shapes mid-run |
| Custom constraints are open (SPARQL / JS) | Such constraints can be arbitrarily expensive or fail to terminate; SHACL's own totality guarantee stops at the vocabulary's edge | Custom constraints are `$compute` islands — closed, step-bounded, no exception to the termination proof |
| Relational reach depends on real graph edges | SHACL's inverse paths and joins work because RDF has edges; a JSON tree has none | Shaxon adds declared, named **indices**, and the **`relations`** sugar over them, as explicit, built-once maps that stand in for the edges a tree doesn't have |

Everything not called out below carries over from Jaxson unchanged.

## 1. What carries over from Jaxson unchanged

- The four roots (`input`, `state`, `output`, `local`) and plain segment-array
  paths.
- Operand forms: `$lit`, `$path`, `$compute`, `$tpl`, and `$opt` inside templates.
- The eight core Jaxson instructions (`set`, `delete`, `append`, `insert`, `for`,
  `if`, `assert`, `halt`), plus one new instruction added in section 8 (`check`).
- The compute sublanguage: closed operator set, `with`-bound operands, no reach
  into state.
- Exact-decimal numbers, the 38-digit magnitude bound, and `div`/`round` as the
  only digit-losing operations.
- Step-counted termination as the semantic resource bound.
- The error-category discipline: stable categories, stable codes within
  `EXECUTION_ERROR`, fixed pipeline order.

A Shaxon runtime that also implements Jaxson can run a Jaxson package unmodified
— `shapes`, `validate`, `indices` and `relations` are all optional package
members with an empty default.

## 2. The package

```json
{
  "shaxon": "3.0",
  "limits": { "steps": 100000, "maxShapeDepth": 8 },
  "input": {},
  "inputSchema": {},
  "indices": {},
  "relations": {},
  "shapes": {},
  "validate": [],
  "program": [],
  "outputSchema": {}
}
```

- `shaxon` is mandatory, same failure discipline as Jaxson's `jaxson` key
  (`VERSION_ERROR` before anything else runs). A runtime implementing version
  `"3.0"` must also correctly execute a package declaring an earlier supported
  version (`"1.0"`, `"1.1"`, `"1.2"`, `"2.0"`), exactly as that version
  specifies. A runtime that does not implement `"3.0"` must reject a package
  declaring it with `VERSION_ERROR`.
- `limits.maxShapeDepth` is mandatory whenever any shape in the package can
  recurse — through a bounded closure path (section 6), through `extends`
  (section 4), or through a field that references a shape reachable from itself
  by named-shape references. Its absence in that situation is a `SHAPE_ERROR`,
  not a silent default — recursion is never accidentally unbounded.
- `inputSchema`/`outputSchema` keep Jaxson's original narrow contract vocabulary
  for the common case where a package only needs type-and-shape gating at the
  two edges. `shapes`/`validate` are for everything a plain schema can't say. A
  package may use either, both, or neither.
- `indices`, `relations` and `shapes` are named registries, read before
  `validate` runs.
- Duplicate keys anywhere, including inside `shapes`, `indices` or `relations`,
  are a `PARSE_ERROR`.
- A `relations` entry and an inline `reference` shorthand that would each build
  an index over the identical `(path, key)` pair are rejected at load — see
  section 4c.

## 3. Indices

A tree has no edges, so anything SHACL gets for free from RDF — "does this ID
exist elsewhere," "what points at me" — has to be declared explicitly. An index
is a named, deterministic map built from a source path and a key expression.

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

- `source` must resolve to an array; each element becomes `local.item` while
  `key` is evaluated against it, exactly like a `for` binding — snapshot
  iteration, no aliasing, same as everywhere else in Jaxson.
- An index key must resolve to a scalar JSON value. A `key` expression that
  resolves to a non-scalar value (an object or array) for any element of
  `source` is `TYPE_ERROR`. A `key` expression that fails to resolve at all for
  any element of `source` is `MISSING_PATH`, exactly as any other unresolved
  path produces; no element is silently skipped.
- Without `"multi": true`, a repeated key is a `SHAPE_ERROR` at build time (an
  index is a function unless you say otherwise). With `"multi": true`, the
  index maps a key to the *set* of matching elements — this is how an inverse
  relationship ("every order for this customer") is recovered without a graph
  edge.
- Indices are rebuilt from their declared source immediately before every use
  — by any `validate` entry and by every `check` instruction (section 8) —
  subject to the reuse rule below, so a `program` that mutates `state` between
  checks is always validated against current data, never a stale snapshot.
- A `reference` node kind (section 5), the `relations` sugar (section 4a), and
  an `$inverse` operand (section 6) are the places indices get consumed.

**Cost.** Every index (re)build costs one step per element visited in
`source`, charged by the same "one step per iteration begun" rule Jaxson
already applies to `for`. An index build is a snapshot iteration like any
other; it is not a hidden operation exempt from the step budget. The charge
lands at the point the build happens: before first use in `program`, before
each `validate` entry resolves its targets, and before each `check`
instruction — except where the reuse rule below applies.

**Reuse.** An index build is reused without recharge whenever no `set`/
`delete`/`append`/`insert` since that build targeted a path equal to, or a
prefix of, its declared `source`. Reuse is mandatory, not an optional
optimisation: whether a package succeeds within its declared `limits.steps`
must not depend on which conformant runtime executes it. Reuse is decidable
from the mutation-path log already required for the no-aliasing guarantee — it
is a lookup against that log, not a deep-equality scan.

## 4. Shapes

A shape is a named predicate over one JSON value (the *focus node*). The
vocabulary is deliberately at SHACL's level of expressiveness, not JSON
Schema's.

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

- `kind` extends Jaxson's `type`: `object` `array` `string` `number` `boolean`
  `null` `any` — plus **`reference`** (must name an `index` or a `relation`;
  section 5) and **`node`** (a shape that constrains structure without
  asserting JSON's primitive type, used for combinator-only shapes — see
  below).
- `closed` + `ignoredProperties` extends Jaxson's `extra: "reject"|"allow"`
  with a named exclusion list: closed objects are the default posture, and
  `ignoredProperties` is the escape hatch for metadata fields no field rule
  should have to name.
- `fields` values are shapes inline or `{"shape": "Name"}` by reference — this
  is how recursion enters (a `Category` shape whose `children` field
  references `Category` itself; bounded by `maxShapeDepth`, section 2).
- `qualified` on an array or object field is "between `min` and `max` of this
  collection's elements match `shape`." Counting runs over the existing finite
  snapshot, so it costs nothing new in termination terms.
- `check` is a `$compute` island: `with` binds named operands (the focus node
  is always available as `local.focus`, without needing to be listed), `expr`
  must reduce to a boolean. Because it's `$compute`, it is closed and
  step-bounded by construction.
- `severity` is `violation` (default) | `warning` | `info`. Only `violation`
  affects `conforms` in the validation report (section 10); `warning`/`info`
  are always collected, never gate.
- Logical combinators, sibling to `fields`:

| Combinator | Meaning |
|---|---|
| `and: [shape, ...]` | focus node must satisfy every listed shape |
| `or: [shape, ...]` | at least one |
| `xone: [shape, ...]` | exactly one |
| `not: shape` | must not satisfy it |

  A shape using only combinators and no `kind`/`fields` is a `node`-kind shape
  — pure composition.
- `extends: "Name"` (single inheritance) merges another shape's members into
  this one. `extends` participates in the same `maxShapeDepth` budget as
  recursive `fields` references.

**The `extends` merge rule.** Every `extends`-mergeable member accumulates
using whatever aggregation is natural for that member, unless the child marks
that member `"override": true`, which replaces instead of accumulating.

| Member | Aggregation | Override behaviour |
|---|---|---|
| `closed` | AND — most restrictive of parent/child wins | `"override": true` makes the child's own value win outright |
| `fields` | union; a name collision is `SHAPE_ERROR` | per-field `"override": true` replaces the parent's field definition |
| `required` | union — required in either parent or child means required in the merged shape | none needed; union only ever gets stricter |
| `and` | concatenate parent's list then child's | `"override": true` on the child's `and` replaces the parent's list instead of concatenating |
| `or` | concatenate parent's list then child's, then apply "at least one" over the combined pool | `"override": true` replaces instead of pooling |
| `xone` | concatenate parent's list then child's, then apply "exactly one" over the combined pool | `"override": true` replaces instead of pooling |
| `not` | parent's and child's `not` targets both apply — focus must satisfy neither | `"override": true` on the child's `not` replaces the parent's instead of adding to it |
| `check` | parent's and child's `check` are ANDed — focus must pass both | `"override": {"with": ..., "expr": ...}` on the child replaces the parent's `check` instead of ANDing |

Pooling `xone` can turn a focus node that satisfied the child's own list into a
violation, because "exactly one" is evaluated over the merged pool, not the
child's list alone. A focus node matching exactly one of the child's own
alternatives, plus one more of the parent's, now matches two — and fails.

**Combinator evaluation is mandatorily minimal.** Alternatives in `and`/`or`/
`xone` are evaluated in listed order (parent's list before child's, per the
merge rule above) and evaluation stops as soon as the result is determined:
`and` stops at the first failing alternative, `or` stops at the first
passing alternative, `xone` stops as soon as a second match is found. This is
mandatory, not an optional optimisation, for the same reason index-build
reuse (section 3) is mandatory: the number of alternatives a conformant
runtime evaluates, and therefore the number of steps it charges, must not
depend on which runtime executes the package. `not` has a single inner shape
and needs no such rule.

## 4a. Relations

A relation is a named, bidirectional relational declaration that desugars to
the index/reference/`$inverse` machinery of section 3.

```json
"relations": {
  "orderCustomer": {
    "from": { "path": ["input", "orders"], "field": "customerId" },
    "to":   { "path": ["input", "customers"], "key": "id" },
    "cardinality": "one-to-many"
  }
}
```

- `to` is the unique side: the runtime builds an index over `to.path` keyed by
  `to.key` — equivalent to `{"source": to.path, "key": {"$path": ["local",
  "item", to.key]}}`. A repeated `to.key` value is a `SHAPE_ERROR` at build
  time, same as any non-`multi` index.
- `from` is the many side: the runtime builds a `multi` index over `from.path`
  keyed by `from.field`, used only to answer `$inverse` — equivalent to
  `{"source": from.path, "key": {"$path": ["local", "item", from.field]},
  "multi": true}`.
- `cardinality: "one-to-many"` (default) leaves `from.field` free to repeat.
  `cardinality: "one-to-one"` additionally requires uniqueness on
  `from.field`, checked at the same build step as the `to.key` uniqueness
  check; a violation is `SHAPE_ERROR`. `one-to-one` means each source value
  identifies at most one source element and each target key identifies at
  most one target element — it does not require that every `to` element be
  referenced.
- Both indices a `relations` entry produces are anonymous: not visible under
  `indices`, not nameable in a bare `{"kind": "reference", "index": ...}`, and
  not checked for name collision against `indices`. A `reference` field
  consumes a relation with `{"kind": "reference", "relation": "orderCustomer"}`
  — always against the `to` side. `$inverse` consumes it with `{"$inverse":
  {"relation": "orderCustomer", "key": operand}}` — always the `from` side.
- A malformed `relations` entry (missing `from`/`to`, an invalid
  `cardinality` value, a `field`/`key` that isn't a plain field name) is
  `SHAPE_ERROR`.

**Inline reference shorthand**, for the common case where only the forward
direction is ever needed:

```json
"customerId": { "kind": "reference", "of": { "$path": ["input", "customers"] }, "by": ["id"] }
```

Desugars to an anonymous index scoped to this one field — equivalent to
`{"source": of, "key": {"$path": ["local", "item"].concat(by)}}` — internal to
the shape/field pair that declared it. It is not nameable, not visible under
`indices`, and has no `$inverse` form.

## 4b. Modeling a tree

| Data shape | Use |
|---|---|
| Genuinely nested — a child is a literal descendant in the JSON (`children: [...]` embedded under its parent) | `$path+`/`$path*` closure (section 6) |
| Flat — one array of same-shaped elements related by an id field (an adjacency list: `{"id": 2, "parentId": 1}`) | `relations` with `from.path == to.path` |

`$path+` has no way to reach an ID-linked parent in a flat array — its
`innerStep` is a relative *path*, and there is no structural path from a node
to a same-array sibling without going through an index. A genuinely nested
tree needs no index at all — you just walk it.

For a dataset denormalised both ways at once (nested *and* carrying redundant
parent-id fields), neither mechanism is canonical over the other: use
whichever one `program` actually navigates by, and add a `check` asserting
that every node reachable via the `$path+` closure from a root is also
reachable via the `relations`-based `$inverse` chain, or vice versa.

## 4c. Redundant declarations

Two declarations that would each build an index over the identical
`(path, key)` pair are a `SHAPE_ERROR` at package load:

- An inline `{"kind": "reference", "of": ..., "by": [...]}` shorthand whose
  normalised `(of, by)` matches the `to` side of a `relations` entry — write
  `{"kind": "reference", "relation": name}` instead.
- Two `relations` entries whose `to` sides normalise to the same
  `(path, key)` — point every consuming field at one of them.

This is pure static comparison of normalised path segments and key field
names, no data inspection needed. It is not extended to a standalone named
`indices` entry that happens to match a relation's derived index — that is
the general escape hatch, and an author reaching for it explicitly is making
a considered choice the shorthand-collision case isn't.

## 5. The `reference` node kind

```json
"customerId": { "kind": "reference", "index": "customersById" }
```

The value must be a scalar. A non-scalar value (an object or array) is
`TYPE_ERROR`. A scalar value not found as a key in the named index (or the
`to` side of a named relation) is `DANGLING_REFERENCE`, an `EXECUTION_ERROR`
code (section 9).

`reference` composes under the existing grammar exactly the way any other
item shape does — no new syntax:

```json
"tagIds": {
  "kind": "array",
  "items": { "kind": "reference", "index": "tagsById" }
}
```

**Reporting rule**, extending the array-iteration convention already used
everywhere (`for`, `$each`, index builds all snapshot-iterate in array order):

- `report` mode: every dangling element gets its **own** violation with its
  own `focusPath` (e.g. `["input", "order", "tagIds", 2]` for the third
  element) — three bad entries in one field produce three violations, not one
  aggregate finding.
- `gate` mode: the **first** dangling element in array-index order raises
  `DANGLING_REFERENCE` and aborts.
- The violation's `shape` field names the nearest enclosing **named** shape
  being checked (the one passed to `check`/`validate`), not the inline,
  unnamed `reference` item.

## 6. Path expressions

Jaxson's plain segment array (`["input", "items", 0, "price"]`) is still the
base case and still the only form allowed where Jaxson itself uses paths
(`set`, `delete`, etc.). Shapes and indices may additionally use:

| Form | Meaning |
|---|---|
| `{"$altPath": [pathA, pathB, ...]}` | the first alternative that resolves; `MISSING_PATH` only if none do |
| `{"$path*": innerStep, "maxDepth": n}` | zero or more repetitions of `innerStep`, bounded |
| `{"$path+": innerStep, "maxDepth": n}` | one or more repetitions, bounded |
| `{"$inverse": {"index": name, "key": operand}}` or `{"$inverse": {"relation": name, "key": operand}}` | every element the named `multi` index (or the `from` side of the named relation) maps that key to |

- **`innerStep` is a relative path** — the same plain segment-array grammar
  as an ordinary path, applied relative to the current position rather than
  from a root. `{"$path+": ["children"], "maxDepth": 5}` means "repeatedly
  append `.children`." A multi-segment step (`{"$path+": ["next", "value"],
  ...}`) is a single hop that happens to cross more than one field.
- **The result of a closure is an ordered, duplicate-free sequence of
  paths**, not an unordered set. Paths are visited breadth-first by hop count
  — the base (for `$path*`), then every one-hop extension, then every
  two-hop extension, and so on up to `maxDepth` — with ties at the same hop
  count broken by the existing array-index/code-point-sorted-key ordering
  used everywhere else paths are visited. A path reachable by more than one
  route is included once, at the position of its first visit; identity is
  path-segment equality. From base path `B`: `$path*` visits `B` then every
  reachable extension of it up to `maxDepth` hops, stopping the chain — not
  erroring — the first time a hop doesn't resolve. `$path+` is the same
  sequence minus `B`; it is a failure only if zero hops resolve at all.
- **As a target (section 7)**, a closure needs an explicit base, since there
  is no ambient focus yet: `{"$path*": step, "from": basePath, "maxDepth":
  n}` (`from` is omitted when the same form appears embedded inside a
  shape's `check`, an index's `key`, or any other operand position that
  already has an ambient `local.focus` or `local.item` — there, the ambient
  position is the base).
- **`local.step`** is available inside `innerStep` for a step whose direction
  depends on the node's own data rather than a fixed field name, bound to the
  current traversal position, parallel to `local.item` in `for` and in index
  builds.
- `maxDepth` is mandatory on both closure forms — its absence is a static
  `PROGRAM_ERROR`. Exceeding it while the chain still resolves further raises
  `PATH_DEPTH_EXCEEDED` (section 9) — a violation, not a silent stop, and
  distinct from the shape-recursion `SHAPE_DEPTH_EXCEEDED`.
- `$inverse` reads a `multi` index (section 3), directly or via a `relations`
  entry's `from` side (section 4a), rather than walking anything. It is only
  valid where an index build has already made the answer a lookup, never a
  search.

## 7. Targets and the `validate` list

```json
"validate": [
  { "target": { "$path": ["input"] }, "shape": "Order", "mode": "gate" },
  { "target": { "$each": ["input", "customers"] }, "shape": "Customer", "mode": "gate" },
  { "target": { "$discriminator": { "at": ["input", "lines"], "field": "kind", "value": "discount" } },
    "shape": "DiscountLine", "mode": "report", "into": ["state", "lineWarnings"] },
  { "target": { "$path*": ["next"], "from": ["input", "list", "head"], "maxDepth": 10 },
    "shape": "ListNode", "mode": "report" }
]
```

- `{"$path": [...]}` — one focus node.
- `{"$each": [...]}` — every element of the array/object at that path is its
  own focus node.
- `{"$discriminator": {...}}` — every element under `at` whose `field` equals
  `value`.
- `{"$indexed": name}` — every entry of a named index.
- `{"$path*": step, "from": basePath, "maxDepth": n}` / `{"$path+": ...}` —
  every path in the closure (section 6) is its own focus node.
- `mode: "gate"` — first violation aborts the pipeline with
  `VALIDATION_ERROR` (section 9); `program` does not run.
  `["input"]`/`["output"]` gate targets are the direct generalisation of
  Jaxson's `INPUT_ERROR`/`OUTPUT_ERROR`.
- `mode: "report"` — every violation for that target is collected, never
  aborts; the report is written to the path named by `into`, or returned
  alongside `output` if `into` is omitted.
- Validation entries run in declared order; each entry's index rebuilds
  (section 3) happen first; focus nodes within a target are visited in
  array-index or code-point-sorted-key order.
- When more than one `report`-mode target or `check` instruction omits
  `into`, their violations are concatenated into a single combined report, in
  the order the corresponding entries execute; the combined report's
  `conforms` follows the ordinary rule of section 10 applied to the
  concatenated violation list.

## 8. The `check` instruction

A ninth instruction, sibling to `assert`, usable anywhere in `program`:

```json
{"op": "check", "target": {"$path": ["state", "board"]}, "shape": "BoardInvariant", "mode": "gate"}
```

This lets validation be a callable operation rather than only a pipeline
phase: a turn-based program (a game, a workflow with intermediate states) can
assert a shape against `state` after every step, the same way `assert` checks
a boolean today. Mode semantics are identical to section 7; a `report`-mode
`check` appends into the named `into` path rather than replacing it, so a
program can accumulate a running log of non-fatal findings turn over turn.

## 9. Errors

| Category | Raised when | Codes |
|---|---|---|
| `SHAPE_ERROR` | malformed shape, index or `relations` entry; `extends` cycle without a depth bound; missing `maxShapeDepth` when required; a repeated key in a non-`multi` index or the `to` side of a relation; `cardinality: "one-to-one"` uniqueness violation on `from.field`; a redundant relation/shorthand pair (section 4c) | |
| `VALIDATION_ERROR` | a `gate`-mode target fails to conform | `SHAPE_MISMATCH` |
| `EXECUTION_ERROR` | (existing Jaxson category, extended) | `DANGLING_REFERENCE`, `SHAPE_DEPTH_EXCEEDED`, `PATH_DEPTH_EXCEEDED`, plus Jaxson's own `TYPE_ERROR`/`MISSING_PATH` where index keys and `reference` values resolve to the wrong shape |

`VALIDATION_ERROR` on `["input"]` or `["output"]` targets subsumes Jaxson's
`INPUT_ERROR`/`OUTPUT_ERROR`. A compatibility shim can re-emit the old names
for a package that declares `jaxson` rather than `shaxon` as its version key.

Pipeline order: parse, version, schemas and shapes and indices and relations
(all static), program validation, `input` schema/gate-shapes, execution
(including any `check` instructions), `output` schema/gate-shapes. Earlier
stage failure means later stages do not run.

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

- `conforms` reflects `violation`-severity findings only; `warning`/`info`
  entries never flip it.
- `focusPath` is a plain Jaxson path — always resolvable, always addressable.
- `code` is present (one of the `EXECUTION_ERROR` codes above) for structural
  findings like a dangling reference or a depth overrun, and `null` for an
  author-written `check`/combinator failure, whose only identity is its
  `message`.

**Fixed messages for structural findings.** A structural finding's `message`
is fixed by the runtime, not author-configurable:

| Code | Fixed message |
|---|---|
| `DANGLING_REFERENCE` | `"referenced value is not present in the named index"` |
| `SHAPE_DEPTH_EXCEEDED` | `"shape recursion exceeded maxShapeDepth"` |
| `PATH_DEPTH_EXCEEDED` | `"path closure exceeded maxDepth"` |

## 11. What Shaxon does not take from SHACL

- **The RDF/triple data model**, blank nodes, and IRI identity — a Shaxon
  focus node is a JSON value at a path; identity is "same index key," not
  "same IRI."
- **Open-ended custom constraints** (`sh:sparql`, `sh:js`) — replaced entirely
  by `$compute`.
- **Property paths with unbounded closure** — SHACL's `sh:path` zero-or-more
  has no required depth bound; Shaxon's does, unconditionally (section 6).

## 12. Open questions

1. Whether `extends` should ever take more than one parent name, and if so,
   what resolves a field or combinator conflict between two parents.
2. Whether a population worth targeting can be identified only by structural
   shape rather than by an explicit discriminator field — `$discriminator`
   is currently the only population-targeting mechanism for tagged data, and
   there is no fallback for untagged populations.
3. Whether exceeding `maxShapeDepth`/`maxDepth` is best understood as a
   violation of the data (the document doesn't conform) or as a resource
   ceiling on the validator (the validator wasn't asked to look further) —
   and, relatedly, whether it should always be reportable, as currently
   specified, or configurably fatal.
4. Whether `message` on a shape/`check` may contain a `$tpl` for
   interpolating focus-node data, versus staying a static string as
   currently specified.
5. The precise type and depth-accounting semantics of `local.step` inside a
   dynamic path-closure step.
6. Whether the elements `$inverse` yields are JSON values or focus paths;
   this matters where the result needs to compose with other
   path-expression forms.

## 13. Implementation status

No reference interpreter exists for this specification. A conformant
implementation requires, at minimum: an index builder, including the
mandatory reuse rule of section 3; a `relations` desugarer (section 4a) with
the redundant-declaration check of section 4c; a shape evaluator covering
`kind`/`fields`/`closed`/`ignoredProperties`/combinators/`qualified`/`check`/
`extends` (section 4); a target resolver for `$path`/`$each`/
`$discriminator`/`$indexed` and closure targets (section 7); a
path-expression evaluator for `$altPath`/`$path*`/`$path+`/`$inverse`
(section 6); and a fixture format whose `expect` member can hold
`{"report": {...}}` in addition to `{"output": ...}`/`{"error": {...}}`.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
