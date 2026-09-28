# Shaxon Delta v0.0.1

The cleanest way to position Shaxon is that its **graph mutation/delta model sits between graph editing and graph transformation**, but currently does not go as far as the algebraic graph-transformation literature in making transformations themselves a mathematically rich object.

The distinction is important because Shaxon appears to be aiming at something slightly different from classical graph rewriting: a **deterministic, bounded, explicitly reachable graph computation in which validation remains pure and mutation is a separate state transition**.

## 1. The basic correspondence

The correspondence can be stated quite compactly:

| Shaxon                        | Graph-transformation terminology                 |
| ----------------------------- | ------------------------------------------------ |
| Graph/state                   | Host graph                                       |
| Node/resource identity        | Persistent graph object                          |
| Triple/property/relationship  | Edge or attributed edge                          |
| Mutation                      | Graph transformation                             |
| Delta                         | Transformation instance / derivation             |
| Mutation preconditions        | Match + application conditions                   |
| Applying mutation             | Rule application                                 |
| Sequence of mutations         | Derivation / transformation sequence             |
| `diff(G,H)`                   | Transformation/derivation from \(G\) to \(H\)    |
| Delta composition             | Composition/sequential transformation            |
| Inverse delta                 | Reverse transformation, when reversible          |
| Mutation conflict             | Critical overlap / incompatible rule application |
| Validation of resulting graph | Constraint checking on the target graph          |
| Validation of mutation        | Constraint on a transformation/derivation        |

But there are significant differences between the four traditions you asked about.

---

# 2. Shaxon ↔ graph transformation systems

A **graph transformation system (GTS)** generally consists of:

$$
G \xRightarrow{r,m} H
$$

where:

* \(r\) is a transformation rule;
* \(m\) is a match of the rule's left-hand side into \(G\);
* applying \(r\) at \(m\) produces \(H\).

A rule is conventionally represented as something like:

$$
L \leftarrow K \rightarrow R
$$

where:

* \(L\) is what must exist;
* \(R\) is what will exist afterward;
* \(K\) is what survives the transformation.

### Shaxon's mutation is naturally an instance of this

Suppose Shaxon says, conceptually:

```text
delete (A, p, B)
add    (A, q, B)
```

Then we can model it as:

$$
L=
\{A\xrightarrow{p}B\}
$$

and

$$
R=
\{A\xrightarrow{q}B\}.
$$

The mutation is:

$$
G
\xRightarrow{m}
H.
$$

So **Shaxon's mutation corresponds very naturally to a graph-transformation step**.

The interesting part is that Shaxon's explicit reach/read/write machinery could be interpreted as a particularly constrained form of application condition. Its emphasis on explicit reach and bounded computation makes it substantially less permissive than a general-purpose GTS.

### The important difference

Classical GTS asks:

> Does there exist a match \(m\) to which this rule can be applied?

Shaxon tends toward:

> Here is the explicitly reachable portion of the graph; perform this deterministic operation there.

That is a significant semantic restriction.

It means Shaxon isn't naturally a nondeterministic rewriting system of the form

$$
G\Rightarrow G_1
$$

where many rules or matches may be possible.

Instead it is closer to:

$$
(G,\Delta)\mapsto G'
$$

with explicitly bounded execution.

That fits Shaxon's design goals considerably better than unrestricted rewriting.

---

# 3. Shaxon ↔ graph rewriting

Graph rewriting is the broader conceptual family.

A rewriting rule says:

$$
L\rightarrow R.
$$

Find \(L\) in the graph and replace it by \(R\).

This is almost exactly the mental model behind a Shaxon mutation.

But graph rewriting brings several concepts that are useful for Shaxon.

## Match

A pattern \(L\) is matched against \(G\):

$$
m:L\hookrightarrow G.
$$

For Shaxon, this corresponds to identifying the resources/relationships on which the mutation operates.

## Rewrite

Then:

$$
G\xRightarrow{r,m}G'.
$$

This corresponds to applying the Shaxon mutation.

## Derivation

Multiple mutations give:

$$
G_0
\xRightarrow{\Delta_1}
G_1
\xRightarrow{\Delta_2}
G_2
\xRightarrow{\Delta_3}
G_3.
$$

This is precisely a rewriting derivation.

---

# 4. Where Shaxon is actually more disciplined than ordinary rewriting

