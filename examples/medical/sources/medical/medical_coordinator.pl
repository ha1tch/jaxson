% medical_coordinator.pl - TOSID Medical Knowledge Coordinator
% Loads four separate knowledge bases and performs cross-domain inferences

% CRITICAL: Declare tosid as multifile BEFORE any files are loaded
:- multifile(tosid/5).
:- dynamic(tosid/5).

% FORCE: Declare all predicates as dynamic to prevent caching issues
:- dynamic(supplement_addresses_function/2).
:- dynamic(recommend_treatment_enhanced/2).
:- dynamic(consolidate_all_recommendations/2).

% Load all knowledge bases
:- consult('biochemical.pl').
:- consult('chemistry.pl'). 
:- consult('pharma.pl').
:- consult('medicine.pl').

% =================================================================
% ESSENTIAL HELPER PREDICATES (DEFINED EARLY)
% =================================================================

% Helper for functional matching - DEFINED FIRST
supplement_addresses_function(Function, Indications) :-
    (   Function = oxygen_transport, member(iron_deficiency, Indications)
    ;   Function = calcium_regulation, member(bone_health, Indications)  
    ;   Function = dna_synthesis, member(b12_deficiency, Indications)
    ).

% Consolidate all recommendations including patterns
consolidate_all_recommendations(Recs, Consolidated) :-
    % Convert pattern_rec to rec format for consistency
    findall(rec(Symptom, Treatment, Mechanism, Type, Confidence),
        (member(R, Recs),
         normalize_recommendation_format(R, rec(Symptom, Treatment, Mechanism, Type, Confidence))),
        NormalizedRecs),
    consolidate_recommendations(NormalizedRecs, Consolidated).

% Normalize different recommendation formats  
normalize_recommendation_format(rec(S, T, M, Tp, C), rec(S, T, M, Tp, C)).
normalize_recommendation_format(pattern_rec(Pattern, T, M, Tp, C), rec(Pattern, T, M, Tp, C)).

% =================================================================
% TOSID-Based Cross-Domain Inference Rules
% =================================================================

% Identify entities by TOSID classification
is_biochemical_entity(Entity) :-
    tosid(_, _, _, Entity, Props),
    member(domain(biochemical), Props).

is_chemical_entity(Entity) :-
    tosid(_, _, _, Entity, Props),
    member(domain(chemistry), Props).

is_pharmaceutical_product(Entity) :-
    tosid(_, _, _, Entity, Props),
    member(domain(pharma), Props).

is_medical_condition(Entity) :-
    tosid(_, _, _, Entity, Props),
    member(domain(medicine), Props).

% =================================================================
% SYMPTOM-TO-BIOCHEMICAL DEFICIENCY MAPPING
% =================================================================

% Map symptoms to potential biochemical deficiencies
symptom_suggests_deficiency(Symptom, DeficiencyEntity, Confidence) :-
    % Find biochemical entities with deficiency symptoms
    tosid(_, _, _, DeficiencyEntity, Props),
    member(domain(biochemical), Props),
    member(deficiency_symptoms(Symptoms), Props),
    member(Symptom, Symptoms),
    Confidence = high.

% =================================================================
% DEFICIENCY-TO-PHARMACEUTICAL MAPPING  
% =================================================================

% Map biochemical deficiencies to available pharmaceutical products
deficiency_treated_by_supplement(DeficiencyEntity, PharmProduct, Mechanism) :-
    % Get the deficient biochemical
    tosid(_, _, _, DeficiencyEntity, DefProps),
    member(domain(biochemical), DefProps),
    
    % Find pharmaceutical that contains the needed compound
    tosid(_, _, _, PharmProduct, PharmProps),
    member(domain(pharma), PharmProps),
    member(indication(Indications), PharmProps),
    
    % Match based on deficiency type
    (   (member(contains_element(Element), DefProps),
         member(active_ingredient(Ingredient), PharmProps),
         element_in_compound(Element, Ingredient),
         Mechanism = element_replacement)
    ;   (DeficiencyEntity = 'Vitamin D3',
         member(active_ingredient('cholecalciferol'), PharmProps),
         Mechanism = vitamin_replacement)
    ;   (DeficiencyEntity = 'Vitamin B12',
         member(active_ingredient('cyanocobalamin'), PharmProps),
         Mechanism = vitamin_replacement)
    ;   (member(function(Function), DefProps),
         supplement_addresses_function(Function, Indications),
         Mechanism = functional_support)
    ).

