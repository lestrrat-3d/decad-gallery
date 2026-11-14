package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth section on plane at the signed
// angle, as a solved and verified six-curve profile. Embedded gears are
// rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil sketch world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: nil sketch plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: embedded flanks have no standalone tooth section")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth at angle. It does not
// validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	last := len(left) - 1
	fp := drawFlankPoints(s, left, right)
	roots := drawRoots(s, d, left[0], right[0])

	s.CreateLine(roots.right, fp.right[0])
	if _, err := s.CreateFitSpline(fp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section right flank: %w", err)
	}
	s.CreateArc(center, fp.right[last], fp.left[last])
	if _, err := s.CreateFitSpline(fp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section left flank: %w", err)
	}
	s.CreateLine(fp.left[0], roots.left)
	s.CreateArc(center, roots.right, roots.left)

	if err := solveVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
