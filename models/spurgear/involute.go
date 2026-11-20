package spurgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate in sketch space.
type point struct {
	x, y float64
}

// dimensions holds the derived lengths of one gear, in millimeters.
type dimensions struct {
	pitch, base, root, tip, fillet, backlash float64
}

// derive evaluates the parameter table of p on a scratch document. It returns
// zero dimensions when the table cannot be defined or read.
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

// readDimensions reads the derived dimensions from the document's parameter table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	pitch, err := doc.Param("pitch_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read pitch_radius: %w", err)
	}
	base, err := doc.Param("base_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read base_radius: %w", err)
	}
	root, err := doc.Param("root_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read root_radius: %w", err)
	}
	tip, err := doc.Param("tip_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read tip_radius: %w", err)
	}
	fillet, err := doc.Param("fillet_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read fillet_radius: %w", err)
	}
	backlash, err := doc.Param("backlash")
	if err != nil {
		return dimensions{}, fmt.Errorf("read backlash: %w", err)
	}
	return dimensions{
		pitch:    pitch.Base(),
		base:     base.Base(),
		root:     root.Base(),
		tip:      tip.Base(),
		fillet:   fillet.Base(),
		backlash: backlash.Base(),
	}, nil
}

// filletRadius returns the root fillet radius of the gear.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// rotate turns (x, y) by theta about the origin.
func rotate(x, y, theta float64) point {
	return point{x*math.Cos(theta) - y*math.Sin(theta), x*math.Sin(theta) + y*math.Cos(theta)}
}

// rootContact scales a flank's base point to the root circle.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle returns the rotation of tooth k.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the involute of the base circle at radius r.
func involutePoint(base, r float64) point {
	t := math.Tan(math.Acos(base / r))
	return point{base * (math.Cos(t) + t*math.Sin(t)), base * (math.Sin(t) - t*math.Cos(t))}
}

// flankRotation returns the rotation beta that places the mirrored involute as
// the left flank, so the flank crosses the pitch circle at pi/(2N) - delta.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	ip := involutePoint(base, pitch)
	px, py := ip.x, -ip.y
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(py, px)
}

// leftFlankPoint returns the unrotated left flank point at radius r.
func leftFlankPoint(base, r, beta float64) point {
	ip := involutePoint(base, r)
	return rotate(ip.x, -ip.y, beta)
}

// sampleFlanks samples the left and right flanks of one tooth from radius
// start to tip, both base-to-tip, turned by angle about the origin.
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

// InvoluteFlanks samples one tooth's left and right involute flanks, base to
// tip, with real-valued effectiveTeeth and no backlash, turned by angle.
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
	return pairs(left), pairs(right), nil
}

// pairs converts points to coordinate pairs.
func pairs(points []point) [][2]float64 {
	out := make([][2]float64, len(points))
	for i, pt := range points {
		out[i] = [2]float64{pt.x, pt.y}
	}
	return out
}
