package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothSketch holds the sketch points and flank curves of one ordinary tooth.
type toothSketch struct {
	rightRoot, leftRoot *sketch.Point
	rightPts, leftPts   []*sketch.Point
	rightSpline         *sketch.FitSpline
	leftSpline          *sketch.FitSpline
}

// drawOrdinaryTooth draws the right radial line, right spline, tip arc, left
// spline and left radial line of one ordinary tooth. When prevLeftRoot is not
// nil it first draws the root arc from that point to this tooth's right root.
// It does not draw the closing root arc of the tooth.
func drawOrdinaryTooth(s *sketch.Sketch, left, right []point, d dimensions,
	prevLeftRoot *sketch.Point) (*toothSketch, error) {
	n := len(right)
	t := &toothSketch{
		rightPts: make([]*sketch.Point, n),
		leftPts:  make([]*sketch.Point, n),
	}
	for i, q := range right {
		t.rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == n-1 {
			t.leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			t.leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	lr := rootContact(left[0], d)
	rr := rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, rr.x, rr.y)

	center := s.Origin()
	if prevLeftRoot != nil {
		s.CreateArc(center, prevLeftRoot, t.rightRoot)
	}
	s.CreateLine(t.rightRoot, t.rightPts[0])
	var err error
	if t.rightSpline, err = s.CreateFitSpline(t.rightPts...); err != nil {
		return nil, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, t.rightPts[n-1], t.leftPts[n-1])
	if t.leftSpline, err = s.CreateFitSpline(t.leftPts...); err != nil {
		return nil, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(t.leftPts[0], t.leftRoot)
	return t, nil
}

// buildToothSection builds the six-curve section of one ordinary tooth at the
// given angle on plane.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	t, err := drawOrdinaryTooth(s, left, right, d, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildToothSection builds one ordinary tooth, turned by angle about the gear
// axis, as a solved and verified sketch on plane. The profile has six curves.
// It rejects embedded teeth, whose flanks cross the root circle.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("tooth section: nil sketch world")
	}
	if plane == nil {
		return nil, nil, errors.New("tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth section: angle %v is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks (base radius below root radius) are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
