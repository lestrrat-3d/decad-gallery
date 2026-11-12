package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/sketch"
)

// Crutch, pendulum and cap heights in mm.
const (
	crutchLow     = 34.25
	crutchHigh    = 37.25
	crutchPinLow  = 36.75
	crutchPinHigh = 43.25
	pendulumLow   = 37.75
	pendulumHigh  = 43.75
	pocketLow     = 38.75
	pocketRadius  = 7.05
	lipRadius     = 5.0
	capLow        = 44.25
	capHigh       = 45.75
	slotLow       = -61.25
	slotHigh      = -56.75
	slotRadius    = 1.75
	pressToolLow  = 1.0 // a press-bore tool starts this far below the part
)

// buildCrutch builds the crutch: a hub, an arm and a round end, the pin on its end and the D press socket.
func (b *builder) buildCrutch() (*decad.Body, error) {
	const what = "crutch outline"
	g := &b.g
	s, err := b.flat(what, g.P.X, g.P.Y, crutchLow)
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	ar, al := s.CreatePoint(g.CrutchAR.X, g.CrutchAR.Y), s.CreatePoint(g.CrutchAL.X, g.CrutchAL.Y)
	bl, br := s.CreatePoint(g.CrutchBL.X, g.CrutchBL.Y), s.CreatePoint(g.CrutchBR.X, g.CrutchBR.Y)
	e := s.CreatePoint(0, -CrutchLength)
	hub := s.CreateArc(o, ar, al)
	left := s.CreateLine(al, bl)
	end := s.CreateArc(e, bl, br)
	right := s.CreateLine(br, ar)
	s.AddConstraint(
		sketch.NewHorizontalDistance(o, e, 0), sketch.NewVerticalDistance(o, e, -CrutchLength),
		sketch.NewRadius(hub, 6), sketch.NewRadius(end, 4),
		sketch.NewVertical(left), sketch.NewVertical(right),
		sketch.NewHorizontalDistance(o, ar, 2.5), sketch.NewHorizontalDistance(o, al, -2.5),
	)
	body, err := b.extrude(s, 0, crutchHigh-crutchLow, what)
	if err != nil {
		return nil, err
	}
	pin, err := b.discAt("crutch pin", g.P.X, g.P.Y-CrutchLength, crutchPinLow, 1.5, crutchPinHigh-crutchPinLow)
	if err != nil {
		return nil, err
	}
	if body, err = b.union("crutch", body, pin); err != nil {
		return nil, err
	}
	tool, err := b.dTool("crutch socket", g.P.X, g.P.Y, crutchLow-pressToolLow, pressRadius, pressFlat,
		crutchHigh-crutchLow+2*pressToolLow)
	if err != nil {
		return nil, err
	}
	return b.cut(body, tool, "crutch socket")
}

// buildPendulum builds the pendulum: an eye, a rod and a bob, with a lip bore, the crutch slot and the bearing pocket.
func (b *builder) buildPendulum() (*decad.Body, error) {
	const what = "pendulum outline"
	g := &b.g
	lp := g.Params.PendulumLength
	s, err := b.flat(what, g.S.X, g.S.Y, pendulumLow)
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	er, el := s.CreatePoint(g.PendER.X, g.PendER.Y), s.CreatePoint(g.PendEL.X, g.PendEL.Y)
	ql, qr := s.CreatePoint(g.PendQL.X, g.PendQL.Y), s.CreatePoint(g.PendQR.X, g.PendQR.Y)
	bob := s.CreatePoint(0, -lp)
	eye := s.CreateArc(o, er, el)
	left := s.CreateLine(el, ql)
	round := s.CreateArc(bob, ql, qr)
	right := s.CreateLine(qr, er)
	s.AddConstraint(
		sketch.NewHorizontalDistance(o, bob, 0), sketch.NewVerticalDistance(o, bob, -lp),
		sketch.NewRadius(eye, 10), sketch.NewRadius(round, 25),
		sketch.NewVertical(left), sketch.NewVertical(right),
		sketch.NewHorizontalDistance(o, er, 4), sketch.NewHorizontalDistance(o, el, -4),
	)
	circleAt(s, lipRadius)
	slot, err := s.CreateSlot(0, slotLow, 0, slotHigh, slotRadius)
	if err != nil {
		return nil, fmt.Errorf("%s: slot: %w", what, err)
	}
	s.AddConstraint(
		sketch.NewHorizontalDistance(o, slot.C1, 0), sketch.NewVerticalDistance(o, slot.C1, slotLow),
		sketch.NewHorizontalDistance(o, slot.C2, 0), sketch.NewVerticalDistance(o, slot.C2, slotHigh),
		sketch.NewRadius(slot.A1, slotRadius),
	)
	body, err := b.extrude(s, 2, pendulumHigh-pendulumLow, what)
	if err != nil {
		return nil, err
	}
	pocket, err := b.discAt("pendulum pocket", g.S.X, g.S.Y, pocketLow, pocketRadius, 6)
	if err != nil {
		return nil, err
	}
	if body, err = b.cut(body, pocket, "pendulum pocket"); err != nil {
		return nil, err
	}
	body, _, err = edgefinish.Chamfer(b.ctx, body, zAxis, []float64{pendulumLow, pendulumHigh}, rimKeep(g.S, pocketRadius,
		lipRadius), edgefinish.MinChamfer)
	if err != nil {
		return nil, fmt.Errorf("pendulum edges: %w", err)
	}
	return body, nil
}

// rimKeep keeps the outer loops and the single-edge circular loops whose start lies within 0.01 mm of one of the radii
// about c.
func rimKeep(c Pt, radii ...float64) edgefinish.LoopFilter {
	return rimKeepAll([]Pt{c}, radii...)
}

// rimKeepAll is rimKeep about several centres.
func rimKeepAll(centres []Pt, radii ...float64) edgefinish.LoopFilter {
	return func(l *decad.Loop, e []*decad.Edge) bool {
		if l.IsOuter() {
			return true
		}
		if len(e) != 1 {
			return false
		}
		at := e[0].Start().Position().Value
		for _, c := range centres {
			for _, r := range radii {
				if math.Abs(math.Hypot(at.X-c.X, at.Y-c.Y)-r) < 0.01 {
					return true
				}
			}
		}
		return false
	}
}

// buildCap builds the suspension cap: a disc on the pin's front peg with a D press socket.
func (b *builder) buildCap() (*decad.Body, error) {
	g := &b.g
	body, err := b.discAt("suspension cap", g.S.X, g.S.Y, capLow, CollarRadius, capHigh-capLow)
	if err != nil {
		return nil, err
	}
	tool, err := b.dTool("cap socket", g.S.X, g.S.Y, capLow-pressToolLow, pressRadius, pressFlat,
		capHigh-capLow+2*pressToolLow)
	if err != nil {
		return nil, err
	}
	return b.cut(body, tool, "cap socket")
}
