package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in sketch (frame) space.
type point struct{ x, y float64 }

// dimensions holds the derived radii (mm) and backlash of one gear.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// derive evaluates the parameter table of p on a scratch document. It returns
// a zero dimensions value when the table cannot be defined or read.
func derive(p Params) dimensions {
	d, err := deriveDimensions(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// deriveDimensions is derive with its error preserved.
func deriveDimensions(p Params) (dimensions, error) {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}, err
	}
	return readDimensions(doc)
}

// readDimensions reads the derived dimensions from the document's table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	fields := []struct {
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
	for _, f := range fields {
		v, err := doc.Param(f.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("spurgear: read parameter %s: %w", f.name, err)
		}
		*f.dst = v.Base()
	}
	return d, nil
}

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	sin, cos := math.Sincos(theta)
	return point{x: x*cos - y*sin, y: x*sin + y*cos}
}

// rootContact scales a flank base point onto the root circle. Each coordinate
// is multiplied by the root radius before dividing by the base radius.
func rootContact(basePoint point, d dimensions) point {
	return point{x: basePoint.x * d.root / d.base, y: basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of the base circle at radius r.
func involutePoint(base, r float64) point {
	c := base / r
	if c > 1 {
		c = 1
	}
	t := math.Tan(math.Acos(c))
	sin, cos := math.Sincos(t)
	return point{x: base * (cos + t*sin), y: base * (sin - t*cos)}
}

// mirroredInvolute is the involute at r mirrored across X.
func mirroredInvolute(base, r float64) point {
	q := involutePoint(base, r)
	return point{x: q.x, y: -q.y}
}

// flankBeta is the rotation that places the mirrored involute's pitch crossing
// at pi/(2*teeth) - delta, where delta = backlash/(2*pitch).
func flankBeta(pitch, base, teeth, backlash float64) float64 {
	pp := mirroredInvolute(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// halfAngle is h(r): the angle between the tooth axis and the left flank at
// radius r, using the same rotation as sampleFlanks.
func halfAngle(pitch, base, teeth, backlash, r float64) float64 {
	beta := flankBeta(pitch, base, teeth, backlash)
	m := mirroredInvolute(base, r)
	q := rotate(m.x, m.y, beta)
	return math.Atan2(q.y, q.x)
}

// sampleFlanks samples one tooth's flanks from radius start to tip, both
// endpoints included, and turns them by angle. It returns the left then the
// right base-to-tip arrays.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64,
) ([]point, []point) {
	beta := flankBeta(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := mirroredInvolute(base, r)
		l := rotate(m.x, m.y, beta)
		rt := point{x: l.x, y: -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples involute flanks for callers with their own radii and
// a real-valued effective tooth count. It applies no backlash and returns the
// left then the right base-to-tip arrays, turned by angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64,
) ([][2]float64, [][2]float64, error) {
	for _, v := range []struct {
		name string
		val  float64
	}{
		{"pitch", pitch},
		{"base", base},
		{"tip", tip},
		{"effectiveTeeth", effectiveTeeth},
		{"angle", angle},
	} {
		if math.IsNaN(v.val) || math.IsInf(v.val, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: %s must be finite, got %v", v.name, v.val)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: base radius must be positive, got %v", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: effective teeth must be positive, got %v", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: samples must be at least 3, got %d", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return toPairs(left), toPairs(right), nil
}

func toPairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, q := range pts {
		out[i] = [2]float64{q.x, q.y}
	}
	return out
}
