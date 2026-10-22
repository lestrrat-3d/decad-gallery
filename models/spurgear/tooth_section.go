package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth, rotated by angle, as a solved
// and verified six-curve profile on plane. Embedded flanks are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nonfinite angle %v", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section: embedded flanks are not supported")
	}

	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	t, err := drawTooth(s, left, right, d, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	s.CreateArc(center, t.rightRoot, t.leftRoot)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section: got %d outer curves, want 6", got)
	}
	return s, profile, nil
}

// toothPoints are the sketch points shared with a tooth's neighboring curves.
type toothPoints struct {
	rightRoot, leftRoot *sketch.Point
	rightRootXY         point
	leftRootXY          point
}

// drawTooth creates the dimensioned points, radial lines, splines, and tip
// arc of one ordinary tooth. When prevLeftRoot is non-nil the short root arc
// from it to this tooth's right root is drawn first. The caller closes the
// remaining root arcs.
func drawTooth(s *sketch.Sketch, left, right []point, d dimensions,
	prevLeftRoot *sketch.Point) (toothPoints, error) {
	n := len(right)
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
	// Multiply before dividing; the root coordinates must not use a
	// precomputed d.root/d.base factor.
	var t toothPoints
	t.leftRootXY = point{left[0].x * d.root / d.base, left[0].y * d.root / d.base}
	t.rightRootXY = point{right[0].x * d.root / d.base, right[0].y * d.root / d.base}
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)

	if prevLeftRoot != nil {
		s.CreateArc(s.Origin(), prevLeftRoot, t.rightRoot)
	}
	s.CreateLine(t.rightRoot, rightPts[0])
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return t, fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rightPts[n-1], leftPts[n-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return t, fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(leftPts[0], t.leftRoot)
	return t, nil
}
