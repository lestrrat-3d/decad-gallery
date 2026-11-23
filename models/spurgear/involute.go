package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar sketch coordinate in millimeters.
type point struct{ x, y float64 }

// dimensions holds the derived radii and backlash read from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// readDimensions reads the derived dimensions from the document's parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	targets := []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch},
		{"base_radius", &d.base},
		{"root_radius", &d.root},
		{"tip_radius", &d.tip},
		{"fillet_radius", &d.fillet},
		{"backlash", &d.backlash},
	}
	for _, t := range targets {
		v, err := doc.Param(t.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read parameter %q: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// derive computes the gear dimensions through a scratch document. It returns a
// zero dimensions value when the parameters cannot be defined or read.
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

// filletRadius returns the root fillet radius of the derived dimensions.
func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) about the origin by theta radians.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base-circle point onto the root circle. Each coordinate
// is multiplied by the root radius before it is divided by the base radius.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the angular position of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the original involute of a base circle at radius r.
func involutePoint(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankShift returns the rotation that places the mirrored involute's pitch
// crossing at pi/(2*teeth) - backlash/(2*pitch) from the tooth axis.
func flankShift(pitch, base, teeth, backlash float64) float64 {
	pp := involutePoint(base, pitch)
	py := -pp.y
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, pp.x)
}

// sampleFlanks samples both flanks of one tooth from radius start to tip. It
// returns the left array then the right array, each running start-to-tip, both
// turned by angle. teeth may be non-integral.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankShift(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(ip.x, -ip.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks of one tooth with
// caller-supplied radii and a real-valued effective tooth count. Both arrays
// run from the base circle to the tip circle and are turned by angle. No
// backlash is applied.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite input")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("involute flanks: base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("involute flanks: effective teeth %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("involute flanks: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	toPairs := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, q := range in {
			out[i] = [2]float64{q.x, q.y}
		}
		return out
	}
	return toPairs(left), toPairs(right), nil
}
