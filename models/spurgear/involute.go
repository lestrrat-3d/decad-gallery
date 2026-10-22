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
func filletRadius(p Params) float64 {
	d := derive(p)
	return 0.9 * d.root * (math.Pi/float64(p.Teeth) -
		2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
}

// rotate turns a planar point counterclockwise by theta.
func rotate(pt point, theta float64) point {
	s, c := math.Sincos(theta)
	return point{pt.x*c - pt.y*s, pt.x*s + pt.y*c}
}

// involutePoint evaluates the original involute at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	s, c := math.Sincos(t)
	return point{base * (c + t*s), base * (s - t*c)}
}

// sampleFlanks returns the left and right base-to-tip flank samples for a
// tooth rotated by angle. effectiveTeeth may be non-integral.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) (left, right []point) {
	ip := involutePoint(base, pitch)
	ip.y = -ip.y
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(ip.y, ip.x)

	left = make([]point, samples)
	right = make([]point, samples)
	for i := 0; i < samples; i++ {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		m := involutePoint(base, r)
		m.y = -m.y
		l := rotate(m, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l, angle)
		right[i] = rotate(rt, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth,
// each from base to tip, rotated by angle. effectiveTeeth replaces the tooth
// count and is never rounded.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: nonfinite scalar input %v", v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: base radius %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: effective teeth %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: samples %d must be at least 3", samples)
	}
	l, r := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	toPairs := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, pt := range in {
			out[i] = [2]float64{pt.x, pt.y}
		}
		return out
	}
	return toPairs(l), toPairs(r), nil
}
