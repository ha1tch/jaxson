% medicine.pl - Medical Knowledge Base
% Diseases, symptoms, treatments, and clinical guidelines

% Declare tosid as multifile to allow definitions across multiple files
:- multifile(tosid/5).
:- dynamic(tosid/5).

% tosid(TaxonomyCode, Netmask, Identifier, EntityName, Properties).

% Medical Conditions (Artificial Conceptual - Systemic Scale)
tosid('11', 'B3', 'DIA-BET-T2D:MET-DIS-D01', 'Type 2 Diabetes', [
    type(disease),
    domain(medicine),
    category(metabolic_disorder),
    pathophysiology(insulin_resistance),
    primary_symptoms([hyperglycemia, polyuria, polydipsia, fatigue]),
    secondary_symptoms([blurred_vision, slow_wound_healing]),
    complications([neuropathy, retinopathy, nephropathy]),
    biochemical_markers([elevated_glucose, elevated_hba1c]),
    diagnostic_criteria([fasting_glucose_gt_126, hba1c_gt_6_5]),
    treatment_approach([lifestyle, medication, monitoring]),
    severity_levels([mild, moderate, severe])
]).

tosid('11', 'B3', 'IRN-DEF-ANE:NUT-DIS-A01', 'Iron Deficiency Anemia', [
    type(disease),
    domain(medicine),
    category(nutritional_disorder),
    pathophysiology(inadequate_iron),
    primary_symptoms([fatigue, weakness, shortness_of_breath, pale_skin]),
    secondary_symptoms([cold_hands_feet, brittle_nails, cravings_ice]),
    biochemical_markers([low_hemoglobin, low_ferritin, low_iron]),
    diagnostic_criteria([hemoglobin_lt_12_women, hemoglobin_lt_13_men]),
    treatment_approach([iron_supplementation, dietary_changes]),
    severity_levels([mild, moderate, severe])
]).

tosid('11', 'B3', 'DEP-MAJ-MDD:PSY-DIS-M01', 'Major Depressive Disorder', [
    type(disease),
    domain(medicine),
    category(psychiatric_disorder),
    pathophysiology(neurotransmitter_imbalance),
    primary_symptoms([depression, anhedonia, fatigue, sleep_disturbance]),
    secondary_symptoms([appetite_changes, concentration_problems, guilt]),
    biochemical_markers([low_serotonin, low_dopamine]),
    treatment_approach([medication, therapy, lifestyle]),
    severity_levels([mild, moderate, severe]),
    duration_criteria(symptoms_gt_2_weeks)
]).

tosid('11', 'B3', 'HEA-DAC-TTH:NEU-DIS-H01', 'Tension Type Headache', [
    type(disease),
    domain(medicine),
    category(neurological_disorder),
    pathophysiology(muscle_tension),
    primary_symptoms([head_pain, neck_tension, pressure_sensation]),
    triggers([stress, dehydration, poor_posture, eye_strain]),
    treatment_approach([analgesics, stress_management, lifestyle]),
    severity_levels([mild, moderate, severe]),
    frequency_types([episodic, chronic])
]).

tosid('11', 'B3', 'FEV-VIR-URT:INF-DIS-F01', 'Viral Upper Respiratory Infection', [
    type(disease),
    domain(medicine),
    category(infectious_disease),
    pathophysiology(viral_infection),
    primary_symptoms([fever, cough, congestion, sore_throat]),
    secondary_symptoms([body_aches, runny_nose, sneezing]),
    duration_days(7, 10),
    treatment_approach([symptomatic_care, rest, hydration]),
    severity_levels([mild, moderate])
]).

tosid('11', 'B3', 'VIT-D-DEF:NUT-DIS-V01', 'Vitamin D Deficiency', [
    type(disease),
    domain(medicine),
    category(nutritional_disorder),
    pathophysiology(inadequate_vitamin_d),
    primary_symptoms([bone_pain, muscle_weakness, fatigue]),
    secondary_symptoms([depression, frequent_infections]),
    biochemical_markers([low_25_oh_vitamin_d]),
    diagnostic_criteria([vitamin_d_lt_30_ng_ml]),
    treatment_approach([vitamin_d_supplementation, sun_exposure]),
    severity_levels([mild, moderate, severe])
]).

