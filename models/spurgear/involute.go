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
	var err error
	if err = defineParameters(doc.Params(), p); err != nil {
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

func filletRadius(_ Params, d dimensions) float64 { return d.fillet }
func toothAngle(p Params, k int) float64          { return 2 * math.Pi * float64(k) / float64(p.Teeth) }
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int, angle float64) ([]point, []point) {
	left, right := make([]point, samples), make([]point, samples)
	delta := backlash / (2 * pitch)
	pitchI := involute(base, pitch)
	px, py := pitchI.x, -pitchI.y
	beta := math.Pi/(2*teeth) - delta - math.Atan2(py, px)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		v := involute(base, r)
		v.y = -v.y
		v = rotate(v.x, v.y, beta)
		mirrored := point{v.x, -v.y}
		left[i] = rotate(v.x, v.y, angle)
		right[i] = rotate(mirrored.x, mirrored.y, angle)
	}
	return left, right
}

func involute(base, radius float64) point {
	t := math.Tan(math.Acos(base / radius))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int, angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute inputs must be finite")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("invalid involute inputs")
	}
	l, r := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	out := func(in []point) [][2]float64 {
		a := make([][2]float64, len(in))
		for i, v := range in {
			a[i] = [2]float64{v.x, v.y}
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
