package spurgear

import (
	"context"
	"errors"
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

// solveAndVerify solves the sketch and requires that it is fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	return nil
}

// singleProfile requires the sketch to hold exactly one valid profile.
func singleProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// dimensionFlanks creates the dimensioned points of both flanks. Every right
// point takes both coordinate dimensions. Every left point does too, except
// the tip and, when arcFirst is set, the first point, which take one.
func dimensionFlanks(s *sketch.Sketch, left, right []point, arcFirst bool) (rp, lp []*sketch.Point) {
	rp = make([]*sketch.Point, len(right))
	for i, pt := range right {
		rp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	lp = make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == len(left)-1 || (arcFirst && i == 0) {
			lp[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
	}
	return rp, lp
}

// addToothPoints samples one ordinary tooth turned by angle and creates its
// flank and root points. The left root is created before the right root.
func addToothPoints(s *sketch.Sketch, p Params, d dimensions, angle float64) toothPoints {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	var t toothPoints
	t.right, t.left = dimensionFlanks(s, left, right, false)
	t.leftRootXY = rootContact(left[0], d)
	t.rightRootXY = rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)
	return t
}

// drawToothSides draws the right line and spline, the tip arc, the left
// spline and the left line of one tooth. It returns both splines.
func drawToothSides(s *sketch.Sketch, t toothPoints, k int) (right, left *sketch.FitSpline, err error) {
	center := s.Origin()
	last := len(t.right) - 1
	s.CreateLine(t.rightRoot, t.right[0])
	right, err = s.CreateFitSpline(t.right...)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth %d: right flank: %w", k, err)
	}
	s.CreateArc(center, t.right[last], t.left[last])
	left, err = s.CreateFitSpline(t.left...)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth %d: left flank: %w", k, err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return right, left, nil
}

// BuildToothSection draws one ordinary tooth turned by angle on plane and
// returns its solved, verified sketch and its profile. Embedded gears are
// rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: tooth section requires a context, world and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle %v must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section does not support embedded flanks (base radius below root radius)")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section sketch: %w", err)
	}
	t := addToothPoints(s, p, d, angle)
	if _, _, err := drawToothSides(s, t, 0); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, expected 6", got)
	}
	return s, profile, nil
}
