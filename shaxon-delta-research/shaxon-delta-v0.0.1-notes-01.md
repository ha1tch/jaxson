# Shaxon Delta v0.0.1

I'll call the proposed system **ΔG (Delta-G)** below.

## 1. Graphs are states

Let a graph be

$$
G=(V,E,\lambda_V,\lambda_E,\alpha)
$$

where:

* \(V\) is a finite set of persistent node identities;
* \(E\subseteq V\times L\times V\) is a set of directed labeled edges;
* \(\lambda_V\) and \(\lambda_E\) provide node/edge types if desired;
* \(\alpha\) contains attributes or values.

The important assumption is **persistent identity**.

If Alice exists in both \(G\) and \(H\), she is the same node. We do not solve graph differencing by first deciding that an arbitrary node in \(G\) "probably corresponds" to another node in \(H\).

Thus:

$$
G \xrightarrow{\Delta} H
$$

means that \(\Delta\) is a transformation whose source is \(G\) and whose result is \(H\).

---

# 2. A delta is a typed transition

The fundamental judgment is:

$$
G \vdash \Delta : H
$$

which means:

> Applying delta \(\Delta\) to \(G\) succeeds and produces \(H\).

Equivalently:

$$
\operatorname{apply}(G,\Delta)=H.
$$

I would make this the central semantic definition of the calculus.

A delta therefore isn't intrinsically just a set of additions and deletions. It is a **partial function on graphs**:

$$
\Delta : \mathcal G \rightharpoonup \mathcal G.
$$

The partiality is important. A delta can require things about its input.

For example:

$$
\Delta =
\operatorname{deleteEdge}(a,p,b)
$$

is only applicable to graphs containing that edge.

---

# 3. Primitive mutations

A minimal language could have these primitives:

$$
\begin{aligned}
\operatorname{addNode}(v,\tau)\\
\operatorname{delNode}(v)\\
\operatorname{addEdge}(e,u,p,v)\\
\operatorname{delEdge}(e)\\
\operatorname{setAttr}(x,k,a)\\
\operatorname{delAttr}(x,k)
\end{aligned}
$$

where \(e\) is itself persistent identity if edges need identity.

But I would distinguish **structural primitives** from **semantic mutations**.

For example:

$$
\operatorname{replaceEdge}(e,p,q)
$$

could be defined as a compound delta:

$$
\operatorname{delLabel}(e,p)
;
\operatorname{addLabel}(e,q).
$$

Yet a higher-level graph system may want `replaceEdge` to remain a first-class operation because it conveys information that two unrelated edits don't.

This leads to an important principle:

> **The primitive operations determine what the calculus considers a meaningful notion of change.**

---

# 4. Delta sequences

If

$$
\Delta_1:G_0\rightarrow G_1
$$

and

$$
\Delta_2:G_1\rightarrow G_2,
$$

we can compose them:

$$
\Delta_2\circ\Delta_1:G_0\rightarrow G_2.
$$

Application is then:

$$
\operatorname{apply}(G_0,\Delta_2\circ\Delta_1)
=
\operatorname{apply}(
\operatorname{apply}(G_0,\Delta_1),
\Delta_2).
$$

This gives us the first major law:

### Composition law

$$
\boxed{
\operatorname{apply}(G,\Delta_2\circ\Delta_1)
=
\operatorname{apply}(
\operatorname{apply}(G,\Delta_1),
\Delta_2)
}
$$

whenever both sides are defined.

This sounds obvious, but it is the foundation for almost everything else.

---

# 5. Identity delta

For every graph \(G\), there is an identity delta:

$$
1_G:G\rightarrow G.
$$

Its behavior is:

$$
\operatorname{apply}(G,1_G)=G.
$$

And:

$$
\boxed{
\Delta\circ1_G=\Delta
}
$$

and

$$
\boxed{
1_H\circ\Delta=\Delta
}
$$

for

$$
\Delta:G\rightarrow H.
$$

So graphs and deltas form something very close to a **category**:

* graphs are objects;
* deltas are morphisms;
* composition is delta composition;
* identity is the no-op delta.

This is not merely an analogy. It gives the calculus a very useful mathematical structure.

---

# 6. Associativity

If

$$
\Delta_1:G_0\to G_1,
$$

$$
\Delta_2:G_1\to G_2,
$$

$$
\Delta_3:G_2\to G_3,
$$

then:

$$
\boxed{
(\Delta_3\circ\Delta_2)\circ\Delta_1
=
\Delta_3\circ(\Delta_2\circ\Delta_1)
}
$$

provided the transformations are defined.

This is one of the most important laws.

