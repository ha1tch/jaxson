% conformance.pl - conformance suite for the declarative reasoning language.
% Copyright (c) 2026 haitch <h@ual.li>
% SPDX-License-Identifier: Apache-2.0
%
% Loaded by bench.pl. Every fixture is knowledge only (tosid/5, bridge/4, rel_pred/2,
% rel_axiom/1, compose_rule/3, schema/4). No engine file is touched by any of them.
% Entities are synthetic ("Cf ..."): these tests exercise inference validity, not
% medical truth.

:- encoding(utf8).
:- multifile fixture/2, test/5, check/1.
:- discontiguous fixture/2, test/5, check/1.

% ---- fixture helpers ---------------------------------------------------------------------

cf_ent(Name, tosid('10', 'F2', Id, Name, [type(compound), domain(chemistry)])) :-
    cf_id(Name, Id).
cf_prod(Name, Extra, tosid('10', 'E4', Id, Name,
                           [type(pharmaceutical), domain(pharma)|Extra])) :-
    cf_id(Name, Id).
cf_id(Name, Id) :-
    atom_string(Name, S0), string_upper(S0, S1),
    split_string(S1, " ", "", Ps), atomic_list_concat(Ps, '-', Tag),
    atomic_list_concat(['CNF-', Tag, ':X'], Id).
cf_br(A, P, B, bridge(A, P, B, [source(fixture)])).

cf_ents(Names, Ts) :- maplist(cf_ent, Names, Ts).

% ---- fixture: axioms ---------------------------------------------------------------------
% Alpha competes_with Beta (one direction stored). Precursor chain Alpha > Beta > Gamma >
% Delta. Alpha inhibits Eps (plain inhibition, must stay one-directional). ProdA contains
% Alpha, ProdB contains Beta.

fixture(cf_axioms, Ts) :-
    cf_ents(['Cf Alpha', 'Cf Beta', 'Cf Gamma', 'Cf Delta', 'Cf Eps'], Es),
    cf_prod('Cf ProdA', [], PA), cf_prod('Cf ProdB', [], PB),
    findall(B, ( member(A-P-C, [ 'Cf Alpha'-competes_with-'Cf Beta',
                                 'Cf Alpha'-precursor_of-'Cf Beta',
                                 'Cf Beta'-precursor_of-'Cf Gamma',
                                 'Cf Gamma'-precursor_of-'Cf Delta',
                                 'Cf Alpha'-inhibits-'Cf Eps',
                                 'Cf ProdA'-contains-'Cf Alpha',
                                 'Cf ProdB'-contains-'Cf Beta' ]),
                 cf_br(A, P, C, B) ), Bs),
    append([Es, [PA, PB], Bs], Ts).

% ---- fixture: transitive cycle of length 12 ---------------------------------------------

fixture(cf_ring, Ts) :-
    numlist(1, 12, Is),
    findall(N, ( member(I, Is), format(atom(N), 'Cf Ring~d', [I]) ), Names),
    cf_ents(Names, Es),
    findall(B, ( member(I, Is), J is I mod 12 + 1,
                 format(atom(A), 'Cf Ring~d', [I]), format(atom(C), 'Cf Ring~d', [J]),
                 cf_br(A, precursor_of, C, B) ), Bs),
    append(Es, Bs, Ts).

% ---- fixture: invalid declarations ------------------------------------------------------

fixture(cf_bad_vocab, [
    rel_axiom(transitive(cf_undeclared)),
    rel_axiom(symmetric(supplies_precursor_of)),
    rel_axiom(subrel(inhibits, competes_with)),
    compose_rule(contains, cf_nope, provides),
    schema(cf_bad_schema, rel(_, may_address, _), [rel(_, contains, _)], supported),
    schema(cf_bad_guard, rel(P, boosts, Q),
           [rel(P, contains, Q), guard(cf_no_such_guard(P))], supported)
]).

% ---- fixture: evidence merging -----------------------------------------------------------

fixture(cf_evidence, Ts) :-
    cf_ents(['Cf E1', 'Cf E2', 'Cf E3'], Es),
    Bs = [ bridge('Cf E1', provides, 'Cf E2', [source(curated)]),
           bridge('Cf E1', provides, 'Cf E2', [source(hypothesis)]),
           bridge('Cf E1', provides, 'Cf E3', [source(hypothesis)]),
           bridge('Cf E1', enhances, 'Cf E2', [source(curated)]),
           bridge('Cf E1', enhances, 'Cf E2', [source(curated)]) ],
    append(Es, Bs, Ts).

% ---- fixture: defeaters ------------------------------------------------------------------

fixture(cf_defeat, Ts) :-
    cf_ent('Cf Sub', Sub),
    cf_prod('Cf DefA', [indication([cf_symp]), not_indicated_for([cf_symp])], A),
    cf_prod('Cf DefB', [indication([cf_symp])], B),
    cf_prod('Cf DefC', [not_indicated_for([cf_symp2])], C),
    Sub = tosid(T1, T2, Id, Nm, Props),
    Sub1 = tosid(T1, T2, Id, Nm, [deficiency_symptoms([cf_symp2])|Props]),
    cf_br('Cf DefC', contains, 'Cf Sub', Br),
    Ts = [Sub1, A, B, C, Br].

% ---- fixture: two independent routes to one conclusion ----------------------------------

fixture(cf_alt, Ts) :-
    cf_ent('Cf A1', A1), cf_ent('Cf A2', A2),
    cf_ent('Cf S', tosid(T1, T2, Id, Nm, Pr)),
    S1 = tosid(T1, T2, Id, Nm, [deficiency_symptoms([cf_symp3])|Pr]),
    cf_prod('Cf AltProd', [], P),
    findall(B, ( member(X-Pd-Y, [ 'Cf AltProd'-contains-'Cf A1', 'Cf AltProd'-contains-'Cf A2',
                                  'Cf A1'-provides-'Cf S', 'Cf A2'-provides-'Cf S' ]),
                 cf_br(X, Pd, Y, B) ), Bs),
    append([[A1, A2, S1, P], Bs], Ts).

% ---- fixture: new relation classes declared as knowledge only ---------------------------

fixture(cf_newrel, Ts) :-
    cf_ents(['Cf Chelator', 'Cf Metal', 'Cf Syn1', 'Cf Syn2'], Es),
    cf_prod('Cf PChel', [], P1), cf_prod('Cf PMetal', [], P2),
    cf_prod('Cf PSyn1', [], P3), cf_prod('Cf PSyn2', [], P4),
    findall(B, ( member(X-Pd-Y, [ 'Cf PChel'-contains-'Cf Chelator',
                                  'Cf PMetal'-contains-'Cf Metal',
                                  'Cf PSyn1'-contains-'Cf Syn1',
                                  'Cf PSyn2'-contains-'Cf Syn2',
                                  'Cf Chelator'-cf_chelates-'Cf Metal',
                                  'Cf Syn1'-cf_synergises-'Cf Syn2' ]),
                 cf_br(X, Pd, Y, B) ), Bs),
    Decl = [ rel_pred(cf_chelates,   [kind(base), polarity(negative), doc("Subject binds the object")]),
             rel_pred(cf_synergises, [kind(base), polarity(positive), doc("Mutual potentiation")]),
             rel_axiom(subrel(cf_chelates, inhibits)),
             rel_axiom(symmetric(cf_synergises)),
             rel_axiom(subrel(cf_synergises, enhances)) ],
    append([Decl, Es, [P1, P2, P3, P4], Bs], Ts).

% ---- tests: symmetric, subrel, transitive, inverse, composition -------------------------

test(s1, cf_symmetric, [cf_axioms], 'symmetric: stored one way, derivable the other',
     derives(rel(n('Cf Beta'), competes_with, n('Cf Alpha')))).
