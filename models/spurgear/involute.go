package spurgear

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in sketch (frame) space.
type point struct {
	x, y float64
}

// dimensions holds the derived gear radii, the root fillet radius and the
// backlash, all in millimetres.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive evaluates the parameter table of p on a scratch document. It returns
// a zero dimensions value when the table cannot be defined or read.
func derive(p Params) dimensions {
	d, err := deriveChecked(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// deriveChecked is derive with the table error preserved.
func deriveChecked(p Params) (dimensions, error) {
	doc := decad.New()
	if err := defineParameters(doc.Params(), p); err != nil {
		return dimensions{}, err
	}
	return readDimensions(doc)
}

// readDimensions reads the derived radii and the backlash from the document's
// parameter table.
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
			return dimensions{}, fmt.Errorf("read parameter %s: %w", f.name, err)
		}
		*f.dst = v.Base()
	}
	return d, nil
}

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) counter-clockwise by theta about the origin.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a flank's base point onto the root circle. Each
// coordinate multiplies before dividing.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involuteAt returns the original involute of the base circle at radius r.
func involuteAt(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankRotation returns β, the rotation that places the mirrored involute on
// the left flank so its pitch crossing lies at π/(2N) - δ.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	ip := involuteAt(base, pitch)
	px, py := ip.x, -ip.y
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, px)
}

// leftFlankPoint returns the unrotated left-flank point at radius r.
func leftFlankPoint(base, r, beta float64) point {
	i := involuteAt(base, r)
	return rotate(i.x, -i.y, beta)
}

// sampleFlanks samples the left and right flanks of one tooth from radius
// start to radius tip, both endpoints included, turned by angle. Both arrays
// run from start to tip.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		l := leftFlankPoint(base, r, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples one tooth's left and right involute flanks from the
// base circle to the tip circle, turned by angle. effectiveTeeth may be any
// positive real tooth count. No backlash is applied.
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
			return nil, nil, fmt.Errorf("involute flanks: %s must be finite, got %v", v.name, v.value)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("involute flanks: base radius must be positive, got %v", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("involute flanks: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("involute flanks: effective teeth must be positive, got %v", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("involute flanks: samples must be at least 3, got %d", samples)
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

// errNoClosingRadius reports an embedded tooth whose neighbours meet at the
// root circle.
var errNoClosingRadius = errors.New("embedded tooth: adjacent teeth meet at or beyond the root circle")

// closingRadius returns the radius of the arc that closes an embedded tooth
// section: the base radius when the teeth stay apart there, otherwise the
// midpoint between the radius where adjacent teeth meet and the root radius.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	limit := math.Pi / teeth
	half := func(r float64) float64 {
		pt := leftFlankPoint(d.base, r, beta)
		return math.Atan2(pt.y, pt.x)
	}
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("root circle radius %v: %w", d.root, errNoClosingRadius)
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if half(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}
