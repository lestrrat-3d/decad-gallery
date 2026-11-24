package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in the gear's sketch frame.
type point struct{ x, y float64 }

// dimensions holds the derived radii and the backlash read from the
// parameter table, all in millimeters.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

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
			return dimensions{}, fmt.Errorf("read parameter %q: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// derive computes the dimensions of p through a scratch document. It returns
// a zero value when the parameter table cannot be defined or read.
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
func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
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

// involutePoint returns the original involute point at radius r, mirrored
// across X (negative Y).
func involutePoint(base, r float64) point {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	x := base * (math.Cos(t) + t*math.Sin(t))
	y := base * (math.Sin(t) - t*math.Cos(t))
	return point{x, -y}
}

// flankRotation returns the angle beta that turns the mirrored involute so its
// pitch crossing lies at pi/(2*teeth) - backlash/(2*pitch).
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	delta := backlash / (2 * pitch)
	pp := involutePoint(base, pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(pp.y, pp.x)
}

// flankHalfAngle returns the angle between the tooth axis and the left flank
// where the flank meets radius r.
func flankHalfAngle(pitch, base, teeth, backlash, r float64) float64 {
	beta := flankRotation(pitch, base, teeth, backlash)
	ip := involutePoint(base, r)
	q := rotate(ip.x, ip.y, beta)
	return math.Atan2(q.y, q.x)
}

// sampleFlanks samples the left and right flanks of one tooth from radius
// start to radius tip, both ends included, and turns them by angle. Both
// arrays run from start to tip. It returns the left array then the right one.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := 0; i < samples; i++ {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ip := involutePoint(base, r)
		l := rotate(ip.x, ip.y, beta)
		rt := point{l.x, -l.y}
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(rt.x, rt.y, angle)
	}
	return left, right
}
