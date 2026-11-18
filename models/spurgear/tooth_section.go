package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// sketch is solved and verified, and its six-curve profile is returned.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth at the given angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	fp := createOrdinaryPoints(s, d, left, right)
	n := p.Samples

	s.CreateLine(fp.rightRoot, fp.right[0])
	if _, err := s.CreateFitSpline(fp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section right flank: %w", err)
	}
	s.CreateArc(center, fp.right[n-1], fp.left[n-1])
	if _, err := s.CreateFitSpline(fp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section left flank: %w", err)
	}
	s.CreateLine(fp.left[0], fp.leftRoot)
	s.CreateArc(center, fp.rightRoot, fp.leftRoot)

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("tooth section solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section verify: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
