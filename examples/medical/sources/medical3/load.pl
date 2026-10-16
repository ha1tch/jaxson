% load.pl - load knowledge bases, vocabulary, validation, ingest and solver.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
:- encoding(utf8).
:- multifile tosid/5.
:- dynamic tosid/5.
:- consult('kb/biochemical.pl').
:- consult('kb/chemistry.pl').
:- consult('kb/pharma.pl').
:- consult('kb/medicine.pl').
:- consult('kb/amino_acids.pl').
:- consult('engine/vocab.pl').
:- consult('kb/ext_relations.pl').
:- consult('kb/bridges.pl').
:- consult('engine/validate.pl').
:- consult('engine/ingest.pl').
:- consult('engine/solve.pl').
:- consult('engine/evidence.pl').
:- consult('kb/lineage.pl').
:- initialization(ingest_all).
