package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Drive dimensions in mm. The great wheel, pawl carrier, retainer and knob are built about the origin and then moved onto the great
// wheel arbor.
const (
	greatRear         = 1.5
	greatLength       = 7.0
	greatBore         = 6.9
	ratchetFloor      = 5.0
	ratchetTool       = 4.5
	ratchetDisc       = 11.5
	ratchetInner      = 11.0
	ratchetOuter      = 14.0 // the triangle's outer vertex before its round, with ratchetRound: the wedge root
	ratchetRound      = 0.5
	ratchetTeeth      = 12
	carrierCoreRadius = 7.5
	layerLow          = 5.25
	layerLength       = 4.0
	knobRadius        = 9.0
	knobLow           = 35.30
	knobHigh          = 38.30
	// The great wheel's rear counterbore takes the arbor's collar: radius 4.10 over the collar's 3.7 (a 0.40 Regular
	// fit, the wheel turns on the arbor), floor 0.25 past the collar's end at 3.25.
	greatCounterRadius = 4.10
	greatCounterLow    = 0.5 // the tool starts this high, below the wheel's rear face
	greatCounterFloor  = 3.50
)

// ratchetToolBody builds the internal ratchet's cutter: a disc and twelve triangles.
func (b *builder) ratchetToolBody() (*decad.Body, error) {
	disc, err := b.discAt("ratchet disc", 0, 0, ratchetFloor, ratchetDisc, ratchetTool)
	if err != nil {
		return nil, err
	}
	const what = "ratchet triangle"
	s, err := b.flat(what, 0, 0, ratchetFloor)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	step := 2 * math.Pi / ratchetTeeth
	a := polar(s, ref, ratchetInner, 0)
	c := s.CreatePoint(ratchetOuter*math.Cos(step), ratchetOuter*math.Sin(step))
	d := polar(s, ref, ratchetInner, step)
	ac := s.CreateLine(a, c)
	cd := s.CreateLine(c, d)
	s.CreateLine(d, a)
	// The corner at c is removed by its round, so the two legs carry their own direction: c-d is the radial face on
	// the ray through the origin, and a-c leaves a at the angle it had to the old vertex.
	s.AddConstraint(sketch.NewPointOnLine(s.Origin(), cd),
		sketch.NewAngle(ref, ac, math.Atan2(c.Y()-a.Y(), c.X()-a.X())/deg))
	if _, err := sketch.CreateFillet(s, ac, cd, c, ratchetRound); err != nil {
		return nil, fmt.Errorf("%s: round the tooth root: %w", what, err)
	}
	tooth, err := b.extrude(s, 0, ratchetTool, what)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(b.ctx, tooth, decad.CircularPattern{Center: r3.NewVec(0, 0, 0), Axis: zAxis,
		Count: ratchetTeeth})
	if err != nil {
		return nil, fmt.Errorf("ratchet teeth: pattern: %w", err)
	}
	return b.union("ratchet tool", disc, append([]*decad.Body{tooth}, copies...)...)
}

// buildGreatWheel builds the 48-tooth great wheel with its rear counterbore and internal ratchet, about the origin with
// tooth 0 on +X.
func (b *builder) buildGreatWheel() (*decad.Body, error) {
	wheel, err := b.buildGear("great wheel", GreatTeeth, greatLength, greatRear, 0, greatBore)
	if err != nil {
		return nil, err
	}
	counter, err := b.discAt("great wheel counterbore", 0, 0, greatCounterLow, greatCounterRadius,
		greatCounterFloor-greatCounterLow)
	if err != nil {
		return nil, err
	}
	if wheel, err = b.cut(wheel, counter, "great wheel counterbore"); err != nil {
		return nil, err
	}
	// Only the counterbore's rim on the rear face is chamfered; the tooth flanks and the ratchet stay sharp.
	wheel, err = b.finish(wheel, []float64{greatRear}, func(l *decad.Loop, e []*decad.Edge) bool {
		return !l.IsOuter() && len(e) == 1 && math.Abs(math.Hypot(e[0].Start().Position().Value.X,
			e[0].Start().Position().Value.Y)-greatCounterRadius) < 0.01
	})
	if err != nil {
		return nil, fmt.Errorf("great wheel edges: %w", err)
	}
	tool, err := b.ratchetToolBody()
	if err != nil {
		return nil, err
	}
	return b.cut(wheel, tool, "great wheel ratchet")
}

