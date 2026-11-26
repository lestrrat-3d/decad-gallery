package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the gear frame, in millimeters.
type point struct{ x, y float64 }

// dimensions holds the derived radii and the backlash, read from the
// document's parameter table, in millimeters.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// readDimensions reads the derived dimensions from the document's table.
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
			return dimensions{}, fmt.Errorf("parameter %s: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// derive computes the dimensions of p on a scratch document. It returns a
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

// filletRadius returns the root fillet radius of d.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) about the origin by theta radians.
func rotate(x, y, theta float64) point {
	c, s := math.Cos(theta), math.Sin(theta)
	return point{x*c - y*s, x*s + y*c}
}

// rootContact scales a base point to the root circle. It multiplies before
// dividing so each coordinate matches the sketch point exactly.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involuteAt returns the original involute point at radius r on a circle of
// radius base.
func involuteAt(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{
		base * (math.Cos(t) + t*math.Sin(t)),
		base * (math.Sin(t) - t*math.Cos(t)),
	}
}

// flankRotation returns the angle that turns the mirrored involute so its
// pitch crossing sits at pi/(2*teeth) - backlash/(2*pitch).
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	at := involuteAt(base, pitch)
	py := -at.y
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, at.x)
}

// leftFlankPoint returns the left flank point at radius r before any tooth
// rotation: the mirrored involute turned by beta.
func leftFlankPoint(base, r, beta float64) point {
	at := involuteAt(base, r)
	return rotate(at.x, -at.y, beta)
}

// sampleFlanks samples the left and right flanks of one tooth from radius
// start to tip, base-to-tip, and turns both by angle. Left comes first.
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

// InvoluteFlanks samples the left and right involute flanks of one tooth from
// the base radius to the tip radius, base-to-tip, with no backlash. The pitch
// crossings lie at +/-pi/(2*effectiveTeeth) before the rotation by angle.
// effectiveTeeth may be fractional.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite input %v", v)
		}
	}
	switch {
	case base <= 0:
		return nil, nil, fmt.Errorf("involute flanks: base radius %v must be positive", base)
	case pitch <= base:
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %v must exceed base radius %v", pitch, base)
	case tip <= pitch:
		return nil, nil, fmt.Errorf("involute flanks: tip radius %v must exceed pitch radius %v", tip, pitch)
	case effectiveTeeth <= 0:
		return nil, nil, fmt.Errorf("involute flanks: effective teeth %v must be positive", effectiveTeeth)
	case samples < 3:
		return nil, nil, fmt.Errorf("involute flanks: samples %d must be at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	outLeft := make([][2]float64, len(left))
	outRight := make([][2]float64, len(right))
	for i := range left {
		outLeft[i] = [2]float64{left[i].x, left[i].y}
		outRight[i] = [2]float64{right[i].x, right[i].y}
	}
	return outLeft, outRight, nil
}
