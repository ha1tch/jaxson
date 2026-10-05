// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the Apache License, Version 2.0.
// https://www.apache.org/licenses/LICENSE-2.0
package shaxon

// Phase 3 (plan section 6): the violation record and validation report of
// core section 10. A Violation is built exactly per that section's
// granularity table; see shapes.go for where each row is produced.

import (
	"fmt"
	"math/big"
	"strings"
)

const (
	SeverityViolation = "violation"
	SeverityWarning   = "warning"
	SeverityInfo      = "info"
)

// Fixed messages for structural findings (core section 10: "not
// author-configurable"). Severity is always SeverityViolation.
const (
	msgDanglingReference  = "referenced value is not present in the named index"
	msgShapeDepthExceeded = "shape recursion exceeded maxShapeDepth"
	msgPathDepthExceeded  = "path closure exceeded maxDepth" // raised by Phase 4 (path closures)

	kindStructural          = "structural"
	kindConstraint          = "constraint"
	reportConformsKey       = "conforms"
	reportViolationsKey     = "violations"
	violationConstraintPath = "constraintPath"
)

// Violation is one entry of a validation report (core section 10).
//
// FocusPath and ConstraintPath hold plain Jaxson path segments: strings
// for object members and ints for array indices. ConstraintPath is nil
// when the finding is not attributable to a specific member — and is then
// omitted from Value(), not rendered as null. ConstraintID and Code are ""
// when absent and render as JSON null. Kind is derived, never stored.
type Violation struct {
	FocusPath      []any
	ConstraintPath []any
	Shape          string
	ConstraintID   string
	Severity       string
	Message        string
	Code           string
}

// Kind is "structural" whenever Code is non-empty, "constraint" otherwise
// (core section 10: mechanically derived, never author-set).
func (v Violation) Kind() string {
	if v.Code != "" {
		return kindStructural
	}
	return kindConstraint
}

// Value renders the violation in Jaxson's value model, exactly as core
// section 10 shows it: there is no "value" member, and "constraintPath" is
// absent (not null) when nil.
func (v Violation) Value() map[string]any {
	m := map[string]any{
		"focusPath":    pathValue(v.FocusPath),
		"kind":         v.Kind(),
		"shape":        nilIfEmpty(v.Shape),
		"constraintId": nilIfEmpty(v.ConstraintID),
		"severity":     v.Severity,
		"message":      v.Message,
		"code":         nilIfEmpty(v.Code),
	}
	if v.ConstraintPath != nil {
		m[violationConstraintPath] = pathValue(v.ConstraintPath)
	}
	return m
}

// Report is the collected result of checking one target. Violations are in
// the order the evaluator produced them, which is deterministic.
type Report struct {
	Violations []Violation
}

// Conforms reflects violation-severity findings only; warning and info
// entries never flip it (core section 10).
func (r Report) Conforms() bool {
	for _, v := range r.Violations {
		if v.Severity == SeverityViolation {
			return false
		}
	}
	return true
}

// Value renders the report as core section 10's {"conforms", "violations"}.
func (r Report) Value() map[string]any {
	list := make([]any, 0, len(r.Violations))
	for _, v := range r.Violations {
		list = append(list, v.Value())
	}
	return map[string]any{reportConformsKey: r.Conforms(), reportViolationsKey: list}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// pathValue converts an internal path (int indices) to the value model
// (numbers are *big.Rat), copying it.
func pathValue(p []any) []any {
	out := make([]any, len(p))
	for i, s := range p {
		if n, ok := s.(int); ok {
			out[i] = big.NewRat(int64(n), 1)
		} else {
			out[i] = s
		}
	}
	return out
}

func appendPath(p []any, seg any) []any {
	out := make([]any, len(p)+1)
	copy(out, p)
	out[len(p)] = seg
	return out
}

func copyPath(p []any) []any {
	if p == nil {
		return nil
	}
	out := make([]any, len(p))
	copy(out, p)
	return out
}

// pathString renders a path for error messages: input.orders[4].status.
func pathString(p []any) string {
	var b strings.Builder
	for i, s := range p {
		switch t := s.(type) {
		case int:
			fmt.Fprintf(&b, "[%d]", t)
		case string:
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(t)
		}
	}
	return b.String()
}
