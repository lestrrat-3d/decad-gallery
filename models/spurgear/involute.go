package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii read from the parameter table.
type dimensions struct{ pitch, base, root, tip, fillet float64 }

// derive evaluates the parameter table for p on a scratch document. It returns
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

// readDimensions reads the derived radii from the document's parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, e := range []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch},
		{"base_radius", &d.base},
		{"root_radius", &d.root},
		{"tip_radius", &d.tip},
		{"fillet_radius", &d.fillet},
	} {
		v, err := doc.Param(e.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read parameter %s: %w", e.name, err)
		}
		*e.dst = v.Base()
	}
	return d, nil
}

// filletRadius returns the derived root fillet radius.
func filletRadius(p Params, d dimensions) float64 { return d.fillet }

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// rotate applies the planar rotation by theta.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base-circle point out to the root circle. Each
// coordinate multiplies before it divides.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// mirroredInvolute returns the involute point at radius r, mirrored across X.
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

// flankShift returns the rotation that centers the pitch crossings at
// plus and minus pi/(2*teeth).
func flankShift(pitch, base, teeth float64) float64 {
	pm := mirroredInvolute(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(pm.y, pm.x)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both endpoints included, and rotates both by angle. Both arrays run
// start-to-tip; the left array is returned first.
func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	beta := flankShift(pitch, base, teeth)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := mirroredInvolute(base, r)
		l := rotate(m.x, m.y, beta)
		rr := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rr.x, rr.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks, base to tip,
// using a real-valued effective tooth count and a signed rotation angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite input")
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
