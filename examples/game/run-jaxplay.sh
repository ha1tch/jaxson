#!/usr/bin/env bash
# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the Apache License, Version 2.0.
# https://www.apache.org/licenses/LICENSE-2.0
#
# Launches jaxplay against the Navy Wars example, on the new pkg/jaxson
# + pkg/jaxtools stack (src/jaxson-shaxon-v0.3.1). Navy Wars carries its
# state under "game", separately from "view" and "messages", so
# -carry game is the flag it needs -- jaxplay's default carry mode
# (output["game"] -> input["game"]) matches this out of the box; pass
# -carry-whole only if you deliberately want the old whole-output
# splice instead (see cmd/jaxplay/main.go's doc comment).
#
# jaxplay itself requires its flags before the package path; this
# script hides that by always appending the package path last, so any
# arguments given here are just jaxplay flags, in any order:
#
#   ./run-jaxplay.sh                                        # interactive
#   ./run-jaxplay.sh -carry game                             # interactive, state carried
#   ./run-jaxplay.sh -carry game -turns turns-seed0.json -format json
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$HERE/../../src/jaxson-shaxon-v0.3.1"
PACKAGE="$HERE/navywars.json"

# Build in the module directory (go run needs to resolve go.mod there),
# then run the resulting binary from wherever this script was invoked
# from -- not from MODULE_DIR -- so a relative path you pass (e.g.
# -turns turns-seed0.json, typed from examples/game) resolves against
# your own shell, not this script's internals.
BIN="$(mktemp -t jaxplay.XXXXXX)"
trap 'rm -f "$BIN"' EXIT
(cd "$MODULE_DIR" && go build -o "$BIN" ./cmd/jaxplay)
exec "$BIN" "$@" "$PACKAGE"
