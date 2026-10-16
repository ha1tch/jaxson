% solve.pl - proof-producing meta-interpreter over rel/4, composition rules, relation
% axioms and schemas.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Proof terms:
%   fact(rel(S,P,O), Evidence)
%   composed(P1-P2, rel(S,P,O), [Proof1, Proof2])
%   axiom(Axiom, rel(S,P,O), [SubProofs])
%   derived(SchemaName, Status, rel(S,P,O), [SubProofs])
%
% Only active vocabulary is used: a schema, axiom or composition rule with a validation
% error (validate.pl) is disabled, never half-applied.

:- encoding(utf8).

% ---- active vocabulary -------------------------------------------------------------------

active_schema(Name, Head, Body, Status) :-
    schema(Name, Head, Body, Status),
    \+ vocab_issue(error, _, schema(Name)).

active_axiom(Ax) :-
    rel_axiom(Ax),
    \+ vocab_issue(error, _, axiom(Ax)).

active_compose(P1, P2, P3) :-
    compose_rule(P1, P2, P3),
    \+ vocab_issue(error, _, compose_rule(P1, P2, P3)).

% ---- closed guard vocabulary -----------------------------------------------------------

product(Id) :-
    tosid(_, _, Id, _, Props),
    memberchk(domain(pharma), Props),
    memberchk(type(pharmaceutical), Props).

entity_domain(Id, D) :- tosid(_, _, Id, _, Props), memberchk(domain(D), Props).

run_guard(distinct(A, B))      :- A \== B.
run_guard(before(A, B))        :- A @< B.
run_guard(is_product(P))       :- product(P).
run_guard(entity_domain(E, D)) :- entity_domain(E, D).

% ---- solver ----------------------------------------------------------------------------

solve(Goal, Proof) :- solve(Goal, 6, Proof).

% a depth-0 call never succeeds here: it records whether a rule could have continued
solve(Goal, 0, _) :- bound_probe(Goal), fail.
solve(rel(S, P, O), _, fact(rel(S, P, O), M)) :-
    rel(S, P, O, M).
solve(rel(S, P, O), D, composed(P1-P2, rel(S, P, O), [A, B])) :-
    D > 0, D1 is D - 1,
    active_compose(P1, P2, P),
    solve(rel(S, P1, M), D1, A),
    solve(rel(M, P2, O), D1, B).
solve(rel(S, P, O), D, axiom(Ax, rel(S, P, O), Subs)) :-
    D > 0, D1 is D - 1,
    active_axiom(Ax),
    axiom_step(Ax, rel(S, P, O), D1, Subs).
solve(Head, D, derived(Name, Status, Head, Subs)) :-
    D > 0, D1 is D - 1,
    active_schema(Name, Head, Body, Status),
    solve_body(Body, D1, Subs).

% symmetric uses only the stored converse fact, so it cannot loop
axiom_step(symmetric(P), rel(S, P, O), _, [fact(rel(O, P, S), M)]) :-
    S \== O, rel(O, P, S, M).
axiom_step(subrel(P, Q), rel(S, Q, O), D, [Sub]) :-
    solve(rel(S, P, O), D, Sub).
axiom_step(inverse(P, Q), rel(S, P, O), D, [Sub]) :-
    solve(rel(O, Q, S), D, Sub).
% S \== O: a cycle in the data never manufactures a reflexive conclusion (ingest reports it)
axiom_step(transitive(P), rel(S, P, O), D, [Edge, Sub]) :-
    S \== O,
    edge(S, P, M, Edge), M \== S,
    solve(rel(M, P, O), D, Sub).

% One-step edges of a transitive relation: a stored fact, the stored converse of a symmetric
% relation, or a stored fact of a declared subrelation. Edges never recurse into solve/3, so
% closure sees every one-step fact the declarations make true without left recursion.
edge(S, P, M, Edge) :- stored_edge(S, P, M, Edge).
edge(S, P, M, axiom(subrel(Q, P), rel(S, P, M), [Sub])) :-
    active_axiom(subrel(Q, P)), stored_edge(S, Q, M, Sub).

stored_edge(S, P, M, fact(rel(S, P, M), M0)) :- rel(S, P, M, M0).
stored_edge(S, P, M, axiom(symmetric(P), rel(S, P, M), [fact(rel(M, P, S), M0)])) :-
    S \== M, active_axiom(symmetric(P)), rel(M, P, S, M0).

solve_body([], _, []).
solve_body([guard(G)|T], D, Subs) :- !, run_guard(G), solve_body(T, D, Subs).
solve_body([G|T], D, [P|Ps])      :- solve(G, D, P), solve_body(T, D, Ps).