test(s2, cf_symmetric, [cf_axioms], 'subrel: competes_with counts as inhibits',
     derives(rel(n('Cf Alpha'), inhibits, n('Cf Beta')))).
test(s3, cf_symmetric, [cf_axioms], 'symmetric and subrel combine for the reverse direction',
     derives(rel(n('Cf Beta'), inhibits, n('Cf Alpha')))).
test(s4, cf_symmetric, [cf_axioms], 'plain inhibits stays one-directional',
     cannot_derive(rel(n('Cf Eps'), inhibits, n('Cf Alpha')))).
test(s5, cf_symmetric, [cf_axioms], 'transitive precursor_of is not symmetric',
     cannot_derive(rel(n('Cf Beta'), precursor_of, n('Cf Alpha')))).

test(t1, cf_transitive, [cf_axioms], 'transitive: three-hop chain closes',
     derives(rel(n('Cf Alpha'), precursor_of, n('Cf Delta')))).
test(t2, cf_transitive, [cf_axioms], 'transitive: closure does not run backwards',
     cannot_derive(rel(n('Cf Delta'), precursor_of, n('Cf Alpha')))).
test(t3, cf_transitive, [cf_axioms, cf_ring], 'cycle of 12: failing query terminates quickly',
     terminates_failing(rel(n('Cf Ring1'), precursor_of, n('Cf Alpha')))).
test(t4, cf_transitive, [cf_ring], 'cycle of 12: no reflexive proof is manufactured',
     cannot_derive(rel(n('Cf Ring1'), precursor_of, n('Cf Ring1')))).
test(t5, cf_transitive, [cf_ring], 'cycle in a transitive relation is reported at ingest',
     issue(cyclic(precursor_of, _))).

test(i1, cf_inverse, [cf_axioms], 'inverse: derived_from is the converse of precursor_of',
     derives(rel(n('Cf Beta'), derived_from, n('Cf Alpha')))).
test(i2, cf_inverse, [cf_axioms], 'inverse: no derivation in the wrong direction',
     cannot_derive(rel(n('Cf Alpha'), derived_from, n('Cf Beta')))).
test(i3, cf_inverse, [cf_axioms], 'inverse composes with transitive closure',
     derives(rel(n('Cf Delta'), derived_from, n('Cf Alpha')))).

test(c1, cf_compose, [cf_axioms], 'composition over a stored precursor fact',
     derives(rel(n('Cf ProdA'), supplies_precursor_of, n('Cf Beta')))).
test(c2, cf_compose, [cf_axioms], 'composition over a transitively derived precursor fact',
     derives(rel(n('Cf ProdA'), supplies_precursor_of, n('Cf Delta')))).
test(c3, cf_compose, [cf_axioms], 'composition does not run against the precursor direction',
     cannot_derive(rel(n('Cf ProdB'), supplies_precursor_of, n('Cf Alpha')))).
test(c4, cf_compose, [cf_axioms], 'unmodified interference schema sees competition through subrel',
     flagged('Cf ProdA', interferes_with, 'Cf ProdB')).
test(c5, cf_compose, [cf_axioms], 'interference is also found in the reverse direction',
     flagged('Cf ProdB', interferes_with, 'Cf ProdA')).
test(c6, cf_compose, [cf_axioms], 'retracting the first chain link removes the downstream conclusion',
     impact(rel(n('Cf Alpha'), precursor_of, n('Cf Beta')),
            rel(n('Cf ProdA'), supplies_precursor_of, n('Cf Delta')))).

% ---- tests: validation ------------------------------------------------------------------

test(v0, cf_validate, [], 'shipped vocabulary has no validation errors', vocab_clean).
test(v1, cf_validate, [cf_bad_vocab], 'axiom over an undeclared predicate is rejected',
     rejected(axiom(transitive(cf_undeclared)))).
test(v2, cf_validate, [cf_bad_vocab], 'symmetric over a derived predicate is rejected',
     rejected(axiom(symmetric(supplies_precursor_of)))).
test(v3, cf_validate, [cf_bad_vocab], 'subrel cycle is rejected on both members',
     ( rejected(axiom(subrel(inhibits, competes_with))),
       rejected(axiom(subrel(competes_with, inhibits))) )).
test(v4, cf_validate, [cf_bad_vocab], 'composition over an undeclared predicate is rejected',
     rejected(compose_rule(contains, cf_nope, provides))).
test(v5, cf_validate, [cf_bad_vocab], 'schema with unbound head variables is rejected',
     rejected(schema(cf_bad_schema))).
test(v6, cf_validate, [cf_bad_vocab], 'schema with an unknown guard is rejected',
     rejected(schema(cf_bad_guard))).
test(v7, cf_validate, [cf_axioms, cf_bad_vocab], 'a rejected axiom is disabled, not half-applied',
     ( inactive_axiom(subrel(competes_with, inhibits)),
       cannot_derive(rel(n('Cf Alpha'), inhibits, n('Cf Beta'))) )).
test(v8, cf_validate, [cf_axioms, cf_bad_vocab], 'rejections do not disable unrelated vocabulary',
     derives(rel(n('Cf ProdA'), supplies_precursor_of, n('Cf Delta')))).

% ---- tests: evidence ---------------------------------------------------------------------

test(ev1, cf_evidence, [cf_evidence], 'two sources for one fact are both retained',
     evidence_sources(rel(n('Cf E1'), provides, n('Cf E2')), [curated, hypothesis])).
test(ev2, cf_evidence, [cf_evidence], 'merged fact takes the strongest source status',
     assess_is(rel(n('Cf E1'), provides, n('Cf E2')), supported)).
test(ev3, cf_evidence, [cf_evidence], 'a hypothesis-only fact stays conditional',
     assess_is(rel(n('Cf E1'), provides, n('Cf E3')), conditional)).
test(ev4, cf_evidence, [cf_evidence], 'an identical assertion is not counted twice',
     ev_count(rel(n('Cf E1'), enhances, n('Cf E2')), 1)).

% ---- tests: defeaters --------------------------------------------------------------------

test(d1, cf_defeat, [cf_defeat], 'supported against supported: conflicted',
     assess_is(rel(n('Cf DefA'), may_address, c(cf_symp)), conflicted)).
test(d2, cf_defeat, [cf_defeat], 'a product without a defeater stays supported',
     assess_is(rel(n('Cf DefB'), may_address, c(cf_symp)), supported)).
test(d3, cf_defeat, [cf_defeat], 'supported defeater against a conditional conclusion: refuted',
     assess_is(rel(n('Cf DefC'), may_address, c(cf_symp2)), refuted)).
test(d4, cf_defeat, [cf_defeat], 'recommendations leave out a refuted product',
     rec_excludes([cf_symp2], 'Cf DefC')).
test(d5, cf_defeat, [cf_defeat], 'recommendations keep a conflicted product',
     rec_includes([cf_symp], 'Cf DefA')).
test(d6, cf_defeat, [cf_defeat], 'defeaters leave unrelated conclusions untouched',
     addresses('Iron Supplement 65mg', fatigue, conditional)).

% ---- tests: alternative proofs and impact -----------------------------------------------

test(alt1, cf_alternatives, [cf_alt], 'both routes to the conclusion are preserved',
     proof_count_ge(rel(n('Cf AltProd'), may_address, c(cf_symp3)), 2)).
test(alt2, cf_alternatives, [cf_alt], 'retracting one route leaves the conclusion standing',
     impact_set([rel(n('Cf AltProd'), contains, n('Cf A1'))],
                rel(n('Cf AltProd'), may_address, c(cf_symp3)), survives)).
