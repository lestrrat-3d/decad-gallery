package spurgear

import (
	"context"
	"fmt"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
	"math"
)

// BuildToothSection returns a verified six-curve tooth profile rotated by angle
// in the supplied plane. It rejects embedded flanks and nonfinite angles.
// ctx, w, and plane must not be nil.
func BuildToothSection(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || plane == nil {
		return nil, nil, fmt.Errorf("nil context, world or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("invalid angle")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth section is unsupported")
	}
	return buildToothSection(ctx, w, plane, p, d, angle)
}
func flankPoints(s *sketch.Sketch, l, r []point, embedded bool) ([]*sketch.Point, []*sketch.Point) {
	lp, rp := make([]*sketch.Point, len(l)), make([]*sketch.Point, len(r))
	for i := range l {
		rp[i] = sketchcheck.DimensionPoint(s, r[i].x, r[i].y)
		if i == len(l)-1 || (embedded && i == 0) {
			lp[i] = sketchcheck.DimensionArcEnd(s, l[i].x, l[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, l[i].x, l[i].y)
		}
	}
	return lp, rp
}
func finishSketch(ctx context.Context, s *sketch.Sketch, count int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profile count %d, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("profile is invalid")
	}
	if len(profiles[0].Outer) != count {
		return nil, fmt.Errorf("profile has %d curves, expected %d", len(profiles[0].Outer), count)
	}
	return profiles[0], nil
}
func buildToothSection(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	lp, rp := flankPoints(s, l, r, false)
	lr, rr := rootContact(l[0], d), rootContact(r[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	s.CreateLine(rightRoot, rp[0])
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 right spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 left spline: %w", err)
	}
	s.CreateLine(lp[0], leftRoot)
	s.CreateArc(s.Origin(), rightRoot, leftRoot)
	profile, err := finishSketch(ctx, s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}
func buildEmbeddedTooth(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	half := func(r float64) float64 {
		l, _ := sampleFlanks(d.pitch, d.base, r, d.tip, float64(p.Teeth), d.backlash, 3, 0)
		return math.Atan2(l[0].y, l[0].x)
	}
	limit := math.Pi / float64(p.Teeth)
	if half(d.root) >= limit {
		return nil, nil, fmt.Errorf("root circle has overlapping teeth")
	}
	rc := d.base
	if half(d.base) >= limit {
		lo, hi := d.base, d.root
		for range 200 {
			mid := (lo + hi) / 2
			if half(mid) >= limit {
				lo = mid
			} else {
				hi = mid
			}
		}
		rc = (hi + d.root) / 2
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	l, r := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	lp, rp := flankPoints(s, l, r, true)
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth 0 right spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth 0 left spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	profile, err := finishSketch(ctx, s, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth section: %w", err)
	}
	return s, profile, nil
}
