package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// flankPoints holds the dimensioned sketch points of one tooth's flanks.
type flankPoints struct {
	left, right []*sketch.Point
}

// dimensionFlanks creates the sketch points of one tooth's flanks. Every
// right point and every left point but the tip receives both coordinate
// dimensions; the left tip receives one, and the tip arc supplies the other.
func dimensionFlanks(s *sketch.Sketch, left, right []point) flankPoints {
	fp := flankPoints{
		left:  make([]*sketch.Point, len(left)),
		right: make([]*sketch.Point, len(right)),
	}
	for i, q := range right {
		fp.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	last := len(left) - 1
	for i, q := range left {
		if i == last {
			fp.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		fp.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	return fp
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	return nil
}

// singleProfile returns the one valid profile of a solved, verified sketch.
func singleProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// BuildToothSection draws one ordinary tooth section on plane, turned by a
// signed angle about the sketch origin, and returns the solved, verified
// sketch and its six-curve profile. The section is closed by the short root
// arc from the right root to the left root. Embedded gears have no
// standalone tooth section.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: angle must be finite, got %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section: embedded flanks (base radius %g below root radius %g) "+
			"have no standalone tooth section", d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section at angle
// without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	const stage = "tooth section"
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: create sketch: %w", stage, err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	fp := dimensionFlanks(s, left, right)
	leftRootXY := rootContact(left[0], d)
	rightRootXY := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)
	last := p.Samples - 1

	s.CreateLine(rightRoot, fp.right[0])
	if _, err := s.CreateFitSpline(fp.right...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: right flank: %w", stage, err)
	}
	s.CreateArc(center, fp.right[last], fp.left[last])
	if _, err := s.CreateFitSpline(fp.left...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: %s: left flank: %w", stage, err)
	}
	s.CreateLine(fp.left[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, stage)
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: %s: got %d outer curves, expected 6", stage, got)
	}
	return s, profile, nil
}
