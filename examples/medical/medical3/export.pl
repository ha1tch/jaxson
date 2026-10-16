% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
% export.pl - dump medical3's ingested relations, active vocabulary, entities and the engine's own
% outputs (assess/3, assessment/2, recommend_tiers/2) as JSON, for comparison.
% BEGIN-LOAD
:- prolog_load_context(directory, D), atom_concat(D, '/../sources/medical3/load.pl', F), consult(F).
% END-LOAD

jq(A, Out) :- format(atom(Out), "\"~w\"", [A]).

term_json(concept(X), Out) :- !, format(atom(Out), "{\"k\":\"c\",\"v\":\"~w\"}", [X]).
term_json(A, Out) :- format(atom(Out), "{\"k\":\"e\",\"v\":\"~w\"}", [A]).

pat_json(T, Out) :- var(T), !, format(atom(Out), "{\"var\":\"_~w\"}", [T]).
pat_json('$VAR'(N), Out) :- !, format(atom(Out), "{\"var\":\"V~w\"}", [N]).
pat_json(concept(X), Out) :- !, pat_json(X, I), format(atom(Out), "{\"concept\":~w}", [I]).
pat_json(A, Out) :- format(atom(Out), "{\"atom\":\"~w\"}", [A]).

join_commas(L) :- atomic_list_concat(L, ',', A), write(A).

quoted_list(L, A) :- maplist(jq, L, Qs), atomic_list_concat(Qs, ',', A).

ev_json(ev(Src, St, Detail), J) :-
    findall(G, member(lineage(G), Detail), Gs0), sort(Gs0, Gs), quoted_list(Gs, GA),
    format(atom(J), "{\"src\":\"~w\",\"strength\":\"~w\",\"lineage\":[~w]}", [Src, St, GA]).

export_facts :-
    findall(J, ( rel(S, P, O, M),
                 term_json(S, SJ), term_json(O, OJ),
                 maplist(ev_json, M, EJs), atomic_list_concat(EJs, ',', EA),
                 format(atom(J), "{\"s\":~w,\"p\":\"~w\",\"o\":~w,\"ev\":[~w]}", [SJ, P, OJ, EA]) ), Js),
    join_commas(Js).

export_lineage :-
    findall(Src, ( rel(_, _, _, M), member(ev(Src, _, _), M) ), Ss0), sort(Ss0, Ss),
    findall(J, ( member(Src, Ss), lineage_group(Src, G), format(atom(J), "\"~w\":\"~w\"", [Src, G]) ), Js),
    join_commas(Js).

export_entities :-
    findall(J, ( tosid(_, _, Id, Name, Props),
                 ( memberchk(domain(D), Props) -> true ; D = none ),
                 ( memberchk(type(T), Props) -> true ; T = none ),
                 format(atom(J), "{\"id\":\"~w\",\"name\":\"~w\",\"domain\":\"~w\",\"type\":\"~w\"}", [Id, Name, D, T]) ), Js),
    join_commas(Js).

export_preds :-
    findall(J, ( rel_pred(P, O), memberchk(kind(K), O),
                 ( memberchk(headline(true), O) -> H = true ; H = false ),
                 format(atom(J), "{\"p\":\"~w\",\"kind\":\"~w\",\"headline\":~w}", [P, K, H]) ), Js),
    join_commas(Js).

export_axioms :-
    findall(J, ( active_axiom(A), A =.. [Kind|Args], quoted_list(Args, AA),
                 format(atom(J), "{\"kind\":\"~w\",\"args\":[~w]}", [Kind, AA]) ), Js),
    join_commas(Js).

export_compose :-
    findall(J, ( active_compose(A, B, C),
                 format(atom(J), "{\"p1\":\"~w\",\"p2\":\"~w\",\"p3\":\"~w\"}", [A, B, C]) ), Js),
    join_commas(Js).

goal_json(guard(G), Out) :- !,
    G =.. [N|Args], maplist(pat_json, Args, As), atomic_list_concat(As, ',', AA),
    format(atom(Out), "{\"guard\":\"~w\",\"args\":[~w]}", [N, AA]).
goal_json(rel(S, P, O), Out) :-
    pat_json(S, SJ), pat_json(O, OJ),
    format(atom(Out), "{\"s\":~w,\"p\":\"~w\",\"o\":~w}", [SJ, P, OJ]).