tosid('11', 'B3', 'B12-DEF-000:NUT-DIS-B01', 'Vitamin B12 Deficiency', [
    type(disease),
    domain(medicine),
    category(nutritional_disorder),
    pathophysiology(inadequate_b12),
    primary_symptoms([fatigue, weakness, memory_problems]),
    secondary_symptoms([tingling_hands_feet, balance_problems]),
    biochemical_markers([low_b12, elevated_methylmalonic_acid]),
    diagnostic_criteria([b12_lt_200_pg_ml]),
    treatment_approach([b12_supplementation, dietary_changes]),
    severity_levels([mild, moderate, severe])
]).

% Symptoms (Artificial Conceptual - Local Scale)
tosid('11', 'E1', 'FEV-SYM-000:TMP-ELE-F01', 'Fever', [
    type(symptom),
    domain(medicine),
    definition(elevated_body_temperature),
    measurement_celsius(gt_38),
    measurement_fahrenheit(gt_100_4),
    severity_levels([low_grade, moderate, high]),
    causes([infection, inflammation, medication]),
    treatment_options([acetaminophen, ibuprofen, cooling_measures])
]).

tosid('11', 'E1', 'PAI-HEA-000:NEU-SYM-P01', 'Headache', [
    type(symptom),
    domain(medicine),
    definition(head_or_neck_pain),
    severity_scale(1, 10),
    types([tension, migraine, cluster, sinus]),
    causes([stress, dehydration, muscle_tension, vascular]),
    treatment_options([acetaminophen, ibuprofen, aspirin, rest])
]).

tosid('11', 'E1', 'FAT-IGU-000:GEN-SYM-F01', 'Fatigue', [
    type(symptom),
    domain(medicine),
    definition(extreme_tiredness),
    severity_levels([mild, moderate, severe]),
    causes([anemia, depression, sleep_disorders, metabolic_disorders]),
    associated_conditions([iron_deficiency, b12_deficiency, depression])
]).

tosid('11', 'E1', 'PAI-MSC-000:MSC-SYM-P02', 'Muscle Pain', [
    type(symptom),
    domain(medicine),
    definition(muscle_discomfort),
    severity_scale(1, 10),
    causes([overuse, injury, inflammation, electrolyte_imbalance]),
    treatment_options([rest, heat, cold, anti_inflammatories])
]).

% Treatment Guidelines (Artificial Conceptual - Process Scale)
tosid('11', 'D1', 'FEV-TRT-GDL:SYM-MAN-F01', 'Fever Treatment Guideline', [
    type(treatment_guideline),
    domain(medicine),
    condition(fever),
    first_line_treatment(acetaminophen),
    alternative_treatment(ibuprofen),
    dosing_interval_hours(4, 6),
    non_pharmacological([rest, fluids, cooling_measures]),
    when_to_seek_care([fever_gt_39_5c, persistent_gt_3_days])
]).

tosid('11', 'D1', 'HEA-TRT-GDL:NEU-MAN-H01', 'Headache Treatment Guideline', [
    type(treatment_guideline),
    domain(medicine),
    condition(headache),
    first_line_treatment(acetaminophen),
    alternative_treatments([ibuprofen, aspirin]),
    non_pharmacological([rest, hydration, stress_management]),
    when_to_seek_care([sudden_severe, neurological_symptoms])
]).

tosid('11', 'D1', 'ANE-TRT-GDL:HEM-MAN-A01', 'Anemia Treatment Guideline', [
    type(treatment_guideline),
    domain(medicine),
    condition(iron_deficiency_anemia),
    first_line_treatment(iron_supplementation),
    dosing_mg(65),
    duration_weeks(12, 24),
    monitoring([hemoglobin, ferritin]),
    adjuvant_therapy(vitamin_c),
    dietary_recommendations([red_meat, spinach, beans])
]).