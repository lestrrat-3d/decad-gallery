package spurgear

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived gear dimensions, in millimeters.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// paramError wraps a parameter table failure with the parameter name.
func paramError(name string, err error) error {
	return fmt.Errorf("spurgear: define parameter %q: %w", name, err)
}

// defineParameters writes the spur gear parameter table into t. Every name
// and expression is a string literal so that a generator can read them.
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

// readDimensions reads the derived dimensions from the document's table.
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
			return dimensions{}, fmt.Errorf("spurgear: read parameter %q: %w", f.name, err)
		}
		*f.dst = v.Base()
	}
	return d, nil
}

// derive computes the gear dimensions on a scratch document. It returns the
// zero value when the parameters cannot be defined or read.
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

// filletRadius returns the derived root fillet radius.
func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base-circle point to the root circle, multiplying
// before dividing so that each coordinate matches the helical join exactly.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the angle of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of a circle of radius base at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankRotation returns the angle that turns the mirrored involute so that
// its pitch crossing lies at pi/(2*teeth) - backlash/(2*pitch).
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	pp := involutePoint(base, pitch)
	px, py := pp.x, -pp.y
	return math.Pi/(2*teeth) - delta - math.Atan2(py, px)
}

// sampleFlanks samples both flanks of one tooth from radius start to tip,
// including both endpoints, and turns them by angle. It returns the left
// flank first and the right flank second, both running base-to-tip.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(ip.x, -ip.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right flanks of one tooth between the
// base and tip radii, turned by angle. It applies no backlash and accepts a
// real-valued effective tooth count. Both arrays run base-to-tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, errors.New("spurgear: involute flank inputs must be finite")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	pairs := func(pts []point) [][2]float64 {
		out := make([][2]float64, len(pts))
		for i, pt := range pts {
			out[i] = [2]float64{pt.x, pt.y}
		}
		return out
	}
	return pairs(left), pairs(right), nil
}

// closingRadius chooses the radius of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth int) (float64, error) {
	n := float64(teeth)
	beta := flankRotation(d.pitch, d.base, n, d.backlash)
	half := func(r float64) float64 {
		ip := involutePoint(d.base, r)
		l := rotate(ip.x, -ip.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / n
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: adjacent teeth overlap at the root circle (radius %v)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}
