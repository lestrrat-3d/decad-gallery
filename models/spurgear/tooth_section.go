package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary six-curve tooth section on plane,
// turned by angle about the sketch origin. It returns the solved, verified
// sketch and its profile; it applies no solid features.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: angle must be finite, got %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section: embedded flanks (base radius %g below root radius %g) "+
			"have no standalone tooth section", d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection implements the ordinary standalone tooth section without
// re-validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section: embedded flanks (base radius %g below root radius %g)",
			d.base, d.root)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	last := len(right) - 1

	rightPts := dimensionFlank(s, right, false, false)
	leftPts := dimensionFlank(s, left, false, true)
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	leftRoot := dimensionArcEnd(s, lr)
	rightRoot := dimensionPoint(s, rr)

	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %d outer curves, expected 6", got)
	}
	return s, profile, nil
}
