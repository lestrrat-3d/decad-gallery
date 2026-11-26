package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints are the dimensioned sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootR, rootL        point
}

// flankPoints creates the dimensioned flank points. Every right point and every
// left point except the tip receives both coordinate dimensions; the left tip
// receives one.
func flankPoints(s *sketch.Sketch, left, right []point) (rp, lp []*sketch.Point) {
	rp = make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp = make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rp, lp
}

// ordinaryToothPoints creates the flank points and both root points of one
// ordinary tooth. The left root is created before the right root.
func ordinaryToothPoints(s *sketch.Sketch, d dimensions, left, right []point) toothPoints {
	var tp toothPoints
	tp.right, tp.left = flankPoints(s, left, right)
	tp.rootL = rootContact(left[0], d)
	tp.rootR = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.rootL.x, tp.rootL.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rootR.x, tp.rootR.y)
	return tp
}

// drawOrdinaryFlanks draws the right line, right spline, tip arc, left spline
// and left line of one tooth, in that order.
func drawOrdinaryFlanks(s *sketch.Sketch, tp toothPoints) error {
	last := len(tp.right) - 1
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return fmt.Errorf("left flank: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	return nil
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

// singleProfile requires exactly one valid profile of s with the given outer
// curve count.
func singleProfile(s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("got %d profiles, expected 1", len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, errors.New("profile is not valid")
	}
	if curves > 0 && len(profile.Outer) != curves {
		return nil, fmt.Errorf("got %d boundary curves, expected %d", len(profile.Outer), curves)
	}
	return profile, nil
}

// BuildToothSection builds one ordinary tooth section on plane, rotated by
// angle. The sketch is solved and verified; the profile has six curves.
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
		return nil, nil, errors.New("tooth section: angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection builds one six-curve ordinary tooth at angle without
// validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := ordinaryToothPoints(s, d, left, right)
	if err := drawOrdinaryFlanks(s, tp); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), tp.rightRoot, tp.leftRoot)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}
