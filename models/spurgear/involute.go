package spurgear

import (
	"fmt"
	"math"
)

type point struct{ x, y float64 }

type dimensions struct{ pitch, base, root, tip float64 }

func derive(p Params) dimensions {
	pitch := p.Module * float64(p.Teeth) / 2
	return dimensions{
		pitch: pitch,
		base:  pitch * math.Cos(p.PressureAngle),
		root:  (p.Module*float64(p.Teeth) - 2.5*p.Module) / 2,
		tip:   (p.Module*float64(p.Teeth) + 2*p.Module) / 2,
	}
}

func filletRadius(p Params, d dimensions) float64 {
	return 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
}

func rotate(v point, angle float64) point {
	s, c := math.Sincos(angle)
	return point{v.x*c - v.y*s, v.x*s + v.y*c}
}

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	s, c := math.Sincos(t)
	return point{base * (c + t*s), base * (s - t*c)}
}

func sampleFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([]point, []point) {
	pitchPoint := involute(base, pitch)
	pitchPoint.y = -pitchPoint.y
	beta := math.Pi/(2*effectiveTeeth) - math.Atan2(pitchPoint.y, pitchPoint.x)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		radius := base + (tip-base)*float64(i)/float64(samples-1)
		v := involute(base, radius)
		v.y = -v.y
		v = rotate(v, beta)
		left[i] = rotate(v, angle)
		right[i] = rotate(point{v.x, -v.y}, angle)
	}
	return left, right
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// InvoluteFlanks samples the left and right flanks from base to tip, then rotates them by angle.
// All scalar arguments must be finite; effectiveTeeth may be nonintegral.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	if !finite(pitch) || !finite(base) || !finite(tip) || !finite(effectiveTeeth) || !finite(angle) {
		return nil, nil, fmt.Errorf("involute flanks: scalar inputs must be finite")
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("involute flanks: require 0 < base < pitch < tip, positive effective teeth, and at least 3 samples")
	}
	left, right := sampleFlanks(pitch, base, tip, effectiveTeeth, samples, angle)
	l := make([][2]float64, samples)
	r := make([][2]float64, samples)
	for i := range samples {
		l[i] = [2]float64{left[i].x, left[i].y}
		r[i] = [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
