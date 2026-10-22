package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// six-curve section is closed by the short root arc. Embedded gears have no
// standalone tooth.
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
	d, err := validate(p)
	if err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle %v must be finite", angle)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: embedded profile has no standalone tooth section")
	}
	return buildTooth(ctx, world, plane, p, d, angle)
}

// buildTooth draws the tooth section without validating p.
func buildTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create tooth sketch: %w", err)
	}
	center := s.Origin()

	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	leftPts, rightPts := flankPoints(s, left, right)
	leftRoot := scaleToRoot(d, left[0][0], left[0][1])
	rightRoot := scaleToRoot(d, right[0][0], right[0][1])
	leftRootPt := sketchcheck.DimensionArcEnd(s, leftRoot.x, leftRoot.y)
	rightRootPt := sketchcheck.DimensionPoint(s, rightRoot.x, rightRoot.y)

	last := p.Samples - 1
	s.CreateLine(rightRootPt, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], leftRootPt)
	s.CreateArc(center, rightRootPt, leftRootPt)

	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if n := len(profile.Outer); n != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d outer curves, expected 6", n)
	}
	return s, profile, nil
}

// flankPoints creates the dimensioned flank sample points. Every right point
// and every left point except its tip receives both coordinate dimensions.
func flankPoints(s *sketch.Sketch, left, right [][2]float64) (leftPts, rightPts []*sketch.Point) {
	rightPts = make([]*sketch.Point, len(right))
	for i, c := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, c[0], c[1])
	}
	leftPts = make([]*sketch.Point, len(left))
	for i, c := range left {
		if i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, c[0], c[1])
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, c[0], c[1])
		}
	}
	return leftPts, rightPts
}