It means we can regroup a sequence of mutations without changing its meaning.

---

# 7. `diff`

Now we introduce the operation you're particularly interested in:

$$
\operatorname{diff}(G,H)=\Delta
$$

such that

$$
G\xrightarrow{\Delta}H.
$$

The fundamental correctness law is:

$$
\boxed{
\operatorname{apply}(G,\operatorname{diff}(G,H))=H
}
$$

This should be non-negotiable.

A useful stronger law is:

$$
\boxed{
\operatorname{diff}(G,G)=1_G
}
$$

The really interesting question is what *kind* of diff we want.

---

# 8. Minimal diff

Suppose there is a cost function

$$
c:\Delta\rightarrow\mathbb R_{\geq0}.
$$

Then define:

$$
\operatorname{diff}_{\min}(G,H)
=
\arg\min_{\Delta}
\left\{
c(\Delta)
\mid
\operatorname{apply}(G,\Delta)=H
\right\}.
$$

This turns graph differencing into an optimization problem.

For example, suppose:

$$
G=
\{A\xrightarrow{p}B\}
$$

and

$$
H=
\{A\xrightarrow{q}B\}.
$$

A primitive calculus might produce:

$$
\Delta=
[
\operatorname{delete}(A,p,B),
\operatorname{add}(A,q,B)
].
$$

But a richer calculus might have:

$$
\Delta=
\operatorname{replaceProperty}(A,p,q,B).
$$

If the latter has cost 1 and the former cost 2, the minimum diff is the semantic replacement.

This gives a profound design principle:

> **Diff is relative to the mutation algebra.**

There isn't necessarily one mathematically privileged graph diff.

---

# 9. Canonicalization

There is another problem.

Two deltas can have exactly the same effect:

$$
\Delta_1:G\to H
$$

and

$$
\Delta_2:G\to H.
$$

We therefore distinguish **semantic equality** from syntactic equality.

Define:

$$
\Delta_1\equiv_G\Delta_2
$$

iff

$$
\operatorname{apply}(G,\Delta_1)
=
\operatorname{apply}(G,\Delta_2).
$$

A canonicalizer can then choose one representative:

$$
\operatorname{canon}(\Delta).
$$

Ideally:

$$
\operatorname{canon}(\operatorname{canon}(\Delta))
=
\operatorname{canon}(\Delta)
$$

and

$$
\operatorname{apply}(G,\operatorname{canon}(\Delta))
=
\operatorname{apply}(G,\Delta).
$$

This matters enormously for things like serialization, caching, hashing, version control, and reproducibility.

---

# 10. Inversion

For every successful delta

$$
\Delta:G\to H
$$

we want:

$$
\Delta^{-1}:H\to G.
$$

The defining law is:

$$
\boxed{
\operatorname{apply}
(H,\Delta^{-1})=G
}
$$

and therefore:

$$
\boxed{
\Delta^{-1}\circ\Delta=1_G
}
$$

and

$$
\boxed{
\Delta\circ\Delta^{-1}=1_H.
}
$$

Also:

$$
\boxed{
(\Delta^{-1})^{-1}=\Delta.
}
$$

And:

$$
\boxed{
(\Delta_2\circ\Delta_1)^{-1}
=
\Delta_1^{-1}\circ\Delta_2^{-1}.
}
$$

That reversal is essential.

---

# 11. Why inversion isn't automatically possible

There is an interesting subtlety.

Suppose we have:

$$
\operatorname{deleteNode}(A).
$$

Can we invert it?

Not unless the delta contains enough information to reconstruct \(A\).

So a mathematically invertible delta cannot simply say:

> delete \(A\).

It must contain the information necessary to restore what was deleted:

$$
\operatorname{deleteNode}
(
A,
\operatorname{snapshot}(A)
).
$$

Likewise:

$$
\operatorname{set}(x,k,new)
$$

needs to know the old value:

$$
\operatorname{set}(x,k,old,new).
$$

Thus an invertible calculus naturally distinguishes:

* **forward deltas**
* **reversible deltas**

This is a useful distinction for an implementation.

---

# 12. A more compact delta representation

For many graphs, a delta can be represented conceptually as:

$$
\Delta=(D,A,U,P)
$$

where:

* \(D\) = things deleted;
* \(A\) = things added;
* \(U\) = modifications;
* \(P\) = preconditions.

For example:

$$
\Delta=
\left(
\begin{array}{l}
D=\{(a,p,b)\}\\
A=\{(a,q,b)\}\\
U=\varnothing\\
P=\{(a,p,b)\in G\}
\end{array}
\right).
$$

This is essentially a generalized patch.

