# Shaxon v0.3.0 — orientation: classifying the SHACL gaps

This document collects every "SHACL can do X, Shaxon can't" item raised across
the project's own limitations log, an outside review of v0.3.0, and a working
critique session, and sorts each one into:

1. **Shaxon does it a different way, and that way is better** — a deliberate,
   permanent substitution, not a hole.
2. **Shaxon will plausibly do it better in a future version** — a real gap
   with a proposed or plausible path to closing it.
3. **Shaxon can't do it, and won't** — permanently out of scope, because
   closing it would cost Shaxon something the design isn't willing to spend.

Several items straddle two buckets because the sources themselves disagree,
or because a later source reframes an earlier one. Those are flagged rather
than forced into a single row.

## Sources and citation key

| Shorthand | Document | Cited by |
|---|---|---|
| **Core** | `shaxon-v0.3.0-core.md` | section (§) |
| **Limitations** | `shaxon-v0.3.0-limitations.md` | numbered item (its own `## N.` headings) |
| **Editorial** | `shaxon-v0.3.0-editorial.md` | list name (Confirmed / Pending / Deferred / Rejected) |
| **Jaxson** | `jaxson-v0.1.0-core-design.md` | section (§) |
| **Review** | `review.md`, the external reviewer's pass on v0.3.0 | numbered item (its own `# N.` headings), or "final section" for the unnumbered closing note |
| **Internal** | working critique from the review session that produced this document | marked explicitly — not part of the four project documents, included for completeness |

---

## (1) Shaxon does it a different way, and that way is better

These are not gaps. Each is a place where Shaxon deliberately forgoes a SHACL
capability, and every source that discusses the substitution treats it as an
improvement, not a compromise still being negotiated.

| SHACL capability | Shaxon's substitute | Sourced to |
|---|---|---|
| RDF/triple data model, blank nodes, IRI identity | JSON tree, path-based focus nodes; identity is "same index key," not "same IRI" | Core §0 (stance table, row 1), §11 |
| Open custom constraints (`sh:sparql`, `sh:js`) | Closed `$compute` islands over a closed operator table; no exception to the termination proof | Core §0 (row 3), §11; Jaxson §7–§8 (the closed operation set this is built on); Review item 31, names this the project's "closed-world philosophy" and calls it a strength |
| Relational reach via real graph edges, joinable arbitrarily at query time | Declared `indices` / `relations`, built once, invalidated by a decidable check against the mutation-path log | Limitations item 11; Editorial — Rejected ("Making index-key/reference resource costs implementation-dependent by design... Superseded by making reuse mandatory instead"); Review item 18, explicit: *"make declaration cheaper, not traversal more powerful"* |
| Unbounded property-path closure (`sh:path` zero-or-more has no required depth bound) | `maxDepth` mandatory and unconditional on every closure form (`$path*`, `$path+`) | Core §6, §11 (row 3 of "What Shaxon does not take from SHACL"); Review item 33, names "Boundedness" as one of Shaxon's four load-bearing invariants |
| Every constraint on a node evaluates independently; SHACL reports every failure | Mandatory minimal (short-circuit) combinator evaluation — `and` stops at first failure, `or` at first success, `xone` at second match | Core §4 ("mandatory, not an optional optimisation"); Editorial — Confirmed decisions, ties this explicitly to the index-reuse rule as "the same class of gap"; Review item 6, endorses the rule in full |
| `sh:rule` (SHACL-AF): entailment can write inferred data before validation runs | `check`/`validate` stay read-only over `state` by construction; any derivation is an explicit `program` step run first | Limitations item 10; Review item 10, wants this promoted from an implementation consequence to an explicit stated design invariant |

**Two qualifications, not corrections:**

- *Relational reach* is the same fact under two descriptions. Where the
  query is known when the package is authored, declared indices are a
  strictly better mechanism — bounded, portable cost, no hidden traversal.
  Where it genuinely isn't known in advance, Shaxon has no path there at
  all, full stop. That second framing is this table's twin entry in
  category (3), not a contradiction of this one.
- *Minimal combinator evaluation* only removes exhaustiveness at the
  combinator-branch level (an `and`/`or`/`xone` list). It has no bearing on
  whether a plain `fields`/`required` failure gets split into separate
  violations — that is a distinct, unresolved question, and it is the
  single most-agreed-on item in category (2) below. (Internal — this
  distinction was not stated explicitly in Limitations or Review, though
  both are consistent with it: Review item 6 endorses the combinator rule
  without qualification, and the granularity discussion in item 12 onward
  is entirely about `fields`, never about combinator branches.)

