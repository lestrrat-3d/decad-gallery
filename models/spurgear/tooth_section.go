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
		return nil, nil, fmt.Errorf("nil tooth section argument")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("invalid angle")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth section is not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func flankPoints(s *sketch.Sketch, left, right []point, embedded bool) ([]*sketch.Point, []*sketch.Point) {
	lp, rp := make([]*sketch.Point, len(left)), make([]*sketch.Point, len(right))
	for i := range left {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == len(left)-1 || (embedded && i == 0) {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	return lp, rp
}

func sectionProfile(ctx context.Context, s *sketch.Sketch, count int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("profile count %d, expected one valid profile", len(profiles))
	}
	if count > 0 && len(profiles[0].Outer) != count {
		return nil, fmt.Errorf("boundary curves %d, expected %d", len(profiles[0].Outer), count)
	}
	return profiles[0], nil
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	lp, rp := flankPoints(s, l, r, false)
	lroot, rroot := rootContact(l[0], d), rootContact(r[0], d)
	lr := sketchcheck.DimensionArcEnd(s, lroot.x, lroot.y)
	rr := sketchcheck.DimensionPoint(s, rroot.x, rroot.y)
	s.CreateLine(rr, rp[0])
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank: %w", err)
	}
	s.CreateLine(lp[0], lr)
	s.CreateArc(s.Origin(), rr, lr)
	profile, err := sectionProfile(ctx, s, 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	pitchPoint := involute(d.base, d.pitch)
	beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(pitchPoint.y, pitchPoint.x)
	half := func(r float64) float64 {
		v := involute(d.base, r)
		v = rotate(v.x, v.y, beta)
		return math.Atan2(v.y, v.x)
	}
	limit := math.Pi / float64(p.Teeth)
	if half(d.root) >= limit {
		return nil, nil, fmt.Errorf("root circle does not leave room for a tooth")
	}
	rc := d.base
	if half(d.base) >= limit {
		lo, hi := d.base, d.root
		for i := 0; i < 200; i++ {
			mid := (lo + hi) / 2
			if half(mid) >= limit {
				lo = mid
			} else {
				hi = mid
			}
		}
		rc = (hi + d.root) / 2
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	l, r := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	lp, rp := flankPoints(s, l, r, true)
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	profile, err := sectionProfile(ctx, s, 4)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}
