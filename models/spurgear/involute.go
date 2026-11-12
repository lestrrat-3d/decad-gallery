package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in sketch space.
type point struct{ x, y float64 }

// dimensions holds the derived gear radii, the root fillet radius, and the
// backlash, all in millimeters, as read from the parameter table.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// derive evaluates the parameter table of p on a scratch document. It returns
// a zero dimensions when the table cannot be defined or read.
func derive(p Params) dimensions {
	d, err := deriveChecked(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// deriveChecked is derive with its error reported.
func deriveChecked(p Params) (dimensions, error) {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}, err
	}
	return readDimensions(doc)
}

// readDimensions reads the derived dimensions from the document's parameter
// table, keeping each value in its base unit.
func readDimensions(doc *decad.Document) (dimensions, error) {
	read := func(name string) (float64, error) {
		v, err := doc.Param(name)
		if err != nil {
			return 0, fmt.Errorf("spurgear: read parameter %s: %w", name, err)
		}
		return v.Base(), nil
	}
	var d dimensions
	var err error
	if d.pitch, err = read("pitch_radius"); err != nil {
		return dimensions{}, err
	}
	if d.base, err = read("base_radius"); err != nil {
		return dimensions{}, err
	}
	if d.root, err = read("root_radius"); err != nil {
		return dimensions{}, err
	}
	if d.tip, err = read("tip_radius"); err != nil {
		return dimensions{}, err
	}
	if d.fillet, err = read("fillet_radius"); err != nil {
		return dimensions{}, err
	}
	if d.backlash, err = read("backlash"); err != nil {
		return dimensions{}, err
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
	return point{x*cos - y*sin, x*sin + y*cos}
}

// toothAngle returns the angular position of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of the base circle at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(math.Min(1, base/r)))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankBeta returns the rotation that places the mirrored involute so the
// left flank crosses the pitch circle at π/(2·teeth) − δ, with δ = backlash/(2·pitch).
func flankBeta(pitch, base, teeth, backlash float64) float64 {
	ip := involutePoint(base, pitch)
	px, py := ip.x, -ip.y
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, px)
}

// flankHalfAngle returns the polar angle of the unrotated left flank at radius r.
func flankHalfAngle(base, r, beta float64) float64 {
	ip := involutePoint(base, r)
	q := rotate(ip.x, -ip.y, beta)
	return math.Atan2(q.y, q.x)
}

// sampleFlanks samples one tooth's left and right flanks from radius start to
// tip, both arrays running from start to tip, and turns them by angle.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankBeta(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(ip.x, -ip.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// closingRadius returns the radius of the arc that closes an embedded tooth
// section: the base radius when adjacent flanks stay apart there, otherwise
// the midpoint between the radius where they meet and the root radius.
func closingRadius(p Params, d dimensions) (float64, error) {
	limit := math.Pi / float64(p.Teeth)
	beta := flankBeta(d.pitch, d.base, float64(p.Teeth), d.backlash)
	if flankHalfAngle(d.base, d.base, beta) < limit {
		return d.base, nil
	}
	if flankHalfAngle(d.base, d.root, beta) >= limit {
		return 0, fmt.Errorf("spurgear: adjacent tooth flanks meet at or outside the root circle (radius %g)",
			d.root)
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if flankHalfAngle(d.base, mid, beta) >= limit {
			lo = mid
			continue
		}
		hi = mid
	}
	return ((lo+hi)/2 + d.root) / 2, nil
}

// InvoluteFlanks samples one unbacklashed tooth's left and right involute
// flanks from base to tip, using a real-valued effective tooth count, and turns
// them by angle. Both arrays run base to tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flank inputs must be finite")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: base radius must be positive, got %g", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: effective teeth must be positive, got %g", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: samples must be at least 3, got %d", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return pairs(left), pairs(right), nil
}

func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, pt := range pts {
		out[i] = [2]float64{pt.x, pt.y}
	}
	return out
}
