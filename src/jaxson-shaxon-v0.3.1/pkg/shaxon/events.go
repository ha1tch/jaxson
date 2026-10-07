// Copyright (c) 2026 haitch <h@ual.li>
// Licensed under the GNU General Public License, version 3.
// https://www.gnu.org/licenses/gpl-3.0.html
package shaxon

// Event names Shaxon adds to a jaxson.CostTable (docs/proposals/step-cost-model.md,
// section 4.3). The core language prices instructions, operators and loop
// iterations; the events below are the ones only a Shaxon evaluation
// performs, keyed in CostTable.Event. Under the default `unit` table every
// event costs one step, which is what Phase 3 charged before the table
// existed.
//
// Only events that some code path charges today are declared here; Phase 4
// adds its own (index element visit, target resolution, closure hop) in the
// same commit as the code that charges them, so no name exists unpriced.
const (
	// EventShapeActivation: one shape activation (Phase 3 decision G1).
	EventShapeActivation = "shape.activation"
)
