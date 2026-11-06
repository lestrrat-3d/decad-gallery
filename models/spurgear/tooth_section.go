package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// returned sketch is solved and verified, and the profile is the bounded
// six-curve tooth region. No solid feature is applied.
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
		return nil, nil, fmt.Errorf("tooth section: angle %v must be finite", angle)
	}
	d := derive(p)
	if d.pitch <= 0 {
		return nil, nil, errors.New("tooth section: cannot derive gear dimensions")
	}
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported; use BuildProfile")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the ordinary six-curve tooth section at angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, d, left, right, true)

	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, tp.right[p.Samples-1], tp.left[p.Samples-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)

	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section", 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}
