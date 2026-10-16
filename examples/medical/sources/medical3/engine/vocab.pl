% vocab.pl - core relation vocabulary, property mapping, evidence strengths, guards, schemas.
% Copyright (c) 2026 haitch <h@ual.li>  https://oldbytes.space/@haitchfive
% SPDX-License-Identifier: Apache-2.0  (https://www.apache.org/licenses/LICENSE-2.0)
%
% Everything in this file is data. Schemas contain only rel/3 goals and guard/1 goals
% from the closed guard vocabulary below, never arbitrary Prolog. New relation classes
% are added from other files (see kb/ext_relations.pl) using the same predicates.
% engine/validate.pl checks all of it statically at every ingest.

:- encoding(utf8).

:- multifile rel_pred/2, rel_axiom/1, compose_rule/3, prop_map/3, schema/4, alias/2, bridge/4,
             source_strength/2.
:- dynamic   rel_pred/2, rel_axiom/1, compose_rule/3, prop_map/3, schema/4, alias/2, bridge/4,
             source_strength/2.
:- discontiguous schema/4.

% rel_pred(Name, Options)
%   kind(base)     stored as rel/4 facts
%   kind(derived)  produced only by compose_rule/3, rel_axiom/1 or schema/4
%   polarity(P)    positive, negative or neutral; used by static validation
%   headline(true) reported as a conclusion (benchmark and impact analysis)
rel_pred(has_function,            [kind(base),    polarity(neutral),  doc("Entity performs a function")]).
rel_pred(contains,                [kind(base),    polarity(positive), doc("Subject is constituted by, or contains, the object")]).
rel_pred(provides,                [kind(base),    polarity(positive), doc("Subject supplies the object substance")]).
rel_pred(inhibits,                [kind(base),    polarity(negative), doc("Subject reduces the effect or absorption of the object")]).
rel_pred(enhances,                [kind(base),    polarity(positive), doc("Subject increases the effect or absorption of the object")]).
rel_pred(deficiency_manifests_as, [kind(base),    polarity(neutral),  doc("Deficiency of subject may show as the object symptom")]).
rel_pred(indicated_for,           [kind(base),    polarity(positive), doc("Product is labelled for the object concept")]).
rel_pred(is_a,                    [kind(base),    polarity(neutral),  doc("Concept subsumption, e.g. headache is_a pain")]).
rel_pred(may_address,             [kind(derived), polarity(positive), headline(true), doc("Product may address symptom (never a diagnosis)")]).
rel_pred(candidate_cause,         [kind(derived), polarity(neutral),  headline(true), doc("Symptom has this deficiency as a candidate cause")]).
rel_pred(duplicates,              [kind(derived), polarity(neutral),  headline(true), doc("Two products supply the same active")]).
rel_pred(interferes_with,         [kind(derived), polarity(negative), headline(true), doc("An active in one product inhibits an active in the other")]).
rel_pred(boosts,                  [kind(derived), polarity(positive), headline(true), doc("An active in one product enhances an active in the other")]).
rel_pred(active_in,               [kind(derived), polarity(positive), doc("Substance or compound is active in product")]).
rel_pred(promotes,                [kind(derived), polarity(positive), doc("Only derivable if a compose_rule declares it")]).

% prop_map(PropertyFunctor, Predicate, Mode): how a one-argument tosid property becomes rel/4.
%   concept     object is concept(Value) for each value
%   entity      object is the entity the value label resolves to
%   entity_inv  the resolved entity is the SUBJECT and the owning entity is the object
prop_map(function,             has_function,            concept).
prop_map(biological_role,      has_function,            concept).
prop_map(deficiency_symptoms,  deficiency_manifests_as, concept).
prop_map(indication,           indicated_for,           concept).
prop_map(contains_element,     contains,                entity).
prop_map(active_ingredient,    contains,                entity).
prop_map(active_ingredients,   contains,                entity).

% Status lattice, strongest first. Proofs produce supported and conditional; apply_defeat/4
% (solve.pl) produces conflicted and refuted by weighing declared defeaters.
status_rank(supported,   4).
status_rank(conditional, 3).
status_rank(conflicted,  2).
status_rank(unknown,     1).
status_rank(refuted,     0).

% Evidence strength. Every stored fact carries one ev(Source, Strength, Detail) entry per
% source that asserted it (a second source is merged, never dropped). Strength is the
% declared default for the source unless the asserting term carries status(S).
% A source with no entry here is treated as conditional.
source_strength(kb,         supported).
source_strength(curated,    supported).
source_strength(fixture,    supported).
source_strength(hypothesis, conditional).

% rel_axiom(Axiom) - interpreted generically by the solver
%   symmetric(P)       P(a,b) gives P(b,a)                  (P base)
%   transitive(P)      closure over P, cycle safe           (P base)
%   inverse(P,Q)       P(a,b) is DEFINED as Q(b,a)          (P derived; one direction only)
%   subrel(P,Q)        P(a,b) gives Q(a,b)
%   defeats(D,T)       D(a,b) weighs against T(a,b) in assess/3
% (none in the core vocabulary)

% compose_rule(P1, P2, P3): rel(A,P1,B) and rel(B,P2,C) give rel(A,P3,C).
% Nothing composes unless declared. In particular inhibits o inhibits does not.
compose_rule(contains, provides, provides).

% Closed guard vocabulary usable inside schema bodies as guard(G).
% A guard may only use variables already bound by an earlier rel/3 goal (validated).
guard_spec(distinct,      2).
guard_spec(before,        2).
guard_spec(is_product,    1).
guard_spec(entity_domain, 2).

% schema(Name, Head, Body, Status)
% Status is the ceiling this step imposes; a proof is only as strong as its weakest step.

schema(candidate_cause,
       rel(concept(Sym), candidate_cause, S),
       [rel(S, deficiency_manifests_as, concept(Sym))],
       conditional).

schema(self_provision,
       rel(S, provides, S),
       [rel(S, deficiency_manifests_as, _)],
       supported).

schema(treat_direct,
       rel(P, may_address, concept(Sym)),
       [rel(S, deficiency_manifests_as, concept(Sym)), rel(P, provides, S), guard(is_product(P))],
       conditional).

schema(treat_constituent,
       rel(P, may_address, concept(Sym)),
       [rel(E, deficiency_manifests_as, concept(Sym)), rel(E, contains, S),
        rel(P, provides, S), guard(is_product(P))],
       conditional).

schema(treat_indicated,
       rel(P, may_address, concept(Sym)),
       [rel(P, indicated_for, concept(Sym)), guard(is_product(P))],
       supported).
schema(treat_indicated_class,
       rel(P, may_address, concept(Sym)),
       [rel(P, indicated_for, C), rel(concept(Sym), is_a, C), guard(is_product(P))],
       supported).

schema(active_compound,  rel(P, active_in, X), [rel(P, contains, X)], supported).
schema(active_substance, rel(P, active_in, X), [rel(P, provides, X)], supported).

schema(duplicate_therapy,
       rel(P1, duplicates, P2),
       [rel(P1, active_in, X), rel(P2, active_in, X),
        guard(before(P1, P2)), guard(is_product(P1)), guard(is_product(P2))],
       supported).
schema(interference,
       rel(P1, interferes_with, P2),
       [rel(P1, active_in, X), rel(P2, active_in, Y), rel(X, inhibits, Y),
        guard(distinct(P1, P2)), guard(is_product(P1)), guard(is_product(P2))],
       supported).
schema(synergy,
       rel(P1, boosts, P2),
       [rel(P1, active_in, X), rel(P2, active_in, Y), rel(X, enhances, Y),
        guard(distinct(P1, P2)), guard(is_product(P1)), guard(is_product(P2))],
       supported).
