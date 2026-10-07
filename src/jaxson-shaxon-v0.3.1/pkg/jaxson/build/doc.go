// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html

// Package build constructs Jaxson packages in Go with a fluent, chained,
// type-checked API, in the same spirit as queryfy's builders and minty's
// HTML builder. It replaces hand-written instruction JSON, and the
// one-off Python script (examples/game/build_navywars.py) that used to
// stand in for it.
//
// Import it with a short alias:
//
//	import jb "github.com/ha1tch/jaxson/pkg/jaxson/build"
//
// A program is assembled from four kinds of value, each its own Go type,
// so an IDE offers only the methods that make sense at each point:
//
//   - Operand: something that yields a value at run time. Constants (Int,
//     Str, Bool, Null, Dec), paths (Input, State, Output, Loop), Data, Lit,
//     Tpl and Calc are all Operands.
//   - Expr: something valid inside a compute expression. Constants, Var and
//     the operator constructors (Add, Eq, Select, Concat ...) are Exprs. A
//     path is deliberately NOT an Expr: a compute island reads the outside
//     world only through named bindings, exactly as in the language.
//   - Instr: one instruction. Set, Append, Insert, Delete, Assert, Halt,
//     If(...).Then(...).Else(...) and For(...).As(...).Do(...).
//   - Schema: a contract schema (Integer, Number, String, Array, Object ...).
//
// Several mistakes are compile errors rather than run-time surprises: an
// If without Then, a For without As or Do, an operator with too few
// operands (Add takes at least two), a path used inside an Expr, an
// unknown operator, an unknown rounding mode. The rest (an unbound Var,
// writing to input, a shadowed loop name, a sample input that violates its
// own schema) is caught when the package is assembled by Package.Map,
// Package.JSON or Package.Check, using the runtime's own static checker, so
// the builder cannot accept a program the runtime would refuse.
//
// Every builder value is immutable: each method returns a modified copy, so
// partial values can be shared and reused safely.
//
// A complete, runnable example is in example_test.go, and a full port of
// the Navy Wars game is in navywars_test.go.
package build
