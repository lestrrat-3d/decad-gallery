package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

type point struct{ x, y float64 }
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

func derive(p Params) dimensions {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
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
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(p point, d dimensions) point {
	return point{p.x * d.root / d.base, p.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	s, c := math.Sincos(t)
	return point{base * (c + t*s), base * (s - t*c)}
}

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	p := involute(base, pitch)
	beta := math.Pi/(2*teeth) - backlash/(2*pitch) - math.Atan2(-p.y, p.x)
	left, right := make([]point, samples), make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		q := involute(base, r)
		q = rotate(q.x, -q.y, beta)
		left[i] = rotate(q.x, q.y, angle)
		right[i] = rotate(q.x, -q.y, angle)
	}
	return left, right
}

func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("nonfinite involute input")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute dimensions")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	left, right := make([][2]float64, samples), make([][2]float64, samples)
	for i := range samples {
		left[i], right[i] = [2]float64{l[i].x, l[i].y}, [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
}
