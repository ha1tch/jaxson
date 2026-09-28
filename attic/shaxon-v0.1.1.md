# Shaxon v0.1.1: step-accounting, path closures as targets, relation sugar

Status: proposal, unimplemented — still no reference interpreter (see shaxon-v0.0.1.md
section 14). This revision adds a first fixture set (shaxon-v0.1.1-fixtures.json,
3 fixtures) pinning down the behavior below, but until an interpreter exists to run
them, they are assertions of intended behavior, not verified results — the same
caveat shaxon-v0.0.1.md placed on itself.

This is a delta on shaxon-v0.0.1.md the same way that document was a delta on
Jaxson: everything not named in the changelog below is unchanged, including every
worked example in v0.0.1 that doesn't involve `$path*`/`$path+`, indices, or
references.

Format version: `"shaxon": "1.1"`. A "1.0"-only runtime must reject a "1.1" package
with `VERSION_ERROR`. A "1.1" runtime runs any "1.0" package unmodified — every
addition here is either a new optional package member (`relations`) or a narrower
rule layered onto existing behavior (index step cost, closure semantics); nothing
that validated or computed a particular way under "1.0" changes meaning under "1.1".

## Changelog

1. **Section 3 (indices)** — index (re)builds now cost steps. Resolves v0.0.1
   section 12, item 3.
