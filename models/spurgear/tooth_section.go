package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
	for i := range right {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	rootR := point{right[0].x * d.root / d.base, right[0].y * d.root / d.base}
	rootL := point{left[0].x * d.root / d.base, left[0].y * d.root / d.base}
	rightRoot := sketchcheck.DimensionPoint(s, rootR.x, rootR.y)
	leftRoot := sketchcheck.DimensionArcEnd(s, rootL.x, rootL.y)
	s.CreateLine(rightRoot, rp[0])
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(lp[0], leftRoot)
	s.CreateArc(s.Origin(), rightRoot, leftRoot)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth sketch: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth profile count/validity: got %d", len(profiles))
	}
	if len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth profile has %d curves, want 6", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}

func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("standalone tooth is unsupported for embedded flanks")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	half := func(r float64) float64 {
		t := math.Tan(math.Acos(d.base / r))
		x := d.base * (math.Cos(t) + t*math.Sin(t))
		y := -d.base * (math.Sin(t) - t*math.Cos(t))
		inv := math.Tan(p.PressureAngle) - p.PressureAngle
		beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) + inv
		v := rotate(x, y, beta)
		return math.Abs(math.Atan2(v.y, v.x))
	}
	limit := math.Pi / float64(p.Teeth)
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("no closing radius inside root circle")
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) > limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return ((lo+hi)/2 + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("closing radius: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
	for i := range right {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify: %w", err)
	}
	ps := s.Profiles()
	if len(ps) != 1 {
		return nil, nil, fmt.Errorf("embedded tooth has %d profiles, want 1", len(ps))
	}
	if !ps[0].Valid {
		return nil, nil, fmt.Errorf("embedded tooth profile is invalid")
	}
	if len(ps[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth profile has %d outer curves, want 4", len(ps[0].Outer))
	}
	return s, ps[0], nil
}
