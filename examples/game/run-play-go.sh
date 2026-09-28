#!/usr/bin/env bash
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
#
# Launches the original Navy Wars driver: play.go + jaxrun.go, the
# pre-module, single-package reference tree at src/jaxson-v0.1.0. This
# is the project's verification oracle -- it stays untouched and
# working on purpose, so the new pkg/jaxson + pkg/jaxtools + jaxplay
# stack's output can be diffed against it (see run-jaxplay.sh).
#
# Usage:
#   ./run-play-go.sh [-seed N]
#
# No go.mod exists at src/jaxson-v0.1.0 on purpose -- it predates the
# module split -- so this is run with GO111MODULE=off rather than
# `go run .` from inside a module.
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OLD_DIR="$HERE/../../src/jaxson-v0.1.0"
PACKAGE="$HERE/navywars.json"

cd "$OLD_DIR"
GO111MODULE=off exec go run . play "$PACKAGE" "$@"
