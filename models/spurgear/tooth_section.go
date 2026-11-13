package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	// rightRootAt and leftRootAt are the root coordinates.
	rightRootAt, leftRootAt point
}

// addToothPoints creates the dimensioned flank and root points of one ordinary
// tooth. The right flank points and the right root receive both coordinate
// dimensions. The left flank points do too, except the tip and the root, which
// receive one.
func addToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	n := len(right)
	tp := toothPoints{
		right: make([]*sketch.Point, n),
		left:  make([]*sketch.Point, n),
	}
	for i, pt := range right {
		tp.right[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	for i, pt := range left {
		if i == n-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		tp.left[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	tp.leftRootAt = rootContact(left[0], d)
	tp.rightRootAt = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootAt.x, tp.leftRootAt.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootAt.x, tp.rightRootAt.y)
	return tp
}

// BuildToothSection draws one ordinary tooth, rotated by angle, as a
// six-curve closed section on plane. It applies no solid features.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %v is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks are not supported (base radius %g below root radius %g)",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section at angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, left, right, d)
	last := p.Samples - 1

	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("tooth section: solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: verify: %w", err)
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
