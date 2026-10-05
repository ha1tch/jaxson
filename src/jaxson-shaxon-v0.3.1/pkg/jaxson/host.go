// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

// The exported surface a dialect built on this package needs and cannot
// otherwise reach: raising a documented Err, binding a `local.*` name for
// the duration of a nested evaluation, and statically checking an operand
// or path against a caller-declared set of locals. Nothing here changes
// what a core program does.

// recoverErr converts a panic carrying an *Err into a returned error and
// re-panics anything else. It must be deferred directly.
func recoverErr(err **Err) {
	if r := recover(); r != nil {
		e, ok := r.(*Err)
		if !ok {
			panic(r)
		}
		*err = e
	}
}

// Fail raises a failure with the given category and code. It does not
// return: it panics with an *Err, which Profile.Run (and Run) recover and
// return, exactly as the core instructions' own failures are. Call it
// from a Hooks method or a registered instruction's Check or Exec.
func Fail(cat, code, format string, a ...any) { fail(cat, code, format, a...) }

// WithLocal binds `local.<name>` to v while fn runs, then restores
// whatever was bound before (or unbinds it), including when fn panics.
// Nested calls for the same name therefore behave like nested scopes, which
// a recursive evaluation needs. Core programs cannot shadow a local (the
// static checker rejects it); a host that rebinds a name it owns is
// responsible for that being intended.
func (m *Machine) WithLocal(name string, v any, fn func()) {
	n := len(m.locals)
	m.locals = append(m.locals, localVar{name, v})
	defer m.dropLocals(n)
	fn()
}

func hostChecker(locals []string) *Checker {
	c := NewChecker(CoreInstructions())
	for _, l := range locals {
		c.Locals[l] = true
	}
	return c
}

// CheckOperand statically checks an operand (`$path`, `$lit`, `$compute`,
// `$tpl`, or a plain literal) as the core would check an instruction's
// operand field, with the named `local.*` bindings treated as in scope.
// It returns a documented Err rather than panicking.
func CheckOperand(x any, locals ...string) (err *Err) {
	defer recoverErr(&err)
	checkOperand(x, hostChecker(locals))
	return nil
}

// CheckPath statically checks a path used for reading (a segment array
// whose segments may themselves be operand forms), with the named
// `local.*` bindings treated as in scope. It returns a documented Err
// rather than panicking.
func CheckPath(p any, locals ...string) (err *Err) {
	defer recoverErr(&err)
	checkPath(p, hostChecker(locals), false)
	return nil
}
