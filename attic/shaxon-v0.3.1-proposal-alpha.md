# Shaxon v0.3.1 — proposal

Status: proposal, unimplemented. Targets exactly the items
`shaxon-v0.3.0-orientation.md` placed in category (2) with a settled-enough
mechanism to specify now, plus the internal ambiguities that document
excluded from its SHACL-parity classification because they aren't
comparisons to SHACL at all. Nothing in category (3) moves. Several
category (2) items are deliberately *not* taken up here — section 3 says
why, and what would have to be true first.

`"shaxon": "3.1"` — additive over `"shaxon": "3.0"`. No existing conformant
3.0 package changes behaviour under this proposal; every change either adds
an optional member, or states explicitly what a well-formed 3.0
implementation was already presumably doing (structural-finding severity;
step-counting `qualified`).

## 0. Method

Each confirmed change cites the orientation-document item it closes, so the
provenance chain (limitations → review → orientation → this proposal) stays
auditable. See §6 for the full traceability table.

## 1. Confirmed changes

### 1.1 Violation granularity, `constraintPath`, `constraintId`

**Closes:** orientation (2), "Per-constraint diagnostic granularity" — the
consensus P0 across Limitations §4, Review items 1/12–14/28.

Core §10's report record is extended:

```json
{
  "focusPath": ["input", "orders", 4],
  "constraintPath": ["status"],
  "shape": "Order",
  "constraintId": "ORDER_STATUS_ENUM",
  "severity": "violation",
  "message": "status must be one of draft, placed, shipped",
  "code": null
}
```

- **`constraintPath`** is a plain path, relative to `focusPath`, naming the
  member the finding is about. It is present whenever a finding is
  attributable to a specific field or `required` member, and **omitted**
  (not `null` — the jsonplate `$opt` convention Jaxson §5 already uses)
  otherwise.
- **`constraintId`** is an optional author-supplied stable string, settable
  on any `required` entry (as `{"name": "customerId", "id": "..."}`,
  replacing the bare string form when an id is wanted), any field shape, and
  any `check` block. `null` when the author didn't supply one, exactly
  paralleling `code`'s existing "`null` for author-written findings" rule.
  `code` keeps its existing role (a fixed runtime-defined string for
  structural findings); `constraintId` is never runtime-assigned.

**The granularity rule itself**, replacing the current single sentence in
§10 that says nothing about multiplicity:

