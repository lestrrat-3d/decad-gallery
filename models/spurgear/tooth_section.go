package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the dimensioned points of one ordinary tooth section.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
}

// addToothPoints creates the dimensioned flank and root points of one tooth.
// Every right point has both coordinate dimensions. Every left point has both
// except its tip and its root, which have one.
func addToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var tp toothPoints
	tp.right = make([]*sketch.Point, len(right))
	tp.left = make([]*sketch.Point, len(left))
	for i, q := range right {
		tp.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			tp.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, rr.x, rr.y)
	return tp
}

// addToothCurves draws the right line, right spline, tip arc, left spline and
// left line of one ordinary tooth.
func addToothCurves(s *sketch.Sketch, tp toothPoints) error {
	center := s.Origin()
	last := len(tp.right) - 1
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	return nil
}

// solveVerified solves s and requires it to be fully constrained.
func solveVerified(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// onlyProfile returns the single valid profile of a solved sketch.
func onlyProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, want 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// section is a solved, verified sketch and its single profile of six curves.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle %v must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: a standalone tooth section does not support the embedded flank branch")
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
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, left, right, d)
	if err := addToothCurves(s, tp); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), tp.rightRoot, tp.leftRoot)
	if err := solveVerified(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, want 6", got)
	}
	return s, profile, nil
}
