package escapement

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/sketch"
)

// Anchor heights in mm.
const (
	anchorLow    = 21.0
	anchorHigh   = 26.0
	anchorBoss   = 25.5
	anchorBossH  = 2.25
	anchorBoreAt = 20.0
	anchorBoreH  = 8.75
)

// anchorOutline draws the anchor's closed loop of 14 pieces about the pallet arbor and extrudes it.
func (b *builder) anchorOutline() (*decad.Body, error) {
	const what = "anchor outline"
	g := &b.g
	s, err := b.flat(what, g.P.X, g.P.Y, anchorLow)
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	ref := xRef(s)
	at := func(p Pt) *sketch.Point { return s.CreatePoint(p.X, p.Y) }

	ce, fe, ge, he, ke, me, ne := at(g.CE), at(g.FE), at(g.GE), at(g.HE), at(g.KE), at(g.ME), at(g.NE)
	cx, fx, gx, hx, kx, mx, nx := at(g.CX), at(g.FX), at(g.GX), at(g.HX), at(g.KX), at(g.MX), at(g.NX)

	// Pieces, in the order of the loop.
	s.CreateLine(ce, fe)
	arcInE := s.CreateArc(o, ge, fe) // r_in, entry
	innerE := s.CreateLine(he, ge)
	arcHubLow := s.CreateArc(o, he, hx)
	innerX := s.CreateLine(hx, gx)
	arcInX := s.CreateArc(o, cx, gx)
	s.CreateLine(cx, fx)
	arcOutX := s.CreateArc(o, fx, kx)
	endX := s.CreateLine(kx, mx)
	outerX := s.CreateLine(mx, nx)
	arcHubTop := s.CreateArc(o, nx, ne)
	outerE := s.CreateLine(ne, me)
	endE := s.CreateLine(me, ke)
	arcOutE := s.CreateArc(o, ke, ce)

	s.AddConstraint(
		sketch.NewRadius(arcInE, g.RIn), sketch.NewRadius(arcInX, g.RIn),
		sketch.NewRadius(arcOutE, g.ROut), sketch.NewRadius(arcOutX, g.ROut),
		sketch.NewRadius(arcHubLow, HubRadius), sketch.NewRadius(arcHubTop, HubRadius),
	)
	// Pallet corners are polar points: on their arc, at their angle from the origin.
	for _, c := range []struct {
		pt *sketch.Point
		at Pt
	}{{ce, g.CE}, {fe, g.FE}, {cx, g.CX}, {fx, g.FX}} {
		polarOnArc(s, ref, c.pt, c.at.Angle())
	}
	// Each arm: the inner side runs from the hub to the pallet's top corner along a ray through the arbor; the arm end is
	// perpendicular to it and the outer side parallel, an arm's width away.
	for _, a := range []struct {
		inner, end, outer *sketch.Line
		k, m              *sketch.Point
		top               float64
	}{{innerE, endE, outerE, ke, me, g.TopE}, {innerX, endX, outerX, kx, mx, g.TopX}} {
		s.AddConstraint(
			sketch.NewAngle(ref, a.inner, a.top/deg), sketch.NewPointOnLine(o, a.inner),
			sketch.NewPointOnLine(a.k, a.inner),
			sketch.NewPerpendicular(a.end, a.inner), sketch.NewDistancePointLine(a.m, a.inner, ArmWidth),
			sketch.NewParallel(a.outer, a.inner),
		)
	}
	body, err := b.extrude(s, 0, anchorHigh-anchorLow, what)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// polarOnArc fixes pt, an endpoint of a radius-constrained arc about the origin, by the angle of the line from the
// origin to it.
func polarOnArc(s *sketch.Sketch, ref *sketch.Line, pt *sketch.Point, angle float64) {
	l := s.CreateLine(s.Origin(), pt)
	l.SetConstruction(true)
	s.AddConstraint(sketch.NewAngle(ref, l, angle/deg))
}

// buildAnchor builds the anchor: the outline, its front boss, the press-fit D bore and the outer-loop chamfers.
func (b *builder) buildAnchor() (*decad.Body, error) {
	g := &b.g
	body, err := b.anchorOutline()
	if err != nil {
		return nil, err
	}
	boss, err := b.discAt("anchor boss", g.P.X, g.P.Y, anchorBoss, CollarRadius, anchorBossH)
	if err != nil {
		return nil, err
	}
	if body, err = b.union("anchor", body, boss); err != nil {
		return nil, err
	}
	tool, err := b.dTool("anchor bore", g.P.X, g.P.Y, anchorBoreAt, pressRadius, pressFlat, anchorBoreH)
	if err != nil {
		return nil, err
	}
	if body, err = b.cut(body, tool, "anchor bore"); err != nil {
		return nil, err
	}
	body, err = b.finish(body, []float64{anchorLow, anchorHigh}, edgefinish.Outer)
	if err != nil {
		return nil, fmt.Errorf("anchor edges: %w", err)
	}
	return body, nil
}
