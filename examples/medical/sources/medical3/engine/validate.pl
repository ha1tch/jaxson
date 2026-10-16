% validate.pl - static validation of the declared vocabulary (relations, axioms,
% composition rules, property maps, schemas). Runs at every ingest_all/0.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Findings go to vocab_issue(Severity, Kind, Context):
%   error    the offending schema is DISABLED by the solver (see active_schema/4)
%   warning  reported, rule still active
% Data-level problems (unresolved names, ambiguous labels) are a different class and
% live in ingest_issue/2 (ingest.pl).

:- encoding(utf8).

:- dynamic vocab_issue/3.
:- discontiguous vocab_check/3.

pk(P, K)  :- rel_pred(P, O), memberchk(kind(K), O).
pol(P, V) :- rel_pred(P, O), memberchk(polarity(V), O).

note_vocab(Sev, Kind, Ctx) :-
    (   vocab_issue(Sev, Kind, Ctx) -> true ; assertz(vocab_issue(Sev, Kind, Ctx)) ).

validate_vocab :-
    retractall(vocab_issue(_, _, _)),
    forall(vocab_check(Sev, Kind, Ctx), note_vocab(Sev, Kind, Ctx)).

vocab_ok :- \+ vocab_issue(error, _, _).

% ---- relation declarations ------------------------------------------------------------
vocab_check(error, malformed_rel_pred, P) :-
    rel_pred(P, Opts), \+ well_formed(Opts).
well_formed(Opts) :-
    memberchk(kind(K), Opts),      memberchk(K, [base, derived]),
    memberchk(polarity(V), Opts),  memberchk(V, [positive, negative, neutral]).

vocab_check(error, duplicate_rel_pred, P) :-
    findall(X, rel_pred(X, _), L), msort(L, S), append(_, [P, P|_], S).

% ---- composition rules ----------------------------------------------------------------
vocab_check(error, undeclared_predicate(P), compose_rule(P1, P2, P3)) :-
    compose_rule(P1, P2, P3), member(P, [P1, P2, P3]), \+ rel_pred(P, _).

% the result polarity must be the product of the operand polarities (neutral is a wildcard)
vocab_check(warning, polarity_mismatch, compose_rule(P1, P2, P3)) :-
    compose_rule(P1, P2, P3),
    pol(P1, A), pol(P2, B), pol(P3, C),
    pol_mul(A, B, E), E \== neutral, C \== neutral, E \== C.

pol_mul(neutral, _, neutral) :- !.
pol_mul(_, neutral, neutral) :- !.
pol_mul(X, X, positive) :- !.
pol_mul(_, _, negative).

% ---- axioms ---------------------------------------------------------------------------
known_axiom(symmetric(_)).
known_axiom(transitive(_)).
known_axiom(inverse(_, _)).
known_axiom(subrel(_, _)).
known_axiom(defeats(_, _)).

vocab_check(error, unknown_axiom, A) :-
    rel_axiom(A), \+ known_axiom(A).

vocab_check(error, undeclared_predicate(P), axiom(A)) :-
    rel_axiom(A), known_axiom(A), A =.. [_|Ps], member(P, Ps), \+ rel_pred(P, _).

vocab_check(error, axiom_needs_base_predicate(P), axiom(A)) :-
    rel_axiom(A), ( A = symmetric(P) ; A = transitive(P) ), pk(P, derived).

vocab_check(error, inverse_must_define_derived(P), axiom(inverse(P, Q))) :-
    rel_axiom(inverse(P, Q)), pk(P, base).

vocab_check(error, self_referential_axiom, axiom(A)) :-
    rel_axiom(A), ( A = inverse(P, P) ; A = subrel(P, P) ; A = defeats(P, P) ).

vocab_check(error, inverse_cycle, axiom(inverse(P, Q))) :-
    rel_axiom(inverse(P, Q)), rel_axiom(inverse(Q, P)).

vocab_check(error, subrel_cycle, axiom(subrel(P, Q))) :-
    rel_axiom(subrel(P, Q)), subrel_reach(Q, P, [Q]).

subrel_reach(A, B, _) :- rel_axiom(subrel(A, B)).
subrel_reach(A, B, V) :-
    rel_axiom(subrel(A, M)), \+ memberchk(M, V), subrel_reach(M, B, [M|V]).

vocab_check(warning, polarity_mismatch, axiom(subrel(P, Q))) :-
    rel_axiom(subrel(P, Q)), pol(P, A), pol(Q, B),
    A \== neutral, B \== neutral, A \== B.

% ---- property maps --------------------------------------------------------------------
vocab_check(error, undeclared_predicate(P), prop_map(F)) :-
    prop_map(F, P, _), \+ rel_pred(P, _).
