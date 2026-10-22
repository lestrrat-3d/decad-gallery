package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// orderedTooth holds the dimensioned sketch points of one ordinary tooth.
type orderedTooth struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// scaleToRoot places a base-circle sample on the root circle. It multiplies
// before dividing so the result matches the root contact used by the join.
func scaleToRoot(q point, d dimensions) point {
	return point{q.x * d.root / d.base, q.y * d.root / d.base}
}

// dimensionOrdinaryTooth creates the dimensioned flank and root points of one
// ordinary tooth from its rotated flank samples.
func dimensionOrdinaryTooth(s *sketch.Sketch, d dimensions, leftXY, rightXY []point) orderedTooth {
	var t orderedTooth
	for _, q := range rightXY {
		t.right = append(t.right, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	last := len(leftXY) - 1
	for i, q := range leftXY {
		if i == last {
			t.left = append(t.left, sketchcheck.DimensionArcEnd(s, q.x, q.y))
			continue
		}
		t.left = append(t.left, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	t.leftRootXY = scaleToRoot(leftXY[0], d)
	t.rightRootXY = scaleToRoot(rightXY[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)
	return t
}

// drawOrdinaryTooth draws the right line, right spline, tip arc, left spline
// and left line of one tooth.
func drawOrdinaryTooth(s *sketch.Sketch, t orderedTooth) error {
	center := s.Origin()
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, t.right[len(t.right)-1], t.left[len(t.left)-1])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// solveVerify solves s and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// onlyProfile returns the single valid profile of a solved, verified sketch.
func onlyProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// buildOrdinaryTooth draws one six-curve ordinary tooth at angle on plane.
// It assumes p and d are already validated and d.base >= d.root.
func buildOrdinaryTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	t := dimensionOrdinaryTooth(s, d, left, right)
	if err := drawOrdinaryTooth(s, t); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	// The short root arc runs from the right root to the left root.
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if n := len(profile.Outer); n != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, expected 6", n)
	}
	return s, profile, nil
}

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane and
// returns its solved, verified sketch and its six-curve profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("tooth section: nil plane")
	}
	d, err := validate(p)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %v is not finite", angle)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %v below root radius %v) "+
			"are not supported", d.base, d.root)
	}
	return buildOrdinaryTooth(ctx, world, plane, p, d, angle)
}
