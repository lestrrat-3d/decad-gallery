package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a plane-local coordinate pair in millimetres.
type point struct{ x, y float64 }

// dimensions holds the derived radii, read back from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// defineParameters writes the spur gear parameter table into t.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "module", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "teeth", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "pressure_angle", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "thickness", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "samples", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "bore_diameter", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "cap_chamfer", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("define parameter %q: %w", "backlash", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "tool_margin", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "pitch_radius", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "base_radius", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "root_radius", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "tip_radius", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("define parameter %q: %w", "involute_angle", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "fillet_radius", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("define parameter %q: %w", "bore_tool_length", err)
	}
	return nil
}

// readDimensions reads the derived dimensions from the document's parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	read := func(name string) (float64, error) {
		v, err := doc.Param(name)
		if err != nil {
			return 0, fmt.Errorf("read parameter %q: %w", name, err)
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

// derive computes the gear dimensions on a scratch document. It returns the
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

func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta radians about the origin.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base point out to the root circle, multiplying before
// dividing.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle is the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// mirroredInvolute evaluates the involute of base radius base at radius r and
// mirrors it across X.
func mirroredInvolute(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	x := base * (math.Cos(t) + t*math.Sin(t))
	y := base * (math.Sin(t) - t*math.Cos(t))
	return point{x, -y}
}

// flankBeta is the rotation that places the mirrored involute so its pitch
// crossing sits at pi/(2*teeth) - backlash/(2*pitch).
func flankBeta(pitch, base, teeth, backlash float64) float64 {
	pp := mirroredInvolute(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// sampleFlanks samples the left and right involute flanks from radius start to
// radius tip, base-to-tip, rotated by angle. It returns left then right.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankBeta(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := mirroredInvolute(base, r)
		l := rotate(m.x, m.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks, base-to-tip, for
// the given radii and a real-valued effective tooth count. It applies no
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
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	left := make([][2]float64, len(l))
	right := make([][2]float64, len(r))
	for i := range l {
		left[i] = [2]float64{l[i].x, l[i].y}
		right[i] = [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
}
