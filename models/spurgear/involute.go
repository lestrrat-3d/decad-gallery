package spurgear

import (
	"errors"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the gear frame, in millimeters.
type point struct{ x, y float64 }

// dimensions holds the derived radii and backlash read from the parameter table.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive computes the dimensions of p on a scratch document. It returns a zero
// value when the parameter table cannot be defined or read.
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
func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) about the origin by theta radians.
func rotate(x, y, theta float64) point {
	s, c := math.Sincos(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base-circle point out to the root circle.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the angle of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint returns the mirrored involute point at radius r on a base circle.
func involutePoint(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	sin, cos := math.Sincos(t)
	return point{base * (cos + t*sin), -base * (sin - t*cos)}
}

// flankRotation returns beta, the rotation that places the pitch crossing of
// the mirrored involute at pi/(2*teeth) - delta.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	pp := involutePoint(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// sampleFlanks samples the left and right involute flanks from radius start to
// tip, both base-to-tip, rotated by angle. It returns the left array then the
// right array.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := involutePoint(base, r)
		l := rotate(m.x, m.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right involute flanks for the given
// radii and real-valued effective tooth count, base-to-tip, rotated by angle.
// It applies no backlash.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, errors.New("involute flanks: nonfinite input")
		}
	}
	switch {
	case base <= 0:
		return nil, nil, errors.New("involute flanks: base radius must be positive")
	case pitch <= base:
		return nil, nil, errors.New("involute flanks: pitch radius must exceed base radius")
	case tip <= pitch:
		return nil, nil, errors.New("involute flanks: tip radius must exceed pitch radius")
	case effectiveTeeth <= 0:
		return nil, nil, errors.New("involute flanks: effective teeth must be positive")
	case samples < 3:
		return nil, nil, errors.New("involute flanks: at least 3 samples required")
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	l := make([][2]float64, len(left))
	r := make([][2]float64, len(right))
	for i := range left {
		l[i] = [2]float64{left[i].x, left[i].y}
		r[i] = [2]float64{right[i].x, right[i].y}
	}
	return l, r, nil
}
