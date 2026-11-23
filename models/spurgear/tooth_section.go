package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// emitDrawnTooth holds the root points and root coordinates of one drawn tooth.
type emitDrawnTooth struct {
	rightRoot, leftRoot     *sketch.Point
	rightRootXY, leftRootXY point
}

// emitDrawOrdinaryTooth draws one ordinary tooth turned by angle: both flank
// splines, both radial lines and the tip arc. When prevLeft is non-nil it also
// draws the root arc from the previous tooth's left root to this right root.
func emitDrawOrdinaryTooth(s *sketch.Sketch, p Params, d dimensions, angle float64,
	prevLeft *sketch.Point, label int) (emitDrawnTooth, error) {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	n := len(right)
	rightPoints := make([]*sketch.Point, n)
	leftPoints := make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rightPoints[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == n-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	rootRight := rootContact(right[0], d)
	rootLeft := rootContact(left[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, rootLeft.x, rootLeft.y)
	rightRoot := sketchcheck.DimensionPoint(s, rootRight.x, rootRight.y)

	center := s.Origin()
	if prevLeft != nil {
		s.CreateArc(center, prevLeft, rightRoot)
	}
	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return emitDrawnTooth{}, fmt.Errorf("tooth %d right flank spline: %w", label, err)
	}
	s.CreateArc(center, rightPoints[n-1], leftPoints[n-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return emitDrawnTooth{}, fmt.Errorf("tooth %d left flank spline: %w", label, err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	return emitDrawnTooth{
		rightRoot: rightRoot, leftRoot: leftRoot,
		rightRootXY: rootRight, leftRootXY: rootLeft,
	}, nil
}

// buildToothSection builds the six-curve section of one ordinary tooth turned
// by angle on plane. It applies no tooth-index rotation.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	tooth, err := emitDrawOrdinaryTooth(s, p, d, angle, nil, 0)
	if err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), tooth.rightRoot, tooth.leftRoot)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve tooth sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify tooth sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth sketch: got %d profiles, want exactly one valid profile", len(profiles))
	}
	profile := profiles[0]
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildToothSection builds one ordinary tooth section at a signed angle on
// plane. The sketch is solved and verified. Embedded gears are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("tooth section: nil sketch world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle %g", angle)
	}
	d := derive(p)
	if d.base == 0 {
		return nil, nil, fmt.Errorf("tooth section: derive dimensions failed")
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %g below root radius %g) are not supported",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
