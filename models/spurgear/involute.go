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

// dimensions holds the derived radii, read from the document parameter table.
type dimensions struct{ pitch, base, root, tip, fillet float64 }

// defineParameters writes the gear's named inputs and derived dimensions to t.
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
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2",
		units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// readDimensions reads the derived radii from the document parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	pitch, err := doc.Param("pitch_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read pitch_radius: %w", err)
	}
	base, err := doc.Param("base_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read base_radius: %w", err)
	}
	root, err := doc.Param("root_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read root_radius: %w", err)
	}
	tip, err := doc.Param("tip_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read tip_radius: %w", err)
	}
	fillet, err := doc.Param("fillet_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read fillet_radius: %w", err)
	}
	d.pitch, d.base, d.root, d.tip, d.fillet = pitch.Base(), base.Base(), root.Base(), tip.Base(), fillet.Base()
	return d, nil
}

// derive computes the gear dimensions on a scratch document. It returns the
// zero value when the parameters cannot be evaluated.
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
func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

// rotate applies the planar rotation by theta.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x: x*c - y*s, y: x*s + y*c}
}

// rootContact scales a base-circle point to the root circle, multiplying
// before dividing for each coordinate.
func rootContact(basePoint point, d dimensions) point {
	return point{x: basePoint.x * d.root / d.base, y: basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

// involutePoint evaluates the original involute of base circle radius base at
// radius r.
func involutePoint(base, r float64) point {
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

// flankShift returns the rotation that centers a tooth on the X axis.
func flankShift(pitch, base, teeth float64) float64 {
	pp := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(-pp.y, pp.x)
}

// sampleFlanks samples both flanks of one tooth from radius start to tip. It
// returns the left then the right array, both running base-to-tip, rotated by
// angle.
func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	beta := flankShift(pitch, base, teeth)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(ip.x, -ip.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// halfAngle returns the angle between the tooth axis and the left flank at
// radius r.
func halfAngle(pitch, base, teeth, r float64) float64 {
	beta := flankShift(pitch, base, teeth)
	ip := involutePoint(base, r)
	l := rotate(ip.x, -ip.y, beta)
	return math.Atan2(l.y, l.x)
}

// InvoluteFlanks samples the left and right involute flanks of one tooth from
// the base radius to the tip radius. Both arrays run base-to-tip and are
// rotated by angle. effectiveTeeth may be non-integral.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: flank inputs must be finite")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective tooth count %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples %d must be at least 3", samples)
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
