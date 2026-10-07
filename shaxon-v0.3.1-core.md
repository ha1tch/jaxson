# Shaxon v3.1 core: a deterministic, JSON-native constraint and validation layer over Jaxson

Status: proposal. Implemented in Go (`src/jaxson-shaxon-v0.3.1`) and checked against 196 conformance fixtures; see `docs/IMPLEMENTATION-STATUS.md`.

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

**The four invariants.** Everything in this document serves one of four
organising principles:

- *Determinism* — same package, same input, same result, same step count,
  on every conformant runtime (above), enforced by mandatory index reuse
  (section 3) and mandatory minimal combinator evaluation (section 4).
- *Boundedness* — no operation may secretly escape the step or depth model
  (section 11; Jaxson's own termination discipline): every closure has a
  `maxDepth`, every recursive shape has a `maxShapeDepth`, every loop
  iterates a finite snapshot.
- *Explicit reach* — all relational reach goes through a declared `index`
  or `relation`; nothing is discoverable at query time that wasn't declared
  at load time (section 3, section 4a, section 11).
- *Validation purity* — `check` and `validate` never write to `state`,
  `output`, or any root, under any circumstance (section 1).

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

**Validation is a pure predicate.** `check` and `validate` never write to
`state`, `output`, or any root, under any circumstance. Any data a
validation needs must be derived by an explicit, ordinary `program` step
that runs before the `check` that reads it. This is what keeps "program
state transition" and "predicate evaluation" from blurring into each other.

A Shaxon runtime that also implements Jaxson can run a Jaxson package unmodified
— `shapes`, `validate`, `indices` and `relations` are all optional package
members with an empty default.

## 1a. Core semantic objects

- **Path** — a JSON array of segments; the base addressing unit (Jaxson's
  own path grammar).
- **FocusPath** — a Path a target has resolved as the subject of one shape
  check; the canonical identity of a finding (section 7, section 10).
- **TargetResult** — the ordered, duplicate-free sequence of FocusPaths a
  `target` expression produces before any shape runs (section 7).
- **ShapeResult** — the pass/fail (plus, on failure, severity/message/
  constraintId) from checking one shape against one FocusPath (section 4).
- **Violation** — one report entry: FocusPath, optional ConstraintPath,
  `kind`, `shape`, optional `constraintId`, `severity`, `message`, `code`
  (section 10).
- **Index** — a named, rebuildable map from a source array to scalar
  key(s), consumed by `reference`, `relations`, `$inverse`, `$indexed`
  (section 3).

Relationship, in one line: a `target` produces a `TargetResult`; each
`FocusPath` in it is checked, producing a `ShapeResult`; each failing
`ShapeResult` — or failing field/`required` member within it, section 10 —
produces one `Violation`. `Index` feeds the `reference`/`$inverse`/
`$indexed` machinery that some FocusPaths and Targets are built from.

## 2. The package

```json
{
  "shaxon": "3.1",
  "limits": { "steps": 100000, "maxShapeDepth": 8 },
  "input": {},
  "inputSchema": {},
  "indices": {},
  "relations": {},
  "shapes": {},
  "computes": {},
  "validate": [],
  "program": [],
  "outputSchema": {}
}
```

- `shaxon` is mandatory, same failure discipline as Jaxson's `jaxson` key
  (`VERSION_ERROR` before anything else runs). A runtime implementing version
  `"3.1"` must also correctly execute a package declaring an earlier supported
  version (`"1.0"`, `"1.1"`, `"1.2"`, `"2.0"`, `"3.0"`), exactly as that
  version specifies. A runtime that does not implement `"3.1"` must reject a
  package declaring it with `VERSION_ERROR`. The `shaxon` version string is
  the protocol's own version lineage; it is independent of any specification
  document's own filename revision, which has no bearing on what a package
  must declare.
- `limits.maxShapeDepth` is mandatory whenever any shape in the package can
  recurse — through a bounded closure path (section 6), through `extends`
  (section 4), or through a field that references a shape reachable from itself
  by named-shape references. Its absence in that situation is a `SHAPE_ERROR`,
  not a silent default — recursion is never accidentally unbounded.
  `maxShapeDepth` and `maxDepth` (section 6) are **validation horizons**: a
  declared boundary past which the language does not promise to look, not a
  claim that no further structure exists past it. `maxShapeDepth` counts
  named-shape descents only: the root shape is depth 1 and each `{"shape": N}`
  followed is one deeper; an inline structured field is not a descent. Going
  past the bound is one structural `SHAPE_DEPTH_EXCEEDED` finding at that
  node, whose subtree is not visited; in `gate` mode it raises `EXECUTION_ERROR`
  with that code.
- `inputSchema`/`outputSchema` keep Jaxson's original narrow contract vocabulary
  for the common case where a package only needs type-and-shape gating at the
  two edges. `shapes`/`validate` are for everything a plain schema can't say. A
  package may use either, both, or neither.
- `indices`, `relations`, `shapes` and `computes` are named registries, read
  before `validate` runs.
- Duplicate keys anywhere, including inside `shapes`, `indices`, `relations`
  or `computes`, are a `PARSE_ERROR`.
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
`delete`/`append`/`insert` since that build targeted a path that overlaps
what the build reads. Reuse is mandatory, not an optional
optimisation: whether a package succeeds within its declared `limits.steps`
must not depend on which conformant runtime executes it. Reuse is decidable
from the mutation-path log already required for the no-aliasing guarantee — it
is a lookup against that log, not a deep-equality scan.

A build reads its `source` path and every non-`local` path named by a `$path`
operand in its `source` or `key` (a key that reads `state.rate` depends on
`state.rate`); a path with a computed segment counts as the whole subtree up to
that segment. Two paths **overlap** when one is equal to, or a prefix of, the
other, so a write below the source (`state.customers[3].id` for the source
`state.customers`) invalidates the build, as does a write above it. Reading
the rule as "a prefix of the source" alone would serve a stale index after
such a write, against this section's own rule that a program is always
validated against current data.

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
    "requiredIds": { "customerId": "ORDER_NEEDS_CUSTOMER" },
    "check": {
      "with": { "total": { "$path": ["local", "focus", "total"] } },
      "expr": ["ge", { "$v": "total" }, 0],
      "id": "ORDER_TOTAL_NONNEGATIVE"
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
  collection's elements match `shape`." Counting terminates over the existing
  finite snapshot — that is what "costs nothing new" refers to. It is not a
  step-cost exemption: each element checked against `qualified`'s `shape` is
  charged the ordinary shape-evaluation step cost, the same as if each had
  been checked individually. **One step per shape activation**: a named
  shape, a combinator alternative, a `qualified` element, an `items` element,
  or an inline field that has structure. Leaf primitive checks cost nothing.
- `check` is a `$compute` island: `with` binds named operands (the focus node
  is always available as `local.focus`, without needing to be listed), `expr`
  must reduce to a boolean, and an optional `"id"` gives it a stable
  `constraintId` (section 10). Because it's `$compute`, it is closed and
  step-bounded by construction. A `check` may reference a named, reusable
  computation instead of writing `with`/`expr` inline — section 4d.
- `severity` is `violation` (default) | `warning` | `info`. Only `violation`
  affects `conforms` in the validation report (section 10); `warning`/`info`
  are always collected, never gate.
- `enum` is legal on `string`, `number`, `boolean`, and `null` kinds, with the
  same "must be one of these exact values" semantics regardless of kind.
  Deliberately not extended to `object`/`array` — deep-equality-against-a-list
  there is treated as a different feature.
- `required` names members that must be present; `requiredIds` is an optional
  sibling map from a subset of `required`'s entries to a stable
  `constraintId` (section 10) for that member's absence finding. A
  `requiredIds` key naming a member not present in `required` is
  `SHAPE_ERROR`. `required` itself stays a plain array of strings.
- Logical combinators, sibling to `fields`:

| Combinator | Meaning |
|---|---|
| `and: [shape, ...]` | focus node must satisfy every listed shape |
| `or: [shape, ...]` | at least one |
| `xone: [shape, ...]` | exactly one |
| `not: shape` | must not satisfy it |

  A shape using only combinators and no `kind`/`fields` is a `node`-kind shape
  — pure composition.
- `extends: "Name"` merges another shape's members into this one.
  `extends` takes a single parent name by design: composing several shapes
  without merging their fields is `and: [...]`; `extends` is for when
  field-level merging into one flattened object schema is the actual goal.
  `extends` participates in the same `maxShapeDepth` budget as recursive
  `fields` references.

**The `extends` merge rule.** Every `extends`-mergeable member accumulates
using whatever aggregation is natural for that member, unless the child marks
that member `"override": true`, which replaces instead of accumulating.

| Member | Aggregation | Override behaviour |
|---|---|---|
| `closed` | AND — most restrictive of parent/child wins | `"override": true` makes the child's own value win outright |
| `fields` | union; a name collision is `SHAPE_ERROR` | per-field `"override": true` replaces the parent's field definition |
| `required` | union — required in either parent or child means required in the merged shape | none needed; union only ever gets stricter |
| `requiredIds` | union by key | per-key `"override": true` replaces the parent's id for that key |
| `and` | concatenate parent's list then child's | `"override": true` on the child's `and` replaces the parent's list instead of concatenating |
| `or` | concatenate parent's list then child's, then apply "at least one" over the combined pool | `"override": true` replaces instead of pooling |
| `xone` | concatenate parent's list then child's, then apply "exactly one" over the combined pool | `"override": true` replaces instead of pooling |
| `not` | parent's and child's `not` targets both apply — focus must satisfy neither | `"override": true` on the child's `not` replaces the parent's instead of adding to it |
| `check` | parent's and child's `check` are ANDed — focus must pass both | `"override": {"with": ..., "expr": ...}` on the child replaces the parent's `check` instead of ANDing |
| `kind` | the child's if the parent's is unset, `node` or `any` (all three accept every value); otherwise the child repeats the parent's or omits it, and a different kind is `SHAPE_ERROR` | none; a child cannot change a kind |
| `index`, `relation`, `of`/`by` (a `reference` shape's target) | inherited with the kind; a child that names a different target is `SHAPE_ERROR` | none; a child cannot retarget a reference |
| `items` | the child's if the parent has none; a child that states the same items as the parent's is accepted; different items are `SHAPE_ERROR` | none; a child cannot replace the parent's `items` |
| `qualified` | the child's if the parent has none; the same rule restated is accepted; a different rule is `SHAPE_ERROR` | none; a child cannot replace the parent's `qualified` |
| Primitive keywords (`min`, `max`, `int`, `minLen`, `maxLen`, `enum`, `minItems`, `maxItems`) | both apply: the larger of two minimums (`min`, `minLen`, `minItems`), the smaller of two maximums, `int` if either asks, and the intersection of two enums; a keyword only one shape states stands as stated. The merged shape is then validated like any other, so a merged minimum above its maximum is `SHAPE_ERROR` | none needed; a child only ever narrows |
| `severity`, `message` | the child's own only; the parent's are **not** carried | restate them in the child |

A shape that extends another therefore never accepts a value the parent would
refuse by `kind`, a reference target, a primitive keyword, `items` or
`qualified`: the child can narrow these and cannot loosen them. The other
members follow their own rows. Pooling `or` alternatives, adding to
`ignoredProperties`, and every `override` can make the merged shape accept more
than its parent does.

Two `items`, `qualified` rules or reference targets are the same when they are
equal once parsed, so a field written as `{"shape": "X"}` and an inline copy of
shape `X` are different, and a child that restates the parent's rule in the
other form is `SHAPE_ERROR`. Two enums whose intersection is empty are not an
error: the merged shape accepts no value of its kind.

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

Because evaluation stops as soon as the combinator's result is determined,
alternatives never evaluated produce no findings of their own: a failing
`and`/`or`/`xone` always produces exactly **one** violation for the
enclosing shape (section 10), never one per failing alternative and never
one per unevaluated alternative — there is nothing further to report about
an alternative that genuinely was never checked.

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

## 4d. Named compute reuse

```json
"computes": {
  "nonNegativeRoundedTotal": {
    "with": { "x": { "$path": ["local", "focus", "total"] } },
    "expr": ["ge", ["round", { "$v": "x" }, 2], 0]
  }
},
"shapes": {
  "Order": {
    "check": { "compute": "nonNegativeRoundedTotal" }
  }
}
```

A package-level `computes` registry, parallel to `indices`/`relations`/
`shapes`. A `check` may reference one by name (`{"compute": name}`) instead
of writing `with`/`expr` inline. A named compute is semantically identical
to an inline `$compute` — same closed operator set, same `with`-binding
rule, same step cost, no reach into `state`; this is authoring convenience
only, never a new capability. A `computes` entry that would be malformed as
an inline `$compute` (unknown operator, wrong arity, undeclared `$v`, etc.)
is `SHAPE_ERROR` at load, same as a malformed inline one is at the point
it's parsed.

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
| `{"$inverse": {"index": name, "key": operand}}` or `{"$inverse": {"relation": name, "key": operand}}` | every path the named `multi` index (or the `from` side of the named relation) maps that key to, in the index's own iteration order (section 3) |

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
  builds. `local.step` must resolve to a single path segment — a string or a
  non-negative integer, never a path array, an object, or any other
  non-scalar value; resolving to anything else is `TYPE_ERROR`. Using
  `local.step` counts as exactly one hop toward `maxDepth`, the same as a
  fixed-name segment would.
- `maxDepth` is mandatory on both closure forms — its absence is a static
  `PROGRAM_ERROR`. `maxDepth` is a **validation horizon** (section 2): a
  declared boundary past which the language does not promise to look.
  Exceeding it while the chain still resolves further raises
  `PATH_DEPTH_EXCEEDED` (section 9) — a violation, not a silent stop, and
  distinct from the shape-recursion `SHAPE_DEPTH_EXCEEDED` — reportable
  precisely because it means the document asked for more traversal than the
  package declared itself willing to perform. The finding is attached at the
  last node the closure took.
- `$inverse` reads a `multi` index (section 3), directly or via a `relations`
  entry's `from` side (section 4a), rather than walking anything. It yields
  paths, each composable with any other path-expression form exactly as a
  closure result is. It is only valid where an index build has already made
  the answer a lookup, never a search.

## 7. Targets and the `validate` list

```json
"validate": [
  { "target": { "$path": ["input"] }, "shape": "Order", "mode": "gate" },
  { "target": { "$each": ["input", "customers"] }, "shape": "Customer", "mode": "gate" },
  { "target": { "$discriminator": { "at": ["input", "lines"], "field": "kind", "value": "discount" } },
    "shape": "DiscountLine", "mode": "report", "into": ["state", "lineWarnings"] },
  { "target": { "$path*": ["next"], "from": ["input", "list", "head"], "maxDepth": 10 },
    "shape": "ListNode", "mode": "report" },
  { "target": { "$each": ["input", "lines"] },
    "unique": { "field": "sku", "id": "LINE_SKU_UNIQUE" },
    "severity": "warning", "message": "duplicate sku in order lines", "mode": "report" }
]
```

- `{"$path": [...]}` — one focus node.
- `{"$each": [...]}` — every element of the array/object at that path is its
  own focus node.
- `{"$discriminator": {...}}` — every element under `at` whose `field` equals
  `value`.
- `{"$indexed": name}` — every source element's path underlying the named
  index becomes its own focus node — never the index's key or value. For a
  non-`multi` index, elements are visited in ascending key order (section 3's
  sorted-key convention). For a `multi` index, elements are visited grouped
  by key in ascending key order, then in array-index order within each key's
  matching set. Every source element belongs to exactly one key, so no
  element is visited twice.
- `{"$path*": step, "from": basePath, "maxDepth": n}` / `{"$path+": ...}` —
  every path in the closure (section 6) is its own focus node.
- A `validate`/`check` entry has exactly one of `shape` or `unique`;
  declaring both is `SHAPE_ERROR`. `unique` checks the named `field` (or, for
  non-object elements, the element itself) for repeats across the target's
  population, in target-visiting order. Every element sharing a key with an
  earlier one produces a violation at the declared `severity`; the first
  occurrence of each key is never flagged. An element that is an object
  lacking `field` has no key and is skipped, not an error (an index build over
  the same field raises `MISSING_PATH`, section 3). `unique` may carry the same
  optional `"id"` member `check` and field shapes do (section 10). This does
  not replace the existing build-time uniqueness check on a non-`multi`
  index (section 3, still `SHAPE_ERROR`, still fatal at build time) — that
  check is about an index's own well-formedness as a function; `unique` is
  an ordinary, `report`-mode-compatible data constraint over any population,
  independent of whether an index happens to exist over the same field.
- `mode` is required on every `validate` entry and `check` instruction; there
  is no default, and omitting it is `SHAPE_ERROR`.
- `mode: "gate"` — the first violation-severity finding aborts the pipeline with
  `VALIDATION_ERROR` (section 9); `program` does not run. A `warning` or
  `info` finding, of a shape or of a `unique` entry, never aborts a gate: it is
  collected and the pipeline goes on, as `conforms` ignores it.
  `["input"]`/`["output"]` gate targets are the direct generalisation of
  Jaxson's `INPUT_ERROR`/`OUTPUT_ERROR`. For a `unique` entry in `gate` mode,
  the first element, in target-visiting order, that repeats an already-seen
  key aborts with `VALIDATION_ERROR`/`SHAPE_MISMATCH` — the same "first
  offender in visiting order" rule `reference`'s gate mode already uses
  (section 5).
- `mode: "report"` — every violation for that target is collected, never
  aborts; the report is written to the path named by `into`, or returned
  alongside `output` if `into` is omitted. `into` is valid only in `report`
  mode and must be a writable path (`state` or `output`). A `validate` entry
  **sets** the whole report value, `{"conforms", "violations"}`, at `into`; a
  `check` **appends** each violation to the array already at `into`. Either
  write is an ordinary program write: it costs what the equivalent `set` or
  `append` costs (one step per violation appended) and is visible to the
  mutation log, so an index over the written path is rebuilt on its next use.
- Validation entries run in declared order within two stages. An entry whose
  target is rooted wholly in `input` (for `$indexed`, the root of the index's
  source) runs before the program, as the generalisation of the input schema;
  every other entry, including one whose root cannot be known statically,
  runs after it, as the generalisation of the output schema. Declared order is
  kept within each stage. Each entry's index rebuilds
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
| `SHAPE_ERROR` | malformed shape, index, `relations`, or `computes` entry; `extends` cycle without a depth bound; missing `maxShapeDepth` when required; a repeated key in a non-`multi` index or the `to` side of a relation; `cardinality: "one-to-one"` uniqueness violation on `from.field`; a redundant relation/shorthand pair (section 4c); a malformed `unique` declaration, or a `validate`/`check` entry declaring both `shape` and `unique` (section 7); a `requiredIds` key naming a member not in `required` (section 4); `enum` present on a `kind` other than `string`/`number`/`boolean`/`null` (section 4); an `extends` whose child conflicts with its parent: a different `kind`, reference target, `items` or `qualified`, or a merged range that is empty (a minimum above its maximum), or a field-name collision without `override` (section 4) | |
| `VALIDATION_ERROR` | a `gate`-mode target fails to conform | `SHAPE_MISMATCH` |
| `EXECUTION_ERROR` | (existing Jaxson category, extended) | `DANGLING_REFERENCE`, `SHAPE_DEPTH_EXCEEDED`, `PATH_DEPTH_EXCEEDED`, plus Jaxson's own `TYPE_ERROR`/`MISSING_PATH` where index keys and `reference` values resolve to a value of the wrong JSON type, and `TYPE_ERROR` where `local.step` resolves to a non-scalar-segment value |

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
      "focusPath": ["input", "orders", 4],
      "constraintPath": ["status"],
      "kind": "constraint",
      "shape": "Order",
      "constraintId": "ORDER_STATUS_ENUM",
      "severity": "violation",
      "message": "status must be one of draft, placed, shipped",
      "code": null
    }
  ]
}
```

- `conforms` reflects `violation`-severity findings only; `warning`/`info`
  entries never flip it.
- `focusPath` is a plain Jaxson path — always resolvable, always addressable.
  There is no `value` member: the offending value is always derivable from
  `focusPath` (or `focusPath` + `constraintPath`) against the original input,
  consistent with the language's general preference for paths over copied
  data.
- `constraintPath` is a plain path, relative to `focusPath`, naming the
  member the finding is about. It is present whenever a finding is
  attributable to a specific field or `required` member, and omitted
  (not present at all, not `null`) otherwise.
- `kind` is `"structural"` whenever `code` is non-`null`, and `"constraint"`
  whenever `code` is `null`. It is mechanically derived, never independently
  authored.
- `constraintId` is an optional author-supplied stable string, settable via
  an `"id"` member on a field shape, a `check` block, a `unique` declaration,
  or, for a `required` member, via `requiredIds` (section 4). It is `null`
  when the author didn't supply one. `constraintId` values are not required
  to be unique within a package — an author may reuse one id across
  constraints representing the same logical rule; tooling wanting per-rule
  aggregation keys on `constraintId` when present and falls back to
  `(shape, constraintPath)` otherwise.
- `code` is present (one of the `EXECUTION_ERROR` codes below) for
  structural findings like a dangling reference or a depth overrun, and
  `null` for an author-written `check`/combinator/`unique` failure, whose
  identity is its `constraintId` (if supplied) and `message`. `code` is
  never author-assigned.

**Violation granularity.** This table is exhaustive: every way a `report`-
mode check can fail produces violations exactly as follows, and nothing
produces more than one violation per row's own unit.

| Failure | Violations produced |
|---|---|
| A `required` member absent | One violation per absent member. `focusPath` = the object; `constraintPath` = `[memberName]`; `constraintId` from `requiredIds` if declared. |
| A `fields` member present but failing its own primitive kind/keyword check (`type`, `enum`, `minLen`, etc.) | One violation. `focusPath` = the object; `constraintPath` = `[memberName]`. |
| A `fields` member whose value is itself checked against a named or inline nested shape | The nested shape check runs as its own target: its own `focusPath` (the member's path), its own violation if it fails. No `constraintPath` on the parent — this is the general rule that `reference`-in-array (section 5) is a special case of. |
| A shape's own `check` (a `$compute` island) fails | One violation. `focusPath` = the shape's own focus node; no `constraintPath`. |
| A shape's own `and`/`or`/`xone`/`not` combinator fails | One violation for the whole shape (section 4), using the shape's own `severity`/`message` — never one per failing or unevaluated alternative. |
| A `unique` declaration finds a repeat | One violation per repeating element (section 7). `focusPath` = the repeating element's own path; no `constraintPath`. |
| A closed object has members its shape does not name | One violation per unexpected member, in code-point order of the member names. `focusPath` = the object; `constraintPath` = `[memberName]`. |
| A `qualified` count is not met | One violation for the whole collection. `focusPath` = the collection's own focus node; no `constraintPath`. |

`gate` mode aborts on the *first* failure, in the fixed order `required`
names (listed order), then `fields` members (code-point order), then
`check`, raising `VALIDATION_ERROR`/`SHAPE_MISMATCH` for that one failure.
`report` mode collects every independently failing `required` member and
`fields` member per this table, not only the shape's overall pass/fail.

**Fixed messages and severity for structural findings.** A structural
finding's `message` and `severity` are fixed by the runtime, not
author-configurable, and always carry `kind: "structural"`:

| Code | Fixed message | Fixed severity |
|---|---|---|
| `DANGLING_REFERENCE` | `"referenced value is not present in the named index"` | `violation` |
| `SHAPE_DEPTH_EXCEEDED` | `"shape recursion exceeded maxShapeDepth"` | `violation` |
| `PATH_DEPTH_EXCEEDED` | `"path closure exceeded maxDepth"` | `violation` |

## 11. What Shaxon does not take from SHACL

- **The RDF/triple data model**, blank nodes, and IRI identity — a Shaxon
  focus node is a JSON value at a path; identity is "same index key," not
  "same IRI."
- **Open-ended custom constraints** (`sh:sparql`, `sh:js`) — replaced entirely
  by `$compute`.
- **Property paths with unbounded closure** — SHACL's `sh:path` zero-or-more
  has no required depth bound; Shaxon's does, unconditionally (section 6).

## 12. Open questions

1. Whether a population worth targeting can be identified only by structural
   shape rather than by an explicit discriminator field — `$discriminator`
   is currently the only population-targeting mechanism for tagged data, and
   there is no fallback for untagged populations. A structural target form
   would need shape evaluation to run as part of target resolution itself,
   which risks reintroducing recursion (`target → shape → target → shape`)
   into a model whose boundedness this document otherwise treats as
   non-negotiable (the four invariants, section 0); any proposal here needs
   an answer to that recursion before it needs syntax.

## 13. Implementation status

A Go implementation exists (`src/jaxson-shaxon-v0.3.1`), with conformance
fixtures in `pkg/shaxon/shaxon-v0.3.1-fixtures.json`; the rulings it
raised, settled in this document, are recorded in its `TRACKER.md`. A conformant implementation requires, at
minimum: an index builder, including the
mandatory reuse rule of section 3; a `relations` desugarer (section 4a) with
the redundant-declaration check of section 4c; a `computes` registry
(section 4d); a shape evaluator covering
`kind`/`fields`/`closed`/`ignoredProperties`/combinators/`qualified`/`check`/
`extends` (section 4), producing violations at the granularity section 10
specifies (per-`required`-member, per-field, per-nested-shape, per-combinator,
per-`unique`-repeat); a target resolver for `$path`/`$each`/
`$discriminator`/`$indexed`/closure targets, and `unique` declarations
(section 7); a path-expression evaluator for
`$altPath`/`$path*`/`$path+`/`$inverse` (section 6); and a fixture format
whose `expect` member can hold `{"report": {...}}` in addition to
`{"output": ...}`/`{"error": {...}}`.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
