package spurgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	leftPoints := make([]*sketch.Point, len(left))
	rightPoints := make([]*sketch.Point, len(right))
	for i := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := range left {
		if i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			continue
		}
		leftPoints[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
	}
	rightRoot := rootContact(right[0], d)
	leftRoot := rootContact(left[0], d)
	rr := sketchcheck.DimensionPoint(s, rightRoot.x, rightRoot.y)
	lr := sketchcheck.DimensionArcEnd(s, leftRoot.x, leftRoot.y)
	s.CreateLine(rr, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("create right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("create left flank spline: %w", err)
	}
	s.CreateLine(leftPoints[0], lr)
	s.CreateArc(s.Origin(), rr, lr)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth section: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth section: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth section has %d valid profiles, expected 1", len(profiles))
	}
	if len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d curves, expected 6", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}

func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("tooth angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth sections are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
