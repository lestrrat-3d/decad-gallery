package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a point in the gear's sketch plane.
type point struct{ x, y float64 }

type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

func derive(p Params) dimensions {
	doc := decad.New()
	if defineParameters(doc.Params(), p) != nil {
		return dimensions{}
	}
	d, err := readDimensions(doc)
	if err != nil {
		return dimensions{}
	}
	return d
}

func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	left, right := make([]point, samples), make([]point, samples)
	delta := backlash / (2 * pitch)
	t := math.Tan(math.Acos(base / pitch))
	pitchAngle := math.Atan2(-base*(math.Sin(t)-t*math.Cos(t)), base*(math.Cos(t)+t*math.Sin(t)))
	beta := math.Pi/(2*teeth) - delta - pitchAngle
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		if r < base {
			r = base
		}
		q := math.Acos(base / r)
		v := math.Tan(q)
		x := base * (math.Cos(v) + v*math.Sin(v))
		y := -base * (math.Sin(v) - v*math.Cos(v))
		l := rotate(x, y, beta)
		rp := rotate(l.x, -l.y, 0)
		left[i], right[i] = rotate(l.x, l.y, angle), rotate(rp.x, rp.y, angle)
	}
	return left, right
}

// InvoluteFlanks returns left and right base-to-tip samples.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	vals := []float64{pitch, base, tip, effectiveTeeth, angle}
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("nonfinite involute input")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute inputs")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	out := func(in []point) [][2]float64 {
		result := make([][2]float64, len(in))
		for i, p := range in {
			result[i] = [2]float64{p.x, p.y}
		}
		return result
	}
	return out(l), out(r), nil
}
