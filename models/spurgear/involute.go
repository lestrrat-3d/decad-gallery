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

// dimensions holds the derived radii read from the parameter table.
type dimensions struct{ pitch, base, root, tip, fillet float64 }

// defineParameters writes the spur gear parameter table.
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

// derive computes the derived dimensions on a scratch document. It returns
// zero dimensions when the table cannot be defined or read.
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

func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate applies the planar rotation by theta.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x: x*c - y*s, y: x*s + y*c}
}

// rootContact scales a base point onto the root circle, multiplying before
// dividing.
func rootContact(basePoint point, d dimensions) point {
	return point{x: basePoint.x * d.root / d.base, y: basePoint.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involuteAt returns the original involute point at radius r for base radius
// base.
func involuteAt(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	return point{
		x: base * (math.Cos(t) + t*math.Sin(t)),
		y: base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankShift returns the rotation beta that places the mirrored involute's
// pitch crossing at pi/(2 teeth).
func flankShift(pitch, base, teeth float64) float64 {
	pi := involuteAt(base, pitch)
	px, py := pi.x, -pi.y
	return math.Pi/(2*teeth) - math.Atan2(py, px)
}

// sampleFlanks samples both flanks base-to-tip with radii running linearly
// from start to tip. It returns the left array, then the right array.
func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	beta := flankShift(pitch, base, teeth)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involuteAt(base, r)
		l := rotate(ip.x, -ip.y, beta)
		rt := point{x: l.x, y: -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right flanks of one tooth from base to
// tip. The tooth count may be real-valued and is never rounded.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip, "effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: %s must be finite", name)
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
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	l := make([][2]float64, len(left))
	r := make([][2]float64, len(right))
	for i := range left {
		l[i] = [2]float64{left[i].x, left[i].y}
		r[i] = [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