test(alt3, cf_alternatives, [cf_alt], 'retracting the other route also leaves it standing',
     impact_set([rel(n('Cf AltProd'), contains, n('Cf A2'))],
                rel(n('Cf AltProd'), may_address, c(cf_symp3)), survives)).
test(alt4, cf_alternatives, [cf_alt], 'retracting both routes together removes the conclusion',
     impact_set([rel(n('Cf AltProd'), contains, n('Cf A1')),
                 rel(n('Cf AltProd'), contains, n('Cf A2'))],
                rel(n('Cf AltProd'), may_address, c(cf_symp3)), lost)).

% ---- tests: new relation classes, knowledge only ---------------------------------------

test(nr1, cf_new_relation, [cf_axioms, cf_newrel], 'new relation plugs into existing interference reasoning',
     flagged('Cf PChel', interferes_with, 'Cf PMetal')).
test(nr2, cf_new_relation, [cf_axioms, cf_newrel], 'new relation is not symmetric unless declared',
     cannot_derive(rel(n('Cf PMetal'), interferes_with, n('Cf PChel')))).
test(nr3, cf_new_relation, [cf_axioms, cf_newrel], 'new symmetric relation reaches the existing synergy schema',
     flagged('Cf PSyn1', boosts, 'Cf PSyn2')).
test(nr4, cf_new_relation, [cf_axioms, cf_newrel], 'symmetry of the new relation gives the reverse conclusion',
     flagged('Cf PSyn2', boosts, 'Cf PSyn1')).
test(nr5, cf_new_relation, [cf_newrel], 'the new declarations validate cleanly', vocab_clean).

% ---- checks ------------------------------------------------------------------------------

check(assess_is(G, S)) :- conv(G, C), assess(C, Got, _), Got == S.
check(rejected(Ctx)) :- vocab_issue(error, _, Ctx).
check(inactive_axiom(Ax)) :- rel_axiom(Ax), \+ active_axiom(Ax).
check(vocab_clean) :- \+ vocab_issue(error, _, _).
check(evidence_sources(T, Want)) :-
    conv(T, rel(A, P, B)), rel(A, P, B, M),
    findall(Sr, member(ev(Sr, _, _), M), L),
    msort(L, S1), msort(Want, S2), S1 == S2.
check(ev_count(T, N)) :-
    conv(T, rel(A, P, B)), rel(A, P, B, M), length(M, N).
check(rec_includes(Syms, PN)) :-
    eid(PN, P), recommend(Syms, Recs), memberchk(rec(P, _, _), Recs).
check(rec_excludes(Syms, PN)) :-
    eid(PN, P), recommend(Syms, Recs), \+ memberchk(rec(P, _, _), Recs).
check(proof_count_ge(T, N)) :-
    conv(T, C), findall(Pr, solve(C, Pr), L0), sort(L0, L), length(L, K), K >= N.
check(impact_set(Ts0, C0, Outcome)) :-
    maplist(conv, Ts0, Ts), conv(C0, C),
    once(solve(C, _)),
    retract_impact_set(Ts, Lost),
    (   Outcome == lost -> memberchk(C, Lost) ; \+ memberchk(C, Lost) ).
check(terminates_failing(T)) :-
    conv(T, C), call_with_time_limit(5, \+ solve(C, _)).
check(all(Cs)) :- maplist(check, Cs).
check((A, B)) :- check(A), check(B).

% ============ batch 2: depth bound, validator warnings, remaining error classes ==========

% ---- fixture: ten-entity precursor chain (nine hops) ------------------------------------

fixture(cf_chain, Ts) :-
    numlist(1, 10, Is),
    findall(N, ( member(I, Is), format(atom(N), 'Cf Chain~d', [I]) ), Names),
    cf_ents(Names, Es),
    findall(B, ( between(1, 9, I), J is I + 1,
                 format(atom(A), 'Cf Chain~d', [I]), format(atom(C), 'Cf Chain~d', [J]),
                 cf_br(A, precursor_of, C, B) ), Bs),
    append(Es, Bs, Ts).

% ---- fixture: constructs that earn warnings but stay active ------------------------------

fixture(cf_warn, Ts) :-
    cf_ents(['Cf W1', 'Cf W2', 'Cf W3'], Es),
    findall(B, member(B, [ bridge('Cf W1', cf_w_pos, 'Cf W2', [source(fixture)]),
                           bridge('Cf W2', cf_w_pos, 'Cf W3', [source(fixture)]) ]), Bs),
    Decl = [ rel_pred(cf_w_pos,  [kind(base),    polarity(positive), doc("warning fixture")]),
             rel_pred(cf_w_negb, [kind(base),    polarity(negative), doc("warning fixture")]),
             rel_pred(cf_w_neg,  [kind(derived), polarity(negative), doc("warning fixture")]),
             rel_pred(cf_w_der,  [kind(derived), polarity(positive), doc("no producer")]),
             compose_rule(cf_w_pos, cf_w_pos, cf_w_neg),
             rel_axiom(subrel(cf_w_pos, cf_w_negb)),
             schema(cf_w_unreach, rel(A, cf_w_pos, B), [rel(A, cf_w_der, B)], supported),
             schema(cf_w_rec,     rel(A, cf_w_pos, B), [rel(A, cf_w_pos, B)], supported) ],
    append([Decl, Es, Bs], Ts).

% ---- fixture: the remaining error classes -----------------------------------------------

fixture(cf_bad2, [
    rel_pred(cf_mal, [kind(base)]),
    rel_pred(cf_dup, [kind(base), polarity(neutral), doc("dup")]),
    rel_pred(cf_dup, [kind(base), polarity(neutral), doc("dup")]),
    rel_axiom(frobnicate(cf_dup)),
    rel_axiom(inverse(inhibits, enhances)),
    rel_axiom(subrel(enhances, enhances)),
    rel_axiom(inverse(precursor_of, derived_from)),
    schema(cf_s_status, rel(P, boosts, Q), [rel(P, contains, Q)], bogus_status),
    schema(cf_s_dup,    rel(P, boosts, Q), [rel(P, contains, Q)], supported),
    schema(cf_s_dup,    rel(P, boosts, Q), [rel(P, contains, Q)], supported),
    schema(cf_s_head,   not_a_head, [], supported),
    schema(cf_s_varp,   rel(P, boosts, Q), [rel(P, _Pred, Q)], supported),
    schema(cf_s_goal,   rel(P, boosts, Q), [rel(P, contains, Q), not_a_goal(x)], supported),
    schema(cf_s_unb,    rel(P, boosts, Q), [guard(distinct(P, _Z)), rel(P, contains, Q)], supported),
    prop_map(cf_prop1, supplies_precursor_of, concept),
    prop_map(cf_prop2, cf_undeclared_pred, concept),
    prop_map(cf_prop3, contains, bogus_mode)
]).

% ---- tests: depth bound ------------------------------------------------------------------

test(b1, cf_depth, [cf_chain], 'seven hops are derivable at the default depth',
     derives(rel(n('Cf Chain1'), precursor_of, n('Cf Chain8')))).
test(b2, cf_depth, [cf_chain], 'eight hops are beyond the default depth bound',
     cannot_derive(rel(n('Cf Chain1'), precursor_of, n('Cf Chain9')))).
test(b3, cf_depth, [cf_chain], 'the same nine-hop conclusion is found at a deeper explicit bound',
     derives_within(rel(n('Cf Chain1'), precursor_of, n('Cf Chain10')), 12)).
test(b4, cf_depth, [cf_chain], 'failure at the bound is reported unknown, never refuted',
     assess_is(rel(n('Cf Chain1'), precursor_of, n('Cf Chain10')), unknown)).
test(b5, cf_depth, [cf_chain], 'the reverse direction stays underivable at any bound',
     cannot_within(rel(n('Cf Chain10'), precursor_of, n('Cf Chain1')), 20)).

