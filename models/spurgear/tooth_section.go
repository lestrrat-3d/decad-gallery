package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth on plane, rotated by angle,
// as a six-curve section: right radial line, right spline, tip arc, left
// spline, left radial line and the short root arc. It applies no solid
// feature. Embedded gears are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: embedded flanks have no standalone tooth section")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws one ordinary tooth at the supplied angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	t := newTooth(s, d, left, right)
	center := s.Origin()
	s.CreateLine(t.rightRoot, t.rightPts[0])
	if _, err := s.CreateFitSpline(t.rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: right flank spline: %w", err)
	}
	s.CreateArc(center, t.rightPts[len(t.rightPts)-1], t.leftPts[len(t.leftPts)-1])
	if _, err := s.CreateFitSpline(t.leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: left flank spline: %w", err)
	}
	s.CreateLine(t.leftPts[0], t.leftRoot)
	s.CreateArc(center, t.rightRoot, t.leftRoot)
	prof, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if n := len(prof.Outer); n != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, want 6", n)
	}
	return s, prof, nil
}