element_in_compound(iron, 'ferrous_sulfate').
element_in_compound(magnesium, 'magnesium_oxide').

% Helper for functional matching
% =================================================================
% CLINICAL PATTERN RECOGNITION
% =================================================================

% Recognize common clinical patterns and suggest comprehensive treatment
recognize_clinical_patterns(Symptoms, PatternTreatments) :-
    findall(pattern_rec(Pattern, Treatment, Mechanism, Type, Confidence),
        (clinical_pattern(Pattern, PatternSymptoms, Treatment, Mechanism),
         symptoms_match_pattern(Symptoms, PatternSymptoms),
         Type = pattern_based,
         Confidence = high),
        PatternTreatments),
    !.  % Cut to prevent backtracking issues
recognize_clinical_patterns(_, []).  % Fallback: return empty list if pattern matching fails

% Define clinical patterns
clinical_pattern(iron_deficiency_anemia, 
    [fatigue, shortness_of_breath, pale_skin],
    'Iron + Vitamin C Combination',
    enhanced_iron_absorption).

clinical_pattern(magnesium_deficiency,
    [muscle_cramps, irregular_heartbeat],
    'Magnesium Supplement 400mg',
    electrolyte_replacement).

clinical_pattern(vitamin_b12_deficiency,
    [fatigue, memory_problems, weakness],
    'Vitamin B12 1000mcg Sublingual',
    neurological_support).

clinical_pattern(acute_pain_fever,
    [headache, fever],
    'Paracetamol 500mg Tablet',
    safe_symptomatic_relief).

% Check if symptoms match a clinical pattern (at least 70% overlap) - with safety
symptoms_match_pattern(Symptoms, PatternSymptoms) :-
    is_list(Symptoms),
    is_list(PatternSymptoms),
    intersection(Symptoms, PatternSymptoms, Common),
    length(Common, CommonCount),
    length(PatternSymptoms, PatternCount),
    PatternCount > 0,  % Prevent division by zero
    CommonCount >= PatternCount * 0.7.  % 70% match threshold

% =================================================================
% ENHANCED SYMPTOM MAPPING WITH MISSING CONNECTIONS
% =================================================================

% Enhanced symptom mapping to catch obvious missing connections
symptom_relieved_by_drug(Symptom, PharmProduct, Mechanism) :-
    % Find pharmaceuticals that treat the symptom
    tosid(_, _, _, PharmProduct, PharmProps),
    member(domain(pharma), PharmProps),
    member(indication(Indications), PharmProps),
    (   member(Symptom, Indications)
    ;   symptom_matches_indication(Symptom, Indications)  % Enhanced matching
    ),
    
    % Get the mechanism from chemistry knowledge or pharma properties
    member(active_ingredient(ActiveIngredient), PharmProps),
    (   tosid(_, _, _, ActiveIngredient, ChemProps),
        member(domain(chemistry), ChemProps),
        member(mechanism(Mechanism), ChemProps)
    ;   tosid(_, _, _, _, ChemProps),
        member(domain(chemistry), ChemProps),
        member(chemical_name(ActiveIngredient), ChemProps),
        member(mechanism(Mechanism), ChemProps)
    ;   Mechanism = symptomatic_relief  % fallback
    ).

% Enhanced symptom-to-indication matching
symptom_matches_indication(headache, Indications) :- member(pain, Indications).
symptom_matches_indication(head_pain, Indications) :- member(pain, Indications).
symptom_matches_indication(muscle_pain, Indications) :- member(pain, Indications).
symptom_matches_indication(body_aches, Indications) :- member(pain, Indications).
symptom_matches_indication(irregular_heartbeat, Indications) :- member(magnesium_deficiency, Indications).
symptom_matches_indication(sleep_disorders, Indications) :- member(mood_support, Indications).
symptom_matches_indication(depression, Indications) :- member(mood_support, Indications).

% =================================================================
% SIMPLIFIED BUT ROBUST TREATMENT RECOMMENDATION ENGINE
% =================================================================

% Main predicate: recommend treatment for a list of symptoms (SIMPLIFIED)
recommend_treatment_simple(Symptoms, CleanRecommendations) :-
    findall(rec(Symptom, Treatment, Mechanism, Type, Confidence),
        (member(Symptom, Symptoms),
         find_treatment_for_symptom(Symptom, Treatment, Mechanism, Type, Confidence)),
        AllRecs),
    remove_exact_duplicates(AllRecs, CleanRecommendations).

