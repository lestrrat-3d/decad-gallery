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
	if err := finiteAngle(angle); err != nil {
		return nil, nil, err
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth sections are unsupported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	rr, lr := rootContact(right[0], d), rootContact(left[0], d)
	rroot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	lroot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rp := make([]*sketch.Point, len(right))
	lp := make([]*sketch.Point, len(left))
	for i, v := range right {
		rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
	}
	for i, v := range left {
		if i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
	}
	s.CreateLine(rroot, rp[0])
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank: %w", err)
	}
	s.CreateLine(lp[0], lroot)
	s.CreateArc(s.Origin(), rroot, lroot)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth topology: got %d profiles, expected one profile with six curves", len(profiles))
	}
	return s, profiles[0], nil
}

func finiteAngle(v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("angle must be finite")
	}
	return nil
}
