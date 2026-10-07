// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

import "github.com/ha1tch/jaxson/pkg/jaxson"

// Core section 4d: a computes entry has exactly the with/expr shape an
// inline $compute does, and is malformed under exactly the same rules
// ("unknown operator, wrong arity, undeclared $v, etc.") — checked here
// via the same delegation checkComputeBody uses for an inline check.

func parseComputes(pkg map[string]any, r *Registries) {
	raw, has := pkg["computes"]
	if !has {
		return
	}
	m, ok := raw.(map[string]any)
	if !ok {
		failLoad("computes must be an object")
	}
	for _, name := range jaxson.SortedKeys(m) {
		entry, ok := m[name].(map[string]any)
		if !ok {
			failLoad("computes.%s must be an object", name)
		}
		for k := range entry {
			if k != "with" && k != "expr" {
				failLoad("computes.%s: unknown key %q", name, k)
			}
		}
		with, _ := entry["with"].(map[string]any)
		if _, present := entry["with"]; present && with == nil {
			failLoad("computes.%s: with must be an object", name)
		}
		expr, hasExpr := entry["expr"]
		if !hasExpr {
			failLoad("computes.%s: needs an expr", name)
		}
		checkComputeBody(with, expr, "computes."+name)
		r.Computes[name] = ComputeDecl{Name: name, With: with, Expr: expr}
	}
}
