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
		{"shaft_diameter", units.Millimeters(p.ShaftDiameter)},
		{"pin_diameter", units.Millimeters(p.PinDiameter)},
		{"pin_clearance", units.Millimeters(p.PinClearance)},
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
		{"ring_root_radius", "module * ring_teeth / 2 + 1.25 * module"},
		{"ring_outer_radius", "ring_root_radius + ring_wall"},
		{"carrier_radius", "ring_tip_radius - module / 2"},
		{"planet_bore_diameter", "pin_diameter + 2 * pin_clearance"},
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
		{"carrier_radius", &d.carrier},
		{"planet_bore_diameter", &d.planetBore},
		{"stage_pitch", &d.stagePitch},
		{"ring_length", &d.ringLength},
	} {
		v, err := doc.Param(t.name)
		if err != nil {
			return dims{}, fmt.Errorf("read parameter %s: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	d.planetRoot = (p.Module*float64(p.PlanetTeeth) - 2.5*p.Module) / 2
	d.planetTip = (p.Module*float64(p.PlanetTeeth) + 2*p.Module) / 2
	if want := derive(p); !closeTo(d.orbit, want.orbit) || !closeTo(d.ringLength, want.ringLength) {
		return dims{}, fmt.Errorf("parameter table disagrees with the inputs: orbit %v vs %v", d.orbit, want.orbit)
	}
	return d, nil
}

func closeTo(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
