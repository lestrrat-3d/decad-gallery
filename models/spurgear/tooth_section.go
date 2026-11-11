package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth on plane, turned by angle about
// the sketch origin. The returned profile has six curves: the right radial
// line, the right flank spline, the tip arc, the left flank spline, the left
// radial line and the short root arc. The sketch is solved and fully
// constrained. Embedded teeth (base radius below root radius) are rejected.
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
		return nil, nil, fmt.Errorf("tooth section: angle %v is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded teeth (base radius %v below root radius %v) are not supported",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section at angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, right, left, d)

	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank: %w", err)
	}
	s.CreateArc(center, tp.right[len(tp.right)-1], tp.left[len(tp.left)-1])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	s.CreateArc(center, tp.rightRoot, tp.leftRoot)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, want 6", got)
	}
	return s, profile, nil
}

// toothPoints holds the sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// addToothPoints creates the dimensioned flank and root points of one tooth.
// Every right point takes both dimensions. Every left point does too, except
// the tip, and the left root takes one. The root arc fixes the remaining radius.
func addToothPoints(s *sketch.Sketch, right, left []point, d dimensions) toothPoints {
	var tp toothPoints
	for _, q := range right {
		tp.right = append(tp.right, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	for i, q := range left {
		if i == len(left)-1 {
			tp.left = append(tp.left, sketchcheck.DimensionArcEnd(s, q.x, q.y))
		} else {
			tp.left = append(tp.left, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
	}
	tp.leftRootXY = rootContact(left[0], d)
	tp.rightRootXY = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootXY.x, tp.leftRootXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootXY.x, tp.rightRootXY.y)
	return tp
}
