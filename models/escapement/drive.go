package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Drive dimensions in mm. The great wheel, pawl carrier, spacer and knob are built about the origin and then moved onto the great
// wheel arbor.
const (
	greatRear         = 1.5
	greatLength       = 7.0
	greatBore         = 6.9
	ratchetFloor      = 5.0
	ratchetTool       = 4.5
	ratchetDisc       = 11.5
	ratchetInner      = 11.0
	ratchetOuter      = 13.5
	ratchetTeeth      = 12
	carrierCoreRadius = 7.5
	layerLow          = 5.25
	layerLength       = 4.0
	knobRadius        = 9.0
	knobLow           = 34.25
	knobHigh          = 37.25
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
	c := polar(s, ref, ratchetOuter, step)
	d := polar(s, ref, ratchetInner, step)
	s.CreateLine(a, c)
	s.CreateLine(c, d)
	s.CreateLine(d, a)
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

// pawlLoop draws pawl j's closed loop (g is its base angle) on s and returns nothing: its pieces are the profile.
func pawlLoop(s *sketch.Sketch, ref *sketch.Line, g float64) {
	o := s.Origin()
	at := func(r, a float64) *sketch.Point { return s.CreatePoint(r*math.Cos(a), r*math.Sin(a)) }
	p1, p2, p3 := at(7.0, g+10*deg), at(7.0, g), at(9.0, g)
	p4, p5 := at(9.0, g-120*deg), at(10.5, g-120*deg)
	p6, p7 := at(12.5, g-116*deg), at(12.5, g-112*deg)
	p8, p9 := at(10.5, g-112*deg), at(10.5, g+10*deg)
	a7 := s.CreateArc(o, p2, p1)
	s.CreateLine(p2, p3)
	a9 := s.CreateArc(o, p4, p3)
	s.CreateLine(p4, p5)
	s.CreateLine(p5, p6)
	a125 := s.CreateArc(o, p6, p7)
	s.CreateLine(p7, p8)
	a105 := s.CreateArc(o, p8, p9)
	s.CreateLine(p9, p1)
	s.AddConstraint(sketch.NewRadius(a7, 7.0), sketch.NewRadius(a9, 9.0), sketch.NewRadius(a125, 12.5),
		sketch.NewRadius(a105, 10.5))
	for _, c := range []struct {
		p *sketch.Point
		a float64
	}{{p1, g + 10*deg}, {p2, g}, {p3, g}, {p4, g - 120*deg}, {p6, g - 116*deg}, {p7, g - 112*deg},
		{p8, g - 112*deg}, {p9, g + 10*deg}} {
		polarOnArc(s, ref, c.p, c.a)
	}
	// p5 is on no arc: its radius and angle fix it.
	l := s.CreateLine(o, p5)
	l.SetConstruction(true)
	s.AddConstraint(sketch.NewDistance(o, p5, 10.5), sketch.NewAngle(ref, l, (g-120*deg)/deg))
}

// buildPawlCarrier builds the pawl carrier about the origin: the pawl layer (a core and two pawls), a short boss in
// front of it, and the D press bore. Its bore flat is on +X, and its pawls are drawn CarrierOffset behind the
// positions of the old drum's, so that pose by the arbor's turn leaves each pawl at the same angle from its ratchet face.
func (b *builder) buildPawlCarrier() (*decad.Body, error) {
	cs, err := b.flat("carrier core", 0, 0, layerLow)
	if err != nil {
		return nil, err
	}
	circleAt(cs, carrierCoreRadius)
	core, err := b.extrude(cs, 0, layerLength, "carrier core")
	if err != nil {
		return nil, err
	}
	var rest []*decad.Body
	for j, g := range []float64{90*deg - b.g.CarrierOffset, 270*deg - b.g.CarrierOffset} {
		what := fmt.Sprintf("carrier pawl %d", j+1)
		s, err := b.flat(what, 0, 0, layerLow)
		if err != nil {
			return nil, err
		}
		pawlLoop(s, xRef(s), g)
		pawl, err := b.extrude(s, 0, layerLength, what)
		if err != nil {
			return nil, err
		}
		rest = append(rest, pawl)
	}
	boss, err := b.discAt("carrier boss", 0, 0, carrierBossLow, carrierBossRadius, carrierBossLength)
	if err != nil {
		return nil, err
	}
	carrier, err := b.union("pawl carrier", core, append(rest, boss)...)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("carrier bore", 0, 0, layerLow-1, drivePressRadius, bodyPressFlat, 7.75)
	if err != nil {
		return nil, err
	}
	return b.cut(carrier, bore, "carrier bore")
}

// buildSpringSpacer builds the pressed tube from the spring hub's front face to the front bearing, about the origin.
func (b *builder) buildSpringSpacer() (*decad.Body, error) {
	spacer, err := b.discAt("spring spacer", 0, 0, spacerLow, CollarRadius, spacerHigh-spacerLow)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("spacer bore", 0, 0, spacerLow-1, bodyPressRadius, bodyPressFlat, spacerHigh-spacerLow+2)
	if err != nil {
		return nil, err
	}
	return b.cut(spacer, bore, "spring spacer bore")
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
	tool, err := b.dTool("knob socket", 0, 0, knobLow-pressToolLow, pressRadius, pressFlat,
		knobHigh-knobLow+2*pressToolLow)
	if err != nil {
		return nil, err
	}
	return b.cut(body, tool, "knob socket")
}
