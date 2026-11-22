package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws one ordinary tooth section on plane, turned by the
// signed angle about the sketch origin. The section has six curves: the right
// radial line, right flank, tip arc, left flank, left radial line and the
// short root arc that closes it. It returns the solved, verified sketch and
// its profile. Gears whose base circle lies inside the root circle (embedded
// flanks) are rejected.
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
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth section: nonfinite angle %g", angle)
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	d, err := computeDimensions(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	if d.base < d.root {
		return nil, nil, fmt.Errorf("spurgear: tooth section: embedded flanks (base radius %g below root radius %g) have no standalone section",
			d.base, d.root)
	}
	return buildToothSection(ctx, world, plane, p, d, angle)
}

// buildToothSection draws the six-curve ordinary tooth section at angle
// without validating p.
func buildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
	pts := dimensionOrdinaryTooth(s, left, right, d)
	last := len(pts.right) - 1

	s.CreateLine(pts.rightRoot, pts.right[0])
	if _, err := s.CreateFitSpline(pts.right...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: right flank: %w", err)
	}
	s.CreateArc(center, pts.right[last], pts.left[last])
	if _, err := s.CreateFitSpline(pts.left...); err != nil {
		return nil, nil, fmt.Errorf("tooth section: left flank: %w", err)
	}
	s.CreateLine(pts.left[0], pts.leftRoot)
	s.CreateArc(center, pts.rightRoot, pts.leftRoot)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("tooth section: %w", err)
	}
	if got := len(profile.Outer); got != 6 {
		return nil, nil, fmt.Errorf("tooth section: %d boundary curves, want 6", got)
	}
	return s, profile, nil
}
