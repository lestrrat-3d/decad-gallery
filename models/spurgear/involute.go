package spurgear

import (
	"fmt"
	"math"
)

type point struct{ x, y float64 }

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
func filletRadius(p Params) float64 {
	d := derive(p)
	return 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
}

func rotate(x, y, theta float64) (float64, float64) {
	c, s := math.Cos(theta), math.Sin(theta)
	return x*c - y*s, x*s + y*c
}

func involutePoint(base, r float64) (float64, float64) {
	t := math.Tan(math.Acos(base / r))
	return base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))
}

// sampleFlanks returns the left and right flank arrays, base to tip, rotated by angle.
func sampleFlanks(pitch, base, tip, teeth float64, samples int, angle float64) (left, right []point) {
	px, py := involutePoint(base, pitch)
	py = -py
	beta := math.Pi/(2*teeth) - math.Atan2(py, px)
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

// InvoluteFlanks samples left and right involute flanks, base to tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite scalar input")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("involute flanks: invalid arguments base=%g pitch=%g tip=%g teeth=%g samples=%d",
			base, pitch, tip, effectiveTeeth, samples)
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