% ---- status ----------------------------------------------------------------------------

pair_rank(S, R-S) :- status_rank(S, R).

weakest(L, W) :- maplist(pair_rank, L, Ps), keysort(Ps, [_-W|_]).
best_of(L, B) :- maplist(pair_rank, L, Ps), keysort(Ps, Sorted), last(Sorted, _-B).

% a fact is as strong as the strongest source asserting it, and is lifted only by a declared
% corroboration_lift/3 rule (evidence.pl); none is shipped
fact_status(M, S) :-
    findall(St, member(ev(_, St, _), M), L),
    (   L == [] -> S0 = conditional ; best_of(L, S0) ),
    lifted_status(S0, M, S).

proof_status(fact(_, M), S) :- fact_status(M, S).
proof_status(composed(_, _, Ps), S) :- maplist(proof_status, Ps, Ss), weakest(Ss, S).
proof_status(axiom(_, _, Ps), S)    :- maplist(proof_status, Ps, Ss), weakest(Ss, S).
proof_status(derived(_, St, _, Ps), S) :- maplist(proof_status, Ps, Ss), weakest([St|Ss], S).

% assess(+Goal, -Status, -Defeaters): best proof status, then weigh declared defeaters.
% A defeater is a supported-or-conditional fact of a relation D with defeats(D, T) for the
% goal's relation T on the same subject and object. If it is stronger than the best proof
% the goal is refuted; otherwise the goal is conflicted. A goal with no proof is unknown.
assess(Goal, Status, Defeaters) :-
    findall(S, (solve(Goal, P), proof_status(P, S)), L),
    (   L == [] -> Base = unknown ; best_of(L, Base) ),
    defeaters(Goal, Defeaters),
    settle(Base, Defeaters, Status).

defeaters(rel(S, T, O), Ds) :-
    findall(D-Dr,
            ( active_axiom(defeats(D, T)),
              solve(rel(S, D, O), Pr), proof_status(Pr, Dr),
              status_rank(Dr, R), R >= 3 ),
            Ds0),
    sort(Ds0, Ds).

settle(unknown, _, unknown) :- !.
settle(Base, [], Base) :- !.
settle(Base, Ds, Status) :-
    findall(Dr, member(_-Dr, Ds), Drs), best_of(Drs, Best),
    status_rank(Best, DR), status_rank(Base, BR),
    (   DR > BR -> Status = refuted ; Status = conflicted ).

best_status(Goal, Best) :- assess(Goal, Best, _).

% ---- proof inspection --------------------------------------------------------------------

proof_conclusion(fact(T, _), T).
proof_conclusion(composed(_, T, _), T).
proof_conclusion(axiom(_, T, _), T).
proof_conclusion(derived(_, _, T, _), T).

proof_leaves(fact(T, M), [T-M]).
proof_leaves(composed(_, _, Ps), L) :- maplist(proof_leaves, Ps, Ls), append(Ls, L).
proof_leaves(axiom(_, _, Ps), L)    :- maplist(proof_leaves, Ps, Ls), append(Ls, L).
proof_leaves(derived(_, _, _, Ps), L) :- maplist(proof_leaves, Ps, Ls), append(Ls, L).

proof_triples(Pr, Ts) :- proof_leaves(Pr, L), findall(T, member(T-_, L), Ts).

headline_conclusions(Cs) :-
    findall(C, ( rel_pred(P, O), memberchk(headline(true), O),
                 solve(rel(_, P, _), Pr), proof_conclusion(Pr, C) ), L0),
    sort(L0, Cs).

% retract_impact_set(+Triples, -Lost): headline conclusions for which EVERY discoverable
% proof uses at least one of Triples. A conclusion with an alternative proof avoiding all
% of them survives. retract_impact/2 is the single-fact case.
retract_impact_set(Triples, Lost) :-
    proof_table(Table),
    findall(C, ( member(C-Alts, Table),
                 forall(member(Ts, Alts), ( member(T, Triples), memberchk(T, Ts) )) ),
            Lost).

% proof_table(-Table): Conclusion-[LeafTriples, ...] for every headline conclusion, built by
% ONE enumeration of the headline relations and grouped by conclusion, instead of solving each
% conclusion again. Same proofs, same order of conclusions as solving them one by one.
proof_table(Table) :-
    findall(C-Ts, ( rel_pred(P, O), memberchk(headline(true), O),
                    solve(rel(_, P, _), Pr), proof_conclusion(Pr, C), proof_triples(Pr, Ts) ),
            Pairs),
    keysort(Pairs, Sorted),
    group_pairs_by_key(Sorted, Table).

