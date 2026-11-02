package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection returns one solved ordinary tooth on plane. Arguments ctx, world, and plane must not be nil.
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
	l, r := make([]*sketch.Point, len(left)), make([]*sketch.Point, len(right))
	for i := range right {
		r[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := range left {
		if i == len(left)-1 || embedded && i == 0 {
			l[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			continue
		}
		l[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
	}
	return l, r
}

func drawFlanks(s *sketch.Sketch, left, right []*sketch.Point) error {
	if _, err := s.CreateFitSpline(right...); err != nil {
		return fmt.Errorf("right flank: %w", err)
	}
	s.CreateArc(s.Origin(), right[len(right)-1], left[len(left)-1])
	if _, err := s.CreateFitSpline(left...); err != nil {
		return fmt.Errorf("left flank: %w", err)
	}
	return nil
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	l, r := flankPoints(s, left, right, false)
	lc, rc := rootContact(left[0], d), rootContact(right[0], d)
	lroot := sketchcheck.DimensionArcEnd(s, lc.x, lc.y)
	rroot := sketchcheck.DimensionPoint(s, rc.x, rc.y)
	s.CreateLine(rroot, r[0])
	if err := drawFlanks(s, l, r); err != nil {
		return nil, nil, fmt.Errorf("tooth 0: %w", err)
	}
	s.CreateLine(l[0], lroot)
	s.CreateArc(s.Origin(), rroot, lroot)
	profile, err := solveSingle(ctx, s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}

func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	beta := flankBeta(d.pitch, d.base, float64(p.Teeth))
	halfAngle := func(radius float64) float64 {
		q := involute(d.base, radius)
		q = rotate(q.x, -q.y, beta)
		return math.Atan2(q.y, q.x)
	}
	limit := math.Pi / float64(p.Teeth)
	if halfAngle(d.base) < limit {
		return d.base, nil
	}
	if halfAngle(d.root) >= limit {
		return 0, fmt.Errorf("root circle leaves no embedded tooth closing radius")
	}
	low, high := d.base, d.root
	for range 200 {
		mid := (low + high) / 2
		if halfAngle(mid) >= limit {
			low = mid
		} else {
			high = mid
		}
	}
	return ((low+high)/2 + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	closing, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, closing, d.tip, float64(p.Teeth), p.Samples, 0)
	l, r := flankPoints(s, left, right, true)
	if err := drawFlanks(s, l, r); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth 0: %w", err)
	}
	s.CreateArc(s.Origin(), r[0], l[0])
	profile, err := solveSingle(ctx, s, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth section: %w", err)
	}
	return s, profile, nil
}
