% lineage.pl - which evidence sources share an origin. Knowledge, not engine code.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% The curated bridges were authored by the tech team from the entities of the legacy
% knowledge base (several carry a from_entity detail), so kb and curated assertions are one
% origin, not two. Sources not listed here, such as hypothesis and fixture, are their own
% origin.
:- encoding(utf8).
:- multifile source_lineage/2.
:- dynamic source_lineage/2.

source_lineage(kb,      legacy_kb).
source_lineage(curated, legacy_kb).
