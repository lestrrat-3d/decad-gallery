package spurgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection returns one solved, fully constrained ordinary tooth at angle radians.
// ctx, world, and plane must not be nil. The angle must be finite.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("build tooth section: context, world, and plane must not be nil")
	}
	if err := validateParams(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("build tooth section: angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("build tooth section: embedded flanks require a complete gear profile")
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	l, r := flankPoints(s, left, right)
	leftRoot, rightRoot, _, _ := rootPoints(s, left[0], right[0], d)
	s.CreateLine(rightRoot, r[0])
	if err := drawFlanks(s, l, r, 0); err != nil {
		return nil, nil, err
	}
	s.CreateLine(l[0], leftRoot)
	s.CreateArc(s.Origin(), rightRoot, leftRoot)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}
