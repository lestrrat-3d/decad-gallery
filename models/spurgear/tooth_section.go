package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth, rotated by angle (radians), on
// plane. The profile has six curves: right radial line, right spline, tip arc,
// left spline, left radial line, and the short root arc.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spur gear tooth section: angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spur gear tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spur gear tooth section sketch: %w", err)
	}
	t := ordinaryToothPoints(s, p, d, angle)
	if err := drawFlanks(s, t, "spur gear tooth section"); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	profile, err := solveSingleProfile(ctx, s, "spur gear tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spur gear tooth section: got %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
