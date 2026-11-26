package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth turned by angle on plane and
// returns the solved, verified sketch with its six-curve profile: right
// radial line, right flank, tip arc, left flank, left radial line and the
// short root arc between the two root contacts.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("tooth section: nil plane")
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle must be finite, got %v", angle)
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	d, err := deriveChecked(p)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf(
			"tooth section: base radius %v is below root radius %v; embedded flanks have no standalone tooth section",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the ordinary tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	leftSamples, rightSamples := sampleFlanks(d.pitch, d.base, d.base, d.tip,
		float64(p.Teeth), d.backlash, p.Samples, angle)
	rightPoints, leftPoints := flankPoints(s, rightSamples, leftSamples)

	leftRootAt := rootContact(leftSamples[0], d)
	rightRootAt := rootContact(rightSamples[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftRootAt.x, leftRootAt.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightRootAt.x, rightRootAt.y)

	last := len(rightPoints) - 1
	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: profile has %d outer curves, want 6", len(profile.Outer))
	}
	return s, profile, nil
}

// flankPoints creates the dimensioned flank points of one tooth. Every right
// point receives both coordinate dimensions; every left point does too except
// its tip, which receives one and takes its radius from the tip arc.
func flankPoints(s *sketch.Sketch, right, left []point) ([]*sketch.Point, []*sketch.Point) {
	rightPoints := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	leftPoints := make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		leftPoints[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return rightPoints, leftPoints
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// singleProfile requires s to hold exactly one valid profile and returns it.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("sketch has %d profiles, want 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("sketch profile is not valid")
	}
	return profiles[0], nil
}
