% fixtures.pl - knowledge-only additions for Experiment 1. No engine edits.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% tosid/5 terms are in the legacy format, so the same fixtures run against the old
% coordinator (legacy_baseline.pl). bridge/alias/compose_rule terms are the new
% engine's declarative equivalents and are knowledge, not code.
% These fixtures test inference validity, not medical truth.

:- encoding(utf8).
:- multifile fixture/2.

fixture(iron_fumarate, [
    tosid('10','F2','FER-FUM-CMP:FE1-FUM-F02','Ferrous Fumarate',
          [type(compound), domain(chemistry), chemical_class(iron_salt), mechanism(iron_replacement)]),
    tosid('10','E4','IRN-FUM-100:FER-FUM-S02','Iron Fumarate 100mg',
          [type(pharmaceutical), domain(pharma), active_ingredient('ferrous_fumarate'),
           strength_mg(100), formulation(tablet), indication([iron_deficiency, anemia]), availability(otc)]),
    bridge('Ferrous Fumarate', provides, 'Iron', [source(fixture)])
]).

fixture(zinc, [
    tosid('10','F2','ZIN-SUL-CMP:ZN1-SO4-F03','Zinc Sulfate',
          [type(compound), domain(chemistry), molecular_formula('ZnSO4'), chemical_class(zinc_salt)]),
    tosid('10','E4','ZIN-TAB-50:ZIN-SUL-S03','Zinc 50mg',
          [type(pharmaceutical), domain(pharma), active_ingredient('zinc_sulfate'),
           strength_mg(50), formulation(tablet), indication([zinc_deficiency, immune_support]), availability(otc)]),
    bridge('Zinc Sulfate', provides, 'Zinc', [source(fixture)])
]).

% Folate shares the function label dna_synthesis with B12: the legacy engine joined on it.
fixture(folate, [
    tosid('01','B8','FOL-ATE-VIT:B9-VIT-C02','Folate',
          [type(vitamin), domain(biochemical), function(dna_synthesis),
           deficiency_symptoms([glossitis, fatigue])]),
    tosid('10','E4','FOL-TAB-1MG:FOL-ACI-B02','Folic Acid 1mg',
          [type(pharmaceutical), domain(pharma), active_ingredient('folic_acid'),
           strength_mg(1), formulation(tablet), indication([folate_deficiency]), availability(otc)]),
    alias(folic_acid, 'Folate')
]).

% Vitamin K has a deficiency profile but no product.
fixture(vitamin_k, [
    tosid('01','B8','VIT-K-COA:K-VIT-C03','Vitamin K',
          [type(vitamin), domain(biochemical), function(coagulation),
           deficiency_symptoms([easy_bruising, fatigue])])
]).

fixture(inhibit_chain, [
    tosid('10','F2','SUB-A:TST-001','Substance A', [type(compound), domain(chemistry)]),
    tosid('10','F2','SUB-B:TST-002','Substance B', [type(compound), domain(chemistry)]),
    tosid('10','F2','SUB-C:TST-003','Substance C', [type(compound), domain(chemistry)]),
    bridge('Substance A', inhibits, 'Substance B', [source(fixture)]),
    bridge('Substance B', inhibits, 'Substance C', [source(fixture)])
]).

fixture(declare_promotes, [ compose_rule(inhibits, inhibits, promotes) ]).
