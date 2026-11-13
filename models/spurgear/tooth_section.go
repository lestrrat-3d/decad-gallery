package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth section at the given angle on
// plane. The section has six curves; no solid feature is applied.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("tooth section does not support embedded flanks")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection builds the six-curve ordinary tooth section without
// validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
		p.Samples, angle)
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	leftPts, rightPts := dimensionedFlanks(s, left, right)
	last := p.Samples - 1
	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank: %w", err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)
	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: expected 6 boundary curves, got %d", got)
	}
	return s, profile, nil
}
