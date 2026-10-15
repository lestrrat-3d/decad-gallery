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
	mn := p.Module * float64(p.Teeth)
	pitch := mn / 2
	return dimensions{
		pitch: pitch,
		base:  pitch * math.Cos(p.PressureAngle),
		root:  (mn - 2.5*p.Module) / 2,
		tip:   (mn + 2*p.Module) / 2,
	}
}

// filletRadius returns the root fillet radius Rf. It may be nonpositive or
// nonfinite; callers decide whether to fillet.
func filletRadius(p Params, d dimensions) float64 {
	n := float64(p.Teeth)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
}

// rotate rotates (x, y) about the origin by an angle with the given cosine and sine.
func rotate(x, y, c, s float64) (float64, float64) {
	return x*c - y*s, x*s + y*c
}

// involutePoint evaluates the original involute of a base circle at radius r.
func involutePoint(base, r float64) (float64, float64) {
	t := math.Tan(math.Acos(base / r))
	return base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))
}

// sampleFlanks returns the left and right base-to-tip flank arrays, both
// rotated by angle. effectiveTeeth replaces the tooth count in the pitch crossing.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([]point, []point) {
	px, py := involutePoint(base, pitch)
	py = -py
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(py, px)
	cb, sb := math.Cos(beta), math.Sin(beta)
	ct, st := math.Cos(angle), math.Sin(angle)

	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		x, y := involutePoint(base, r)
		y = -y
		lx, ly := rotate(x, y, cb, sb)
		rx, ry := lx, -ly
		lx, ly = rotate(lx, ly, ct, st)
		rx, ry = rotate(rx, ry, ct, st)
		left[i] = point{lx, ly}
		right[i] = point{rx, ry}
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth,
// each from base to tip, with the supplied radii and a real-valued effective
// tooth count. Both arrays are rotated by the signed angle (radians).
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks require finite scalar inputs")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	return pairs(left), pairs(right), nil
}

func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i] = [2]float64{p.x, p.y}
	}
	return out
}
