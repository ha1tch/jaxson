% demo.pl - small walkthrough. Run: swipl demo.pl
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
:- encoding(utf8).
:- initialization(main, main).
:- consult('load.pl').

eid(Name, Id) :- tosid(_, _, Id, Name, _), !.

main :-
    ingest_all, ingest_report, nl,
    format("Recommendations for [fatigue, memory_problems, pale_skin]:~n"),
    recommend([fatigue, memory_problems, pale_skin], Recs),
    forall(member(rec(_, Name, Covers), Recs),
           format("  ~w  ~w~n", [Name, Covers])),
    nl,
    format("Why does Iron Supplement 65mg address fatigue?~n"),
    eid('Iron Supplement 65mg', P),
    solve(rel(P, may_address, concept(fatigue)), Pr), !,
    explain(Pr), nl,
    format("Regimen check: Iron 65mg + Iron/Vit C combination + Calcium-free Vitamin C~n"),
    maplist(eid, ['Iron Supplement 65mg', 'Iron + Vitamin C Combination', 'Vitamin C 500mg Tablet'], Ps),
    check_regimen(Ps, Fs),
    forall(member(finding(K, A, B), Fs),
           ( label(A, LA), label(B, LB), format("  ~w ~w ~w~n", [LA, K, LB]) )),
    halt(0).