% ---- tests: warnings keep the rule active -----------------------------------------------

test(w1, cf_warnings, [cf_warn], 'composition with a polarity mismatch is warned',
     warned(polarity_mismatch, compose_rule(cf_w_pos, cf_w_pos, cf_w_neg))).
test(w2, cf_warnings, [cf_warn], 'subrel across opposite polarities is warned',
     warned(polarity_mismatch, axiom(subrel(cf_w_pos, cf_w_negb)))).
test(w3, cf_warnings, [cf_warn], 'a derived predicate with no producer is warned',
     warned(no_producer, pred(cf_w_der))).
test(w4, cf_warnings, [cf_warn], 'a schema over a producer-less predicate is warned as unreachable',
     warned(unreachable_schema(cf_w_der), schema(cf_w_unreach))).
test(w5, cf_warnings, [cf_warn], 'a self-recursive schema is warned',
     warned(self_recursive_schema, schema(cf_w_rec))).
test(w6, cf_warnings, [cf_warn], 'a warning does not disable the composition rule',
     derives(rel(n('Cf W1'), cf_w_neg, n('Cf W3')))).
test(w7, cf_warnings, [cf_warn], 'warnings are not errors', vocab_clean).

% ---- tests: remaining error classes -----------------------------------------------------

test(v9,  cf_validate, [cf_bad2], 'relation without polarity is rejected',
     rejected_as(malformed_rel_pred, cf_mal)).
test(v10, cf_validate, [cf_bad2], 'relation declared twice is rejected',
     rejected_as(duplicate_rel_pred, cf_dup)).
test(v11, cf_validate, [cf_bad2], 'unknown axiom form is rejected',
     rejected_as(unknown_axiom, frobnicate(cf_dup))).
test(v12, cf_validate, [cf_bad2], 'inverse must define a derived relation',
     rejected_as(inverse_must_define_derived(inhibits), axiom(inverse(inhibits, enhances)))).
test(v13, cf_validate, [cf_bad2], 'self-referential subrel is rejected',
     rejected_as(self_referential_axiom, axiom(subrel(enhances, enhances)))).
test(v14, cf_validate, [cf_bad2], 'inverse cycle disables the legitimate member too',
     ( rejected_as(inverse_cycle, axiom(inverse(derived_from, precursor_of))),
       inactive_axiom(inverse(derived_from, precursor_of)) )).
test(v15, cf_validate, [cf_bad2], 'schema with an unknown status is rejected',
     rejected_as(bad_status(bogus_status), schema(cf_s_status))).
test(v16, cf_validate, [cf_bad2], 'duplicate schema name is rejected',
     rejected_as(duplicate_schema_name, schema(cf_s_dup))).
test(v17, cf_validate, [cf_bad2], 'schema with a non-relation head is rejected',
     rejected_as(malformed_head, schema(cf_s_head))).
test(v18, cf_validate, [cf_bad2], 'schema with a variable predicate is rejected',
     rejected_as(variable_predicate, schema(cf_s_varp))).
test(v19, cf_validate, [cf_bad2], 'schema with an unknown body goal is rejected',
     rejected_as(unknown_body_goal(_), schema(cf_s_goal))).
test(v20, cf_validate, [cf_bad2], 'guard over a not-yet-bound variable is rejected',
     rejected_as(guard_uses_unbound_variable(_), schema(cf_s_unb))).
test(v21, cf_validate, [cf_bad2], 'property map onto a derived relation is rejected',
     rejected_as(prop_map_needs_base_predicate(supplies_precursor_of), prop_map(cf_prop1))).
test(v22, cf_validate, [cf_bad2], 'property map onto an undeclared relation is rejected',
     rejected_as(undeclared_predicate(cf_undeclared_pred), prop_map(cf_prop2))).
test(v23, cf_validate, [cf_bad2], 'property map with an unknown mode is rejected',
     rejected_as(bad_prop_mode(bogus_mode), prop_map(cf_prop3))).

check(derives_within(T, D)) :- conv(T, C), once(solve(C, D, _)).
check(warned(K, Ctx)) :- vocab_issue(warning, K, Ctx).
check(rejected_as(K, Ctx)) :- vocab_issue(error, K, Ctx).
check(cannot_within(T, D)) :- conv(T, C), \+ solve(C, D, _).

% ============ batch 3: closure over derived edges, mixed alternatives, cycles, patterns ==

% ---- fixture: closure must see subrel and symmetric edges --------------------------------

fixture(cf_closure, Ts) :-
    cf_ents(['Cf In1', 'Cf Mid', 'Cf Top', 'Cf K1', 'Cf K2', 'Cf K3', 'Cf K4'], Es),
    findall(B, ( member(X-Pd-Y, [ 'Cf In1'-cf_inside-'Cf Mid', 'Cf Mid'-cf_part_of-'Cf Top',
                                  'Cf K1'-cf_linked-'Cf K2', 'Cf K2'-cf_linked-'Cf K3' ]),
                 cf_br(X, Pd, Y, B) ), Bs),
    Decl = [ rel_pred(cf_part_of, [kind(base), polarity(positive), doc("transitive containment")]),
             rel_pred(cf_inside,  [kind(base), polarity(positive), doc("immediately inside")]),
             rel_pred(cf_linked,  [kind(base), polarity(neutral),  doc("equivalence-like link")]),
             rel_axiom(transitive(cf_part_of)),
             rel_axiom(subrel(cf_inside, cf_part_of)),
             rel_axiom(symmetric(cf_linked)),
             rel_axiom(transitive(cf_linked)) ],
    append([Decl, Es, Bs], Ts).

% ---- fixture: alternatives of different strength -----------------------------------------

fixture(cf_mixed, Ts) :-
    cf_ent('Cf MixSub', tosid(T1, T2, Id, Nm, Pr)),
    Sub = tosid(T1, T2, Id, Nm, [deficiency_symptoms([cf_symp4])|Pr]),
    cf_prod('Cf MixP', [indication([cf_symp4])], P),
    cf_br('Cf MixP', contains, 'Cf MixSub', B),
    Ts = [Sub, P, B].

% ---- fixture: a cycle with two exits ----------------------------------------------------

fixture(cf_cyc, Ts) :-
    cf_ents(['Cf CA', 'Cf CB', 'Cf CC', 'Cf CD'], Es),
    cf_prod('Cf ProdCyc', [], P),
    findall(B, ( member(X-Pd-Y, [ 'Cf CA'-precursor_of-'Cf CB', 'Cf CB'-precursor_of-'Cf CC',
                                  'Cf CC'-precursor_of-'Cf CA', 'Cf CA'-precursor_of-'Cf CD',
                                  'Cf CB'-precursor_of-'Cf CD', 'Cf ProdCyc'-contains-'Cf CA' ]),
                 cf_br(X, Pd, Y, B) ), Bs),
    append([Es, [P], Bs], Ts).

% ---- fixture: a defeater that is itself derived ------------------------------------------

fixture(cf_defeat2, Ts) :-
    cf_prod('Cf DefD', [indication([cf_symp5])], P),
    Ts = [ rel_pred(cf_forbids, [kind(base), polarity(negative), doc("hard contraindication")]),
           rel_axiom(subrel(cf_forbids, not_indicated_for)),
           P,
           bridge('Cf DefD', cf_forbids, concept(cf_symp5), [source(fixture)]) ].

% ---- fixture: a new reasoning pattern, knowledge only -----------------------------------
% A product whose active inhibits a substance that a deficiency condition contains may
% aggravate that condition's symptom; that is a defeater of any claim to address the same symptom.

