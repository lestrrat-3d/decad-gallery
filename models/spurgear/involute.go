package spurgear

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
)

// point is a planar coordinate pair in sketch (frame) coordinates.
type point struct{ x, y float64 }

// dimensions holds the derived gear radii read from the parameter table.
type dimensions struct{ pitch, base, root, tip, fillet, backlash float64 }

// derive computes the derived dimensions of p on a scratch document. It
// returns a zero dimensions value when the parameter table cannot be built
// or read.
func derive(p Params) dimensions {
	d, err := computeDimensions(p)
	if err != nil {
		return dimensions{}
	}
	return d
}

// computeDimensions is derive with the table error preserved.
func computeDimensions(p Params) (dimensions, error) {
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
	pitch, err := doc.Param("pitch_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read pitch_radius: %w", err)
	}
	d.pitch = pitch.Base()
	base, err := doc.Param("base_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read base_radius: %w", err)
	}
	d.base = base.Base()
	root, err := doc.Param("root_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read root_radius: %w", err)
	}
	d.root = root.Base()
	tip, err := doc.Param("tip_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read tip_radius: %w", err)
	}
	d.tip = tip.Base()
	fillet, err := doc.Param("fillet_radius")
	if err != nil {
		return dimensions{}, fmt.Errorf("read fillet_radius: %w", err)
	}
	d.fillet = fillet.Base()
	backlash, err := doc.Param("backlash")
	if err != nil {
		return dimensions{}, fmt.Errorf("read backlash: %w", err)
	}
	d.backlash = backlash.Base()
	return d, nil
}

// rotate turns (x,y) by theta about the origin.
func rotate(x, y, theta float64) point {
	sin, cos := math.Sincos(theta)
	return point{x*cos - y*sin, x*sin + y*cos}
}

func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// toothAngle is the rotation of tooth k about the gear axis.
func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// involutePoint evaluates the original (unmirrored) involute of the base
// circle at radius r.
func involutePoint(base, r float64) (float64, float64) {
	ratio := base / r
	if ratio > 1 {
		ratio = 1
	}
	t := math.Tan(math.Acos(ratio))
	sin, cos := math.Sincos(t)
	return base * (cos + t*sin), base * (sin - t*cos)
}

// flankRotation is the angle β that turns the mirrored involute into the left
// flank, so that the flank crosses the pitch circle at π/(2N) - δ.
func flankRotation(pitch, base, teeth, backlash float64) float64 {
	px, py := involutePoint(base, pitch)
	delta := backlash / (2 * pitch)
	return math.Pi/(2*teeth) - delta - math.Atan2(-py, px)
}

// sampleFlanks samples the left and right flanks of one tooth from radius
// start to radius tip, both endpoints included, rotated by angle. Both arrays
// run from start to tip.
func sampleFlanks(pitch, base, start, tip, teeth, backlash float64, samples int,
	angle float64) ([]point, []point) {
	beta := flankRotation(pitch, base, teeth, backlash)
	left := make([]point, samples)
	right := make([]point, samples)
	for i := range samples {
		r := start + (tip-start)*float64(i)/float64(samples-1)
		ix, iy := involutePoint(base, r)
		l := rotate(ix, -iy, beta)
		left[i] = rotate(l.x, l.y, angle)
		right[i] = rotate(l.x, -l.y, angle)
	}
	return left, right
}

// InvoluteFlanks samples one tooth's left and right involute flanks from the
// base circle to the tip circle, rotated by angle. effectiveTeeth may be any
// positive real count and is never rounded. No backlash is applied.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, errors.New("spurgear: involute flanks: nonfinite input")
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

func pairs(pts []point) [][2]float64 {
	out := make([][2]float64, len(pts))
	for i, pt := range pts {
		out[i] = [2]float64{pt.x, pt.y}
	}
	return out
}

// closingRadius chooses the radius Rc of the arc that closes an embedded
// tooth section. h(r) is the angle between the tooth axis and the left flank
// at radius r; neighbouring teeth stay apart while h(r) < π/N.
func closingRadius(d dimensions, teeth int) (float64, error) {
	n := float64(teeth)
	beta := flankRotation(d.pitch, d.base, n, d.backlash)
	limit := math.Pi / n
	half := func(r float64) float64 {
		ix, iy := involutePoint(d.base, r)
		q := rotate(ix, -iy, beta)
		return math.Atan2(q.y, q.x)
	}
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: embedded tooth: neighbouring flanks still meet at the root circle (radius %g)", d.root)
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
	return (hi + d.root) / 2, nil
}