% Find treatment for a specific symptom
find_treatment_for_symptom(Symptom, Treatment, Mechanism, Type, Confidence) :-
    % Try deficiency-based treatment first
    (   symptom_suggests_deficiency(Symptom, Deficiency, DefConf),
        deficiency_treated_by_supplement(Deficiency, Treatment, Mechanism),
        Type = deficiency_correction,
        Confidence = DefConf
    ;   % Try direct symptomatic treatment
        symptom_relieved_by_drug(Symptom, Treatment, Mechanism),
        Type = symptomatic_relief,
        Confidence = moderate
    ).

% Simple duplicate removal
remove_exact_duplicates([], []).
remove_exact_duplicates([H|T], [H|Result]) :-
    \+ member(H, T),
    remove_exact_duplicates(T, Result).
remove_exact_duplicates([H|T], Result) :-
    member(H, T),
    remove_exact_duplicates(T, Result).

% =================================================================
% ENHANCED DRUG INTERACTION CHECKING
% =================================================================

% Check for interactions between recommended drugs
check_interactions(Drugs, Interactions) :-
    findall(interaction(Drug1, Drug2, Type, Significance),
        (member(Drug1, Drugs),
         member(Drug2, Drugs),
         Drug1 \= Drug2,
         drug_interaction(Drug1, Drug2, Type, Significance)),
        AllInteractions),
    remove_duplicate_interactions(AllInteractions, Interactions).

% Remove duplicate interactions (A-B same as B-A)
remove_duplicate_interactions([], []).
remove_duplicate_interactions([interaction(D1, D2, T, S)|Rest], [interaction(D1, D2, T, S)|CleanRest]) :-
    \+ member(interaction(D2, D1, T, S), Rest),
    remove_duplicate_interactions(Rest, CleanRest).
remove_duplicate_interactions([interaction(D1, D2, T, S)|Rest], CleanRest) :-
    member(interaction(D2, D1, T, S), Rest),
    remove_duplicate_interactions(Rest, CleanRest).

% Enhanced drug interaction detection
drug_interaction(Drug1, Drug2, Type, Significance) :-
    tosid(_, _, _, Drug1, Props1),
    tosid(_, _, _, Drug2, Props2),
    member(active_ingredient(Ingredient1), Props1),
    member(active_ingredient(Ingredient2), Props2),
    % Check for known chemical interactions
    (   interaction_between_ingredients(Ingredient1, Ingredient2, Type, Significance)
    ;   interaction_between_ingredients(Ingredient2, Ingredient1, Type, Significance)
    ).

% Known ingredient interactions
interaction_between_ingredients('ferrous_sulfate', 'ascorbic_acid', synergistic, high).
interaction_between_ingredients(calcium, 'ferrous_sulfate', antagonistic, moderate).
interaction_between_ingredients('acetylsalicylic_acid', ibuprofen, antagonistic, moderate).
interaction_between_ingredients('acetylsalicylic_acid', 'acetaminophen', neutral, low).

% =================================================================
% COST-EFFECTIVENESS ANALYSIS
% =================================================================

% Calculate treatment cost
calculate_treatment_cost(Treatments, TotalCost) :-
    findall(Cost,
        (member(rec(_, Treatment, _, _, _), Treatments),
         tosid(_, _, _, Treatment, Props),
         member(cost_per_tablet(Cost), Props)),
        Costs),
    sum_list(Costs, TotalCost).

% =================================================================
% PATIENT CASE EXAMPLES AND QUERIES
% =================================================================

% Example patient cases
patient_case(case1, [fatigue, shortness_of_breath, pale_skin]).
patient_case(case2, [headache, fever]).
patient_case(case3, [depression, sleep_disorders, fatigue]).
patient_case(case4, [muscle_cramps, irregular_heartbeat]).
patient_case(case5, [fatigue, memory_problems, weakness]).

% =================================================================
% REPORTING AND ANALYSIS FUNCTIONS
% =================================================================

