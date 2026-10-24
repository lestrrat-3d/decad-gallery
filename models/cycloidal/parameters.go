package cycloidal

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// planeGap is the gap, in millimeters, between the disc stack and the housing base and the output plate.
const planeGap = 1

// defineParameters writes the inputs and the derived dimensions into t, one call per name. The pin radius
// and the output hole diameter are numeric snapshots of the auto-or-override choice, which no expression
// can reproduce; every other derived dimension is an expression over the names defined before it.
func defineParameters(t *param.Table, p Params) error {
	d := resolve(p)
	mm := units.Millimeters
	var first error
	record := func(name string, err error) {
		if err != nil && first == nil {
			first = fmt.Errorf("parameter %s: %w", name, err)
		}
	}
	record("pin_count", t.SetNumber("pin_count", float64(p.PinCount)))
	record("pin_circle_diameter", t.SetValue("pin_circle_diameter", mm(p.PinCircleDiameter)))
	record("pin_diameter", t.SetValue("pin_diameter", mm(p.PinDiameter)))
	record("eccentricity", t.SetValue("eccentricity", mm(p.Eccentricity)))
	record("disk_clearance", t.SetValue("disk_clearance", mm(p.DiskClearance)))
	record("disc_count", t.SetNumber("disc_count", float64(p.DiscCount)))
	record("disc_thickness", t.SetValue("disc_thickness", mm(p.DiscThickness)))
	record("disc_gap", t.SetValue("disc_gap", mm(p.DiscGap)))
	record("center_bearing_diameter", t.SetValue("center_bearing_diameter", mm(p.CenterBearingDiameter)))
	record("input_shaft_diameter", t.SetValue("input_shaft_diameter", mm(p.InputShaftDiameter)))
	record("bearing_clearance", t.SetValue("bearing_clearance", mm(p.BearingClearance)))
	record("output_pin_circle_diameter", t.SetValue("output_pin_circle_diameter", mm(p.OutputPinCircleDiameter)))
	record("output_pin_count", t.SetNumber("output_pin_count", float64(p.OutputPinCount)))
	record("output_pin_diameter", t.SetValue("output_pin_diameter", mm(p.OutputPinDiameter)))
	record("wall", t.SetValue("wall", mm(p.HousingWall)))
	record("base_thickness", t.SetValue("base_thickness", mm(p.BaseThickness)))
	record("output_plate_thickness", t.SetValue("output_plate_thickness", mm(p.OutputPlateThickness)))
	record("chamfer_size", t.SetValue("chamfer_size", mm(p.ChamferSize)))
	record("lobes", t.SetExpr("lobes", "pin_count - 1", units.One))
	record("pin_circle_radius", t.SetExpr("pin_circle_radius", "pin_circle_diameter / 2", units.Millimeter))
	record("output_pin_circle_radius",
		t.SetExpr("output_pin_circle_radius", "output_pin_circle_diameter / 2", units.Millimeter))
	record("pin_radius", t.SetValue("pin_radius", mm(d.pinRadius)))
	record("output_hole_diameter", t.SetValue("output_hole_diameter", mm(d.outputHoleDiameter)))
	record("root_circle_diameter", t.SetExpr("root_circle_diameter",
		"2 * (pin_circle_radius - pin_radius - disk_clearance - eccentricity)", units.Millimeter))
	record("housing_inner_diameter", t.SetExpr("housing_inner_diameter",
		"2 * (pin_circle_radius - pin_radius - wall)", units.Millimeter))
	record("housing_outer_diameter", t.SetExpr("housing_outer_diameter",
		"2 * (pin_circle_radius - pin_radius + 2 * eccentricity + wall)", units.Millimeter))
	record("disc_bore_diameter", t.SetExpr("disc_bore_diameter",
		"center_bearing_diameter + bearing_clearance", units.Millimeter))
	record("resolved_output_pin_diameter", t.SetExpr("resolved_output_pin_diameter",
		"output_hole_diameter - 2 * eccentricity", units.Millimeter))
	record("output_plate_diameter", t.SetExpr("output_plate_diameter",
		"output_pin_circle_diameter + resolved_output_pin_diameter + 2 * wall", units.Millimeter))
	record("plane_gap", t.SetValue("plane_gap", mm(planeGap)))
	record("disc_pitch", t.SetExpr("disc_pitch", "disc_thickness + disc_gap", units.Millimeter))
	record("stack_height", t.SetExpr("stack_height",
		"disc_thickness * disc_count + disc_gap * (disc_count - 1)", units.Millimeter))
	record("output_plate_height", t.SetExpr("output_plate_height", "stack_height + plane_gap", units.Millimeter))
	record("output_pin_length", t.SetExpr("output_pin_length",
		"output_plate_height + output_plate_thickness", units.Millimeter))
	return first
}

// recordParameters defines the drive's parameters in doc and reads the derived dimensions back, so the
// document's Bindings ledger shows what the build uses.
func recordParameters(doc *decad.Document, p Params) error {
	if err := defineParameters(doc.Params(), p); err != nil {
		return fmt.Errorf("cycloidal: define parameters: %w", err)
	}
	return nil
}