But the calculus should **not** identify a delta with this representation. The representation is an implementation detail; the semantic object is the partial graph transformation.

That allows richer deltas later.

---

# 13. Composition of deltas

Suppose:

$$
\Delta_1=
(A_1,D_1,U_1)
$$

and

$$
\Delta_2=
(A_2,D_2,U_2).
$$

Naïvely concatenating them gives:

$$
[
\Delta_1,\Delta_2
].
$$

But we can normalize.

For example:

$$
+\!x;\ -\!x
$$

might simplify to:

$$
\operatorname{identity}
$$

provided \(x\) did not exist initially.

Conversely:

$$
-\!x;\ +\!x
$$

is not necessarily the identity if the new \(x\) has different attributes.

Thus composition should preserve **temporal semantics**, not merely union the two sets.

The general law is:

$$
\boxed{
\operatorname{compose}(\Delta_1,\Delta_2)
\equiv
\Delta_2\circ\Delta_1
}
$$

and normalization may then produce a more compact equivalent delta.

---

# 14. Independence and commutation

Two deltas may be independent.

Define:

$$
\Delta_1\parallel\Delta_2
$$

if they can both be applied and neither changes anything on which the other depends.

Then we would like:

$$
\boxed{
\Delta_2\circ\Delta_1
\equiv
\Delta_1\circ\Delta_2
}
$$

when they are independent.

This is extremely useful.

Suppose:

$$
\Delta_1=\operatorname{add}(A,p,B)
$$

and

$$
\Delta_2=\operatorname{add}(C,q,D).
$$

They commute:

$$
\Delta_1\circ\Delta_2
\equiv
\Delta_2\circ\Delta_1.
$$

But:

$$
\Delta_1=\operatorname{delete}(A,p,B)
$$

and

$$
\Delta_2=\operatorname{delete}(A)
$$

do not commute in the ordinary sense because the second invalidates the first.

This gives us a formal foundation for **conflict detection**.

---

# 15. Merge should be three-way

I would *not* define merge simply as:

$$
\operatorname{merge}(\Delta_1,\Delta_2).
$$

Instead use:

$$
\operatorname{merge}(B,\Delta_L,\Delta_R)
$$

where:

$$
B\xrightarrow{\Delta_L}L
$$

and

$$
B\xrightarrow{\Delta_R}R.
$$

Thus we have:

```text
        ΔL
   B --------> L
   |
   | ΔR
   v
   R
```

The purpose of merge is to construct:

$$
M
$$

and ideally:

$$
B\xrightarrow{\Delta_M}M.
$$

---

# 16. The easiest merge case

Suppose:

$$
\Delta_L\parallel\Delta_R.
$$

Then:

$$
\boxed{
\operatorname{merge}(B,\Delta_L,\Delta_R)
=
\Delta_R\circ\Delta_L
}
$$

and because they commute:

$$
=
\Delta_L\circ\Delta_R.
$$

This gives the important law:

$$
\boxed{
\operatorname{merge}(B,\Delta_L,1_B)=\Delta_L
}
$$

and symmetrically:

$$
\boxed{
\operatorname{merge}(B,1_B,\Delta_R)=\Delta_R.
}
$$

---

# 17. Conflicts

Now suppose both branches modify the same property:

$$
B:
A.name="Alice"
$$

Left:

$$
A.name="Alicia"
$$

Right:

$$
A.name="Alison".
$$

We have:

$$
\Delta_L:
Alice\rightarrow Alicia
$$

and

$$
\Delta_R:
Alice\rightarrow Alison.
$$

They are not commutative.

So:

$$
\operatorname{merge}(B,\Delta_L,\Delta_R)
$$

should produce something like:

$$
\operatorname{Conflict}(C)
$$

rather than silently choosing one.

I would make this a fundamental principle:

> **Merge is not required to produce a graph. It may produce a conflict set.**

Formally:

$$
\operatorname{merge}:
(B,\Delta_L,\Delta_R)
\mapsto
\operatorname{MergedDelta}
\;|\;
\operatorname{ConflictSet}.
$$

That makes the calculus deterministic and avoids baking an arbitrary conflict-resolution policy into its mathematics.

---

# 18. Conflict objects

A conflict could be:

$$
C=(S,\Delta_L,\Delta_R)
$$

where \(S\) identifies the overlapping semantic region.

For example:

$$
C=
(
\operatorname{property}(A,name),
\Delta_L,
\Delta_R
).
$$

A merge operation can therefore report:

```text
conflict:
  subject: A
  property: name

  left:
    "Alice" -> "Alicia"

  right:
    "Alice" -> "Alison"
```

A higher-level system can then resolve it by producing a new delta.

