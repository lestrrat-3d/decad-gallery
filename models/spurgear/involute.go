package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii read from the parameter table.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

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
			return dimensions{}, fmt.Errorf("spurgear: read parameter %s: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// rootContact scales a base-circle point onto the root circle. Each
// coordinate is multiplied before it is divided.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// involuteAt returns the original involute point at radius r.
func involuteAt(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	s, c := math.Sincos(t)
	return point{base * (c + t*s), base * (s - t*c)}
}

// flankBeta returns the rotation that places the mirrored involute's pitch
// crossing at pi/(2*teeth) - backlash/(2*pitch).
func flankBeta(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	ip := involuteAt(base, pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(-ip.y, ip.x)
}

// leftFlankPoint returns the unrotated-by-tooth left flank point at radius r.
func leftFlankPoint(base, r, beta float64) point {
	ip := involuteAt(base, r)
	return rotate(ip.x, -ip.y, beta)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both endpoints included, and rotates both by angle.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankBeta(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		switch i {
		case 0:
			r = start
		case samples - 1:
			r = tip
		}
		m := leftFlankPoint(base, r, beta)
		left[i] = rotate(m.x, m.y, angle)
		right[i] = rotate(m.x, -m.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks from base to tip
// using real-valued effectiveTeeth, with no backlash, and rotates both by
// angle. Both arrays run base to tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks need finite inputs")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute effective teeth %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute samples %d must be at least 3", samples)
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
