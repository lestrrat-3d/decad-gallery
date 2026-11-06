package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth at the signed angle on plane. The
// section is solved, verified and returned with its single profile.
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
		return nil, nil, fmt.Errorf("tooth section: angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks are not supported (base radius %g below root radius %g)",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth at angle. It does not
// validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	leftPts, rightPts := flankPoints(s, left, right, false)
	leftRoot, rightRoot := rootPoints(s, left[0], right[0], d)

	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[p.Samples-1], leftPts[p.Samples-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}

// rootPoints creates the left then right root points by scaling the first flank
// samples onto the root circle. The right root gets both dimensions, the left one.
func rootPoints(s *sketch.Sketch, leftBase, rightBase point, d dimensions) (*sketch.Point, *sketch.Point) {
	lr := rootContact(leftBase, d)
	rr := rootContact(rightBase, d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	return leftRoot, rightRoot
}