retract_impact(Triple, Lost) :- retract_impact_set([Triple], Lost).

% ---- reporting ---------------------------------------------------------------------------

entity_name(Id, Name) :- tosid(_, _, Id, Name, _), !.

label(concept(C), C) :- !.
label(Id, Name) :- atom(Id), entity_name(Id, Name), !.
label(X, X).

% Products with a refuted may_address conclusion are left out.
recommend(Symptoms0, Recs) :-
    sort(Symptoms0, Symptoms),
    findall(P-Sym,
            ( member(Sym, Symptoms), solve(rel(P, may_address, concept(Sym)), _) ),
            Pairs0),
    sort(Pairs0, Pairs),
    findall(P-Sym-St,
            ( member(P-Sym, Pairs),
              best_status(rel(P, may_address, concept(Sym)), St), St \== refuted ),
            Rows),
    findall(P, member(P-_-_, Rows), Ps0), sort(Ps0, Ps),
    findall(rec(P, Name, Covers),
            ( member(P, Ps), entity_name(P, Name),
              findall(Sym-St, member(P-Sym-St, Rows), Covers) ),
            Recs0),
    sort(2, @=<, Recs0, Recs).

check_regimen(Ps, Findings) :-
    findall(finding(K, A, B),
            ( member(K, [duplicates, interferes_with, boosts]),
              member(A, Ps), member(B, Ps), A \== B,
              once(solve(rel(A, K, B), _)) ),
            F0),
    sort(F0, Findings).

explain(P) :- explain(P, 0).

