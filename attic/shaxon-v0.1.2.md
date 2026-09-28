# Shaxon v0.1.2: extends merge semantics, reference-in-array reporting, tree-modeling guidance, and redundant-declaration detection

Status: proposal, unimplemented — still no reference interpreter (see shaxon-v0.0.1.md
section 14). This revision adds shaxon-v0.1.2-fixtures.json (3 fixtures); the same
caveat as v0.1.1 applies — these pin down intended behavior, they don't verify it.

This is a delta on shaxon-v0.1.1.md. Everything not named in the changelog below is
unchanged.

Format version: `"shaxon": "1.2"`. A "1.1"-only runtime must reject a "1.2" package
with `VERSION_ERROR`. A "1.2" runtime runs a declared-"1.1" or declared-"1.0" package
exactly as that package's own version specifies — in particular, the new
redundant-declaration check in section 4c only applies to a package that itself
declares `"shaxon": "1.2"`; an older package doesn't retroactively become invalid
just because the runtime reading it also knows 1.2.

## Changelog

1. **Section 4 (shapes)** — the `extends` merge rule is generalized from "closed is
   AND'd, fields union" plus an unspecified "etc." into one rule covering every
   mergeable member, with a worked table.
2. **Section 4b (new)** — guidance on `$path+`/`$path*` versus a self-referential
   `relations` entry for modeling a tree, resolving v0.1.1 section 12 item 1.
3. **Section 4c (extends v0.1.1 section 4a)** — a `SHAPE_ERROR` check at package load
   for a `reference` shorthand and a `relations` entry that build structurally
   identical indices, resolving v0.1.1 section 12 item 2.
4. **Section 5 (reference kind)** — a worked example of `reference` inside an array,
   plus the reporting-granularity rule for dangling elements in `gate` vs `report`
   mode.
5. **Section 9/10 (errors, report)** — fixed message text for structural violations
   (`DANGLING_REFERENCE`, `SHAPE_DEPTH_EXCEEDED`, `PATH_DEPTH_EXCEEDED`), needed to
   make the section 5 fixture's expected output well-defined rather than
   implementation-specific.

Nothing in v0.0.1 section 12's original list (single-inheritance-only `extends`,
`$discriminator` as sole `sh:targetClass` stand-in, `SHAPE_DEPTH_EXCEEDED` as
violation vs. hard error, message templating) is touched this round — see the
updated list at the end. This revision resolves exactly the two items v0.1.1 itself
introduced, and introduces none new.

## 4. Shapes — the `extends` merge rule, fully specified (revises v0.0.1 section 4)

v0.0.1 stated the rule for two members (`closed`, `fields`) and gestured at the rest
with "required/and etc." One principle covers all of them:

**Every `extends`-mergeable member accumulates using whatever aggregation is natural
for that member, unless the child marks that member `"override": true`, which
replaces instead of accumulating.**

| Member | Aggregation | Override behavior |
|---|---|---|
| `closed` | AND — most restrictive of parent/child wins | `"override": true` makes the child's own value win outright (unchanged from v0.0.1) |
| `fields` | union; a name collision is `SHAPE_ERROR` | per-field `"override": true` replaces the parent's field definition (unchanged from v0.0.1) |
| `required` | union — required in either parent or child means required in the merged shape | none needed; union only ever gets stricter |
| `and` | concatenate parent's list then child's | `"override": true` on the child's `and` replaces the parent's list instead of concatenating |
| `or` | concatenate parent's list then child's, then apply "at least one" over the combined pool | `"override": true` replaces instead of pooling |
| `xone` | concatenate parent's list then child's, then apply "exactly one" over the combined pool | `"override": true` replaces instead of pooling |
| `not` | parent's and child's `not` targets both apply — focus must satisfy neither (equivalent to AND-ing the two negations) | `"override": true` on the child's `not` replaces the parent's instead of adding to it |
| `check` | parent's and child's `check` are ANDed — focus must pass both | `"override": {"with": ..., "expr": ...}` on the child replaces the parent's `check` instead of ANDing |

One consequence is worth stating explicitly rather than leaving it to be discovered:
**pooling `xone` can turn a focus node that satisfied the child's own list into a
violation**, because "exactly one" is evaluated over the merged pool, not the
child's list alone. A focus node matching exactly one of the child's own
alternatives, plus one more of the parent's, now matches two — and fails. This
isn't a bug in the merge rule; it's what "exactly one, generalized to inherited
alternatives" has to mean. `check-in-loop...` — see
`extends-pools-xone-alternatives-not-just-childs-own` in
shaxon-v0.1.2-fixtures.json — exercises exactly this: a shape with a single-element
`xone` passes on its own, then fails once it `extends` a parent contributing a
second alternative the same focus also matches.

## 4b. Modeling a tree: `$path+`/`$path*` versus a self-referential `relations` entry (new — resolves v0.1.1 section 12, item 1)

On inspection this wasn't really one choice between two equally-valid mechanisms —
the two apply to two different data shapes:

| Data shape | Use |
|---|---|
| Genuinely nested — a child is a literal descendant in the JSON (`children: [...]` embedded under its parent) | `$path+`/`$path*` closure (section 6) |
| Flat — one array of same-shaped elements related by an id field (an adjacency list: `{"id": 2, "parentId": 1}`) | `relations` with `from.path == to.path` |

`$path+` has no way to reach an ID-linked parent in a flat array — its `innerStep`
is a relative *path*, and there is no structural path from a node to a same-array
sibling without going through an index. A genuinely nested tree needs no index at
all — you just walk it. In the ordinary case, the data's own shape decides which
mechanism applies; there's nothing to make canonical.

