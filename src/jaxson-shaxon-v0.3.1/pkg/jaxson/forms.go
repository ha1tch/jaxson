// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

// Operand forms a dialect adds. The core operand forms are `$path`, `$lit`,
// `$compute` and `$tpl`; a dialect may register more (Shaxon adds
// `$altPath`, `$inverse`, `$path*` and `$path+`), each with a static check
// and an evaluation, without the core knowing what they mean.
//
// A form is a JSON object in which exactly one key is a registered name.
// Unlike a core form it may carry other keys (`{"$path*": step, "maxDepth":
// n}`); the registered definition receives the whole object and is
// responsible for checking all of it. Nothing about the core language
// changes for a program that uses no registered form.

// FormDef is one registered operand form.
type FormDef struct {
	// Check statically checks the whole form object. It raises a failure
	// with Fail, as an instruction's Check does.
	Check func(form map[string]any, c *Checker)
	// Eval evaluates the form to a value. It may be nil in a table used
	// only for static checking (CheckOperandWith).
	Eval func(m *Machine, form map[string]any) any
}

// form returns the registered definition named by one of mp's keys.
func formOf(forms map[string]FormDef, mp map[string]any) (FormDef, bool) {
	if forms == nil {
		return FormDef{}, false
	}
	for k := range mp {
		if d, ok := forms[k]; ok {
			return d, true
		}
	}
	return FormDef{}, false
}

// SetForms installs the operand forms this machine evaluates. Call it
// before anything runs (a dialect's Start hook, or Profile.Run, which does
// it from Hooks.Forms).
func (m *Machine) SetForms(forms map[string]FormDef) {
	m.forms = forms
	m.cache, m.ccache = nil, nil // compiled operands embed the forms
}

// HasForm reports whether the named operand form (for example "$inverse")
// is installed on this machine.
func (m *Machine) HasForm(name string) bool {
	_, ok := m.forms[name]
	return ok
}

// CheckOperandWith is CheckOperand with the given operand forms recognised.
func CheckOperandWith(forms map[string]FormDef, x any, locals ...string) (err *Err) {
	defer recoverErr(&err)
	c := hostChecker(locals)
	c.Forms = forms
	checkOperand(x, c)
	return nil
}

// CheckPathWith is CheckPath with the given operand forms recognised in
// segments.
func CheckPathWith(forms map[string]FormDef, p any, locals ...string) (err *Err) {
	defer recoverErr(&err)
	c := hostChecker(locals)
	c.Forms = forms
	checkPath(p, c, false)
	return nil
}

// CheckProgramForms is CheckProgramHost with operand forms recognised, for
// a dialect that checks a program outside Profile.Run.
func CheckProgramForms(program any, instructions map[string]InstructionDef, host any, forms map[string]FormDef) (err *Err) {
	defer recoverErr(&err)
	c := NewChecker(instructions)
	c.Host = host
	c.Forms = forms
	checkBlock(program, c)
	return nil
}
