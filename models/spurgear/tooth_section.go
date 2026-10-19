package spurgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth, rotated by angle, closed by the
// short root arc. Embedded flanks are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("spurgear: tooth section angle %v must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section does not support embedded flanks (base %v < root %v)",
			d.base, d.root)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	lp, rp := flankPoints(s, left, right)
	lr := rootCoords(left[0], d)
	rr := rootCoords(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)

	s.CreateLine(rightRoot, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section right flank spline: %w", err)
	}
	s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section left flank spline: %w", err)
	}
	s.CreateLine(lp[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section: got %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