This separation is important:

$$
\boxed{
\text{merge detects conflicts;}
\quad
\text{policy resolves conflicts.}
}
$$

---

# 19. Merge laws

Several desirable laws follow.

### Identity

$$
\boxed{
\operatorname{merge}(B,\Delta,1_B)=\Delta
}
$$

### Symmetry

If the branches are swapped, the result should be equivalent except for ordering of conflict information:

$$
\boxed{
\operatorname{merge}(B,\Delta_L,\Delta_R)
\equiv
\operatorname{merge}(B,\Delta_R,\Delta_L)
}
$$

where equivalence ignores left/right presentation.

### Agreement

If the two deltas make exactly the same change:

$$
\Delta_L\equiv\Delta_R,
$$

then:

$$
\boxed{
\operatorname{merge}(B,\Delta_L,\Delta_R)
\equiv
\Delta_L.
}
$$

### Non-interference

If the deltas affect disjoint regions:

$$
\Delta_L\parallel\Delta_R,
$$

then merge succeeds without conflict.

### Preservation

If merge produces \(\Delta_M\), then:

$$
M=\operatorname{apply}(B,\Delta_M)
$$

must contain the effects of both branches.

---

# 20. A crucial additional operation: `restrict`

For graph systems, I would add one operation beyond the five you requested:

$$
\operatorname{restrict}(\Delta,S).
$$

It asks:

> What part of this delta affects graph region \(S\)?

For example, if:

$$
\Delta
$$

changes ten nodes, then:

$$
\operatorname{restrict}(\Delta,\{A,B,C\})
$$

extracts the portion concerning those nodes.

This makes conflict analysis much more principled.

Two deltas conflict if their relevant restrictions overlap in an incompatible way.

---

# 21. Another useful operation: `impact`

We can define:

$$
\operatorname{impact}(\Delta)
$$

as the set of graph elements that the delta reads or writes.

For example:

$$
\operatorname{impact}(\Delta)
=
(\operatorname{read}(\Delta),
\operatorname{write}(\Delta)).
$$

Then an independence test can be expressed approximately as:

$$
\operatorname{write}(\Delta_1)
\cap
(\operatorname{read}(\Delta_2)\cup\operatorname{write}(\Delta_2))
=
\varnothing
$$

and vice versa.

This provides a formal bridge between the calculus and efficient implementations.

---

# 22. The complete algebra

We can now write the core interface as:

$$
\boxed{
\begin{aligned}
\operatorname{diff}&:G\times G\to\Delta\\
\operatorname{apply}&:G\times\Delta\rightharpoonup G\\
\operatorname{compose}&:\Delta\times\Delta\rightharpoonup\Delta\\
\operatorname{invert}&:\Delta\rightharpoonup\Delta\\
\operatorname{merge}&:G\times\Delta\times\Delta
\to\Delta+\operatorname{ConflictSet}.
\end{aligned}
}
$$

with:

$$
\operatorname{diff}(G,H):G\to H.
$$

---

# 23. The fundamental laws

I would regard these as the **axioms of the calculus**.

### Application / diff

$$
\boxed{
\operatorname{apply}
(G,\operatorname{diff}(G,H))
=H
}
$$

### Identity

$$
\boxed{
\operatorname{apply}(G,1_G)=G
}
$$

### Composition

$$
\boxed{
\operatorname{apply}
(G,\Delta_2\circ\Delta_1)
=
\operatorname{apply}
(
\operatorname{apply}(G,\Delta_1),
\Delta_2)
}
$$

### Associativity

$$
\boxed{
(\Delta_3\circ\Delta_2)\circ\Delta_1
=
\Delta_3\circ(\Delta_2\circ\Delta_1)
}
$$

### Identity composition

$$
\boxed{
\Delta\circ1_G=\Delta
}
$$

$$
\boxed{
1_H\circ\Delta=\Delta
}
$$

### Inverse

For \(\Delta:G\to H\):

$$
\boxed{
\Delta^{-1}:H\to G
}
$$

and:

$$
\boxed{
\Delta^{-1}\circ\Delta=1_G
}
$$

$$
\boxed{
\Delta\circ\Delta^{-1}=1_H.
}
$$

### Inverse of composition

$$
\boxed{
(\Delta_2\circ\Delta_1)^{-1}
=
\Delta_1^{-1}\circ\Delta_2^{-1}
}
$$

### Inverse of inverse

$$
\boxed{
(\Delta^{-1})^{-1}=\Delta
}
$$

### Diff identity

$$
\boxed{
\operatorname{diff}(G,G)=1_G
}
$$

### Merge identity

