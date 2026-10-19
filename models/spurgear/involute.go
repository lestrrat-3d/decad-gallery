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

// filletRadius returns the root fillet radius Rf for the gear.
func filletRadius(p Params, d dimensions) float64 {
	n := float64(p.Teeth)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
}

// rotate turns (x, y) counterclockwise by theta.
func rotate(x, y, theta float64) (float64, float64) {
	s, c := math.Sincos(theta)
	return x*c - y*s, x*s + y*c
}

// involutePoint evaluates the involute of the base circle at radius r.
func involutePoint(base, r float64) (float64, float64) {
	t := math.Tan(math.Acos(base / r))
	s, c := math.Sincos(t)
	return base * (c + t*s), base * (s - t*c)
}

// sampleFlanks samples the left and right flanks, both base-to-tip, for a
// tooth rotated by angle. effectiveTeeth replaces the tooth count.
func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) (left, right []point) {
	px, py := involutePoint(base, pitch)
	py = -py
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(py, px)

	left = make([]point, samples)
	right = make([]point, samples)
	for i := 0; i < samples; i++ {
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

// InvoluteFlanks samples the left and right involute flanks of one tooth, each
// running from the base radius to the tip radius, rotated by angle. The tooth
// count may be non-integer so that virtual tooth counts can be sampled.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: nonfinite input %v", v)
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
	left := make([][2]float64, len(l))
	right := make([][2]float64, len(r))
	for i := range l {
		left[i] = [2]float64{l[i].x, l[i].y}
		right[i] = [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
}
