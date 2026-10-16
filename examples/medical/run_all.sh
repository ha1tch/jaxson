#!/bin/sh
# Re-runs the whole spike. JAXRUN and SHAXONRUN name the binaries built from the jaxson repository
# (go build ./cmd/jaxrun ./cmd/shaxonrun); swipl (SWI-Prolog 9) is needed for the medical3 steps.
set -e
cd "$(dirname "$0")"
JAXRUN="${JAXRUN:-jaxrun}"; SHAXONRUN="${SHAXONRUN:-shaxonrun}"; export JAXRUN
echo "== v1 coordinator: Jaxson program against the original Prolog output"
python3 medical/spike.py
"$JAXRUN" medical/medical_pkg.json > medical/run_out.json
python3 medical/compare.py
echo "== v1 knowledge base: Shaxon validation"
python3 medical/shaxon_kb.py
"$SHAXONRUN" medical/kb_shaxon.json > medical/kb_report.json || true
python3 -c "import json; r = json.load(open('medical/kb_report.json'))['report']; print('findings:', len(r['violations']))"
echo "== medical3 0.10.0: export from the Prolog engine, derive in Jaxson, compare"
(cd medical3 && swipl -q export.pl < /dev/null > data/export_raw.txt)
python3 -c "t = open('medical3/data/export_raw.txt').read(); open('medical3/data/m3.json', 'w').write(t[t.find('{\"facts\"'):])"
python3 medical3/gen.py
"$JAXRUN" medical3/data/m3_pkg.json > medical3/data/m3_out.json
python3 medical3/check.py
echo "== medical3: knowledge-only overlays and the contract properties"
python3 medical3/overlays.py
