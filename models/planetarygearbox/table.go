package planetarygearbox

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// defineParameters writes the gearbox inputs and derived dimensions into t.
func defineParameters(t *param.Table, p Params) error {
	values := []struct {
		name  string
		value units.Value
	}{
		{"module", units.Millimeters(p.Module)},
		{"pressure_angle", units.Radians(p.PressureAngle)},
		{"gear_thickness", units.Millimeters(p.GearThickness)},
		{"carrier_thickness", units.Millimeters(p.CarrierThickness)},
		{"stage_gap", units.Millimeters(p.StageGap)},
		{"cover_thickness", units.Millimeters(p.CoverThickness)},
		{"bolt_hole_diameter", units.Millimeters(p.BoltHoleDiameter)},
		{"input_shaft_length", units.Millimeters(p.InputShaftLength)},
		{"key_depth", units.Millimeters(p.KeyDepth)},
		{"key_width", units.Millimeters(p.KeyWidth)},
		{"seat_chamfer", units.Millimeters(p.SeatChamfer)},
		{"bearing_bore_diameter", units.Millimeters(planetBearing().BoreDiameter)},
		{"bearing_outer_diameter", units.Millimeters(planetBearing().OuterDiameter)},
		{"bearing_width", units.Millimeters(planetBearing().Width)},
		{"bearing_clearance", units.Millimeters(p.BearingClearance)},
		{"shaft_fit_clearance", units.Millimeters(fitClearance)},
		{"gear_root_clearance", units.Millimeters(gearRootClearance(p.Module))},
		{"gear_root_margin", units.Millimeters(gearRootMeasurementMargin)},
		{"gear_tip_measurement_margin", units.Millimeters(gearRootMeasurementMargin)},
		{"planet_bearing_lift", units.Millimeters(planetBearingLift)},
		{"lip_width", units.Millimeters(lipWidth)},
		{"arm_margin", units.Millimeters(armMargin)},
		{"collar_margin", units.Millimeters(collarMargin)},
		{"carrier_wall", units.Millimeters(p.CarrierWall)},
		{"ring_wall", units.Millimeters(p.RingWall)},
		{"tooth_backlash", units.Millimeters(p.Backlash)},
		{"output_shaft_length", units.Millimeters(p.OutputShaftLength)},
	}
	for _, v := range values {
		if err := t.SetValue(v.name, v.value); err != nil {
			return fmt.Errorf("define parameter %s: %w", v.name, err)
		}
	}
	for _, v := range []struct {
		name  string
		value int
	}{
		{"sun_teeth", p.SunTeeth}, {"planet_teeth", p.PlanetTeeth}, {"ring_teeth", p.RingTeeth},
		{"planet_count", p.PlanetCount}, {"stage_count", p.Stages},
	} {
		if err := t.SetNumber(v.name, float64(v.value)); err != nil {
			return fmt.Errorf("define parameter %s: %w", v.name, err)
		}
	}
	for _, e := range []struct{ name, expr string }{
		{"orbit_radius", "module * (sun_teeth + planet_teeth) / 2"},
		{"ring_tip_radius", "module * ring_teeth / 2 - module"},
		{"ring_root_radius", "module * ring_teeth / 2 + module + gear_root_clearance"},
		{"root_radius", "module * planet_teeth / 2 - module - gear_root_clearance - gear_root_margin"},
		{"tip_radius", "module * planet_teeth / 2 + module - gear_tip_measurement_margin"},
		{"ring_outer_radius", "ring_root_radius + ring_wall"},
		{"pin_diameter", "bearing_bore_diameter - 2 * shaft_fit_clearance"},
		{"shaft_diameter", "bearing_bore_diameter - 2 * shaft_fit_clearance"},
		{"shaft_bore_diameter", "shaft_diameter + 2 * shaft_fit_clearance"},
		{"cover_boss_diameter", "pocket_diameter + 2 * carrier_wall"},
		{"cover_hub_diameter", "cover_boss_diameter + 2 * carrier_wall"},
		{"bolt_circle_radius", "(ring_root_radius + ring_outer_radius) / 2"},
		{"pocket_diameter", "bearing_outer_diameter + 2 * bearing_clearance"},
		{"lip_bore_diameter", "bearing_outer_diameter - 2 * lip_width"},
		{"shaft_collar_diameter", "shaft_diameter + 2 * collar_margin"},
		{"boss_diameter", "pin_diameter + 2 * carrier_wall"},
		{"arm_width", "pin_diameter + 2 * arm_margin"},
		{"stage_pitch", "gear_thickness + carrier_thickness + 2 * stage_gap"},
		{"ring_length", "stage_count * stage_pitch + stage_gap"},
	} {
		if err := t.SetExpr(e.name, e.expr, units.Millimeter); err != nil {
			return fmt.Errorf("define parameter %s: %w", e.name, err)
		}
	}
	return nil
}

// readDims reads the derived dimensions back from the document's table.
func readDims(doc *decad.Document, p Params) (dims, error) {
	var d dims
	for _, t := range []struct {
		name string
		dst  *float64
	}{
		{"orbit_radius", &d.orbit},
		{"ring_tip_radius", &d.ringTip},
		{"ring_root_radius", &d.ringRoot},
		{"ring_outer_radius", &d.ringOuter},
		{"pin_diameter", &d.pin},
		{"shaft_diameter", &d.shaft},
		{"shaft_bore_diameter", &d.shaftBore},
		{"cover_boss_diameter", &d.coverBoss},
		{"cover_hub_diameter", &d.coverHub},
		{"bolt_circle_radius", &d.boltCircle},
		{"pocket_diameter", &d.pocket},
		{"lip_bore_diameter", &d.lip},
		{"shaft_collar_diameter", &d.collar},
		{"boss_diameter", &d.boss},
		{"arm_width", &d.arm},
		{"stage_pitch", &d.stagePitch},
		{"ring_length", &d.ringLength},
	} {
		v, err := doc.Param(t.name)
		if err != nil {
			return dims{}, fmt.Errorf("read parameter %s: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	d.planetRoot = p.Module*float64(p.PlanetTeeth)/2 - gearRootDepth(p.Module) - gearRootMeasurementMargin
	d.planetTip = (p.Module*float64(p.PlanetTeeth)+2*p.Module)/2 - gearRootMeasurementMargin
	d.rimInner = d.boltCircle - p.BoltHoleDiameter/2 - boltRimMargin
	if want := derive(p); !closeTo(d.orbit, want.orbit) || !closeTo(d.ringLength, want.ringLength) {
		return dims{}, fmt.Errorf("parameter table disagrees with the inputs: orbit %v vs %v", d.orbit, want.orbit)
	}
	return d, nil
}

func closeTo(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
