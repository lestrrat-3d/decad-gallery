package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds the six-curve section of one ordinary tooth turned
// by angle on plane. The section is solved and fully verified.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("build tooth section: nil context, world or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("build tooth section: %w", err)
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("build tooth section: angle %g must be finite", angle)
	}
	d := derive(p)
	if d == (dimensions{}) {
		return nil, nil, errors.New("build tooth section: cannot derive dimensions")
	}
	if d.base < d.root {
		return nil, nil, errors.New("build tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection builds the ordinary tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
		p.Samples, angle)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	lp, rp, lRoot, rRoot, _, _ := drawFlankPoints(s, left, right, d)
	s.CreateLine(rRoot, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("tooth right flank: %w", err)
	}
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("tooth left flank: %w", err)
	}
	s.CreateLine(lp[0], lRoot)
	s.CreateArc(center, rRoot, lRoot)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: boundary curves %d, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