---

## (2) Shaxon will plausibly do it better in a future version

Real gaps, each with either a proposed mechanism or an argued-plausible
path to one.

| SHACL capability | Proposed Shaxon path | Sourced to | Status |
|---|---|---|---|
| Per-constraint diagnostic granularity + stable constraint identity (`sh:sourceConstraintComponent`, `sh:resultPath`) | Sub-`focusPath` down to the failing field; a `constraintId` distinct from the runtime-fixed `code` | Limitations item 4; Review items 1 ("*the atomic unit of a Shaxon validation finding*"), 12–14, 28; Internal (raised independently in the prior critique as the report model's central open question) | **Consensus P0.** Every source that touches the report model converges here; Review's closing section calls it "the most important conceptual hole left by the current version" |
| Per-constraint severity (`sh:severity` on individual constraints) | Per-field `severity` override, mirroring the `"override": true` mechanism `extends` already uses | Limitations item 3; Review item 15 | Real, but Review explicitly wants it sequenced **after** the granularity item above — severity of an `and`, of an inherited field, of `qualified`, all need answers first |
| `sh:in` on non-string RDF terms | `enum` generalised to `number` / `boolean` / `null` kinds (not `object`/`array`, deliberately) | Limitations item 6 | Narrow, low-risk, uncontested in Review |
| Report-mode uniqueness / functional-dependency constraint | A `qualified`-adjacent, target-level `unique` declaration that produces a `report`-mode finding at a chosen severity instead of aborting with `SHAPE_ERROR` | Limitations item 7 | Proposed mechanism exists; not discussed in Review, no objection on record |
| `sh:function` (SHACL-AF), narrowly | Named, package-level `$compute` reuse — condition: must stay semantically identical to inline `$compute`, "not a new capability" | Limitations item 8; Review item 19, calls it "the most obvious safe future feature"; Review item 34 lists "named compute reuse" as the one exception to its own "don't add yet" list | Best-supported item after the granularity P0 |
| `sh:pattern`, the underlying *need* (SKU/date-shaped strings) | A closed, named-profile pattern operator (fixed grammar — e.g. a specific date format, fixed-width alphanumeric codes), not open regex | Limitations item 5; Jaxson §8 (states the "named profile, fixed grammar" shape of the eventual fix) | The narrow substitute is (2); the actual SHACL feature it stands in for is permanently (3) — see below |
| `sh:targetShape` / `sh:target` bound to a shape (structural population targeting) | A `{"$conforms": "ShapeName"}` target form, filtering a declared population by shape-satisfaction rather than a tag field | Core §12, open question 2; Limitations item 2 | **Weaker than it looks** — see flag below |
| Composable property-path algebra (`$altPath`/`$path*`/`$path+`/`$inverse` nested inside one another) | A path-expression grammar allowing nesting, plus a rule for how `maxDepth` composes across nested forms | Limitations item 1 | **Most speculative item on this list** — see flag below |
| `owl:imports` / cross-document shape reuse | A package-reference mechanism, pulling named `shapes`/`indices`/`relations` from another package into the current one's registries | Limitations item 9; Review item 20 | Legitimate, but Review explicitly wants it deferred until the base language has a working reference implementation — versioning, name collision, and dependency cycles all need designing first |

**Contested within this bucket:** Limitations item 4 proposes adding a
`value` member (the offending value) to violation records. Review item 16
argues against it — large JSON values would turn diagnostics into an
uncontrolled duplication cost, and it cuts against the language's existing
preference for paths over copied data; Review would rather derive the value
from `focusPath` on demand. **If Review's version wins, this specific
sub-item moves to category (1)** — "does it better by never copying" — not
(2). This should be resolved explicitly rather than left to default one way.

**Two items flagged as weaker than their table row suggests:**

- **Structural targeting (`$conforms`).** Review item 21 raises a problem
  Limitations item 2 doesn't: letting a shape serve as its own target filter
  means target resolution runs shape evaluation, which can recurse
  (`target → shape → target → shape`) — precisely the unboundedness Shaxon
  exists to rule out. This may not be a "not yet" so much as a "not without
  a real answer to that specific recursion," which would put it in
  category (3) instead. Treat it as the item most likely to be
  reclassified once someone actually attempts to spec it.
- **Composable path algebra.** Review item 17 raises an unsolved accounting
  problem: nesting `$path*` inside `$altPath` inside `$inverse` makes
  `maxDepth` stop having an obvious meaning (does a three-segment sequence
  cost one hop or three? does an unchosen alternative burn budget? does a
  nested closure need its own budget?). Limitations item 1 names the same
  problem itself, in the "What Shaxon would need" paragraph. Nobody across
  the sources has an answer yet. This is a "yes, once the accounting problem
  is solved" item, not a scheduled one — and could stall into category (3)
  if that problem turns out to be unsolvable without abandoning the current,
  simple `innerStep` + `maxDepth` cost model.

---

## (3) Shaxon can't do it, and won't

| SHACL capability | Why it's permanent |
|---|---|
| `sh:pattern` as SHACL actually has it — open, arbitrary regular expressions | Rejected outright at the Jaxson layer (§8: "Regex engines disagree at the edges. If it is added, it is a named profile with a fixed grammar") and carried forward unchanged into Shaxon (Limitations item 5). The category-2 named-profile operator answers the *need*; it is not this feature, and open regex is not coming back into scope. |
| `sh:function` in its full generality — arbitrary, parameterised, potentially recursive user-defined functions | What's actually proposed in category (2) is deliberately narrower: named reuse of the existing closed `$compute`, explicitly "not a new capability" (Limitations item 8). Review item 34 separately and explicitly lists "generalized `$compute` functions" among what it would *not* add, distinguishing this from the narrower named-reuse idea it does endorse. |
| Genuinely ad hoc, undeclared relational reach — a join or aggregate nobody named when the package was written | Definitionally incompatible with a declared, bounded execution model decided at package-load time, not a missing feature awaiting design work. Editorial — Rejected, and Review item 18 ("Any narrowing of this gap should come from making index/relation declaration cheaper or more automatic, not from adding an escape hatch that reintroduces open-ended traversal") both treat this as closed. Same underlying fact as the category-1 relational-reach entry, viewed from the "but what if the query really is unknown in advance" angle rather than the "most queries are knowable in advance" angle. |

---

## The organising principle behind this split

Review's item 38 makes a point worth keeping in view whenever a new
candidate feature comes up: most of what lands in categories (2) and (3) —
path algebra, `$conforms`, uniqueness-as-query, named functions, imports,
rules — are not six unrelated omissions. They are six places where SHACL
assumes a live graph that can be queried on demand, and Shaxon assumes a
finite, declared execution model fixed at package-load time. The question
worth asking of any future addition isn't "can we technically reproduce this
SHACL feature," it's "does closing this gap require the live-graph
assumption, or can it be done as one more declared, bounded thing."
Everything in category (1) already answered that question with "declared
and bounded, and better for it." Category (3) answered it with "no, this one
needs the live graph." Category (2) is exactly the set where the answer
isn't settled — which is also why structural targeting and path composition,
the two shakiest rows in that table, are the two where the live-graph
assumption is creeping back in hardest.

---

## Excluded from this classification

A number of items raised in Review are real and worth tracking, but are not
"SHACL can do X, Shaxon can't" comparisons, so they're deliberately left out
of the tables above:

- Whether `$inverse` yields paths or values (Review item 24), the exact
  focus-path semantics of `{"$indexed": name}` (Review item 25), the
  step-cost of `qualified` (Review item 26), the severity of structural
  findings like `PATH_DEPTH_EXCEEDED` (Review item 27), and the precise
  type of `local.step` (Review item 23) — these are internal specification
  ambiguities, not places where Shaxon falls short of a SHACL capability.
- Single- vs. multiple-parent `extends` (Core §12, open question 1;
  Editorial — Pending item 1) — SHACL doesn't have a direct analogue to
  Shaxon's field-merging inheritance, so this isn't a SHACL-parity question
  at all. (Internal note: multi-shape composition without field merging is
  already available via the `and` combinator, which covers most of what a
  second `extends` parent would be reached for.)

---

*This document should be treated as a living index: when a category-2 item
gets a settled mechanism, or a category-1 substitution gets revisited, move
the row rather than leave two documents disagreeing about where something
stands.*

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0