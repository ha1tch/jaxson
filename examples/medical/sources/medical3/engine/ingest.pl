% ingest.pl - convert tosid/5 knowledge into rel/4 facts keyed on tosid identifiers.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Names are used only here, at the authoring boundary, and every resolution failure is
% recorded in ingest_issue/2. Property ingestion is driven by prop_map/3 (vocab.pl and
% extension files), so a new property needs no change here.
%
% Stored facts are rel(Subject, Predicate, Object, Evidence). Evidence is a list with one
% ev(Source, Strength, Detail) entry per source that asserted the fact. A second source
% is merged into the list, never dropped.

:- encoding(utf8).

:- dynamic rel/4, ingest_issue/2, label_id/2.

norm(Name, Norm) :-
    atom_string(Name, S0), string_lower(S0, S1),
    split_string(S1, " -", "", Parts),
    atomic_list_concat(Parts, '_', Norm).

note(Kind, Ctx) :-
    (   ingest_issue(Kind, Ctx) -> true ; assertz(ingest_issue(Kind, Ctx)) ).

add_label(L, Id) :-
    (   label_id(L, Id) -> true ; assertz(label_id(L, Id)) ).

build_label_index :-
    forall(tosid(_, _, Id, Name, Props),
           (   norm(Name, N), add_label(N, Id),
               forall(member(chemical_name(CN), Props),
                      (   norm(CN, N2), add_label(N2, Id) )) )),
    forall(alias(Label, EntityName),
           (   norm(EntityName, EN),
               findall(I, (tosid(_, _, I, Nm, _), norm(Nm, EN)), Is),
               (   Is = [Id]
               ->  norm(Label, L), add_label(L, Id)
               ;   note(bad_alias(Label, EntityName), alias)
               ) )).

resolve(Label, Ctx, Id) :-
    norm(Label, N),
    findall(I, label_id(N, I), Is0), sort(Is0, Is),
    (   Is = [Id] -> true
    ;   Is = []   -> note(unresolved(N), Ctx), fail
    ;   note(ambiguous(N, Is), Ctx), fail
    ).

resolve_term(concept(C), _, concept(C)) :- !.
resolve_term(Label, Ctx, Id) :- resolve(Label, Ctx, Id).

% ---- evidence -------------------------------------------------------------------------

% make_ev(+Source, +Meta, -Ev): strength is the asserting term's status(S) if present,
% else the declared default for the source, else conditional.
make_ev(Src, Meta, ev(Src, Str, Detail)) :-
    (   memberchk(status(S0), Meta) -> Str = S0
    ;   source_strength(Src, S1)    -> Str = S1
    ;   Str = conditional
    ),
    exclude(is_ev_control, Meta, Detail).

is_ev_control(source(_)).
is_ev_control(status(_)).

merge_ev(Old, New, Merged) :-
    foldl(add_ev, New, Old, Merged).
add_ev(E, Acc, Out) :- ( memberchk(E, Acc) -> Out = Acc ; append(Acc, [E], Out) ).

emit(S, P, O, Ev) :-
    (   rel_pred(P, Opts), memberchk(kind(base), Opts)
    ->  (   retract(rel(S, P, O, Old))
        ->  merge_ev(Old, [Ev], New), assertz(rel(S, P, O, New))
        ;   assertz(rel(S, P, O, [Ev]))
        )
    ;   note(undeclared_predicate(P), emit(S))
    ).

% ---- property ingestion, driven by prop_map/3 ------------------------------------------

as_list(X, L) :- ( is_list(X) -> L = X ; L = [X] ).

ingest_prop(Id, Prop) :-
    Prop =.. [F, V],
    prop_map(F, P, Mode), !,
    as_list(V, Vs),
    forall(member(X, Vs), ingest_value(Mode, Id, F, P, X)).
ingest_prop(_, _).

prop_ev(Id, F, Ev) :-
    make_ev(kb, [from(Id, F)], Ev).

ingest_value(concept, Id, F, P, X) :-
    prop_ev(Id, F, Ev), emit(Id, P, concept(X), Ev).
ingest_value(entity, Id, F, P, X) :-
    (   resolve(X, ctx(Id, F), T) -> prop_ev(Id, F, Ev), emit(Id, P, T, Ev) ; true ).
ingest_value(entity_inv, Id, F, P, X) :-
    (   resolve(X, ctx(Id, F), T) -> prop_ev(Id, F, Ev), emit(T, P, Id, Ev) ; true ).

