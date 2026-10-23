package bevelgear

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// defineParameters writes one member's inputs and derived dimensions into t.
// faceWidthAuto selects the default face width over the
// face_width_input value.
func defineParameters(t *param.Table, p Params, faceWidthAuto bool) error {
	for _, e := range []struct {
		name  string
		value units.Value
	}{
		{"module", units.Millimeters(p.Module)},
		{"shaft_angle", units.Radians(p.ShaftAngle)},
		{"pressure_angle", units.Radians(p.PressureAngle)},
		{"face_width_input", units.Millimeters(p.FaceWidth)},
	} {
		if err := t.SetValue(e.name, e.value); err != nil {
			return fmt.Errorf("parameter %s: %w", e.name, err)
		}
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("parameter %s: %w", "teeth", err)
	}
	if err := t.SetNumber("mate_teeth", float64(p.MateTeeth)); err != nil {
		return fmt.Errorf("parameter %s: %w", "mate_teeth", err)
	}
	faceWidth := "default_face_width"
	if !faceWidthAuto {
		faceWidth = "face_width_input"
	}
	for _, e := range []struct{ name, expr, unit string }{
		{"pitch_radius", "module * teeth / 2", "mm"},
		{"mate_pitch_radius", "module * mate_teeth / 2", "mm"},
		{"pitch_span", "hypot(2 * pitch_radius, 2 * mate_pitch_radius)", "mm"},
		{"cone_angle", "atan2(sin(shaft_angle) * pitch_radius, mate_pitch_radius + cos(shaft_angle) * pitch_radius)", "rad"},
		{"mate_cone_angle", "shaft_angle - cone_angle", "rad"},
		{"cone_distance", "pitch_radius / sin(cone_angle)", "mm"},
		{"max_face_width", "0.95 * cone_distance * min(pow(sin(cone_angle), 2), pow(sin(mate_cone_angle), 2))", "mm"},
		{"default_face_width", "min(pitch_span / 6, max_face_width)", "mm"},
		{"face_width", faceWidth, "mm"},
		{"toe_scale", "1 - face_width / cone_distance", ""},
		{"toe_height", "face_width * cos(cone_angle)", "mm"},
		{"dedendum", "1.25 * module", "mm"},
		{"virtual_pitch_radius", "module * teeth / (2 * cos(cone_angle))", "mm"},
		{"virtual_base_radius", "virtual_pitch_radius * cos(pressure_angle)", "mm"},
		{"virtual_tip_radius", "virtual_pitch_radius + module", "mm"},
		{"virtual_root_radius", "virtual_pitch_radius - (1.25 + 0.1125) * module", "mm"},
		{"virtual_teeth", "teeth / cos(cone_angle)", ""},
	} {
		if err := t.SetExpr(e.name, e.expr, unitOf(e.unit)); err != nil {
			return fmt.Errorf("parameter %s: %w", e.name, err)
		}
	}
	return nil
}

func unitOf(s string) units.Unit {
	switch s {
	case "mm":
		return units.Millimeter
	case "rad":
		return units.Radian
	}
	return units.One
}

// recordParameters defines the member's parameters in doc and reads the derived
// dimensions back, so the document's Bindings ledger shows what the build uses.
func recordParameters(doc *decad.Document, p Params) error {
	if err := defineParameters(doc.Params(), p, p.FaceWidth == 0); err != nil {
		return fmt.Errorf("bevelgear: define parameters: %w", err)
	}
	for _, name := range []string{"cone_angle", "cone_distance", "face_width", "toe_scale", "toe_height"} {
		if _, err := doc.Param(name); err != nil {
			return fmt.Errorf("bevelgear: read %s: %w", name, err)
		}
	}
	return nil
}
