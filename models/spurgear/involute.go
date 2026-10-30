package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

type point struct{ x, y float64 }

type dimensions struct{ pitch, base, root, tip, fillet float64 }

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
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	pc := involute(base, pitch)
	beta := math.Pi/(2*teeth) - math.Atan2(-pc.y, pc.x)
	left, right := make([]point, samples), make([]point, samples)
	for i := range samples {
		radius := start + (tip-start)*float64(i)/float64(samples-1)
		v := involute(base, radius)
		v = rotate(v.x, -v.y, beta)
		left[i] = rotate(v.x, v.y, angle)
		right[i] = rotate(v.x, -v.y, angle)
	}
	return left, right
}

// InvoluteFlanks returns the left and right base-to-tip samples at angle radians.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, value := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, nil, fmt.Errorf("involute arguments must be finite")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute radii, effective teeth, or samples")
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	l, r := make([][2]float64, samples), make([][2]float64, samples)
	for i := range samples {
		l[i], r[i] = [2]float64{left[i].x, left[i].y}, [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