% Generate treatment report for a patient case (SIMPLIFIED AND ROBUST)
generate_treatment_report(CaseId) :-
    patient_case(CaseId, Symptoms),
    format('~n=== TREATMENT REPORT FOR ~w ===~n', [CaseId]),
    format('Symptoms: ~w~n~n', [Symptoms]),
    
    % Use simplified recommendation system
    recommend_treatment_simple(Symptoms, Recommendations),
    
    (   Recommendations = [] ->
        format('RECOMMENDED TREATMENTS: None found~n~n')
    ;   (   format('RECOMMENDED TREATMENTS:~n'),
            forall(member(rec(Symptom, Treatment, Mechanism, Type, Confidence), Recommendations),
                (format('- ~w~n', [Treatment]),
                 format('  For: ~w~n', [Symptom]),
                 format('  Mechanism: ~w~n', [Mechanism]),
                 format('  Type: ~w | Confidence: ~w~n~n', [Type, Confidence]))))
    ),
    
    % Get unique treatments for interaction checking
    findall(Treatment, member(rec(_, Treatment, _, _, _), Recommendations), Treatments),
    list_to_set(Treatments, UniqueTreatments),
    
    % Check interactions
    check_interactions(UniqueTreatments, Interactions),
    (   Interactions = [] ->
        format('DRUG INTERACTIONS: None detected~n~n')
    ;   (   format('DRUG INTERACTIONS:~n'),
            forall(member(interaction(D1, D2, Type, Sig), Interactions),
                format('- ~w + ~w: ~w (~w significance)~n', [D1, D2, Type, Sig])),
            nl)
    ),
    
    % Calculate cost
    calculate_treatment_cost(Recommendations, Cost),
    format('ESTIMATED DAILY COST: $~2f~n', [Cost]).

% Cross-domain entity analysis
analyze_cross_domain_connections :-
    format('~n=== CROSS-DOMAIN ENTITY ANALYSIS ===~n~n'),
    
    format('BIOCHEMICAL → PHARMACEUTICAL CONNECTIONS:~n'),
    forall(deficiency_treated_by_supplement(Biochemical, Pharma, Mechanism),
        format('- ~w → ~w (~w)~n', [Biochemical, Pharma, Mechanism])),
    nl,
    
    format('SYMPTOM → DIRECT TREATMENT CONNECTIONS:~n'),
    forall(symptom_relieved_by_drug(Symptom, Drug, Mechanism),
        format('- ~w → ~w (~w)~n', [Symptom, Drug, Mechanism])),
    nl.

% =================================================================
% MAIN EXECUTION AND EXAMPLES
% =================================================================

run_medical_analysis :-
    format('=== TOSID MEDICAL KNOWLEDGE SYSTEM ===~n'),
    format('Integrating: Biochemical + Chemistry + Pharma + Medicine~n~n'),
    
    % Debug: Show all loaded entities by domain
    format('DEBUG: Checking loaded entities...~n'),
    findall(E-D, (tosid(_, _, _, E, P), member(domain(D), P)), AllEntities),
    length(AllEntities, TotalCount),
    format('Total entities loaded: ~w~n', [TotalCount]),
    forall(member(Entity-Domain, AllEntities), 
           format('  ~w (~w)~n', [Entity, Domain])),
    nl,
    
    % Show knowledge base statistics
    findall(E, (tosid(_, _, _, E, P), member(domain(biochemical), P)), BioEntities),
    findall(E, (tosid(_, _, _, E, P), member(domain(chemistry), P)), ChemEntities),
    findall(E, (tosid(_, _, _, E, P), member(domain(pharma), P)), PharmaEntities),
    findall(E, (tosid(_, _, _, E, P), member(domain(medicine), P)), MedEntities),
    
    length(BioEntities, BioCount),
    length(ChemEntities, ChemCount),
    length(PharmaEntities, PharmaCount),
    length(MedEntities, MedCount),
    
    format('Knowledge Base Statistics:~n'),
    format('- Biochemical entities: ~w~n', [BioCount]),
    format('- Chemistry entities: ~w~n', [ChemCount]),
    format('- Pharmaceutical products: ~w~n', [PharmaCount]),
    format('- Medical conditions/symptoms: ~w~n', [MedCount]),
    nl,
    
    % Only proceed if we have entities from all domains
    (   BioCount > 0, ChemCount > 0, PharmaCount > 0, MedCount > 0 ->
        (   % Analyze cross-domain connections
            analyze_cross_domain_connections,
            
            % Generate reports for all patient cases using simplified system
            format('~n=== PATIENT CASE ANALYSIS ===~n'),
            forall(patient_case(CaseId, _), 
                (catch(generate_treatment_report(CaseId), 
                       Error, 
                       (format('Error in case ~w: ~w~n', [CaseId, Error]))))
            ),
            format('~n=== ANALYSIS COMPLETE ===~n')
        )
    ;   format('ERROR: Not all knowledge bases loaded properly!~n')
    ).

% Script entry point - FIXED to run only once
:- initialization(main).

main :-
    run_medical_analysis,
    halt.