ingest_bridge(A, P, B, Meta) :-
    (   memberchk(source(Src), Meta) -> true ; Src = curated ),
    (   resolve_term(A, bridge(P), IA), resolve_term(B, bridge(P), IB)
    ->  make_ev(Src, Meta, Ev), emit(IA, P, IB, Ev),
        check_assertion_lineage(Meta, P, IA, IB)
    ;   true
    ).

check_assertion_lineage(Meta, P, IA, IB) :-
    findall(G, member(lineage(G), Meta), Gs0), sort(Gs0, Gs),
    (   Gs = [_, _|_] -> note(conflicting_assertion_lineage(P), bridge(IA, IB)) ; true ).

% A cycle in a relation declared transitive is a data problem: report it, never hide it.
check_cycles :-
    forall(( rel_axiom(transitive(P)), \+ vocab_issue(error, _, axiom(transitive(P))) ),
           check_cycles(P)).

% Tarjan's strongly connected components, one linear pass per transitive relation. A node
% is cyclic iff its component has several members or it has an edge to itself.
:- dynamic tj_index/2, tj_low/2, tj_on/1, tj_next/1, tj_stack/1.

check_cycles(P) :-
    tj_reset,
    (   setof(S, O^M^rel(S, P, O, M), Ss) -> true ; Ss = [] ),
    forall(member(S, Ss), ( tj_index(S, _) -> true ; tj_visit(P, S) )),
    tj_reset.

tj_reset :-
    retractall(tj_index(_, _)), retractall(tj_low(_, _)), retractall(tj_on(_)),
    retractall(tj_stack(_)), retractall(tj_next(_)),
    assertz(tj_next(0)), assertz(tj_stack([])).

tj_visit(P, V) :-
    retract(tj_next(I)), I1 is I + 1, assertz(tj_next(I1)),
    assertz(tj_index(V, I)), assertz(tj_low(V, I)),
    retract(tj_stack(St)), assertz(tj_stack([V|St])), assertz(tj_on(V)),
    forall(rel(V, P, W, _), tj_edge(P, V, W)),
    tj_low(V, L),
    (   L =:= I -> tj_component(P, V) ; true ).

tj_edge(P, V, W) :-
    (   \+ tj_index(W, _)
    ->  tj_visit(P, W), tj_low(V, LV), tj_low(W, LW), L is min(LV, LW), tj_set_low(V, L)
    ;   tj_on(W)
    ->  tj_low(V, LV), tj_index(W, IW), L is min(LV, IW), tj_set_low(V, L)
    ;   true
    ).

tj_set_low(V, L) :- retract(tj_low(V, _)), assertz(tj_low(V, L)).

tj_component(P, V) :-
    tj_pop_until(V, Members),
    (   ( Members = [_, _|_] ; Members = [X], rel(X, P, X, _) )
    ->  forall(member(M, Members), note(cyclic(P, M), transitive))
    ;   true
    ).

tj_pop_until(V, Members) :-
    once(retract(tj_stack([T|Rest]))), assertz(tj_stack(Rest)), retract(tj_on(T)),
    (   T == V -> Members = [T] ; Members = [T|More], tj_pop_until(V, More) ).

check_duplicate_ids :-
    findall(Id, tosid(_, _, Id, _, _), Ids),
    msort(Ids, Sorted),
    forall(( append(_, [X, X|_], Sorted) ), note(duplicate_id(X), tosid)).

ingest_all :-
    retractall(rel(_, _, _, _)),
    retractall(ingest_issue(_, _)),
    retractall(label_id(_, _)),
    validate_vocab,
    build_label_index,
    forall(tosid(_, _, Id, _, Props), forall(member(Pr, Props), ingest_prop(Id, Pr))),
    forall(bridge(A, P, B, M), ingest_bridge(A, P, B, M)),
    check_duplicate_ids,
    check_cycles.

ingest_report :-
    aggregate_all(count, rel(_, _, _, _), N),
    aggregate_all(count, tosid(_, _, _, _, _), E),
    format("Ingested ~d entities into ~d relations.~n", [E, N]),
    (   ingest_issue(_, _)
    ->  format("Ingest issues (not silent):~n"),
        forall(ingest_issue(K, C), format("  ~q  in  ~q~n", [K, C]))
    ;   format("No ingest issues.~n")
    ),
    vocab_report.
