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

// dimensions holds the derived radii of a gear, in millimeters.
type dimensions struct{ pitch, base, root, tip, fillet float64 }

func paramError(name string, err error) error {
	return fmt.Errorf("define parameter %q: %w", name, err)
}

// defineParameters writes the gear's inputs and derived dimensions into t.
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
			return dimensions{}, fmt.Errorf("read parameter %q: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// deriveDimensions evaluates the gear's derived radii on a scratch document.
func deriveDimensions(p Params) (dimensions, error) {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}, err
	}
	return readDimensions(doc)
}

// derive returns the gear's derived radii, or zero dimensions on failure.
func derive(p Params) dimensions {
	d, err := deriveDimensions(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// filletRadius returns the root fillet radius of d.
func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

// rotate turns (x, y) counterclockwise by theta about the origin.
func rotate(q point, theta float64) point {
	s, c := math.Sincos(theta)
	return point{q.x*c - q.y*s, q.x*s + q.y*c}
}

// mirroredInvolute evaluates the base-circle involute at radius r and mirrors
// it across X.
func mirroredInvolute(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	sin, cos := math.Sincos(t)
	return point{base * (cos + t*sin), -(base * (sin - t*cos))}
}

// flankBeta is the rotation that places the pitch crossing of the left flank
// at pi/(2*effectiveTeeth).
func flankBeta(pitch, base, effectiveTeeth float64) float64 {
	m := mirroredInvolute(base, pitch)
	return math.Pi/(2*effectiveTeeth) - math.Atan2(m.y, m.x)
}

// sampleRange samples both flanks at samples radii running linearly from
// from to to, then rotates them by angle. Both arrays run from from to to.
func sampleRange(pitch, base, from, to, effectiveTeeth float64, samples int,
	angle float64) (left, right []point) {
	beta := flankBeta(pitch, base, effectiveTeeth)
	left = make([]point, 0, samples)
	right = make([]point, 0, samples)
	for i := 0; i < samples; i++ {
		r := from + (to-from)*float64(i)/float64(samples-1)
		l := rotate(mirroredInvolute(base, r), beta)
		rt := point{l.x, -l.y}
		left = append(left, rotate(l, angle))
		right = append(right, rotate(rt, angle))
	}
	return left, right
}

// sampleFlanks samples both flanks from the base to the tip radius.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) (left, right []point) {
	return sampleRange(pitch, base, base, tip, effectiveTeeth, samples, angle)
}

// toothAngle is the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// InvoluteFlanks samples the left and right involute flanks of one tooth, each
// from the base radius to the tip radius, rotated by angle. effectiveTeeth is
// used unrounded in place of the tooth count.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip, "effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: %s %v is not finite", name, v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("involute flanks: base %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("involute flanks: pitch %v must exceed base %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("involute flanks: tip %v must exceed pitch %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("involute flanks: effectiveTeeth %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("involute flanks: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	toPairs := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, q := range in {
			out[i] = [2]float64{q.x, q.y}
		}
		return out
	}
	return toPairs(left), toPairs(right), nil
}