// Pawl layer dimensions in mm: the slot between a pawl arm and the core is 1.5 wide and ends in a full round, the
// arm's tooth root takes a 0.5 round, and the back of the root block takes a 1.0 round into the core.
const (
	slotEndRadius   = 0.75
	toothRootRound  = 0.5
	blockBackRound  = 1.0
	carrierArmInner = 9.0
	carrierArmOuter = 10.5
)

// pawlLayer draws the carrier's whole pawl layer on s as one closed loop: the core, both slots
// and both pawls are its pieces. For pawl j, with base angle g_j, the core arc about the origin ends at the slot end,
// a full round of radius slotEndRadius tangent to the core and to the arm's inner arc. The arm runs clockwise from the
// slot end to its tooth, returns along its outer arc to the root block's back line, and the back line meets the next
// core arc in a round. The tooth's contact face meets the arm's outer arc in a root round.
func pawlLayer(s *sketch.Sketch, ref *sketch.Line, gs []float64) error {
	o := s.Origin()
	at := func(r, a float64) *sketch.Point { return s.CreatePoint(r*math.Cos(a), r*math.Sin(a)) }
	slotAngle := slotEndRadius / (carrierCoreRadius + slotEndRadius)
	n := len(gs)
	corners := make([]*sketch.Point, n) // the back corner of each pawl's root block
	for j, g := range gs {
		corners[j] = at(carrierCoreRadius, g+10*deg)
	}
	cores := make([]*sketch.Arc, n)
	backs := make([]*sketch.Line, n)
	contacts := make([]*sketch.Line, n)
	outers := make([]*sketch.Arc, n)
	roots := make([]*sketch.Point, n)
	for j, g := range gs {
		c := g - slotAngle
		prev := corners[(j+n-1)%n]
		slotStart, slotTop := at(carrierCoreRadius, c), at(carrierArmInner, c)
		center := at(carrierCoreRadius+slotEndRadius, c)
		p4, p5 := at(carrierArmInner, g-120*deg), at(carrierArmOuter, g-120*deg)
		p6, p7 := at(12.5, g-116*deg), at(12.5, g-112*deg)
		p8, p9 := at(carrierArmOuter, g-112*deg), at(carrierArmOuter, g+10*deg)

		core := s.CreateArc(o, prev, slotStart)
		cores[j] = core
		slot := s.CreateArc(center, slotTop, slotStart) // bulges toward larger angles
		inner := s.CreateArc(o, p4, slotTop)
		s.CreateLine(p4, p5)
		s.CreateLine(p5, p6)
		tip := s.CreateArc(o, p6, p7)
		contact := s.CreateLine(p7, p8)
		outer := s.CreateArc(o, p8, p9)
		back := s.CreateLine(p9, corners[j])
		contacts[j], outers[j], backs[j], roots[j] = contact, outer, back, p8
		s.AddConstraint(sketch.NewRadius(core, carrierCoreRadius), sketch.NewRadius(inner, carrierArmInner), sketch.NewTangentCircles(core, slot, false),
			sketch.NewRadius(tip, 12.5), sketch.NewRadius(outer, carrierArmOuter),
			sketch.NewAngle(ref, contact, (g-112*deg+math.Pi)/deg), sketch.NewAngle(ref, back, (g+10*deg+math.Pi)/deg))
		for _, p := range []struct {
			pt *sketch.Point
			a  float64
		}{{slotStart, c}, {slotTop, c}, {p4, g - 120*deg}, {p6, g - 116*deg}, {p7, g - 112*deg}, {p9, g + 10*deg}} {
			polarOnArc(s, ref, p.pt, p.a)
		}
		// p5 is on no arc: its radius and angle fix it.
		l := s.CreateLine(o, p5)
		l.SetConstruction(true)
		s.AddConstraint(sketch.NewDistance(o, p5, carrierArmOuter), sketch.NewAngle(ref, l, (g-120*deg)/deg))
	}
	// The corners are removed by their rounds, so each leg carries its own place before the rounds: the radial lines
	// pass through the origin and the arcs have their radii.
	for j := range gs {
		if _, err := sketch.CreateFillet(s, contacts[j], outers[j], roots[j], toothRootRound); err != nil {
			return fmt.Errorf("round pawl %d tooth root: %w", j+1, err)
		}
		if _, err := sketch.CreateFillet(s, backs[j], cores[(j+1)%n], corners[j], blockBackRound); err != nil {
			return fmt.Errorf("round pawl %d block back: %w", j+1, err)
		}
	}
	return nil
}

