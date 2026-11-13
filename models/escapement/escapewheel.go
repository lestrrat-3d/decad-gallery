package escapement

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

var zAxis = r3.Vec{Z: 1}

// Escape wheel heights in mm.
const (
	wheelLow      = 22.0 // the wheel's rear face
	wheelHigh     = 25.0
	pinionRear    = 1.5
	pinionLength  = 21.0
	spokeHalf     = 2.5
	windowInner   = 13.0
	windowOuter   = 28.0
	pressRadius   = 2.55 // the press-fit D bore: arc radius, 0.05 over the arbor's 2.5
	pressFlat     = 2.05
	reliefRadius  = 2.8
	reliefFlat    = 2.3
	boreToolReach = 0.75
	// The collar counterbores: radius 3.75 over the collar's 3.7 (a 0.05 press). The front one opens on the wheel's front
	// face and stops on the arbor collar; the rear one opens on the pinion's rear face and stops the rear collar ring.
	wheelCounterRadius = 3.75
	wheelCounterFloor  = 23.95
	wheelCounterTop    = 26.0
	pinionCounterLow   = 0.5 // the rear counterbore tool starts this high, below the pinion's rear face
	pinionCounterFloor = 3.30
)

// escapeDisc draws the root disc with its six windows and extrudes it.
func (b *builder) escapeDisc() (*decad.Body, error) {
	const what = "escape disc"
	s, err := b.flat(what, 0, 0, wheelLow)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	circleAt(s, RootRadius)
	o := s.Origin()
	for k := 0; k < 6; k++ {
		s1 := (90 + 60*float64(k)) * deg
		s2 := s1 + 60*deg
		d13, d28 := math.Asin(spokeHalf/windowInner), math.Asin(spokeHalf/windowOuter)
		a := s.CreatePoint(windowInner*math.Cos(s1+d13), windowInner*math.Sin(s1+d13))
		bb := s.CreatePoint(windowInner*math.Cos(s2-d13), windowInner*math.Sin(s2-d13))
		c := s.CreatePoint(windowOuter*math.Cos(s2-d28), windowOuter*math.Sin(s2-d28))
		d := s.CreatePoint(windowOuter*math.Cos(s1+d28), windowOuter*math.Sin(s1+d28))
		inner := s.CreateArc(o, a, bb)
		outer := s.CreateArc(o, d, c)
		side1 := s.CreateLine(a, d)  // along the spoke at s1, inner to outer
		side2 := s.CreateLine(bb, c) // along the spoke at s2
		s.AddConstraint(
			sketch.NewRadius(inner, windowInner), sketch.NewRadius(outer, windowOuter),
			sketch.NewAngle(ref, side1, s1/deg), sketch.NewDistancePointLine(o, side1, spokeHalf),
			sketch.NewAngle(ref, side2, s2/deg), sketch.NewDistancePointLine(o, side2, spokeHalf),
		)
	}
	return b.extrude(s, 6, wheelHigh-wheelLow, what)
}

// escapeTooth draws tooth 0 and extrudes it through the wheel's thickness.
func (b *builder) escapeTooth() (*decad.Body, error) {
	const what = "escape tooth"
	s, err := b.flat(what, 0, 0, wheelLow)
	if err != nil {
		return nil, err
	}
	g := &b.g
	tip := s.CreatePoint(g.ToothTip.X, g.ToothTip.Y)
	l := s.CreatePoint(g.ToothL.X, g.ToothL.Y)
	bb := s.CreatePoint(g.ToothB.X, g.ToothB.Y)
	to := s.CreateLine(tip, s.Origin())
	to.SetConstruction(true)
	tl := s.CreateLine(tip, l)
	s.CreateLine(l, bb)
	tb := s.CreateLine(bb, tip)
	_ = tb
	rf := RootRadius - Overlap
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), tip, 0), sketch.NewVerticalDistance(s.Origin(), tip, TipRadius),
		sketch.NewDistance(s.Origin(), l, rf), sketch.NewAngle(to, tl, -Rake/deg),
		sketch.NewDistance(s.Origin(), bb, rf),
	)
	tbDir := s.CreateLine(tip, bb)
	tbDir.SetConstruction(true)
	s.AddConstraint(sketch.NewAngle(to, tbDir, -BackAngle/deg))
	return b.extrude(s, 0, wheelHigh-wheelLow, what)
}

// buildEscapeWheel builds the escape wheel with its pinion, bored for the arbor and counterbored at both ends for the
// arbor collar and the rear collar ring, in the model frame about the origin.
func (b *builder) buildEscapeWheel() (*decad.Body, error) {
	disc, err := b.escapeDisc()
	if err != nil {
		return nil, err
	}
	tooth, err := b.escapeTooth()
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(b.ctx, tooth, decad.CircularPattern{Center: r3.NewVec(0, 0, 0), Axis: zAxis, Count: Teeth})
	if err != nil {
		return nil, fmt.Errorf("escape wheel: pattern teeth: %w", err)
	}
	wheel, err := b.union("escape wheel teeth", disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, err
	}
	front, err := b.discAt("escape front counterbore", 0, 0, wheelCounterFloor, wheelCounterRadius,
		wheelCounterTop-wheelCounterFloor)
	if err != nil {
		return nil, err
	}
	if wheel, err = b.cut(wheel, front, "escape front counterbore"); err != nil {
		return nil, err
	}
	// The six windows and the front counterbore's rim are chamfered; the teeth keep their edges.
	rim := rimKeep(Pt{}, wheelCounterRadius)
	wheel, err = b.finish(wheel, []float64{wheelLow, wheelHigh},
		func(l *decad.Loop, e []*decad.Edge) bool { return !l.IsOuter() && (len(e) > 1 || rim(l, e)) })
	if err != nil {
		return nil, fmt.Errorf("escape wheel windows: %w", err)
	}
	pinion, err := b.buildGear("pinion", PinionTeeth, pinionLength, pinionRear, b.g.PinionPhase, 0)
	if err != nil {
		return nil, err
	}
	part, err := b.union("escape wheel", wheel, pinion)
	if err != nil {
		return nil, err
	}
	tool, err := b.dTool("escape wheel bore", 0, 0, -boreToolReach, bodyPressRadius, bodyPressFlat, wheelHigh+2*boreToolReach+1)
	if err != nil {
		return nil, err
	}
	rear, err := b.discAt("escape rear counterbore", 0, 0, pinionCounterLow, wheelCounterRadius,
		pinionCounterFloor-pinionCounterLow)
	if err != nil {
		return nil, err
	}
	bore, err := b.union("escape wheel bore tool", tool, rear)
	if err != nil {
		return nil, err
	}
	return b.cut(part, bore, "escape wheel bore")
}
