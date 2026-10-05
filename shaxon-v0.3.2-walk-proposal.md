# Shaxon v0.3.2 — proposal: the Walk profile

Status: proposal, unimplemented. Updated: 2026-10-02. Revision 2.

Targets `shaxon-v0.3.1-core.md` (`"shaxon": "3.1"`). "Walk" is a working
name — a session is a bounded walk over a reference graph — and can be
renamed freely.

Revision 2 re-examines every open question of revision 1 from the position
of a SHACL developer, and changes the design accordingly: the model moves
out of the payload and into the package (shapes graph and data graph are
kept apart); host-built adjacency becomes a keyed set, so no ordering is
ever trusted from the host; tier shapes carry literal thresholds and are
class-guarded; and the revision-1 open questions O-1 to O-8 are resolved
(§13). The changes and their reasons are listed in §10.

This proposal develops one application of Shaxon concretely: checking
observed **access behaviour** against a normative, identity-free model, and
emitting a deterministic, explainable decision record that a host acts on.
It is a **profile** — payload encodings, package conventions, a set of
generated shapes and a bounded scoring kernel — not a change to the
language. Every mechanism it needs exists in v0.3.1, with three findings
(§9, F1–F3) and one core candidate (§9, C1) recorded where the fit is
imperfect. Nothing in the v0.3.1 core is altered by anything below.

## 0. Method and versioning

Three tiers of claim are kept apart throughout, because mixing them is how
a proposal ends up promising more than the language can deliver:

- **Tier 0 — expressible today.** Built only from v0.3.1 members. This is
  the whole of the profile (§3–§8).
- **Tier 1 — core candidate.** A change to the language, proposed only where
  Tier 0 has a concrete cost, and gated on a fixture showing the cost is
  real (§9). Exactly one is proposed.
- **Tier 2 — host contract.** Things Shaxon cannot verify and must not
  pretend to: how the RDF graph is lifted, whether degrees are true, where
  a package came from, what the host does with a finding (§4.5, §8.4).

A walk package is an ordinary `"shaxon": "3.1"` package. Profile conformance
is established by the fixture suite in §11 and by the `WALK_` identifier
prefix, not by a package-level declaration (§13, O-1). The v0.3.1 proposal
closed with the recommendation that a reference interpreter and fixtures
come before another round of expansion (v0.3.1 proposal §8); this document
follows it by proposing no new language surface beyond C1 and by putting
fixtures first.

## 1. Purpose

The question the profile answers is not *who is this?* but *is what is
being done, and how, consistent with standard practice?* Identity never
enters the model. A session is an anonymous handle; the evidence is the
sequence of resources touched.

A person may legitimately reach a resource they have not touched before for
two reasons, and the model has exactly one explainer for each:

1. **Inertia** — the resource is close, in the reference graph, to what the
   session has recently touched.
2. **Procedure** — the resource is what the logical sequence of this kind of
   work demands next, whether or not it is near.

An access explained by neither is the **residual**, and the residual is the
signal. It is graded, not boolean, and the grade selects a response tier
(§8).

What Shaxon contributes, in terms of its own invariants (Core §0):

| Invariant | What it gives this use |
|---|---|
| Determinism | Replay: the same logged payload, run through the same package, yields the same vector, the same findings and the same step count, on every conformant runtime. A decision can be reproduced months later in an audit. |
| Boundedness | Per-event cost is a function of declared horizons, never of graph size. A hub cannot make a single event expensive. |
| Explicit reach | The reference neighbourhood the host supplies is declared and checked; nothing is discovered at scoring time. |
| Validation purity | Scoring is an ordinary `program` computation; tiering is a pure predicate over its result. Findings cannot mutate anything. |

What it does not contribute, stated early so it is not assumed: **learning**
the model (§12), **acting** on a finding (§8.4), authentication or
authorisation, and any detection of behaviour that is normal but malicious.

## 2. The model in one page