vocab_check(error, prop_map_needs_base_predicate(P), prop_map(F)) :-
    prop_map(F, P, _), pk(P, derived).
vocab_check(error, bad_prop_mode(M), prop_map(F)) :-
    prop_map(F, _, M), \+ memberchk(M, [concept, entity, entity_inv]).

% ---- schemas --------------------------------------------------------------------------
vocab_check(error, duplicate_schema_name, schema(N)) :-
    findall(X, schema(X, _, _, _), L), msort(L, S), append(_, [N, N|_], S).

vocab_check(error, bad_status(St), schema(N)) :-
    schema(N, _, _, St), \+ status_rank(St, _).

vocab_check(error, malformed_head, schema(N)) :-
    schema(N, H, _, _), \+ ( nonvar(H), H = rel(_, P, _), atom(P) ).

vocab_check(error, variable_predicate, schema(N)) :-
    schema(N, H, Body, _), member(rel(_, P, _), [H|Body]), var(P).

vocab_check(error, undeclared_predicate(P), schema(N)) :-
    schema(N, H, Body, _), member(rel(_, P, _), [H|Body]), atom(P), \+ rel_pred(P, _).

vocab_check(error, unknown_body_goal(G), schema(N)) :-
    schema(N, _, Body, _), member(G, Body), \+ body_goal(G).
body_goal(rel(_, _, _)).
body_goal(guard(_)).

vocab_check(error, unknown_guard(G), schema(N)) :-
    schema(N, _, Body, _), member(guard(G), Body),
    \+ ( callable(G), functor(G, F, A), guard_spec(F, A) ).

vocab_check(error, guard_uses_unbound_variable(G), schema(N)) :-
    schema(N, _, Body, _), guard_unbound(Body, [], G).

guard_unbound([guard(G)|_], Bound, G) :-
    term_variables(G, Vs), member(V, Vs), \+ ( member(B, Bound), B == V ).
guard_unbound([guard(_)|T], Bound, G) :-
    guard_unbound(T, Bound, G).
guard_unbound([rel(S, _, O)|T], Bound, G) :-
    term_variables(S-O, Vs), append(Bound, Vs, B1), guard_unbound(T, B1, G).

% every head variable must be bound by some rel/3 goal of the body
is_rel_goal(G) :- nonvar(G), G = rel(_, _, _).
vocab_check(error, head_variable_not_bound_by_body, schema(N)) :-
    schema(N, H, Body, _),
    term_variables(H, HV),
    include(is_rel_goal, Body, Rs),      % include/3 does not copy, so variable identity holds
    term_variables(Rs, BV),
    member(V, HV), \+ ( member(B, BV), B == V ).

% ---- reachability and recursion -------------------------------------------------------
produced(P) :- pk(P, base).
produced(P) :- schema(_, rel(_, P, _), _, _).
produced(P) :- compose_rule(_, _, P).
produced(P) :- rel_axiom(subrel(_, P)).
produced(P) :- rel_axiom(inverse(P, _)).

vocab_check(warning, no_producer, pred(P)) :-
    rel_pred(P, _), pk(P, derived), \+ produced(P).

vocab_check(warning, unreachable_schema(P), schema(N)) :-
    schema(N, _, Body, _), member(rel(_, P, _), Body),
    atom(P), rel_pred(P, _), \+ produced(P).

vocab_check(warning, self_recursive_schema, schema(N)) :-
    schema(N, rel(_, P, _), Body, _), member(rel(_, P, _), Body).

% ---- evidence independence ------------------------------------------------------------
vocab_check(error, conflicting_lineage(S), lineage(S)) :-
    source_lineage(S, G1), source_lineage(S, G2), G1 \== G2.
vocab_check(warning, lineage_to_self(S), lineage(S)) :-
    source_lineage(S, S).

vocab_check(error, bad_lift_status(S), lift(Min, From, To)) :-
    corroboration_lift(Min, From, To), member(S, [From, To]), \+ status_rank(S, _).
vocab_check(error, lift_from_non_positive(From), lift(Min, From, To)) :-
    corroboration_lift(Min, From, To), status_rank(From, R), R < 3.
vocab_check(error, lift_not_upward, lift(Min, From, To)) :-
    corroboration_lift(Min, From, To),
    status_rank(From, RF), status_rank(To, RT), RT =< RF.
vocab_check(error, bad_lift_threshold(Min), lift(Min, From, To)) :-
    corroboration_lift(Min, From, To), \+ ( integer(Min), Min >= 2 ).

% ---- reporting ------------------------------------------------------------------------
vocab_report :-
    (   vocab_issue(_, _, _)
    ->  format("Vocabulary findings:~n"),
        forall(vocab_issue(Sev, K, C), format("  ~w  ~q  in  ~q~n", [Sev, K, C]))
    ;   format("Vocabulary clean.~n")
    ).
