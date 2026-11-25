package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth section on plane, rotated by
// angle, and returns its solved, verified sketch and six-curve profile.
// It rejects embedded teeth.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %g is not finite", angle)
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 {
		return nil, nil, errors.New("tooth section: cannot derive gear dimensions")
	}
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks cross the root circle, use BuildProfile")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth at angle: two radial lines, two
// flank splines, the tip arc and the short root arc.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := dimensionTooth(s, left, right, d)
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section right flank: %w", err)
	}
	s.CreateArc(center, tp.right[len(tp.right)-1], tp.left[len(tp.left)-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section left flank: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
