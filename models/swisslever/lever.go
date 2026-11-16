package swisslever

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Lever heights and dimensions in mm.
const (
	leverLow       = 3.30
	leverBandSplit = 5.80 // band S (with the dart) below, band F above
	leverHigh      = 9.80
	leverThickness = leverHigh - leverLow
	armRound       = 3.0
	vx1Round       = 1.5
	dartRootRound  = 0.6
	notchTop       = 44.0
	notchTopHalf   = 1.0
	notchCut       = 5.0
	leverBoreReach = 0.75
	leverBoreLong  = 8.0
)

// slotLines holds the four sides of one slot hole.
type slotLines struct {
	impulse, locking, back, top *sketch.Line
}

// leverSlot draws one slot quadrilateral (a hole of the lever sketch) from its corners C', F', G', H': C' and F' are
// polar points, the locking side and the back side keep the stone's directions and the top is parallel to the impulse
// side.
func leverSlot(s *sketch.Sketch, ref *sketch.Line, slot [4]Pt, uAngle float64) slotLines {
	c := polar(s, ref, slot[0].Len(), slot[0].Angle())
	f := polar(s, ref, slot[1].Len(), slot[1].Angle())
	g := s.CreatePoint(slot[2].X, slot[2].Y)
	h := s.CreatePoint(slot[3].X, slot[3].Y)
	l := slotLines{
		locking: s.CreateLine(c, h),
		impulse: s.CreateLine(c, f),
		back:    s.CreateLine(f, g),
		top:     s.CreateLine(g, h),
	}
	s.AddConstraint(
		sketch.NewAngle(ref, l.locking, uAngle/deg), sketch.NewDistance(c, h, slot[3].Sub(slot[0]).Len()),
		sketch.NewAngle(ref, l.back, (uAngle+StoneKey)/deg), sketch.NewParallel(l.top, l.impulse),
	)
	return l
}

// leverSketch draws the lever outline with its two slots at z 3.30 (band S: the dart in place of the notch).
func (b *builder) leverSketch() (*sketch.Sketch, error) {
	const what = "lever"
	g := &b.g
	s, err := b.flat(what, g.P.X, g.P.Y, leverLow)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	o := s.Origin()
	e := leverSlot(s, ref, g.EntrySlot, g.UEAngle)
	x := leverSlot(s, ref, g.ExitSlot, g.UXAngle)

	pt := func(p Pt) *sketch.Point { return s.CreatePoint(p.X, p.Y) }
	// wall places a vertex SlotWall outside two slot sides.
	wall := func(p Pt, a, c *sketch.Line) *sketch.Point {
		v := pt(p)
		s.AddConstraint(sketch.NewDistancePointLine(v, a, SlotWall), sketch.NewDistancePointLine(v, c, SlotWall))
		return v
	}
	ve1, ve2, ve3 := wall(g.VE1, e.locking, e.top), wall(g.VE2, e.locking, e.impulse), wall(g.VE3, e.impulse, e.back)
	vx2, vx3, vx4 := wall(g.VX2, x.impulse, x.back), wall(g.VX3, x.back, x.top), wall(g.VX4, x.top, x.locking)
	// JL, JR and VX1 are corners that take a round: their position comes from the legs.
	jl, jr, vx1 := pt(g.JL), pt(g.JR), pt(g.VX1)

	// The fork end. Each corner round is level with its centre at both ends and sits on its arm side; the gap rounds
	// are level with the corner round's inner end, so each is tangent to it and to the dart side without a radius.
	ky := g.KR.Y
	level := func(pts ...*sketch.Point) {
		for _, p := range pts[1:] {
			s.AddConstraint(sketch.NewHorizontalPoints(pts[0], p))
		}
	}
	cornerRound := func(c, start, end *sketch.Point) *sketch.Arc {
		arc := s.CreateArc(c, start, end)
		s.AddConstraint(sketch.NewVerticalDistance(o, c, ky), sketch.NewRadius(arc, g.CornerRadius))
		level(c, start, end)
		return arc
	}
	a1, a2 := pt(Pt{ArmHalfWidth, ky}), pt(Pt{g.KR.X - g.CornerRadius, ky})
	b1, b2 := pt(Pt{-ArmHalfWidth, ky}), pt(Pt{g.KL.X + g.CornerRadius, ky})
	cornerRound(pt(g.KR), a1, a2)
	cornerRound(pt(g.KL), b2, b1)
	d1, d2 := pt(Pt{DartRadius, ky}), pt(Pt{DartRadius, g.DartTip.Y})
	d3, d4 := pt(Pt{-DartRadius, g.DartTip.Y}), pt(Pt{-DartRadius, ky})
	gr, gl := pt(g.GapRight), pt(g.GapLeft)
	s.CreateArc(gr, d1, a2)
	level(gr, a2, d1)
	s.CreateArc(gl, b2, d4)
	level(gl, b2, d4)
	td := pt(g.DartTip)
	tip := s.CreateArc(td, d2, d3)
	s.AddConstraint(sketch.NewHorizontalDistance(o, td, 0), sketch.NewVerticalDistance(o, td, g.DartTip.Y),
		sketch.NewRadius(tip, DartRadius))
	level(td, d2, d3)

	rightSide, leftSide := s.CreateLine(jr, a1), s.CreateLine(b1, jl)
	dartR, dartL := s.CreateLine(d1, d2), s.CreateLine(d3, d4)
	s.AddConstraint(
		sketch.NewVertical(rightSide), sketch.NewDistancePointLine(o, rightSide, ArmHalfWidth),
		sketch.NewVertical(leftSide), sketch.NewDistancePointLine(o, leftSide, ArmHalfWidth),
		sketch.NewVertical(dartR), sketch.NewVertical(dartL),
	)

	// The frame lines. The three that meet a corner round carry their own direction.
	upperL, upperR := s.CreateLine(jl, ve1), s.CreateLine(vx4, jr)
	lower := s.CreateLine(ve3, vx1)
	exitImpulse := s.CreateLine(vx1, vx2)
	s.CreateLine(ve1, ve2)
	s.CreateLine(ve2, ve3)
	s.CreateLine(vx2, vx3)
	s.CreateLine(vx3, vx4)
	s.AddConstraint(
		sketch.NewAngle(ref, upperL, g.VE1.Sub(g.JL).Angle()/deg),
		sketch.NewAngle(ref, upperR, g.JR.Sub(g.VX4).Angle()/deg),
		sketch.NewAngle(ref, lower, g.VX1.Sub(g.VE3).Angle()/deg),
		sketch.NewParallel(exitImpulse, x.impulse),
	)
	for _, r := range []struct {
		first, second sketch.Entity
		at            *sketch.Point
		radius        float64
		name          string
	}{
		{leftSide, upperL, jl, armRound, "JL"},
		{upperR, rightSide, jr, armRound, "JR"},
		{lower, exitImpulse, vx1, vx1Round, "VX1"},
	} {
		if err := sketchFillet(s, r.first, r.second, r.at, r.radius); err != nil {
			return nil, fmt.Errorf("%s: round %s: %w", what, r.name, err)
		}
	}
	return s, nil
}

