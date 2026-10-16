% bench.pl - inference-validity benchmark. Run: swipl bench/bench.pl  (from package root)
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Known-positive, known-negative and adversarial cases, plus Experiment 1.
% It tests inference validity, not medical truth.

:- encoding(utf8).
:- initialization(main, main).
:- consult('../load.pl').
:- consult('fixtures.pl').
:- multifile test/5, check/1.
:- discontiguous test/5, check/1.

test(p1, positive, [], 'Iron Supplement 65mg may address fatigue, conditional only',
     addresses('Iron Supplement 65mg', fatigue, conditional)).
test(p2, positive, [], 'B12 sublingual may address memory_problems via deficiency',
     addresses('Vitamin B12 1000mcg Sublingual', memory_problems, conditional)).
test(p3, positive, [], 'B12 sublingual is labelled for fatigue, so supported',
     addresses('Vitamin B12 1000mcg Sublingual', fatigue, supported)).
test(p4, positive, [], 'Vitamin D3 2000 IU may address bone_pain',
     addresses('Vitamin D3 2000 IU', bone_pain, conditional)).
test(p5, positive, [], 'Magnesium 400mg may address irregular_heartbeat via Mg deficiency',
     addresses('Magnesium Supplement 400mg', irregular_heartbeat, conditional)).
test(p6, positive, [], 'Paracetamol is labelled for fever',
     addresses('Paracetamol 500mg Tablet', fever, supported)).
test(p7, positive, [], 'Paracetamol addresses headache through declared headache is_a pain',
     addresses('Paracetamol 500mg Tablet', headache, supported)).

test(n1, negative, [], 'Iron Supplement does not address headache',
     not_addresses('Iron Supplement 65mg', headache)).
test(n2, negative, [], 'Ibuprofen does not address fatigue',
     not_addresses('Ibuprofen 200mg Capsule', fatigue)).
test(n3, negative, [], 'No active ingredient label is left unresolved',
     no_unresolved_ingredients).
test(n4, negative, [inhibit_chain], 'inhibits o inhibits does not compose unless declared',
     cannot_derive(rel(n('Substance A'), promotes, n('Substance C')))).
test(n5, positive, [inhibit_chain, declare_promotes], 'inhibits o inhibits composes once declared',
     derives(rel(n('Substance A'), promotes, n('Substance C')))).

test(a1, adversarial, [], 'Iron 65mg and Iron+Vitamin C combination flagged as duplicate therapy',
     flagged('Iron Supplement 65mg', duplicates, 'Iron + Vitamin C Combination')).
test(a2, adversarial, [], 'Vitamin C 500mg flagged as boosting Iron 65mg',
     flagged('Vitamin C 500mg Tablet', boosts, 'Iron Supplement 65mg')).
test(a3, adversarial, [], 'Fatigue yields at least 5 candidate causes, none labelled above conditional',
     candidates_all_conditional(fatigue, 5)).
test(a4, adversarial, [], 'Unresolvable label (cobalt) is reported at ingest, not dropped silently',
     issue(unresolved(cobalt))).
test(a5, adversarial, [], 'Every headline conclusion rests only on facts with provenance',
     proofs_have_provenance).
test(a6, adversarial, [], 'Recommendations list each product once however many symptoms it covers',
     recommend_unique([fatigue, weakness, memory_problems, anemia, shortness_of_breath, pale_skin, depression])).
test(a7, adversarial, [], 'Retracting Ferrous Sulfate provides Iron removes the iron product conclusion',
     impact(rel(n('Ferrous Sulfate'), provides, n('Iron')),
            rel(n('Iron Supplement 65mg'), may_address, c(fatigue)))).

test(e1a, exp1_intended, [iron_fumarate], 'Iron Fumarate 100mg addresses fatigue via iron replacement',
     addresses('Iron Fumarate 100mg', fatigue, conditional)).
test(e1b, exp1_intended, [zinc], 'Zinc 50mg addresses hair_loss via zinc deficiency',
     addresses('Zinc 50mg', hair_loss, conditional)).
test(e1c, exp1_intended, [folate], 'Folic Acid 1mg addresses glossitis via folate deficiency',
     addresses('Folic Acid 1mg', glossitis, conditional)).
test(e1c_wrong, exp1_wrong, [folate], 'B12 product is NOT derived from Folate (shared label dna_synthesis)',
     not_addresses('Vitamin B12 1000mcg Sublingual', glossitis)).
