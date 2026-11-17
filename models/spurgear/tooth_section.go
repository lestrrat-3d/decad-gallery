package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth section is not supported")
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	rl, rr := rootContact(right[0], d), rootContact(left[0], d)
	rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
	for i := range right {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := range left {
		if i == p.Samples-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	center := s.Origin()
	pl := sketchcheck.DimensionArcEnd(s, rr.x, rr.y)
	pr := sketchcheck.DimensionPoint(s, rl.x, rl.y)
	s.CreateLine(pr, rp[0])
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(lp[0], pl)
	s.CreateArc(center, pr, pl)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth section: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth section: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth profile count %d, expected one valid profile", len(profiles))
	}
	if len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth profile has %d curves, expected 6", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}

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
		return nil, nil, fmt.Errorf("embedded tooth section is not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
