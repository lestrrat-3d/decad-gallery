package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("tooth section requires context, world, and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("standalone tooth section does not support embedded flanks")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func flankPoints(s *sketch.Sketch, left, right []point, embedded bool) ([]*sketch.Point, []*sketch.Point) {
	lp, rp := make([]*sketch.Point, len(left)), make([]*sketch.Point, len(right))
	for i, pt := range right {
		rp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	for i, pt := range left {
		if i == len(left)-1 || (embedded && i == 0) {
			lp[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		lp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return lp, rp
}

func solveSketch(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify sketch: %w", err)
	}
	return nil
}

func singleProfile(s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profile count %d, expected 1", len(profiles))
	}
	p := profiles[0]
	if !p.Valid {
		return nil, fmt.Errorf("profile is invalid")
	}
	if curves > 0 && len(p.Outer) != curves {
		return nil, fmt.Errorf("outer curve count %d, expected %d", len(p.Outer), curves)
	}
	return p, nil
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	lp, rp := flankPoints(s, left, right, false)
	lr, rr := rootContact(left[0], d), rootContact(right[0], d)
	lroot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rroot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	s.CreateLine(rroot, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 left flank: %w", err)
	}
	s.CreateLine(lp[0], lroot)
	s.CreateArc(s.Origin(), rroot, lroot)
	if err := solveSketch(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}

func closingRadius(p Params, d dimensions) (float64, error) {
	ip := involute(d.base, d.pitch)
	beta := math.Pi/(2*float64(p.Teeth)) - math.Atan2(-ip.y, ip.x)
	halfAngle := func(radius float64) float64 {
		v := involute(d.base, radius)
		v = rotate(v.x, -v.y, beta)
		return math.Atan2(v.y, v.x)
	}
	limit := math.Pi / float64(p.Teeth)
	if halfAngle(d.root) >= limit {
		return 0, fmt.Errorf("no embedded tooth closing radius inside root circle")
	}
	if halfAngle(d.base) < limit {
		return d.base, nil
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if halfAngle(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return ((lo+hi)/2 + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	radius, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, radius, d.tip, float64(p.Teeth), p.Samples, 0)
	lp, rp := flankPoints(s, left, right, true)
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 embedded right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 embedded left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if err := solveSketch(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	return s, profile, nil
}