$$
\boxed{
\operatorname{merge}(G,\Delta,1_G)=\Delta
}
$$

### Merge agreement

$$
\boxed{
\Delta_L\equiv\Delta_R
\implies
\operatorname{merge}(G,\Delta_L,\Delta_R)
\equiv\Delta_L
}
$$

### Independent merge

$$
\boxed{
\Delta_L\parallel\Delta_R
\implies
\operatorname{merge}(G,\Delta_L,\Delta_R)
\equiv
\Delta_R\circ\Delta_L
}
$$

These give us a surprisingly substantial algebra.

---

# 24. But there is an important limitation: `diff` is not necessarily unique

Consider:

$$
G\rightarrow H.
$$

There may be:

$$
\Delta_1:G\to H
$$

and

$$
\Delta_2:G\to H.
$$

Both are valid.

Therefore:

$$
\operatorname{diff}(G,H)
$$

is only a true mathematical function if we impose a selection criterion.

We could define:

$$
\operatorname{diff}(G,H)
=
\operatorname{canonical}
\left(
\arg\min_\Delta c(\Delta)
\right).
$$

Then `diff` becomes deterministic.

But there is a deeper possibility:

$$
\boxed{
\operatorname{diff}(G,H)
=
\{\Delta\mid\Delta:G\to H\}.
}
$$

That is, `diff` could return the **space of all possible explanations of the change**.

Then:

* one delta might be the cheapest;
* another might be the most semantic;
* another might preserve provenance;
* another might maximize reuse of existing subgraphs.

For a sophisticated graph language, that distinction could be extremely valuable.

---

# 25. This suggests a two-level delta calculus

I would ultimately distinguish:

### Extensional delta

“What graph elements changed?”

$$
\Delta_{\mathrm{ext}}
$$

versus:

### Intentional delta

“What operation was performed?”

$$
\Delta_{\mathrm{int}}.
$$

For example:

$$
\Delta_{\mathrm{ext}}
=
-\{A\xrightarrow{p}B\}
+\{A\xrightarrow{q}B\}
$$

while:

$$
\Delta_{\mathrm{int}}
=
\operatorname{replaceRelation}(A,p,q,B).
$$

Both apply to the same graph transformation:

$$
G\xrightarrow{\Delta}H.
$$

But they preserve different information.

This is analogous to the distinction between:

> “these bytes changed”

and

> “the user renamed this field.”

The latter is vastly more informative.

---

# 26. This becomes particularly interesting for Shaxon

This formalism gives Shaxon a possible conceptual layer that is orthogonal to its current validation machinery.

Instead of thinking solely in terms of:

$$
G\models S
$$

you could have:

$$
G\xrightarrow{\Delta}G'
$$

and then potentially:

$$
(G,\Delta)\models C.
$$

That permits constraints on **changes**, not merely states.

For example, a constraint could express:

> Whenever a `status` changes from `draft` to `published`, a `publishedAt` value must simultaneously be introduced.

Formally:

$$
\Delta
\supset
\operatorname{set}(status,draft,published)
$$

implies:

$$
\Delta
\supset
\operatorname{set}(publishedAt,\_,t).
$$

Or:

> A mutation may not delete a node that is still referenced.

That's a property of the transition:

$$
G\xrightarrow{\Delta}G'
$$

rather than merely of either endpoint.

This is where the delta calculus becomes much more than a graph-diff utility.

---

# 27. The conceptual picture

The resulting structure is:

```text
                    diff
              G ------------> H
               \              ^
                \             |
                 \            | invert
                  \           |
                   Δ ---------+
                    \
                     \
                      apply
                       \
                        > H


        Δ₁                 Δ₂
   G ----------> G₁ ------------> G₂
         \________________________/
                  compose
```

And branching becomes:

```text
                    ΔL
               ┌──────────> L
               │
               B
               │
               └──────────> R
                    ΔR

                    │
                    ▼
                  merge

                    │
                    ▼

                    M
```

So the fundamental mathematical object isn't actually the graph.

It is the **arrow between graphs**:

$$
\boxed{G\xrightarrow{\Delta}H}
$$

Graphs are states; deltas are transformations; composition is temporal sequencing; inversion is reversal; merge is reconciliation of divergent histories.

That gives you a genuine **calculus of graph evolution**, rather than merely a graph-difference algorithm.

And, importantly for Shaxon, it provides a clean place to put a future notion of **mutation validation**: constraints can be predicates over \(G\), predicates over \(G'\), or predicates over the transition \(G\xrightarrow{\Delta}G'\).

Refine the delta calculus

* Formalize merge conflicts
* Add preconditions and postconditions
