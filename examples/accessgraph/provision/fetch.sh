#!/bin/sh
# Copyright (c) 2026 haitch <h@ual.li>
# SPDX-License-Identifier: Apache-2.0
#
# Fetch the two external datasets used to check accessgraph against data it was
# not written for. About 13 MB is downloaded and about 70 MB kept.
#
#   usage: ./fetch.sh [DATA_DIR]        (default: ../data, which is ignored by git)
#
# amazon-access        real anonymized access data (UCI, CC BY 4.0). Only a
#                      sample is read, straight from the compressed stream:
#                      the archive's main CSV expands to 4.8 GB and must never
#                      be extracted.
# openfga-sample-stores  hand-built authorization models with test assertions
#                      (Apache-2.0), pinned to one commit.
set -eu

HERE="$(cd "$(dirname "$0")" && pwd)"
DATA="${1:-$HERE/../data}"
AMAZON_URL="https://archive.ics.uci.edu/static/public/216/amazon+access+samples.zip"
AMAZON_SHA256="d493e3378508b7f3141253c421c2267923fd8ca60ea4088afe72b21b8228d224"
OPENFGA_URL="https://github.com/openfga/sample-stores.git"
OPENFGA_COMMIT="6802b4d859d2773eea72c2315b3a9f027a2893a0"
SAMPLE_PEOPLE="${SAMPLE_PEOPLE:-36063}"   # the whole file; one to three minutes

mkdir -p "$DATA/amazon-access" "$DATA/openfga-sample-stores"

# 1. Amazon Access Samples
A="$DATA/amazon-access"
if [ ! -f "$A/amazon-access-samples.zip" ]; then
    curl -L --fail -sS -o "$A/amazon-access-samples.zip" "$AMAZON_URL"
fi
echo "$AMAZON_SHA256  $A/amazon-access-samples.zip" | sha256sum -c -
unzip -oq "$A/amazon-access-samples.zip" amzn-anon-access-samples.tgz -d "$A"
python3 "$HERE/amazon_sample.py" "$A/amzn-anon-access-samples.tgz" "$A/derived" "$SAMPLE_PEOPLE"

# 2. OpenFGA sample stores, pinned
O="$DATA/openfga-sample-stores/src"
if [ ! -d "$O/.git" ]; then
    mkdir -p "$O"
    git -C "$O" init -q
    git -C "$O" remote add origin "$OPENFGA_URL"
    # take only the models, the licence and the readme, not the repository's tooling files
    git -C "$O" config core.sparseCheckout true
    printf '%s\n' '/stores/' '/LICENSE' '/NOTICE.txt' '/README.md' > "$O/.git/info/sparse-checkout"
    git -C "$O" fetch -q --depth 1 origin "$OPENFGA_COMMIT"
    git -C "$O" checkout -q FETCH_HEAD
fi
test "$(git -C "$O" rev-parse HEAD)" = "$OPENFGA_COMMIT"
echo "openfga sample stores at $OPENFGA_COMMIT: $(ls "$O/stores" | wc -l) stores"
echo "done"
