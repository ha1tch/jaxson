% amino_acids.pl - precursor entities referenced by the original biochemical.pl
% (serotonin precursor(tryptophan), dopamine precursor(tyrosine)) but never defined there.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0

:- encoding(utf8).
:- multifile(tosid/5).
:- dynamic(tosid/5).

tosid('01', 'B8', 'TRP-AMI-ACD:AMI-ACD-P01', 'Tryptophan', [
    type(amino_acid),
    domain(biochemical),
    function(protein_synthesis)
]).

tosid('01', 'B8', 'TYR-AMI-ACD:AMI-ACD-P02', 'Tyrosine', [
    type(amino_acid),
    domain(biochemical),
    function(protein_synthesis)
]).
