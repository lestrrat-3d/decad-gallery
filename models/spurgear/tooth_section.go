package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	half := math.Pi / float64(p.Teeth)
	angleAt := func(r float64) float64 {
		t := math.Tan(math.Acos(d.base / r))
		x := d.base * (math.Cos(t) + t*math.Sin(t))
		y := -d.base * (math.Sin(t) - t*math.Cos(t))
		tp := math.Tan(math.Acos(d.base / d.pitch))
		pitchAngle := math.Atan2(-d.base*(math.Sin(tp)-tp*math.Cos(tp)), d.base*(math.Cos(tp)+tp*math.Sin(tp)))
		beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - pitchAngle
		q := rotate(x, y, beta)
		return math.Abs(math.Atan2(q.y, q.x))
	}
	if angleAt(d.root) >= half {
		return 0, fmt.Errorf("no closing radius exists inside the root circle")
	}
	if angleAt(d.base) < half {
		return d.base, nil
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if angleAt(mid) > half {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (hi + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
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
		return nil, nil, fmt.Errorf("embedded tooth has %d valid profiles, expected 1", len(profiles))
	}
	if got := len(profiles[0].Outer); got != 4 {
		return nil, nil, fmt.Errorf("embedded tooth has %d boundary curves, expected 4", got)
	}
	return s, profiles[0], nil
}

func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	center := s.Origin()
	leftRootXY := rootContact(left[0], d)
	rightRootXY := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)
	rightPoints := make([]*sketch.Point, len(right))
	leftPoints := make([]*sketch.Point, len(left))
	for i, q := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	s.CreateLine(rightRoot, rightPoints[0])
	if _, err = s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[len(right)-1], leftPoints[len(left)-1])
	if _, err = s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth section: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth section: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth section has %d valid profiles, expected 1", len(profiles))
	}
	if len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d boundary curves, expected 6", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}

// BuildToothSection creates a fully constrained ordinary tooth profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	angle float64) (*sketch.Sketch, *sketch.Profile, error) {
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
		return nil, nil, fmt.Errorf("embedded tooth section is unsupported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
