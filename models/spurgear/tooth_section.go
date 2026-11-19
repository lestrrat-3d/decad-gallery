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
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("invalid angle")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth section unsupported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	lr, rr := rootContact(l[0], d), rootContact(r[0], d)
	lroot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rroot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	lp, rp := flankPoints(s, l, r, false)
	s.CreateLine(rroot, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank: %w", err)
	}
	s.CreateLine(lp[0], lroot)
	s.CreateArc(s.Origin(), rroot, lroot)
	profile, err := oneProfile(ctx, s, 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

func flankPoints(s *sketch.Sketch, left, right []point, firstArc bool) ([]*sketch.Point, []*sketch.Point) {
	lp, rp := make([]*sketch.Point, len(left)), make([]*sketch.Point, len(right))
	for i := range right {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := range left {
		if i == len(left)-1 || (firstArc && i == 0) {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	return lp, rp
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	pitch := involute(d.base, d.pitch)
	beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(pitch.y, pitch.x)
	h := func(r float64) float64 {
		q := involute(d.base, r)
		q = rotate(q.x, q.y, beta)
		return math.Abs(math.Atan2(q.y, q.x))
	}
	limit := math.Pi / float64(p.Teeth)
	rc := d.base
	if h(d.base) >= limit {
		if h(d.root) >= limit {
			return nil, nil, fmt.Errorf("tooth does not fit at root circle")
		}
		lo, hi := d.base, d.root
		for range 200 {
			mid := (lo + hi) / 2
			if h(mid) >= limit {
				lo = mid
			} else {
				hi = mid
			}
		}
		rc = ((lo+hi)/2 + d.root) / 2
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	l, r := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	lp, rp := flankPoints(s, l, r, true)
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	profile, err := oneProfile(ctx, s, 4)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}