fixture(cf_pattern, Ts) :-
    cf_prod('Cf CalcOnly', [], P1),
    cf_prod('Cf CalcInd', [indication([fatigue])], P2),
    Decl = [ rel_pred(cf_may_aggravate,
                      [kind(derived), polarity(negative), headline(true),
                       doc("Product may aggravate the deficiency symptom")]),
             schema(cf_aggravate,
                    rel(P, cf_may_aggravate, concept(Sym)),
                    [ rel(P, active_in, X), rel(X, inhibits, Y), rel(E, contains, Y),
                      rel(E, deficiency_manifests_as, concept(Sym)), guard(is_product(P)) ],
                    conditional),
             rel_axiom(defeats(cf_may_aggravate, may_address)) ],
    Ts = [ P1, P2, bridge('Cf CalcOnly', contains, 'Calcium', [source(fixture)]),
           bridge('Cf CalcInd',  contains, 'Calcium', [source(fixture)]) | Decl ].

% ---- tests: closure over derived edges --------------------------------------------------

test(cl1, cf_closure, [cf_closure], 'closure sees a subrel edge as its first hop',
     derives(rel(n('Cf In1'), cf_part_of, n('Cf Top')))).
test(cl2, cf_closure, [cf_closure], 'the subrel edge alone is derivable',
     derives(rel(n('Cf In1'), cf_part_of, n('Cf Mid')))).
test(cl3, cf_closure, [cf_closure], 'closure over subrel does not run backwards',
     cannot_derive(rel(n('Cf Top'), cf_part_of, n('Cf In1')))).
test(cl4, cf_closure, [cf_closure], 'symmetric plus transitive closes in the reverse direction',
     derives(rel(n('Cf K3'), cf_linked, n('Cf K1')))).
test(cl5, cf_closure, [cf_closure], 'symmetric plus transitive closes in the stored direction',
     derives(rel(n('Cf K1'), cf_linked, n('Cf K3')))).
test(cl6, cf_closure, [cf_closure], 'an unlinked entity stays unlinked',
     cannot_derive(rel(n('Cf K4'), cf_linked, n('Cf K1')))).
test(cl7, cf_closure, [cf_closure], 'no reflexive link is manufactured',
     cannot_derive(rel(n('Cf K1'), cf_linked, n('Cf K1')))).
test(cl8, cf_closure, [cf_closure], 'the closure declarations validate cleanly', vocab_clean).

% ---- tests: alternatives of different strength ------------------------------------------

test(m1, cf_alternatives, [cf_mixed], 'both a supported and a conditional proof are preserved',
     proof_statuses(rel(n('Cf MixP'), may_address, c(cf_symp4)), [conditional, supported])).
test(m2, cf_alternatives, [cf_mixed], 'the assessment takes the strongest alternative',
     assess_is(rel(n('Cf MixP'), may_address, c(cf_symp4)), supported)).

% ---- tests: impact over a cyclic dependency ---------------------------------------------

test(y1, cf_cycle_impact, [cf_cyc], 'the cycle is reported at ingest', issue(cyclic(precursor_of, _))).
test(y2, cf_cycle_impact, [cf_cyc], 'the conclusion is derivable despite the cycle',
     derives(rel(n('Cf ProdCyc'), supplies_precursor_of, n('Cf CD')))).
test(y3, cf_cycle_impact, [cf_cyc], 'retracting the direct exit leaves the route through the cycle',
     impact_set([rel(n('Cf CA'), precursor_of, n('Cf CD'))],
                rel(n('Cf ProdCyc'), supplies_precursor_of, n('Cf CD')), survives)).
test(y4, cf_cycle_impact, [cf_cyc], 'retracting the second exit leaves the direct route',
     impact_set([rel(n('Cf CB'), precursor_of, n('Cf CD'))],
                rel(n('Cf ProdCyc'), supplies_precursor_of, n('Cf CD')), survives)).
test(y5, cf_cycle_impact, [cf_cyc], 'retracting a cycle edge leaves the direct route',
     impact_set([rel(n('Cf CA'), precursor_of, n('Cf CB'))],
                rel(n('Cf ProdCyc'), supplies_precursor_of, n('Cf CD')), survives)).
test(y6, cf_cycle_impact, [cf_cyc], 'retracting both exits removes the conclusion',
     impact_set([rel(n('Cf CA'), precursor_of, n('Cf CD')), rel(n('Cf CB'), precursor_of, n('Cf CD'))],
                rel(n('Cf ProdCyc'), supplies_precursor_of, n('Cf CD')), lost)).

% ---- tests: a derived defeater ----------------------------------------------------------

test(df1, cf_defeat, [cf_defeat2], 'a defeater obtained through subrel still defeats',
     assess_is(rel(n('Cf DefD'), may_address, c(cf_symp5)), conflicted)).
test(df2, cf_defeat, [cf_defeat2], 'the derived-defeater fixture validates cleanly', vocab_clean).

% ---- tests: second reasoning pattern, knowledge only ------------------------------------

test(pt1, cf_pattern, [cf_pattern], 'a declared schema derives a new kind of conclusion',
     derives(rel(n('Cf CalcOnly'), cf_may_aggravate, c(fatigue)))).
test(pt2, cf_pattern, [cf_pattern], 'the new conclusion carries the schema ceiling',
     assess_is(rel(n('Cf CalcOnly'), cf_may_aggravate, c(fatigue)), conditional)).
test(pt3, cf_pattern, [cf_pattern], 'its proof rests on the curated Calcium-inhibits-Iron fact',
     uses_fact(rel(n('Cf CalcOnly'), cf_may_aggravate, c(fatigue)),
               rel(n('Calcium'), inhibits, n('Iron')))).
test(pt4, cf_pattern, [cf_pattern], 'a product with no inhibiting active is not flagged',
     cannot_derive(rel(n('Iron Supplement 65mg'), cf_may_aggravate, c(fatigue)))).
test(pt5, cf_pattern, [cf_pattern], 'the new conclusion defeats a claim to address the same symptom',
     assess_is(rel(n('Cf CalcInd'), may_address, c(fatigue)), conflicted)).
test(pt6, cf_pattern, [cf_pattern], 'existing conclusions are unchanged',
     addresses('Iron Supplement 65mg', fatigue, conditional)).
test(pt7, cf_pattern, [cf_pattern], 'the pattern validates cleanly', vocab_clean).
test(pt8, cf_pattern, [cf_pattern], 'retracting the inhibition removes the new conclusion',
     impact(rel(n('Calcium'), inhibits, n('Iron')),
            rel(n('Cf CalcOnly'), cf_may_aggravate, c(fatigue)))).

check(proof_statuses(T, Want)) :-
    conv(T, C), findall(S, ( solve(C, Pr), proof_status(Pr, S) ), L0), sort(L0, L), L == Want.
check(uses_fact(G, T)) :-
    conv(G, C), conv(T, F), solve(C, Pr), proof_triples(Pr, Ts), memberchk(F, Ts), !.

% ============ batch 4: independence of evidence (lineage) =================================

% IProd contains I2: two sources that share a lineage (textbook_a, review_b).
% I2 provides I4: two undeclared sources (each its own origin).
% I2 enhances I4: one source asserting twice with different detail.
% I4 provides I2: three undeclared sources, each of conditional strength.

