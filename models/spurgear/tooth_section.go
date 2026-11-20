package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("tooth section requires context, world and plane")
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth section is not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection constructs one closed, ordinary tooth in the supplied frame.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	leftRoot, rightRoot := rootContact(left[0], d), rootContact(right[0], d)
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
	rr := sketchcheck.DimensionPoint(s, rightRoot.x, rightRoot.y)
	lr := sketchcheck.DimensionArcEnd(s, leftRoot.x, leftRoot.y)
	s.CreateLine(rr, rp[0])
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(lp[0], lr)
	s.CreateArc(s.Origin(), rr, lr)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth sketch: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth sketch: %w", err)
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

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
	for i, v := range right {
		rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
	}
	for i, v := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
	}
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded left flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve embedded tooth: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify embedded tooth: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("embedded tooth profile count %d, expected one valid profile", len(profiles))
	}
	if len(profiles[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth has %d curves, expected 4", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}

func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	half := func(r float64) float64 {
		v := involute(d.base, r)
		v.y = -v.y
		pitchPoint := involute(d.base, d.pitch)
		pitchPoint.y = -pitchPoint.y
		beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(pitchPoint.y, pitchPoint.x)
		v = rotate(v.x, v.y, beta)
		return math.Abs(math.Atan2(v.y, v.x))
	}
	limit := math.Pi / float64(p.Teeth)
	if half(d.root) >= limit {
		return 0, fmt.Errorf("no embedded tooth closing radius inside root circle")
	}
	if half(d.base) < limit {
		return d.base, nil
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return ((lo+hi)/2 + d.root) / 2, nil
}
