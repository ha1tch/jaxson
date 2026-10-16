% release_check.pl - release hygiene for the tracking documents and the dormant guards.
% Run from the package root:
%   swipl tools/release_check.pl             check only; exit 0 if every check passes
%   swipl tools/release_check.pl --record    also stamp headers, the dormant-guard record
%                                            and the resolution-record baseline, then re-check
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Checks: documents carry the Version in VERSION and a Last reviewed line; the register holds
% no done items and its status table agrees with its detail sections; RESOLVED.md only grew
% since the recorded baseline; every dormant guard was exercised at the current version; the
% default suite, scale.pl and compare.sh all pass.

:- encoding(utf8).
:- initialization(main, main).
:- use_module(library(process)).
:- use_module(library(readutil)).

docs(['docs/TRACKING.md', 'docs/KNOWN_ISSUES.md', 'docs/CONTRACT.md']).
snapshot('docs/snapshot/RESOLVED.md').

main :-
    (   exists_file('VERSION') -> true ; format("run from the package root~n"), halt(2) ),
    current_prolog_flag(argv, Argv),
    (   memberchk('--record', Argv) -> Rec = true ; Rec = false ),
    read_file_to_string('VERSION', V0, []), normalize_space(string(V), V0),
    today(Today),
    run_checks_for_record(Rec, V, Today, GuardRun),
    Checks = [ check(headers,      headers(V)),
               check(register,     register),
               check(resolved,     resolved(Rec)),
               check(guards,       guards(V)),
               check(suite,        suite(GuardRun)),
               check(scale,        scale(GuardRun)),
               check(comparison,   comparison(GuardRun)) ],
    maplist(run_check, Checks, Results),
    format("Release check for version ~w~n~n| Check | Result | Detail |~n|-------|--------|--------|~n", [V]),
    forall(member(r(N, S, D), Results), format("| ~w | ~w | ~w |~n", [N, S, D])),
    (   forall(member(r(_, S, _), Results), S == ok)
    ->  (   Rec == true -> record_snapshot ; true ),
        format("~nAll checks passed.~n"), halt(0)
    ;   format("~nChecks failed.~n"), halt(1)
    ).

today(D) :- get_time(T), format_time(string(D), '%Y-%m-%d', T).

run_check(check(Name, Goal), r(Name, Status, Detail)) :-
    catch(( call(Goal, D) -> Status = ok, Detail = D ; Status = fail, Detail = "failed" ),
          fail(R), ( Status = fail, Detail = R )).

failwith(Fmt, Args) :- format(string(R), Fmt, Args), throw(fail(R)).

% ---- running the executable checks once (and stamping, when recording) --------------------

run_checks_for_record(Rec, V, Today, run(Bench, Scale, Cmp)) :-
    (   Rec == true -> stamp_headers(V, Today) ; true ),
    proc(swipl, ['bench/bench.pl'], Bench),
    proc(swipl, ['bench/scale.pl'], Scale),
    proc(sh, ['compare.sh'], Cmp),
    (   Rec == true, Bench = proc(0, _), Scale = proc(0, _), Cmp = proc(0, _)
    ->  stamp_guards(V, Today)
    ;   true ).

proc(Exe, Args, proc(Status, Out)) :-
    process_create(path(Exe), Args, [stdout(pipe(O)), stderr(null), process(Pid)]),
    read_string(O, _, Out), close(O),
    process_wait(Pid, exit(Status)).

suite(run(proc(S, Out), _, _), D) :-
    split_string(Out, "\n", "", Ls),
    aggregate_all(count, ( member(L, Ls), string_concat("PASS", _, L) ), N),
    (   S == 0, N > 0 -> format(string(D), "~d tests passed", [N])
    ;   failwith("suite exit ~w, ~d passed", [S, N]) ).

scale(run(_, proc(S, Out), _), D) :-
    split_string(Out, "\n", "", Ls),
    aggregate_all(count, ( member(L, Ls), member(P, ["| ring ", "| chain ", "| fan "]), string_concat(P, _, L) ), N),
    (   S == 0, N =:= 12 -> format(string(D), "~d measurement rows", [N])
    ;   failwith("scale exit ~w, ~d rows", [S, N]) ).

comparison(run(_, _, proc(S, Out)), D) :-
    split_string(Out, "\n", "", Ls),
    findall(A-B, ( member(L, Ls), sub_string(L, _, _, _, "exp1_"), \+ string_concat("PASS", _, L),
                   split_string(L, " ", " ", Ts0), exclude(==(""), Ts0, Ts),
                   append(_, [As, "/", Bs], Ts), number_string(A, As), number_string(B, Bs) ), Ps),
    (   S == 0, Ps = [_, _, _], forall(member(A-B, Ps), A =:= B) -> D = "Experiment 1 cases all pass"
    ;   failwith("comparison exit ~w, results ~w", [S, Ps]) ).

% ---- documents --------------------------------------------------------------------------------

doc_lines(F, Ls) :-
    read_file_to_string(F, S, [encoding(utf8)]), split_string(S, "\n", "", Ls).

line_value(Ls, Prefix, Val) :-
    member(L, Ls), string_concat(Prefix, V0, L), !, normalize_space(string(Val), V0).

