# Shaxon v2.0 core: SHACL's expressiveness on Jaxson's total, deterministic core

Status: proposal, unimplemented — still no reference interpreter (see section 14).
This document **supersedes** shaxon-v0.0.1.md, shaxon-v0.1.1.md and shaxon-v0.1.2.md.
It introduces no new behaviour of its own: every rule below is the final state those
three documents left the language in, written once as a single spec instead of a
delta stacked on a delta stacked on a delta. Where the three source documents
disagreed with themselves over time (the `extends` merge rule, the tree-modelling
question, the "is redundant declaration an error" question), what appears below is
the *last* answer, not a synthesis of all three.

## Front matter: what this document replaces, and why the version number moves

The three predecessor documents used `"shaxon"` version strings `"1.0"`, `"1.1"`,
`"1.2"`, each one runnable by a runtime one step ahead of it, per each document's own
compatibility clause. This document's package member is `"shaxon": "2.0"` — a new
number, not `"1.3"` — for the same reason `jaxson-v0.1.0-core-design.md` consolidated
"the existing five parts" into one coherent document rather than issuing a sixth
patch: this is an editorial consolidation, not a behavioural revision, and the
version bump exists to mark that the document, not the language, changed shape.
Concretely:

- A **"2.0" runtime is byte-for-byte identical in its rulebook to a "1.2" runtime**.
  Nothing in sections 0–14 below describes behaviour that a correct "1.2"
  implementation didn't already have.
- A **"2.0" runtime still runs a declared-`"1.2"`, `"1.1"` or `"1.0"` package exactly
  as that package's own version specifies** — the compatibility chain established by
  shaxon-v0.1.1.md and shaxon-v0.1.2.md is unbroken by this document. A "1.2"-only
  runtime, symmetrically, must reject a "2.0" package with `VERSION_ERROR`, the same
  discipline every prior version bump used.
- Everything that shaxon-v0.0.1.md, shaxon-v0.1.1.md and shaxon-v0.1.2.md separately
  said is folded in below at the point in the document where it now permanently
  lives, not as a changelog entry. The three source documents remain the historical
  record of *how* the language got here; this document is the record of *where* it
  is.

| Superseded document | What it contributed, now living in this document |
|---|---|
| shaxon-v0.0.1.md | Sections 0–14 baseline: stance, package shape, indices, shapes, `reference`, path expressions, targets/`validate`, `check`, errors, report, non-goals, lineage |
| shaxon-v0.1.1.md | Index build cost/reuse (§3); `relations` sugar (§4a); full `$path*`/`$path+` specification incl. `local.step` and closure-as-target (§6–7); `PATH_DEPTH_EXCEEDED` (§9) |
| shaxon-v0.1.2.md | Fully generalised `extends` merge table (§4); tree-modelling guidance (§4b); redundant-declaration rejection (§4c); `reference` inside arrays and its reporting granularity (§5); fixed structural violation messages (§10) |

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
| Relational reach depends on real graph edges | SHACL's inverse paths and joins work because RDF has edges; a JSON tree has none | Shaxon adds declared, named **indices** (and the `relations` sugar over them) — explicit, built-once maps that stand in for the edges a tree doesn't have |

Everything not called out below carries over from Jaxson v1 unchanged; this document
only specifies the delta from Jaxson.

## 1. What carries over from Jaxson unchanged

- The four roots (`input`, `state`, `output`, `local`) and plain segment-array paths.
- Operand forms: `$lit`, `$path`, `$compute`, `$tpl`, and `$opt` inside templates.
- The eight core Jaxson instructions (`set`, `delete`, `append`, `insert`, `for`,
  `if`, `assert`, `halt`), plus one new instruction added in section 8 (`check`).
- The compute sublanguage: closed operator set, `with`-bound operands, no reach into
  state.
- Exact-decimal numbers, the 38-digit magnitude bound, and `div`/`round` as the only
  digit-losing operations.
- Step-counted termination as the semantic resource bound.
- The error-category discipline: stable categories, stable codes within
  `EXECUTION_ERROR`, fixed pipeline order.

A Shaxon runtime that also implements Jaxson v1 can run a Jaxson package unmodified —
`shapes`, `validate`, `indices` and `relations` are all optional package members with
an empty default.

## 2. The package

