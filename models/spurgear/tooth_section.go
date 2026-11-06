package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth rotated by angle on plane and
// returns its solved, verified sketch and its single bounded profile.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := guardProfile(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %v is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %v below root radius %v) are not supported",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section without
// revalidating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	t, err := drawTooth(s, d, left, right, 0, nil)
	if err != nil {
		return nil, nil, err
	}
	s.CreateArc(center, t.rightRoot, t.leftRoot)
	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section", 6)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// toothCurves holds the sketch points of one drawn ordinary tooth.
type toothCurves struct {
	rightRoot, leftRoot *sketch.Point
}

// drawTooth creates the dimensioned points and the right line, right spline,
// tip arc, left spline and left line of one ordinary tooth. It does not draw
// the tooth's own root arc. When prevLeft is non-nil, the root arc from it to
// this tooth's right root is drawn first. index names the tooth in errors.
func drawTooth(s *sketch.Sketch, d dimensions, left, right []point, index int,
	prevLeft *sketch.Point) (toothCurves, error) {
	n := len(right)
	center := s.Origin()
	rightPts := make([]*sketch.Point, n)
	leftPts := make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rightPts[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == n-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)

	if prevLeft != nil {
		s.CreateArc(center, prevLeft, rightRoot)
	}
	s.CreateLine(rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return toothCurves{}, fmt.Errorf("tooth %d: right flank spline: %w", index, err)
	}
	s.CreateArc(center, rightPts[n-1], leftPts[n-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return toothCurves{}, fmt.Errorf("tooth %d: left flank spline: %w", index, err)
	}
	s.CreateLine(leftPts[0], leftRoot)
	return toothCurves{rightRoot: rightRoot, leftRoot: leftRoot}, nil
}
