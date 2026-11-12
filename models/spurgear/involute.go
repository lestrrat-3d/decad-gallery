package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar sketch coordinate pair.
type point struct {
	x, y float64
}

// dimensions holds the derived radii read from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive evaluates the parameter table for p on a scratch document. It returns
// a zero dimensions when the table cannot be defined or read.
func derive(p Params) dimensions {
	d, err := deriveDimensions(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// deriveDimensions is derive with the table error preserved.
func deriveDimensions(p Params) (dimensions, error) {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}, err
	}
	return readDimensions(doc)
}

// readDimensions reads the derived radii and the backlash from doc's table.
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

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a flank base point onto the root circle. Each coordinate
// is multiplied by the root radius before dividing by the base radius.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle is the angular position of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the original (unmirrored) involute of the base
// circle at radius r.
func involutePoint(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	s, c := math.Sincos(t)
	return point{base * (c + t*s), base * (s - t*c)}
}

// flankRotation is the angle β that turns the mirrored involute into the left
// flank, so the flank crosses the pitch circle at π/(2N) - δ.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	pp := involutePoint(base, pitch)
	px, py := pp.x, -pp.y
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, px)
}

// sampleFlanks samples one tooth's left and right flanks from radius start to
// tip, both arrays running from start to tip, turned by angle.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	if samples < 2 {
		return nil, nil
	}
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		if i == samples-1 {
			r = tip
		}
		ip := involutePoint(base, r)
		l := rotate(ip.x, -ip.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// flankHalfAngle is h(r): the angle between the tooth axis and the left flank
// where it meets radius r.
func flankHalfAngle(d dimensions, teeth, r float64) float64 {
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	ip := involutePoint(d.base, r)
	q := rotate(ip.x, -ip.y, beta)
	return math.Atan2(q.y, q.x)
}

// closingRadius chooses the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	if flankHalfAngle(d, teeth, d.base) < limit {
		return d.base, nil
	}
	if flankHalfAngle(d, teeth, d.root) >= limit {
		return 0, fmt.Errorf("spurgear: embedded tooth: adjacent flanks meet at or outside the root circle (radius %g)",
			d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if flankHalfAngle(d, teeth, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// InvoluteFlanks samples one tooth's left and right base-to-tip flanks for the
// given radii and real-valued tooth count, turned by angle. It applies no
// backlash.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []struct {
		name string
		x    float64
	}{
		{"pitch", pitch}, {"base", base}, {"tip", tip},
		{"effectiveTeeth", effectiveTeeth}, {"angle", angle},
	} {
		if math.IsNaN(v.x) || math.IsInf(v.x, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: %s must be finite, got %g", v.name, v.x)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: base radius must be positive, got %g", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: effective teeth must be positive, got %g",
			effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: samples must be at least 3, got %d", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return pairs(left), pairs(right), nil
}

func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i] = [2]float64{p.x, p.y}
	}
	return out
}
