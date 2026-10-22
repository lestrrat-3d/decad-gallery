package spurgear

import (
	"fmt"
	"math"
)

// point is a planar coordinate in the gear frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii of a gear.
type dimensions struct{ pitch, base, root, tip float64 }

// derive computes the radii without rounding.
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

// filletRadius returns the root fillet radius Rf.
func filletRadius(p Params, d dimensions) float64 {
	n := float64(p.Teeth)
	a := p.PressureAngle
	return 0.9 * d.root * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
}

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// rotate turns (x, y) counterclockwise by theta.
func rotate(x, y, theta float64) (float64, float64) {
	s, c := math.Sincos(theta)
	return x*c - y*s, x*s + y*c
}

// scaleToRoot locates the root contact of a base sample. It multiplies before
// dividing so the result matches the helical join bit for bit.
func scaleToRoot(d dimensions, x, y float64) point {
	return point{x * d.root / d.base, y * d.root / d.base}
}

// sampleFlanks samples the left and right involute flanks, base to tip, for a
// tooth rotated by angle. teeth is the effective (possibly fractional) count.
func sampleFlanks(pitch, base, tip, teeth float64, samples int, angle float64) (left, right [][2]float64) {
	mirrored := func(r float64) (float64, float64) {
		t := math.Tan(math.Acos(base / r))
		x := base * (math.Cos(t) + t*math.Sin(t))
		y := base * (math.Sin(t) - t*math.Cos(t))
		return x, -y
	}
	px, py := mirrored(pitch)
	beta := math.Pi/(2*teeth) - math.Atan2(py, px)

	left = make([][2]float64, samples)
	right = make([][2]float64, samples)
	for i := 0; i < samples; i++ {
		r := base + (tip-base)*float64(i)/float64(samples-1)
		mx, my := mirrored(r)
		lx, ly := rotate(mx, my, beta)
		rx, ry := lx, -ly
		left[i][0], left[i][1] = rotate(lx, ly, angle)
		right[i][0], right[i][1] = rotate(rx, ry, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth from
// the base circle to the tip circle, rotated by angle. effectiveTeeth need not
// be an integer. Both arrays run base to tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks require finite inputs")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	return left, right, nil
}