// buildPawlCarrier builds the pawl carrier about the origin: the pawl layer (one closed loop of the core and both
// pawls), a short boss in front of it, and the D press bore. Its bore flat is on +X, and its pawls are drawn
// CarrierOffset behind the positions of the old drum's, so that pose by the arbor's turn leaves each pawl at the same
// angle from its ratchet face.
func (b *builder) buildPawlCarrier() (*decad.Body, error) {
	const what = "carrier layer"
	s, err := b.flat(what, 0, 0, layerLow)
	if err != nil {
		return nil, err
	}
	if err := pawlLayer(s, xRef(s), []float64{90*deg - b.g.CarrierOffset, 270*deg - b.g.CarrierOffset}); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	layer, err := b.extrude(s, 0, layerLength, what)
	if err != nil {
		return nil, err
	}
	boss, err := b.discAt("carrier boss", 0, 0, carrierBossLow, carrierBossRadius, carrierBossLength)
	if err != nil {
		return nil, err
	}
	carrier, err := b.union("pawl carrier", layer, boss)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("carrier bore", 0, 0, layerLow-1, bodyPressRadius, carrierPressFlat, 7.75)
	if err != nil {
		return nil, err
	}
	return b.cut(carrier, bore, "carrier bore")
}

// buildSpringRetainer builds the pressed ring that stops the spring hub on the front, about the origin: a wide disc whose
// rear face lies 0.05 from the end of the arbor's body flat.
func (b *builder) buildSpringRetainer() (*decad.Body, error) {
	retainer, err := b.discAt("spring retainer", 0, 0, retainerLow, retainerRadius, retainerHigh-retainerLow)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("retainer bore", 0, 0, retainerLow-1, bodyPressRadius, bodyPressFlat,
		retainerHigh-retainerLow+2)
	if err != nil {
		return nil, err
	}
	return b.cut(retainer, bore, "spring retainer bore")
}

// buildSpringAnchorPin builds the round rod that carries the spring's eye, standing at its place.
func (b *builder) buildSpringAnchorPin() (*decad.Body, error) {
	q := b.g.SpringAnchor
	return b.discAt("spring anchor pin", q.X, q.Y, springAnchorLow, springAnchorPinR, springAnchorHigh-springAnchorLow)
}

// buildMainspring builds the spiral mainspring with spiralspring.Build, placed on the great wheel arbor with its eye
// straight below it, and records the item's profile sketch.
func (b *builder) buildMainspring() (*decad.Body, error) {
	g := &b.g
	sp := SpringParams()
	frame, err := r3.NewFrame(r3.NewVec(g.G.X, g.G.Y, springZ),
		r3.NewVec(math.Cos(g.SpringTurn), math.Sin(g.SpringTurn), 0),
		r3.NewVec(-math.Sin(g.SpringTurn), math.Cos(g.SpringTurn), 0))
	if err != nil {
		return nil, fmt.Errorf("mainspring: placement: %w", err)
	}
	sp.Placement = &frame
	m, err := spiralspring.Build(b.ctx, b.doc, sp)
	if err != nil {
		return nil, fmt.Errorf("mainspring: %w", err)
	}
	b.sketches = append(b.sketches, m.Profile)
	return m.Body, nil
}

// buildKnob builds the winding knob about the origin.
func (b *builder) buildKnob() (*decad.Body, error) {
	body, err := b.discAt("winding knob", 0, 0, knobLow, knobRadius, knobHigh-knobLow)
	if err != nil {
		return nil, err
	}
	counter, err := b.keyedCounterbore("knob counterbore", 0, 0)
	if err != nil {
		return nil, err
	}
	if body, err = b.cut(body, counter, "knob counterbore"); err != nil {
		return nil, err
	}
	tool, err := b.dTool("knob socket", 0, 0, knobLow-pressToolLow, endPressRadius, endPressFlat,
		knobHigh-knobLow+2*pressToolLow)
	if err != nil {
		return nil, err
	}
	return b.cut(body, tool, "knob socket")
}
