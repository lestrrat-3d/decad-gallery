package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
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
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	left := make([]point, samples)
	right := make([]point, samples)
	delta := backlash / (2 * pitch)
	pitchT := math.Tan(math.Acos(base / pitch))
	px := base * (math.Cos(pitchT) + pitchT*math.Sin(pitchT))
	py := -base * (math.Sin(pitchT) - pitchT*math.Cos(pitchT))
	beta := math.Pi/(2*teeth) - delta - math.Atan2(py, px)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		t := math.Tan(math.Acos(base / r))
		x := base * (math.Cos(t) + t*math.Sin(t))
		y := -base * (math.Sin(t) - t*math.Cos(t))
		l := rotate(x, y, beta)
		left[i] = l
		right[i] = rotate(l.x, -l.y, angle)
		left[i] = rotate(l.x, l.y, angle)
	}
	return left, right
}

func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	values := []float64{pitch, base, tip, effectiveTeeth, angle}
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flank inputs must be finite")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute flank inputs")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	convert := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, p := range in {
			out[i] = [2]float64{p.x, p.y}
		}
		return out
	}
	return convert(l), convert(r), nil
}

func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}