explain(fact(rel(S, P, O), M), I) :-
    label(S, LS), label(O, LO),
    findall(Src-St, member(ev(Src, St, _), M), Evs),
    format("~*c~w ~w ~w   [fact ~w]~n", [I, 0' , LS, P, LO, Evs]).
explain(composed(R, rel(S, P, O), Ps), I) :-
    label(S, LS), label(O, LO),
    format("~*c~w ~w ~w   [composed ~w]~n", [I, 0' , LS, P, LO, R]),
    I1 is I + 2, forall(member(X, Ps), explain(X, I1)).
explain(axiom(Ax, rel(S, P, O), Ps), I) :-
    label(S, LS), label(O, LO),
    format("~*c~w ~w ~w   [axiom ~w]~n", [I, 0' , LS, P, LO, Ax]),
    I1 is I + 2, forall(member(X, Ps), explain(X, I1)).
explain(derived(N, St, rel(S, P, O), Ps), I) :-
    label(S, LS), label(O, LO),
    format("~*c~w ~w ~w   [~w, ceiling ~w]~n", [I, 0' , LS, P, LO, N, St]),
    I1 is I + 2, forall(member(X, Ps), explain(X, I1)).

% ---- bound certificate -----------------------------------------------------------------
% solve_certified(+Goal, [+Depth,] -Proofs, -Cert)
%   Cert = exhaustive     no search branch was cut by the depth bound where a rule could
%                         have continued: a deeper bound finds the same proofs
%   Cert = bound_reached  some branch was cut where a rule could have continued: proofs
%                         beyond the bound may exist
% The probe is sound and conservative: it reports bound_reached whenever a depth-consuming
% clause could make progress, never exhaustive when one could. Not re-entrant.
:- nb_setval(solve_bound_hit, false).

solve_certified(Goal, Proofs, Cert) :- solve_certified(Goal, 6, Proofs, Cert).
solve_certified(Goal, D, Proofs, Cert) :-
    nb_setval(solve_bound_hit, false),
    findall(P, solve(Goal, D, P), Proofs),
    nb_getval(solve_bound_hit, Hit),
    (   Hit == true -> Cert = bound_reached ; Cert = exhaustive ).

bound_probe(Goal) :-
    (   \+ \+ would_progress(Goal) -> nb_setval(solve_bound_hit, true) ; true ).

would_progress(rel(S, P, O)) :-
    (   active_compose(P1, _, P), operand_possible(S, P1)
    ;   active_axiom(Ax), axiom_could_progress(Ax, S, P, O)
    ;   active_schema(_, rel(S, P, O), Body, _), first_rel_possible(Body)
    ), !.

operand_possible(S, P1) :- ( rel(S, P1, _, _) -> true ; has_rules(P1) ).

has_rules(P) :-
    (   active_compose(_, _, P)
    ;   active_axiom(Ax), axiom_head(Ax, P)
    ;   active_schema(_, rel(_, P, _), _, _)
    ), !.

axiom_head(symmetric(P), P).
axiom_head(transitive(P), P).
axiom_head(inverse(P, _), P).
axiom_head(subrel(_, P), P).

axiom_could_progress(symmetric(P), S, P, O) :- S \== O, rel(O, P, S, _).
axiom_could_progress(transitive(P), S, P, O) :- S \== O, edge(S, P, M, _), M \== S.
axiom_could_progress(subrel(Q, P), S, P, O) :- ( rel(S, Q, O, _) -> true ; has_rules(Q) ).
axiom_could_progress(inverse(P, Q), S, P, O) :- ( rel(O, Q, S, _) -> true ; has_rules(Q) ).

% a clause whose first relation goal can neither be a stored fact nor be derived cannot succeed
first_rel_possible(Body) :-
    (   member(G, Body), G = rel(S1, P1, O1)
    ->  ( rel(S1, P1, O1, _) -> true ; has_rules(P1) )
    ;   true
    ).

% ---- structured assessment ---------------------------------------------------------------
% assessment(+Goal, -Record): the outcome as separate parts, so a consumer need not read them
% off one ordered status. assess/3 is unchanged; the record extends it.
%   status                the assess/3 status (projection of the parts below)
%   proof_strength        best proof status (supported, conditional) or none if no proof
%   independent_support   best independent-origin count over the proofs (0 if none)
%   conflict              none | contested (a defeater does not outweigh) | defeated
%   defeaters             Defeater-Status pairs, as assess/3
%   completeness          exhaustive | bound_reached (see solve_certified/3)
assessment(Goal, [status(St), proof_strength(PS), independent_support(N), conflict(Cf),
                  defeaters(Ds), completeness(Cert)]) :-
    assess(Goal, St, Ds),
    solve_certified(Goal, Proofs, Cert),
    findall(S, ( member(Pr, Proofs), proof_status(Pr, S) ), Ss),
    (   Ss == [] -> PS = none ; best_of(Ss, PS) ),
    (   independent_support(Goal, N0) -> N = N0 ; N = 0 ),
    conflict_kind(PS, St, Ds, Cf).

% a defeater cannot contest a conclusion that has no proof: absence is not refutation
conflict_kind(none, _, _, none) :- !.
conflict_kind(_, _, [], none) :- !.
conflict_kind(_, refuted, _, defeated) :- !.
conflict_kind(_, _, _, contested).

defeater_proofs(rel(S, T, O), L) :-
    findall(defeater(D, Dr, Pr),
            ( active_axiom(defeats(D, T)), once(solve(rel(S, D, O), Pr)),
              proof_status(Pr, Dr), status_rank(Dr, R), R >= 3 ),
            L).

% ---- recommendation tiers ----------------------------------------------------------------
% recommend_tiers(+Symptoms, -tiers(Supported, NeedsReview, Withheld))
% Each tier is a list of rec(Product, Name, [row(Symptom, Assessment, Why)]), sorted by name.
%   supported     proof strength supported and no defeater
%   needs_review  conditional proof strength, or a defeater that does not outweigh the support
%   withheld      a declared defeater outweighs the support; Why lists the defeating proofs
% Membership of any tier is not a diagnosis or an endorsement: may_address is a candidate
% relation. recommend/2 (supported and needs_review together) is unchanged.
recommend_tiers(Symptoms0, tiers(Sup, Rev, Wh)) :-
    sort(Symptoms0, Symptoms),
    findall(P-Sym, ( member(Sym, Symptoms), solve(rel(P, may_address, concept(Sym)), _) ), Pairs0),
    sort(Pairs0, Pairs),
    findall(T-P-Sym-A-Why,
            ( member(P-Sym, Pairs),
              Goal = rel(P, may_address, concept(Sym)),
              assessment(Goal, A), tier_of(A, T),
              (   T == supported -> Why = [] ; defeater_proofs(Goal, Why) ) ),
            Rows),
    tier_list(supported, Rows, Sup),
    tier_list(needs_review, Rows, Rev),
    tier_list(withheld, Rows, Wh).

tier_of(A, T) :-
    memberchk(proof_strength(PS), A), memberchk(conflict(C), A),
    (   C == defeated -> T = withheld
    ;   C == contested -> T = needs_review
    ;   PS == supported -> T = supported
    ;   T = needs_review
    ).

tier_list(T, Rows, List) :-
    findall(P, member(T-P-_-_-_, Rows), Ps0), sort(Ps0, Ps),
    findall(rec(P, Name, Covers),
            ( member(P, Ps), entity_name(P, Name),
              findall(row(Sym, A, Why), member(T-P-Sym-A-Why, Rows), Covers) ),
            L0),
    sort(2, @=<, L0, List).
