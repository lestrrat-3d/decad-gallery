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

// filletRadius returns the root fillet radius Rf.
func filletRadius(p Params, d dimensions) float64 {
	return 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
}

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// rotate turns (x, y) by theta about the origin.
func rotate(theta, x, y float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// involutePoint evaluates the original involute of the base circle at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankShift returns the rotation that places the mirrored involute's pitch
// crossing at pi/(2*effectiveTeeth).
func flankShift(pitch, base, effectiveTeeth float64) float64 {
	pp := involutePoint(base, pitch)
	return math.Pi/(2*effectiveTeeth) - math.Atan2(-pp.y, pp.x)
}

// sampleFlanks samples the left and right flanks from radius first to tip,
// both endpoints included, then rotates both by angle. Both arrays run
// base-to-tip (first-to-tip).
func sampleFlanks(pitch, base, first, tip, effectiveTeeth float64, samples int,
	angle float64) (left, right []point) {
	beta := flankShift(pitch, base, effectiveTeeth)
	left = make([]point, samples)
	right = make([]point, samples)
	for i := 0; i < samples; i++ {
		r := first + (tip-first)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(beta, ip.x, -ip.y)
		right0 := point{l.x, -l.y}
		left[i] = rotate(angle, l.x, l.y)
		right[i] = rotate(angle, right0.x, right0.y)
	}
	return left, right
}

// halfAngle is h(r): the angle between the tooth axis and the left flank where
// the flank meets radius r.
func halfAngle(pitch, base, effectiveTeeth, r float64) float64 {
	beta := flankShift(pitch, base, effectiveTeeth)
	ip := involutePoint(base, r)
	l := rotate(beta, ip.x, -ip.y)
	return math.Atan2(l.y, l.x)
}

// InvoluteFlanks samples the left and right involute flanks between the base
// and tip radii with a caller-supplied (possibly fractional) tooth count, and
// rotates both by angle. Both returned arrays run base-to-tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks require finite inputs, got %v", v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute base radius %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective tooth count %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	outLeft := make([][2]float64, samples)
	outRight := make([][2]float64, samples)
	for i := range left {
		outLeft[i] = [2]float64{left[i].x, left[i].y}
		outRight[i] = [2]float64{right[i].x, right[i].y}
	}
	return outLeft, outRight, nil
}