**Reference graph.** The sources of truth are RDF. The host lifts them to a
labelled graph: nodes with scalar identifiers and a class (`type`), edges
with a predicate. Shaxon never sees an IRI (Core §11: identity is "same
index key"); the lift is a host duty (§4.5).

**Practice pieces.** The normative model is a set of small class-level
graphs — **pieces** — each describing one standard practice ("from a
customer, open an order, then its invoice"). Their nodes and edges are
labelled with classes and predicates, never with instances.

**Composition rule.** Pieces compose by **label-preserving union**: nodes
carrying the same label in different pieces are the same junction, and the
edges are pooled. The composed graph is the model's picture of normal
traffic. Two consequences, stated because they limit what the formalism
buys:

- With global labels, a decomposition into pieces always exists (one piece
  per edge). The composition rule therefore constrains nothing by itself;
  a model is only *meaningful* if its pieces are few and reused. That is a
  compression criterion, and choosing pieces by it is a learning concern
  outside Shaxon (§12). The bounded-treewidth characterisation applies to
  the label-*forgetting* variant of composition and is not used here.
- What Shaxon *can* verify is the **coverage invariant** (§4.3): the union
  of the pieces equals the set of class-level edges the model attributes
  non-zero likelihood to. A model that fails it is rejected at admission
  (§8.3), before any package is built from it.

**Composition history.** Which pairs of pieces have been seen joined at which
junction is recorded (the `fusions` table). An access that composes two
pieces at a junction where they have never composed is a **novel fusion**,
a deviation class a flat edge whitelist cannot express: every edge involved
is individually known, and the combination is the anomaly.

**Session.** A session is a walk. Each event is the smallest graph — one
accessed node and, optionally, the edge traversed to reach it — fused by
node identity into the session's bounded overlay (the carry, §4.4). The
overlay references the reference graph by id; it never copies it.

## 3. What v0.3.1 already provides

| Need | v0.3.1 mechanism | Fit |
|---|---|---|
| Reject payloads carrying identity | `closed` shapes on input gates (Core §4, §7) | Native |
| Hold the model apart from the data | `$lit` literals installed into `state` by `program` (Jaxson core design) | Native, size cost to be measured (§13, P-1) |
| Node and edge tables, forward and inverse reach | `relations`, `indices`, `reference`, `$inverse` (Core §3–§6) | Native, with friction F1; used by Encoding A only |
| Neighbourhood expansion in `program` | `for` over snapshots; `get`/`has`/`keys` in compute | Native for the keyed encoding (§4.2, Encoding B); needs C1 for the array encoding |
| Exact scoring | Exact decimals; `div`/`round` the only digit-losing operations | Native |
| Bounded cost | `limits.steps`, structural loops | Native |
| Count constraints | `check` over `len` | Native; no new keyword (§13, O-4) |
| Per-session state across events | Host-side `Carry` convention (`pkg/jaxtools`), as Navy Wars already uses | Convention |
| Graded response with stable identifiers | Report-mode shapes, `severity`, `constraintId` (Core §10) | Native |
| Mitigation | — | None, by validation purity; host (§8.4) |
| Learning the model | — | Out of scope (§12) |

### 3.1 The SHACL reading

A SHACL developer's expectations are the review criterion for the
decisions in this proposal, so they are stated once, here, and referred to
by name elsewhere. They are SHACL 1.0 conventions; the 1.2 draft has not
been checked against them.

- **Shapes graph and data graph are separate.** The package is the shapes
  graph, including every threshold and horizon, as constants. The payload
  (event, slice, carry) is the data graph and contains no configuration.
- **Multi-valued means a set.** Where a structure is built by the host, it
  is a keyed set, and order is unobservable. Order is semantic only in
  structures the package itself writes.
- **Cardinality is a constraint over the number of values**, as
  `sh:minCount`/`sh:maxCount` are, expressed here as a `check` over `len`.
- **Severity belongs to a shape, not to a constraint.** One shape per tier
  is therefore the idiom, not a workaround for the per-constraint severity
  gap (Limitations §3).
- **Validation always yields a report.** SHACL has no equivalent of a gate;
  behaviour is therefore never gated (§8.2), and results are cumulative.
- **Conditional constraints are `or` with a negated guard.** Class-specific
  thresholds use that form (§8.1).
- **`closed` shapes correspond to `sh:closed`.**
- **No analogue exists** for sessions, state or time. The carry is data and
  `tick` is an input; nothing in SHACL constrains how they are handled.

## 4. Payload encoding

The package receives one JSON payload per event: the event, the slice and
the carry. The host splices the carry from the previous invocation's output
into this one's input, exactly as hosts of multi-turn Jaxson packages
already do. The model is **not** part of the payload (§4.3).

### 4.1 Event

```json
{
  "sid": "s-7f3a",
  "tick": 1042,
  "epoch": 17,
  "event": {
    "node": "r:97",
    "via": { "from": "r:42", "pred": "hasOrder" }
  }
}
```

`sid` is an opaque, anonymous handle with a host-side TTL. `tick` is a
host-supplied integer: **time enters only as input**, because Jaxson has no
clock. The input gate requires `tick` to be no smaller than the last entry
of the carry's `ticks`; a non-monotone tick is a payload fault, not a
finding. `epoch` identifies the reference-graph snapshot. `via` is optional;
an access with no traversed edge (a direct open, a search hit) omits it. In
v1 an event is exactly one accessed node (batch events are deferred, §12).

### 4.2 Slice

The host supplies the **slice**: the reference neighbourhood of the
accessed node to a declared radius. The slice is a fragment of the
reference graph, not a copy of it; per-invocation payload size is therefore
bounded by the radius and by the host's own fanout cap, independent of
graph size.

Two encodings are specified. **Encoding B is the v1 default** because it
runs on unmodified v0.3.1.

**Encoding B — keyed sets.** Nodes, adjacency and the frontier are objects
keyed by node id, so compute can use `get`, `get_or` and `has` for
constant-time lookup and `keys` (sorted) for deterministic iteration.
Adjacency is a **set of neighbours**: `adj[u]` is an object whose keys are
the neighbours of `u` and whose values are the predicates linking them.

```json
{
  "radius": 2,
  "nodes": {
    "r:42": { "type": "Customer", "deg": 31, "region": "sales" },
    "r:97": { "type": "Order", "deg": 6, "region": "sales" },
    "r:88": { "type": "Invoice", "deg": 3, "region": "billing" }
  },
  "adj": {
    "r:42": { "r:97": ["hasOrder"] },
    "r:97": { "r:42": ["hasOrder"], "r:88": ["hasInvoice"] }
  },
  "frontier": { "r:88": true }
}
```

- **Order is never trusted from the host.** The kernel truncates by fanout
  by taking the first *F* of `keys(adj[u])`, and `keys` is sorted by the
  runtime. Two hosts that serialise the same slice differently therefore
  produce the same vector. This removes the "sorted adjacency" obligation
  that revision 1 placed on the host.
- Predicate lists are carried so that the walk can later be reweighted by
  predicate (workflow relations are not the same as data relations; §12).
  **The v1 kernel does not use them**: its walk is predicate-agnostic and
  undirected.
- `deg` is the node's degree in the **full** reference graph, not the
  slice, because the slice is truncated at its boundary and the hub
  discount (§6.2) needs the true value. It cannot be verified from the
  slice; it is a host contract (§4.5).
- `frontier` is the set of nodes at the slice boundary whose adjacency is
  deliberately absent. Any other neighbour must itself be a key of `nodes`.
  The kernel counts violations in `program` and asserts a count of zero with
  a `check` instruction (§7, step 2); it is a payload fault (fixture W2).
- Edges flagged `inferred` by the host are excluded from `adj` unless the
  package was generated with `useInferred`. Whether distances are measured
  over asserted or entailed triples changes the geometry (Limitations §6);
  this profile starts from asserted triples.

**Encoding A — arrays.** `nodes` and `edges` as arrays, with `relations`
(`edges.s` to `nodes.id`) and a `multi` index over `edges.o`. This is the
shape a SHACL developer expects of a graph — triples, with inverse reach
free — and validation is fully native (`reference`, `DANGLING_REFERENCE`).
It cannot be *traversed* inside `program` today, because no operand form
in `program` consumes an index (finding F2). Encoding A becomes the default
if C1 is accepted (§9).

### 4.3 Model

The model is **not payload**. It is a set of constants of the package,
generated once per model version by the host's build tooling and installed
at the start of `program` by a first `set` instruction whose value is
`{"$lit": <model>}`. A new model version is a new package; the package
hash is the model version, so replay pins the model for free.

```json
{
  "modelVersion": 5,
  "hops": 2,
  "fanout": 8,
  "working": 16,
  "matches": 8,
  "beta": 0.5,
  "decay": 0.9,
  "window": 50,
  "anchorN": 5,
  "sensCuts": [0.3, 0.7],
  "pieces": {
    "order-lookup": {
      "from": "Customer",
      "steps": [
        { "pred": "hasOrder", "to": "Order" },
        { "pred": "hasInvoice", "to": "Invoice" }
      ]
    }
  },
  "lik": [
    { "from": "Customer", "pred": "hasOrder", "to": "Order", "p": 0.62 },
    { "from": "Order", "pred": "hasInvoice", "to": "Invoice", "p": 0.48 }
  ],
  "fusions": [],
  "sens": { "Order": 0.3, "Invoice": 0.8 },
  "provenance": "rev-0031"
}
```

- `hops` is the length of the literal round list the walk iterates (§7);
  because the model is baked in, the loop bound and the model value are the
  same constant and cannot disagree.
- A **piece** in v1 is a path-shaped sequence: a starting class and an
  ordered list of `(pred, to)` steps. General small graphs (branching,
  joining) are deferred (§12).
- `lik` is the model's picture of normal traffic at class level: each
  traversed class-level edge with its observed likelihood. It is the "G1"
  of the original formulation.
- **Coverage invariant**, checked at admission (§8.3): every `lik` entry
  with `p > 0` appears as an edge of some piece (counting the edge from a
  piece's `from` class to its first step), and every piece edge appears in
  `lik`. The pieces then compose to exactly the model's baseline.
- Threshold bands are **not** in the literal: they are constants inside the
  generated tier shapes (§8.1), where a SHACL developer looks for them.
  `sensCuts` partitions `sens` into the three classes `low`, `normal` and
  `high`.
- `provenance` is an opaque review reference; it names no person.

### 4.4 Carry

The carry is the session overlay. The package writes it; the host only
passes it back.

```json
{
  "working": { "r:42": 0.8 },
  "matches": [
    { "piece": "order-lookup", "pos": 0, "at": "r:42", "w": 1, "done": false }
  ],
  "anchor": { "n": 5, "hist": { "Customer": 3, "Order": 2 } },
  "ticks": [1031, 1040],
  "opened": { "Order": [1040] },
  "prevRegion": "sales",
  "modelVersion": 5
}
```

- `working` is a keyed set of node id to recency-decayed weight, so
  membership is a lookup and no derived map is built each event.
- `matches` is the one ordered structure. Its order is canonical and
  produced by the package (descending weight, then piece name, then `at`),
  which is why it does not offend the rule that order is never trusted: the
  host never builds it.
- A match records the piece, how many steps it has consumed (`pos`) and the
  node it stands on (`at`). Completed matches stay for a bounded number of
  events (`done: true`) so fusions are detectable after a practice has ended.
- `modelVersion` lets the package notice that a session began under a
  different model (§5).

### 4.5 Identity-free by construction, and the host contract

Every payload shape — event, slice, carry — is `closed`. A payload that
smuggles a principal, a username or any member the shapes do not name fails
the input gate. This enforces the behavioural stance at the boundary, not
by convention.

It does not make the system identity-proof: a resource identifier can itself
denote a person, and no shape can know. The following remain **host
contract**, outside what Shaxon verifies:

- lifting RDF to the graph encoding, including skolemising blank nodes and
  interning IRIs to string ids;
- choosing asserted versus inferred triples;
- supplying true full-graph degrees and a slice complete to the declared
  radius (the completeness check catches only internal inconsistency);
- mapping named graphs to `region`;
- stripping every principal before the payload is formed;
- **package provenance**: that the package in use is the one generated from
  the admitted model, and that its hash is recorded with each decision.
  Because the model is now part of the package, the integrity of the model
  is the integrity of the package; Shaxon cannot verify either.

## 5. State by lifetime

| Lifetime | What it is | Where it lives | Bounds and rules |
|---|---|---|---|
| Reference graph (slow-changing) | The lifted sources of truth | Host. Only the slice enters the package | Pinned by `epoch`; a new epoch is a new snapshot |
| Learned model (changes only through review) | Pieces, `lik`, `fusions`, `sens`, horizons, and the thresholds inside the tier shapes | The package: literal constants and generated shapes | Immutable per version; the package hash is the version; sessions record the `modelVersion` they began under |
| Session overlay (hot, ephemeral) | Working set, matches, anchor, ticks, junction counters | `input.carry` in, `output.carry` out | Caps held by eviction and asserted by an output `check`; discarded by host TTL |
| Population baseline (aggregate) | Class-level transition and incidence statistics | Host. Never enters the package | Counters only; no per-session or per-person content |

**Carry caps.** `working` ≤ `working`, `matches` ≤ `matches`, `ticks` and
each `opened` list ≤ `window` entries (literal constants of the package).
The kernel enforces them by eviction; an output `check` over `len` (and
`keys` for the keyed members) asserts them as a safety net, failing with
`VALIDATION_ERROR` if the kernel ever violates its own invariant. Total
memory is bounded by concurrent sessions times a constant. No count keyword
is needed in the shape vocabulary (§13, O-4); the check form depends on
`len` accepting the carry's member types (§13, P-2).

**Eviction is deterministic.** When a cap is exceeded, the entry with the
lowest weight is dropped; ties break by node id, then piece name, in
code-point order. Never insertion order, never an unordered iteration.

**Epoch change mid-session.** A new `epoch` is a new snapshot, and the
carry is simply data against it. A carry entry whose id is absent from the
new slice contributes nothing to scoring and is retained until evicted;
the record sets `horizon.epoch`, so the score is marked a lower bound on
proximity. The overlay is **not** reset: resetting it would give an
attacker, or an ordinary habit, a way to wipe drift evidence by provoking an
epoch change. Whether to reset is a host policy decision, made visibly.

**Model change mid-session.** A carry whose `modelVersion` differs from the
package's is treated the same way: matches naming a piece absent from the
new model are dropped and counted, `horizon.model` is set, and the session
continues against the new model.

## 6. The vector

### 6.1 Numeric conventions

- All numbers are exact decimals or integers; there are no floats anywhere.
- Every division specifies **scale 4** and **omits the rounding mode**, so
  the language default (`half_even`) applies everywhere and mixing modes is
  impossible. The showcase's `half_up` is a money convention and does not
  apply to a scoring kernel.
- Every ordering is a code-point ordering of ids or names, as in Core §3
  and §6. Breadth-first visitation follows the Core §6 closure rule:
  by hop count, ties by sorted key.
- Unseen values are zero; a bounded quantity beyond its horizon is
  reported as the horizon plus one, never as an absence.

### 6.2 The twelve dimensions

The vector is the per-event summary the checker emits. It is a **feature
vector**, not the session state; twelve numbers cannot reconstruct which
nodes were touched, which is why the carry exists (§4.4). Dimensions are
grouped by the explainer they test, so a finding can say *which* explainer
failed.

| # | Name | Group | Type | Meaning |
|---|---|---|---|---|
| 1 | `prox` | Inertia | decimal 0..1 | Proximity of the accessed node to the working set: the sum over working-set nodes within the horizon of normalised weight times `beta` to the hop distance |
| 2 | `hop` | Inertia | integer 0..H+1 | Smallest hop distance from the accessed node to any working-set node; H+1 means beyond the horizon |
| 3 | `proxd` | Inertia | decimal 0..1 | `prox` with each intermediate node's contribution divided by its true degree, so passing through a hub counts for little |
| 4 | `prog` | Procedure | decimal 0..1 | Best partial-match progress after this event: consumed steps over piece length, maximised over matches |
| 5 | `lik` | Procedure | decimal 0..1 | The model's likelihood for the class-level edge just traversed; 0 if the model never saw it |
| 6 | `compat` | Procedure | integer | Number of distinct pieces that this event started or advanced |
| 7 | `fuse` | Composition | 0 or 1 | 1 if the event composes two pieces at a junction where the `fusions` table has never recorded them composing |
| 8 | `drift` | Composition | decimal 0..1 | Total-variation distance between the class histogram of the working set and the session's anchor histogram |
| 9 | `load` | Incidence | integer | Pieces opened through this junction class within the window |
| 10 | `rate` | Incidence | integer | Events within the window |
| 11 | `sens` | Context | decimal 0..1 | The model's sensitivity weight for the accessed node's class |
| 12 | `cross` | Context | 0 or 1 | 1 if the accessed node's region differs from the previous event's |

Dimensions 11 and 12 are not deviations. They **modulate**: `sens` is
bucketed by `sensCuts` into a sensitivity class, which selects the tier
shapes that apply (§8.1), so the same distance is graded more severely at a
sensitive class.

### 6.3 The decision record and the horizon indicator

The vector sits inside a small **decision record**, which is what the
package writes to `state.record` and `output.record` and what the tier
shapes validate:

```json
{
  "vector": { "prox": 0.31, "hop": 1, "proxd": 0.12, "prog": 0.5, "lik": 0.62, "compat": 1, "fuse": 0, "drift": 0.05, "load": 1, "rate": 2, "sens": 0.3, "cross": 0 },
  "sensClass": "normal",
  "viaPresent": true,
  "modelVersion": 5,
  "horizon": { "hops": false, "fanout": true, "working": false, "matches": false, "slice": false, "epoch": false, "model": false }
}
```

Truncation by a horizon is not an error: a monitor that aborts on a hub is
a monitor an attacker can switch off. It is reported in `horizon`, not in
the vector, so the vector stays at twelve dimensions. A true member means
the corresponding score is a lower bound on proximity or matching, which
the host may take into account. This is distinct from the existing
`PATH_DEPTH_EXCEEDED` (Core §6), a violation about a document exceeding a
declared closure depth; here the horizon is the monitor's own, and
exceeding it is information.

### 6.4 Banding

Eight dimensions are graded against thresholds. A dimension is either
**low-bad** (`prox`, `proxd`, `prog`, `lik`: the finding fires when the
value falls below the threshold) or **high-bad** (`hop`, `drift`, `load`,
`rate`: it fires when the value exceeds it). Each has three monotone
thresholds per sensitivity class, one per tier, baked into the generated
shapes. `compat`, `fuse` and the uncovered-edge condition are binary event
conditions with their own shapes; `sens` and `cross` are context and are not
graded.

Correlated dimensions (1 and 3, 9 and 10) are therefore not independent
evidence. Tiering is per dimension and never over a single scalar distance,
so a large deviation in one dimension is not diluted by quiet ones, and
each finding names the dimension that caused it. A covariance-based
distance is deferred (§12).

## 7. The scoring kernel

The kernel is the `program` phase. It runs in a fixed order, and **scoring
precedes updating**: the access is judged against the overlay as it stood
before the access, so an access cannot explain itself.

0. **Load the model.** `set` `state.model` from the package literal.
   One instruction; whether a large `$lit` is one step is fixture W10
   (§13, P-1).
1. **Validate the payload.** Gates on event, slice and carry, all `closed`,
   including the tick monotonicity check. Failure is `VALIDATION_ERROR` and
   the host keeps its previous carry (a failed turn leaves `Session` state
   untouched).
2. **Check slice completeness.** Loop over `keys(adj)` and their neighbours,
   counting ids that are neither keys of `nodes` nor in `frontier`; assert
   zero with a `check` instruction.
3. **Classify.** Derive `sensClass` from `sens` of the accessed node's class
   and `sensCuts`.
4. **Neighbourhood walk.** Breadth-first from the accessed node to `hops`
   rounds, taking at most `fanout` neighbours per node from `keys(adj[u])`.
   Each discovered node records its hop and its BFS-tree parent (the first
   discoverer in visiting order). Working-set members encountered contribute
   to `prox`, `hop` and, with the degree discount applied to intermediate
   nodes only, to `proxd`. The accessed node itself is never discounted by
   its own degree.
5. **Piece matching.** For each active match, in canonical order: it
   advances if `via.from` equals its `at` and `(via.pred, type(node))`
   equals its next step; reaching the final step marks it `done`.
   Separately, any piece whose `from` equals the accessed node's class opens
   a new match at position 0. Over-cap matches are evicted per §5. This
   yields `prog`, `compat` and `lik`.
6. **Fusion.** A match parked at a node of class *t* (active or recently
   done) for piece B, with an event that starts or advances piece A at *t*,
   is a composition of A after B at *t*. `fuse` is 1 if `(t, B, A)` is
   absent from `fusions`.
7. **Drift and incidence.** `drift` from the two class histograms in time
   proportional to the number of classes; `load` and `rate` by counting
   `opened` and `ticks` entries within `window` of `tick`.
8. **Update the overlay and write the record.** Multiply every working
   weight by `decay`, add the accessed node at weight 1, evict to the cap.
   Update `anchor` until it has seen `anchorN` events, then freeze it.
   Append `tick`. Write the new carry and the decision record to `output`.

**Loop bounds are structural.** The walk iterates a literal list of round
numbers of length `hops`, generated into the package. `for` iterates finite
snapshots, so nothing here can fail to terminate.

**Cost.** Per-event steps are bounded by a function of the model's constants
only: on the order of `fanout^hops` visited nodes for the walk, plus
`working` for the update, plus `matches` times piece length for matching,
plus the class count for drift, plus a fixed cost for the generated shapes
(§8.1) and a slice-completeness pass proportional to the slice. Constant
factors are to be measured, not asserted (fixture W10), and `limits.steps`
is set from those measurements. Exceeding it is `RESOURCE_ERROR` at the same
point on every runtime, which the host must treat as *unscored*, not as
*allowed*.

## 8. Response

### 8.1 Tiers are generated, class-guarded, report-mode shapes

The package contains a family of shapes **generated** from the model, in
the way SHACL shapes graphs are routinely generated from an ontology. Each
graded dimension has one shape per tier per sensitivity class. A shape
describes **normal** behaviour at that tier's threshold, so a normal session
conforms and a deviation is a finding. The class is applied as a guard: the
constraint is `or` of "not this class" and the threshold test, so a shape
for another class conforms vacuously. This is SHACL's own form for a
conditional constraint:

```json
{
  "shapes": {
    "WalkHopT2Normal": {
      "kind": "object",
      "check": {
        "with": {
          "hop": { "$path": ["local", "focus", "vector", "hop"] },
          "cls": { "$path": ["local", "focus", "sensClass"] }
        },
        "expr": ["or", ["not", ["eq", { "$v": "cls" }, "normal"]], ["le", { "$v": "hop" }, 2]],
        "id": "WALK_HOP_T2_NORMAL"
      },
      "severity": "warning",
      "message": "access is further from recent work than usual for this kind of resource"
    }
  },
  "validate": [
    { "target": { "$path": ["state", "record"] }, "shape": "WalkHopT2Normal", "mode": "report" }
  ]
}
```

- **Counts.** At the default settings: 8 graded dimensions × 3 tiers × 3
  classes = 72 shapes, plus the novel-fusion and uncovered-edge shapes at 3
  classes each, for **78 generated shapes**. The number is a property of the
  generator, not something an author writes or reviews by hand. Whether 78
  `validate` entries per event is acceptable is fixture W10 (§13, P-3).
- Tier 1 (`info`) is guidance, tier 2 (`warning`) is review or
  justification, tier 3 (`violation`) is the highest. Only `violation`
  flips `conforms` (Core §10). For the binary shapes, severity is set per
  class: `info` at `low`, `warning` at `normal`, `violation` at `high`.
- The uncovered-edge shape conforms unless `viaPresent` is true and `lik`
  is 0: an edge was traversed that the model has never seen.
- Entries omit `into`, so their violations concatenate into one combined
  report in execution order (Core §7).
- Findings are **cumulative by design**: a tier-3 deviation also fails the
  tier-2 and tier-1 shapes of its dimension, and the host takes the highest
  severity. The `constraintId` is `WALK_<DIM>_T<n>_<CLASS>`, so a consumer
  can aggregate by dimension, tier and class without reading messages. The
  class is carried in the identifier because whether the report also names
  the source shape, as `sh:sourceShape` does, is not settled (§13, P-4).
- `message` stays a static string (v0.3.1 proposal §1.12). Specific guidance
  — "the usual next step from here is the invoice" — is produced by the
  host from the `constraintId` and the record, not by the message.

### 8.2 Behaviour never gates

Tier shapes are always `report` mode, never `gate`. A gate aborts the
pipeline with `VALIDATION_ERROR` and the run produces no output, so the new
carry would be lost and the session state corrupted by the very event being
judged. SHACL has no gate at all and a SHACL developer would expect a
report every time. Gates are reserved for payload well-formedness (§7,
step 1), where losing the turn is the correct outcome and the host's
previous carry is exactly what should survive.

### 8.3 Model admission and promotion

A separate package takes a candidate model and its predecessor as **input
data** and validates it once, at promotion time, before the host's build
tooling may generate a package from it. It is the only place the model is
data, and it runs once per version, not once per event. It checks the
well-formedness of every field of §4.3, the coverage invariant, and a
**delta limit**: at most a declared number of new pieces and new fusions per
version, and no new fusion at a class whose sensitivity exceeds a threshold
without an explicit approval reference. This addresses slow-drift poisoning
(an attacker, or ordinary habit, shifting behaviour one plausible step at a
time until the model absorbs it) as a validation of the *model change*, in
the same language and under the same determinism as everything else.

### 8.4 What the host does

Shaxon emits a decision record: the vector, the horizon indicator, the new
carry and the report. It performs no mitigation, by validation purity. The
following mapping is non-normative:

| Highest finding | Typical host response |
|---|---|
| none | proceed |
| tier 1, `info` | show guidance: the standard path that completes the practice |
| tier 2, `warning` | log; ask for a justification; allow |
| tier 3, `violation`, low sensitivity | step-up authentication or scoped throttling |
| tier 3 with `fuse` or an incidence finding at a sensitive class | hold for approval; alert |
| run failed or `RESOURCE_ERROR` | **unscored**: a host policy decision, never silently "allowed" |

## 9. Findings against v0.3.1, and one core candidate

These are recorded here, not silently patched into the core. They belong on
the core's pending list; each has a workable Tier 0 route.

**F1 — §4c against a bidirectional graph.** A graph needs two inverse
directions over one node table: edges by subject and edges by object. Core
§4c rejects two `relations` entries whose `to` sides normalise to the same
`(path, key)`, so the second direction must be a standalone `multi` entry
under `indices`, which §4c deliberately leaves unchecked. In RDF inverse
paths are free, so a SHACL developer finds the restriction surprising.
**Recommended for adoption independently of this profile:** let relations
share a `to` side when their `from` sides differ, building the `to` index
once under the existing mandatory reuse rule (Core §3). It affects only
Encoding A.

**F2 — index use in `program`.** Core §3 charges index builds "before first
use in `program`", but the only forms that consume an index (`reference`,
`$inverse`, `$indexed`) are shape, target and index-key constructs, and
Core §6 restricts path-expression forms to shapes and indices. No operand
form in `program` or compute appears to return a value from an index. The
cost sentence and the operand set disagree, and the disagreement is what
forces Encoding B. With keyed-set adjacency it no longer blocks this
profile; it remains an editorial defect in the core.

**F3 — `limits` ownership.** In the implementation a `limits` member
outside the dialect's declared set is a `VERSION_ERROR`
(`Profile.Limits`). A profile therefore cannot add loop-bound members under
`limits` without being a dialect that owns them. **Avoided in revision 2**:
horizons are literal constants of the package (§4.3), so no new `limits`
member is needed. The finding stands for any future profile that wants
package-level limits.

**C1 — `$lookup`, a value operand for declared indices.** Sketch only:

```json
{ "$lookup": { "index": "nodesById", "key": { "$path": ["local", "id"] } } }
```

It returns the element an index maps the key to (a value for a non-`multi`
index, an ordered array for a `multi` one), is valid in `program` and in
`with` bindings, and follows `get` for a missing key (`MISSING_PATH`).
Accounting, to be settled by fixture rather than asserted here: one step
per lookup, the index build charged under Core §3's existing rule and
reuse, and no escape from the explicit-reach invariant because the index
must be declared at load. It would resolve F2, make Encoding A traversable
and let the package derive adjacency from edge arrays instead of trusting a
host-built `adj`.

Revision 2 weights C1 more favourably than revision 1 did: Encoding A is
the encoding a SHACL developer expects, and an index that can be declared
but not read where values are computed is the surprising part. The gate is
unchanged, because the profile does not need it: ship on Encoding B first,
and adopt C1 if fixture W3 or W10 shows Encoding B cannot be validated or
evaluated within a reasonable `limits.steps`, or if hosts find building
keyed adjacency a burden.

## 10. Recorded decisions, and deviations

Where this proposal departs from an earlier position, the departure is
recorded rather than edited away. The first four rows compare the original
design sketch with revision 1; the rest are the changes made in revision 2.

| Item | Earlier position | Now | Reason |
|---|---|---|---|
| Inertia score | Personalised random walk with restart (sketch) | Truncated hop kernel with a BFS tree (§6.2, §7) | Random-walk scores need iterated real arithmetic to a convergence criterion. Under exact decimals and mandatory step accounting the hop kernel is exact, single-pass and cheap. The cost is that it considers one route per node, not all of them. |
| Pieces | Small graphs of any shape (sketch) | Path-shaped sequences (§4.3) | Matching a general graph incrementally needs a richer automaton. Sequences cover the workflows that motivated the model. |
| Correlation handling | Covariance or Mahalanobis (sketch) | Per-dimension banded thresholds (§6.4) | The audit value of a finding that names its dimension is higher in v1. Deferred, not rejected. |
| Tiers and mode | Unspecified (sketch) | Always `report`, never `gate` (§8.2) | A gate would discard the carry. |
| Model location | Spliced into each payload (rev 1) | Package literals and generated shapes (§4.3, §5) | A SHACL developer keeps shapes graph and data graph apart. `$lit` makes a constant registry unnecessary, shrinks per-event payload, moves model checks to promotion time and pins the model by package hash. Gated on P-1; the fallback is revision 1's payload model. |
| Adjacency | Sorted array, order a host obligation (rev 1) | Keyed set (§4.2) | Multi-valued means a set. `keys` is sorted by the runtime, so the host's order is unobservable and the obligation disappears. |
| Band thresholds | Data copied into `state.bands` (rev 1) | Literals in generated, class-guarded shapes (§8.1) | Thresholds are constants of the shapes graph and should be visible there. Costs 78 shapes (P-3); the `state.bands` copy remains the fallback. |
| Tier granularity | Per dimension group (rev 1) | Per dimension (§8.1) | A finding that names its dimension is more actionable than one that names a group. |
| Count constraints | Assumed `maxItems` (rev 1) | `check` over `len` (§5) | The shape vocabulary has no count keyword; `len` expresses `sh:maxCount` without adding a second vocabulary. |
| Rounding | Mode to be chosen (rev 1) | Scale 4, mode omitted (§6.1) | The language default applies everywhere; nothing to mix. |
| Time windows | Ticks or counts open (rev 1) | Ticks, monotone-checked (§4.1) | Count windows make slow and fast sessions indistinguishable, and `rate` loses its meaning. |
| Model admission | Per-event model gate (rev 1) | Once, at promotion (§8.3) | The model is no longer payload, so it is validated when it changes, not every time it is used. |

## 11. Fixtures: before calling this stable

No reference interpreter exists for Shaxon (Core §13), so everything above
is unexecuted. The Jaxson core does run, and the repository ships a Python
cross-check executor beside its showcase examples, so the kernel's compute
and `for` structure can be fixture-tested in Jaxson terms before the
validation layer exists. The needed fixtures, in priority order:

| ID | Checks |
|---|---|
| W3 | Hop kernel golden vectors: tie-breaking over sorted keys, degree discount, rounding; cross-checked against the Python executor |
| W10 | Step budget: kernel by horizon setting, cost of the model `$lit` at realistic size, cost of 78 `validate` entries; decides P-1, P-3 and the C1 gate |
| W7 | Tier shapes: class guards, severity, cumulative findings, and the carry still emitted when a violation fires |
| W11 | Replay: the same payload and the same package twice give identical output and an identical step count; the record carries the model version |
| W1 | Input gate rejects any identity-bearing member (`closed`) |
| W2 | Slice completeness by the `program` count, including the frontier exemption |
| W13 | Tick monotonicity: a non-monotone tick is a payload fault and leaves the host's carry untouched |
| W5 | Piece matching: advance, open, and over-cap eviction determinism |
| W6 | Novel fusion, including a fusion detected after a piece completed |
| W8 | Carry caps by eviction and by the output `check`; eviction tie-breaking |
| W4 | Horizon flags set, and scores correctly marked as lower bounds |
| W9 | Model admission: coverage invariant and delta limits, with a poisoned-delta case |
| W12 | Epoch change and model-version change mid-session |

## 12. Deferred and non-goals

**Deferred**, each with a stated reason it is not ready:

- *General (non-path) pieces.* Needs a branching match automaton and a
  sharper definition of progress.
- *Batch events* (an export touching many nodes in one call). Needs a rule
  for ordering and for whether one vector or many is emitted.
- *Predicate-weighted proximity.* Workflow relations are not data
  relations, and observed traversals should reweight edges. The predicate
  lists in `adj` are carried for this (§4.2) but unused by the v1 kernel.
- *Covariance-based distance.* Expressible; deferred for audit value (§10).
- *A native kernel instruction mounted through a `Profile`.* Only if the
  interpreted kernel proves too expensive at W10; it would move semantics
  into host code that the specification would then have to pin exactly.
- *A declared-exception path* for incident response and break-glass work,
  which look like deviations and need a guidance response that is helpful
  rather than punitive. It must itself be counted as an incidence event,
  or it becomes the bypass.
- *Personalised-walk scoring.* Revisit if single-route proximity proves too
  coarse in fixtures.

**Non-goals**, unchanged and reaffirmed:

- *Learning the model.* Mining frequent subgraphs and choosing pieces by
  description length is unbounded search with no step-accounting story; it
  runs in the host. Shaxon validates and admits what it produces (§8.3).
- *Acting on findings.* By validation purity (Core §1).
- *Identity, authentication, authorisation.* The profile is complementary
  to all three and replaces none.
- *Detecting normal-but-malicious behaviour.* A session that follows
  standard practice at malicious intent is visible only through the
  incidence dimensions (9, 10). The profile does not claim more.
- *Privacy beyond the boundary.* Closed shapes keep principals out of the
  payload (§4.5); they do not make resource identifiers anonymous. The
  carry is per-session behavioural data and the host owns its retention
  and access.

## 13. Resolved and open questions

### 13.1 Resolved in revision 2

Each revision-1 question is resolved, with the SHACL-expectation reasoning
that decided it.

| ID | Question | Resolution |
|---|---|---|
| O-1 | Profile marker? | None. SHACL has no profile marker; conformance is whichever shapes graph was run. The `WALK_` prefix and the fixture suite identify the profile. |
| O-2 | Rounding mode and scale | Scale 4, mode omitted, so the language default applies (§6.1). |
| O-3 | Is adjacency order validated? | Moot: adjacency is a keyed set, and order is unobservable (§4.2). |
| O-4 | Does the shape vocabulary have a count keyword? | No, and none is added. Counts are `check` over `len` (§5). |
| O-5 | Epoch change mid-session | Keep the overlay, score against the new slice, set `horizon.epoch` (§5). Resetting is a visible host policy. |
| O-6 | How is the sensitivity class selected? | Generated shapes guarded by class with `or` (§8.1); the `state.bands` copy is the fallback. |
| O-7 | Ticks or event counts? | Ticks, with monotonicity checked at the input gate (§4.1). |
| O-8 | Do horizons belong in `limits`? | No. They are literal constants of the package (§4.3); F3 is avoided. |

### 13.2 Still open

| ID | Question |
|---|---|
| P-1 | Is a large `$lit` one step, and is it legal at the size of a realistic model? The model-in-package design is gated on this. If it fails, fall back to the model in the payload, validated per event, as in revision 1. |
| P-2 | Does `len` accept the carry's member types (arrays and keyed objects), or does the cap check need `len` of `keys`? Decides the exact form of the carry-cap `check`. |
| P-3 | Is 78 `validate` entries per event acceptable in steps? If not, collapse the class dimension back into a `state.bands` copy selected by `program`. |
| P-4 | Does a report entry name its source shape, as SHACL's `sh:sourceShape` does? The class is encoded in `constraintId` until this is settled. |
| P-5 | Who owns the generator that builds a package from an admitted model, and where does it live? It is host tooling, not Shaxon, but its fidelity is a host contract (§4.5). |
| P-6 | Clock-skew policy: a skewed tick is a host fault the package can only partly detect (monotonicity). |
| P-7 | When should predicate-weighted proximity be specified, given that predicate lists are carried but unused by v1? |

## 14. Traceability

| Proposal item | Depends on | Notes |
|---|---|---|
| §3.1 The SHACL reading | SHACL 1.0 conventions | 1.2 draft not checked |
| §4.1–§4.5 Payload encodings | Core §2, §3, §4a, §5, §7, §11 | `closed` shapes, input gates; relations and indices for Encoding A |
| §4.3 Model as literals | Jaxson core design (`$lit`, `set`) | Gated on P-1 |
| §4.3 Coverage invariant | Core §4 (`check`), §7 | Run at admission, §8.3 |
| §5 Carry caps | Core §7 (output targets); compute `len`, `keys` | P-2 |
| §6 Vector | Jaxson number model: exact decimals, `div`/`round` | No floats |
| §6.3 Horizon indicator | Core §6 (`PATH_DEPTH_EXCEEDED`) | Distinct by intent |
| §7 Kernel | Jaxson `for`, `set`, `get`/`has`/`keys`; Core §6 ordering rule | Structural loop bounds |
| §8.1 Tier shapes | Core §4, §7, §10; Limitations §3 | Generated, class-guarded |
| §8.2 Report not gate | Core §7 gate semantics; `Session.Step` failure behaviour in `pkg/jaxtools` | Carry survival |
| §8.3 Model admission | Core §1 (purity), §4, §7 | Delta limits |
| §9 F1 | Core §4a, §4c | Recommended for adoption |
| §9 F2 | Core §3 cost paragraph, §6 | Editorial: wording and operand set disagree |
| §9 F3 | `pkg/jaxson/profile.go` (`Profile.Limits`) | Avoided |
| §9 C1 | Core §3, Limitations §7 | Gated on fixtures |
| §12 Non-goals | Core §11; Limitations §6, §7 | Unchanged |

Copyright (c) 2026 haitch. Licensed under the Apache License, Version 2.0: https://www.apache.org/licenses/LICENSE-2.0