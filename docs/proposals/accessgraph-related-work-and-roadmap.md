# accessgraph: related work, positioning and roadmap

Updated: 2026-10-26

Status: proposal. It records what a survey of comparable work found, where the
accessgraph example is stronger or weaker than that work, and what remains to
be done, so that it can be picked up without the conversation that produced
it. The example was version 0.0.2 when this was written and is now under
`examples/accessgraph`. Open work is tracked in `docs/TRACKING.md`: T-31 to T-36
in waves 8 and 9, and T-37 to T-40 in wave 10.

Reconciliation, 2026-10-27. Since this was written, T-20 is closed (SHACL Core
alone expresses `chinese-wall` and `break-glass`, so the Core boundary is
narrower than assumed here), T-31, T-32, T-36, T-37 and T-38 are closed (the limits and the id width
are policy, the program asserts the limits, the step limit is derived from the
data; the gate's stage 2 was decided against and the Amazon-scale run moved to
T-39), and T-22 is partial. The survey
below omitted four systems that bear on its claims. CEL computes a static
worst-case cost before evaluation and enforces a runtime cost limit, and
Starlark counts and limits execution steps, so a cost declared before the run
and enforced is not distinctive. CUE and OPA with Rego are further JSON-native
relatives. These come from their documentation, not from running them; Starlark has since been run on the task and CEL probed (`FINDINGS.md`, section 5). The
content below is unchanged and historical.

Evidence level: the survey was one round of web searches on 2026-10-25. The
claims about other systems rest on their abstracts, documentation pages and
advisories, not on a full reading of each paper. Treat them as pointers to
check before anyone repeats them elsewhere.

## 1. What accessgraph is for

The Shaxon authorisation examples judge a trail of events against a rule.
accessgraph derives what a standing structure entails: nested groups, a role
hierarchy, a resource tree, allow and deny grants and bounded delegation
chains. It answers "may this user do this" and says why, with a deterministic
choice of supporting grant. The point is to show Jaxson doing inference, and
to show it checked by an independent reference rather than by its own
author's expectations.

## 2. What we found

No work found combines all of: a general-purpose total language, bounds that
are structural, a program checked against an independently written reference,
and a validation layer that gates the input. Each part has close relatives.

| Work | Relevant to us because | Reference |
|---|---|---|
| Cedar (AWS) | Closest in method. An executable model is proved correct, then differential random testing compares it with the production engine; this found 4 validator bugs by proof and 21 more by testing. Policies of bounded size are documented to terminate. A policy validator and an SMT-based analysis (for example policy equivalence) sit beside the engine. | arxiv.org/abs/2403.04651; arxiv.org/abs/2407.01688; docs.cedarpolicy.com/other/security.html |
| Biscuit and the agent identity protocol built on it | Authorisation written in a Datalog variant, with delegation by attenuating blocks. The protocol's profiles forbid recursion in the simple case and cap evaluation at a fixed iteration count in the advanced one. | github.com/CleverCloud/biscuit; arxiv.org/pdf/2603.24775 |
| Zanzibar and SpiceDB | Permissions derived from a relationship graph with nested groups, and an explanation of why a check passed or failed. A published advisory (CVE-2024-46989) describes a wrong denial for caveated groups, which is the failure an independent reference exists to catch. | research.google/pubs/zanzibar-googles-consistent-global-authorization-system; cvelogic.com/cve/CVE-2024-46989 |
| Delegation Logic and RT (Li, Mitchell, Winsborough) | The theory behind our rules. Delegation Logic treats delegation depth explicitly. RT translates its credentials to Datalog with a tractable semantics and covers separation of duty through manifold roles. A companion paper analyses safety and availability. | cs.purdue.edu/homes/ninghui/papers/dl_ibmrr.pdf; cs.purdue.edu/homes/ninghui/abstracts/rt_oakland02.html; cs.purdue.edu/homes/ninghui/papers/analysis_oakland03.pdf |

## 3. Where accessgraph is stronger, and how

Each claim carries its basis and its status. "Observed" means measured in this
work; "structural" means a property of the language; "planned" means not yet
true.

