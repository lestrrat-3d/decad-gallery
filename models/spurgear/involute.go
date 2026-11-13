package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii and backlash read from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive evaluates the parameter table on a scratch document. It returns a
// zero dimensions value when the table cannot be defined or read.
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

// readDimensions reads the derived dimensions from the document's table.
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

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x: x*c - y*s, y: x*s + y*c}
}

// rootContact scales a base point onto the root circle. Each coordinate is
// multiplied before it is divided.
func rootContact(basePoint point, d dimensions) point {
	return point{
		x: basePoint.x * d.root / d.base,
		y: basePoint.y * d.root / d.base,
	}
}

// toothAngle returns the angle of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the original involute of the base circle at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{
		x: base * (math.Cos(t) + t*math.Sin(t)),
		y: base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankRotation returns the angle that carries the mirrored involute onto the
// left flank so that it crosses the pitch circle at pi/(2*teeth) - delta.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	ip := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(-ip.y, ip.x)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both endpoints included, then turns them by angle. Both arrays run
// base-to-tip and the left array is returned first.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(ip.x, -ip.y, beta)
		right[i] = rotate(l.x, -l.y, angle)
		left[i] = rotate(l.x, l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples left and right involute flanks from base to tip for
// a caller-supplied pitch, base and tip radius and a real-valued effective
// tooth count. No backlash applies.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip,
		"effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: %s must be finite", name)
		}
	}
	if base <= 0 {
		return nil, nil, fmt.Errorf("involute flanks: base radius %g must be positive", base)
	}
	if pitch <= base {
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	}
	if tip <= pitch {
		return nil, nil, fmt.Errorf("involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	}
	if effectiveTeeth <= 0 {
		return nil, nil, fmt.Errorf("involute flanks: effective teeth %g must be positive", effectiveTeeth)
	}
	if samples < 3 {
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
