package bevelgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/modelsketch"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// BuildBackConeToothProfile places the single virtual tooth on its real back-cone plane.
// ctx and world must not be nil. The common pitch apex is the world origin.
func BuildBackConeToothProfile(ctx context.Context, world *sketch.World,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil {
		return nil, nil, fmt.Errorf("context and world must not be nil")
	}
	d, err := derive(p)
	if err != nil {
		return nil, nil, err
	}
	angle := d.coneAngle
	center := r3.NewVec(d.coneDist/math.Cos(angle)+p.ToothSpacing*math.Sin(angle),
		-p.ToothSpacing*math.Cos(angle), 0)
	frame, err := r3.NewFrame(center, r3.NewVec(math.Sin(angle), -math.Cos(angle), 0),
		r3.NewVec(0, 0, 1))
	if err != nil {
		return nil, nil, fmt.Errorf("create back-cone frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("create back-cone plane: %w", err)
	}
	return BuildVirtualToothProfile(ctx, world, plane, p)
}

// BuildVirtualToothProfile draws one back-cone tooth. ctx, world, and plane must not be nil.
// The fitted flanks retain the real-valued virtual tooth count from the pitch cone.
func BuildVirtualToothProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane must not be nil")
	}
	d, err := derive(p)
	if err != nil {
		return nil, nil, err
	}
	pitch := p.Module * float64(p.Teeth) / (2 * math.Cos(d.coneAngle))
	base := pitch * math.Cos(p.PressureAngle)
	tip := pitch + p.Module
	root := pitch - (1.25+0.1125)*p.Module
	if !finite(pitch, base, tip, root) || root <= 0 {
		return nil, nil, fmt.Errorf("virtual tooth radii are invalid")
	}
	virtualTeeth := float64(p.Teeth) / math.Cos(d.coneAngle)
	left, right, err := spurgear.InvoluteFlanks(pitch, base, tip, virtualTeeth, p.Samples, math.Pi)
	if err != nil {
		return nil, nil, fmt.Errorf("sample virtual involute: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create virtual tooth sketch: %w", err)
	}
	polar := modelsketch.NewPolar(s, s.CreatePoint(0, 0))
	center := polar.Center()
	leftPoints := make([]*sketch.Point, p.Samples)
	rightPoints := make([]*sketch.Point, p.Samples)
	for i := range left {
		// The left tip point is tied to the right tip point by the tip arc about the center.
		if leftPoints[i], err = polar.Point(left[i][0], left[i][1], i != len(left)-1); err != nil {
			return nil, nil, fmt.Errorf("left virtual sample %d: %w", i, err)
		}
		if rightPoints[i], err = polar.Point(right[i][0], right[i][1], true); err != nil {
			return nil, nil, fmt.Errorf("right virtual sample %d: %w", i, err)
		}
	}
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("right virtual flank: %w", err)
	}
	s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("left virtual flank: %w", err)
	}
	embedded := base < root
	if embedded {
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(center, root), root))
	} else {
		// The root arc about the center ties the left root radius to the right root radius.
		rightRoot, err := polar.Point(right[0][0]*root/base, right[0][1]*root/base, true)
		if err != nil {
			return nil, nil, fmt.Errorf("right virtual root: %w", err)
		}
		leftRoot, err := polar.Point(left[0][0]*root/base, left[0][1]*root/base, false)
		if err != nil {
			return nil, nil, fmt.Errorf("left virtual root: %w", err)
		}
		s.CreateLine(rightRoot, rightPoints[0])
		s.CreateLine(leftPoints[0], leftRoot)
		s.CreateArc(center, rightRoot, leftRoot)
	}
	if err := modelsketch.Verify(ctx, "virtual tooth sketch", s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if embedded {
		var regions []string
		for _, profile := range profiles {
			var edges []string
			for _, edge := range profile.Outer {
				edges = append(edges, fmt.Sprintf("%T(partial=%t exact=%t)", edge.Entity, edge.Partial, edge.TExact))
			}
			regions = append(regions, fmt.Sprintf("valid=%t outer=%d holes=%d edges=%v",
				profile.Valid, len(profile.Outer), len(profile.Holes), edges))
			if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4 {
				continue
			}
			trimmed := 0
			for _, edge := range profile.Outer {
				flank, ok := edge.Entity.(*sketch.FitSpline)
				if !ok {
					continue
				}
				if !edge.Partial || !edge.TExact {
					return nil, nil, fmt.Errorf("embedded virtual flank lacks exact root trim")
				}
				x, y := flank.Eval(edge.TStart)
				if math.Abs(math.Hypot(x, y)-root) > 1e-8 {
					return nil, nil, fmt.Errorf("embedded virtual flank trim misses root circle")
				}
				trimmed++
			}
			if trimmed == 2 {
				return s, profile, nil
			}
		}
		return nil, nil, fmt.Errorf("found no valid embedded virtual tooth among %d regions: %v",
			len(profiles), regions)
	}
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("virtual tooth sketch has %d profiles; want one valid region", len(profiles))
	}
	if len(profiles[0].Outer) != 6 || len(profiles[0].Holes) != 0 {
		return nil, nil, fmt.Errorf("virtual tooth outline has %d edges and %d holes; want six edges and no holes",
			len(profiles[0].Outer), len(profiles[0].Holes))
	}
	return s, profiles[0], nil
}
