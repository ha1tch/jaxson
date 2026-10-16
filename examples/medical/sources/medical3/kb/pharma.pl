% pharma.pl - Pharmaceutical Manufacturing Knowledge Base
% Drug manufacturing, formulations, dosages, and availability

% Declare tosid as multifile to allow definitions across multiple files
:- multifile(tosid/5).
:- dynamic(tosid/5).

% tosid(TaxonomyCode, Netmask, Identifier, EntityName, Properties).

% Pharmaceutical Products (Artificial Material - Component Scale)
tosid('10', 'E4', 'ASP-TAB-325:ACE-SAL-T01', 'Aspirin 325mg Tablet', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient('acetylsalicylic_acid'),
    strength_mg(325),
    formulation(tablet),
    manufacturer('Generic Pharma'),
    indication([pain, fever, inflammation, cardiovascular_protection]),
    contraindications([bleeding_disorders, gastric_ulcers, children_with_viral_illness]),
    dosage_adult_mg(325, 650),
    dosage_frequency('every_4_6_hours'),
    max_daily_dose_mg(4000),
    cost_per_tablet(0.05),
    availability(otc),
    shelf_life_months(36)
]).

tosid('10', 'E4', 'IBU-CAP-200:IBU-PRO-C01', 'Ibuprofen 200mg Capsule', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient(ibuprofen),
    strength_mg(200),
    formulation(capsule),
    manufacturer('Pain Relief Corp'),
    indication([pain, fever, inflammation]),
    contraindications([kidney_disease, heart_disease, gastric_ulcers]),
    dosage_adult_mg(200, 400),
    dosage_frequency('every_6_hours'),
    max_daily_dose_mg(1200),
    cost_per_capsule(0.12),
    availability(otc),
    shelf_life_months(24)
]).

tosid('10', 'E4', 'PAR-TAB-500:ACE-MIN-T02', 'Paracetamol 500mg Tablet', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient(acetaminophen),
    strength_mg(500),
    formulation(tablet),
    manufacturer('Safe Meds Inc'),
    indication([pain, fever]),
    contraindications([liver_disease, chronic_alcohol_use]),
    dosage_adult_mg(500, 1000),
    dosage_frequency('every_4_6_hours'),
    max_daily_dose_mg(4000),
    cost_per_tablet(0.08),
    availability(otc),
    pregnancy_category(safe),
    shelf_life_months(48)
]).

tosid('10', 'E4', 'IRN-SUP-65M:FER-SUL-S01', 'Iron Supplement 65mg', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient('ferrous_sulfate'),
    elemental_iron_mg(65),
    strength_mg(325),
    formulation(tablet),
    manufacturer('Nutrient Labs'),
    indication([iron_deficiency, anemia]),
    contraindications([hemochromatosis, hemosiderosis]),
    dosage_adult_mg(65),
    dosage_frequency('once_daily'),
    side_effects([constipation, stomach_upset, dark_stools]),
    cost_per_tablet(0.15),
    availability(otc),
    take_with_food(false),
    shelf_life_months(36)
]).

tosid('10', 'E4', 'MAG-TAB-400:MAG-OXI-M01', 'Magnesium Supplement 400mg', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient('magnesium_oxide'),
    elemental_magnesium_mg(240),
    strength_mg(400),
    formulation(tablet),
    manufacturer('Mineral Health Co'),
    indication([magnesium_deficiency, muscle_cramps, constipation]),
    dosage_adult_mg(400),
    dosage_frequency('once_daily'),
    side_effects([diarrhea, stomach_upset]),
    cost_per_tablet(0.20),
    availability(otc),
    take_with_food(true),
    shelf_life_months(24)
]).

tosid('10', 'E4', 'VIT-C-500:ASC-ACI-V01', 'Vitamin C 500mg Tablet', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient('ascorbic_acid'),
    strength_mg(500),
    formulation(tablet),
    manufacturer('Vitamin Works'),
    indication([vitamin_c_deficiency, immune_support, iron_absorption_aid]),
    dosage_adult_mg(500),
    dosage_frequency('once_daily'),
    side_effects([kidney_stones_high_dose, stomach_upset]),
    cost_per_tablet(0.10),
    availability(otc),
    take_with_food(false),
    shelf_life_months(30)
]).

tosid('10', 'E4', 'VIT-D3-2K:CHO-CAL-V02', 'Vitamin D3 2000 IU', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient('cholecalciferol'),
    strength_iu(2000),
    formulation(softgel),
    manufacturer('Bone Health Labs'),
    indication([vitamin_d_deficiency, bone_health, mood_support]),
    dosage_adult_iu(2000),
    dosage_frequency('once_daily'),
    side_effects([hypercalcemia_high_dose]),
    cost_per_softgel(0.18),
    availability(otc),
    take_with_food(true),
    shelf_life_months(36)
]).

tosid('10', 'E4', 'B12-SUB-1MG:CYA-COB-B01', 'Vitamin B12 1000mcg Sublingual', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredient('cyanocobalamin'),
    strength_mcg(1000),
    formulation(sublingual_tablet),
    manufacturer('Neuro Nutrients'),
    indication([b12_deficiency, pernicious_anemia, fatigue]),
    dosage_adult_mcg(1000),
    dosage_frequency('once_daily'),
    cost_per_tablet(0.25),
    availability(otc),
    absorption_route(sublingual),
    shelf_life_months(24)
]).

% Combination Products
tosid('10', 'E4', 'IRN-VIT-CMB:FE-ASC-C01', 'Iron + Vitamin C Combination', [
    type(pharmaceutical),
    domain(pharma),
    active_ingredients(['ferrous_sulfate', 'ascorbic_acid']),
    iron_content_mg(65),
    vitamin_c_content_mg(250),
    formulation(tablet),
    manufacturer('Enhanced Absorption Co'),
    indication([iron_deficiency, enhanced_absorption]),
    advantage(improved_iron_absorption),
    cost_per_tablet(0.22),
    availability(otc),
    shelf_life_months(30)
]).

% Manufacturing Information
tosid('10', 'C1', 'GMP-FAC-001:PHR-MFG-F01', 'GMP Manufacturing Facility Alpha', [
    type(manufacturing_facility),
    domain(pharma),
    certification(gmp, fda_approved),
    capacity_tablets_per_day(1000000),
    specialization([tablets, capsules, softgels]),
    quality_rating(a_grade),
    location('New Jersey, USA'),
    produces([aspirin, ibuprofen, paracetamol, supplements])
]).