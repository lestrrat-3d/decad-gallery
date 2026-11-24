package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// toothDrawing records the root points of one drawn ordinary tooth.
type toothDrawing struct {
	rightRoot, leftRoot     *sketch.Point
	rightRootXY, leftRootXY point
}

// drawOrdinaryTooth adds one tooth's two radial lines, two splines and tip arc
// to s from sampled flanks. When prevLeft is non-nil it also adds the root arc
// from that previous left root to this tooth's right root.
func drawOrdinaryTooth(s *sketch.Sketch, d dimensions, left, right []point,
	prevLeft *sketch.Point) (toothDrawing, error) {
	n := len(right)
	center := s.Origin()

	rightPoints := make([]*sketch.Point, n)
	leftPoints := make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rightPoints[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == n-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}

	var t toothDrawing
	t.leftRootXY = rootContact(left[0], d)
	t.rightRootXY = rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)

	if prevLeft != nil {
		s.CreateArc(center, prevLeft, t.rightRoot)
	}
	s.CreateLine(t.rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return toothDrawing{}, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[n-1], leftPoints[n-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return toothDrawing{}, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(leftPoints[0], t.leftRoot)
	return t, nil
}

// BuildToothSection draws one ordinary tooth at the signed angle on plane. The
// section is the six-curve region bounded by the two radial lines, the two flank
// splines, the tip arc and the short root arc. It applies no solid feature.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("tooth section: nil sketch world")
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
		return nil, nil, errors.New("tooth section: embedded flanks are not supported; " +
			"use the complete gear profile")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the ordinary tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
		p.Samples, angle)
	t, err := drawOrdinaryTooth(s, d, left, right, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)

	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: %d boundary curves, expected 6", got)
	}
	return s, profile, nil
}
