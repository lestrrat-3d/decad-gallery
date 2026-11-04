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

// defineParameters writes the gear's inputs and derived dimensions into t.
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

// rotate turns (x, y) counter-clockwise by theta.
func rotate(x, y, theta float64) point {
	sin, cos := math.Sincos(theta)
	return point{x*cos - y*sin, x*sin + y*cos}
}

// rootContact scales a base-circle point onto the root circle. Each
// coordinate is multiplied before it is divided.
func rootContact(basePoint point, d dimensions) point {
	return point{
		basePoint.x * d.root / d.base,
		basePoint.y * d.root / d.base,
	}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of the base circle at radius r.
func involutePoint(base, r float64) (float64, float64) {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	return base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))
}

// flankOffset returns the rotation that places the pitch crossing of the
// left flank at pi/(2*teeth).
func flankOffset(pitch, base, teeth float64) float64 {
	px, py := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(-py, px)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both included, rotated by angle. Both arrays run base-to-tip.
func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	beta := flankOffset(pitch, base, teeth)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		x, y := involutePoint(base, r)
		l := rotate(x, -y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks, base to tip,
// for a gear with effectiveTeeth teeth, rotated by angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip,
		"effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: %s must be finite, got %v", name, v)
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
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	l := make([][2]float64, len(left))
	r := make([][2]float64, len(right))
	for i := range left {
		l[i] = [2]float64{left[i].x, left[i].y}
		r[i] = [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
