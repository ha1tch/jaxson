# `extends` narrows: layers 2 and 3

Updated: 2026-10-07

Status: proposal. Layer 1 is implemented and specified (core section 4, tracker
row G11, CHANGELOG "Unreleased"). This document records the work that is still
open, in the order it should be done, so that it can be picked up without the
conversation that produced it.

## 1. The principle

A shape that extends another may only narrow what it extends, except through a
loosening the specification names. Layer 1 made this true for `kind`, a
reference target, the primitive keywords, `items` and `qualified`. Before it, a
child lost all of these unless it restated them, which is a fail-open defect:
extending a `minLen` shape silently dropped the bound, and extending a
`reference` shape panicked at load.

What layer 1 did not settle is the rest of the merge table, which accumulates or
replaces by its own rules, and the statement of the principle in the
specification. That is layer 2. Layer 3 is the set of features the principle
makes possible or exposes, none of which is needed for correctness.

## 2. Layer 2: consistency (do next)

| ID | Item | Where | Done when |
|---|---|---|---|
| G11-2a | `severity` and `message` are inherited when the child does not state them; a lower severity needs an explicit override (option C) | `parse_shapes.go` (`parseShapeBody` sets `Severity` to `"violation"` when unstated, so an unstated value cannot be told from an explicit one; add `SeveritySet` and `MessageSet` beside `ClosedSet`), `mergeShapes`; core section 4 row | an extending shape reports with its parent's severity and message unless it states its own; an unmarked lower severity is a `SHAPE_ERROR`; fixtures for each direction, a chain and the override; a mutant for each |
| G11-2b | Rewrite the core section 4 merge table around the principle | `shaxon-v0.3.1-core.md` section 4 | every member of a shape has a row; each row says whether it narrows, accumulates or can loosen; the loosenings are listed in one place |
| G11-2c | A permanent property test: a child accepts only what its parent accepts, apart from the listed loosenings | new `pkg/shaxon/extends_property_test.go` | a seeded generator of parent and child pairs over the narrowing members, a fixed seed in the default run, a larger count behind a build tag registered in the dormant guards table (tracker, "Dormant guards") in the same change |
| G11-2d | Housekeeping | CHANGELOG, tracker row G11, `FIXTURES.md` group table and counts, `oracle_test.go` threshold, README and status counts, `fixtures_mutants.py` | the counts agree everywhere (a tree-wide sweep, tracker "Keeping this current") |

### 2.1 Severity and message

`severity` decides whether a finding flips `conforms`; `message` is the text
reported. Today the merged shape takes the child's value, and an unstated
severity becomes `"violation"`. Two consequences follow:

- A child extending a parent whose severity is `warning`, and stating none,
  reports `violation`. The report becomes stricter than the parent's.
- A child that states `warning` or `info` over a parent whose severity is
  `violation` stops the same finding from flipping `conforms`. This is a
  loosening, and a heavier one than it looks (section 2.1.2).

The first is the same defect as layer 1 and is fixed by inheritance. The second
is a decision, not a defect.

#### 2.1.1 Options for a child that lowers the parent's severity

| Option | Effect | Cost |
|---|---|---|
| A. The child's explicit severity wins | simple and predictable | a downgrade is silent, and it changes `conforms` (section 2.1.2); "severity" joins the list of loosenings with no marker at the point of use |
| B. Only an equal or more severe value is accepted | the principle holds with no exception | a child cannot downgrade a finding for one case; a lower value is a `SHAPE_ERROR` |
| C. A lower value needs `"override": true` on the severity (recommended) | the principle holds, and a downgrade is explicit where it is written, like every other loosening in the merge table | one more spelling to specify: `"severity": "warning"` plus the override marker, in the form the other override members use (core section 4) |

Recommendation: option C. Raising a severity, or restating it, needs no marker.
An unmarked lower value is a `SHAPE_ERROR`, which is option B's behaviour with a
way out. An explicitly empty `message` is treated as stated, so that a child can
clear the parent's text and fall back to the generated one. `message` carries no
such question: it never changes `conforms`.

#### 2.1.2 Why the SHACL comparison matters

In SHACL a severity is a label on a result; in Shaxon it is also a gate. So
lowering one costs less in SHACL than in Shaxon, and the question is harder
here. The table compares the two. SHACL columns come from the W3C SHACL 1.1
text (sections 2.1.4, 2.1.5 and 3.6.1.1, and the part on nested shapes) and the
1.2 Working Draft of 18 September 2026; the engine rows come from running pySHACL
0.40.1 and Apache Jena 6.2.0 on a shape of severity `sh:Warning` that the data
violates (2026-10-07). The same facts are in `shaxon-v0.3.1-limitations.md`
section 8, which is the document to link from outside the project.

