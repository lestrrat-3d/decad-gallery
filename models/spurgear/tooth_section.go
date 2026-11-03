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

// solveVerify solves s and requires full constraint.
func solveVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", what, err)
	}
	return nil
}

// onlyProfile returns the single valid profile of s.
func onlyProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, want 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", what)
	}
	return profiles[0], nil
}

// makeToothPoints samples one tooth at angle and creates its dimensioned
// points, left root before right root.
func makeToothPoints(s *sketch.Sketch, p Params, d dimensions, angle float64) toothPoints {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
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
	t.leftRootXY = rootContact(left[0], d)
	t.rightRootXY = rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)
	return t
}

// drawToothCurves creates the right line, right spline, tip arc, left spline
// and left line of one tooth.
func drawToothCurves(s *sketch.Sketch, t toothPoints) error {
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
// short root arc. It does not validate p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	t := makeToothPoints(s, p, d, angle)
	if err := drawToothCurves(s, t); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), t.rightRoot, t.leftRoot)
	if err := solveVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if n := len(profile.Outer); n != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d outer curves, want 6", n)
	}
	return s, profile, nil
}

// BuildToothSection builds one ordinary tooth section rotated by angle on
// plane. It rejects embedded flanks.
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
		return nil, nil, fmt.Errorf("tooth section: nonfinite angle %g", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("tooth section: embedded flanks (base radius %g below root radius %g) "+
			"are not supported", d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}
