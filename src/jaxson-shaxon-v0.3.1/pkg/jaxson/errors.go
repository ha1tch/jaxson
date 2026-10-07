// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package jaxson

import "fmt"

// ---------------------------------------------------------------- errors

// Err is the one error type Run, RunJSON, the parsers and Shaxon's entry
// points return. Cat is the category, from a fixed set: PARSE_ERROR (not JSON,
// a duplicate key, trailing data), VERSION_ERROR (unsupported version, limit or
// cost table), PROGRAM_ERROR (the package is not valid before anything runs),
// INPUT_ERROR and OUTPUT_ERROR (a schema check failed), EXECUTION_ERROR (a
// failure while running, with a Code such as TYPE_ERROR, MISSING_PATH or
// DIV_ZERO), RESOURCE_ERROR (a limit was exceeded, Code STEPS among them), and
// Shaxon's own SHAX_VALIDATION_ERROR and SHAX_SHAPE_ERROR. Code is the stable
// detail within a category and may be empty. Msg is for people and may change.
// Callers should compare Cat and Code, not Msg.
type Err struct{ Cat, Code, Msg string }

func (e *Err) Error() string {
	if e.Code == "" {
		return e.Cat + ": " + e.Msg
	}
	return e.Cat + "/" + e.Code + ": " + e.Msg
}

func fail(cat, code, format string, a ...any) {
	panic(&Err{cat, code, fmt.Sprintf(format, a...)})
}
func execFail(code, format string, a ...any) { fail("EXECUTION_ERROR", code, format, a...) }
func progFail(format string, a ...any)       { fail("PROGRAM_ERROR", "", format, a...) }

type haltSignal struct{}
type omitT struct{}

var omit = omitT{}
