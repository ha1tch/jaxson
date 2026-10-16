% legacy_baseline.pl - runs the Experiment 1 fixtures against the ORIGINAL coordinator
% (legacy/coordinator_lib.pl, the unmodified file minus its auto-run entry point).
% Run from the package root: swipl bench/legacy_baseline.pl
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Only tosid/5 terms are applied; bridge/alias/compose_rule are the new engine's concepts.

:- encoding(utf8).
:- initialization(main, main).
:- consult('../legacy/coordinator_lib.pl').

:- dynamic fixture/2, bridge/4, alias/2, compose_rule/3.
:- consult('fixtures.pl').

apply(Names) :-
    forall(member(N, Names),
           ( fixture(N, Ts), forall(( member(T, Ts), T = tosid(_,_,_,_,_) ), assertz(T)) )).
undo(Names) :-
    forall(member(N, Names),
           ( fixture(N, Ts), forall(( member(T, Ts), T = tosid(_,_,_,_,_) ), ignore(retract(T))) )).

% case(Id, Description, Fixtures, Expectation, Product, Symptom)
% Expectation: derived (intended) or absent (wrong/spurious must not appear).
case(e1a, 'Iron Fumarate 100mg for fatigue',     [iron_fumarate], derived, 'Iron Fumarate 100mg', fatigue).
case(e1b, 'Zinc 50mg for hair_loss',             [zinc],          derived, 'Zinc 50mg',           hair_loss).
case(e1c, 'Folic Acid 1mg for glossitis',        [folate],        derived, 'Folic Acid 1mg',      glossitis).
case(e1c_wrong, 'B12 product for glossitis (must be absent)', [folate], absent,
     'Vitamin B12 1000mcg Sublingual', glossitis).

found(Product, Symptom, Mechanism, Type) :-
    catch(find_treatment_for_symptom(Symptom, Product, Mechanism, Type, _), _, fail).

main :-
    forall(case(Id, Desc, Fx, Exp, Prod, Sym),
           setup_call_cleanup(
               apply(Fx),
               (   findall(M-T, found(Prod, Sym, M, T), Ms0), sort(Ms0, Ms),
                   (   Exp == derived, Ms \== [] -> V = 'derived'
                   ;   Exp == derived           -> V = 'MISSING'
                   ;   Exp == absent, Ms == []  -> V = 'absent (ok)'
                   ;   V = 'PRESENT (wrong)' ),
                   format("~w  ~w: ~w  ~q~n", [Id, Desc, V, Ms]) ),
               undo(Fx))),
    % Vitamin K leak: does a deficiency claim for a product-less vitamin appear?
    setup_call_cleanup(
        apply([vitamin_k]),
        (   (   symptom_suggests_deficiency(fatigue, 'Vitamin K', Conf)
            ->  format("e1d_spurious  fatigue suggests Vitamin K deficiency, confidence ~w~n", [Conf])
            ;   format("e1d_spurious  not produced~n") ) ),
        undo([vitamin_k])),
    halt(0).
