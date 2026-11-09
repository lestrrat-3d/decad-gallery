package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth section is unsupported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	lp, rp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
	for i := range p.Samples {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == p.Samples-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	lroot, rroot := rootContact(left[0], d), rootContact(right[0], d)
	lr := sketchcheck.DimensionArcEnd(s, lroot.x, lroot.y)
	rr := sketchcheck.DimensionPoint(s, rroot.x, rroot.y)
	s.CreateLine(rr, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[p.Samples-1], lp[p.Samples-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, err
	}
	s.CreateLine(lp[0], lr)
	s.CreateArc(s.Origin(), rr, lr)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("invalid tooth profile")
	}
	return s, profiles[0], nil
}