```json
{
  "shaxon": "2.0",
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
  (`VERSION_ERROR` before anything else runs).
- `limits.maxShapeDepth` is mandatory whenever any shape in the package uses a
  bounded closure path (section 6) or `extends` (section 4); its absence in that
  situation is a `SHAPE_ERROR`, not a silent default — recursion is never
  accidentally unbounded.
- `inputSchema`/`outputSchema` keep Jaxson's original narrow contract vocabulary
  (section 8 of the Jaxson core design) for the common case where a package only
  needs type-and-shape gating at the two edges. `shapes`/`validate` are for
  everything a plain schema can't say. A package may use either, both, or neither.
- `indices`, `relations` and `shapes` are named registries, read before `validate`
  runs.
- Duplicate keys anywhere, including inside `shapes`, `indices` or `relations`, are a
  `PARSE_ERROR` — unchanged from Jaxson.
- A `relations` entry and an inline `reference` shorthand that would each build an
  index over the identical `(path, key)` pair are rejected at load — see section 4c.

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
- Without `"multi": true`, a repeated key is a `SHAPE_ERROR` at build time (an index
  is a function unless you say otherwise). With `"multi": true`, the index maps a
  key to the *set* of matching elements — this is how an inverse relationship
  ("every order for this customer") is recovered without a graph edge.
- Indices are rebuilt from their declared source immediately before every use — by
  any `validate` entry and by every `check` instruction (section 8) — so a `program`
  that mutates `state` between checks is always validated against current data,
  never a stale snapshot. This follows directly from Jaxson's copy-and-no-aliasing
  discipline; it would be a surprising exception to cache it.
- A `reference` node kind (section 5), the `relations` sugar (section 4a) and an
  `$inverse` operand (section 6) are the places indices get consumed.

**Cost.** Every index (re)build costs one step per element visited in `source`,
charged by the same "one step per iteration begun" rule Jaxson already applies to
`for` (see the Jaxson core design's step-accounting section). An index build is a
snapshot iteration like any other; it is not a hidden operation exempt from the step
budget. The charge lands at the point the build happens: before first use in
`program`, before each `validate` entry resolves its targets, and before each `check`
instruction — i.e., at every point this section already says a rebuild occurs. This
is not a cosmetic bookkeeping addition: without it, a package can declare a tiny
`limits.steps` and still perform unbounded backing work behind a `check` call in a
loop, which would contradict the premise that the declared step limit is a real
resource bound. See `check-in-loop-must-charge-index-rebuild-steps` in
shaxon-v0.1.1-fixtures.json for the concrete case: an 8-element index, rebuilt
twice, against a 10-step budget — free rebuilds finish under budget and produce
`output`; charged rebuilds hit `RESOURCE_ERROR`/`STEPS` on the very first `check`.

**Reuse.** An index build may be reused without recharge if no `set`/`delete`/
`append`/`insert` since that build targeted a path equal to, or a prefix of, its
declared `source`. This is decidable from the mutation-path log the runtime already
keeps for the no-aliasing guarantee — checking "has anything touched this prefix
since" is a lookup against that existing log, not a new mechanism, and not a
deep-equality scan. Skipping a charge for an index that provably wasn't touched is
not an exception to totality; charging every build unconditionally, with reuse as a
best-effort skip on top, is. A runtime that never implements the reuse rule is still
conformant — it simply pays the full cost every time, which is exactly what an
author who wrote `check` in a hot loop asked for.

## 4. Shapes

A shape is a named predicate over one JSON value (the *focus node*). The vocabulary
is deliberately at SHACL's level of expressiveness, not JSON Schema's.

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
  `null` `any` — plus **`reference`** (must name an `index` or a `relation`; section
  5) and **`node`** (a shape that constrains structure without asserting JSON's
  primitive type, used for combinator-only shapes — see below).
- `closed` + `ignoredProperties` extends Jaxson's `extra: "reject"|"allow"` with a
  named exclusion list, matching `sh:closed` + `sh:ignoredProperties` exactly:
  closed objects are the default posture (per Jaxson's own stance), and
  `ignoredProperties` is the escape hatch for metadata fields no field rule should
  have to name.
- `fields` values are shapes inline or `{"shape": "Name"}` by reference — this is
  how recursion enters (a `Category` shape whose `children` field references
  `Category` itself; bounded by `maxShapeDepth`, section 6).
- `qualified` on an array or object field is `sh:qualifiedValueShape` +
  `qualifiedMinCount`/`qualifiedMaxCount` folded into one field: "between `min` and
  `max` of this collection's elements match `shape`." Counting runs over the
  existing finite snapshot, so it costs nothing new in termination terms.
- `check` is the constraint SHACL would reach for `sh:sparql` or `sh:js` to
  express. It is a `$compute` island: `with` binds named operands (the focus node
  is always available as `local.focus`, without needing to be listed), `expr` must
  reduce to a boolean. Because it's `$compute`, it is closed and step-bounded by
  construction — there is no analogue of an unbounded SPARQL query here.
- `severity` is `violation` (default) | `warning` | `info`. Only `violation` affects
  `conforms` in the validation report (section 10); `warning`/`info` are always
  collected, never gate.
- Logical combinators, sibling to `fields`:

| Combinator | Meaning |
|---|---|
| `and: [shape, ...]` | focus node must satisfy every listed shape |
| `or: [shape, ...]` | at least one |
| `xone: [shape, ...]` | exactly one |
| `not: shape` | must not satisfy it |

  A shape using only combinators and no `kind`/`fields` is a `node`-kind shape —
  pure composition, the way SHACL lets a shape be nothing but `sh:and`/`sh:or` over
  other shapes.
- `extends: "Name"` (singular — single inheritance only; see section 12 on why
  multiple inheritance is deliberately excluded) merges another shape's members
  into this one. `extends` participates in the same `maxShapeDepth` budget as
  recursive `fields` references — an `extends` chain has to terminate too.

**The `extends` merge rule, fully specified.** One principle covers every mergeable
member: **every `extends`-mergeable member accumulates using whatever aggregation is
natural for that member, unless the child marks that member `"override": true`,
which replaces instead of accumulating.**

| Member | Aggregation | Override behaviour |
|---|---|---|
| `closed` | AND — most restrictive of parent/child wins | `"override": true` makes the child's own value win outright |
| `fields` | union; a name collision is `SHAPE_ERROR` | per-field `"override": true` replaces the parent's field definition |
| `required` | union — required in either parent or child means required in the merged shape | none needed; union only ever gets stricter |
| `and` | concatenate parent's list then child's | `"override": true` on the child's `and` replaces the parent's list instead of concatenating |
| `or` | concatenate parent's list then child's, then apply "at least one" over the combined pool | `"override": true` replaces instead of pooling |
| `xone` | concatenate parent's list then child's, then apply "exactly one" over the combined pool | `"override": true` replaces instead of pooling |
| `not` | parent's and child's `not` targets both apply — focus must satisfy neither (equivalent to AND-ing the two negations) | `"override": true` on the child's `not` replaces the parent's instead of adding to it |
| `check` | parent's and child's `check` are ANDed — focus must pass both | `"override": {"with": ..., "expr": ...}` on the child replaces the parent's `check` instead of ANDing |

One consequence is worth stating explicitly rather than leaving it to be
discovered: **pooling `xone` can turn a focus node that satisfied the child's own
list into a violation**, because "exactly one" is evaluated over the merged pool,
not the child's list alone. A focus node matching exactly one of the child's own
alternatives, plus one more of the parent's, now matches two — and fails. This
isn't a bug in the merge rule; it's what "exactly one, generalised to inherited
alternatives" has to mean. See `extends-pools-xone-alternatives-not-just-childs-own`
in shaxon-v0.1.2-fixtures.json: a shape with a single-element `xone` passes on its
own, then fails once it `extends` a parent contributing a second alternative the
same focus also matches.

## 4a. Relations

A relation is a named, bidirectional relational declaration that desugars to
exactly the index/reference/`$inverse` machinery of section 3, so it changes
nothing about what is expressible — only how much of it has to be written out by
hand.

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
  `to.key` — equivalent to `{"source": to.path, "key": {"$path": ["local", "item",
  to.key]}}` — exactly as if the author had written that index by hand. A repeated
  `to.key` value is a `SHAPE_ERROR` at build time, same as any non-`multi` index.
- `from` is the many side: the runtime builds a `multi` index over `from.path`
  keyed by `from.field`, used only to answer `$inverse` — equivalent to `{"source":
  from.path, "key": {"$path": ["local", "item", from.field]}, "multi": true}`.
- `cardinality: "one-to-many"` (default) leaves `from.field` free to repeat (many
  orders may share a customer). `cardinality: "one-to-one"` additionally requires
  uniqueness on `from.field`, checked at the same build step as the `to.key`
  uniqueness check; a violation is `SHAPE_ERROR`, matching how a bare index already
  rejects a repeated key without `multi: true`.
- Both indices a `relations` entry produces are anonymous: they are not visible
  under `indices`, cannot be named in a bare `{"kind": "reference", "index": ...}`,
  and do not participate in name collision checks against `indices`. A `reference`
  field consumes a relation with `{"kind": "reference", "relation": "orderCustomer"}`
  — always against the `to` side, since a reference only ever points at the unique
  side of a relationship. `$inverse` consumes it with `{"$inverse": {"relation":
  "orderCustomer", "key": operand}}` — always the `from` side, since recovering
  "what points at me" is the entire purpose of `$inverse`. Neither needs a `side`
  parameter; there is exactly one sensible reading for each.
- A malformed `relations` entry (missing `from`/`to`, an invalid `cardinality`
  value, a `field`/`key` that isn't a plain field name) is `SHAPE_ERROR`.

**Inline reference shorthand**, for the common case where only the forward
direction is ever needed and a full `relations` entry would be overkill:

```json
"customerId": { "kind": "reference", "of": { "$path": ["input", "customers"] }, "by": ["id"] }
```

Desugars to an anonymous index scoped to this one field — equivalent to
`{"source": of, "key": {"$path": ["local", "item"].concat(by)}}` — internal to the
shape/field pair that declared it. It is not nameable, not visible under `indices`,
and has no `$inverse` form; if the reverse direction or reuse across multiple
fields turns out to be needed later, that is the signal to promote it to a named
index or a `relations` entry instead, not to add parameters to the shorthand.

**Before/after**, using the Order/Customer example above. Without `relations`, one
relationship costs three declarations plus a call site:

```json
"indices": {
  "customersById": { "source": {"$path":["input","customers"]}, "key": {"$path":["local","item","id"]} },
  "ordersByCustomer": { "source": {"$path":["input","orders"]}, "key": {"$path":["local","item","customerId"]}, "multi": true }
},
"shapes": { "Order": { "fields": { "customerId": {"kind":"reference","index":"customersById"} } } }
```
```json
{"$inverse": {"index": "ordersByCustomer", "key": {"$path":["local","focus","id"]}}}
```

With `relations`, one declaration replaces both indices, and both call sites name
the relationship instead of a hand-built index:

```json
"relations": {
  "orderCustomer": {
    "from": {"path": ["input","orders"], "field": "customerId"},
    "to":   {"path": ["input","customers"], "key": "id"}
  }
},
"shapes": { "Order": { "fields": { "customerId": {"kind":"reference","relation":"orderCustomer"} } } }
```
```json
{"$inverse": {"relation": "orderCustomer", "key": {"$path":["local","focus","id"]}}}
```

See `relations-block-desugars-to-forward-reference-and-inverse-lookup` in
shaxon-v0.1.1-fixtures.json for a full round-trip: forward validation of every
order against the relation, and an `$inverse` lookup recovering a customer's
orders, against a package that never names an index at all.

## 4b. Modeling a tree: `$path+`/`$path*` versus a self-referential `relations` entry

This isn't really one choice between two equally-valid mechanisms — the two apply
to two different data shapes:

| Data shape | Use |
|---|---|
| Genuinely nested — a child is a literal descendant in the JSON (`children: [...]` embedded under its parent) | `$path+`/`$path*` closure (section 6) |
| Flat — one array of same-shaped elements related by an id field (an adjacency list: `{"id": 2, "parentId": 1}`) | `relations` with `from.path == to.path` |

`$path+` has no way to reach an ID-linked parent in a flat array — its `innerStep`
is a relative *path*, and there is no structural path from a node to a same-array
sibling without going through an index. A genuinely nested tree needs no index at
all — you just walk it. In the ordinary case, the data's own shape decides which
mechanism applies; there's nothing to make canonical.

The real overlap is narrower: a dataset denormalised **both** ways at once (nested
*and* carrying redundant parent-id fields). For that case only: neither mechanism
is canonical over the other; use whichever one `program` actually navigates by for
its own purposes, and add a `check` asserting the two views agree — every node
reachable via the `$path+` closure from a root is also reachable via the
`relations`-based `$inverse` chain, or vice versa. That turns "pick a winner" into
an assertion of consistency, which is what `check` already exists to express, not
a case for new machinery.

## 4c. Redundant declarations

Two declarations that would each build an index over the identical `(path, key)`
pair are a `SHAPE_ERROR` at package load — the same class of static check as the
existing duplicate-index-key rejection, just applied across declaration forms
instead of within one:

- An inline `{"kind": "reference", "of": ..., "by": [...]}` shorthand whose
  normalised `(of, by)` matches the `to` side of a `relations` entry — write
  `{"kind": "reference", "relation": name}` instead.
- Two `relations` entries whose `to` sides normalise to the same `(path, key)` —
  point every consuming field at one of them.

This is pure static comparison of normalised path segments and key field names, no
data inspection needed — decidable at the same "parse, version, schemas and shapes
and indices (all static)" pipeline stage that already checks malformed shapes and
indices (section 9).

Deliberately **not** extended to a standalone named `indices` entry that happens to
match a relation's derived index — that's the general escape hatch, and an author
reaching for it explicitly is making a considered choice the shorthand-collision
case isn't.

See `redundant-relation-and-shorthand-rejected-at-load` in
shaxon-v0.1.2-fixtures.json.

## 5. The `reference` node kind

```json
"customerId": { "kind": "reference", "index": "customersById" }
```

The value must be a scalar found as a key in the named index (or the `to` side of a
named relation, section 4a); if not, `DANGLING_REFERENCE` (an `EXECUTION_ERROR`
code, section 9). This is the direct, JSON-native replacement for the referential
integrity SHACL gets for free from RDF identity: instead of the runtime knowing
what an IRI is, Shaxon requires you to say, once, which array is the universe of
valid values for this field — then every use of that field is checked against it.

**Inside an array**, `reference` composes under the existing grammar exactly the
way any other item shape does — no new syntax:

```json
"tagIds": {
  "kind": "array",
  "items": { "kind": "reference", "index": "tagsById" }
}
```

**Reporting rule**, extending the array-iteration convention already used
everywhere (`for`, `$each`, index builds all snapshot-iterate in array order):

- `report` mode: every dangling element gets its **own** violation with its own
  `focusPath` (e.g. `["input", "order", "tagIds", 2]` for the third element) —
  three bad entries in one field produce three violations, not one aggregate
  finding.
- `gate` mode: the **first** dangling element in array-index order raises
  `DANGLING_REFERENCE` and aborts, exactly like a gate target aborts on its first
  violation anywhere else.
- The violation's `shape` field names the nearest enclosing **named** shape being
  checked (the one passed to `check`/`validate`), not the inline, unnamed
  `reference` item — consistent with the section 10 example, where
  `"shape": "DiscountLine"` names the shape passed at the call site, not some
  constituent of it.

See `array-of-references-reports-one-violation-per-dangling-element` in
shaxon-v0.1.2-fixtures.json.

## 6. Path expressions

Jaxson's plain segment array (`["input", "items", 0, "price"]`) is still the base
case and still the only form allowed where Jaxson itself uses paths (`set`,
`delete`, etc.). Shapes and indices may additionally use:

| Form | Meaning |
|---|---|
| `{"$altPath": [pathA, pathB, ...]}` | the first alternative that resolves; `MISSING_PATH` only if none do |
| `{"$path*": innerStep, "maxDepth": n}` | zero or more repetitions of `innerStep`, bounded |
| `{"$path+": innerStep, "maxDepth": n}` | one or more repetitions, bounded |
| `{"$inverse": {"index": name, "key": operand}}` or `{"$inverse": {"relation": name, "key": operand}}` | every element the named `multi` index (or the `from` side of the named relation) maps that key to |

- **`innerStep` is a relative path** — the same plain segment-array grammar as an
  ordinary path, applied relative to the current position rather than from a root.
  No new grammar: `{"$path+": ["children"], "maxDepth": 5}` means "repeatedly
  append `.children`." A multi-segment step (`{"$path+": ["next", "value"], ...}`)
  is a single hop that happens to cross more than one field.
- **The result is a set of paths**, not a single value. This is what actually earns
  the "SHACL-equivalent expressiveness" claim: SHACL's `sh:path` zero-or-more/
  one-or-more is set-producing — it returns every node reachable within the bound —
  and a version that only returned the deepest frontier would not be parity with
  that. From base path `B`: `$path*` = `{B, B+step, B+step+step, ...}` up to
  `maxDepth` hops, stopping the chain — not erroring — the first time a hop doesn't
  resolve (`MISSING_PATH` mid-chain is expected, the way an RDF property simply not
  being present further down is expected, not an error). `$path+` is the same set
  minus `B`; it is a failure only if zero hops resolve at all (nothing reachable,
  which is what "+" meaning "at least one" requires).
- **As a target (section 7)**, a closure needs an explicit base, since there is no
  ambient focus yet: `{"$path*": step, "from": basePath, "maxDepth": n}` (`from` is
  omitted when the same form appears embedded inside a shape's `check`, an index's
  `key`, or any other operand position that already has an ambient `local.focus` or
  `local.item` — there, the ambient position is the base).
- **`local.step`** is available inside `innerStep` for the case a static field name
  can't express — a step whose direction depends on the node's own data rather
  than a fixed field — bound to the current traversal position, parallel to
  `local.item` in `for` and in index builds. The common case
  (`{"$path+": ["children"]}`) never needs it.
- `maxDepth` remains mandatory on both closure forms — its absence is a static
  `PROGRAM_ERROR`, exactly parallel to Jaxson's undeclared-`local` and
  undeclared-`$v` checks. It is not merely documentation: **exceeding it while the
  chain still resolves further** raises `PATH_DEPTH_EXCEEDED` (section 9) — a
  violation, not a silent stop, and distinct from the shape-recursion
  `SHAPE_DEPTH_EXCEEDED`, so a violation record never conflates two structurally
  different overruns under one code. A `Category` tree deeper than its declared
  bound fails to conform; it doesn't just stop being checked partway down. This is
  the deliberate design choice that lets recursive shapes exist at all inside a
  total language — SHACL's own zero-or-more paths have no such bound and can, in
  principle, walk an infinite or cyclic graph forever.
- `$inverse` is how "what points at me" is expressed without a graph edge: it
  reads a `multi` index (section 3), directly or via a `relations` entry's `from`
  side (section 4a), rather than walking anything. It is only valid where an index
  build has already made the answer a lookup, never a search.

See `path-plus-closure-target-reports-violation-mid-chain` in
shaxon-v0.1.1-fixtures.json: `$path*` as a section 7 target over a 4-node linked
structure, `report` mode, one violation at depth 2.

## 7. Targets and the `validate` list

```json
"validate": [
  { "target": { "$path": ["input"] }, "shape": "Order", "mode": "gate" },
  { "target": { "$each": ["input", "customers"] }, "shape": "Customer", "mode": "gate" },
  { "target": { "$discriminator": { "at": ["input", "lines"], "field": "kind", "value": "discount" } },
    "shape": "DiscountLine", "mode": "report", "into": ["state", "lineWarnings"] },
  { "target": { "$path*": ["next"], "from": ["input", "list", "head"], "maxDepth": 10 },
    "shape": "ListNode", "mode": "report", "into": ["state", "listWarnings"] }
]
```

- `{"$path": [...]}` — one focus node.
- `{"$each": [...]}` — every element of the array/object at that path is its own
  focus node (SHACL's `sh:targetObjectsOf` analogue for a tree: there's no property
  to target objects *of*, so this targets a collection directly).
- `{"$discriminator": {...}}` — every element under `at` whose `field` equals
  `value` (the tree-native stand-in for `sh:targetClass`, since a JSON element has
  no `rdf:type`, only whatever discriminator field the schema author chose).
- `{"$indexed": name}` — every entry of a named index (useful when the interesting
  population is exactly what an index already enumerates).
- `{"$path*": step, "from": basePath, "maxDepth": n}` / `{"$path+": ...}` — every
  path in the closure (section 6) is its own focus node. This is what lets a
  package validate every node of a recursive structure (a `Category` tree, a
  linked list) directly, without a separately hand-written recursive shape for the
  sole purpose of walking it.
- `mode: "gate"` — first violation aborts the pipeline with `VALIDATION_ERROR`
  (section 9); `program` does not run. `["input"]`/`["output"]` gate targets are
  the direct generalisation of Jaxson's `INPUT_ERROR`/`OUTPUT_ERROR`.
- `mode: "report"` — every violation for that target is collected, never aborts;
  the full report is written to the path named by `into`, or returned alongside
  `output` if `into` is omitted.
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
turn-based program (a game, a workflow with intermediate states) can assert a
shape against `state` after every step, the same way `assert` checks a boolean
today. Mode semantics are identical to section 7; a `report`-mode `check` appends
into the named `into` path rather than replacing it, so a program can accumulate a
running log of non-fatal findings turn over turn.

## 9. Errors

| Category | Raised when | Codes |
|---|---|---|
| `SHAPE_ERROR` | malformed shape, index or `relations` entry; `extends` cycle without a depth bound; missing `maxShapeDepth` when required; `cardinality: "one-to-one"` uniqueness violation on `from.field` at build time; a redundant relation/shorthand pair (section 4c) | |
| `VALIDATION_ERROR` | a `gate`-mode target fails to conform | `SHAPE_MISMATCH` |
| `EXECUTION_ERROR` | (existing Jaxson category, extended) | `DANGLING_REFERENCE`, `SHAPE_DEPTH_EXCEEDED`, `PATH_DEPTH_EXCEEDED` |

`VALIDATION_ERROR` on `["input"]` or `["output"]` targets subsumes Jaxson's
`INPUT_ERROR`/`OUTPUT_ERROR` — a Shaxon runtime reports those under the new
category name; a compatibility shim can re-emit the old names for a package that
declares `jaxson` rather than `shaxon` as its version key.

Pipeline order: parse, version, schemas and shapes and indices and relations (all
static), program validation, `input` schema/gate-shapes, execution (including any
`check` instructions), `output` schema/gate-shapes. Same "earlier stage wins"
discipline as Jaxson.

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
- `code` is present (one of the `EXECUTION_ERROR` codes above) for structural
  findings like a dangling reference or a depth overrun, and `null` for an
  author-written `check`/combinator failure, whose only identity is its `message`.

**Fixed messages for structural findings.** A structural finding's `message` is
**fixed by the runtime**, not author-configurable — the fact that makes `code`
meaningful as the identity for these, since an author-written `check`/combinator
failure has no `code` and only its (author-supplied) `message` to identify it by:

| Code | Fixed message |
|---|---|
| `DANGLING_REFERENCE` | `"referenced value is not present in the named index"` |
| `SHAPE_DEPTH_EXCEEDED` | `"shape recursion exceeded maxShapeDepth"` |
| `PATH_DEPTH_EXCEEDED` | `"path closure exceeded maxDepth"` |

## 11. What Shaxon deliberately does not take from SHACL

- **The RDF/triple data model**, blank nodes, and IRI identity — a Shaxon focus
  node is a JSON value at a path; identity is "same index key," not "same IRI."
- **Open-ended custom constraints** (`sh:sparql`, `sh:js`) — replaced entirely by
  `$compute`. SHACL itself already validates under a closed-world assumption, same
  as Jaxson; that convergence needed no work. What needed replacing was only the
  escape hatch SHACL leaves for arbitrary code.
- **Property paths with unbounded closure** — SHACL's `sh:path` zero-or-more has no
  required depth bound; Shaxon's does, unconditionally (section 6).

## 12. Where I am least sure

The four items below are the ones that stood in shaxon-v0.0.1.md's original list
and were never subsequently touched — nothing in v0.1.1 or v0.1.2 resolved them,
and nothing in this consolidation does either:

1. **Single inheritance only (`extends` takes one name, not a list).** Diamond
   conflicts under multiple inheritance need a resolution rule I don't have yet;
   restricting `extends` to single inheritance sidesteps the question rather than
   answering it, and should be revisited once real shapes exist to test against.
   Section 4's fully-specified merge table settles *how* a single parent merges —
   it does not settle *whether* more than one parent should be allowed.
2. **`$discriminator` as the sole stand-in for `sh:targetClass`.** It assumes every
   population worth targeting carries an explicit tag field. That's usually true
   for hand-authored JSON but not guaranteed, and there's no fallback target form
   for a population identified only by structural shape rather than a tag.
3. **`SHAPE_DEPTH_EXCEEDED`/`PATH_DEPTH_EXCEEDED` as a violation versus a hard
   `RESOURCE_ERROR`.** Treating either as a reportable violation (so `report`-mode
   targets keep going) is more useful than aborting the whole pipeline, but it
   means a badly-bounded depth bound produces a very large violation list rather
   than a fast failure — might want a secondary cap on violation count per target.
   Splitting the two codes in v0.1.1 answered "which budget overflowed," not
   "should overflowing be reportable or fatal" — that question is exactly as open
   for both codes as it was for the one code in v0.0.1.
4. **Message templating.** Static strings are the safe default; whether `message`
   should accept a `$tpl` for interpolating the focus value in is tempting for
   readability and untested for how much it complicates the no-side-effect story.

**Resolved on the way here**, for reference — none of these are open questions any
more, and nothing below should be reopened without a fresh reason:

| Originally raised in | Item | Resolved in |
|---|---|---|
| v0.0.1 §12.3 | Index-rebuild cost in a hot loop | v0.1.1 §3 (this document's section 3): rebuilds cost one step per element, with a decidable reuse rule |
| v0.1.1 §12.1 | `$path+`/self-referential-`relations` overlap for tree modelling | v0.1.2 §4b (this document's section 4b): the two mechanisms apply to two different data shapes; only doubly-denormalised data needs a `check` reconciling both views |
| v0.1.1 §12.2 | Undetected redundant declarations | v0.1.2 §4c (this document's section 4c): identical `(path, key)` pairs across a shorthand and a relation are a load-time `SHAPE_ERROR` |

## 13. Lineage table

| From | Idea kept | Changed for Shaxon |
|---|---|---|
| Jaxson | roots, paths, operand forms, instructions, compute, exact decimals, step accounting | paths gain alternative/closure forms; one new instruction (`check`) |
| Jaxson | closed contract schemas (`type`, `fields`, `required`, `extra`) | kept verbatim as `inputSchema`/`outputSchema` for the simple case; superseded by `shapes` wherever SHACL-level expressiveness is needed |
| SHACL | node shapes, property constraints, `sh:closed`/`sh:ignoredProperties` | ported to JSON `kind`/`fields`/`closed`/`ignoredProperties` |
| SHACL | `sh:and`/`sh:or`/`sh:not`/`sh:xone` | ported as shape combinators, with an explicit, fully-specified `extends` pooling rule for how inherited alternatives interact with them |
| SHACL | `sh:qualifiedValueShape` + min/max count | folded into a field-level `qualified` clause |
| SHACL | severity levels, validation report shape | ported as `severity` + the `{conforms, violations}` report, with fixed messages for structural findings |
| SHACL | inverse/alternative property paths, referential reach via graph edges | recovered without graph edges via named, declared, rebuildable **indices**, the **`relations`** sugar over them, and `$inverse` |
| SHACL | `sh:sparql`/`sh:js` custom constraints | replaced by `$compute` — same expressive slot, closed and step-bounded instead of open |
| Shaxon v0.0.1's own indices/reference/`$inverse` machinery | one relationship expressed as three hand-written declarations plus call sites | folded into a single named, bidirectional `relations` declaration (section 4a), with an inline `reference` shorthand for the forward-only case |

## 14. What would need to exist before this is real

Jaxson earned its "checked, not just proposed" status from `jaxrun.go` plus 28
fixtures. Shaxon has neither yet. A reference interpreter would need, on top of
everything `jaxrun.go` already does:

- An index builder (section 3), including the step-cost and reuse rules, run
  before validation and before every `check`.
- A `relations` desugarer (section 4a) producing the same anonymous indices a
  hand-written pair would, plus the redundant-declaration check at load (section
  4c).
- A shape evaluator: `kind`/`fields`/`closed`/`ignoredProperties`/combinators/
  `qualified`/`check`/the full `extends` merge table (section 4), producing either
  a single pass/fail (`gate`) or an accumulated violation list (`report`), with
  per-element reporting for a `reference` inside an array (section 5).
- A target resolver for `$path`/`$each`/`$discriminator`/`$indexed`/closure targets.
- The path-expression evaluator for `$altPath`/`$path*`/`$path+`/`$inverse`,
  including `local.step` and the depth-overrun checks wired to produce
  `SHAPE_DEPTH_EXCEEDED` or `PATH_DEPTH_EXCEEDED` (as appropriate) rather than
  looping.
- A fixture format extended with an `expect` that can hold `{"report": {...}}` in
  addition to Jaxson's existing `{"output": ...}`/`{"error": {...}}`, so a
  `report`-mode fixture can assert the exact violation list, not just conformance.

Six fixtures now exist, pinning down intended behaviour but not yet verifying it —
until an interpreter exists to run them, they are assertions, the same caveat
shaxon-v0.0.1.md placed on itself:

- `check-in-loop-must-charge-index-rebuild-steps` (shaxon-v0.1.1-fixtures.json) —
  forces the section 3 cost rule.
- `path-plus-closure-target-reports-violation-mid-chain`
  (shaxon-v0.1.1-fixtures.json) — `$path*` as a section 7 target over a 4-node
  linked structure, `report` mode, one violation at depth 2.
- `relations-block-desugars-to-forward-reference-and-inverse-lookup`
  (shaxon-v0.1.1-fixtures.json) — one `relations` entry validated in both
  directions.
- `extends-pools-xone-alternatives-not-just-childs-own`
  (shaxon-v0.1.2-fixtures.json) — forces the `xone`-pooling rule in section 4.
- `array-of-references-reports-one-violation-per-dangling-element`
  (shaxon-v0.1.2-fixtures.json) — a 4-element reference array with two dangling
  entries, `report` mode, two independent violations at their own array-index
  `focusPath`s.
- `redundant-relation-and-shorthand-rejected-at-load`
  (shaxon-v0.1.2-fixtures.json) — a `relations` entry and an inline `reference`
  shorthand both targeting the identical `(path, key)`, rejected as `SHAPE_ERROR`
  before any input is processed.

This consolidation adds no new fixtures of its own — it changes nothing that a
fixture could distinguish from v0.1.2's behaviour. The next real step for Shaxon is
still what shaxon-v0.0.1.md said it was: a reference interpreter, checked against
these six fixtures plus whatever new ones exercising `maxShapeDepth` recursion
itself (not yet fixtured anywhere in this lineage) turn out to be needed.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
