// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Phase 1: path static-checking, split out of checks.go per that file's
// own "Phase 1 splits this into..." note. Same logic as before, only the
// context threaded through is now a *Checker (carrying Locals plus the
// instruction table and a host slot) instead of a bare
// map[string]bool — checkPath itself only ever read locals, so this is a
// mechanical signature change, not a behaviour change.

import (
	"math/big"
	"strings"
)

var roots = map[string]bool{"input": true, "state": true, "output": true, "local": true}

func checkPath(p any, c *Checker, write bool) {
	arr, ok := p.([]any)
	if !ok || len(arr) == 0 {
		progFail("a path must be a non-empty array")
	}
	root, ok := arr[0].(string)
	if !ok || !roots[root] {
		progFail("bad path root")
	}
	if write && root != "state" && root != "output" {
		progFail("cannot write to root %q", root)
	}
	if root == "local" {
		if len(arr) < 2 {
			progFail("a local path needs a name")
		}
		name, ok := arr[1].(string)
		if !ok || !c.Locals[name] {
			progFail("undeclared local")
		}
	}
	for _, s := range arr[1:] {
		switch t := s.(type) {
		case string:
		case *big.Rat:
			if !t.IsInt() || t.Sign() < 0 {
				progFail("a literal index must be a non-negative integer")
			}
		case map[string]any:
			isForm := false
			for k := range t {
				if strings.HasPrefix(k, "$") {
					isForm = true
				}
			}
			if !isForm {
				progFail("a path segment must be a string, an index or a form")
			}
			checkOperand(t, c)
		default:
			progFail("bad path segment")
		}
	}
}

func containsForm(x any) bool {
	switch t := x.(type) {
	case map[string]any:
		for k, v := range t {
			if strings.HasPrefix(k, "$") || containsForm(v) {
				return true
			}
		}
	case []any:
		for _, v := range t {
			if containsForm(v) {
				return true
			}
		}
	}
	return false
}
