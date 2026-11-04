package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// point is a planar coordinate in the gear frame, in millimeters.
type point struct{ x, y float64 }

// dimensions holds the derived radii read from the parameter table.
type dimensions struct{ pitch, base, root, tip, fillet float64 }

// defineParameters writes the gear's inputs and derived dimensions to t.
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
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// readDimensions reads the derived radii from doc's parameter table.
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

// derive computes the dimensions of p on a scratch document. It returns the
// zero value when the parameter table cannot be evaluated.
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
func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

// rotate turns (x, y) counterclockwise by theta.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x: x*c - y*s, y: x*s + y*c}
}

// rootContact scales a base-circle point onto the root circle.
func rootContact(basePoint point, d dimensions) point {
	return point{x: basePoint.x * d.root / d.base, y: basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the original involute of the base circle at radius r.
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

// flankShift returns the rotation that puts the left flank's pitch crossing at
// pi/(2*teeth).
func flankShift(pitch, base, teeth float64) float64 {
	ip := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(-ip.y, ip.x)
}

// sampleFlanks samples both flanks of one tooth from radius start to tip. It
// returns the left array and then the right array, each running base to tip.
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

// flankHalfAngle returns the angle between the tooth axis and the left flank at
// radius r.
func flankHalfAngle(pitch, base, teeth, r float64) float64 {
	beta := flankShift(pitch, base, teeth)
	ip := involutePoint(base, r)
	l := rotate(ip.x, -ip.y, beta)
	return math.Atan2(l.y, l.x)
}
