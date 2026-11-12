package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Drive dimensions in mm. The great wheel, drum and knob are built about the origin and then moved onto the great
// wheel arbor.
const (
	greatRear      = 1.5
	greatLength    = 7.0
	greatBore      = 5.8
	ratchetFloor   = 5.0
	ratchetTool    = 4.5
	ratchetDisc    = 11.5
	ratchetInner   = 11.0
	ratchetOuter   = 13.5
	ratchetTeeth   = 12
	drumCoreRadius = 7.5
	layerLow       = 5.25
	layerLength    = 4.0
	knobRadius     = 9.0
	knobLow        = 34.25
	knobHigh       = 37.25
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

// buildGreatWheel builds the 48-tooth great wheel with its internal ratchet, about the origin with tooth 0 on +X.
func (b *builder) buildGreatWheel() (*decad.Body, error) {
	wheel, err := b.buildGear("great wheel", GreatTeeth, greatLength, greatRear, 0, greatBore)
	if err != nil {
		return nil, err
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

// buildDrum builds the drum about the origin: the barrel, cone, flange and boss, the press bore, and the pawl layer.
func (b *builder) buildDrum() (*decad.Body, error) {
	drum, err := b.stepped("drum", 0, 0, meridianPoints{{0, 8.75}, {7, 8.75}, {7, 18.75}, {9.5, 21.25}, {3.7, 21.25},
		{3.7, 27.75}, {0, 27.75}})
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("drum bore", 0, 0, 7.75, pressRadius, pressFlat, 21)
	if err != nil {
		return nil, err
	}
	if drum, err = b.cut(drum, bore, "drum bore"); err != nil {
		return nil, err
	}
	cs, err := b.flat("drum core", 0, 0, layerLow)
	if err != nil {
		return nil, err
	}
	circleAt(cs, drumCoreRadius)
	dLoop(cs, reliefRadius, reliefFlat)
	core, err := b.extrude(cs, 1, layerLength, "drum core")
	if err != nil {
		return nil, err
	}
	var pawls []*decad.Body
	for j, g := range []float64{90 * deg, 270 * deg} {
		what := fmt.Sprintf("drum pawl %d", j+1)
		s, err := b.flat(what, 0, 0, layerLow)
		if err != nil {
			return nil, err
		}
		pawlLoop(s, xRef(s), g)
		pawl, err := b.extrude(s, 0, layerLength, what)
		if err != nil {
			return nil, err
		}
		pawls = append(pawls, pawl)
	}
	layer, err := b.union("drum pawl layer", core, pawls...)
	if err != nil {
		return nil, err
	}
	return b.union("drum", drum, layer)
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
