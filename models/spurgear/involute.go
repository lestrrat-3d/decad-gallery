package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate pair in sketch (frame) coordinates.
type point struct{ x, y float64 }

// dimensions holds the derived gear dimensions read from the parameter table,
// in millimetres.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// derive computes the derived dimensions of p on a scratch document. It returns
// a zero dimensions when the parameter table cannot be written or read.
func derive(p Params) dimensions {
	d, err := deriveChecked(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// deriveChecked is derive with the table error reported.
func deriveChecked(p Params) (dimensions, error) {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}, err
	}
	return readDimensions(doc)
}

// readDimensions reads the derived dimensions from the document's parameter table.
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
			return dimensions{}, fmt.Errorf("read %s: %w", f.name, err)
		}
		*f.dst = v.Base()
	}
	return d, nil
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a flank's base point onto the root circle.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle is the angle of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involute evaluates the base-circle involute at radius r (r >= base).
func involute(base, r float64) (float64, float64) {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	return base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))
}

// flankRotation is the angle beta that turns the mirrored involute into the
// left flank, so the flank crosses the pitch circle at pi/(2N) - delta.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	px, py := involute(base, pitch)
	py = -py
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, px)
}

// leftFlankPoint is the unrotated-tooth left flank point at radius r.
func leftFlankPoint(base, r, beta float64) point {
	ix, iy := involute(base, r)
	return rotate(ix, -iy, beta)
}

// flankHalfAngle is h(r): the angle between the tooth axis and the left flank
// at radius r.
func flankHalfAngle(pitch, base, teeth, backlash, r float64) float64 {
	q := leftFlankPoint(base, r, flankRotation(pitch, base, teeth, backlash))
	return math.Atan2(q.y, q.x)
}

// sampleFlanks samples the left and right flanks from radius start to tip,
// both endpoints included, base-to-tip, turned by angle. It returns left then
// right.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		switch i {
		case 0:
			r = start
		case samples - 1:
			r = tip
		}
		l := leftFlankPoint(base, r, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples one tooth's left and right involute flanks from the
// base circle to the tip circle, base-to-tip, turned by angle. effectiveTeeth
// may be any positive real tooth count; no backlash is applied.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []struct {
		name  string
		value float64
	}{
		{"pitch", pitch}, {"base", base}, {"tip", tip},
		{"effectiveTeeth", effectiveTeeth}, {"angle", angle},
	} {
		if math.IsNaN(v.value) || math.IsInf(v.value, 0) {
			return nil, nil, fmt.Errorf("spurgear: involute flanks: %s is not finite", v.name)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: base radius %g must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: effective teeth %g must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("spurgear: involute flanks: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	return pairs(left), pairs(right), nil
}

// pairs converts points to coordinate pairs.
func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, q := range pts {
		out[i] = [2]float64{q.x, q.y}
	}
	return out
}