headers(V, D) :-
    docs(Fs),
    forall(member(F, Fs),
           ( doc_lines(F, Ls),
             (   line_value(Ls, "Version: ", Got) -> true ; failwith("~w has no Version line", [F]) ),
             (   Got == V -> true ; failwith("~w is at ~w, VERSION is ~w", [F, Got, V]) ),
             (   line_value(Ls, "Last reviewed: ", _) -> true ; failwith("~w has no Last reviewed line", [F]) ) )),
    length(Fs, N), format(string(D), "~d documents at ~w", [N, V]).

stamp_headers(V, Today) :-
    docs(Fs),
    forall(member(F, Fs),
           rewrite(F, [ prefix("Version: ", V), prefix("Last reviewed: ", Today) ])).

rewrite(F, Rules) :-
    doc_lines(F, Ls),
    maplist(rewrite_line(Rules), Ls, New),
    atomic_list_concat(New, '\n', Text),
    setup_call_cleanup(open(F, write, S, [encoding(utf8)]), write(S, Text), close(S)).

rewrite_line(Rules, L, New) :-
    (   member(prefix(P, Val), Rules), string_concat(P, _, L)
    ->  string_concat(P, Val, New)
    ;   New = L ).

register(D) :-
    doc_lines('docs/TRACKING.md', Ls),
    findall(row(Id, Th, Pr, St),
            ( member(L, Ls), string_concat("| T-", _, L),
              split_string(L, "|", " ", [_, Id, _, Th, Pr, St|_]) ), Rows),
    findall(det(Id, Th, Pr, St),
            ( nth0(I, Ls, L), string_concat("### ", Rest, L), split_string(Rest, " ", "", [Id|_]),
              I1 is I + 1, nth0(I1, Ls, F), split_string(F, "·", " ", [T, P, S|_]),
              string_concat("Theme: ", Th, T), string_concat("Priority: ", Pr, P),
              string_concat("Status: ", St, S) ), Dets),
    (   Rows \== [] -> true ; failwith("no register rows", []) ),
    forall(member(row(Id, _, _, St), Rows),
           (   sub_string(St, _, _, _, "✓") -> failwith("~w is marked done but still in the register", [Id]) ; true )),
    forall(member(row(Id, Th, Pr, St), Rows),
           (   member(det(Id, Th2, Pr2, St2), Dets)
           ->  (   ( Th, Pr, St ) == ( Th2, Pr2, St2 ) -> true
               ;   failwith("~w: table and detail section disagree", [Id]) )
           ;   failwith("~w has a table row but no detail section", [Id]) )),
    forall(member(det(Id, _, _, _), Dets),
           (   member(row(Id, _, _, _), Rows) -> true ; failwith("~w has a detail section but no table row", [Id]) )),
    length(Rows, N), format(string(D), "~d open items, table and details agree, none done", [N]).

resolved(Rec, D) :-
    snapshot(Snap),
    doc_lines('docs/RESOLVED.md', Cur),
    (   exists_file(Snap)
    ->  doc_lines(Snap, Old),
        (   subseq(Old, Cur) -> length(Old, N), format(string(D), "all ~d baseline lines still present, in order", [N])
        ;   failwith("RESOLVED.md lost or reordered lines present in the baseline", []) )
    ;   Rec == true -> D = "no baseline yet; one will be recorded"
    ;   failwith("no baseline snapshot; run with --record", [])
    ).

subseq([], _).
subseq([X|Xs], [X|Ys]) :- !, subseq(Xs, Ys).
subseq(Xs, [_|Ys]) :- subseq(Xs, Ys).

record_snapshot :-
    snapshot(Snap), file_directory_name(Snap, Dir), make_directory_path(Dir),
    copy_file('docs/RESOLVED.md', Snap).

guards_rows(Rows) :-
    doc_lines('docs/TRACKING.md', Ls),
    findall(g(G, Last), ( member(L, Ls), string_concat("| `", _, L),
                          split_string(L, "|", " ", [_, G, _, _, _, Last|_]) ), Rows).

guards(V, D) :-
    guards_rows(Rows),
    (   Rows \== [] -> true ; failwith("no dormant guard rows", []) ),
    forall(member(g(G, Last), Rows),
           (   sub_string(Last, _, _, _, V) -> true
           ;   failwith("~w last exercised: ~w (not at ~w)", [G, Last, V]) )),
    length(Rows, N), format(string(D), "~d guards exercised at ~w", [N, V]).

stamp_guards(V, Today) :-
    current_prolog_flag(arch, Arch), current_prolog_flag(version, Ver),
    Maj is Ver // 10000, Min is (Ver // 100) mod 100, Pat is Ver mod 100,
    format(string(Stamp), "~w, ~w, swipl ~d.~d.~d, ~w", [Today, Arch, Maj, Min, Pat, V]),
    doc_lines('docs/TRACKING.md', Ls),
    maplist(stamp_line(Stamp), Ls, New),
    atomic_list_concat(New, '\n', Text),
    setup_call_cleanup(open('docs/TRACKING.md', write, S, [encoding(utf8)]), write(S, Text), close(S)).

stamp_line(Stamp, L, New) :-
    (   string_concat("| `", _, L), split_string(L, "|", " ", [_, G, Ga, Hw, In, _|_])
    ->  atomic_list_concat([G, Ga, Hw, In, Stamp], ' | ', Mid),
        format(string(New), "| ~w |", [Mid])
    ;   New = L ).
