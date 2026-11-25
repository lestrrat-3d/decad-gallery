package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// addToothPoints creates the dimensioned flank and root points of one ordinary tooth.
func addToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	tp := toothPoints{
		right: make([]*sketch.Point, len(right)),
		left:  make([]*sketch.Point, len(left)),
	}
	for i, q := range right {
		tp.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		tp.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	tp.leftRootXY = rootContact(left[0], d)
	tp.rightRootXY = rootContact(right[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootXY.x, tp.leftRootXY.y)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootXY.x, tp.rightRootXY.y)
	return tp
}

// drawToothCurves draws the right line, right spline, tip arc, left spline and
// left line of one ordinary tooth.
func drawToothCurves(s *sketch.Sketch, tp toothPoints) error {
	last := len(tp.right) - 1
	s.CreateLine(tp.rightRoot, tp.right[0])
	if _, err := s.CreateFitSpline(tp.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), tp.right[last], tp.left[last])
	if _, err := s.CreateFitSpline(tp.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(tp.left[0], tp.leftRoot)
	return nil
}

// buildToothSection builds one ordinary tooth section at the given angle.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	tp := addToothPoints(s, left, right, d)
	if err := drawToothCurves(s, tp); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), tp.rightRoot, tp.leftRoot)
	profile, err := solveSingleProfile(ctx, s, "tooth section", 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildToothSection builds the six-curve section of one ordinary tooth,
// rotated by angle, on plane. It returns a solved and verified sketch and its
// profile. ctx, world and plane must not be nil. An embedded gear has no
// ordinary tooth section and is rejected.
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
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks have no ordinary tooth section")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
