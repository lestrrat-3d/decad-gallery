package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii and the backlash read from the
// parameter table, all in millimeters.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

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

// readDimensions reads the derived radii and the backlash from the document.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	targets := []struct {
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
	for _, t := range targets {
		v, err := doc.Param(t.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read parameter %s: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// derive computes the dimensions of p on a scratch document. It returns a
// zero dimensions value when the parameters cannot be evaluated.
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
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base-circle point out to the root circle.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint returns the original involute at radius r, mirrored across X.
func involutePoint(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	return point{
		x: base * (math.Cos(t) + t*math.Sin(t)),
		y: -(base * (math.Sin(t) - t*math.Cos(t))),
	}
}

// flankShift returns the rotation that places the pitch crossing of the
// mirrored involute at π/(2·teeth) minus the backlash half-angle.
func flankShift(pitch, base, teeth, backlash float64) float64 {
	pp := involutePoint(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// halfAngle returns the angle between the tooth axis and the left flank at
// radius r.
func halfAngle(pitch, base, teeth, backlash, r float64) float64 {
	beta := flankShift(pitch, base, teeth, backlash)
	q := involutePoint(base, r)
	rq := rotate(q.x, q.y, beta)
	return math.Atan2(rq.y, rq.x)
}

// sampleFlanks samples the left and right involute flanks from radius start
// to radius tip, both ends included, and turns both by angle. Both arrays run
// from start to tip, left array first.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankShift(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		q := involutePoint(base, r)
		l := rotate(q.x, q.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of a tooth for
// the given pitch, base and tip radii and a real-valued effective tooth
// count, with no backlash. Both arrays run base to tip and are turned by
// angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite input")
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
	l := make([][2]float64, len(left))
	r := make([][2]float64, len(right))
	for i := range left {
		l[i] = [2]float64{left[i].x, left[i].y}
		r[i] = [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
