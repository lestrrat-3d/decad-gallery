package spurgear

import (
	"fmt"
	"math"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii of a gear.
type dimensions struct{ pitch, base, root, tip float64 }

// derive computes the pitch, base, root and tip radii without rounding.
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

// filletRadius returns the root fillet radius Rf.
func filletRadius(p Params, d dimensions) float64 {
	n := float64(p.Teeth)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
}

// rotate rotates (x, y) about the origin by theta.
func rotate(x, y, theta float64) (float64, float64) {
	s, c := math.Sincos(theta)
	return x*c - y*s, x*s + y*c
}

// involutePoint evaluates the original involute of base radius rb at radius r.
func involutePoint(rb, r float64) (float64, float64) {
	t := math.Tan(math.Acos(rb / r))
	return rb * (math.Cos(t) + t*math.Sin(t)), rb * (math.Sin(t) - t*math.Cos(t))
}

// sampleFlanks returns the left and right base-to-tip flank samples of one
// tooth rotated by angle. effectiveTeeth need not be an integer.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) (left, right [][2]float64) {
	px, py := involutePoint(base, pitch)
	py = -py
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(py, px)

	left = make([][2]float64, samples)
	right = make([][2]float64, samples)
	for i := range samples {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		x, y := involutePoint(base, r)
		y = -y
		lx, ly := rotate(x, y, beta)
		rx, ry := lx, -ly
		lx, ly = rotate(lx, ly, angle)
		rx, ry = rotate(rx, ry, angle)
		left[i] = [2]float64{lx, ly}
		right[i] = [2]float64{rx, ry}
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth from
// base to tip, rotated by angle. effectiveTeeth replaces the tooth count and
// is never rounded.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for name, v := range map[string]float64{
		"pitch": pitch, "base": base, "tip": tip, "effectiveTeeth": effectiveTeeth, "angle": angle,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: %s must be finite", name)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius must be positive, got %g", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth must be positive, got %g", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples must be at least 3, got %d", samples)
	}
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	return left, right, nil
}
