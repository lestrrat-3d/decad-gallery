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
	rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
	for i := range right {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	rr, lr := rootContact(right[0], d), rootContact(left[0], d)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
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
		return nil, nil, fmt.Errorf("solve tooth section: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth section: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth section must have one valid profile")
	}
	if len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d curves, want 6", len(profiles[0].Outer))
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
		return nil, nil, fmt.Errorf("tooth angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth flanks do not support standalone sections")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func embeddedClosingRadius(d dimensions, teeth int, backlash float64) (float64, error) {
	half := func(r float64) float64 {
		p := involute(d.base, r)
		p.y = -p.y
		pp := involute(d.base, d.pitch)
		pp.y = -pp.y
		beta := math.Pi/(2*float64(teeth)) - backlash/(2*d.pitch) - math.Atan2(pp.y, pp.x)
		p = rotate(p.x, p.y, beta)
		return math.Abs(math.Atan2(p.y, p.x))
	}
	limit := math.Pi / float64(teeth)
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("no embedded tooth closing radius inside root circle")
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if half(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (hi + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(d, p.Teeth, d.backlash)
	if err != nil {
		return nil, nil, err
	}
	l, r := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
	for i := range r {
		rp[i] = sketchcheck.DimensionPoint(s, r[i].x, r[i].y)
	}
	for i := range l {
		if i == 0 || i == len(l)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, l[i].x, l[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, l[i].x, l[i].y)
		}
	}
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve embedded tooth: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify embedded tooth: %w", err)
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid || len(ps[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth profile has %d regions", len(ps))
	}
	return s, ps[0], nil
}