General rewriting systems frequently allow:

$$
G\Rightarrow G_1
$$

and

$$
G\Rightarrow G_2
$$

from the same state.

That creates questions about:

* nondeterminism;
* confluence;
* termination;
* rule ordering;
* critical pairs.

Shaxon's bounded/deterministic orientation pushes strongly in the opposite direction.

Ideally:

$$
(G,\Delta)\rightarrow G'
$$

has exactly one result.

That gives Shaxon a useful property:

$$
\boxed{
\operatorname{apply}(G,\Delta)=G'
}
$$

is a function, rather than a relation.

This is a major conceptual distinction.

A general rewriting system is often modeled as:

$$
\Rightarrow\subseteq G\times G.
$$

A Shaxon mutation engine is better modeled as a partial function:

$$
\operatorname{apply}:\mathcal G\times\Delta\rightharpoonup\mathcal G.
$$

That makes Shaxon closer to a **deterministic rewriting machine** than to an unconstrained graph-rewriting system.

---

# 5. Shaxon ↔ graph edit distance

The relationship here is quite different.

Graph edit distance asks:

$$
d(G,H)=
\min_{\Delta:G\to H}
c(\Delta).
$$

It therefore treats a mutation primarily as a **cost-bearing sequence of elementary edits**.

Shaxon's delta can serve as precisely such an edit sequence.

For example:

$$
G
\xrightarrow{
\{
-\,(A,p,B),
+\,(A,q,B)
\}
}
H.
$$

Then:

$$
d(G,H)\leq 2
$$

if deletion and insertion each cost one.

But this exposes a fundamental limitation of ordinary graph edit distance:

### The representation of the delta determines the distance.

Suppose Shaxon recognizes:

$$
\operatorname{replace}(A,p,q,B)
$$

as one semantic mutation.

Then:

$$
c(\operatorname{replace})=1
$$

could give:

$$
d(G,H)=1.
$$

A primitive edit-distance system might instead give:

$$
d(G,H)=2.
$$

Neither is mathematically "the" correct answer.

They are measuring different notions of change.

This is particularly relevant to Shaxon because its mutation model can preserve **semantic intent**, whereas graph edit distance traditionally cares primarily about the resulting edit structure.

---

# 6. This suggests two distinct Shaxon distances

If Shaxon ever wanted a formal notion of graph difference, I would distinguish:

### Structural distance

$$
d_s(G,H)
$$

based on primitive graph modifications.

### Semantic mutation distance

$$
d_m(G,H)
$$

based on Shaxon's higher-level mutation vocabulary.

Then:

$$
d_s(G,H)
$$

answers:

> How many low-level graph changes are necessary?

while:

$$
d_m(G,H)
$$

answers:

> How many Shaxon mutations describe the transition?

Those can legitimately differ.

That is actually a feature, not a problem.

---

# 7. Shaxon ↔ algebraic graph transformation

This is the deepest correspondence.

Algebraic graph transformation gives a mathematically rigorous way of representing graph rewriting using constructions such as **pushouts** and **pushout complements**.

The basic rule is:

$$
L \xleftarrow{l} K \xrightarrow{r} R.
$$

Conceptually:

```text
       L
       ↑
       K
       ↓
       R
```

\(K\) represents the part preserved by the transformation.

Suppose a match is:

$$
m:L\rightarrow G.
$$

Then the transformation constructs the result through categorical constructions.

Very roughly:

$$
G
\quad\leadsto\quad
H
$$

by removing the part corresponding to \(L-K\) and adding the part \(R-K\).

---

# 8. Shaxon mutations have a natural \(L/K/R\) decomposition

This is where I think the mapping becomes particularly fruitful.

Take:

$$
G:
A\xrightarrow{p}B
$$

and mutation:

$$
A\xrightarrow{p}B
\quad\longrightarrow\quad
A\xrightarrow{q}B.
$$

Then:

$$
L=
\{A\xrightarrow{p}B\}
$$

$$
K=
\{A,B\}
$$

$$
R=
\{A\xrightarrow{q}B\}.
$$

The nodes survive; the relationship changes.

So:

$$
L-K=\{\text{old edge}\}
$$

and

$$
R-K=\{\text{new edge}\}.
$$

That is exactly the conceptual structure of a graph delta:

