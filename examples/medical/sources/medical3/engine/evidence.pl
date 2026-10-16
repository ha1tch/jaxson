% evidence.pl - independence of evidence: how many independent origins stand behind a fact
% or a proof. Independence is reported; a status changes only through a declared
% corroboration_lift/3 rule, and none is shipped.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% An evidence entry ev(Source, Strength, Detail) is mapped to an ORIGIN: the lineage group
% the source is declared to belong to (source_lineage/2, knowledge), or the source label
% itself if it has no declaration. Independence is the number of distinct origins, so two
% sources that share an origin count once, and one source asserting twice counts once.

:- encoding(utf8).
:- multifile source_lineage/2.
:- dynamic source_lineage/2.

% A lineage(Group) entry in an assertion's own detail takes precedence over the lineage
% declared for its source. Contradictory entries on one assertion fall back to the source
% label (ingest.pl reports them as conflicting_assertion_lineage).
ev_origin(ev(Src, _, Detail), G) :-
    findall(G0, member(lineage(G0), Detail), Gs0), sort(Gs0, Gs),
    (   Gs = [G1] -> G = G1
    ;   Gs = []   -> lineage_group(Src, G)
    ;   G = Src
    ).

% a conflicting declaration (validate.pl: conflicting_lineage) is disabled whole
lineage_group(Src, G) :-
    (   source_lineage(Src, G0), \+ vocab_issue(error, _, lineage(Src)) -> G = G0 ; G = Src ).

fact_origins(M, Gs) :-
    findall(G, ( member(E, M), ev_origin(E, G) ), L),
    sort(L, Gs).

fact_independence(M, N) :- fact_origins(M, Gs), length(Gs, N).

% a proof is corroborated only as far as its least-corroborated leaf
proof_independence(Pr, N) :-
    proof_leaves(Pr, Ls),
    findall(K, ( member(_-M, Ls), fact_independence(M, K) ), Ks),
    (   Ks == [] -> N = 0 ; min_list(Ks, N) ).

% best support over the discoverable proofs of a goal; fails if there is none
independent_support(Goal, N) :-
    findall(K, ( solve(Goal, Pr), proof_independence(Pr, K) ), Ks),
    Ks \== [], max_list(Ks, N).

explain_support(Goal) :-
    forall(solve(Goal, Pr),
           ( proof_independence(Pr, N), proof_conclusion(Pr, rel(S, P, O)),
             label(S, LS), label(O, LO),
             format("~w ~w ~w: ~d independent origin(s)~n", [LS, P, LO, N]),
             proof_leaves(Pr, Ls),
             forall(member(rel(A, Pd, B)-M, Ls),
                    ( fact_origins(M, Gs), label(A, LA), label(B, LB),
                      format("  ~w ~w ~w   origins ~w~n", [LA, Pd, LB, Gs]) )) )).

% ---- declared corroboration lift ------------------------------------------------------------
% corroboration_lift(MinOrigins, From, To): a fact whose base status is From, with at least
% MinOrigins independent origins among its entries at least as strong as From, is read as To.
% Knowledge, validated by validate.pl (a rejected rule is disabled whole). Applied once to a
% fact, never chained; the highest MinOrigins that fires wins. Proofs inherit the lift through
% weakest-step propagation. No rule is shipped.
:- multifile corroboration_lift/3.
:- dynamic corroboration_lift/3.

active_lift(Min, From, To) :-
    corroboration_lift(Min, From, To),
    \+ vocab_issue(error, _, lift(Min, From, To)).

lifted_status(Base, M, S) :-
    findall(Min-To, active_lift(Min, Base, To), Rules),
    (   Rules == [] -> S = Base
    ;   status_rank(Base, BR),
        findall(G, ( member(E, M), E = ev(_, St, _), status_rank(St, R), R >= BR,
                     ev_origin(E, G) ), L0),
        sort(L0, L1), length(L1, N),
        findall(Min-To, ( member(Min-To, Rules), N >= Min ), Fire),
        (   Fire == [] -> S = Base ; keysort(Fire, Sorted), last(Sorted, _-S) )
    ).
