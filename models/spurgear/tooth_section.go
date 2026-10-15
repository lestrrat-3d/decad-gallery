package spurgear

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth rotated by angle on plane. The
// region is closed by the short root arc from the right root to the left root.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: nil context, world or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle must be finite, got %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section does not support embedded profiles")
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	tp := dimensionTooth(s, left, right, d, true)
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth right flank: %w", err)
	}
	s.CreateArc(center, tp.right[p.Samples-1], tp.left[p.Samples-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth left flank: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)
	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d regions, want 1 valid", len(profiles))
	}
	if got := len(profiles[0].Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d curves, want 6", got)
	}
	return s, profiles[0], nil
}
