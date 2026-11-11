package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a planar coordinate in sketch space.
type point struct{ x, y float64 }

// dimensions holds the derived radii and the backlash read from the
// parameter table, all in millimeters.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// defineParameters writes the spur parameter table into t.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("define parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("define parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("define parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("define parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("define parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("define parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("define parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("define parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("define parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter bore_tool_length: %w", err)
	}
	return nil
}

// readDimensions reads the derived radii from the document's parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	read := func(name string) (float64, error) {
		v, err := doc.Param(name)
		if err != nil {
			return 0, fmt.Errorf("read parameter %s: %w", name, err)
		}
		return v.Base(), nil
	}
	var d dimensions
	var err error
	if d.pitch, err = read("pitch_radius"); err != nil {
		return dimensions{}, err
	}
	if d.base, err = read("base_radius"); err != nil {
		return dimensions{}, err
	}
	if d.root, err = read("root_radius"); err != nil {
		return dimensions{}, err
	}
	if d.tip, err = read("tip_radius"); err != nil {
		return dimensions{}, err
	}
	if d.fillet, err = read("fillet_radius"); err != nil {
		return dimensions{}, err
	}
	if d.backlash, err = read("backlash"); err != nil {
		return dimensions{}, err
	}
	return d, nil
}

// derive computes the dimensions of p on a scratch document. It returns the
// zero value when the parameter table cannot be defined or read.
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

// rotate turns (x, y) about the origin by theta.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base-circle point onto the root circle, multiplying
// before dividing.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of base radius base at radius r,
// mirrored across the X axis.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	x := base * (math.Cos(t) + t*math.Sin(t))
	y := base * (math.Sin(t) - t*math.Cos(t))
	return point{x, -y}
}

// flankRotation returns the rotation beta that places the pitch crossing of
// the mirrored involute at pi/(2 teeth) - delta.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	pp := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both base-to-tip, then rotates both by angle. It returns left then right.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := involutePoint(base, r)
		l := rotate(m.x, m.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks, both base-to-tip,
// for the given radii and real-valued effective tooth count, rotated by angle.
// It applies no backlash.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip, "effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: %s must be finite", name)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius must be positive")
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius must exceed base radius")
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius must exceed pitch radius")
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth must be positive")
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples must be at least 3")
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	pairs := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, q := range in {
			out[i] = [2]float64{q.x, q.y}
		}
		return out
	}
	return pairs(left), pairs(right), nil
}
