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

func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, item := range []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root},
		{"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash},
	} {
		v, err := doc.Param(item.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read %s: %w", item.name, err)
		}
		*item.dst = v.Base()
	}
	return d, nil
}

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
	inv := func(r float64) point {
		t := math.Tan(math.Acos(base / r))
		return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
	}
	pitchPoint := inv(pitch)
	py := -pitchPoint.y
	beta := math.Pi/(2*teeth) - delta - math.Atan2(py, pitchPoint.x)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		v := inv(r)
		v.y = -v.y
		l := rotate(v.x, v.y, beta)
		rp := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rp.x, rp.y, angle)
	}
	return left, right
}

func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	if !finite(pitch) || !finite(base) || !finite(tip) || !finite(effectiveTeeth) || !finite(angle) || base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute flank arguments")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	outL, outR := make([][2]float64, len(l)), make([][2]float64, len(r))
	for i := range l {
		outL[i] = [2]float64{l[i].x, l[i].y}
		outR[i] = [2]float64{r[i].x, r[i].y}
	}
	return outL, outR, nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
