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
func involute(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	s, c := math.Sincos(t)
	return point{base * (c + t*s), -base * (s - t*c)}
}
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	p := involute(base, pitch)
	beta := math.Pi/(2*teeth) - backlash/(2*pitch) - math.Atan2(p.y, p.x)
	left, right := make([]point, samples), make([]point, samples)
	for i := range left {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		p := involute(base, r)
		p = rotate(p.x, p.y, beta)
		left[i] = rotate(p.x, p.y, angle)
		right[i] = rotate(p.x, -p.y, angle)
	}
	return left, right
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	if !finite(pitch) || !finite(base) || !finite(tip) || !finite(effectiveTeeth) || !finite(angle) || base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute inputs")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	left, right := make([][2]float64, len(l)), make([][2]float64, len(r))
	for i := range l {
		left[i] = [2]float64{l[i].x, l[i].y}
		right[i] = [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
}
