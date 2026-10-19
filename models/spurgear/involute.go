package spurgear

import (
	"fmt"
	"math"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii of a gear.
type dimensions struct{ pitch, base, root, tip float64 }

// derive computes the radii from the gear parameters without rounding.
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

// rotate rotates (x, y) about the origin by theta radians.
func rotate(x, y, theta float64) (float64, float64) {
	s, c := math.Sincos(theta)
	return x*c - y*s, x*s + y*c
}

// involutePoint evaluates the original involute of the base circle at radius r.
func involutePoint(base, r float64) (float64, float64) {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	sin, cos := math.Sincos(t)
	return base * (cos + t*sin), base * (sin - t*cos)
}

// sampleFlanks samples the left and right flank arrays (base to tip) for a
// tooth rotated by angle. effectiveTeeth replaces the tooth count and is never
// rounded.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) (left, right [][2]float64) {
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

// InvoluteFlanks samples one tooth's left and right flanks from base to tip.
// effectiveTeeth replaces the tooth count and may be non-integer; angle rotates
// the tooth. The left array is returned first.
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
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	return left, right, nil
}