fixture(cf_indep, Ts) :-
    cf_ents(['Cf I2', 'Cf I4'], Es),
    cf_prod('Cf IProd', [], P),
    Ts0 = [ source_lineage(textbook_a, cf_trial), source_lineage(review_b, cf_trial),
            bridge('Cf IProd', contains, 'Cf I2', [source(textbook_a)]),
            bridge('Cf IProd', contains, 'Cf I2', [source(review_b)]),
            bridge('Cf I2', provides, 'Cf I4', [source(independent_c)]),
            bridge('Cf I2', provides, 'Cf I4', [source(independent_d)]),
            bridge('Cf I2', enhances, 'Cf I4', [source(textbook_a), page(1)]),
            bridge('Cf I2', enhances, 'Cf I4', [source(textbook_a), page(2)]),
            bridge('Cf I4', provides, 'Cf I2', [source(src_x)]),
            bridge('Cf I4', provides, 'Cf I2', [source(src_y)]),
            bridge('Cf I4', provides, 'Cf I2', [source(src_z)]) ],
    append([Es, [P], Ts0], Ts).

% a conflicting declaration for textbook_a (already declared above): disabled whole
fixture(cf_conflict, [ source_lineage(textbook_a, cf_other) ]).

test(in1, cf_independence, [cf_indep], 'two sources of one lineage count as one origin',
     origins_of(rel(n('Cf IProd'), contains, n('Cf I2')), 1)).
test(in2, cf_independence, [cf_indep], 'one source asserting twice is one origin, two evidence entries',
     ( origins_of(rel(n('Cf I2'), enhances, n('Cf I4')), 1),
       ev_count(rel(n('Cf I2'), enhances, n('Cf I4')), 2) )).
test(in3, cf_independence, [cf_indep], 'undeclared sources are each their own origin',
     origins_of(rel(n('Cf I2'), provides, n('Cf I4')), 2)).
test(in4, cf_independence, [cf_indep], 'proof support is its least-corroborated leaf',
     support_of(rel(n('Cf IProd'), provides, n('Cf I4')), 1)).
test(in5, cf_independence, [cf_indep], 'corroboration is reported but does not raise a status',
     ( origins_of(rel(n('Cf I4'), provides, n('Cf I2')), 3),
       assess_is(rel(n('Cf I4'), provides, n('Cf I2')), conditional) )).
test(in6, cf_independence, [cf_indep, cf_conflict], 'conflicting lineage declarations are rejected',
     rejected_as(conflicting_lineage(textbook_a), lineage(textbook_a))).
test(in7, cf_independence, [cf_indep, cf_conflict], 'a rejected lineage declaration is disabled, not half-applied',
     origins_of(rel(n('Cf IProd'), contains, n('Cf I2')), 2)).

check(origins_of(T, N)) :-
    conv(T, rel(A, P, B)), rel(A, P, B, M), fact_independence(M, K), K == N.
check(support_of(G, N)) :-
    conv(G, C), independent_support(C, K), K == N.

% ============ batch 5: impact equivalence, assertion-level lineage, corroboration lift ====

% ---- reference implementation of impact analysis (the previous algorithm, kept as oracle) --

ref_impact_set(Triples, Lost) :-
    headline_conclusions(Cs),
    findall(C, ( member(C, Cs),
                 findall(Ts, ( solve(C, Pr), proof_triples(Pr, Ts) ), Alts),
                 Alts \== [],
                 forall(member(Ts, Alts), ( member(T, Triples), memberchk(T, Ts) )) ),
            Lost).

test(im1, cf_impact, [], 'grouped impact analysis equals the reference on every single retraction',
     impact_matches_reference(single)).
test(im2, cf_impact, [], 'grouped impact analysis equals the reference on retraction pairs',
     impact_matches_reference(pairs)).
test(im3, cf_impact, [cf_alt], 'grouped impact analysis equals the reference on alternative routes',
     impact_matches_reference(single)).

check(impact_matches_reference(Mode)) :-
    findall(rel(S, P, O), rel(S, P, O, _), Ts),
    Ts \== [],
    (   Mode == single -> findall([T], member(T, Ts), Sets)
    ;   findall([A, B], ( nth1(I, Ts, A), I1 is I + 1, nth1(I1, Ts, B) ), Sets)
    ),
    forall(member(Set, Sets),
           ( retract_impact_set(Set, L1), ref_impact_set(Set, L2), msort(L1, M1), msort(L2, M2), M1 == M2 )).

% ---- assertion-level lineage -------------------------------------------------------------

fixture(cf_alin, Ts) :-
    cf_ents(['Cf L1', 'Cf L2'], Es),
    Ts0 = [ source_lineage(textbook_a, cf_trial), source_lineage(review_b, cf_trial),
            bridge('Cf L1', provides, 'Cf L2', [source(src_p), lineage(g1)]),
            bridge('Cf L1', provides, 'Cf L2', [source(src_q), lineage(g1)]),
            bridge('Cf L1', enhances, 'Cf L2', [source(textbook_a), lineage(cf_own)]),
            bridge('Cf L1', enhances, 'Cf L2', [source(review_b)]),
            bridge('Cf L2', provides, 'Cf L1', [source(src_p), lineage(g1), lineage(g2)]),
            bridge('Cf L2', provides, 'Cf L1', [source(src_q), lineage(g1)]) ],
    append(Es, Ts0, Ts).

test(al1, cf_independence, [cf_alin], 'two sources sharing an assertion-level lineage are one origin',
     origins_of(rel(n('Cf L1'), provides, n('Cf L2')), 1)).
test(al2, cf_independence, [cf_alin], 'assertion-level lineage overrides the source-level declaration',
     origins_of(rel(n('Cf L1'), enhances, n('Cf L2')), 2)).
test(al3, cf_independence, [cf_alin], 'contradictory assertion-level lineage is reported at ingest',
     issue(conflicting_assertion_lineage(provides))).
test(al4, cf_independence, [cf_alin], 'a contradictory assertion falls back to its source label',
     origins_of(rel(n('Cf L2'), provides, n('Cf L1')), 2)).

% ---- corroboration lift ----------------------------------------------------------------------

fixture(cf_liftdata, Ts) :-
    cf_ents(['Cf LX', 'Cf LY'], Es),
    cf_prod('Cf LProd', [], P),
    Ts0 = [ source_lineage(textbook_a, cf_trial), source_lineage(review_b, cf_trial),
            bridge('Cf LProd', contains, 'Cf LX', [source(fixture)]),
            bridge('Cf LX', provides, 'Cf LY', [source(src_m)]),
            bridge('Cf LX', provides, 'Cf LY', [source(src_n)]),
            bridge('Cf LY', provides, 'Cf LX', [source(src_m)]),
            bridge('Cf LY', provides, 'Cf LX', [source(src_n)]),
            bridge('Cf LY', provides, 'Cf LX', [source(src_o)]),
            bridge('Cf LX', enhances, 'Cf LY', [source(textbook_a)]),
            bridge('Cf LX', enhances, 'Cf LY', [source(review_b)]),
            bridge('Cf LY', enhances, 'Cf LX', [source(src_m), page(1)]),
            bridge('Cf LY', enhances, 'Cf LX', [source(src_m), page(2)]) ],
    append([Es, [P], Ts0], Ts).

fixture(cf_lift2, [ corroboration_lift(2, conditional, supported) ]).
fixture(cf_lift3, [ corroboration_lift(3, conditional, supported) ]).
fixture(cf_bad_lift, [
    corroboration_lift(1, conditional, supported),
    corroboration_lift(2, supported, conditional),
    corroboration_lift(2, bogus, supported),
    corroboration_lift(2, unknown, supported) ]).

test(lf0, cf_lift, [cf_liftdata], 'with no rule declared, corroboration never raises a status',
     assess_is(rel(n('Cf LX'), provides, n('Cf LY')), conditional)).
test(lf1, cf_lift, [cf_liftdata, cf_lift2], 'two independent origins lift a conditional fact',
     assess_is(rel(n('Cf LX'), provides, n('Cf LY')), supported)).
test(lf2, cf_lift, [cf_liftdata, cf_lift2], 'sources of one lineage do not lift',
     assess_is(rel(n('Cf LX'), enhances, n('Cf LY')), conditional)).
