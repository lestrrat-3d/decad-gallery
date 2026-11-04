package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth on plane, rotated by angle, as
// a solved and verified six-curve section. It applies no solid feature.
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
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle must be finite, got %v", angle)
	}
	d := derive(p)
	if d.root <= 0 {
		return nil, nil, errors.New("tooth section: cannot derive gear dimensions")
	}
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth: right radial line,
// right flank, tip arc, left flank, left radial line and the short root arc.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	tp := newToothPoints(s, d, left, right)
	if err := drawToothFlanks(s, tp); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}

// toothPoints holds the sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootAt         point
	leftRootAt          point
}

// newToothPoints creates the dimensioned flank points and the two root
// points of one ordinary tooth. The left root is created before the right.
func newToothPoints(s *sketch.Sketch, d dimensions, left, right []point) toothPoints {
	var tp toothPoints
	tp.right = make([]*sketch.Point, len(right))
	for i, q := range right {
		tp.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	tp.left = make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			tp.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	tp.leftRootAt = rootContact(left[0], d)
	tp.rightRootAt = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootAt.x, tp.leftRootAt.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootAt.x, tp.rightRootAt.y)
	return tp
}

// drawToothFlanks draws the five curves between the two roots of a tooth.
func drawToothFlanks(s *sketch.Sketch, tp toothPoints) error {
	last := len(tp.right) - 1
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	return nil
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

// singleProfile requires exactly one valid profile and, when curves is
// positive, exactly that many outer curves.
func singleProfile(s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("expected 1 profile, got %d", len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, errors.New("profile is not valid")
	}
	if curves > 0 && len(profile.Outer) != curves {
		return nil, fmt.Errorf("expected %d outer curves, got %d", curves, len(profile.Outer))
	}
	return profile, nil
}
