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

// filletRadius returns the root fillet radius Rf. It may be nonpositive or
// nonfinite; callers decide whether to fillet.
func filletRadius(p Params) float64 {
	d := derive(p)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*(math.Tan(a)-a)) / 2
}

// rotate rotates (x, y) about the origin by theta radians.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// involutePoint evaluates the base-circle involute at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// sampleFlanks returns the left and right flank samples, both running base to
// tip, rotated by angle. effectiveTeeth replaces the tooth count and is never
// rounded.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) (left, right []point) {
	mirrored := func(r float64) point {
		q := involutePoint(base, r)
		return point{q.x, -q.y}
	}
	pm := mirrored(pitch)
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(pm.y, pm.x)

	left = make([]point, samples)
	right = make([]point, samples)
	for i := 0; i < samples; i++ {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		m := mirrored(r)
		l := rotate(m.x, m.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples a tooth's left and right involute flanks from the
// base radius to the tip radius, inclusive, rotated by angle. Both returned
// arrays run base to tip; the left array comes first.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip,
		"effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: %s must be finite, got %v", name, v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius must be positive, got %v", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth must be positive, got %v", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples must be at least 3, got %d", samples)
	}
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	toPairs := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, q := range in {
			out[i] = [2]float64{q.x, q.y}
		}
		return out
	}
	return toPairs(left), toPairs(right), nil
}