:- dynamic tmp_schema_json/1.
export_schemas :-
    retractall(tmp_schema_json(_)),
    forall(active_schema(N, H, B, St),
           \+ \+ ( copy_term(N-H-B, N2-H2-B2), numbervars(H2-B2, 0, _),
                   goal_json(H2, HJ), maplist(goal_json, B2, BJs), atomic_list_concat(BJs, ',', BA),
                   format(atom(J), "{\"name\":\"~w\",\"ceiling\":\"~w\",\"head\":~w,\"body\":[~w]}", [N2, St, HJ, BA]),
                   assertz(tmp_schema_json(J)) )),
    findall(J, tmp_schema_json(J), Js), join_commas(Js).

defs_json(Ds, J) :-
    findall(X, ( member(D-St, Ds), format(atom(X), "{\"d\":\"~w\",\"st\":\"~w\"}", [D, St]) ), Xs),
    atomic_list_concat(Xs, ',', J).

assess_fields(A, St, PS, N, Cf, DJ, Cm) :-
    memberchk(status(St), A), memberchk(proof_strength(PS), A), memberchk(independent_support(N), A),
    memberchk(conflict(Cf), A), memberchk(defeaters(Ds), A), memberchk(completeness(Cm), A),
    defs_json(Ds, DJ).

export_expected :-
    headline_conclusions(Cs),
    findall(J, ( member(rel(S, P, O), Cs), assess(rel(S, P, O), St, _),
                 term_json(S, SJ), term_json(O, OJ),
                 format(atom(J), "{\"s\":~w,\"p\":\"~w\",\"o\":~w,\"status\":\"~w\"}", [SJ, P, OJ, St]) ), Js),
    join_commas(Js).

export_assessments :-
    headline_conclusions(Cs),
    findall(J, ( member(rel(S, P, O), Cs),
                 assessment(rel(S, P, O), A), assess_fields(A, St, PS, N, Cf, DJ, Cm),
                 aggregate_all(count, solve(rel(S, P, O), _), N6),
                 aggregate_all(count, solve(rel(S, P, O), 9, _), N9),
                 term_json(S, SJ), term_json(O, OJ),
                 format(atom(J), "{\"s\":~w,\"p\":\"~w\",\"o\":~w,\"status\":\"~w\",\"ps\":\"~w\",\"ind\":~w,\"conflict\":\"~w\",\"defeaters\":[~w],\"completeness\":\"~w\",\"n6\":~w,\"n9\":~w}",
                        [SJ, P, OJ, St, PS, N, Cf, DJ, Cm, N6, N9]) ), Js),
    join_commas(Js).

symptom_sets([all-All, demo-[fatigue, memory_problems, pale_skin]]) :-
    findall(Sym, solve(rel(_, may_address, concept(Sym)), _), L0), sort(L0, All).

why_json(Why, J) :-
    findall(X, ( member(defeater(D, Dr, _), Why), format(atom(X), "{\"d\":\"~w\",\"st\":\"~w\"}", [D, Dr]) ), Xs),
    atomic_list_concat(Xs, ',', J).

export_tiers :-
    symptom_sets(Sets),
    findall(J, ( member(Name-Syms, Sets), recommend_tiers(Syms, tiers(Sup, Rev, Wh)),
                 member(Tier-List, [supported-Sup, needs_review-Rev, withheld-Wh]),
                 member(rec(P, PName, Covers), List), member(row(Sym, A, Why), Covers),
                 assess_fields(A, St, PS, N, Cf, DJ, Cm), why_json(Why, WJ),
                 format(atom(J), "{\"set\":\"~w\",\"tier\":\"~w\",\"p\":\"~w\",\"name\":\"~w\",\"sym\":\"~w\",\"status\":\"~w\",\"ps\":\"~w\",\"ind\":~w,\"conflict\":\"~w\",\"defeaters\":[~w],\"completeness\":\"~w\",\"why\":[~w]}",
                        [Name, Tier, P, PName, Sym, St, PS, N, Cf, DJ, Cm, WJ]) ), Js),
    join_commas(Js).

export_sets :-
    symptom_sets(Sets),
    findall(J, ( member(Name-Syms, Sets), quoted_list(Syms, SA), format(atom(J), "\"~w\":[~w]", [Name, SA]) ), Js),
    join_commas(Js).

main :-
    format("{\"facts\":["), export_facts,
    format("],\"lineage\":{"), export_lineage,
    format("},\"entities\":["), export_entities,
    format("],\"preds\":["), export_preds,
    format("],\"axioms\":["), export_axioms,
    format("],\"compose\":["), export_compose,
    format("],\"schemas\":["), export_schemas,
    format("],\"expected\":["), export_expected,
    format("],\"assessments\":["), export_assessments,
    format("],\"sets\":{"), export_sets,
    format("},\"tiers\":["), export_tiers,
    format("]}~n").

:- initialization((main, halt)).
