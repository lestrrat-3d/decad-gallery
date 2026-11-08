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

// addToothPoints creates the dimensioned flank and root points of one
// ordinary tooth. The right flank and right root receive both coordinate
// dimensions. The left flank and left root receive both, except the left
// tip and the left root, which receive one.
func addToothPoints(s *sketch.Sketch, d dimensions, left, right []point) toothPoints {
	n := len(left)
	tp := toothPoints{
		right: make([]*sketch.Point, n),
		left:  make([]*sketch.Point, n),
	}
	for i := 0; i < n; i++ {
		tp.right[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == n-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			continue
		}
		tp.left[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
	}
	tp.rootLeft = rootContact(left[0], d)
	tp.rootRight = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.rootLeft.x, tp.rootLeft.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rootRight.x, tp.rootRight.y)
	return tp
}

// singleProfile requires exactly one valid profile in s.
func singleProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, expected 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is invalid", what)
	}
	return profiles[0], nil
}

// solveVerify solves s and requires a fully constrained sketch.
func solveVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", what, err)
	}
	return nil
}

// BuildToothSection draws one ordinary tooth, turned by angle, as a
// six-curve section on plane. The section is solved and verified.
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
		return nil, nil, fmt.Errorf("tooth section: embedded flanks cross the root circle; "+
			"use BuildProfile for the complete gear outline (base radius %v, root radius %v)",
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
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, d, left, right)
	n := len(left)
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[n-1], tp.left[n-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)
	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
