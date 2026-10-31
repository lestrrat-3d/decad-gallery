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

func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, entry := range []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch},
		{"base_radius", &d.base},
		{"root_radius", &d.root},
		{"tip_radius", &d.tip},
		{"fillet_radius", &d.fillet},
	} {
		value, err := doc.Param(entry.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read %s: %w", entry.name, err)
		}
		*entry.dst = value.Base()
	}
	return d, nil
}

func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

func flankBeta(pitch, base, teeth float64) float64 {
	p := involute(base, pitch)
	return math.Pi/(2*teeth) - math.Atan2(-p.y, p.x)
}

func sampleFlanks(pitch, base, start, tip, teeth float64, samples int, angle float64) ([]point, []point) {
	beta := flankBeta(pitch, base, teeth)
	left, right := make([]point, samples), make([]point, samples)
	for i := range samples {
		radius := start + (tip-start)*float64(i)/float64(samples-1)
		original := involute(base, radius)
		p := rotate(original.x, -original.y, beta)
		left[i] = rotate(p.x, p.y, angle)
		right[i] = rotate(p.x, -p.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right flanks from base to tip at angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, value := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, nil, fmt.Errorf("involute flanks require finite scalar inputs")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute flank radii, effective teeth, or samples")
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	l, r := make([][2]float64, samples), make([][2]float64, samples)
	for i := range samples {
		l[i], r[i] = [2]float64{left[i].x, left[i].y}, [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
