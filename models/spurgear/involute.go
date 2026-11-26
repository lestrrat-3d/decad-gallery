package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a planar coordinate pair in sketch coordinates.
type point struct{ x, y float64 }

// dimensions holds the derived gear radii and the backlash, in millimeters.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// defineParameters writes the spur gear parameter table into t.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// readDimensions reads the derived radii and backlash from doc's parameter table.
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
			return dimensions{}, fmt.Errorf("parameter %s: %w", f.name, err)
		}
		*f.dst = v.Base()
	}
	return d, nil
}

// derive computes the gear dimensions on a scratch document. It returns the
// zero dimensions when the parameter table cannot be built or read.
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

// filletRadius returns the root fillet radius of d.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	s, c := math.Sin(theta), math.Cos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a flank's base point out to the root circle. It
// multiplies before dividing so the result matches the sketch's root points.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// mirroredInvolute returns the involute of the base circle at radius r with
// its Y coordinate negated.
func mirroredInvolute(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	sin, cos := math.Sin(t), math.Cos(t)
	return point{base * (cos + t*sin), -(base * (sin - t*cos))}
}

// flankShift returns the rotation that places the left flank's pitch crossing
// at pi/(2*teeth) minus the backlash half-angle.
func flankShift(pitch, base, teeth, backlash float64) float64 {
	pp := mirroredInvolute(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// sampleFlanks samples the left and right involute flanks of one tooth from
// radius start to tip, both endpoints included, and rotates them by angle.
// Both returned arrays run from start to tip.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankShift(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := mirroredInvolute(base, r)
		l := rotate(m.x, m.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// flankAngle returns h(r), the angle between the tooth axis and the left
// flank where it meets radius r.
func flankAngle(pitch, base, teeth, backlash, r float64) float64 {
	beta := flankShift(pitch, base, teeth, backlash)
	m := mirroredInvolute(base, r)
	l := rotate(m.x, m.y, beta)
	return math.Atan2(l.y, l.x)
}

// InvoluteFlanks samples the left and right flanks of one tooth between the
// base and tip radii, both base to tip, rotated by angle. It applies no
// backlash and accepts a real-valued effective tooth count.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite input %v", v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("involute flanks: base radius %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("involute flanks: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("involute flanks: effective teeth %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("involute flanks: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return pairs(left), pairs(right), nil
}

// pairs converts points to coordinate pairs.
func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i] = [2]float64{p.x, p.y}
	}
	return out
}
