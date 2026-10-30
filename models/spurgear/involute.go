package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

type point struct{ x, y float64 }
type dimensions struct{ pitch, base, root, tip, fillet float64 }

func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("cap_chamfer: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("bore_tool_length: %w", err)
	}
	return nil
}

func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, entry := range []struct {
		name string
		dest *float64
	}{
		{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root},
		{"tip_radius", &d.tip}, {"fillet_radius", &d.fillet},
	} {
		value, err := doc.Param(entry.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read %s: %w", entry.name, err)
		}
		*entry.dest = value.Base()
	}
	return d, nil
}

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

func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(p point, d dimensions) point {
	return point{p.x * d.root / d.base, p.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

func flankBeta(pitch, base, teeth float64) float64 {
	p := involute(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(-p.y, p.x)
}

func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	left, right := make([]point, samples), make([]point, samples)
	beta := flankBeta(pitch, base, teeth)
	for i := range samples {
		radius := start + (tip-start)*float64(i)/float64(samples-1)
		v := involute(base, radius)
		l := rotate(v.x, -v.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// InvoluteFlanks returns the left and right base-to-tip samples at angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	if !finite(pitch) || !finite(base) || !finite(tip) || !finite(effectiveTeeth) || !finite(angle) {
		return nil, nil, fmt.Errorf("involute inputs must be finite")
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute radii, effective teeth, or sample count")
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	l, r := make([][2]float64, samples), make([][2]float64, samples)
	for i := range samples {
		l[i], r[i] = [2]float64{left[i].x, left[i].y}, [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