$$
\boxed{
\Delta =
(\text{preserve},\text{delete},\text{add})
}
$$

with \(K\) corresponding to the preserved portion.

This gives a much more rigorous mathematical interpretation to what otherwise looks like an ordinary patch.

---

# 9. Deletion is where algebraic transformation becomes especially interesting

Suppose:

$$
G=
\{A\xrightarrow{p}B,\;
C\xrightarrow{q}A\}
$$

and a mutation wants to delete \(A\).

A naïve delta says:

$$
-\!A.
$$

But what happens to:

$$
C\xrightarrow{q}A?
$$

Algebraic graph transformation forces us to confront this structurally.

The transformation needs a valid **gluing condition**: can \(A\) actually be removed without leaving dangling edges?

This is closely related to the **dangling condition** in the double-pushout approach.

For Shaxon, this suggests a useful formal distinction between:

> "delete this node"

and

> "delete this node and all incident relationships."

The latter is a different transformation.

That distinction can prevent a surprisingly large class of ambiguous mutation semantics.

---

# 10. DPO and SPO give two possible interpretations for Shaxon

Algebraic graph transformation has two particularly important approaches.

### Double-pushout (DPO)

The transformation is essentially:

$$
L\leftarrow K\rightarrow R
$$

with deletion performed carefully so that dangling structures aren't silently destroyed.

### Single-pushout (SPO)

Deletion can have more implicit consequences, with incident structure potentially disappearing as part of the transformation.

For Shaxon, this is not merely academic.

It forces a language-design decision:

> When Shaxon deletes a graph object, are relationships incident to that object automatically deleted, or must the mutation explicitly specify them?

A stricter, more explicit Shaxon would lean toward the DPO philosophy.

A convenience-oriented mutation language might choose SPO-like semantics.

Given Shaxon's existing emphasis on **explicit reach, boundedness, determinism, and predictable computation**, the DPO-style discipline seems conceptually closer to its existing design.

---

# 11. Where `diff` fits in algebraic graph transformation

This produces a particularly elegant interpretation of `diff`.

Given:

$$
G,H,
$$

we want to discover a transformation:

$$
G\xRightarrow{\Delta}H.
$$

In algebraic terms, we are looking for:

$$
L\leftarrow K\rightarrow R
$$

plus a match:

$$
L\rightarrow G
$$

such that the resulting transformation is \(H\).

So:

$$
\boxed{
\operatorname{diff}(G,H)
=
\text{find a transformation span explaining }G\to H
}
$$

rather than simply:

$$
G-H.
$$

This is substantially richer.

---

# 12. `compose` is where the categorical formulation pays off

Suppose:

$$
G\xrightarrow{\Delta_1}H
$$

and

$$
H\xrightarrow{\Delta_2}K.
$$

Then:

$$
\Delta_2\circ\Delta_1
$$

is a transformation:

$$
G\to K.
$$

The ordinary patch implementation might concatenate two lists of operations.

The algebraic formulation asks whether the transformations themselves can be **composed structurally**.

That opens the door to transformation laws such as:

$$
(\Delta_3\circ\Delta_2)\circ\Delta_1
=
\Delta_3\circ(\Delta_2\circ\Delta_1).
$$

So the category-like structure we discussed previously isn't accidental: **algebraic graph transformation gives Shaxon's delta algebra a natural mathematical home.**

---

# 13. `invert` is more subtle

Graph rewriting does not automatically imply reversibility.

A transformation:

$$
G\rightarrow H
$$

can lose information.

For example:

$$
\operatorname{deleteNode}(A)
$$

cannot be inverted unless the deleted structure is retained somewhere.

So Shaxon would need one of two designs.

### Reversible delta

Store sufficient information:

$$
\Delta=
(\text{delete},\text{old state})
$$

so that:

$$
\Delta^{-1}:H\rightarrow G.
$$

### Irreversible mutation

Treat:

$$
\Delta:G\rightarrow H
$$

as non-invertible.

This corresponds closely to the difference between a general graph rewrite and an isomorphism-like reversible transformation.

For a proper delta calculus, I would make reversibility **explicit in the delta's type**, rather than pretending every mutation has an inverse.

---

# 14. `merge` connects to rewriting theory in a particularly beautiful way

Suppose:

$$
G
\xrightarrow{\Delta_L}
L
$$

