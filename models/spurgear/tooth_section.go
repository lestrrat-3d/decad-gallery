package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection returns one ordinary tooth on plane, rotated by angle.
// The context, world, and plane must not be nil.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("tooth section requires context, world, and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if !finite(angle) {
		return nil, nil, fmt.Errorf("tooth section angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("standalone tooth section does not support embedded flanks")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

func flankPoints(s *sketch.Sketch, left, right []point, embedded bool) ([]*sketch.Point, []*sketch.Point) {
	l, r := make([]*sketch.Point, len(left)), make([]*sketch.Point, len(right))
	for i := range right {
		r[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := range left {
		if i == len(left)-1 || (embedded && i == 0) {
			l[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			continue
		}
		l[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
	}
	return l, r
}

func solveVerified(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify sketch: %w", err)
	}
	return nil
}

func oneProfile(s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profile count %d, expected 1", len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, fmt.Errorf("expected one valid hole-free profile")
	}
	if len(profile.Outer) != curves {
		return nil, fmt.Errorf("profile curve count %d, expected %d", len(profile.Outer), curves)
	}
	return profile, nil
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	l, r := flankPoints(s, left, right, false)
	lc, rc := rootContact(left[0], d), rootContact(right[0], d)
	lroot := sketchcheck.DimensionArcEnd(s, lc.x, lc.y)
	rroot := sketchcheck.DimensionPoint(s, rc.x, rc.y)
	s.CreateLine(rroot, r[0])
	if _, err := s.CreateFitSpline(r...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 right flank: %w", err)
	}
	s.CreateArc(s.Origin(), r[len(r)-1], l[len(l)-1])
	if _, err := s.CreateFitSpline(l...); err != nil {
		return nil, nil, fmt.Errorf("tooth 0 left flank: %w", err)
	}
	s.CreateLine(l[0], lroot)
	s.CreateArc(s.Origin(), rroot, lroot)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := oneProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}

func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	beta := flankBeta(d.pitch, d.base, float64(p.Teeth))
	halfAngle := func(radius float64) float64 {
		v := involute(d.base, radius)
		l := rotate(v.x, -v.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / float64(p.Teeth)
	if halfAngle(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth cannot close inside the root circle")
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
	radius, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, radius, d.tip, float64(p.Teeth), p.Samples, 0)
	l, r := flankPoints(s, left, right, true)
	if _, err := s.CreateFitSpline(r...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth 0 right flank: %w", err)
	}
	s.CreateArc(s.Origin(), r[len(r)-1], l[len(l)-1])
	if _, err := s.CreateFitSpline(l...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth 0 left flank: %w", err)
	}
	s.CreateArc(s.Origin(), r[0], l[0])
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth section: %w", err)
	}
	profile, err := oneProfile(s, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth section: %w", err)
	}
	return s, profile, nil
}
