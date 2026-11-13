package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

type point struct{ x, y float64 }

type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	delta := backlash / (2 * pitch)
	pitchPoint := involute(base, pitch)
	py := -pitchPoint.y
	beta := math.Pi/(2*teeth) - delta - math.Atan2(py, pitchPoint.x)
	left, right := make([]point, samples), make([]point, samples)
	for i := range samples {
		radius := start + (tip-start)*float64(i)/float64(samples-1)
		p := involute(base, radius)
		p.y = -p.y
		p = rotate(p.x, p.y, beta)
		left[i] = rotate(p.x, p.y, angle)
		right[i] = rotate(p.x, -p.y, angle)
	}
	return left, right
}

// InvoluteFlanks returns left and right base-to-tip flank samples.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute inputs must be finite")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute flank inputs")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	left, right := make([][2]float64, samples), make([][2]float64, samples)
	for i := range l {
		left[i], right[i] = [2]float64{l[i].x, l[i].y}, [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
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