// notchTool draws the band F notch: the semicircle of radius r_n under the notch centre and the narrowing walls up to
// y 44, from z 5.80, tall enough to clear the lever's top.
func (b *builder) notchTool() (*decad.Body, error) {
	const what = "lever notch"
	g := &b.g
	s, err := b.flat(what, g.P.X, g.P.Y, leverBandSplit)
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	ky := g.NotchCentre.Y
	pt := func(x, y float64) *sketch.Point { return s.CreatePoint(x, y) }
	nc := pt(0, ky)
	left, right := pt(-g.NotchRadius, ky), pt(g.NotchRadius, ky)
	arc := s.CreateArc(nc, left, right)
	s.AddConstraint(sketch.NewHorizontalDistance(o, nc, 0), sketch.NewVerticalDistance(o, nc, ky),
		sketch.NewRadius(arc, g.NotchRadius), sketch.NewHorizontalPoints(nc, left), sketch.NewHorizontalPoints(nc, right))
	ur, ul := pt(notchTopHalf, notchTop), pt(-notchTopHalf, notchTop)
	s.CreateLine(right, ur)
	s.CreateLine(ur, ul)
	s.CreateLine(ul, left)
	s.AddConstraint(sketch.NewHorizontalDistance(o, ur, notchTopHalf), sketch.NewVerticalDistance(o, ur, notchTop),
		sketch.NewHorizontalDistance(o, ul, -notchTopHalf), sketch.NewVerticalDistance(o, ul, notchTop))
	return b.extrude(s, 0, notchCut, what)
}

// buildLever builds the lever in its local frame about the pallet arbor: the plate with its two slots and the dart,
// the notch cut from band F, the dart's root rounded, the D press bore and the chamfers on the two outlines.
func (b *builder) buildLever() (*decad.Body, error) {
	s, err := b.leverSketch()
	if err != nil {
		return nil, err
	}
	lever, err := b.extrude(s, 2, leverThickness, "lever")
	if err != nil {
		return nil, err
	}
	notch, err := b.notchTool()
	if err != nil {
		return nil, err
	}
	if lever, err = b.cut(lever, notch, "lever notch"); err != nil {
		return nil, err
	}
	root := decad.Edges(decad.Concave(), decad.Circular()).Exactly(1)
	if lever, err = decad.Fillet(b.ctx, lever, root, units.Millimeters(dartRootRound)); err != nil {
		return nil, fmt.Errorf("lever: dart root fillet: %w", err)
	}
	tool, err := b.dTool("lever bore", b.g.P.X, b.g.P.Y, leverLow-leverBoreReach, BodyBoreRadius, BodyBoreFlat,
		leverBoreLong)
	if err != nil {
		return nil, err
	}
	if lever, err = b.cut(lever, tool, "lever bore"); err != nil {
		return nil, err
	}
	return b.finishWith(lever, []float64{leverLow, leverHigh}, outerLoops, "lever edges")
}