test(lf3, cf_lift, [cf_liftdata, cf_lift2], 'one source asserting twice does not lift',
     assess_is(rel(n('Cf LY'), enhances, n('Cf LX')), conditional)).
test(lf4, cf_lift, [cf_liftdata, cf_lift2], 'a proof through a lifted fact inherits the lift',
     assess_is(rel(n('Cf LProd'), provides, n('Cf LY')), supported)).
test(lf5, cf_lift, [cf_liftdata], 'the same proof is conditional without the rule',
     assess_is(rel(n('Cf LProd'), provides, n('Cf LY')), conditional)).
test(lf6, cf_lift, [cf_liftdata, cf_lift3], 'a threshold of three does not fire on two origins',
     assess_is(rel(n('Cf LX'), provides, n('Cf LY')), conditional)).
test(lf7, cf_lift, [cf_liftdata, cf_lift3], 'a threshold of three fires on three origins',
     assess_is(rel(n('Cf LY'), provides, n('Cf LX')), supported)).
test(lf8, cf_lift, [cf_lift2], 'a valid lift rule validates cleanly', vocab_clean).
test(lf9, cf_lift, [cf_bad_lift], 'threshold below two is rejected',
     rejected_as(bad_lift_threshold(1), lift(1, conditional, supported))).
test(lf10, cf_lift, [cf_bad_lift], 'a lift that is not upward is rejected',
     rejected_as(lift_not_upward, lift(2, supported, conditional))).
test(lf11, cf_lift, [cf_bad_lift], 'a lift over an unknown status is rejected',
     rejected_as(bad_lift_status(bogus), lift(2, bogus, supported))).
test(lf12, cf_lift, [cf_bad_lift], 'a lift from a non-positive status is rejected',
     rejected_as(lift_from_non_positive(unknown), lift(2, unknown, supported))).
test(lf13, cf_lift, [cf_liftdata, cf_bad_lift], 'rejected rules are disabled, not applied',
     ( assess_is(rel(n('Cf LX'), provides, n('Cf LY')), conditional),
       assess_is(rel(n('Cf LY'), enhances, n('Cf LX')), conditional) )).

% ============ batch 6: bound certificate, structured assessment, recommendation tiers =====

fixture(cf_noproof, [ P ]) :- cf_prod('Cf NP', [not_indicated_for([cf_symp6])], P).

% ---- certificate ----------------------------------------------------------------------------

test(ce1, cf_certificate, [cf_chain], 'a search cut by the bound reports bound_reached',
     cert_is(rel(n('Cf Chain1'), precursor_of, n('Cf Chain9')), bound_reached)).
test(ce2, cf_certificate, [], 'a shallow failing search is certified exhaustive',
     cert_is(rel(n('Iron Supplement 65mg'), may_address, c(headache)), exhaustive)).
test(ce3, cf_certificate, [cf_axioms, cf_ring], 'a search along a long cycle reports bound_reached',
     cert_is(rel(n('Cf Ring1'), precursor_of, n('Cf Alpha')), bound_reached)).
test(ce4, cf_certificate, [cf_chain], 'a shallow positive search is certified exhaustive',
     ( derives(rel(n('Cf Chain1'), precursor_of, n('Cf Chain3'))),
       cert_is(rel(n('Cf Chain1'), precursor_of, n('Cf Chain3')), exhaustive) )).
test(ce5, cf_certificate, [], 'exhaustive implies a deeper bound finds the same proofs (shipped knowledge)',
     cert_sound).
test(ce6, cf_certificate, [cf_axioms, cf_chain, cf_ring, cf_newrel], 'the same holds over the axiom, chain, ring and new-relation fixtures',
     cert_sound).
test(ce7, cf_certificate, [cf_chain], 'a deeper explicit bound finds the cut conclusion and certifies it',
     cert_at(rel(n('Cf Chain1'), precursor_of, n('Cf Chain9')), 12, exhaustive)).

% ---- structured assessment -------------------------------------------------------------------

test(as1, cf_assessment, [cf_defeat], 'supported strength, no conflict',
     assessment_is(rel(n('Cf DefB'), may_address, c(cf_symp)),
                   [status-supported, proof_strength-supported, conflict-none])).
test(as2, cf_assessment, [cf_defeat], 'supported strength, contested by an equal defeater',
     assessment_is(rel(n('Cf DefA'), may_address, c(cf_symp)),
                   [status-conflicted, proof_strength-supported, conflict-contested])).
test(as3, cf_assessment, [cf_defeat], 'conditional strength, defeated by a stronger defeater',
     assessment_is(rel(n('Cf DefC'), may_address, c(cf_symp2)),
                   [status-refuted, proof_strength-conditional, conflict-defeated])).
test(as4, cf_assessment, [], 'no proof: strength none, no conflict, certified exhaustive',
     assessment_is(rel(n('Iron Supplement 65mg'), may_address, c(headache)),
                   [status-unknown, proof_strength-none, conflict-none,
                    independent_support-0, completeness-exhaustive])).
test(as5, cf_assessment, [cf_indep], 'the record carries the independent support of the best proof',
     assessment_is(rel(n('Cf IProd'), provides, n('Cf I4')),
                   [proof_strength-conditional, independent_support-1])).
test(as6, cf_assessment, [cf_noproof], 'a defeater with nothing to defeat is listed but not a refutation',
     ( assessment_is(rel(n('Cf NP'), may_address, c(cf_symp6)),
                     [status-unknown, proof_strength-none, conflict-none]),
       has_defeaters(rel(n('Cf NP'), may_address, c(cf_symp6))) )).
test(as7, cf_assessment, [], 'assess/3 is the status and defeaters of the record, on every shipped headline conclusion',
     assess_projection).
test(as8, cf_assessment, [], 'proof strength is only ever supported, conditional or none',
     proof_strength_domain).

% ---- recommendation tiers ----------------------------------------------------------------------

test(rc1, cf_recommend, [cf_defeat], 'a product without a defeater is in the supported tier',
     tier_has(supported, [cf_symp], 'Cf DefB')).
test(rc2, cf_recommend, [cf_defeat], 'a contested product needs review and is not supported',
     ( tier_has(needs_review, [cf_symp], 'Cf DefA'), tier_lacks(supported, [cf_symp], 'Cf DefA') )).
test(rc3, cf_recommend, [cf_defeat], 'a defeated product is withheld, not supported or reviewable',
     ( tier_has(withheld, [cf_symp2], 'Cf DefC'),
       tier_lacks(supported, [cf_symp2], 'Cf DefC'), tier_lacks(needs_review, [cf_symp2], 'Cf DefC') )).
test(rc4, cf_recommend, [cf_defeat], 'a withheld product carries the defeating relation and its proof',
     withheld_has_defeater([cf_symp2], 'Cf DefC', not_indicated_for)).
test(rc5, cf_recommend, [], 'on shipped knowledge a labelled product is supported and a deficiency-only one needs review',
     ( tier_has(supported, [fatigue], 'Vitamin B12 1000mcg Sublingual'),
       tier_has(needs_review, [fatigue], 'Iron Supplement 65mg') )).
test(rc6, cf_recommend, [], 'supported plus needs_review equals recommend/2, and each candidate row is in exactly one tier',
     tiers_cover_recommend2([fatigue, weakness, memory_problems, anemia, shortness_of_breath, pale_skin, depression])).
test(rc7, cf_recommend, [cf_defeat], 'the same holds with defeaters present',
     tiers_cover_recommend2([cf_symp, cf_symp2])).

check(cert_is(G, Want)) :- conv(G, C), solve_certified(C, _, Got), Got == Want.
check(cert_at(G, D, Want)) :-
    conv(G, C), solve_certified(C, D, Ps, Got), Got == Want, Ps \== [].