test(e1d_spurious, exp1_spurious, [vitamin_k], 'Vitamin K is a conditional candidate for fatigue and has no product',
     vitamin_k_contained).

eid(Name, Id) :- tosid(_, _, Id, Name, _), !.

conv(n(N), Id) :- !, eid(N, Id).
conv(c(C), concept(C)) :- !.
conv(rel(A, P, B), rel(A1, P, B1)) :- !, conv(A, A1), conv(B, B1).

check(addresses(PN, Sym, Want)) :-
    eid(PN, P), best_status(rel(P, may_address, concept(Sym)), Got), Got == Want.
check(not_addresses(PN, Sym)) :-
    eid(PN, P), best_status(rel(P, may_address, concept(Sym)), unknown).
check(flagged(AN, K, BN)) :-
    eid(AN, A), eid(BN, B),
    once(( solve(rel(A, K, B), _) ; K == duplicates, solve(rel(B, K, A), _) )).
check(candidates_all_conditional(Sym, Min)) :-
    findall(S, (solve(rel(concept(Sym), candidate_cause, _), Pr), proof_status(Pr, S)), L),
    length(L, N), N >= Min,
    forall(member(S, L), S == conditional).
check(issue(Kind)) :- ingest_issue(Kind, _), !.
check(no_unresolved_ingredients) :-
    \+ ( ingest_issue(unresolved(_), ctx(_, Prop)),
         memberchk(Prop, [active_ingredient, active_ingredients]) ).
check(proofs_have_provenance) :-
    headline_conclusions(Cs), Cs \== [],
    forall(( member(C, Cs), solve(C, Pr), proof_leaves(Pr, L), member(_-M, L) ), M \== []).
check(recommend_unique(Syms)) :-
    recommend(Syms, Recs),
    findall(P, member(rec(P, _, _), Recs), Ps),
    length(Ps, N), N > 0, sort(Ps, S), length(S, N).
check(impact(R0, L0)) :-
    conv(R0, R), conv(L0, L), retract_impact(R, Lost), memberchk(L, Lost).
check(derives(T)) :- conv(T, C), once(solve(C, _)).
check(cannot_derive(T)) :- conv(T, C), \+ solve(C, _).
check(vitamin_k_contained) :-
    eid('Vitamin K', K),
    best_status(rel(concept(fatigue), candidate_cause, K), conditional),
    \+ ( solve(rel(P, provides, K), _), product(P) ),
    \+ ( solve(rel(P2, may_address, _), Pr), product(P2),
         proof_leaves(Pr, Ls), member(rel(K, _, _)-_, Ls), P2 \== K ).

apply_fixtures(Names) :-
    forall(member(N, Names), ( fixture(N, Ts), forall(member(T, Ts), assertz(T)) )),
    ingest_all.
undo_fixtures(Names) :-
    forall(member(N, Names), ( fixture(N, Ts), forall(member(T, Ts), ignore(retract(T))) )),
    ingest_all.

run_test(test(Id, Kind, Fx, Desc, Chk), r(Id, Kind, Res)) :-
    setup_call_cleanup(
        apply_fixtures(Fx),
        (   catch(check(Chk), E, (print_message(error, E), fail)) -> Res = pass ; Res = fail ),
        undo_fixtures(Fx)),
    (   Res == pass -> Tag = 'PASS' ; Tag = 'FAIL' ),
    format("~w  ~w  ~w  ~w~n", [Tag, Kind, Id, Desc]).

summarise(Results) :-
    findall(K, member(r(_, K, _), Results), Ks0), sort(Ks0, Ks),
    format("~nBy kind:~n"),
    forall(member(K, Ks),
           ( aggregate_all(count, member(r(_, K, _), Results), T),
             aggregate_all(count, member(r(_, K, pass), Results), P),
             format("  ~w~t~20|~d / ~d~n", [K, P, T]) )).

main :-
    findall(r(Id, K, Res),
            ( clause(test(Id, K, Fx, D, C), true), run_test(test(Id, K, Fx, D, C), r(Id, K, Res)) ),
            Results),
    summarise(Results),
    (   member(r(_, _, fail), Results) -> format("~nRESULT: FAILURES~n"), halt(1)
    ;   format("~nRESULT: ALL PASS~n"), halt(0) ).

:- consult('conformance.pl').
