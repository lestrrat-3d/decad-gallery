package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// profile has six curves: two radial lines, two flank splines, the tip arc
// and the short root arc.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: BuildToothSection requires a context, world and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle must be finite, got %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: embedded tooth flanks have no standalone tooth section")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	t := addToothPoints(s, left, right, d)
	if err := drawFlanks(s, t, 0); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveAndVerify(ctx, s, "tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d curves, want 6", len(profile.Outer))
	}
	return s, profile, nil
}
