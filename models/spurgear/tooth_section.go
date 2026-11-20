package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth section at the signed angle on
// plane: right radial line, right flank, tip arc, left flank, left radial line,
// closed by the short root arc. It returns the solved, verified sketch and its
// six-curve profile. A gear whose flanks begin inside the root circle has no
// standalone tooth section; such requests return an error.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: nil sketch world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: nil sketch plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section angle %g is not finite", angle)
	}
	d, err := deriveChecked(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: derive dimensions: %w", err)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section: embedded flanks (base radius %g below root radius %g) have no standalone tooth section",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section at angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	rightPts, leftPts := flankPoints(s, left, right, false)
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	if err := ordinaryToothCurves(s, 0, rightRoot, rightPts, leftPts, leftRoot); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	s.CreateArc(center, rightRoot, leftRoot)
	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section", 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// flankPoints creates the dimensioned flank points. Every right point gets both
// coordinate dimensions; every left point gets both except its tip, which gets
// one, and, when leftFirstOnArc is set, its first point, which also gets one.
func flankPoints(s *sketch.Sketch, left, right []point, leftFirstOnArc bool) ([]*sketch.Point, []*sketch.Point) {
	n := len(right)
	rightPts := make([]*sketch.Point, n)
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts := make([]*sketch.Point, n)
	for i, q := range left {
		if i == n-1 || (i == 0 && leftFirstOnArc) {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	return rightPts, leftPts
}

// ordinaryToothCurves draws the right radial line, right flank spline, tip arc,
// left flank spline and left radial line of tooth k, reusing the given points.
func ordinaryToothCurves(s *sketch.Sketch, k int, rightRoot *sketch.Point, rightPts, leftPts []*sketch.Point,
	leftRoot *sketch.Point) error {
	n := len(rightPts)
	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return fmt.Errorf("tooth %d right flank: %w", k, err)
	}
	s.CreateArc(s.Origin(), rightPts[n-1], leftPts[n-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return fmt.Errorf("tooth %d left flank: %w", k, err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	return nil
}