| Claim | Basis | Status |
|---|---|---|
| Termination needs no cap or trust. Jaxson has no unbounded loop, so every package terminates by construction. A closure is written as an explicit number of rounds that a reader can see and review. Biscuit's profile uses a runtime iteration cap; Cedar guarantees termination for bounded-size policies. | The language definition. | Structural. Not yet fully realised: the round counts still come from the reference per database (T-31). |
| The reference differs in kind from the program. The program builds everything held bottom-up in rounds; the reference enumerates derivations top-down for one question; a third check uses a naive global fixpoint. Three algorithms agreeing is stronger evidence than a model of the same algorithm agreeing with its implementation. | 28,000 decisions across 200 databases between reference and fixpoint; 31,320 decisions across 150 databases between reference, stand-in and real runner; 0 mismatches. | Observed. Limit: one author read the prose rules once (T-34). |
| The checker is itself tested. Planted faults (one hop short or extra, a group, role or ancestor round short, denies ignored) were each detected on the real runner; the unmutated control was clean. | Mutation run on 24 databases on the real runner. | Observed. Hop mutants were detected on 10 of 24 databases only (T-33). Whether the other systems test their checkers was not established. |
| The explanation is compared, not just the answer. The chosen supporting grant (fewest hops, then smallest chain string) is part of the output that all three algorithms must reproduce, so equality covers why as well as whether. | The output contract of `check.py`. | Observed. No evidence either way was found about cross-checking explanations elsewhere. |
| Cost is a portable, reproducible quantity. The package is plain data; the stand-in and the real runner agreed on the step count exactly on three medium databases. | Bisecting the step limit on the real runner. | Observed on one real runtime. The spec does not fully pin the step table, so a second independent runtime is needed before this is claimed generally (T-15). |
| Validation and inference form one stack: Shaxon rejects data that breaks references or exceeds the fixed limits, Jaxson derives only from accepted data. | The design of waves 8 and 9. | Planned. Nothing is written (T-32). |

## 4. Where accessgraph is weaker

| Gap | Compared with |
|---|---|
| No formal proof of any property; assurance is testing only. | Cedar proves properties of a Lean model and its validator. |
| No policy analysis: nothing answers "can this policy set ever grant X" or "are these two sets equivalent". | Cedar's SMT-based analyses. |
| The rules are fixed in the generator, not authored by a user as policy. | Cedar, Biscuit and Rego all have an authoring surface. |
| Synthetic data only, no scale beyond the `medium` preset, no production use. | Cedar and SpiceDB run at scale. |
| One reading of the prose rules and one unreviewed choice on delegation (a delegator must hold the same role, not a role that inherits it). | Delegation Logic and RT give formal semantics for delegation. |
| Attribute conditions, time and caveats are absent. | Cedar and SpiceDB support them. |

## 5. Where we are going, why, and how

Ordered as the register orders them. Each item names the evidence it is meant
to change.

| Order | Item | Why | How |
|---|---|---|---|
| 1 | T-31 Fixed policy limits | Closes the structural weakness: today the program inherits its round counts from the checker, so the strongest claim in section 3 is not yet true. | Fix group nesting, role inheritance and resource depth as constants, as the hop bound already is. The reference evaluates under the same constants. Data beyond a constant is rejected, never truncated. State in the README that this is what totality costs and gives. |
| 2 | T-32 Shaxon data-integrity layer | The program fails on a dangling reference and the reference is undefined on one; the layer is what makes T-31's rejection real. | Add defect injection with a ground-truth list to the generator. The layer's findings must equal that list exactly, including none on a clean database. Mutate the layer and confirm the check notices. |
| 3 | T-33 Coverage | Delegation is about 6% of decisions and the hop bound decides about 0.4%; the least exercised part carries the most risk. | Tune generation; assert minimum counts for delegated decisions, hop-bound flips and denies without support; require each mutant to be detected on a stated share of databases. |
| 4 | T-34 Second reading | One author's reading of the prose is the shared assumption behind every agreement so far. | Obtain an independent reading: expectations derived by hand by another reader from the prose alone, or an encoding in a different engine (a Datalog or RT-style rendering). Record disagreements as questions about the rules. |
| 5 | T-35 Delegation review | The literal rule differs from what most systems allow. | Put it to a reviewer who administers real systems and record the decision and its reason. If the rule changes, re-derive the hand-built expectations by hand. |
| 6 | T-36 Scale | Without it, any statement about where the demonstration stops is unsupported. | Add a `large` preset; record steps and wall time on both runtimes after T-31. |

Candidates not yet in the register, so not commitments:

- A separation-of-duty policy layer: a Jaxson output listing the actions each
  user holds per resource, fed to a Shaxon rule against toxic combinations.
  It needs design first, because it changes the output shape, and it meets
  the existing limit that Shaxon checks cannot aggregate (T-18). RT's
  manifold roles are the reference treatment.
- A second independent runtime for the fixtures (T-15), which is also what
  would make the portability claim in section 3 general.

Open questions we have not decided: whether the rules should become authored
input instead of generator code, and whether any static analysis of the kind
Cedar offers belongs in this project at all.
