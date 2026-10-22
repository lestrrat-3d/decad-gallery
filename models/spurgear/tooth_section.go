package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth, rotated by angle, as a closed
// six-curve profile on plane. Embedded flanks are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nonfinite angle %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section: embedded flanks are not supported")
	}

	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	rp, lp := dimensionFlanks(s, left, right)
	leftRoot, rightRoot := rootPoints(s, d, left[0], right[0])

	s.CreateLine(rightRoot, rp[0])
	rs, err := s.CreateFitSpline(rp...)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: right flank spline: %w", err)
	}
	_ = rs
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: left flank spline: %w", err)
	}
	s.CreateLine(lp[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	if n := len(profile.Outer); n != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section: got %d outer curves, want 6", n)
	}
	return s, profile, nil
}

// dimensionFlanks creates the dimensioned flank points. The left tip gets a
// single coordinate dimension; every other flank point gets both.
func dimensionFlanks(s *sketch.Sketch, left, right []point) (rp, lp []*sketch.Point) {
	rp = make([]*sketch.Point, len(right))
	lp = make([]*sketch.Point, len(left))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rp, lp
}

// rootPoints scales the base samples to the root radius (multiply, then
// divide) and creates the left root before the right root.
func rootPoints(s *sketch.Sketch, d dimensions, leftBase, rightBase point) (leftRoot, rightRoot *sketch.Point) {
	leftRoot = sketchcheck.DimensionArcEnd(s, leftBase.x*d.root/d.base, leftBase.y*d.root/d.base)
	rightRoot = sketchcheck.DimensionPoint(s, rightBase.x*d.root/d.base, rightBase.y*d.root/d.base)
	return leftRoot, rightRoot
}

// solveSingleProfile solves and verifies s, then requires exactly one valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("got %d profiles, want 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is invalid")
	}
	return profiles[0], nil
}
