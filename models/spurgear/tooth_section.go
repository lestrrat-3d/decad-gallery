package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// toothPoints holds the dimensioned sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootRight, rootLeft point
}

// addFlankPoints creates the dimensioned points of both flanks. Every
// right-flank point takes both coordinate dimensions. Every left-flank point
// takes both except the tip, which takes one.
func addFlankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
	rightPts := make([]*sketch.Point, 0, len(right))
	for _, pt := range right {
		rightPts = append(rightPts, sketchcheck.DimensionPoint(s, pt.x, pt.y))
	}
	last := len(left) - 1
	leftPts := make([]*sketch.Point, 0, len(left))
	for i, pt := range left {
		if i == last {
			leftPts = append(leftPts, sketchcheck.DimensionArcEnd(s, pt.x, pt.y))
			continue
		}
		leftPts = append(leftPts, sketchcheck.DimensionPoint(s, pt.x, pt.y))
	}
	return rightPts, leftPts
}

// addToothPoints creates the dimensioned flank and root points of one tooth.
// The right root takes both coordinate dimensions and the left root takes one.
func addToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var t toothPoints
	t.right, t.left = addFlankPoints(s, left, right)
	t.rootLeft = rootContact(left[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.rootLeft.x, t.rootLeft.y)
	t.rootRight = rootContact(right[0], d)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rootRight.x, t.rootRight.y)
	return t
}

// toothCurves creates the right radial line, right spline, tip arc, left
// spline and left radial line of tooth k. It returns the two splines.
func (t toothPoints) toothCurves(s *sketch.Sketch, k int) (*sketch.FitSpline, *sketch.FitSpline, error) {
	last := len(t.right) - 1
	s.CreateLine(t.rightRoot, t.right[0])
	rightSpline, err := s.CreateFitSpline(t.right...)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
	}
	s.CreateArc(s.Origin(), t.right[last], t.left[last])
	leftSpline, err := s.CreateFitSpline(t.left...)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return rightSpline, leftSpline, nil
}

// solveVerified solves s and requires it to be fully constrained.
func solveVerified(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// singleProfile requires exactly one valid profile with outerCount outer curves.
func singleProfile(s *sketch.Sketch, outerCount int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profiles: got %d, expected 1", len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, errors.New("profile is not valid")
	}
	if outerCount > 0 && len(profile.Outer) != outerCount {
		return nil, fmt.Errorf("outer curves: got %d, expected %d", len(profile.Outer), outerCount)
	}
	return profile, nil
}

// buildToothSection draws one ordinary tooth at angle on plane. It does not
// validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	t := addToothPoints(s, left, right, d)
	if _, _, err := t.toothCurves(s, 0); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}

// BuildToothSection draws one ordinary tooth rotated by angle about the
// sketch origin on plane. It rejects embedded flanks, where the base circle
// lies inside the root circle. The caller extrudes the returned profile.
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
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
