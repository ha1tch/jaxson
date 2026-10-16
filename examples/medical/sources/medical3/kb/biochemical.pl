% biochemical.pl - Biochemical Knowledge Base
% Biological molecules, pathways, and processes

% Declare tosid as multifile to allow definitions across multiple files
:- multifile(tosid/5).
:- dynamic(tosid/5).

% tosid(TaxonomyCode, Netmask, Identifier, EntityName, Properties).

% Biological Molecules (Natural Conceptual - Molecular Scale)
tosid('01', 'B8', 'GLU-COS-MOL:C6H-12O-6G1', 'Glucose', [
    type(molecule),
    domain(biochemical),
    molecular_formula('C6H12O6'),
    function(energy_metabolism),
    pathway(glycolysis),
    normal_range_mmol(3.9, 5.5),
    deficiency_symptoms([fatigue, dizziness, confusion]),
    excess_symptoms([hyperglycemia, polyuria, polydipsia])
]).

tosid('01', 'B8', 'INS-ULI-HOR:PRT-HRM-P21', 'Insulin', [
    type(hormone),
    domain(biochemical),
    molecular_type(protein),
    function(glucose_regulation),
    produced_by(pancreas),
    target_tissue([muscle, liver, adipose]),
    deficiency_disease(diabetes),
    deficiency_symptoms([hyperglycemia, polyuria, polydipsia, fatigue])
]).

tosid('01', 'B8', 'SER-OTO-NEU:5HT-NTR-M01', 'Serotonin', [
    type(neurotransmitter),
    domain(biochemical),
    molecular_formula('C10H12N2O'),
    function(mood_regulation),
    pathway(tryptophan_metabolism),
    location([brain, gut]),
    deficiency_symptoms([depression, anxiety, sleep_disorders, irritability]),
    precursor(tryptophan)
]).

tosid('01', 'B8', 'DOP-AMI-NEU:C8H-11N-O2N', 'Dopamine', [
    type(neurotransmitter),
    domain(biochemical),
    molecular_formula('C8H11NO2'),
    function(reward_motivation),
    pathway(tyrosine_metabolism),
    location(brain),
    deficiency_symptoms([depression, lack_motivation, parkinsons]),
    precursor(tyrosine)
]).

tosid('01', 'B8', 'COR-TIS-HOR:STR-HRM-C21', 'Cortisol', [
    type(hormone),
    domain(biochemical),
    molecular_type(steroid),
    function(stress_response),
    produced_by(adrenal_cortex),
    excess_symptoms([anxiety, insomnia, weight_gain, hypertension]),
    deficiency_symptoms([fatigue, muscle_weakness, low_blood_pressure])
]).

tosid('01', 'B8', 'HEM-GLO-PRT:FE2-OXY-P55', 'Hemoglobin', [
    type(protein),
    domain(biochemical),
    function(oxygen_transport),
    contains_element(iron),
    location(red_blood_cells),
    deficiency_symptoms([anemia, fatigue, shortness_of_breath, pale_skin]),
    normal_range_g_dl(12, 16)
]).

tosid('01', 'B8', 'VIT-D3-HOR:CAL-REG-V01', 'Vitamin D3', [
    type(vitamin_hormone),
    domain(biochemical),
    molecular_formula('C27H44O'),
    function(calcium_regulation),
    produced_by(skin),
    activation_site(liver, kidney),
    deficiency_symptoms([bone_pain, muscle_weakness, depression]),
    normal_range_ng_ml(30, 100)
]).

tosid('01', 'B8', 'VIT-B12-COB:B12-VIT-C01', 'Vitamin B12', [
    type(vitamin),
    domain(biochemical),
    molecular_formula('C63H88CoN14O14P'),
    function(dna_synthesis),
    contains_element(cobalt),
    storage_site(liver),
    deficiency_symptoms([fatigue, weakness, memory_problems, anemia]),
    normal_range_pg_ml(200, 900)
]).

% Metabolic Pathways (Natural Conceptual - Process Scale)
tosid('01', 'D2', 'GLY-COL-PTH:GLU-ATP-P01', 'Glycolysis Pathway', [
    type(metabolic_pathway),
    domain(biochemical),
    substrate(glucose),
    product([atp, pyruvate]),
    location(cytoplasm),
    dysfunction_symptoms([muscle_fatigue, exercise_intolerance])
]).

tosid('01', 'D2', 'SER-SYN-PTH:TRP-5HT-P02', 'Serotonin Synthesis Pathway', [
    type(metabolic_pathway),
    domain(biochemical),
    substrate(tryptophan),
    product(serotonin),
    location([brain, gut]),
    dysfunction_symptoms([depression, anxiety, sleep_disorders])
]).