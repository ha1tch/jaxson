% ext_relations.pl - relation classes added as KNOWLEDGE ONLY. No engine file mentions
% any of these predicates. Each relation gets its meaning from declared axioms
% (symmetric, transitive, inverse, subrel, defeats), composition rules and prop_map
% entries, all interpreted by the generic engine.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0

:- encoding(utf8).
:- multifile rel_pred/2, rel_axiom/1, compose_rule/3, prop_map/3.
:- dynamic rel_pred/2, rel_axiom/1, compose_rule/3, prop_map/3.

rel_pred(competes_with,          [kind(base),    polarity(negative), doc("Mutual competition, e.g. for absorption")]).
rel_pred(precursor_of,           [kind(base),    polarity(positive), doc("Subject is a metabolic precursor of the object")]).
rel_pred(derived_from,           [kind(derived), polarity(positive), doc("Object is a precursor of the subject")]).
rel_pred(supplies_precursor_of,  [kind(derived), polarity(positive), headline(true), doc("Product contains a precursor of the object")]).
rel_pred(not_indicated_for,      [kind(base),    polarity(negative), doc("Product label states it is not for the object concept")]).

% competition is mutual, and counts as inhibition, so every inhibition-based schema
% (the interference check) sees it without being edited
rel_axiom(symmetric(competes_with)).
rel_axiom(subrel(competes_with, inhibits)).

% precursor chains close transitively; derived_from is defined as the inverse
rel_axiom(transitive(precursor_of)).
rel_axiom(inverse(derived_from, precursor_of)).

% a supported not_indicated_for statement weighs against may_address (see assess/3)
rel_axiom(defeats(not_indicated_for, may_address)).

% a product containing X supplies whatever X is a (transitive) precursor of
compose_rule(contains, precursor_of, supplies_precursor_of).

% ingestion of properties already present in the legacy knowledge bases
prop_map(precursor,         precursor_of,      entity_inv).
prop_map(not_indicated_for, not_indicated_for, concept).
