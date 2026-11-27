package spurgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	center := s.Origin()
	rightPoints := make([]*sketch.Point, len(right))
	leftPoints := make([]*sketch.Point, len(left))
	for i, q := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	rr, lr := rootContact(right[0], d), rootContact(left[0], d)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("create right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("create left flank spline: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth section: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth section: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section profile count=%d, expected one valid six-curve profile", len(profiles))
	}
	return s, profiles[0], nil
}

func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane must be nonnil")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded flanks do not have an ordinary tooth section")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
