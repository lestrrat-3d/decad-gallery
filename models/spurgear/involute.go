package spurgear

import (
	"fmt"
	"math"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii of a gear.
type dimensions struct{ pitch, base, root, tip float64 }

// derive computes the gear radii without rounding.
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
func filletRadius(p Params, d dimensions) float64 {
	n := float64(p.Teeth)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// rotate rotates (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// involutePoint evaluates the original involute at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

// sampleFlanks returns the left and right flank samples, base to tip, rotated
// by angle. effectiveTeeth replaces the tooth count and is never rounded.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) (left, right []point) {
	ip := involutePoint(base, pitch)
	px, py := ip.x, -ip.y
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(py, px)

	left = make([]point, samples)
	right = make([]point, samples)
	for i := 0; i < samples; i++ {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		s := involutePoint(base, r)
		l := rotate(s.x, -s.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth,
// base to tip, rotated by angle. The returned arrays are left then right.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: nonfinite flank input %v", v)
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
	return pairs(left), pairs(right), nil
}

func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i] = [2]float64{p.x, p.y}
	}
	return out
}
