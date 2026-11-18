package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth section at the given signed angle on plane.
// The section has six curves and is solved and verified. Embedded gears are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	angle float64) (*sketch.Sketch, *sketch.Profile, error) {
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
		return nil, nil, fmt.Errorf("spurgear: tooth section angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section of an embedded gear is not supported: flanks are embedded in the root circle")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth: right radial line, right spline,
// tip arc, left spline, left radial line and the short root arc.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	rightPts, leftPts := flankSketchPoints(s, left, right, false)
	last := p.Samples - 1

	leftRootXY := rootContact(left[0], d)
	rightRootXY := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)

	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section has %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
