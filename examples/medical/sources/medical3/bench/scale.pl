% scale.pl - cost of ingest, failing search, open search, composed search and impact analysis
% on synthetic graphs. Run: swipl bench/scale.pl  (from package root). Prints a markdown table.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Shapes: ring (one cycle), chain (one path), fan (one hub, N spokes), all over precursor_of,
% with one product containing the first entity. Cost is reported in inferences, a
% machine-independent measure, plus CPU milliseconds.

:- encoding(utf8).
:- initialization(main, main).
:- consult('../load.pl').

sizes([12, 50, 200, 1000]).

mk_ent(N, tosid('10', 'F2', Id, N, [type(compound), domain(chemistry)])) :-
    format(atom(Id), 'SCL-~w:X', [N]).
mk_prod(N, tosid('10', 'E4', Id, N, [type(pharmaceutical), domain(pharma)])) :-
    format(atom(Id), 'SCL-~w:X', [N]).
br(A, P, B, bridge(A, P, B, [source(fixture)])).
nm(Pre, I, N) :- format(atom(N), '~w~d', [Pre, I]).

gen(ring, N, Ts, First, Prod) :-
    findall(X, (between(1, N, I), nm(sr, I, X)), Ns), maplist(mk_ent, Ns, Es),
    findall(B, (between(1, N, I), J is I mod N + 1, nm(sr, I, A), nm(sr, J, C),
                br(A, precursor_of, C, B)), Bs),
    First = sr1, Prod = sprod, mk_prod(sprod, P), br(sprod, contains, sr1, Bc),
    append([Es, [P, Bc], Bs], Ts).
gen(chain, N, Ts, First, Prod) :-
    findall(X, (between(1, N, I), nm(sc, I, X)), Ns), maplist(mk_ent, Ns, Es),
    N1 is N - 1,
    findall(B, (between(1, N1, I), J is I + 1, nm(sc, I, A), nm(sc, J, C),
                br(A, precursor_of, C, B)), Bs),
    First = sc1, Prod = sprod, mk_prod(sprod, P), br(sprod, contains, sc1, Bc),
    append([Es, [P, Bc], Bs], Ts).
gen(fan, N, Ts, First, Prod) :-
    findall(X, (between(1, N, I), nm(ss, I, X)), Ns), maplist(mk_ent, [shub|Ns], Es),
    findall(B, (between(1, N, I), nm(ss, I, C), br(shub, precursor_of, C, B)), Bs),
    First = shub, Prod = sprod, mk_prod(sprod, P), br(sprod, contains, shub, Bc),
    append([Es, [P, Bc], Bs], Ts).

eidn(Name, Id) :- tosid(_, _, Id, Name, _), !.

timed(Goal, Val, Inf, Ms) :-
    statistics(inferences, I0), statistics(cputime, T0),
    catch(call_with_time_limit(60, Goal), time_limit_exceeded, Val = timeout),
    statistics(inferences, I1), statistics(cputime, T1),
    Inf is I1 - I0, Ms is round((T1 - T0) * 1000).

row(Shape, N, First, Prod) :-
    timed(ingest_all, _, _, IMs),
    eidn(First, F), eidn(Prod, P),
    timed(( \+ solve(rel(F, precursor_of, nowhere), _) -> true ; true ), _, FI, _),
    timed(aggregate_all(count, solve(rel(F, precursor_of, _), _), C1), C1, OI, _),
    timed(aggregate_all(count, solve(rel(P, supplies_precursor_of, _), _), C2), C2, PI, _),
    once(rel(F, precursor_of, S0, _)),
    timed(check_cycles, _, _, CMs),
    timed(( retract_impact(rel(F, precursor_of, S0), Lost), length(Lost, LN) ), LN, II, IMs2),
    format("| ~w | ~d | ~d (cycle check ~d) | ~d | ~w / ~d | ~w / ~d | ~w / ~d (~d ms) |~n",
           [Shape, N, IMs, CMs, FI, C1, OI, C2, PI, LN, II, IMs2]).

run(Shape, N) :-
    gen(Shape, N, Ts, First, Prod),
    setup_call_cleanup(
        forall(member(T, Ts), assertz(T)),
        row(Shape, N, First, Prod),
        ( forall(member(T, Ts), ignore(retract(T))), ingest_all )).

main :-
    with_output_to(string(_), run(ring, 12)),     % warm-up: first-call indexing is not a cost of the engine
    format("| Shape | Entities | Ingest ms (of which cycle check) | Failing query (inf) | Open query: proofs / inf | Product query: proofs / inf | Impact: lost / inf (ms) |~n"),
    format("|-------|----------|---------------------------------|---------------------|--------------------------|-----------------------------|-------------------------|~n"),
    sizes(Ns),
    forall(( member(Shape, [ring, chain, fan]), member(N, Ns) ), run(Shape, N)),
    halt(0).