2. **Section 6 (path expressions)** — `$path*`/`$path+` fully specified: `innerStep`
   is a relative path (no new grammar), the result is a set of paths (matching
   SHACL's actual property-path semantics), a new target form is added in section 7,
   a `local.step` binding is available for data-dependent steps, and the overrun
   error code is split from `SHAPE_DEPTH_EXCEEDED`.
3. **New section 4a (`relations`)** — sugar for declaring a bidirectional
   relationship once instead of as two indices, a `reference` field, and a manual
   `$inverse` call site.
4. **Section 9 (errors)** — `PATH_DEPTH_EXCEEDED` added.
5. **Section 12 (open questions)** — item 3 from v0.0.1 is resolved (see section 3
   below); items 1, 2, 4, 5 carry over unchanged; two new items are added, below.

## 3. Indices: cost and reuse (revises v0.0.1 section 3)

Everything in v0.0.1 section 3 stands — `source`/`key`, `multi`, snapshot iteration,
rebuild timing. This adds two rules v0.0.1 left unstated.

**Cost.** Every index (re)build costs one step per element visited in `source`,
charged by the same "one step per iteration begun" rule Jaxson already applies to
`for` (jaxson-v0.1.0-core-design.md section on step accounting). An index build is a
snapshot iteration like any other; it is not a hidden operation exempt from the step
budget. The charge lands at the point the build happens: before first use in
`program`, before each `validate` entry resolves its targets, and before each `check`
instruction — i.e., at every point v0.0.1 section 3 already says a rebuild occurs.

This is not a cosmetic bookkeeping addition. Without it, a package can declare a tiny
`limits.steps` and still perform unbounded backing work behind a `check` call in a
loop, which contradicts the premise that the declared step limit is a real resource
bound. See `check-in-loop-must-charge-index-rebuild-steps` in
shaxon-v0.1.1-fixtures.json for the concrete case: an 8-element index, rebuilt twice,
against a 10-step budget — free rebuilds finish under budget and produce `output`;
charged rebuilds hit `RESOURCE_ERROR`/`STEPS` on the very first `check`.

**Reuse.** An index build may be reused without recharge if no `set`/`delete`/
`append`/`insert` since that build targeted a path equal to, or a prefix of, its
declared `source`. This is decidable from the mutation-path log the runtime already
keeps for the no-aliasing guarantee — checking "has anything touched this prefix
since" is a lookup against that existing log, not a new mechanism, and not a
deep-equality scan.

This resolves v0.0.1 section 12, item 3: reuse is a safe optimization *because* the
cost is now real. Skipping a charge for an index that provably wasn't touched is not
an exception to totality; charging every build unconditionally, with reuse as a
best-effort skip on top, is. A runtime that never implements the reuse rule is still
conformant — it simply pays the full cost every time, which is exactly what an author
who wrote `check` in a hot loop asked for.

## 4a. Relations (new)

A relation is a named, bidirectional relational declaration that desugars to exactly
the index/reference/`$inverse` machinery of v0.0.1, so it changes nothing about what
is expressible — only how much of it has to be written out by hand.

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
- `from` is the many side: the runtime builds a `multi` index over `from.path` keyed
  by `from.field`, used only to answer `$inverse` — equivalent to `{"source":
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
  "orderCustomer", "key": operand}}` — always the `from` side, since recovering "what
  points at me" is the entire purpose of `$inverse`. Neither needs a `side`
  parameter; there is exactly one sensible reading for each.
- A malformed `relations` entry (missing `from`/`to`, an invalid `cardinality`
  value, a `field`/`key` that isn't a plain field name) is `SHAPE_ERROR`.

**Inline reference shorthand**, for the common case where only the forward direction
is ever needed and a full `relations` entry would be overkill:

```json
"customerId": { "kind": "reference", "of": { "$path": ["input", "customers"] }, "by": ["id"] }
```

Desugars to an anonymous index scoped to this one field — equivalent to `{"source":
of, "key": {"$path": ["local", "item"].concat(by)}}` — internal to the shape/field
pair that declared it. It is not nameable, not visible under `indices`, and has no
`$inverse` form; if the reverse direction or reuse across multiple fields turns out
to be needed later, that is the signal to promote it to a named index or a
`relations` entry instead, not to add parameters to the shorthand.

**Before/after**, using v0.0.1's own Order/Customer example. Today, one relationship
costs three declarations plus a call site:

```json
"indices": {
  "customersById": { "source": {"$path":["input","customers"]}, "key": {"$path":["local","item","id"]} },
  "ordersByCustomer": { "source": {"$path":["input","orders"]}, "key": {"$path":["local","item","customerId"]}, "multi": true }
},
"shapes": { "Order": { "fields": { "customerId": {"kind":"reference","index":"customersById"} } } }
```
```
{"$inverse": {"index": "ordersByCustomer", "key": {"$path":["local","focus","id"]}}}
```

With `relations`, one declaration replaces both indices, and both call sites name the
relationship instead of a hand-built index:

```json
"relations": {
  "orderCustomer": {
    "from": {"path": ["input","orders"], "field": "customerId"},
    "to":   {"path": ["input","customers"], "key": "id"}
  }
},
"shapes": { "Order": { "fields": { "customerId": {"kind":"reference","relation":"orderCustomer"} } } }
```
```
{"$inverse": {"relation": "orderCustomer", "key": {"$path":["local","focus","id"]}}}
```

See `relations-block-desugars-to-forward-reference-and-inverse-lookup` in
shaxon-v0.1.1-fixtures.json for a full round-trip: forward validation of every order
against the relation, and an `$inverse` lookup recovering a customer's orders,
against a package that never names an index at all.

## 6. Path expressions (revises v0.0.1 section 6)

The plain segment-array path is unchanged and remains the only form Jaxson's own
instructions (`set`, `delete`, etc.) accept. `$altPath` and `$inverse` are unchanged.
`$path*`/`$path+` are now fully specified rather than a one-row table entry:

- **`innerStep` is a relative path** — the same plain segment-array grammar as an
  ordinary path, applied relative to the current position rather than from a root.
  No new grammar: `{"$path+": ["children"], "maxDepth": 5}` means "repeatedly append
  `.children`." A multi-segment step (`{"$path+": ["next", "value"], ...}`) is a
  single hop that happens to cross more than one field.
- **The result is a set of paths**, not a single value. This is the fix that
  actually earns the "SHACL-equivalent expressiveness" claim v0.0.1 makes for
  itself: SHACL's `sh:path` zero-or-more/one-or-more is set-producing — it returns
  every node reachable within the bound — and a version that only returned the
  deepest frontier would not be parity with that, whatever the table entry implied.
  From base path `B`: `$path*` = `{B, B+step, B+step+step, ...}` up to `maxDepth`
  hops, stopping the chain — not erroring — the first time a hop doesn't resolve
  (`MISSING_PATH` mid-chain is expected, the way an RDF property simply not being
  present further down is expected, not an error). `$path+` is the same set minus
  `B`; it is a failure only if zero hops resolve at all (nothing reachable, which is
  what "+" meaning "at least one" requires).
- **As a target (section 7)**, a closure needs an explicit base, since there is no
  ambient focus yet: `{"$path*": step, "from": basePath, "maxDepth": n}` (`from` is
  omitted when the same form appears embedded inside a shape's `check`, an index's
  `key`, or any other operand position that already has an ambient `local.focus` or
  `local.item` — there, the ambient position is the base).
- **`local.step`** is available inside `innerStep` for the case a static field name
  can't express — a step whose direction depends on the node's own data rather than
  a fixed field — bound to the current traversal position, parallel to `local.item`
  in `for` and in index builds. The common case (`{"$path+": ["children"]}`) never
  needs it.
- `maxDepth` remains mandatory on both forms (`PROGRAM_ERROR` if absent, unchanged
  from v0.0.1). Exceeding it while the chain still resolves further raises
  `PATH_DEPTH_EXCEEDED` (new — see section 9) rather than the shape-recursion
  `SHAPE_DEPTH_EXCEEDED`, so a violation record no longer conflates two structurally
  different overruns under one code.

## 7. Targets (revises v0.0.1 section 7 — adds one form)

Adds, alongside `$path`/`$each`/`$discriminator`/`$indexed`:

- `{"$path*": step, "from": basePath, "maxDepth": n}` / `{"$path+": ...}` — every
  path in the closure (section 6) is its own focus node. This is what lets a package
  validate every node of a recursive structure (a `Category` tree, a linked list)
  directly, without a separately hand-written recursive shape for the sole purpose
  of walking it.

Everything else in section 7 (targets run in declared order, index rebuilds happen
first per section 3, gate vs. report modes) is unchanged.

## 9. Errors (revises v0.0.1 section 9)

| Category | Raised when | New codes |
|---|---|---|
| `EXECUTION_ERROR` | (existing category) | `PATH_DEPTH_EXCEEDED` (new — a `$path*`/`$path+` closure resolves beyond its declared `maxDepth`; distinct from `SHAPE_DEPTH_EXCEEDED`, which remains scoped to shape/`extends` recursion) |
| `SHAPE_ERROR` | (existing category, extended) | malformed `relations` entry; `cardinality: "one-to-one"` uniqueness violation on `from.field` at build time |

Everything else in v0.0.1 section 9 — the pipeline order, `SHAPE_MISMATCH`,
`DANGLING_REFERENCE`, `SHAPE_DEPTH_EXCEEDED` itself — is unchanged. Section 10's
report format is unchanged; `code` may now additionally be `PATH_DEPTH_EXCEEDED`.

## 12. Where I am least sure (revises v0.0.1 section 12)

Carried over unchanged from v0.0.1: single-inheritance-only `extends`;
`$discriminator` as the sole stand-in for `sh:targetClass`; `SHAPE_DEPTH_EXCEEDED` as
a violation versus a hard error; message templating. Item 3 (index-rebuild cost in a
hot loop) is resolved by section 3 above and dropped from this list. Two new items
this revision introduces:

1. **Path closures and self-referential `relations` overlap.** A tree of
   same-shaped nodes (parent/child within one array) can be modeled either as a
   `$path*`/`$path+` closure over a pointer-style field, or as a `relations` entry
   with `from.path == to.path`. Both exist now, and it isn't decided which is
   canonical for that case, or whether offering two mechanisms for the same shape of
   problem is a real cost worth resolving before more packages get written against
   either one.
2. **Redundant declarations aren't rejected.** Nothing stops an author from using
   the inline `reference` shorthand on a field *and* declaring an overlapping
   `relations` entry for the same pair of collections — both build their own
   anonymous index over the same data. Not incorrect, just wasteful, and a future
   conformance suite likely wants a lint-level warning here rather than leaving it
   silent indefinitely.

## 14. Fixtures added this revision (shaxon-v0.1.1-fixtures.json)

- `check-in-loop-must-charge-index-rebuild-steps` — forces the section 3 cost rule;
  fails under a "rebuilds are free" reading, passes under "one step per element."
- `path-plus-closure-target-reports-violation-mid-chain` — `$path*` as a section-7
  target over a 4-node linked structure, `report` mode, one violation at depth 2.
- `relations-block-desugars-to-forward-reference-and-inverse-lookup` — one
  `relations` entry validated in both directions: forward `reference` checks on
  every order, reverse `$inverse` lookup by customer id.

What section 14 of shaxon-v0.0.1.md said is still needed before any of this is real
still applies in full — an index builder, shape evaluator, target resolver, and
path-expression evaluator, now additionally handling closures-as-targets and
`relations` desugaring. These three fixtures are the first fixtures such an
interpreter would need to pass; they are not yet evidence that it can.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
