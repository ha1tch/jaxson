// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

import "github.com/ha1tch/jaxson/pkg/jaxson"

// Every error identifier Shaxon introduces is prefixed SHAX_, so a Cat/Code
// pair reveals at a glance whether it originated in Jaxson or in Shaxon —
// see shaxon-v0.3.1-core.md section 9. Two deliberate exceptions, not
// oversights:
//
//   - EXECUTION_ERROR itself keeps Jaxson's own spelling: core section 9
//     calls it "the existing Jaxson category, extended," not a new one.
//     Only the *new codes* Shaxon adds within it get the SHAX_ prefix.
//   - Where Shaxon reuses Jaxson's own TYPE_ERROR/MISSING_PATH codes
//     unchanged — an index key or `reference` value resolving to the
//     wrong JSON type, or `local.step` resolving to a non-scalar value —
//     those keep Jaxson's own spelling too. They are not new conditions
//     Shaxon invented; they are Jaxson's own machinery surfacing the same
//     failure it always has, just reached via a Shaxon-declared path, so
//     jaxson.Fail("EXECUTION_ERROR", "TYPE_ERROR", ...) is called directly
//     wherever that applies, with no Shaxon-side constant for it.
const (
	// CatShapeError is raised for every load-time defect in a package's
	// shapes/indices/relations/computes declarations: malformed entries,
	// an extends cycle, a missing maxShapeDepth where recursion is
	// reachable, and so on (core section 9's full list). Deliberately
	// carries no Code — core section 9's own table leaves this category's
	// Codes column empty, mirroring Jaxson's own PROGRAM_ERROR/
	// SCHEMA_ERROR convention of category-plus-message rather than a
	// further code.
	CatShapeError = "SHAX_SHAPE_ERROR"

	// CatValidationError is raised when a gate-mode target fails to
	// conform (Phase 4/5 — not raised anywhere in Phase 2).
	CatValidationError = "SHAX_VALIDATION_ERROR"
	CodeShapeMismatch  = "SHAX_SHAPE_MISMATCH"

	// New EXECUTION_ERROR codes Shaxon adds (Phase 3/4/5 — not raised
	// anywhere in Phase 2, since no Machine runs during static parsing).
	CodeDanglingReference  = "SHAX_DANGLING_REFERENCE"
	CodeShapeDepthExceeded = "SHAX_SHAPE_DEPTH_EXCEEDED"
	CodePathDepthExceeded  = "SHAX_PATH_DEPTH_EXCEEDED"
)

// failLoad raises a SHAX_SHAPE_ERROR. Every static defect found while
// parsing shapes/indices/relations/computes goes through this one
// function, so the category spelling lives in exactly one place.
func failLoad(format string, a ...any) {
	jaxson.Fail(CatShapeError, "", format, a...)
}
