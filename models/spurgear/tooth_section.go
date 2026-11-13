package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the dimensioned points of one ordinary tooth section.
type toothPoints struct {
	right, left         []*sketch.Point
	rootRight, rootLeft *sketch.Point
	rootRightXY         point
	rootLeftXY          point
}

// addToothPoints creates the dimensioned flank and root points of one ordinary
// tooth. Every right point receives both coordinates. Each left point receives
// both, except the tip, which receives one. The right root receives both
// coordinates and the left root one; the joining arc supplies the radius.
func addToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var t toothPoints
	for _, q := range right {
		t.right = append(t.right, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	last := len(left) - 1
	for i, q := range left {
		if i == last {
			t.left = append(t.left, sketchcheck.DimensionArcEnd(s, q.x, q.y))
		} else {
			t.left = append(t.left, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
	}
	t.rootLeftXY = rootContact(left[0], d)
	t.rootLeft = sketchcheck.DimensionArcEnd(s, t.rootLeftXY.x, t.rootLeftXY.y)
	t.rootRightXY = rootContact(right[0], d)
	t.rootRight = sketchcheck.DimensionPoint(s, t.rootRightXY.x, t.rootRightXY.y)
	return t
}

// addToothCurves draws the right line, right spline, tip arc, left spline and
// left line of one ordinary tooth.
func addToothCurves(s *sketch.Sketch, t toothPoints, tooth int) error {
	s.CreateLine(t.rootRight, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("tooth %d: right flank spline: %w", tooth, err)
	}
	s.CreateArc(s.Origin(), t.right[len(t.right)-1], t.left[len(t.left)-1])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("tooth %d: left flank spline: %w", tooth, err)
	}
	s.CreateLine(t.left[0], t.rootLeft)
	return nil
}

// buildToothSection draws one ordinary tooth at angle and closes it with the
// short root arc. It does not validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	t := addToothPoints(s, left, right, d)
	if err := addToothCurves(s, t, 0); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rootRight, t.rootLeft)
	if err := solveVerified(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := soleProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildToothSection draws one ordinary tooth on plane, turned by angle radians
// about the sketch origin, and returns its solved, verified sketch and profile.
// It applies no solid feature. Embedded teeth are rejected.
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
		return nil, nil, fmt.Errorf("tooth section: angle %g is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %g below root radius %g) are not supported",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
