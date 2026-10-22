package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// rootContact scales a base-circle sample to its root-circle contact. It
// multiplies before dividing so the coordinates match across call sites.
func rootContact(base point, d dimensions) point {
	return point{base.x * d.root / d.base, base.y * d.root / d.base}
}

// toothPoints holds the dimensioned sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// addToothPoints creates the dimensioned flank and root points of one ordinary
// tooth rotated by angle. The left root is created before the right root.
func addToothPoints(s *sketch.Sketch, p Params, d dimensions, angle float64) toothPoints {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
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

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// buildToothSection draws one ordinary tooth rotated by angle. The caller has
// already validated p and checked the branch.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	tp := addToothPoints(s, p, d, angle)
	last := p.Samples - 1
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)
	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth section: want 1 valid profile, got %d", len(profiles))
	}
	if n := len(profiles[0].Outer); n != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, want 6", n)
	}
	return s, profiles[0], nil
}

// BuildToothSection draws one ordinary tooth on plane, rotated by angle, and
// returns its verified sketch and profile. Embedded gears are rejected.
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
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks are not supported (base radius %g < root radius %g)",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
