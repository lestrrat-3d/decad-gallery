package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/sketch"
)

// BuildToothSection builds one ordinary tooth, rotated by the signed angle
// (radians), closed by the short root arc. Embedded gears are rejected.
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: context, world and plane must not be nil")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return nil, nil, fmt.Errorf("spurgear: tooth angle %g must be finite", angle)
	}
	d := derive(p)
	if d.base < d.root {
		return nil, nil, errors.New("spurgear: tooth section does not support embedded profiles")
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	rightPts, leftPts := flankPoints(s, left, right)
	leftRoot, rightRoot, _, _ := rootPoints(s, d, left[0], right[0])

	s.CreateLine(rightRoot, rightPts[0])
	if err := toothCurves(s, 0, p.Samples, rightPts, leftPts); err != nil {
		return nil, nil, err
	}
	s.CreateLine(leftPts[0], leftRoot)
	s.CreateArc(center, rightRoot, leftRoot)

	if err := solveAndVerify(ctx, s, "tooth section"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "tooth section")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 6 {
		return nil, nil, fmt.Errorf("spurgear: tooth section has %d curves, expected 6", len(profile.Outer))
	}
	return s, profile, nil
}
