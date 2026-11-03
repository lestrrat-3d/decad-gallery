package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth, rotated by angle, on plane. The
// section is six curves: right radial line, right spline, tip arc, left
// spline, left radial line and the short root arc. The returned sketch is
// solved and fully constrained.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nonfinite angle %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section: embedded flanks (base radius below root radius) are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the ordinary six-curve tooth section without
// validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	rightPts, leftPts := flankSketchPoints(s, left, right)
	leftContact := rootContact(left[0], d)
	rightContact := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, leftContact.x, leftContact.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightContact.x, rightContact.y)

	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	profile, err := solveSingleProfile(ctx, s, "tooth section", 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}
