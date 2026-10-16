% chemistry.pl - General Chemistry Knowledge Base
% Chemical compounds, elements, and reactions

% Declare tosid as multifile to allow definitions across multiple files
:- multifile(tosid/5).
:- dynamic(tosid/5).

% tosid(TaxonomyCode, Netmask, Identifier, EntityName, Properties).

% Chemical Elements (Natural Material - Microscopic Scale)
tosid('00', 'F1', 'IRN-ELE-FE0:ATM-26P-M01', 'Iron', [
    type(element),
    domain(chemistry),
    atomic_number(26),
    symbol('Fe'),
    biological_role(oxygen_transport),
    deficiency_condition(anemia),
    toxicity_symptoms([nausea, liver_damage, organ_failure]),
    absorption_enhancers([vitamin_c, meat]),
    absorption_inhibitors([calcium, tea, coffee])
]).

tosid('00', 'F1', 'ZIN-ELE-ZN0:ATM-30P-M02', 'Zinc', [
    type(element),
    domain(chemistry),
    atomic_number(30),
    symbol('Zn'),
    biological_role([enzyme_cofactor, immune_function]),
    deficiency_symptoms([immune_dysfunction, wound_healing_impaired, hair_loss]),
    toxicity_symptoms([nausea, copper_deficiency]),
    normal_range_mcg_dl(70, 120)
]).

tosid('00', 'F1', 'MAG-ELE-MG0:ATM-12P-M03', 'Magnesium', [
    type(element),
    domain(chemistry),
    atomic_number(12),
    symbol('Mg'),
    biological_role([enzyme_cofactor, muscle_function, nerve_function]),
    deficiency_symptoms([muscle_cramps, irregular_heartbeat, seizures]),
    normal_range_mg_dl(1.7, 2.2),
    food_sources([nuts, seeds, leafy_greens])
]).

tosid('00', 'F1', 'CAL-ELE-CA0:ATM-20P-M04', 'Calcium', [
    type(element),
    domain(chemistry),
    atomic_number(20),
    symbol('Ca'),
    biological_role([bone_health, muscle_contraction, nerve_transmission]),
    deficiency_symptoms([osteoporosis, muscle_spasms, numbness]),
    normal_range_mg_dl(8.5, 10.5),
    absorption_cofactor(vitamin_d)
]).

% Chemical Compounds (Artificial Material - Molecular Scale)
tosid('10', 'F2', 'ASP-IRI-CMP:C9H-8O4-A01', 'Aspirin', [
    type(compound),
    domain(chemistry),
    molecular_formula('C9H8O4'),
    chemical_name('acetylsalicylic_acid'),
    mechanism(cox_inhibition),
    effects([analgesic, anti_inflammatory, antipyretic, antiplatelet]),
    side_effects([gastric_irritation, bleeding_risk, tinnitus]),
    contraindications([bleeding_disorders, children_with_viral_illness])
]).

tosid('10', 'F2', 'IBU-PRO-CMP:C13-H18-O2I', 'Ibuprofen', [
    type(compound),
    domain(chemistry),
    molecular_formula('C13H18O2'),
    chemical_class(nsaid),
    mechanism(cox_inhibition),
    effects([analgesic, anti_inflammatory, antipyretic]),
    side_effects([gastric_irritation, kidney_stress, cardiovascular_risk]),
    contraindications([kidney_disease, heart_disease])
]).

tosid('10', 'F2', 'PAR-ACE-CMP:C8H-9NO-2P', 'Paracetamol', [
    type(compound),
    domain(chemistry),
    molecular_formula('C8H9NO2'),
    chemical_name(acetaminophen),
    mechanism(cox3_inhibition),
    effects([analgesic, antipyretic]),
    side_effects([liver_toxicity_high_dose]),
    contraindications([liver_disease, chronic_alcohol_use]),
    safe_in_pregnancy(true)
]).

tosid('10', 'F2', 'FER-SUL-CMP:FE1-SO4-F01', 'Ferrous Sulfate', [
    type(compound),
    domain(chemistry),
    molecular_formula('FeSO4'),
    chemical_class(iron_salt),
    mechanism(iron_replacement),
    effects([hemoglobin_synthesis, oxygen_transport_improvement]),
    side_effects([constipation, stomach_upset, dark_stools]),
    absorption_enhancers([vitamin_c, empty_stomach])
]).

tosid('10', 'F2', 'MAG-OXI-CMP:MG1-O01-M01', 'Magnesium Oxide', [
    type(compound),
    domain(chemistry),
    molecular_formula('MgO'),
    chemical_class(mineral_salt),
    mechanism(magnesium_replacement),
    effects([muscle_relaxation, laxative]),
    side_effects([diarrhea, stomach_upset]),
    bioavailability(low)
]).

tosid('10', 'F2', 'VIT-C-CMP:C6H-8O6-V01', 'Ascorbic Acid', [
    type(compound),
    domain(chemistry),
    molecular_formula('C6H8O6'),
    chemical_name('vitamin_c'),
    mechanism(antioxidant),
    effects([collagen_synthesis, immune_support, iron_absorption_enhancement]),
    deficiency_disease(scurvy),
    side_effects([kidney_stones_high_dose])
]).

% Chemical Interactions (Natural Conceptual - Process Scale)
tosid('01', 'D3', 'IRN-ABS-INT:FE-VIT-I01', 'Iron-Vitamin C Interaction', [
    type(chemical_interaction),
    domain(chemistry),
    interaction_type(synergistic),
    mechanism(reduction_enhancement),
    effect(increased_iron_absorption),
    clinical_significance(high)
]).

tosid('01', 'D3', 'CAL-IRN-INT:CA-FE-I02', 'Calcium-Iron Interaction', [
    type(chemical_interaction),
    domain(chemistry),
    interaction_type(antagonistic),
    mechanism(competitive_absorption),
    effect(decreased_iron_absorption),
    clinical_significance(moderate)
]).