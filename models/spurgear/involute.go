package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the sketch frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii and the backlash, in millimeters.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive evaluates the parameter table of p on a scratch document. It returns
// a zero dimensions value when the table cannot be written or read.
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

// rotate turns (x, y) by theta radians about the origin.
func rotate(x, y, theta float64) point {
	sin, cos := math.Sincos(theta)
	return point{x*cos - y*sin, x*sin + y*cos}
}

// rootContact scales a base-circle sample to the root circle, multiplying before dividing.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint returns the original involute of a circle of radius base at
// radius r, mirrored across X.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(math.Min(base/r, 1)))
	x := base * (math.Cos(t) + t*math.Sin(t))
	y := base * (math.Sin(t) - t*math.Cos(t))
	return point{x, -y}
}

// flankPhase returns the rotation that places the mirrored involute's pitch
// crossing at half a tooth thickness from the tooth axis.
func flankPhase(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	m := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(m.y, m.x)
}

// sampleFlanks samples one tooth's flanks at radii running linearly from start
// to tip, then rotates both by angle. It returns the left and right arrays,
// both running from start to tip.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankPhase(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		m := involutePoint(base, r)
		l := rotate(m.x, m.y, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples the left and right flanks of one tooth from the base
// circle to the tip circle, turned by angle, with no backlash. The tooth count
// need not be an integer. Both arrays run base to tip.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, fmt.Errorf("involute flanks: nonfinite argument %g", v)
		}
	}
	if base <= 0 {
		return nil, nil, fmt.Errorf("involute flanks: base radius %g must be positive", base)
	}
	if pitch <= base {
		return nil, nil, fmt.Errorf("involute flanks: pitch radius %g must exceed base radius %g", pitch, base)
	}
	if tip <= pitch {
		return nil, nil, fmt.Errorf("involute flanks: tip radius %g must exceed pitch radius %g", tip, pitch)
	}
	if effectiveTeeth <= 0 {
		return nil, nil, fmt.Errorf("involute flanks: effective teeth %g must be positive", effectiveTeeth)
	}
	if samples < 3 {
		return nil, nil, fmt.Errorf("involute flanks: %d samples, need at least 3", samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, 0, samples, angle)
	leftPairs := make([][2]float64, len(left))
	rightPairs := make([][2]float64, len(right))
	for i := range left {
		leftPairs[i] = [2]float64{left[i].x, left[i].y}
		rightPairs[i] = [2]float64{right[i].x, right[i].y}
	}
	return leftPairs, rightPairs, nil
}
