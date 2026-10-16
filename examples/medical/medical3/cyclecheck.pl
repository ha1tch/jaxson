% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
:- prolog_load_context(directory, D), atom_concat(D, '/../sources/medical3/load.pl', F), consult(F).
:- assertz(bridge('Tryptophan', precursor_of, 'Tyrosine', [source(curated)])).
:- assertz(bridge('Tyrosine', precursor_of, 'Tryptophan', [source(curated)])).
:- assertz(tosid('10', 'E4', 'TRP-TAB-500:TRP-AMI-T01', 'Tryptophan 500mg Tablet',
     [type(pharmaceutical), domain(pharma), active_ingredient('Tryptophan'), indication([sleep_disorders])])).
t :- ingest_all,
     P = 'TRP-TAB-500:TRP-AMI-T01', T = 'TRP-AMI-ACD:AMI-ACD-P01',
     format("open query, objects found for the Tryptophan tablet:~n"),
     forall(( solve(rel(P, supplies_precursor_of, O), _) ), ( label(O, L), format("  ~w~n", [L]) )),
     assess(rel(P, supplies_precursor_of, T), St, _),
     format("assess(tablet supplies_precursor_of Tryptophan) with the object given: ~w~n", [St]),
     ( solve(rel(T, precursor_of, T), _) -> format("solve(Tryptophan precursor_of Tryptophan), both given: reflexive proof found~n")
     ; format("solve(Tryptophan precursor_of Tryptophan), both given: no proof~n") ),
     ( solve(rel(T, precursor_of, X), Pr), X == T -> format("open object: a reflexive proof for Tryptophan IS found~n") ; format("open object: no reflexive proof~n") ).
:- initialization((t, halt)).
