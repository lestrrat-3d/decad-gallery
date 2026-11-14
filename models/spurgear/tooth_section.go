package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth section at the signed angle on
// plane. It returns the solved, verified sketch and its six-curve profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	d, err := validatedDimensions(p)
	if err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth angle %g must be finite", angle)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded flanks are not supported for a standalone tooth section")
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
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	t := createToothPoints(s, left, right, d)
	if err := drawToothCurves(s, t); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveVerified(ctx, s, "tooth section sketch"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section sketch")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d boundary curves, want 6", len(profile.Outer))
	}
	return s, profile, nil
}