| Failure | Violations produced |
|---|---|
| A `required` member absent | One violation per absent member. `focusPath` = the object; `constraintPath` = `[memberName]`. |
| A `fields` member present but failing its own primitive kind/keyword check (`type`, `enum`, `minLen`, etc.) | One violation. `focusPath` = the object; `constraintPath` = `[memberName]`. |
| A `fields` member whose value is itself checked against a named or inline nested shape | The nested shape check runs as its own target: its own `focusPath` (the member's path), its own violation if it fails. No `constraintPath` on the parent — this is unchanged from the existing `reference`-in-array convention (§5), now stated as the general rule for any nested shape, not only `reference`. |
| A shape's own `check` (a `$compute` island) fails | One violation. `focusPath` = the shape's own focus node; no `constraintPath`. |
| A shape's own `and`/`or`/`xone`/`not` combinator fails | One violation, for the whole shape, using the shape's own `severity`/`message`. **No per-alternative violations** — see §1.2, this is a consequence of mandatory minimal evaluation (Core §4), not an oversight. |

`gate` mode is unaffected in spirit: it still aborts on the *first* failure,
in the existing fixed order (`required` names in listed order, then `fields`
members in code-point order, then `check` — mirroring Jaxson §8's contract
validation order), raising `VALIDATION_ERROR`/`SHAPE_MISMATCH` for that one
failure. `report` mode is what changes: it now collects every independently
failing `required` member and `fields` member, not only the shape's overall
pass/fail.

### 1.2 Combinator evaluation and report granularity, made explicit

**Closes:** orientation (2), "consensus P0" cluster; specifically Review
item 4 ("Define combinator/report interaction").

New sentence, appended to Core §4's mandatory-minimal-evaluation paragraph:

> Because evaluation stops as soon as the combinator's result is
> determined, alternatives never evaluated produce no findings of their
> own. A failing `and`/`or`/`xone` therefore always produces exactly one
> violation for the enclosing shape — not one per failing alternative, and
> not one per unevaluated alternative. This is a direct consequence of
> mandatory minimal evaluation, not a separate report-model choice: the
> alternative genuinely was never checked, so there is nothing further to
> report about it.

This is a clarification, not a behaviour change — it states what mandatory
minimal evaluation already implied and removes the ambiguity Review item 4
flagged.

### 1.3 `$inverse` yields paths

**Closes:** Core §12 open question 6; Editorial — Pending item 6; Review
item 24.

Resolution: **paths**, not values. Core §6's `$inverse` row becomes:

> `{"$inverse": {"index": name, "key": operand}}` — every path the named
> `multi` index (or the `from` side of the named relation) maps that key
> to, in the index's own iteration order (§3). Each result is a focus path,
> composable with any other path-expression form, exactly as a closure
> result is (§6).

Rationale (Review item 24): the closure system is already path-producing,
targets already operate over paths, and `focusPath` is the report model's
canonical addressable identity. Making `$inverse` yield values instead would
put it in a different category from everything else path expressions touch,
for no offsetting benefit.

### 1.4 `$indexed` target semantics

**Closes:** Review item 25.

Core §7's `{"$indexed": name}` row is restated to remove the "entry vs.
value" ambiguity:

> `{"$indexed": name}` — every **source element's path** underlying the
> named index becomes its own focus node, not the index's key or value.
> For a non-`multi` index, elements are visited in ascending key order
> (the existing sorted-key convention). For a `multi` index, elements are
> visited grouped by key in ascending key order, then in array-index order
> within each key's matching set. Because every source element belongs to
> exactly one key, no element is visited twice.

This defines every index-derived target purely in terms of source paths,
matching how `$inverse` (§1.3) and ordinary closures (§6) already work —
the fix Review item 25 asked for.

### 1.5 `qualified` cost, clarified

**Closes:** Review item 26.

Core §4's `qualified` paragraph gets one added sentence:

> "Costs nothing new in termination terms" describes the termination proof,
> not the step charge. Each element checked against `qualified`'s `shape`
> is charged the ordinary shape-evaluation step cost (§9) — a `qualified`
> constraint over a million-element array charges for a million shape
> evaluations, exactly as if each had been checked individually. There is
> no exemption.

### 1.6 Structural-finding severity, fixed

**Closes:** Review item 27.

Core §10's fixed-messages table gets a `severity` column:

| Code | Fixed message | Fixed severity |
|---|---|---|
| `DANGLING_REFERENCE` | `"referenced value is not present in the named index"` | `violation` |
| `SHAPE_DEPTH_EXCEEDED` | `"shape recursion exceeded maxShapeDepth"` | `violation` |
| `PATH_DEPTH_EXCEEDED` | `"path closure exceeded maxDepth"` | `violation` |

Structural findings are never `warning`/`info` — only an author's `severity`
choice on a shape can produce those. This was presumably every existing
implementer's assumption; it is now written down.

### 1.7 `local.step`, narrow contract

**Closes:** Core §12 open question 5; Editorial — Pending item 5; Review
item 23.

Resolution, added to Core §6:

> `local.step` must resolve to a single path segment — a string or a
> non-negative integer, never a path array, an object, or any other
> non-scalar value. An expression that resolves `local.step` to anything
> else is `TYPE_ERROR`. Using `local.step` counts as exactly one hop toward
> `maxDepth`, the same as a fixed-name segment would.

Rationale (Review item 23): a `local.step` that could resolve to an
arbitrary path would reintroduce dynamic, effectively unbounded path
computation into a model whose entire cost story rests on "relative segment
array plus a bounded hop count." Locking it to a single scalar segment keeps
that story intact.

### 1.8 Validation purity, promoted to a stated invariant

**Closes:** Review item 10.

Limitations §10 already says `check`/`validate` are "read-only over `state`
by construction." That is a normative claim living in an informative
document. This proposal moves it into Core, as new §1 bullet (alongside the
other things that carry over from Jaxson unchanged):

> **Validation is a pure predicate.** `check` and `validate` never write to
> `state`, `output`, or any root, under any circumstance. Any data a
> validation needs must be derived by an explicit, ordinary `program` step
> that runs before the `check` that reads it. This is a design invariant,
> not an implementation consequence: it is what keeps "program state
> transition" and "predicate evaluation" from blurring into each other.

### 1.9 `value` in reports: decided against

**Closes:** the contested item orientation.md flagged between Limitations
§4 (proposed adding a `value` member) and Review item 16 (argued against
it).

Resolution: **no `value` member.** The offending value is always derivable
from `focusPath` (or `focusPath` + `constraintPath`, per §1.1) against the
original input, and copying it into every violation risks turning
validation of a large document into an uncontrolled duplication cost —
exactly the kind of hidden, unbudgeted operation Shaxon's cost model exists
to rule out elsewhere (indices, `qualified`, closures). This keeps the
report model consistent with the rest of the language's "paths, not
copies" posture, and supersedes Limitations §4's suggestion on this point.
This item now belongs to orientation.md's category (1), not (2) — the
document should be updated accordingly.

## 2. Additive, narrow, low-risk — also confirmed for 3.1

These three were not part of the P0/P1 diagnostic-model cluster, but each
has an uncontested, fully specified mechanism and no open blocker, so
there's no reason to hold them for a later minor version.

### 2.1 `enum` generalised to non-string kinds

**Closes:** Limitations §6.

```json
"priority": { "kind": "number", "enum": [1, 2, 3] }
```

`enum` becomes legal on `number`, `boolean`, and `null` kinds, with the same
semantics as the existing string form. Deliberately **not** extended to
`object`/`array` — deep-equality-against-a-list there is a different
feature (Limitations §6's own reasoning, unchanged).

### 2.2 Report-mode uniqueness

**Closes:** Limitations §7.

```json
"validate": [
  { "target": { "$each": ["input", "lines"] },
    "unique": { "field": "sku" },
    "severity": "warning",
    "message": "duplicate sku in order lines",
    "mode": "report" }
]
```

A `unique` entry, sibling to `shape` in a `validate`/`check` entry, checks
the named `field` (or, for non-object elements, the element itself) for
repeats across the target's population. Every element sharing a key with an
earlier one (population order = target visiting order, §7) produces a
violation at the declared `severity`; the first occurrence of each key is
never flagged. This does not replace the existing build-time uniqueness
check on a non-`multi` index (still `SHAPE_ERROR`, still fatal) — that
check is about an index's own well-formedness as a function; this is an
ordinary, `report`-mode-compatible data constraint over any population,
independent of whether an index happens to exist over the same field.

### 2.3 Named compute reuse

**Closes:** Limitations §8; Review item 19, "the most obvious safe future
feature."

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

A new package-level `computes` registry, parallel to `indices`/`relations`/
`shapes`. A `check` may reference one by name (`{"compute": name}`) instead
of writing `with`/`expr` inline. **Hard constraint, per Review item 19: a
named compute is semantically identical to an inline `$compute`** — same
closed operator set, same `with`-binding rule, same step cost, no reach
into `state`. This is authoring convenience only. A `computes` entry that
would be malformed as an inline `$compute` (unknown operator, wrong arity,
etc.) is `SHAPE_ERROR` at load, same as a malformed inline one is at the
point it's parsed.

## 3. Deferred — not in this revision

Each of these is a real, open item from orientation.md's category (2). None
is rejected; each has a stated reason it isn't ready, per the same
discipline the v0.3.0 editorial log used for its own "Deferred" list.

| Item | Orientation reference | Why not now |
|---|---|---|
| Per-constraint (per-field) severity | Limitations §3; Review item 15 | Review's objection wasn't only "granularity isn't defined yet" (§1.1 now defines it) — it's that severity of an `and`, of an inherited field, of `qualified`, of a nested shape failure all still need answers. §1.1 makes those questions answerable, but doesn't answer them. Revisit once granularity has shipped and been exercised against real packages. |
| Composable path expressions (`$altPath`/`$path*`/`$inverse` nesting) | Limitations §1; Review item 17 | No accounting rule exists for how `maxDepth` composes across nested forms (does a 3-segment sequence cost 1 hop or 3? does an unchosen alternative burn budget?). Proposing syntax before that question has an answer would bake in an arbitrary choice. |
| Structural targeting (`$conforms`) | Core §12 Q2; Limitations §2; Review item 21 | Orientation.md flagged this as the item most likely to be reclassified into category (3): letting a shape serve as its own target filter risks `target → shape → target → shape` recursion, reintroducing exactly the unboundedness the rest of the design rules out. Needs a real answer to that recursion before any syntax is worth proposing. |
| Named-profile string-pattern operator | Limitations §5 | A genuine want, distinct from open regex (which stays permanently out, §4 below). Simply out of scope for a tidy-up revision; candidate for the next minor version once a concrete profile set (fixed-width codes, one date grammar) is chosen. |
| Imports / cross-package shape reuse | Limitations §9; Review item 20 | Review's own recommendation: defer until a reference interpreter exists for the base language. Versioning, name collision, and dependency cycles are a module-system problem in their own right and shouldn't be designed against a language that hasn't been implemented yet. |

## 4. Reaffirmed non-goals — no change

Listed for completeness; orientation.md's category (3), unchanged by this
proposal:

- Open, arbitrary regular-expression matching (`sh:pattern` as SHACL has
  it) — Jaxson §8, Limitations §5.
- Fully general, parameterised, potentially recursive user-defined
  functions (`sh:function` in the SHACL-AF sense) — distinct from the
  narrow, no-new-capability `computes` registry in §2.3.
- Arbitrary, undeclared relational reach — any query not expressible
  through a declared `indices`/`relations` entry — Core §11, Limitations
  §11.
- The RDF/triple data model, unbounded property-path closure, open
  custom constraints (`sh:sparql`/`sh:js`), and validation-time mutation or
  inference — Core §0, §11; Limitations §10.

## 5. Error surface changes

All additive; no existing error condition changes meaning.

| Condition | Category | Code |
|---|---|---|
| Malformed `computes` entry (unknown operator, wrong arity, undeclared `$v`) | `SHAPE_ERROR` | — |
| Malformed `unique` declaration (missing `field`, `field` not a plain name) | `SHAPE_ERROR` | — |
| `local.step` resolves to a non-scalar-segment value | `EXECUTION_ERROR` | `TYPE_ERROR` (existing code, new trigger) |
| `enum` present on a `kind` other than `string`/`number`/`boolean`/`null` | `SHAPE_ERROR` | — |

## 6. Traceability

| Proposal item | Orientation category | Orientation entry | Primary sources |
|---|---|---|---|
| 1.1 Violation granularity | (2), P0 | "Per-constraint diagnostic granularity" | Limitations §4; Review 1, 12–14, 28 |
| 1.2 Combinator/report interaction | (2), P0 | same cluster | Review 4 |
| 1.3 `$inverse` → paths | (2)-adjacent open question | — | Core §12 Q6; Editorial Pending #6; Review 24 |
| 1.4 `$indexed` semantics | excluded (internal ambiguity) | — | Review 25 |
| 1.5 `qualified` cost | excluded (internal ambiguity) | — | Review 26 |
| 1.6 Structural severity | excluded (internal ambiguity) | — | Review 27 |
| 1.7 `local.step` | excluded (internal ambiguity) | — | Core §12 Q5; Editorial Pending #5; Review 23 |
| 1.8 Validation purity invariant | (1) | RDF/entailment substitution cluster | Limitations §10; Review 10 |
| 1.9 No `value` member | (1), resolved from contested | value-in-reports fork | Limitations §4 vs. Review 16 |
| 2.1 `enum` generalised | (2) | uncontested | Limitations §6 |
| 2.2 Report-mode uniqueness | (2) | uncontested | Limitations §7 |
| 2.3 Named compute reuse | (2) | best-supported after P0 | Limitations §8; Review 19, 34 |
| 3.* Deferred items | (2), unresolved blocker | flagged weaker rows | Limitations §1, §2, §3, §5, §9; Review 15, 17, 20, 21 |
| §4 Non-goals | (3) | unchanged | Core §11; Limitations §5, §10, §11 |

## 7. Before calling this stable

No reference interpreter exists yet for any Shaxon version (Core §13).
Per Review's closing recommendation, the right next step is not further
expansion but a fixture set exercising exactly the changes above —
particularly §1.1's granularity split (required vs. field vs. nested-shape
vs. combinator), §1.5's `qualified` step charge, and §2.2's uniqueness
population-ordering rule, since those three are the ones most likely to
tempt a divergent implementation shortcut.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0