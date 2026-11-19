package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
)

type point struct{ x, y float64 }
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func involute(r, base float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	left, right := make([]point, samples), make([]point, samples)
	p := involute(pitch, base)
	py := -p.y
	beta := math.Pi/(2*teeth) - backlash/(2*pitch) - math.Atan2(py, p.x)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		v := involute(r, base)
		v.y = -v.y
		l := rotate(v.x, v.y, beta)
		rp := rotate(l.x, -l.y, 0)
		l = rotate(l.x, l.y, angle)
		rp = rotate(rp.x, rp.y, angle)
		left[i], right[i] = l, rp
	}
	return left, right
}

func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("nonfinite involute argument")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute arguments")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	out := func(in []point) [][2]float64 {
		a := make([][2]float64, len(in))
		for i, p := range in {
			a[i] = [2]float64{p.x, p.y}
		}
		return a
	}
	return out(l), out(r), nil
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
