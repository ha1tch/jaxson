#!/bin/sh
# compare.sh - run Experiment 1 on the legacy coordinator and on medical3.
export LANG=C.UTF-8
echo "=== LEGACY ==="
swipl bench/legacy_baseline.pl 2>&1
echo
echo "=== MEDICAL3 (kinds exp1_*) ==="
swipl bench/bench.pl 2>&1 | grep -E "exp1_"
