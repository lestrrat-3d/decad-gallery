package spurgear

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar sketch coordinate in millimeters.
type point struct {
	x, y float64
}

// dimensions holds the derived radii and lengths of one spur gear in
// millimeters, read back from the gear's parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive evaluates the gear's parameter table on a scratch document and
// returns its derived dimensions. It returns zero dimensions when the table
// cannot be defined or read.
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

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta radians about the origin.
func rotate(x, y, theta float64) point {
	sin, cos := math.Sincos(theta)
	return point{x: x*cos - y*sin, y: x*sin + y*cos}
}

// rootContact scales a flank's base point onto the root circle. Each
// coordinate is multiplied before it is divided so that the result matches
// the other builders bit for bit.
func rootContact(basePoint point, d dimensions) point {
	return point{x: basePoint.x * d.root / d.base, y: basePoint.y * d.root / d.base}
}

// toothAngle returns the angular position of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of the base circle at radius r.
func involutePoint(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	sin, cos := math.Sincos(t)
	return point{x: base * (cos + t*sin), y: base * (sin - t*cos)}
}

// flankRotation returns the angle β that turns the mirrored involute so that
// it crosses the pitch circle at π/(2N) - δ, where δ = backlash/(2·pitch).
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	q := involutePoint(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(-q.y, q.x)
}

// leftFlankPoint returns the unrotated-tooth left flank point at radius r.
func leftFlankPoint(base, r, beta float64) point {
	q := involutePoint(base, r)
	return rotate(q.x, -q.y, beta)
}

// sampleFlanks samples the left and right flanks of one tooth from radius
// start to radius tip, both endpoints included, and turns them by angle.
// Both arrays run from start to tip.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	last := float64(samples - 1)
	for i := range samples {
		r := start + (tip-start)*float64(i)/last
		l := leftFlankPoint(base, r, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// flankHalfAngle returns h(r), the angle between the tooth axis and the left
// flank where it meets radius r, including the backlash rotation.
func flankHalfAngle(p Params, d dimensions, r float64) float64 {
	beta := flankRotation(d.pitch, d.base, float64(p.Teeth), d.backlash)
	l := leftFlankPoint(d.base, r, beta)
	return math.Atan2(l.y, l.x)
}

// closingRadius returns the radius of the arc that closes an embedded tooth
// section inside the root disc, clear of the neighbouring teeth.
func closingRadius(p Params, d dimensions) (float64, error) {
	limit := math.Pi / float64(p.Teeth)
	if flankHalfAngle(p, d, d.base) < limit {
		return d.base, nil
	}
	if flankHalfAngle(p, d, d.root) >= limit {
		return 0, fmt.Errorf("spurgear: embedded tooth: adjacent flanks meet at or outside the root circle (radius %g)",
			d.root)
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if flankHalfAngle(p, d, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// InvoluteFlanks samples one tooth's left and right involute flanks from the
// base circle to the tip circle, both endpoints included, turned by angle.
// effectiveTeeth replaces the tooth count and is never rounded. It applies
// no backlash. It returns the left array then the right array, each running
// base to tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []struct {
		name  string
		value float64
	}{
		{"pitch", pitch},
		{"base", base},
		{"tip", tip},
		{"effectiveTeeth", effectiveTeeth},
		{"angle", angle},
	} {
		if math.IsNaN(v.value) || math.IsInf(v.value, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: %s must be finite, got %g", v.name, v.value)
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
		return nil, nil, errors.New("spurgear: involute flanks: samples must be at least 3")
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return toPairs(left), toPairs(right), nil
}

func toPairs(points []point) [][2]float64 {
	out := make([][2]float64, len(points))
	for i, q := range points {
		out[i] = [2]float64{q.x, q.y}
	}
	return out
}
