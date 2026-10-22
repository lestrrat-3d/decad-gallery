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
	n := float64(p.Teeth)
	pitch := p.Module * n / 2
	return dimensions{
		pitch: pitch,
		base:  pitch * math.Cos(p.PressureAngle),
		root:  (p.Module*n - 2.5*p.Module) / 2,
		tip:   (p.Module*n + 2*p.Module) / 2,
	}
}

// filletRadius returns the root fillet radius Rf.
func filletRadius(p Params) float64 {
	d := derive(p)
	n := float64(p.Teeth)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
}

// rotate rotates (x, y) by theta about the origin.
func rotate(x, y, theta float64) (float64, float64) {
	s, c := math.Sincos(theta)
	return x*c - y*s, x*s + y*c
}

// involutePoint evaluates the original involute at radius r.
func involutePoint(base, r float64) (float64, float64) {
	t := math.Tan(math.Acos(base / r))
	return base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))
}

// sampleFlanks samples the left and right flanks, both running base to tip,
// rotated by angle. effectiveTeeth may be non-integer.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) (left, right []point) {
	px, py := involutePoint(base, pitch)
	py = -py
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(py, px)

	left = make([]point, samples)
	right = make([]point, samples)
	for i := range samples {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		x, y := involutePoint(base, r)
		y = -y
		lx, ly := rotate(x, y, beta)
		rx, ry := lx, -ly
		lx, ly = rotate(lx, ly, angle)
		rx, ry = rotate(rx, ry, angle)
		left[i] = point{lx, ly}
		right[i] = point{rx, ry}
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks, each running from
// base to tip, for a possibly non-integer effective tooth count, rotated by angle.
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
	l, r := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	toPairs := func(ps []point) [][2]float64 {
		out := make([][2]float64, len(ps))
		for i, q := range ps {
			out[i] = [2]float64{q.x, q.y}
		}
		return out
	}
	return toPairs(l), toPairs(r), nil
}
