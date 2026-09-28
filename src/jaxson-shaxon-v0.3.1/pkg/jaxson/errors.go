// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package jaxson

import "fmt"

// ---------------------------------------------------------------- errors

// Err is a Jaxson failure with a category and an optional stable code.
type Err struct{ Cat, Code, Msg string }

func (e *Err) Error() string { return e.Cat + "/" + e.Code + ": " + e.Msg }

func fail(cat, code, format string, a ...any) {
	panic(&Err{cat, code, fmt.Sprintf(format, a...)})
}
func execFail(code, format string, a ...any) { fail("EXECUTION_ERROR", code, format, a...) }
func progFail(format string, a ...any)       { fail("PROGRAM_ERROR", "", format, a...) }

type haltSignal struct{}
type omitT struct{}

var omit = omitT{}
