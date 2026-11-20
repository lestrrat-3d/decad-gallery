package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth section on plane, turned by angle
// about the sketch origin, and returns the solved, verified sketch and its
// six-curve profile. Embedded gears (base radius below root radius) are rejected.
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
		return nil, nil, fmt.Errorf("tooth section: angle must be finite, got %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %v below root radius %v) are not supported",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth at angle: right radial line,
// right flank, tip arc, left flank, left radial line, and the short root arc.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	last := p.Samples - 1

	rightPoints := make([]*sketch.Point, p.Samples)
	for i, pt := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	leftPoints := make([]*sketch.Point, p.Samples)
	for i, pt := range left {
		if i == last {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		leftPoints[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)

	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("tooth section: solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: verify: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth section: got %d profiles, want 1 valid profile", len(profiles))
	}
	profile := profiles[0]
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, want 6", got)
	}
	return s, profile, nil
}