The real overlap is narrower: a dataset denormalized **both** ways at once (nested
*and* carrying redundant parent-id fields). For that case only: neither mechanism
is canonical over the other; use whichever one `program` actually navigates by for
its own purposes, and add a `check` asserting the two views agree — every node
reachable via the `$path+` closure from a root is also reachable via the
`relations`-based `$inverse` chain, or vice versa. That turns "pick a winner" into
an assertion of consistency, which is what `check` already exists to express, not
a case for new machinery.

## 4c. Redundant declarations (extends v0.1.1 section 4a — resolves v0.1.1 section 12, item 2)

Two declarations that would each build an index over the identical `(path, key)`
pair are a `SHAPE_ERROR` at package load — the same class of static check as the
existing duplicate-index-key rejection, just applied across declaration forms
instead of within one:

- An inline `{"kind": "reference", "of": ..., "by": [...]}` shorthand whose
  normalized `(of, by)` matches the `to` side of a `relations` entry — write
  `{"kind": "reference", "relation": name}` instead.
- Two `relations` entries whose `to` sides normalize to the same `(path, key)` —
  point every consuming field at one of them.

This is pure static comparison of normalized path segments and key field names, no
data inspection needed — decidable at the same "parse, version, schemas and shapes
and indices (all static)" pipeline stage v0.0.1 section 9 already checks
malformed shapes and indices at.

Deliberately **not** extended to a standalone named `indices` entry that happens to
match a relation's derived index — that's the general escape hatch, and an author
reaching for it explicitly is making a considered choice the shorthand-collision
case isn't.

See `redundant-relation-and-shorthand-rejected-at-load` in
shaxon-v0.1.2-fixtures.json.

## 5. The `reference` node kind — arrays and reporting granularity (revises v0.0.1 section 5)

v0.0.1 only showed `reference` on a scalar field. It composes under the existing
grammar exactly the way any other item shape does — no new syntax:

```json
"tagIds": {
  "kind": "array",
  "items": { "kind": "reference", "index": "tagsById" }
}
```

**Reporting rule**, extending the array-iteration convention already used
everywhere (`for`, `$each`, index builds all snapshot-iterate in array order):

- `report` mode: every dangling element gets its **own** violation with its own
  `focusPath` (e.g. `["input", "order", "tagIds", 2]` for the third element) — three
  bad entries in one field produce three violations, not one aggregate finding.
- `gate` mode: the **first** dangling element in array-index order raises
  `DANGLING_REFERENCE` and aborts, exactly like a gate target aborts on its first
  violation anywhere else.
- The violation's `shape` field names the nearest enclosing **named** shape being
  checked (the one passed to `check`/`validate`), not the inline, unnamed
  `reference` item — consistent with v0.0.1 section 10's own example, where
  `"shape": "DiscountLine"` names the shape passed at the call site, not some
  constituent of it.

See `array-of-references-reports-one-violation-per-dangling-element` in
shaxon-v0.1.2-fixtures.json.

## 9/10. Fixed structural violation messages (small addition, needed for the above)

Section 10 already says a structural finding (`DANGLING_REFERENCE`,
`SHAPE_DEPTH_EXCEEDED`, `PATH_DEPTH_EXCEEDED`) carries a non-null `code`, unlike an
author-written `check`/combinator failure, whose only identity is its `message`.
What it didn't say: a structural finding's `message` is **fixed by the runtime**,
not author-configurable — the parallel fact that makes `code` meaningful as the
identity for these:

| Code | Fixed message |
|---|---|
| `DANGLING_REFERENCE` | `"referenced value is not present in the named index"` |
| `SHAPE_DEPTH_EXCEEDED` | `"shape recursion exceeded maxShapeDepth"` |
| `PATH_DEPTH_EXCEEDED` | `"path closure exceeded maxDepth"` |

No new error category or code is added by this revision.

## 12. Where I am least sure (revises v0.1.1 section 12)

Unchanged from v0.0.1, still open — this revision doesn't touch them: single
inheritance only in `extends` (section 4's generalization specifies *how* a single
parent merges, not whether more than one should be allowed); `$discriminator` as
the sole stand-in for `sh:targetClass`; `SHAPE_DEPTH_EXCEEDED` as a violation versus
a hard error (and by extension, now, `PATH_DEPTH_EXCEEDED` too — splitting the code
answered "which budget overflowed," not "should overflowing be reportable or
fatal"); message templating.

v0.1.1's two new items — the `$path+`/self-referential-`relations` overlap and
undetected redundant declarations — are resolved by sections 4b and 4c above. This
revision introduces no new open items of its own.

## 14. Fixtures added this revision (shaxon-v0.1.2-fixtures.json)

- `extends-pools-xone-alternatives-not-just-childs-own` — a shape whose own `xone`
  list would pass a focus node in isolation fails once it `extends` a parent
  contributing a second matching alternative; forces the pooling rule in section 4.
- `array-of-references-reports-one-violation-per-dangling-element` — a 4-element
  reference array with two dangling entries, `report` mode, two independent
  violations at their own array-index `focusPath`s.
- `redundant-relation-and-shorthand-rejected-at-load` — a `relations` entry and an
  inline `reference` shorthand both targeting the identical `(path, key)`, rejected
  as `SHAPE_ERROR` before any input is processed.

As with v0.1.1, section 14 of shaxon-v0.0.1.md's list of what a real interpreter
needs still stands in full; these fixtures extend what such an interpreter would
need to reproduce, not what has been shown to run.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0
