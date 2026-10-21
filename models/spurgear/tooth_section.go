package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth, rotated by angle, as a
// six-curve profile closed by the short root arc.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section: embedded flanks are not supported")
	}

	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: create sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	t := addFlankPoints(s, left, right)
	addRootPoints(s, &t, left, right, d)

	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: right flank: %w", err)
	}
	s.CreateArc(s.Origin(), t.right[p.Samples-1], t.left[p.Samples-1])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: left flank: %w", err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)

	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	prof, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if got := len(prof.Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section: got %d outer curves, expected 6", got)
	}
	return s, prof, nil
}
