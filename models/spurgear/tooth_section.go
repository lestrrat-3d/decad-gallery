package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the sketch points of one ordinary tooth section.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// flankPoints creates the dimensioned flank sample points: every right point
// with both dimensions, every left point likewise except the left tip.
func flankPoints(s *sketch.Sketch, left, right []point) (rightPts, leftPts []*sketch.Point) {
	rightPts = make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts = make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rightPts, leftPts
}

// scaleToRoot locates the root contact of a base sample. It multiplies before
// dividing on purpose.
func scaleToRoot(q point, d dimensions) point {
	return point{q.x * d.root / d.base, q.y * d.root / d.base}
}

// createToothPoints creates the flank and root points of one tooth.
func createToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var tp toothPoints
	tp.right, tp.left = flankPoints(s, left, right)
	tp.leftRootXY = scaleToRoot(left[0], d)
	tp.rightRootXY = scaleToRoot(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootXY.x, tp.leftRootXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootXY.x, tp.rightRootXY.y)
	return tp
}

// solveAndVerify solves the sketch and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", what, err)
	}
	return nil
}

// singleProfile requires exactly one valid profile.
func singleProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, expected 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", what)
	}
	return profiles[0], nil
}

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane and
// returns its solved, verified sketch and bounded profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: nonfinite tooth angle %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section is not available for an embedded flank profile")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection is BuildToothSection without parameter validation.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	tp := createToothPoints(s, left, right, d)

	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[len(tp.right)-1], tp.left[len(tp.left)-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)

	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
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
