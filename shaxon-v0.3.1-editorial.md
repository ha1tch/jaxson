# Shaxon v0.3.1 — editorial log

This file accompanies shaxon-v0.3.1-core.md. It holds the decision log and
commentary that the core spec deliberately excludes. Nothing in this file is
normative; shaxon-v0.3.1-core.md is the sole source of truth for behaviour.
This is a new, parallel log for this revision only — it does not restate or
supersede shaxon-v0.3.0-editorial.md, which remains the record of what
changed between v0.3.0 core and the earlier state.

Provenance chain for this revision: shaxon-v0.3.0-limitations.md → a
third-pass external review (filed as `attic/review.md`) → shaxon-v0.3.0-
orientation.md (classified every item into three buckets) → shaxon-v0.3.1-
proposal.md (specified everything orientation.md judged settled enough) →
shaxon-v0.3.1-core.md (this revision's clean spec).

## Confirmed design decisions in this revision

- **Violation granularity is now fully specified.** The report format gains
  `constraintPath`, `constraintId`, and `kind`, and section 10 gains an
  exhaustive table stating exactly how many violations each kind of failure
  produces (one per absent `required` member, one per failing field, the
  nested shape's own violation for a nested-shape field, one per failing
  `check`, one per failing combinator — never per alternative — one per
  `unique` repeat). This was the single most-agreed-on item across every
  source in the provenance chain, named in the third-pass review as "the
  most important conceptual hole left by the current version." §10.
- **Combinator/report interaction stated explicitly.** A failing `and`/`or`/
  `xone` produces exactly one violation for the enclosing shape — a direct
  consequence of mandatory minimal evaluation (already normative in v0.3.0),
  not a new choice, but previously unstated as a report-model fact. §4, §10.
- **`$inverse` resolved to yield paths, not values.** Matches how closures
  and targets already work; composes with every other path-expression form
  the same way a closure result does. §6.
- **`$indexed` target semantics fully specified**, in terms of source-element
  paths rather than index keys or values, with an explicit visiting order
  for both `multi` and non-`multi` indices. §7.
- **`qualified`'s step cost clarified**: "costs nothing new in termination
  terms" was never a step-cost exemption; each element checked against
  `qualified`'s `shape` is charged the ordinary shape-evaluation cost. §4.
- **Structural findings' severity and `kind` fixed.** `DANGLING_REFERENCE`,
  `SHAPE_DEPTH_EXCEEDED`, and `PATH_DEPTH_EXCEEDED` are always `severity:
  "violation"` and always `kind: "structural"`, never author-configurable.
  §10.
- **`local.step` given a narrow, closed contract**: must resolve to a single
  scalar segment (a string or non-negative integer), never a path or other
  non-scalar value; counts as exactly one hop toward `maxDepth`. Closes the
  weakest-specified primitive flagged across two independent reviews. §6.
- **Validation purity promoted to a stated invariant**, alongside three
  others (determinism, boundedness, explicit reach) that were each already
  enforced somewhere in the spec but had never been named as a set. §0, §1.
- **No `value` member in violation records.** Decided against, on the
  grounds that copying arbitrary focus-node data into every violation risks
  an uncontrolled duplication cost — the same reasoning that governs
  `qualified` and index costs elsewhere. The offending value is always
  derivable from `focusPath`/`constraintPath` against the original input.
  This closes a contested item the v0.3.0 orientation document had flagged
  as unresolved between two of its own sources. §10.
- **`message` confirmed to stay a static string** — no `$tpl` interpolation
  — for the same reason as the `value` decision above: allowing
  interpolated focus-node data into `message` would reopen the same
  duplication-cost door through a different field. §12 (closed, removed).
- **`extends` confirmed single-parent, and the question closed** (not
  merely deferred): composing shapes without merging fields is already
  `and: [...]`; nothing in the provenance chain produced a case actually
  requiring a second `extends` parent. §4, §12 (closed, removed).
- **`maxShapeDepth`/`maxDepth` given "validation horizon" terminology**,
  adopted directly from the third-pass review's recommendation, without
  changing what happens when either is exceeded (still reportable, still a
  violation, never silently truncated). This resolves the open question by
  naming it rather than by changing behaviour. §2, §6, §12 (closed,
  removed).
- **`enum` generalised** to `string`/`number`/`boolean`/`null` kinds, not
  `object`/`array`. §4.
- **Report-mode uniqueness added** via a `unique` alternative to `shape` on
  a `validate`/`check` entry, with its own `severity`, gate/report rules,
  and optional `"id"`. Does not replace the existing build-time uniqueness
  check on a non-`multi` index, which is about the index's own
  well-formedness rather than an ordinary data constraint. §7, §9, §10.
- **Named compute reuse added** via a package-level `computes` registry,
  referenced from `check` by name. Hard constraint: semantically identical
  to inline `$compute` — authoring convenience only, never a new
  capability. §2, §4d.
- **One wording fix**: section 9's `EXECUTION_ERROR` description used
  "shape" in its ordinary-English sense immediately after the document
  reserves that word for a specific vocabulary term. Reworded to "a value
  of the wrong JSON type." No semantic change.

## Pending / open items (unresolved, carried into core §12)

1. Whether a population worth targeting can be identified only by
   structural shape (`$conforms`) rather than by an explicit discriminator
   field. This is the only item remaining from v0.3.0's original six open
   questions; the other five are all closed above. Two independent reviews
   flagged the same specific obstacle: a structural target form needs shape
   evaluation to run as part of target resolution, which risks
   `target → shape → target → shape` recursion — a real threat to
   boundedness, not merely an unwritten feature. Not proposed until that
   recursion has an answer.

## Deferred (real items, not undertaken this revision, each with a reason)

- **Per-field/per-constraint `severity`.** The objection was never only
  "granularity isn't defined yet" — this revision's granularity table
  answers that. It's that severity of an `and`, of an inherited field, of
  `qualified`, of a nested shape failure all still need answers of their
  own. Revisit once the granularity model has shipped and been exercised
  against real packages, not before.
- **Composable path expressions** (nesting `$altPath`/`$path*`/`$path+`/
  `$inverse` inside one another). No accounting rule exists for how
  `maxDepth` composes across nested forms. Proposing syntax before that
  question has an answer would bake in an arbitrary choice.
- **Named-profile string-pattern operator** (the closed, fixed-grammar
  substitute for open regex). A genuine want, simply out of scope for a
  tidy-up revision; candidate for the next minor version once a concrete
  profile set is chosen.
- **Imports / cross-package shape reuse.** Recommended deferral, from the
  provenance chain's own review: wait until a reference interpreter exists
  for the base language. Versioning, name collision, and dependency cycles
  are a module-system problem in their own right and shouldn't be designed
  against an unimplemented language.

## Reaffirmed non-goals — no change from v0.3.0

Listed for completeness; nothing here moved:

- Open, arbitrary regular-expression matching, as SHACL has it.
- Fully general, parameterised, potentially recursive user-defined
  functions (distinct from the narrow, no-new-capability `computes`
  registry added this revision).
- Arbitrary, undeclared relational reach — any query not expressible
  through a declared `indices`/`relations` entry.
- The RDF/triple data model, unbounded property-path closure, open custom
  constraints, and validation-time mutation or inference.

## Not addressed by this revision, on record elsewhere

Two substantial bodies of external material exist alongside this revision
and are deliberately not reflected in it:

- A second round of review (seven documents, kept outside this repository) checked
  against the live W3C SHACL 1.2 Core draft, going considerably further
  than the material this revision closes — dynamic node/value expressions,
  set algebra over graph-derived collections, a proposed unifying
  "compositional expression layer," and more. This is raw research
  material, not yet distilled into a proposal, and is out of scope for a
  tidy-up revision by the same reasoning that governs every deferred item
  above.
- A from-scratch delta/diff/merge algebra for graph-shaped state
  (kept outside this repository), which is a versioning/merging concern unrelated to
  validation and explicitly out of scope for the language itself.
