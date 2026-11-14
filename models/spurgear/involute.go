package spurgear

import (
	"fmt"
	"math"
)

type point struct{ x, y float64 }

type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func derive(p Params) dimensions {
	doc, err := decadNewForDerive(p)
	if err != nil {
		return dimensions{}
	}
	d, err := readDimensions(doc)
	if err != nil {
		return dimensions{}
	}
	return d
}

func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	left, right := make([]point, samples), make([]point, samples)
	delta := backlash / (2 * pitch)
	invPitch := math.Tan(math.Acos(base/pitch)) - math.Acos(base/pitch)
	beta := math.Pi/(2*teeth) - delta + invPitch
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		t := math.Tan(math.Acos(base / r))
		// The mirrored involute is rotated by beta; the positive-y side is left.
		x := base * (math.Cos(t) + t*math.Sin(t))
		y := -base * (math.Sin(t) - t*math.Cos(t))
		l := rotate(x, y, beta)
		rightFlank := rotate(l.x, -l.y, 0)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rightFlank.x, rightFlank.y, angle)
	}
	return left, right
}

// InvoluteFlanks returns left and right base-to-tip samples.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute input must be finite")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute dimensions or sample count")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	outL, outR := make([][2]float64, samples), make([][2]float64, samples)
	for i := range l {
		outL[i] = [2]float64{l[i].x, l[i].y}
		outR[i] = [2]float64{r[i].x, r[i].y}
	}
	return outL, outR, nil
}
