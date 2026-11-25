package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothSketch records the corners of one tooth drawn by drawTooth.
type toothSketch struct {
	rightRoot, leftRoot     *sketch.Point
	rightRootXY, leftRootXY point
}

// BuildToothSection draws one ordinary tooth, turned by angle, on plane and
// returns its solved, verified sketch and its six-curve profile. The profile is
// closed by the short root arc. ctx, world and plane must not be nil. An
// embedded tooth is rejected.
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
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %g below root radius %g) "+
			"have no standalone tooth section", d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the ordinary tooth turned by angle, without
// validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	tooth, err := drawTooth(s, d, left, right, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), tooth.rightRoot, tooth.leftRoot)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := soleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d boundary curves, want 6", got)
	}
	return s, profile, nil
}

// drawTooth draws one ordinary tooth's two radial lines, two flank splines and
// tip arc. When prevLeftRoot is not nil, it first joins that point to the new
// right root with a root arc. The caller closes the section.
func drawTooth(s *sketch.Sketch, d dimensions, left, right []point,
	prevLeftRoot *sketch.Point) (toothSketch, error) {
	center := s.Origin()
	rightPoints := dimensionRightFlank(s, right)
	leftPoints := dimensionLeftFlank(s, left, false)
	last := len(right) - 1
	leftRootXY := rootContact(left[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
	rightRootXY := rootContact(right[0], d)
	rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)
	if prevLeftRoot != nil {
		s.CreateArc(center, prevLeftRoot, rightRoot)
	}
	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return toothSketch{}, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return toothSketch{}, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	return toothSketch{
		rightRoot:   rightRoot,
		leftRoot:    leftRoot,
		rightRootXY: rightRootXY,
		leftRootXY:  leftRootXY,
	}, nil
}

// dimensionRightFlank creates every right-flank point with both coordinate
// dimensions.
func dimensionRightFlank(s *sketch.Sketch, pts []point) []*sketch.Point {
	out := make([]*sketch.Point, len(pts))
	for i, pt := range pts {
		out[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return out
}

// dimensionLeftFlank creates the left-flank points with both coordinate
// dimensions, except the tip and, when closedBase is set, the first point,
// which receive one.
func dimensionLeftFlank(s *sketch.Sketch, pts []point, closedBase bool) []*sketch.Point {
	out := make([]*sketch.Point, len(pts))
	for i, pt := range pts {
		if i == len(pts)-1 || (closedBase && i == 0) {
			out[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		out[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return out
}
