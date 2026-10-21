package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth, rotated by angle, as a
// six-curve closed profile on plane. The root arc closes the region from the
// right root to the left root. Embedded gears are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section angle must be finite, got %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section does not support embedded flanks")
	}

	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	rp, lp := flankSketchPoints(s, left, right)
	rootR, rootL, _, _ := rootSketchPoints(s, d, left[0], right[0])

	s.CreateLine(rootR, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section right flank: %w", err)
	}
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section left flank: %w", err)
	}
	s.CreateLine(lp[0], rootL)
	s.CreateArc(center, rootR, rootL)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	profile, err := onlyValidProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	if n := len(profile.Outer); n != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, expected 6", n)
	}
	return s, profile, nil
}
