package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth at the supplied signed angle on
// plane. It returns the solved, verified sketch and its six-curve profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %g is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %g below root radius %g) "+
			"have no standalone tooth section", d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()

	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	rightPoints, leftPoints := flankPoints(s, left, right)
	last := len(rightPoints) - 1

	leftRootXY := rootContact(left[0], d)
	rightRootXY := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)

	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: profile has %d boundary curves, expected 6", got)
	}
	return s, profile, nil
}
