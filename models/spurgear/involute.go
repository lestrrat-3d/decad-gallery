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

func filletRadius(p Params, d dimensions) float64 { return d.fillet }

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func involuteAt(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

func flankBeta(pitch, base, teeth float64) float64 {
	p := involuteAt(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(-p.y, p.x)
}

func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	left, right := make([]point, samples), make([]point, samples)
	beta := flankBeta(pitch, base, teeth)
	for i := range samples {
		radius := start + (tip-start)*float64(i)/float64(samples-1)
		p := involuteAt(base, radius)
		p = rotate(p.x, -p.y, beta)
		left[i] = rotate(p.x, p.y, angle)
		right[i] = rotate(p.x, -p.y, angle)
	}
	return left, right
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// InvoluteFlanks returns the left and right base-to-tip samples at angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	if !finite(pitch) || !finite(base) || !finite(tip) || !finite(effectiveTeeth) || !finite(angle) {
		return nil, nil, fmt.Errorf("involute flanks require finite scalar inputs")
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute radii, effective teeth, or samples")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	left, right := make([][2]float64, samples), make([][2]float64, samples)
	for i := range samples {
		left[i], right[i] = [2]float64{l[i].x, l[i].y}, [2]float64{r[i].x, r[i].y}
	}
	return left, right, nil
}
