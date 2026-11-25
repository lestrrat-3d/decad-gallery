package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// toothGeometry holds the dimensioned points of one ordinary tooth.
type toothGeometry struct {
	rightRoot, leftRoot *sketch.Point
	right, left         []*sketch.Point
	rootR, rootL        point
}

// newToothGeometry samples one ordinary tooth rotated by angle and creates its
// dimensioned sketch points. The left root is created before the right root.
func newToothGeometry(s *sketch.Sketch, d dimensions, teeth, samples int, angle float64) toothGeometry {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(teeth), d.backlash, samples, angle)
	g := toothGeometry{
		right: make([]*sketch.Point, len(right)),
		left:  make([]*sketch.Point, len(left)),
	}
	for i, q := range right {
		g.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			g.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			g.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	g.rootL = rootContact(left[0], d)
	g.rootR = rootContact(right[0], d)
	g.leftRoot = sketchcheck.DimensionArcEnd(s, g.rootL.x, g.rootL.y)
	g.rightRoot = sketchcheck.DimensionPoint(s, g.rootR.x, g.rootR.y)
	return g
}

// drawFlanks draws the right radial line, right spline, tip arc, left spline
// and left radial line of the tooth, in that order.
func (g toothGeometry) drawFlanks(s *sketch.Sketch) error {
	center := s.Origin()
	s.CreateLine(g.rightRoot, g.right[0])
	if _, err := s.CreateFitSpline(g.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(center, g.right[len(g.right)-1], g.left[len(g.left)-1])
	if _, err := s.CreateFitSpline(g.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(g.left[0], g.leftRoot)
	return nil
}

// BuildToothSection builds the verified six-curve section of one ordinary tooth
// on plane, rotated by angle.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("tooth section: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("tooth section: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("tooth section: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, errors.New("tooth section: angle must be finite")
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("tooth section: embedded flanks are not supported")
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection builds the ordinary tooth section without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section sketch: %w", err)
	}
	g := newToothGeometry(s, d, p.Teeth, p.Samples, angle)
	if err := g.drawFlanks(s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	s.CreateArc(s.Origin(), g.rightRoot, g.leftRoot)
	profile, err := solveSingleProfile(ctx, s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth section: got %d boundary curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
