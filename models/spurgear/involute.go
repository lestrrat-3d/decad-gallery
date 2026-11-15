package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a planar coordinate in the sketch frame.
type point struct{ x, y float64 }

// dimensions holds the derived gear dimensions, read from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// paramError wraps a parameter definition failure with the parameter name.
func paramError(name string, err error) error {
	return fmt.Errorf("define parameter %q: %w", name, err)
}

// defineParameters writes the spur gear parameter table into t.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return paramError("module", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return paramError("teeth", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return paramError("pressure_angle", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return paramError("thickness", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return paramError("samples", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return paramError("bore_diameter", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return paramError("cap_chamfer", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return paramError("backlash", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return paramError("tool_margin", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return paramError("pitch_radius", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return paramError("base_radius", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return paramError("root_radius", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return paramError("tip_radius", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return paramError("involute_angle", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return paramError("fillet_radius", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return paramError("bore_tool_length", err)
	}
	return nil
}

// DefineParameters writes the Parameter table of p into t. It does not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// readDimensions reads the derived dimensions from the document's parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	fields := []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch},
		{"base_radius", &d.base},
		{"root_radius", &d.root},
		{"tip_radius", &d.tip},
		{"fillet_radius", &d.fillet},
		{"backlash", &d.backlash},
	}
	for _, f := range fields {
		v, err := doc.Param(f.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read parameter %q: %w", f.name, err)
		}
		*f.dst = v.Base()
	}
	return d, nil
}

// derive computes the dimensions of p on a scratch document. It returns the
// zero dimensions when the parameter table cannot be defined or read.
func derive(p Params) dimensions {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}
	}
	d, err := readDimensions(doc)
	if err != nil {
		return dimensions{}
	}
	return d
}

// filletRadius returns the root fillet radius of the derived dimensions.
func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	sin, cos := math.Sincos(theta)
	return point{x: x*cos - y*sin, y: x*sin + y*cos}
}

// rootContact scales a base point onto the root circle. Each coordinate is
// multiplied before it is divided.
func rootContact(basePoint point, d dimensions) point {
	return point{
		x: basePoint.x * d.root / d.base,
		y: basePoint.y * d.root / d.base,
	}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// mirroredInvolute returns the involute point of the base circle at radius r,
// mirrored across the X axis.
func mirroredInvolute(base, r float64) point {
	t := math.Tan(math.Acos(math.Min(1, base/r)))
	sin, cos := math.Sincos(t)
	return point{
		x: base * (cos + t*sin),
		y: -(base * (sin - t*cos)),
	}
}

// flankRotation returns the angle that turns the mirrored involute into the
// left flank, so that the flank crosses the pitch circle at pi/(2N) - delta.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	pm := mirroredInvolute(base, pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pm.y, pm.x)
}

// sampleFlanks samples both flanks of one tooth from radius start to radius
// tip and returns the left array followed by the right array. Both run from
// start to tip. The backlash thins the tooth by that arc at the pitch circle.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := mirroredInvolute(base, r)
		l := rotate(m.x, m.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of a tooth for the
// given pitch, base and tip radii and a real-valued effective tooth count. Both
// arrays run base to tip, and the whole tooth is turned by angle. It applies no
// backlash.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite input %v", v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("involute flanks: base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("involute flanks: effective teeth %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("involute flanks: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return pairs(left), pairs(right), nil
}

// pairs converts points to coordinate pairs.
func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, pt := range pts {
		out[i] = [2]float64{pt.x, pt.y}
	}
	return out
}
