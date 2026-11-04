package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the dimensioned sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootRight, rootLeft point
}

// dimensionFlanks creates the flank sample points. Every right point receives
// both coordinate dimensions. Every left point does too, except the tip, which
// receives one.
func dimensionFlanks(s *sketch.Sketch, left, right []point) (rightPts, leftPts []*sketch.Point) {
	rightPts = make([]*sketch.Point, len(right))
	leftPts = make([]*sketch.Point, len(left))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rightPts, leftPts
}

// dimensionOrdinaryTooth creates the flank points and the two root points of
// one ordinary tooth rotated by angle. The left root is created before the
// right root; the right root takes both dimensions and the left root one.
func dimensionOrdinaryTooth(s *sketch.Sketch, p Params, d dimensions, angle float64) toothPoints {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	var tp toothPoints
	tp.right, tp.left = dimensionFlanks(s, left, right)
	tp.rootLeft = rootContact(left[0], d)
	tp.rootRight = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.rootLeft.x, tp.rootLeft.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rootRight.x, tp.rootRight.y)
	return tp
}

// BuildToothSection draws one ordinary tooth section, rotated by angle, on
// plane. It returns the solved, verified sketch and its single profile of six
// curves. Embedded teeth are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section requires a context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section requires a sketch world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section requires a sketch plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section does not support embedded flanks (base radius %g below root radius %g)",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section without
// validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	tp := dimensionOrdinaryTooth(s, p, d, angle)
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

	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, expected 6", got)
	}
	return s, profile, nil
}
