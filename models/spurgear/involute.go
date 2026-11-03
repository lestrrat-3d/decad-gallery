package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// dimensions holds the derived radii read from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet float64
}

// derive computes the dimensions on a scratch document. It returns a zero
// dimensions value when the parameter table cannot be defined or read.
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

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate applies the planar rotation by theta about the origin.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x: x*c - y*s, y: x*s + y*c}
}

// rootContact scales a base point to the root circle. Each coordinate is
// multiplied before it is divided.
func rootContact(basePoint point, d dimensions) point {
	return point{x: basePoint.x * d.root / d.base, y: basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// mirroredInvolute returns the involute of the base circle at radius r with
// its Y coordinate negated.
func mirroredInvolute(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	x := base * (math.Cos(t) + t*math.Sin(t))
	y := base * (math.Sin(t) - t*math.Cos(t))
	return point{x: x, y: -y}
}

// flankBeta returns the rotation that places the pitch crossing of the left
// flank at pi/(2*teeth).
func flankBeta(pitch, base, teeth float64) float64 {
	m := mirroredInvolute(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(m.y, m.x)
}

// halfAngle returns the angle between the tooth axis and the left flank where
// it meets radius r.
func halfAngle(pitch, base, teeth, r float64) float64 {
	m := mirroredInvolute(base, r)
	q := rotate(m.x, m.y, flankBeta(pitch, base, teeth))
	return math.Atan2(q.y, q.x)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both endpoints included, and rotates them by angle. Both arrays run from
// start to tip; the left array comes first.
func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	beta := flankBeta(pitch, base, teeth)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		if i == samples-1 {
			r = tip
		}
		m := mirroredInvolute(base, r)
		l := rotate(m.x, m.y, beta)
		rt := point{x: l.x, y: -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth,
// both running from the base radius to the tip radius. effectiveTeeth may be
// non-integer. The flanks are rotated by angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: nonfinite scalar input")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: effective teeth %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: samples %d must be at least 3", samples)
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
