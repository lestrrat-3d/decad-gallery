package spurgear

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth rotated by angle on plane.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: nil context, world, or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("spurgear: nonfinite tooth angle %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section does not support embedded profiles")
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	lp, rp := dimensionFlanks(s, left, right)
	lrc := rootContact(d, left[0])
	rrc := rootContact(d, right[0])
	leftRoot := sketchcheck.DimensionArcEnd(s, lrc.x, lrc.y)
	rightRoot := sketchcheck.DimensionPoint(s, rrc.x, rrc.y)
	s.CreateLine(rightRoot, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section right flank: %w", err)
	}
	s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section left flank: %w", err)
	}
	s.CreateLine(lp[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)
	if err := solveVerify(ctx, s, "spurgear tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section curves: got %d, want 6", len(profile.Outer))
	}
	return s, profile, nil
}
