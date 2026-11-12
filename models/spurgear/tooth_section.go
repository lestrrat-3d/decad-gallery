package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth section on plane, turned by angle
// about the sketch origin, and returns the solved, verified sketch and its
// six-curve profile. ctx, world and plane must not be nil. Gears whose base
// circle lies inside the root circle (embedded flanks) are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section angle must be finite, got %g", angle)
	}
	d, err := deriveChecked(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: derive dimensions: %w", err)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: standalone tooth section does not support embedded flanks "+
			"(base radius %g < root radius %g)", d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section at angle.
// It does not validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create tooth section sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	lp, rp := createFlankPoints(s, left, right)
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	last := len(rp) - 1

	s.CreateLine(rightRoot, rp[0])
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section right flank: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section left flank: %w", err)
	}
	s.CreateLine(lp[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveVerified(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleValidProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, want 6", len(profile.Outer))
	}
	return s, profile, nil
}

// rootContact scales a flank's base point onto the root circle, multiplying
// each coordinate before dividing.
func rootContact(basePoint point, d dimensions) point {
	return point{basePoint.x * d.root / d.base, basePoint.y * d.root / d.base}
}

// createFlankPoints creates the dimensioned fit points of one tooth. Every
// right point receives both coordinate dimensions; every left point does too,
// except its tip, which the tip arc places on the tip circle.
func createFlankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
	rp := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	lp := make([]*sketch.Point, len(left))
	last := len(left) - 1
	for i, pt := range left {
		if i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		lp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return lp, rp
}

// solveVerified solves s and requires it to be fully constrained.
func solveVerified(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s sketch: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s sketch: %w", stage, err)
	}
	return nil
}

// singleValidProfile returns the one valid profile of a verified sketch.
func singleValidProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	var found *sketch.Profile
	count := 0
	for _, profile := range s.Profiles() {
		if !profile.Valid {
			continue
		}
		found = profile
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("spurgear: %s sketch has %d valid profiles, want 1", stage, count)
	}
	return found, nil
}

// profileGuards rejects nil arguments to the profile APIs.
func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	switch {
	case ctx == nil:
		return errors.New("spurgear: nil context")
	case world == nil:
		return errors.New("spurgear: nil sketch world")
	case plane == nil:
		return errors.New("spurgear: nil sketch plane")
	}
	return nil
}