check(cert_sound) :-
    findall(P, rel_pred(P, _), Ps),
    findall(P-C0, ( member(P, Ps), solve_certified(rel(_, P, _), 6, Pr6, C0),
                    (   C0 == exhaustive
                    ->  findall(X, solve(rel(_, P, _), 12, X), Pr12),
                        msort(Pr6, A), msort(Pr12, B), A == B
                    ;   true ) ),
            Rs),
    length(Ps, N), length(Rs, N),
    aggregate_all(count, member(_-exhaustive, Rs), E), E >= 1.
check(assessment_is(G, Pairs)) :-
    conv(G, C), assessment(C, A),
    forall(member(K-V, Pairs), ( T =.. [K, V], memberchk(T, A) )).
check(has_defeaters(G)) :- conv(G, C), assessment(C, A), memberchk(defeaters(Ds), A), Ds \== [].
check(assess_projection) :-
    headline_conclusions(Cs), Cs \== [],
    forall(member(C, Cs),
           ( assess(C, S, D), assessment(C, A),
             memberchk(status(S), A), memberchk(defeaters(D), A) )).
check(proof_strength_domain) :-
    headline_conclusions(Cs), Cs \== [],
    forall(member(C, Cs), ( assessment(C, A), memberchk(proof_strength(PS), A),
                            memberchk(PS, [supported, conditional, none]) )).
check(tier_has(Tier, Syms, PN)) :-
    eid(PN, P), recommend_tiers(Syms, T), tier_pick(Tier, T, L), memberchk(rec(P, _, _), L).
check(tier_lacks(Tier, Syms, PN)) :-
    eid(PN, P), recommend_tiers(Syms, T), tier_pick(Tier, T, L), \+ memberchk(rec(P, _, _), L).
check(withheld_has_defeater(Syms, PN, D)) :-
    eid(PN, P), recommend_tiers(Syms, tiers(_, _, Wh)),
    member(rec(P, _, Rows), Wh), member(row(_, _, Why), Rows),
    member(defeater(D, _, Pr), Why), proof_conclusion(Pr, rel(P, D, _)).
check(tiers_cover_recommend2(Syms)) :-
    recommend_tiers(Syms, tiers(Sup, Rev, Wh)), recommend(Syms, Recs),
    findall(P, ( member(L, [Sup, Rev]), member(rec(P, _, _), L) ), Ps0), sort(Ps0, A),
    findall(P, member(rec(P, _, _), Recs), Qs0), sort(Qs0, B), A == B,
    findall(P-S, ( member(L, [Sup, Rev, Wh]), member(rec(P, _, Rows), L), member(row(S, _, _), Rows) ), All),
    msort(All, M), sort(All, U), M == U.

tier_pick(supported, tiers(L, _, _), L).
tier_pick(needs_review, tiers(_, L, _), L).
tier_pick(withheld, tiers(_, _, L), L).

% ============ batch 7: semantic contract (docs/CONTRACT.md), properties over all conclusions ==

test(k1, contract, [cf_defeat, cf_noproof], 'C1/C2: no proof implies unknown; refuted implies a declared defeater',
     no_proof_implies_unknown).
test(k2, contract, [], 'C3: no proof is stronger than its weakest step (shipped knowledge)', status_ceiling).
test(k2b, contract, [cf_axioms, cf_alt, cf_defeat, cf_mixed, cf_pattern, cf_liftdata], 'C3: the same over every relation and the fixtures, including mixed-strength steps', status_ceiling).
test(k3, contract, [], 'C4: adding a defeater never improves a may_address conclusion', defeat_never_improves).
test(k4, contract, [], 'C5: removing any one fact never creates a headline conclusion', retraction_monotone).
test(k5, contract, [], 'C6: ingestion is deterministic', ingest_idempotent).
test(k6, contract, [cf_indep, cf_alin], 'C7: independence is between one and the number of evidence entries', independence_bounded).
test(k7, contract, [], 'C10: may_address is documented as a candidate relation, never a diagnosis',
     doc_says(may_address, "never a diagnosis")).

check(no_proof_implies_unknown) :-
    findall(P, product(P), Ps0), sort(Ps0, Ps), Ps \== [],
    findall(Sym, ( rel(_, K, concept(Sym), _),
                   memberchk(K, [indicated_for, deficiency_manifests_as, not_indicated_for]) ), Ss0),
    sort(Ss0, Ss),
    forall(( member(P, Ps), member(Sym, Ss) ),
           ( Goal = rel(P, may_address, concept(Sym)),
             assess(Goal, St, Ds),
             (   \+ solve(Goal, _) -> St == unknown ; true ),
             (   St == refuted -> Ds \== [] ; true ) )).

check(status_ceiling) :-
    findall(P, rel_pred(P, _), Ps), Ps \== [],
    forall(( member(P, Ps), solve(rel(_, P, _), Pr) ),
           ( proof_status(Pr, S), status_rank(S, R), cap_list(Pr, L), min_list(L, Cap), R =< Cap )).

cap_list(fact(_, M), [R]) :-
    findall(X, ( member(ev(_, St, _), M), status_rank(St, X) ), Xs),
    (   Xs == [] -> R = 3 ; max_list(Xs, R) ).
cap_list(composed(_, _, Ps), L) :- maplist(cap_list, Ps, Ls), append(Ls, L).
cap_list(axiom(_, _, Ps), L)    :- maplist(cap_list, Ps, Ls), append(Ls, L).
cap_list(derived(_, St, _, Ps), [R|L]) :- status_rank(St, R), maplist(cap_list, Ps, Ls), append(Ls, L).

check(defeat_never_improves) :-
    headline_conclusions(Cs0),
    findall(C, ( member(C, Cs0), C = rel(_, may_address, concept(_)) ), Cands), Cands \== [],
    findall(C-R0, ( member(C, Cands), assess(C, S0, _), status_rank(S0, R0) ), Base),
    forall(member(rel(P, may_address, concept(Sym))-R0, Base),
           setup_call_cleanup(
               ( entity_name(P, Name),
                 B = bridge(Name, not_indicated_for, concept(Sym), [source(fixture)]),
                 assertz(B), ingest_all ),
               ( assess(rel(P, may_address, concept(Sym)), S1, _), status_rank(S1, R1), R1 =< R0 ),
               ( retract(B), ingest_all ))).

check(retraction_monotone) :-
    headline_conclusions(Base), Base \== [],
    findall(rel(S, P, O)-M, rel(S, P, O, M), Facts), Facts \== [],
    forall(member(rel(S, P, O)-M, Facts),
           setup_call_cleanup(
               retract(rel(S, P, O, M)),
               ( headline_conclusions(After), forall(member(C, After), memberchk(C, Base)) ),
               assertz(rel(S, P, O, M)))).

check(ingest_idempotent) :-
    ingest_snapshot(A), ingest_all, ingest_snapshot(B), A == B.
ingest_snapshot(R-I) :-
    findall(rel(S, P, O, M), rel(S, P, O, M), R0), msort(R0, R),
    findall(K-C, ingest_issue(K, C), I0), msort(I0, I).

check(independence_bounded) :-
    forall(rel(_, _, _, M),
           ( M \== [], fact_independence(M, N), length(M, L), N >= 1, N =< L )),
    headline_conclusions(Cs), Cs \== [],
    forall(( member(C, Cs), solve(C, Pr) ),
           ( proof_independence(Pr, N), N >= 1, proof_leaves(Pr, Ls),
             forall(member(_-M, Ls), ( fact_independence(M, K), N =< K )) )).

check(doc_says(P, Text)) :-
    rel_pred(P, O), memberchk(doc(D), O), sub_string(D, _, _, _, Text).
