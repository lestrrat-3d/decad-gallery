package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothPoints holds the dimensioned points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootRight, rootLeft point
}

// addToothPoints samples one tooth at angle and creates its dimensioned flank
// and root points. The left root is created before the right root.
func addToothPoints(s *sketch.Sketch, p Params, d dimensions, angle float64) toothPoints {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	var t toothPoints
	t.right = make([]*sketch.Point, p.Samples)
	t.left = make([]*sketch.Point, p.Samples)
	for i := 0; i < p.Samples; i++ {
		t.right[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == p.Samples-1 {
			t.left[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			t.left[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	t.rootLeft = rootContact(left[0], d)
	t.rootRight = rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.rootLeft.x, t.rootLeft.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rootRight.x, t.rootRight.y)
	return t
}

// addCurves draws the right line and spline, the tip arc, and the left spline
// and line of one tooth.
func (t toothPoints) addCurves(s *sketch.Sketch) error {
	last := len(t.right) - 1
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), t.right[last], t.left[last])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// buildToothSection draws one ordinary tooth rotated by angle, closed by the
// short root arc, and returns its solved, verified sketch and profile.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	t := addToothPoints(s, p, d, angle)
	if err := t.addCurves(s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s, 6)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	return s, profile, nil
}

// BuildToothSection builds one ordinary tooth section rotated by angle on plane.
// It rejects embedded flanks.
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
		return nil, nil, fmt.Errorf("tooth section: angle %g is not finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %g below root radius %g) are not supported",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