| Aspect | SHACL | Shaxon today | Shaxon after G11-2a (option C) |
|---|---|---|---|
| Levels | `sh:Info`, `sh:Warning`, `sh:Violation`; any IRI can be a severity. 1.2 (draft) adds `sh:Trace` and `sh:Debug`, which are not constraint violations | `info`, `warning`, `violation`; nothing else | unchanged |
| Default | `sh:Violation` when `sh:severity` is not stated | `violation` | `violation` for a shape with no parent; the parent's value for one that extends and states none |
| Where it is declared | on a shape; every result the shape produces carries it | on a shape, same | same |
| Effect on `conforms` | none by severity in 1.1: `sh:conforms` is true only if validation produced no results, so a `Warning` or `Info` result makes it false. pySHACL and Jena agree: both returned non-conforming on a warning-only run, and pySHACL conforms only with its `allow_warnings` option. 1.2 draft: not confirmed | only `violation` findings make `conforms` false; `warning` and `info` are collected and never gate | unchanged |
| Effect on a gate | none: SHACL has no abort mode | a `gate` aborts only on a `violation` finding, of a shape or of a `unique` entry (core section 7; two fixtures) | unchanged |
| Severity on a `unique` or `validate` entry | none: no `unique` constraint; severity belongs to a shape | `unique` declares its own severity on the entry; a `validate` entry may not (the shape declares it, V6) | unchanged |
| Closed objects and `ignoredProperties` | `sh:closed` is a component of its shape, so its results carry the shape's severity | the extra-member findings carry the shape's severity | an extending shape's merged `closed` result carries its inherited or own severity |
| A failing shape reached through a combinator (SHACL `sh:node`, `sh:and`, `sh:or`, `sh:not`, `sh:xone`; Shaxon `and`, `or`, `xone`, `not`) | as the specification reads, the referencing shape yields the result, with its own severity | one finding for the shape, with the shape's own severity (core section 10) | unchanged |
| A failing shape reached through a field or `items` | not applicable: a property shape is itself a shape and reports with its own severity | the nested shape reports its own findings, each with its own severity | unchanged |
| Inheritance between shapes | none: there is no `extends`, and a shape that refers to another does not take its severity | none: the child's own only, an unstated value becomes `violation` | the child inherits the parent's severity and message when it states none |
| A shape lowering another's severity | not applicable; lowering a label leaves `conforms` false | the child's value wins, silently, and `conforms` can become true | needs `"override": true`; without it a `SHAPE_ERROR` |
| Message | `sh:message` on a shape replaces the generated message of every result the shape produces; values may carry language tags and there may be several | one string; the most specific authored message wins (field shape, then enclosing shape), else generated text | the child inherits the parent's message when it states none; an explicit empty one clears it |
| Messages and severity of the engine's own structural findings | built-in constraint components take the shape's severity | fixed by the runtime (core section 10) | unchanged |

Layer 3 changes nothing in this table: none of its items touches severity, so
the last column is also the end state after layer 3. The nearest item is G11-3c,
inheriting a shape's `id`, which concerns the constraintId and not the severity.

Three things follow. First, Shaxon's `extends` merge has no SHACL counterpart, so
inheritance here cannot borrow SHACL's behaviour; it follows from the
narrowing principle alone. Second, the gate makes a severity downgrade a change
in what a package accepts, which is why it is treated as a loosening and not as
a relabelling. Third, a package ported from SHACL with `sh:Warning` shapes
reports `conforms: true` where SHACL reports false; that is a difference in
meaning, documented in the limitations document, not something `extends`
changes.

### 2.2 The property test

For each of a few thousand seeded pairs, build a parent from a random subset of
the narrowing members and a child that extends it with another random subset,
and a set of test values. Assert that a value the child accepts is accepted by
the parent. Pairs that fail to load must fail with `SHAX_SHAPE_ERROR` and
nothing else. A one-off version of this ran on 2026-10-07 over 400 pairs and 21
values (6,909 comparisons, no child looser than its parent, 71 pairs rejected at
load); it was a throwaway script and is not in the tree.

The members that may loosen (`closed` with override, `ignoredProperties`,
pooled `or`, every `override`, including a severity lowered under option C) are
outside the generator, and the test says so; the generator also checks that an
unmarked lower severity is rejected at load.

## 3. Layer 3: postponed

None of these is needed for correctness. Each has a trigger and a reason it
waits.

| ID | Item | Question to settle | Trigger | Why it waits |
|---|---|---|---|---|
| G11-3a | Loosening flags for `kind`, `items`, `qualified` and the keywords | the spelling: `"override": true` on the member, or per keyword (`{"minLen": {"value": 2, "override": true}}`) | a package that needs a relaxed variant of a base shape | `and`/`not` composition and not extending cover it today; the spelling is a design choice with no example to test it against |
| G11-3b | More than one `qualified` rule per shape | `qualified` is one entry keyed `""` in the parser; several rules need names, and then the merge is by name, with the narrowing rule applied per name | a shape that needs two counts over one array | a syntax change to the core, not a merge change |
| G11-3c | Inheriting a shape's `id` (the constraintId) | whether a child may share the parent's id, or must state its own, since two shapes with one id would be indistinguishable in a report | a report consumer that groups by id across an extension | today the id is the child's own, which is unambiguous |
| G11-3d | Comparing a named shape and its inline copy as equal | resolve a `{"shape": "X"}` to its definition before the sameness test in section 4 | a package that restates a parent's `items` in the other form | the specification states the current rule (they differ), so this is a refinement, not a gap |
| G11-3e | More than one parent | already decided against: `extends` is single-parent and composition is `and` (editorial 0.3.1, section 4) | none planned | recorded so the question is not reopened by accident |

## 4. Order

1. G11-2a, with its fixtures and mutants, since it is a behaviour change.
2. G11-2c, which will show whether 2a left a hole.
3. G11-2b, written once the code and the test agree on the list of loosenings.
4. G11-2d, as part of each of the above, then a final sweep.

Layer 3 items are taken one at a time when their trigger occurs.

## 5. Not decided here

- Severity option A, B or C (section 2.1.1): the recommendation is C. The choice
  changes the wording of one merge-table row, the severity parser and a few
  fixtures; option C also needs the override spelling for a scalar member to be
  chosen.
- Whether the property test is part of the default run or only behind a tag:
  the default run is preferred if it stays under a second.
