// Package gearroot deepens a spur gear's root so a tooth tip clears its mate's root by the gear-train rule.
package gearroot

import (
	"fmt"

	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Clearance returns the tip-to-root minimum of docs/model-pipeline.md "Gear train clearances": max(0.25·module, 0.5).
func Clearance(module float64) float64 {
	return max(0.25*module, 0.5)
}

// Define returns a spurgear.Variant Define hook. It sets gear_root_clearance and gear_root_margin with SetValue, then
// sets root_radius to pitch_radius - module - gear_root_clearance - gear_root_margin and tip_radius to
// pitch_radius + module - gear_root_margin. The root goes deeper and the tip lower by the margin, so a measurement
// still meets the clearance. Lengths go in through SetValue, because a literal length inside an expression fails.
func Define(clearance, margin float64) func(*param.Table) error {
	return func(t *param.Table) error {
		if err := t.SetValue("gear_root_clearance", units.Millimeters(clearance)); err != nil {
			return fmt.Errorf("define gear_root_clearance: %w", err)
		}
		if err := t.SetValue("gear_root_margin", units.Millimeters(margin)); err != nil {
			return fmt.Errorf("define gear_root_margin: %w", err)
		}
		if err := t.SetExpr("root_radius",
			"pitch_radius - module - gear_root_clearance - gear_root_margin", units.Millimeter); err != nil {
			return fmt.Errorf("define root_radius: %w", err)
		}
		if err := t.SetExpr("tip_radius", "pitch_radius + module - gear_root_margin", units.Millimeter); err != nil {
			return fmt.Errorf("define tip_radius: %w", err)
		}
		return nil
	}
}
