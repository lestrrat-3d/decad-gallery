package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the dimensioned points of one ordinary tooth.
type toothPoints struct {
	rightRoot, leftRoot     *sketch.Point
	rightRootXY, leftRootXY point
	right, left             []*sketch.Point
}

// addToothPoints creates the dimensioned flank and root points of one ordinary
// tooth. Every right-flank point and the right root take both coordinate
// dimensions. Each left-flank point does too, except the tip and the left root,
// which take one.
func addToothPoints(s *sketch.Sketch, d dimensions, left, right []point) toothPoints {
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
	tp.leftRootXY = rootContact(left[0], d)
	tp.rightRootXY = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootXY.x, tp.leftRootXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootXY.x, tp.rightRootXY.y)
	return tp
}

// drawTooth draws the right radial line, right spline, tip arc, left spline and
// left radial line of one ordinary tooth.
func drawTooth(s *sketch.Sketch, tp toothPoints) error {
	center := s.Origin()
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[len(tp.right)-1], tp.left[len(tp.left)-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	return nil
}

// solveOne solves and verifies a sketch and returns its single valid profile.
func solveOne(ctx context.Context, s *sketch.Sketch, what string) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", what, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, want 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", what)
	}
	return profiles[0], nil
}

// buildToothSection builds one ordinary tooth section at the given angle. It
// does not validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, d, left, right)
	if err := drawTooth(s, tp); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), tp.rightRoot, tp.leftRoot)
	profile, err := solveOne(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, want 6", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildToothSection builds one ordinary tooth section on plane, rotated by
// angle. It returns the solved, verified sketch and its profile.
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
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, errors.New("tooth section: nonfinite angle")
	}
	d := derive(p)
	if d.pitch == 0 {
		return nil, nil, errors.New("tooth section: cannot derive dimensions")
	}
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
