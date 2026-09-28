# Shaxon v0.3.1 — proposal

Status: proposal, unimplemented. Targets every item `shaxon-v0.3.0-
orientation.md` placed in category (2) with a settled-enough mechanism to
specify now, every remaining open question `shaxon-v0.3.0-core.md` §12 has
carried since before v0.3.0, and the internal ambiguities orientation.md
excluded from its SHACL-parity classification because they aren't
comparisons to SHACL at all (Review's `$inverse`/`$indexed`/`qualified`/
structural-severity/`local.step` points). Nothing in orientation.md's
category (3) moves. Several category (2) items are deliberately *not*
taken up here — §4 says why, and what would have to be true first.

This revision was checked twice: once against `review.md` and
`shaxon-v0.3.0-orientation.md` for what a first pass should close, and once
more against itself, since two of its own new mechanisms (§1.1's original
`required`-with-id form and §3.2's `unique`) had rough edges on first
draft that are fixed in what follows rather than left as a known issue.

## 0. Method and versioning

Each confirmed change cites the source item it closes, so the provenance
chain (limitations → review → orientation → this proposal) stays auditable.
See §7 for the full traceability table.

`"shaxon": "3.1"` is additive over `"shaxon": "3.0"`. No existing
conformant 3.0 package changes behaviour under this proposal — every change
either adds an optional member/registry, or states explicitly what a
well-formed 3.0 implementation was already presumably doing (structural-
finding severity; step-counting `qualified`). Core §2's version-
compatibility rule extends to: a runtime implementing `"3.1"` must also
correctly execute a package declaring `"1.0"`, `"1.1"`, `"1.2"`, `"2.0"`,
or `"3.0"`, exactly as each version specifies.

Note, since it's easy to conflate the two: the `shaxon` version string
(`"1.0"` through `"3.1"`) is the protocol's own version lineage, independent
of a specification document's filename revision (`v0.1.0`, `v0.3.0`,
`v0.3.1`, ...), which tracks editorial progress on an as-yet-unreleased
document and has no bearing on what a package must declare.

## 1. Confirmed changes

### 1.1 Violation granularity, `constraintPath`, `constraintId`, `kind`

**Closes:** orientation (2), "Per-constraint diagnostic granularity" — the
consensus P0 across Limitations §4, Review items 1/12–14/28; Review items
28–29 (the `kind` field, below).

Core §10's report record is extended:

```json
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
```

- **`constraintPath`** is a plain path, relative to `focusPath`, naming the
  member the finding is about. It is present whenever a finding is
  attributable to a specific field or `required` member, and **omitted**
  (not `null` — the jsonplate `$opt` convention Jaxson §5 already uses)
  otherwise.
- **`constraintId`** is an optional author-supplied stable string. It is
  settable on any field shape and any `check` block directly (an `"id"`
  member alongside the existing ones), and on a `required` member via a
  separate sibling map rather than by changing `required`'s own type:

  ```json
  "required": ["id", "customerId", "lines"],
  "requiredIds": { "customerId": "ORDER_NEEDS_CUSTOMER" }
  ```

  `required` itself stays a plain array of strings — a mixed array of
  strings and `{name, id}` objects would be the first heterogeneous array
  anywhere in the language, and would leave the `extends` merge rule for
  `required` ("union," Core §4) undefined for two parents naming the same
  member with different id-bearing forms. `requiredIds`'s keys must be a
  subset of `required`'s entries (a key naming a non-required member is
  `SHAPE_ERROR`); a member absent from `requiredIds` reports with
  `constraintId: null`. `requiredIds` merges under `extends` the same way
  `fields` does — union by key, per-key `"override": true` for a child
  replacing a parent's id.

  `constraintId` values are **not** required to be unique within a
  package — an author may reuse one id across constraints that represent
  the same logical rule (the same non-negativity check, written inline in
  two shapes). Tooling wanting per-rule aggregation should key on
  `constraintId` when present, and fall back to `(shape, constraintPath)`
  otherwise. `constraintId` is `null` when the author didn't supply one,
  exactly paralleling `code`'s existing "`null` for author-written
  findings" rule; `code` keeps its existing role (a fixed runtime-defined
  string for structural findings) and is never author-assigned.
