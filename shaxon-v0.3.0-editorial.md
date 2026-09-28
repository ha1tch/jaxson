# Shaxon v0.3.0 — editorial log

This file accompanies shaxon-v0.3.0-core.md. It holds the decision log and
commentary that the core spec deliberately excludes. Nothing in this file is
normative; shaxon-v0.3.0-core.md is the sole source of truth for behaviour.

Note on filename: the requested name `shaxon-v0.30-editorial.md` is assumed to
be `shaxon-v0.3.0-editorial.md`, matching the project's `v0.<major>.<minor>`
convention.

## Confirmed design decisions in this revision

- **Index reuse is now mandatory**, not an optional optimisation. Previously a
  conformant runtime could charge the full rebuild cost every time and still
  conform; that made whether a package stays within its declared
  `limits.steps` depend on which runtime ran it, which is inconsistent with
  the step budget being a real, portable resource bound. §3.
- **`maxShapeDepth`'s trigger condition is broadened** to cover any shape that
  can recurse through ordinary named-shape field references, not only
  `extends` chains and bounded closure paths. The `Category`-referencing-
  `Category` worked example was already covered by the field-recursion case
  the mandatory-limit rule didn't actually name. §2.
- **Closure results (`$path*`/`$path+`) are now specified as ordered and
  duplicate-free**, not merely "a set." Order is breadth-first by hop count,
  ties broken by the existing array-index/sorted-key rule; a path reachable
  by more than one route is kept once, at its first-visit position. §6.
- **`reference` failure modes are split.** A non-scalar value is `TYPE_ERROR`,
  raised before any index lookup happens. Only a scalar value absent from the
  index is `DANGLING_REFERENCE`. Previously both cases collapsed into
  `DANGLING_REFERENCE`. §5.
- **Index key resolution is now fully specified.** A `key` expression must
  resolve to a scalar; a non-scalar result is `TYPE_ERROR`, an unresolved
  path is `MISSING_PATH`. No element is silently skipped during an index
  build. §3.
- **Multiple `report`-mode targets/checks that both omit `into`** now
  concatenate into one combined report, in execution order, rather than
  being left unspecified. §7.
- **`one-to-one` cardinality is clarified** as "each side individually
  unique," not a bijection requirement — it does not require every element
  on the `to` side to be referenced. §4a.
- **Combinator (`and`/`or`/`xone`) evaluation is now mandatorily minimal and
  order-fixed** — alternatives evaluate in listed order (parent's before
  child's) and stop as soon as the result is determined (`and` at the first
  failure, `or` at the first pass, `xone` at the second match). This closes
  the same class of gap the mandatory index-reuse decision closed: without
  it, the number of alternatives evaluated, and therefore steps charged,
  could differ between conformant runtimes for the same package and input.
  §4.
- **The stance section (§0) is reframed to lead with the positive claim.**
  It previously described determinism as the thing a naive SHACL/Jaxson
  integration would lose, which read as hedging on whether Shaxon itself
  has it. It now opens by stating plainly that Shaxon is deterministic end
  to end, then names the two rules that make that true by construction
  (mandatory index reuse, §3; mandatory combinator evaluation, §4). No
  behavioural change — this and the SHACL-equivalence rewording below are
  both framing-only edits.
- **The stance section's SHACL-equivalence claim is reworded** from "exactly
  as expressive as SHACL" to a description of what Shaxon takes from SHACL's
  model and what it deliberately changes (JSON tree instead of RDF graph,
  bounded computation instead of open SPARQL/JS). No behavioural change; this
  affects only how the design's scope is described. §0.

## Pending / open items (unresolved, carried into core §12)

1. Single vs. multiple inheritance for `extends`.
2. No structural (shape-based) alternative to `$discriminator` for
   population targeting.
3. Whether `maxShapeDepth`/`maxDepth` overflow is a data violation or a
   validator resource ceiling, and whether it should always be reportable.
4. Whether `message` may take a `$tpl` for interpolation.
5. `local.step`'s exact type and depth-accounting semantics.
6. Whether `$inverse` yields values or focus paths.

## Deferred (acknowledged as good ideas, not undertaken this revision)

- A formal semantic model (target → focus paths → shape → result) expressed
  as a small calculus, rather than prose. Treated as its own future body of
  work, to be done once a reference interpreter exists to keep it honest,
  not retrofitted under a documentation pass.
- Optional author-defined stable identifiers for `check`/combinator
  failures (distinct from the runtime-fixed `code` used for structural
  findings). New report-format surface area; would need its own proposal
  and fixtures.
- An expanded ("semantic law") fixture suite covering combinator
  interaction, reuse/invalidation, and closure/report ordering. Blocked on
  a reference interpreter existing to run fixtures against.

## Rejected (considered and not adopted)

- Renaming `extends` (e.g. to `refines`). The merge table already makes the
  aggregation semantics explicit; a rename doesn't change what a reader
  needs to learn.
- A `maxViolations`/violation-count cap. Would add a third resource
  dimension speculatively, ahead of any evidence that pathological report
  sizes are a practical problem.
- Making index-key/reference resource costs implementation-dependent by
  design. Superseded by making reuse mandatory instead (see above).
