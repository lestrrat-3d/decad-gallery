package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// solveVerified solves s and requires it to be fully constrained with no
// redundant constraints or conflicts.
func solveVerified(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify sketch: %w", err)
	}
	return nil
}

// singleProfile returns the one valid profile of a solved sketch.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("expected 1 profile, got %d", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is not valid")
	}
	return profiles[0], nil
}

// toothPoints holds the sketch points of one ordinary tooth section.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// newToothPoints creates the dimensioned points of one ordinary tooth: the
// right flank, the left flank, then the left and right roots scaled from the
// flank base points.
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
	tp.leftRootXY = rootContact(left[0], d)
	tp.rightRootXY = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootXY.x, tp.leftRootXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootXY.x, tp.rightRootXY.y)
	return tp
}

// drawFlanks draws the right radial line, right spline, tip arc, left spline
// and left radial line of one tooth.
func drawFlanks(s *sketch.Sketch, tooth int, tp toothPoints) error {
	last := len(tp.right) - 1
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return fmt.Errorf("tooth %d right flank: %w", tooth, err)
	}
	s.CreateArc(s.Origin(), tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return fmt.Errorf("tooth %d left flank: %w", tooth, err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	return nil
}

// buildToothSection draws one ordinary tooth, turned by angle, as a
// six-curve section on plane. The short root arc closes it.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := newToothPoints(s, d, left, right)
	if err := drawFlanks(s, 0, tp); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), tp.rightRoot, tp.leftRoot)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d boundary curves, expected 6", got)
	}
	return s, profile, nil
}

// BuildToothSection draws one ordinary tooth, turned by angle radians, on
// plane. It returns the solved, verified sketch and its six-curve profile. It
// applies no solid feature. An embedded tooth, whose flanks cross the root
// circle, is rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("tooth section: nil sketch world")
	}
	if plane == nil {
		return nil, nil, errors.New("tooth section: nil sketch plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks cross the root circle")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