- **`kind`** is `"structural"` whenever `code` is non-`null`, and
  `"constraint"` whenever `code` is `null`. It is mechanically derived,
  never independently authored, and exists so a consumer doesn't have to
  learn the `code == null` convention to tell a structural finding from an
  author-written one.

**The granularity rule itself**, replacing the current single sentence in
§10 that says nothing about multiplicity:

| Failure | Violations produced |
|---|---|
| A `required` member absent | One violation per absent member. `focusPath` = the object; `constraintPath` = `[memberName]`; `constraintId` from `requiredIds` if declared. |
| A `fields` member present but failing its own primitive kind/keyword check (`type`, `enum`, `minLen`, etc.) | One violation. `focusPath` = the object; `constraintPath` = `[memberName]`. |
| A `fields` member whose value is itself checked against a named or inline nested shape | The nested shape check runs as its own target: its own `focusPath` (the member's path), its own violation if it fails. No `constraintPath` on the parent — this is unchanged from the existing `reference`-in-array convention (§5), now stated as the general rule for any nested shape, not only `reference`. |
| A shape's own `check` (a `$compute` island) fails | One violation. `focusPath` = the shape's own focus node; no `constraintPath`. |
| A shape's own `and`/`or`/`xone`/`not` combinator fails | One violation, for the whole shape, using the shape's own `severity`/`message`. **No per-alternative violations** — see §1.2, this is a consequence of mandatory minimal evaluation (Core §4), not an oversight. |

`gate` mode is unaffected in spirit: it still aborts on the *first* failure,
in the existing fixed order (`required` names in listed order, then
`fields` members in code-point order, then `check` — mirroring Jaxson §8's
contract validation order), raising `VALIDATION_ERROR`/`SHAPE_MISMATCH` for
that one failure. `report` mode is what changes: it now collects every
independently failing `required` member and `fields` member, not only the
shape's overall pass/fail.

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
choice on a shape can produce those — and always carry `kind: "structural"`
per §1.1. This was presumably every existing implementer's assumption; it
is now written down.

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

### 1.8 Validation purity, and the four invariants stated together

**Closes:** Review item 10 (purity); Review item 33 (naming all four).

Limitations §10 already says `check`/`validate` are "read-only over `state`
by construction." That is a normative claim living in an informative
document. This proposal moves it into Core, as a new §1 bullet (alongside
the other things that carry over from Jaxson unchanged):

> **Validation is a pure predicate.** `check` and `validate` never write to
> `state`, `output`, or any root, under any circumstance. Any data a
> validation needs must be derived by an explicit, ordinary `program` step
> that runs before the `check` that reads it. This is a design invariant,
> not an implementation consequence: it is what keeps "program state
> transition" and "predicate evaluation" from blurring into each other.

Review item 33 names this as one of four principles the whole design
optimises for, but only this one had been promoted to a stated invariant.
The other three are each already enforced somewhere in the spec; they are
collected here as peers rather than left implicit:

> **The four invariants.**
> - *Determinism* — same package, same input, same result, same step
>   count, on every conformant runtime (§0), enforced by mandatory index
>   reuse (§3) and mandatory minimal combinator evaluation (§4).
> - *Boundedness* — no operation may secretly escape the step or depth
>   model (§11; Jaxson §9): every closure has a `maxDepth`, every recursive
>   shape has a `maxShapeDepth`, every loop iterates a finite snapshot.
> - *Explicit reach* — all relational reach goes through a declared
>   `index` or `relation`; nothing is discoverable at query time that
>   wasn't declared at load time (§3, §4a, §11).
> - *Validation purity* — `check`/`validate` never write to `state`,
>   `output`, or any root (above).

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
This item belongs to orientation.md's category (1), not (2) — the
document should be updated accordingly.

### 1.10 `extends`: single parent stays, and the question closes

**Closes:** Core §12 open question 1; Editorial — Pending item 1. Not
discussed in `review.md`; the argument is Internal, from the first review
pass.

SHACL has no direct analogue to Shaxon's field-merging `extends` — the
comparable SHACL pattern (a node conforming to several independent shapes)
is already what Shaxon's `and` combinator gives you. The only thing a
second `extends` parent adds over `and: [ShapeA, ShapeB]` is *merged*
fields — one flattened object schema instead of an AND of independently-
checked shapes — narrower than "inherit from two things," and no source
document has produced a case requiring it. Core §4 gains:

> `extends` takes a single parent name by design. Composing several shapes
> without merging their fields is `and: [...]`; reach for `extends` only
> when field-level merging into one object schema is the actual goal.

### 1.11 `maxShapeDepth`/`maxDepth`: "validation horizon," not behaviour change

**Closes:** Core §12 open question 3; Review item 22.

Review item 22 recommends resolving this by naming it, not by changing
what happens when it's crossed:

> `maxDepth` = validation horizon... would make it obvious that exceeding
> the horizon is part of the contract.

Adopted directly. `PATH_DEPTH_EXCEEDED`/`SHAPE_DEPTH_EXCEEDED` remain
reportable findings exactly as specified (§1.6 fixes their severity at
`violation`); what changes is vocabulary. Core §2 and §6 gain:

> `maxShapeDepth` and `maxDepth` are **validation horizons**: a declared
> boundary past which the language does not promise to look, not a claim
> that no further structure exists. Crossing one is reportable precisely
> because it means the document asked for more validation than the package
> declared itself willing to perform — the same category of fact as any
> other declared-and-checked constraint.

### 1.12 `message` stays a static string

**Closes:** Core §12 open question 4. Not discussed by name in `review.md`
within the material reviewed; the reasoning follows directly from §1.9 and
should have closed alongside it.

Resolution: **no**, `$tpl` is not permitted inside `message`. §1.9 rejected
a `value` member on the grounds that copying arbitrary focus-node data into
every violation risks an uncontrolled duplication cost. Allowing
`{"$tpl": "value was {status}"}` inside `message` reintroduces exactly that
risk through a different door — an interpolated message can embed
arbitrarily large focus-node data just as effectively as a `value` member
would. Keeping `message` static keeps the two decisions consistent.

## 2. Core semantic objects

**Closes:** Review item 37; connects to Editorial — Deferred ("A formal
semantic model... treated as its own future body of work").

Review item 37 explicitly does not want a formal semantics chapter before a
reference interpreter exists, and Editorial already deferred one for the
same reason. What item 37 asks for instead is small enough to add now — a
map of names to the sections that already define them, not new semantics:

> **Core semantic objects.**
> - **Path** — a JSON array of segments; the base addressing unit
>   (Jaxson §3).
> - **FocusPath** — a Path a target has resolved as the subject of one
>   shape check; the canonical identity of a finding (Core §7, §10).
> - **TargetResult** — the ordered, duplicate-free sequence of FocusPaths a
>   `target` expression produces before any shape runs (Core §7).
> - **ShapeResult** — the pass/fail (plus, on failure, severity/message/
>   constraintId) from checking one shape against one FocusPath (Core §4).
> - **Violation** — one report entry: FocusPath, optional ConstraintPath,
>   `kind`, `shape`, optional `constraintId`, `severity`, `message`, `code`
>   (Core §10, as extended by §1.1 above).
> - **Index** — a named, rebuildable map from a source array to scalar
>   key(s), consumed by `reference`, `relations`, `$inverse`, `$indexed`
>   (Core §3).
>
> Relationship, one paragraph: a `target` produces a `TargetResult`; each
> `FocusPath` in it is checked, producing a `ShapeResult`; each failing
> `ShapeResult` (or failing field/required member within it, per §1.1)
> produces one `Violation`. `Index` feeds the `reference`/`$inverse`/
> `$indexed` machinery that some `FocusPath`s and `Target`s are built from.

## 3. Additive, narrow, low-risk

Each of these has an uncontested, fully specified mechanism and no open
blocker, so there's no reason to hold them for a later minor version.

### 3.1 `enum` generalised to non-string kinds

**Closes:** Limitations §6.

```json
"priority": { "kind": "number", "enum": [1, 2, 3] }
```

`enum` becomes legal on `number`, `boolean`, and `null` kinds, with the same
semantics as the existing string form. Deliberately **not** extended to
`object`/`array` — deep-equality-against-a-list there is a different
feature (Limitations §6's own reasoning, unchanged).

### 3.2 Report-mode uniqueness

**Closes:** Limitations §7.

```json
"validate": [
  { "target": { "$each": ["input", "lines"] },
    "unique": { "field": "sku", "id": "LINE_SKU_UNIQUE" },
    "severity": "warning",
    "message": "duplicate sku in order lines",
    "mode": "report" }
]
```

A `unique` entry checks the named `field` (or, for non-object elements, the
element itself) for repeats across the target's population, in target-
visiting order (§7). Every element sharing a key with an earlier one
produces a violation at the declared `severity`; the first occurrence of
each key is never flagged. A `unique` entry may carry the same optional
`"id"` member `check` and field shapes do, feeding `constraintId`.

- A `validate`/`check` entry has **exactly one** of `shape` or `unique`;
  declaring both is `SHAPE_ERROR` at load. An author wanting both a shape
  check and a uniqueness check over the same target writes two entries,
  combined by the existing report-concatenation rule (§7).
- `gate` mode for `unique`: the first element, in target-visiting order,
  that repeats an already-seen key aborts with `VALIDATION_ERROR`/
  `SHAPE_MISMATCH` — the same "first offender in visiting order" rule
  `reference`'s gate mode already uses (§5).
- A `unique` violation follows §1.1's format: `kind: "constraint"`,
  `focusPath` = the repeating element's own path, no `constraintPath` (it's
  a whole-element finding), `constraintId` from the declaration's `"id"` if
  given, else `null`, `code: null`.

This does not replace the existing build-time uniqueness check on a
non-`multi` index (still `SHAPE_ERROR`, still fatal) — that check is about
an index's own well-formedness as a function; this is an ordinary,
`report`-mode-compatible data constraint over any population, independent
of whether an index happens to exist over the same field.

### 3.3 Named compute reuse

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
point it's parsed. Core §2's duplicate-key rule extends to name it:
"Duplicate keys anywhere, including inside `shapes`, `indices`, `relations`,
or `computes`, are a `PARSE_ERROR`."

## 4. Deferred — not in this revision

Each of these is a real, open item from orientation.md's category (2). None
is rejected; each has a stated reason it isn't ready, per the same
discipline the v0.3.0 editorial log used for its own "Deferred" list.

| Item | Reference | Why not now |
|---|---|---|
| Per-constraint (per-field) severity | Limitations §3; Review item 15 | Review's objection wasn't only "granularity isn't defined yet" (§1.1 now defines it) — it's that severity of an `and`, of an inherited field, of `qualified`, of a nested shape failure all still need answers. §1.1 makes those questions answerable, but doesn't answer them. Revisit once granularity has shipped and been exercised against real packages. |
| Composable path expressions (`$altPath`/`$path*`/`$inverse` nesting) | Limitations §1; Review item 17 | No accounting rule exists for how `maxDepth` composes across nested forms (does a 3-segment sequence cost 1 hop or 3? does an unchosen alternative burn budget?). Proposing syntax before that question has an answer would bake in an arbitrary choice. |
| Structural targeting (`$conforms`) | Core §12 Q2; Limitations §2; Review item 21 | Orientation.md flagged this as the item most likely to be reclassified into category (3): letting a shape serve as its own target filter risks `target → shape → target → shape` recursion, reintroducing exactly the unboundedness the rest of the design rules out. Needs a real answer to that recursion before any syntax is worth proposing. |
| Named-profile string-pattern operator | Limitations §5 | A genuine want, distinct from open regex (which stays permanently out, §5 below). Simply out of scope for a tidy-up revision; candidate for the next minor version once a concrete profile set (fixed-width codes, one date grammar) is chosen. |
| Imports / cross-package shape reuse | Limitations §9; Review item 20 | Review's own recommendation: defer until a reference interpreter exists for the base language. Versioning, name collision, and dependency cycles are a module-system problem in their own right and shouldn't be designed against a language that hasn't been implemented yet. |

## 5. Reaffirmed non-goals — no change

Listed for completeness; orientation.md's category (3), unchanged by this
proposal:

- Open, arbitrary regular-expression matching (`sh:pattern` as SHACL has
  it) — Jaxson §8, Limitations §5.
- Fully general, parameterised, potentially recursive user-defined
  functions (`sh:function` in the SHACL-AF sense) — distinct from the
  narrow, no-new-capability `computes` registry in §3.3.
- Arbitrary, undeclared relational reach — any query not expressible
  through a declared `indices`/`relations` entry — Core §11, Limitations
  §11.
- The RDF/triple data model, unbounded property-path closure, open
  custom constraints (`sh:sparql`/`sh:js`), and validation-time mutation or
  inference — Core §0, §11; Limitations §10.

## 6. Error surface changes

All additive; no existing error condition changes meaning.

| Condition | Category | Code |
|---|---|---|
| Malformed `computes` entry (unknown operator, wrong arity, undeclared `$v`) | `SHAPE_ERROR` | — |
| Malformed `unique` declaration (missing `field`, `field` not a plain name) | `SHAPE_ERROR` | — |
| A `validate`/`check` entry declares both `shape` and `unique` | `SHAPE_ERROR` | — |
| A `requiredIds` key names a member not present in `required` | `SHAPE_ERROR` | — |
| `local.step` resolves to a non-scalar-segment value | `EXECUTION_ERROR` | `TYPE_ERROR` (existing code, new trigger) |
| `enum` present on a `kind` other than `string`/`number`/`boolean`/`null` | `SHAPE_ERROR` | — |

One wording nit while touching Core's error table: §9's description of
`EXECUTION_ERROR` as covering cases "where index keys and reference values
resolve to the wrong shape" uses "shape" in its ordinary-English sense
immediately after the document reserves that word for a specific vocabulary
term. Reword to "resolve to a value of the wrong JSON type" — no semantic
change, just the kind of collision the project's own naming discipline
(`ual`, never `UAL`) exists to catch elsewhere.

## 7. Traceability

| Proposal item | Orientation category | Reference | Primary sources |
|---|---|---|---|
| 1.1 Violation granularity, `constraintPath`, `constraintId`, `kind` | (2), P0 | "Per-constraint diagnostic granularity" | Limitations §4; Review 1, 12–14, 28–29 |
| 1.2 Combinator/report interaction | (2), P0 | same cluster | Review 4 |
| 1.3 `$inverse` → paths | (2)-adjacent open question | — | Core §12 Q6; Editorial Pending #6; Review 24 |
| 1.4 `$indexed` semantics | excluded (internal ambiguity) | — | Review 25 |
| 1.5 `qualified` cost | excluded (internal ambiguity) | — | Review 26 |
| 1.6 Structural severity | excluded (internal ambiguity) | — | Review 27 |
| 1.7 `local.step` | excluded (internal ambiguity) | — | Core §12 Q5; Editorial Pending #5; Review 23 |
| 1.8 Validation purity + four invariants | (1) | RDF/entailment substitution cluster | Limitations §10; Review 10, 33 |
| 1.9 No `value` member | (1), resolved from contested | value-in-reports fork | Limitations §4 vs. Review 16 |
| 1.10 `extends` single-parent, stated | excluded (not a SHACL comparison) | Core §12 Q1 | Internal |
| 1.11 "Validation horizon" terminology | excluded (internal ambiguity) | Core §12 Q3 | Review 22 |
| 1.12 `message` stays static | excluded (internal ambiguity) | Core §12 Q4 | Internal (consistency with §1.9) |
| 2. Core semantic objects | — | — | Review 37; Editorial — Deferred |
| 3.1 `enum` generalised | (2) | uncontested | Limitations §6 |
| 3.2 Report-mode uniqueness | (2) | uncontested | Limitations §7 |
| 3.3 Named compute reuse | (2) | best-supported after P0 | Limitations §8; Review 19, 34 |
| §4 Deferred items | (2), unresolved blocker | flagged weaker rows | Limitations §1, §2, §3, §5, §9; Review 15, 17, 20, 21 |
| §5 Non-goals | (3) | unchanged | Core §11; Limitations §5, §10, §11 |

## 8. Before calling this stable

No reference interpreter exists yet for any Shaxon version (Core §13).
Every P0/P1 item Review numbered is closed above, every open question Core
§12 has carried since before v0.3.0 is closed, and the two structural rough
edges this document's own new mechanisms had on first draft — the
mixed-type `required` array, `unique`'s missing exclusivity/gate/report-
format rules — are fixed rather than left as known issues.

The right next step, per Review's own closing recommendation, is a
reference interpreter and a fixture set — not another round of expansion.
Priority fixtures: §1.1's granularity split (required vs. field vs.
nested-shape vs. combinator), §1.5's `qualified` step charge, and §3.2's
uniqueness population-ordering and gate-mode rule, since those three are
the ones most likely to tempt a divergent implementation shortcut.

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0