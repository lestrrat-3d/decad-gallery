package spurgear

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds the six-curve section of one ordinary tooth, turned
// by angle, on plane. It returns the verified sketch and its profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("build tooth section: context, world and plane must not be nil")
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("build tooth section: %w", err)
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("build tooth section: angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("build tooth section: embedded tooth flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection builds one ordinary tooth section turned by angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
		p.Samples, angle)
	tp := addToothPoints(s, d, left, right)
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section right flank: %w", err)
	}
	s.CreateArc(center, tp.right[p.Samples-1], tp.left[p.Samples-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section left flank: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)
	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
