package spurgear

import (
	"fmt"
	"github.com/lestrrat-3d/decad"
	"math"
)

type point struct{ x, y float64 }
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

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
func filletRadius(p Params, d dimensions) float64 { return d.fillet }
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}
func rootContact(q point, d dimensions) point {
	return point{q.x * d.root / d.base, q.y * d.root / d.base}
}
func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }
func involute(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{base * (math.Cos(t) + t*math.Sin(t)), -base * (math.Sin(t) - t*math.Cos(t))}
}
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	pp := involute(base, pitch)
	beta := math.Pi/(2*teeth) - backlash/(2*pitch) - math.Atan2(pp.y, pp.x)
	left, right := make([]point, samples), make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		q := involute(base, r)
		l := rotate(q.x, q.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// InvoluteFlanks samples the left and right flanks from base to tip, with no
// backlash, and rotates them by angle in the sketch plane. effectiveTeeth may
// be fractional. Radii must satisfy 0 < base < pitch < tip; samples must be
// at least three, effectiveTeeth positive, and scalar inputs finite.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	if !finite(pitch) || !finite(base) || !finite(tip) || !finite(effectiveTeeth) || !finite(angle) || base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute inputs")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	left, right := make([][2]float64, samples), make([][2]float64, samples)
	for i := range l {
		left[i] = [2]float64{l[i].x, l[i].y}
		right[i] = [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
}