and

$$
G
\xrightarrow{\Delta_R}
R.
$$

These are two competing transformations.

Graph transformation theory asks whether the two transformations **overlap**.

This leads to the theory of **critical pairs**.

Conceptually:

$$
G
\begin{array}{ccc}
&\searrow^{\Delta_L}&L\\
G&&\\
&\swarrow_{\Delta_R}&R
\end{array}
$$

The important question is whether there exists some common graph \(M\) such that:

$$
L\Rightarrow M
$$

and

$$
R\Rightarrow M.
$$

If so, the divergent transformations may be **joinable**.

This is remarkably close to graph-delta merge.

---

# 15. Confluence gives a theoretical foundation for merge

A rewriting system is confluent when divergent transformations can eventually be reconciled:

$$
G\Rightarrow^*L
$$

and

$$
G\Rightarrow^*R
$$

implies there exists \(M\) such that:

$$
L\Rightarrow^*M
$$

and

$$
R\Rightarrow^*M.
$$

For Shaxon:

> Two mutations can be merged without conflict when their divergence is joinable.

That suggests a sophisticated definition of merge:

$$
\operatorname{merge}(G,\Delta_L,\Delta_R)
$$

could search for a common successor rather than simply comparing changed triples.

This is a substantially stronger foundation than ordinary three-way textual merge.

---

# 16. Critical pairs could become Shaxon mutation conflicts

Suppose:

$$
\Delta_1:
A.x\leftarrow 2
$$

and:

$$
\Delta_2:
A.x\leftarrow 3.
$$

They overlap on:

$$
(A,x).
$$

There is no obvious common result without an additional policy.

That is a critical pair.

By contrast:

$$
\Delta_1:
A.x\leftarrow2
$$

and

$$
\Delta_2:
B.y\leftarrow3
$$

are structurally independent.

They can commute:

$$
\Delta_1\circ\Delta_2
=
\Delta_2\circ\Delta_1.
$$

That gives Shaxon a mathematically principled definition of mutation independence.

---

# 17. The four disciplines answer four different questions

This is perhaps the most useful way to think about the mapping.

### Graph edit distance

$$
\boxed{\text{How much did the graph change?}}
$$

It supplies:

* edit costs;
* minimal edit sequences;
* similarity/distance;
* optimization.

### Graph rewriting

$$
\boxed{\text{What transformation rule changes the graph?}}
$$

It supplies:

* patterns;
* rewrite rules;
* matches;
* derivations;
* termination/confluence theory.

### Graph transformation systems

$$
\boxed{\text{What transformations are available and how do they execute?}}
$$

It supplies:

* a system of rules;
* application conditions;
* transformation sequences;
* state-space semantics.

### Algebraic graph transformation

$$
\boxed{\text{What is the mathematically precise structure of the transformation?}}
$$

It supplies:

* \(L/K/R\) transformation structure;
* categorical composition;
* pushouts;
* pushout complements;
* gluing conditions;
* formal concurrency/conflict theory.

And Shaxon's delta model can potentially sit above all four.

---

# 18. A useful conceptual stack for Shaxon

I would therefore model Shaxon as five layers:

```text
                    SHAXON
                       │
              ┌────────┴────────┐
              │                 │
           Graph state       Delta state
              │                 │
              │          ┌──────┴──────┐
              │          │             │
              │       primitive     semantic
              │       edits         mutations
              │          │             │
              └──────────┴─────────────┘
                         │
                  Graph transformation
                         │
              ┌──────────┴──────────┐
              │                     │
          rewriting             algebraic
          semantics            transformation
              │                     │
              └──────────┬──────────┘
                         │
                    diff / merge
                         │
                 graph comparison
```

The key point is that **graph edit distance should not define Shaxon's mutation semantics**.

Rather:

$$
\boxed{
\text{Shaxon mutation}
\longrightarrow
\text{graph transformation}
\longrightarrow
\text{optional edit-distance interpretation}
}
$$

not the other way around.

---

# 19. What Shaxon appears to have that classical rewriting doesn't emphasize

There is an interesting inversion here.

Classical graph rewriting starts with:

> Here are rules that transform graphs.

Shaxon starts from a much more constrained computational environment:

> Here is an explicitly bounded graph computation with explicit reach/read/write behavior, deterministic execution, and pure validation.

