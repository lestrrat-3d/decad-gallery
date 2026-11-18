package plaquestand

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// sketchOn starts a sketch on the plane of f; an extrusion runs along f's normal U × V.
func sketchOn(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	return s, nil
}

// solveProfile solves s, requires it fully constrained and returns its one valid profile without holes.
func solveProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != 0 {
			continue
		}
		if chosen != nil {
			return nil, errors.New("several valid profiles without holes")
		}
		chosen = pr
	}
	if chosen == nil {
		return nil, errors.New("no valid profile without holes")
	}
	return chosen, nil
}

// profileSketch sketches the stand's side profile on the plane x = 0, with sketch (u, v) the world (y, z) and the
// normal +X. It draws the 18 segments of dm.Outline: one point per segment start and one per arc centre, nine lines and
// nine counter-clockwise arcs, held by 45 constraints (radius, tangency, orientation, position) and the nine implicit
// arc-radius equalities. Nothing fixes a computed point.
func profileSketch(ctx context.Context, dm Dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if err != nil {
		return nil, nil, fmt.Errorf("create profile frame: %w", err)
	}
	s, err := sketchOn(frame)
	if err != nil {
		return nil, nil, err
	}
	o := dm.Outline
	n := len(o)
	pts := make([]*sketch.Point, n)
	for i, seg := range o {
		pts[i] = s.CreatePoint(seg.Start.Y, seg.Start.Z)
	}
	lines := make([]*sketch.Line, 9)
	arcs := make([]*sketch.Arc, 9)
	for k := 1; k <= 9; k++ {
		lines[k-1] = s.CreateLine(pts[2*k-2], pts[(2*k-1)%n])
		seg := o[2*k-1]
		center := s.CreatePoint(seg.Center.Y, seg.Center.Z)
		start, end := pts[2*k-1], pts[(2*k)%n]
		if !seg.Convex {
			start, end = end, start
		}
		arcs[k-1] = s.CreateArc(center, start, end)
	}
	l := func(k int) *sketch.Line { return lines[k-1] }
	a := func(k int) *sketch.Arc { return arcs[k-1] }

	for k := 1; k <= 9; k++ {
		next := k%9 + 1
		s.AddConstraint(
			sketch.NewRadius(a(k), o[2*k-1].Radius),
			sketch.NewTangent(l(k), a(k)),
			sketch.NewTangent(l(next), a(k)),
		)
	}
	recline := math.Atan2(dm.Axis.Y, dm.Axis.Z) * 180 / math.Pi
	s.AddConstraint(
		sketch.NewHorizontal(l(1)),
		sketch.NewVertical(l(2)),
		sketch.NewHorizontal(l(3)),
		sketch.NewHorizontal(l(8)),
		sketch.NewVertical(l(9)),
		sketch.NewAngle(l(1), l(5), -(90+recline)),
		sketch.NewParallel(l(5), l(4)),
		sketch.NewParallel(l(5), l(6)),
		sketch.NewParallel(l(5), l(7)),
	)
	origin := s.Origin()
	s.AddConstraint(
		sketch.NewPointOnLine(origin, l(1)),
		sketch.NewPointOnLine(origin, l(9)),
		sketch.NewDistancePointLine(origin, l(2), dm.Depth),
		sketch.NewDistancePointLine(origin, l(3), BaseThickness),
		sketch.NewDistancePointLine(origin, l(8), BaseThickness),
		sketch.NewHorizontalDistance(origin, a(5).Center, dm.Seat.Y),
		sketch.NewVerticalDistance(origin, a(5).Center, dm.Seat.Z),
		sketch.NewDistance(l(5).Start, l(5).End, dm.BackWall),
		sketch.NewDistance(l(6).Start, l(6).End, dm.LipWall),
	)
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// plaqueSketch sketches the reference plaque's rectangle W wide and H high on the plane that holds the plaque's
// back-bottom edge: its origin is at x0 = (W_s − W)/2 on the seat's back side, U is +X and V is the slot axis, so the
// extrusion runs toward the front by the plaque's thickness.
func plaqueSketch(ctx context.Context, dm Dimensions, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	x0 := (dm.StandWidth - p.PlaqueWidth) / 2
	t := p.PlaqueThickness
	origin := r3.Vec{X: x0, Y: dm.Seat.Y + t/2*dm.Normal.Y, Z: dm.Seat.Z + t/2*dm.Normal.Z}
	frame, err := r3.NewFrame(origin, r3.Vec{X: 1}, dm.Axis)
	if err != nil {
		return nil, nil, fmt.Errorf("create plaque frame: %w", err)
	}
	s, err := sketchOn(frame)
	if err != nil {
		return nil, nil, err
	}
	rect := s.CreateRectangle(0, 0, p.PlaqueWidth, p.PlaqueHeight)
	s.AddConstraint(
		sketch.NewCoincident(rect.A, s.Origin()),
		sketch.NewHorizontalDistance(rect.A, rect.B, p.PlaqueWidth),
		sketch.NewVerticalDistance(rect.B, rect.C, p.PlaqueHeight),
	)
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}
