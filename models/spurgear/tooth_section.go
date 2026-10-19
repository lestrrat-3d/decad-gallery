package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/modelsketch"
	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection draws the six-curve ordinary spur tooth at angle on plane.
// The returned Sketch profile is also the section used by a helical tooth loft.
// ctx, world, and plane must not be nil. Angles are radians.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane must not be nil")
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("tooth angle must be finite")
	}
	if err := p.validate(); err != nil {
		return nil, nil, err
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, fmt.Errorf("embedded tooth sections are not supported")
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create tooth sketch: %w", err)
	}
	polar := modelsketch.NewPolar(s, s.CreatePoint(0, 0))
	center := polar.Center()
	left, right := toothFlanks(p, d, angle)
	leftPoints, rightPoints, err := flankSketchPoints(polar, left, right)
	if err != nil {
		return nil, nil, fmt.Errorf("place tooth flanks: %w", err)
	}
	rightRoot, leftRoot, err := rootSketchPoints(polar, left[0], right[0], d)
	if err != nil {
		return nil, nil, fmt.Errorf("place tooth roots: %w", err)
	}
	s.CreateLine(rightRoot, rightPoints[0])
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("draw right flank: %w", err)
	}
	s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("draw left flank: %w", err)
	}
	s.CreateLine(leftPoints[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)
	if err := modelsketch.Verify(ctx, "tooth sketch", s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("tooth sketch has %d profiles; want one valid region", len(profiles))
	}
	if len(profiles[0].Outer) != 6 {
		return nil, nil, fmt.Errorf("tooth profile has %d boundary curves; want six", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}
