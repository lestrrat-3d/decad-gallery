package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth on plane, turned by the signed
// angle about the sketch origin, and returns its solved, verified sketch and
// profile. The six-curve profile is two radial lines, two splines, one tip
// arc and the short root arc. Embedded gears are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, errors.New("tooth section: angle must be finite")
	}
	d := derive(p)
	if d.pitch <= 0 {
		return nil, nil, errors.New("tooth section: cannot derive dimensions")
	}
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	rightPts, leftPts := flankPoints(s, left, right)
	leftRoot, rightRoot := rootPoints(s, left[0], right[0], d)

	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: boundary has %d curves, expected 6", got)
	}
	return s, profile, nil
}

// solveSingleProfile solves and verifies s, then returns its one valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", stage, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	return profiles[0], nil
}
