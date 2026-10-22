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
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// scaleRoot moves a base-circle sample onto the root circle. It multiplies
// before dividing; a precomputed ratio changes the last bit.
func scaleRoot(pt point, d dimensions) point {
	return point{pt.x * d.root / d.base, pt.y * d.root / d.base}
}

// solveAndVerify solves the sketch and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// onlyProfile returns the single valid profile of a solved sketch.
func onlyProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profile count %d, want 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is invalid")
	}
	return profiles[0], nil
}

// makeToothPoints creates the flank points and the two root points of one
// ordinary tooth. The left root is created before the right root.
func makeToothPoints(s *sketch.Sketch, d dimensions, left, right []point) toothPoints {
	n := len(right)
	t := toothPoints{right: make([]*sketch.Point, n), left: make([]*sketch.Point, n)}
	for i := 0; i < n; i++ {
		t.right[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n-1; i++ {
		t.left[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
	}
	t.left[n-1] = sketchcheck.DimensionArcEnd(s, left[n-1].x, left[n-1].y)
	t.leftRootXY = scaleRoot(left[0], d)
	t.rightRootXY = scaleRoot(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)
	return t
}

// drawToothSide draws the right radial line, right spline, tip arc, left
// spline and left radial line of one tooth.
func drawToothSide(s *sketch.Sketch, t toothPoints) error {
	center := s.Origin()
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, t.right[len(t.right)-1], t.left[len(t.left)-1])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// six-curve section is closed by the short root arc. It returns the solved,
// verified sketch and its profile and applies no solid feature.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil sketch world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: nil sketch plane")
	}
	d, err := validate(p)
	if err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section angle %v is not finite", angle)
	}
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: standalone tooth section does not support the embedded flank profile")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection builds an ordinary tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	t := makeToothPoints(s, d, left, right)
	if err := drawToothSide(s, t); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, want 6", got)
	}
	return s, profile, nil
}
