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

// profileSketch sketches a bracket's side profile on the plane x = x0, with sketch (u, v) the world (y, z) and the
// normal +X. It draws the 26 segments of dm.Outline: one point per segment start and one per arc centre, 13 lines and
// 13 counter-clockwise arcs, held by 65 constraints (radius, tangency, orientation, position) and the 13 implicit
// arc-radius equalities. Nothing fixes a computed point. Line k is segment 2k-2 and arc k is segment 2k-1; the lines
// are L1, L2, L3a, W1, Fl, W2, L3b, L4, L5, L6, L7, L8 and L9.
func profileSketch(ctx context.Context, dm Dimensions, x0 float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := r3.NewFrame(r3.Vec{X: x0}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
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
	lines := make([]*sketch.Line, n/2)
	arcs := make([]*sketch.Arc, n/2)
	for k := 1; k <= n/2; k++ {
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
	const (
		l1, l2, l3a, w1, fl, w2, l3b, l4, l5, l6, l7, l8, l9 = 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13
		slotRound                                            = 9
	)

	for k := 1; k <= n/2; k++ {
		next := k%(n/2) + 1
		s.AddConstraint(
			sketch.NewRadius(a(k), o[2*k-1].Radius),
			sketch.NewTangent(l(k), a(k)),
			sketch.NewTangent(l(next), a(k)),
		)
	}
	recline := math.Atan2(dm.Axis.Y, dm.Axis.Z) * 180 / math.Pi
	s.AddConstraint(
		sketch.NewHorizontal(l(l1)),
		sketch.NewVertical(l(l2)),
		sketch.NewHorizontal(l(l3a)),
		sketch.NewVertical(l(w1)),
		sketch.NewHorizontal(l(fl)),
		sketch.NewVertical(l(w2)),
		sketch.NewHorizontal(l(l3b)),
		sketch.NewHorizontal(l(l8)),
		sketch.NewVertical(l(l9)),
		sketch.NewAngle(l(l1), l(l5), -(90+recline)),
		sketch.NewParallel(l(l5), l(l4)),
		sketch.NewParallel(l(l5), l(l6)),
		sketch.NewParallel(l(l5), l(l7)),
	)
	origin := s.Origin()
	s.AddConstraint(
		sketch.NewPointOnLine(origin, l(l1)),
		sketch.NewPointOnLine(origin, l(l9)),
		sketch.NewDistancePointLine(origin, l(l2), dm.Depth),
		sketch.NewDistancePointLine(origin, l(l3a), BaseThickness),
		sketch.NewDistancePointLine(origin, l(w1), dm.ChannelBack),
		sketch.NewDistancePointLine(origin, l(fl), dm.ChannelFloor),
		sketch.NewDistancePointLine(origin, l(w2), dm.ChannelFront),
		sketch.NewDistancePointLine(origin, l(l3b), BaseThickness),
		sketch.NewDistancePointLine(origin, l(l8), BaseThickness),
		sketch.NewHorizontalDistance(origin, a(slotRound).Center, dm.Seat.Y),
		sketch.NewVerticalDistance(origin, a(slotRound).Center, dm.Seat.Z),
		sketch.NewDistance(l(l5).Start, l(l5).End, dm.BackWall),
		sketch.NewDistance(l(l6).Start, l(l6).End, dm.LipWall),
	)
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// railSketch sketches the rail's rounded-rectangle section on the plane x = 0, with its origin at the section's centre
// (0, y_ch, T_b - h_r/2 + JointFit), U = +Y and V = +Z, so the extrusion runs along +X. It draws four lines and four
// convex arcs of radius RailRound, counter-clockwise from the bottom line, held by 20 constraints and the four implicit radius
// equalities against its 12 points.
func railSketch(ctx context.Context, dm Dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := r3.NewFrame(r3.Vec{Y: dm.RailCenter, Z: BaseThickness - RailHeight/2 + JointFit}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if err != nil {
		return nil, nil, fmt.Errorf("create rail frame: %w", err)
	}
	s, err := sketchOn(frame)
	if err != nil {
		return nil, nil, err
	}
	hw, hh, rq := RailWidth/2, RailHeight/2, RailRound
	// Corner arcs' centres and the eight tangent points, counter-clockwise from the bottom line's start.
	starts := [][2]float64{
		{-hw + rq, -hh}, {hw - rq, -hh}, {hw, -hh + rq}, {hw, hh - rq}, {hw - rq, hh}, {-hw + rq, hh}, {-hw, hh - rq},
		{-hw, -hh + rq},
	}
	centers := [][2]float64{{hw - rq, -hh + rq}, {hw - rq, hh - rq}, {-hw + rq, hh - rq}, {-hw + rq, -hh + rq}}
	pts := make([]*sketch.Point, len(starts))
	for i, p := range starts {
		pts[i] = s.CreatePoint(p[0], p[1])
	}
	// Line i runs from pts[2i] to pts[2i+1] and arc i from pts[2i+1] to pts[(2i+2)%8].
	lines := make([]*sketch.Line, 4)
	arcs := make([]*sketch.Arc, 4)
	for i := range lines {
		lines[i] = s.CreateLine(pts[2*i], pts[2*i+1])
		center := s.CreatePoint(centers[i][0], centers[i][1])
		arcs[i] = s.CreateArc(center, pts[2*i+1], pts[(2*i+2)%8])
	}
	for i := range arcs {
		s.AddConstraint(
			sketch.NewRadius(arcs[i], rq),
			sketch.NewTangent(lines[i], arcs[i]),
			sketch.NewTangent(lines[(i+1)%4], arcs[i]),
		)
	}
	// Bottom, right, top and left lines.
	o := s.Origin()
	s.AddConstraint(
		sketch.NewHorizontal(lines[0]), sketch.NewVertical(lines[1]), sketch.NewHorizontal(lines[2]),
		sketch.NewVertical(lines[3]),
		sketch.NewDistancePointLine(o, lines[0], hh), sketch.NewDistancePointLine(o, lines[1], hw),
		sketch.NewDistancePointLine(o, lines[2], hh), sketch.NewDistancePointLine(o, lines[3], hw),
	)
	profile, err := solveProfile(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// plaqueSketch sketches the reference plaque's rectangle W wide and H high on the plane that holds the plaque's
// back-bottom edge: its origin is at x0 = (S − W)/2 on the seat's back side, U is +X and V is the slot axis, so the
// extrusion runs toward the front by the plaque's thickness.
func plaqueSketch(ctx context.Context, dm Dimensions, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	x0 := (dm.Span - p.PlaqueWidth) / 2
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
