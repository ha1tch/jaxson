#!/bin/sh
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
#
# Runs the full Shaxon / pyshacl / Jena comparison as a series of small batches,
# one per example and profile, one after another. Each batch writes its own
# files under results/ and is skipped if it finished before, so a run that is
# interrupted (a closed laptop, a timeout) is continued by running the same
# command again.
#
#   ./bench.sh                    all 5 examples x 2 profiles = 10 batches
#   ./bench.sh list               show the batches and which are done
#   ./bench.sh run break-glass    only that example (both profiles)
#   ./bench.sh run break-glass:engine    one batch
#   ./bench.sh report             merge results/*.jsonl into results/REPORT.md
#   ./bench.sh clean              forget results (asks first)
#
# Options (before the command):
#   --sizes "100 1000 4000 20000"   event counts (default as shown)
#   --budget SECONDS                longest any one cell may take (default 120);
#                                   a size whose projected time is over it is
#                                   skipped, and so is every larger one
#   --out DIR                       where results go (default ./results)
#   --force                         redo batches that are already done
#
# Needs: ./setup.sh --jena (or ./setup-mac.sh on macOS) done first, and Go.
# Environment: PYTHON (default .venv/bin/python), JENA_LIB (default .jena/lib).
# POSIX sh; nothing here needs GNU tools, so it runs on macOS as it is.
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
cd "$HERE"

SIZES="100 1000 4000 20000"
BUDGET=120
OUT="$HERE/results"
FORCE=0
while [ $# -gt 0 ]; do
    case "$1" in
        --sizes) SIZES=$2; shift 2 ;;
        --budget) BUDGET=$2; shift 2 ;;
        --out) OUT=$2; shift 2 ;;
        --force) FORCE=1; shift ;;
        -h|--help|help) sed -n '5,30p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) break ;;
    esac
done
CMD=${1:-run}
[ $# -gt 0 ] && shift

# macOS: pick up the Homebrew toolchain that setup-mac.sh installed.
if [ "$(uname -s)" = Darwin ] && [ -f "$HERE/env-mac.sh" ]; then
    # shellcheck disable=SC1091
    . "$HERE/env-mac.sh"
fi

PYTHON=${PYTHON:-$HERE/.venv/bin/python}
EXAMPLES="rolling-quota chinese-wall four-eyes-release delegation-chain break-glass"
PROFILES="end-to-end engine"

batches() {
    # prints "example:profile" for the requested selection
    if [ $# -eq 0 ]; then set -- $EXAMPLES; fi
    for sel in "$@"; do
        case "$sel" in
            *:*) echo "$sel" ;;
            *) for p in $PROFILES; do echo "$sel:$p"; done ;;
        esac
    done
}

done_file() { echo "$OUT/$(echo "$1" | tr ':' '.').done"; }

case "$CMD" in
    list)
        for b in $(batches "$@"); do
            if [ -f "$(done_file "$b")" ]; then echo "done     $b"; else echo "pending  $b"; fi
        done
        ;;
    report)
        exec "$PYTHON" report.py "$OUT"
        ;;
    clean)
        printf 'delete %s ? [y/N] ' "$OUT"; read -r ans
        case "$ans" in y|Y) rm -rf "$OUT"; echo removed ;; *) echo kept ;; esac
        ;;
    run)
        if [ ! -x "$PYTHON" ]; then
            echo "bench.sh: no Python at $PYTHON; run ./setup.sh --jena first (macOS: ./setup-mac.sh)" >&2
            exit 1
        fi
        mkdir -p "$OUT"
        n=0; total=$(batches "$@" | wc -l | tr -d ' ')
        for b in $(batches "$@"); do
            n=$((n + 1))
            ex=${b%%:*}; prof=${b##*:}
            case "$ex" in
                rolling-quota|chinese-wall|four-eyes-release|delegation-chain|break-glass) ;;
                *) echo "bench.sh: unknown example $ex" >&2; exit 2 ;;
            esac
            case "$prof" in end-to-end|engine) ;; *) echo "bench.sh: unknown profile $prof" >&2; exit 2 ;; esac
            stem="$OUT/$ex.$prof"
            if [ -f "$(done_file "$b")" ] && [ "$FORCE" -eq 0 ]; then
                echo "[$n/$total] $b: done already (use --force to redo)"
                continue
            fi
            echo "[$n/$total] $b ($(date '+%H:%M:%S'))"
            rm -f "$stem.jsonl" "$stem.txt"
            # a verdict disagreement exits 1; keep the batch's output either way
            if "$PYTHON" scale.py --profile "$prof" --example "$ex" --budget "$BUDGET" --json "$stem.jsonl" $SIZES > "$stem.txt" 2>&1; then
                :
            else
                rc=$?
                if [ "$rc" -ne 1 ]; then
                    echo "bench.sh: $b failed (exit $rc); see $stem.txt" >&2
                    tail -n 5 "$stem.txt" >&2
                    exit "$rc"
                fi
                echo "  note: the engines disagreed on a verdict in this batch; see $stem.txt"
            fi
            date '+%Y-%m-%d %H:%M:%S' > "$(done_file "$b")"
            tail -n +1 "$stem.txt" | sed -n '1,40p'
        done
        echo "all requested batches done; ./bench.sh report to merge them"
        ;;
    *)
        echo "bench.sh: unknown command $CMD (list, run, report, clean)" >&2
        exit 2
        ;;
esac
