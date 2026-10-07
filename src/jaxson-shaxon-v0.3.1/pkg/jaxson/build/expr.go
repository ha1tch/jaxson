// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package build

// ---------------------------------------------------------------- expressions

// Expr is a term of a compute expression. It is a sealed interface:
// constants (Int, Str, Bool, Null, Dec), Var, and the operator
// constructors below. A Path is not an Expr; bind it to a Var with
// Compute.With instead.
type Expr interface{ exprNode() any }

// Var is a named binding inside a compute expression, bound to an Operand
// with Compute.With. Declare each once and reuse it:
//
//	price := jb.Var("price")
//	jb.Calc(jb.Mul(price, jb.Int(2))).With(price, jb.Input("price"))
type Var string

func (v Var) exprNode() any { return map[string]any{"$v": string(v)} }

type call struct {
	op   string
	args []Expr
}

func (c call) exprNode() any {
	a := make([]any, 0, len(c.args)+1)
	a = append(a, c.op)
	for _, x := range c.args {
		a = append(a, x.exprNode())
	}
	return a
}

func fn(op string, args ...Expr) Expr { return call{op, args} }

func plus(a, b Expr, more []Expr) []Expr { return append([]Expr{a, b}, more...) }

// Rounding is a rounding mode for Div and Round.
type Rounding string

// The rounding modes the runtime defines.
const (
	HalfEven Rounding = "half_even" // the default
	HalfUp   Rounding = "half_up"
	Down     Rounding = "down" // toward zero
)

func modeArgs(scale int, mode []Rounding) []Expr {
	out := []Expr{Int(scale)}
	if len(mode) > 0 {
		out = append(out, Str(string(mode[len(mode)-1])))
	}
	return out
}

// Calc wraps an expression as a compute island: an Operand. Bind its
// variables with With.
func Calc(e Expr) Compute { return Compute{e: e} }

// Compute is the $compute form.
type Compute struct {
	e     Expr
	binds map[Var]Operand
}

// With binds a variable to the value of an operand. Every Var the
// expression uses must be bound; this is checked on package assembly.
func (c Compute) With(v Var, o Operand) Compute {
	n := Compute{e: c.e, binds: make(map[Var]Operand, len(c.binds)+1)}
	for k, x := range c.binds {
		n.binds[k] = x
	}
	n.binds[v] = o
	return n
}

func (c Compute) operandNode() any {
	body := map[string]any{"expr": c.e.exprNode()}
	if len(c.binds) > 0 {
		w := make(map[string]any, len(c.binds))
		for k, o := range c.binds {
			w[string(k)] = o.operandNode()
		}
		body["with"] = w
	}
	return map[string]any{"$compute": body}
}

// Arithmetic. Add, Sub and Mul take at least two operands.
func Add(a, b Expr, more ...Expr) Expr { return fn("add", plus(a, b, more)...) }
func Sub(a, b Expr, more ...Expr) Expr { return fn("sub", plus(a, b, more)...) }
func Mul(a, b Expr, more ...Expr) Expr { return fn("mul", plus(a, b, more)...) }
func Neg(x Expr) Expr                  { return fn("neg", x) }
func Abs(x Expr) Expr                  { return fn("abs", x) }
func Mod(a, b Expr) Expr               { return fn("mod", a, b) }

// Min and Max take one or more operands.
func Min(x Expr, more ...Expr) Expr { return fn("min", append([]Expr{x}, more...)...) }
func Max(x Expr, more ...Expr) Expr { return fn("max", append([]Expr{x}, more...)...) }

// Div divides a by b to the given number of decimal places; a division
// that is not exact needs a scale, so it is required. The mode is optional
// and defaults to HalfEven.
func Div(a, b Expr, scale int, mode ...Rounding) Expr {
	return fn("div", append([]Expr{a, b}, modeArgs(scale, mode)...)...)
}

// Round rounds x to the given number of decimal places.
func Round(x Expr, scale int, mode ...Rounding) Expr {
	return fn("round", append([]Expr{x}, modeArgs(scale, mode)...)...)
}

// Comparison.
func Eq(a, b Expr) Expr { return fn("eq", a, b) }
func Ne(a, b Expr) Expr { return fn("ne", a, b) }
func Lt(a, b Expr) Expr { return fn("lt", a, b) }
func Le(a, b Expr) Expr { return fn("le", a, b) }
func Gt(a, b Expr) Expr { return fn("gt", a, b) }
func Ge(a, b Expr) Expr { return fn("ge", a, b) }

// Logic. And and Or short-circuit, as does Select's untaken branch.
func And(x Expr, more ...Expr) Expr { return fn("and", append([]Expr{x}, more...)...) }
func Or(x Expr, more ...Expr) Expr  { return fn("or", append([]Expr{x}, more...)...) }
func Not(x Expr) Expr               { return fn("not", x) }

// Select is the lazy conditional: cond ? then : otherwise.
func Select(cond, then, otherwise Expr) Expr { return fn("select", cond, then, otherwise) }

// Strings, conversion, collections.
func Concat(x Expr, more ...Expr) Expr { return fn("concat", append([]Expr{x}, more...)...) }
func Len(x Expr) Expr                  { return fn("len", x) }
func ToString(x Expr) Expr             { return fn("to_string", x) }
func ToNumber(x Expr) Expr             { return fn("to_number", x) }
func List(items ...Expr) Expr          { return fn("list", items...) }
func Get(container, key Expr) Expr     { return fn("get", container, key) }
func Has(container, key Expr) Expr     { return fn("has", container, key) }
func GetOr(container, key, def Expr) Expr {
	return fn("get_or", container, key, def)
}
func Keys(x Expr) Expr   { return fn("keys", x) }
func TypeOf(x Expr) Expr { return fn("type_of", x) }
