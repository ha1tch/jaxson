% bridges.pl - curated cross-domain facts. Labels are resolved to tosid identifiers
% once, at ingest. An unresolvable label is reported, never silently ignored.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0

:- encoding(utf8).
:- dynamic alias/2, bridge/4.
:- multifile alias/2, bridge/4.

alias(cholecalciferol, 'Vitamin D3').
alias(cyanocobalamin,  'Vitamin B12').

bridge('Ferrous Sulfate',  provides, 'Iron',      [source(curated)]).
bridge('Magnesium Oxide',  provides, 'Magnesium', [source(curated)]).
bridge('Calcium',          inhibits, 'Iron',      [source(curated), from_entity('Calcium-Iron Interaction')]).
bridge('Ascorbic Acid',    enhances, 'Iron',      [source(curated), from_entity('Iron-Vitamin C Interaction')]).

% Subsumption replacing the legacy symptom_matches_indication/2 table.
% The legacy rows irregular_heartbeat, sleep_disorders and depression were NOT carried
% over: they asserted treatment links under the guise of subsumption.
bridge(concept(headache),    is_a, concept(pain), [source(curated)]).
bridge(concept(head_pain),   is_a, concept(pain), [source(curated)]).
bridge(concept(muscle_pain), is_a, concept(pain), [source(curated)]).
bridge(concept(body_aches),  is_a, concept(pain), [source(curated)]).
