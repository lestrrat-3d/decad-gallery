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
	rightRootXY         point
	leftRootXY          point
}

// drawToothPoints creates the dimensioned flank and root points of one
// ordinary tooth rotated by angle. The left root is created before the right.
func drawToothPoints(s *sketch.Sketch, p Params, d dimensions, angle float64) toothPoints {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	var t toothPoints
	t.right = make([]*sketch.Point, len(right))
	t.left = make([]*sketch.Point, len(left))
	for i, q := range right {
		t.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			t.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		t.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	t.leftRootXY = rootContact(left[0], d)
	t.rightRootXY = rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)
	return t
}

// drawToothCurves creates the right radial line, right spline, tip arc, left
// spline and left radial line of one ordinary tooth, in that order.
func drawToothCurves(s *sketch.Sketch, t toothPoints) error {
	last := len(t.right) - 1
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), t.right[last], t.left[last])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// solveVerify solves s and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s sketch: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s sketch: %w", what, err)
	}
	return nil
}

// onlyProfile requires exactly one valid profile in s.
func onlyProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s sketch has %d profiles, expected 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s profile is invalid", what)
	}
	return profiles[0], nil
}

// buildToothSection draws one ordinary tooth at angle, without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create tooth sketch: %w", err)
	}
	t := drawToothPoints(s, p, d, angle)
	if err := drawToothCurves(s, t); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane and
// returns its solved, verified sketch and profile. Embedded gears are
// rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: nil sketch world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: nil sketch plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: embedded flanks have no standalone tooth section")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