That means Shaxon can potentially treat graph transformations as **controlled computational effects** rather than as an unconstrained rewriting universe.

In particular, you can give every mutation:

$$
\Delta:
(G,\text{environment})
\longrightarrow
(G',\text{environment}')
$$

but constrain:

* which graph region it may inspect;
* which region it may modify;
* how much computation it may perform;
* whether it is deterministic;
* whether validation can observe or modify it.

This is a rather distinctive combination.

---

# 20. The most interesting extension: constraints over transformations

This is where I think the connection becomes genuinely significant for Shaxon.

Ordinary SHACL-style validation is essentially concerned with:

$$
G\models C.
$$

Graph transformation gives you:

$$
G\xrightarrow{\Delta}G'.
$$

Shaxon could conceptually support three different constraint domains:

### State constraint

$$
C_{\text{before}}(G)
$$

### State constraint

$$
C_{\text{after}}(G')
$$

### Transition constraint

$$
C_{\Delta}(G,\Delta,G').
$$

The third is the interesting one.

For example:

$$
\boxed{
\text{if }\Delta\text{ changes }status:draft\to published,
\text{ then }\Delta\text{ must create publishedAt}
}
$$

That's not naturally a property of \(G\) or \(G'\) alone.

It's a property of the **transformation**.

And graph-transformation theory gives you an established vocabulary for precisely this kind of reasoning.

---

# 21. My recommended formal position for Shaxon

If we were formalizing Shaxon rather than merely comparing it to existing mathematics, I would **not** define its delta model as "graph edit distance."

I would define it approximately as:

$$
\boxed{
\text{Shaxon Delta}
=
\text{typed, bounded, deterministic graph transformation}
}
$$

with:

$$
\Delta:G\rightarrow H
$$

and an optional decomposition:

$$
\Delta=(L,K,R,m,\Gamma)
$$

where:

* \(L\) = required/read pattern;
* \(K\) = preserved structure;
* \(R\) = resulting structure;
* \(m\) = explicit match/binding;
* \(\Gamma\) = additional preconditions/application conditions.

Then define:

$$
\operatorname{apply}(G,\Delta)=H
$$

as the operational semantics.

On top of that:

$$
\operatorname{diff}(G,H)
$$

finds a suitable \(\Delta\);

$$
\operatorname{compose}(\Delta_1,\Delta_2)
$$

forms sequential transformations;

$$
\operatorname{invert}(\Delta)
$$

exists only for reversible deltas;

and:

$$
\operatorname{merge}(G,\Delta_1,\Delta_2)
$$

uses overlap/conflict analysis derived from graph-transformation theory.

Finally, graph edit distance becomes merely one possible **cost model over Shaxon deltas**:

$$
d_c(G,H)
=
\min_{\Delta:G\to H}c(\Delta).
$$

That separation is powerful because it prevents a numerical notion of "distance" from dictating what a mutation *means*.

---

## The deepest correspondence

I think the most illuminating formulation is this:

$$
\boxed{
\begin{array}{ccc}
\textbf{Shaxon} & & \textbf{Graph transformation theory}\\[3pt]
G &\leftrightarrow& \text{host graph}\\
\Delta &\leftrightarrow& \text{transformation/derivation}\\
\operatorname{apply} &\leftrightarrow& \text{rule application}\\
\operatorname{compose} &\leftrightarrow& \text{sequential derivation}\\
\operatorname{invert} &\leftrightarrow& \text{reversible transformation}\\
\operatorname{merge} &\leftrightarrow& \text{concurrency/confluence}\\
\operatorname{diff} &\leftrightarrow& \text{discovery of a transformation}\\
\operatorname{cost}(\Delta) &\leftrightarrow& \text{graph edit metric}
\end{array}
}
$$

So I would characterize Shaxon not as reinventing graph rewriting, but as potentially providing a **language-level, deterministic, bounded delta calculus whose mathematical semantics can be grounded in algebraic graph transformation**.

That is a much stronger position than simply calling its mutations "graph edits." It gives you established mathematics for composition, identity, reversibility, overlap, conflicts, concurrency, and ultimately even confluence—while leaving Shaxon free to impose its own restrictions around determinism, explicit reach, boundedness, and validation purity.

I can also create an image showing this Shaxon ↔ graph-transformation correspondence as a formal diagram.
