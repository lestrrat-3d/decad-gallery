package spurgear

import (
	"fmt"
	"math"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii of a gear.
type dimensions struct{ pitch, base, root, tip float64 }

// derive computes the pitch, base, root, and tip radii without rounding.
func derive(p Params) dimensions {
	m := p.Module
	n := float64(p.Teeth)
	pitch := m * n / 2
	return dimensions{
		pitch: pitch,
		base:  pitch * math.Cos(p.PressureAngle),
		root:  (m*n - 2.5*m) / 2,
		tip:   (m*n + 2*m) / 2,
	}
}

// filletRadius returns the root fillet radius Rf. A nonpositive value means
// that no fillet is applied.
func filletRadius(p Params, d dimensions) float64 {
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*(math.Tan(a)-a)) / 2
}

// rotate turns (x, y) counterclockwise by theta.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// involuteAt evaluates the original involute of base radius rb at radius r.
func involuteAt(rb, r float64) point {
	t := math.Tan(math.Acos(rb / r))
	return point{
		rb * (math.Cos(t) + t*math.Sin(t)),
		rb * (math.Sin(t) - t*math.Cos(t)),
	}
}

// sampleFlanks samples the left and right flanks from base to tip. The tooth
// is centered on the +X axis before the final rotation by angle.
func sampleFlanks(pitch, base, tip, teeth float64, samples int, angle float64) (left, right []point) {
	pi := involuteAt(base, pitch)
	beta := math.Pi/(2*teeth) - math.Atan2(-pi.y, pi.x)

	left = make([]point, samples)
	right = make([]point, samples)
	for i := 0; i < samples; i++ {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		s := involuteAt(base, r)
		l := rotate(s.x, -s.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples one tooth's left and right involute flanks from the
// base radius to the tip radius, rotated by angle. effectiveTeeth may be
// fractional.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip, "effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: %s must be finite", name)
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
	l, r := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	toPairs := func(ps []point) [][2]float64 {
		out := make([][2]float64, len(ps))
		for i, p := range ps {
			out[i] = [2]float64{p.x, p.y}
		}
		return out
	}
	return toPairs(l), toPairs(r), nil
}
