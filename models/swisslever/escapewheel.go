package swisslever

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Escape wheel heights and dimensions in mm.
const (
	wheelLow        = 10.35 // the wheel's rear face
	wheelThickness  = 3.0
	wheelHigh       = wheelLow + wheelThickness
	spokeHalf       = 2.5
	windowInner     = 10.0
	windowOuter     = 27.5
	windowRound     = 2.5
	wheelWindows    = 5
	toothRootRound  = 1.4
	wheelBoreReach  = 0.75
	wheelBoreLength = 4.5
)

// escapeDisc draws the root disc with its five windows and extrudes it.
func (b *builder) escapeDisc() (*decad.Body, error) {
	const what = "escape disc"
	s, err := b.flat(what, 0, 0, wheelLow)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	circleAt(s, RootRadius)
	o := s.Origin()
	step := 2 * math.Pi / wheelWindows
	for k := 0; k < wheelWindows; k++ {
		s1 := 90*deg + step*float64(k)
		s2 := s1 + step
		di, do := math.Asin(spokeHalf/windowInner), math.Asin(spokeHalf/windowOuter)
		a := s.CreatePoint(windowInner*math.Cos(s1+di), windowInner*math.Sin(s1+di))
		bb := s.CreatePoint(windowInner*math.Cos(s2-di), windowInner*math.Sin(s2-di))
		c := s.CreatePoint(windowOuter*math.Cos(s2-do), windowOuter*math.Sin(s2-do))
		d := s.CreatePoint(windowOuter*math.Cos(s1+do), windowOuter*math.Sin(s1+do))
		inner := s.CreateArc(o, a, bb)
		outer := s.CreateArc(o, d, c)
		side1 := s.CreateLine(a, d)  // along the spoke at s1, inner to outer
		side2 := s.CreateLine(bb, c) // along the spoke at s2
		s.AddConstraint(
			sketch.NewRadius(inner, windowInner), sketch.NewRadius(outer, windowOuter),
			sketch.NewAngle(ref, side1, s1/deg), sketch.NewDistancePointLine(o, side1, spokeHalf),
			sketch.NewAngle(ref, side2, s2/deg), sketch.NewDistancePointLine(o, side2, spokeHalf),
		)
		// The four corners of the window are inside corners of the wheel: the sides keep their own position and
		// direction, so the rounds leave the sketch fully constrained.
		for i, c := range []struct {
			before, after sketch.Entity
			at            *sketch.Point
		}{{inner, side1, a}, {side2, inner, bb}, {outer, side2, c}, {side1, outer, d}} {
			if err := sketchFillet(s, c.before, c.after, c.at, windowRound); err != nil {
				return nil, fmt.Errorf("%s: round window %d corner %d: %w", what, k+1, i+1, err)
			}
		}
	}
	return b.extrude(s, wheelWindows, wheelThickness, what)
}

// escapeTooth draws tooth 0 (toe, heel, back foot, front foot) and extrudes it through the wheel's thickness.
func (b *builder) escapeTooth() (*decad.Body, error) {
	const what = "escape tooth"
	s, err := b.flat(what, 0, 0, wheelLow)
	if err != nil {
		return nil, err
	}
	g := &b.g
	ref := xRef(s)
	o := s.Origin()
	toe := s.CreatePoint(g.ToothT.X, g.ToothT.Y)
	heel := polar(s, ref, g.HeelRadius, 90*deg+ClubWidth)
	front := s.CreatePoint(g.ToothL.X, g.ToothL.Y)
	back := s.CreatePoint(g.ToothK.X, g.ToothK.Y)
	toeToO := s.CreateLine(toe, o)
	toeToO.SetConstruction(true)
	heelToO := s.CreateLine(heel, o)
	heelToO.SetConstruction(true)
	// Each face leaves its start point, so the angle at the toe and at the heel is read from the line to the centre.
	frontFace := s.CreateLine(toe, front)
	s.CreateLine(toe, heel)
	backFace := s.CreateLine(heel, back)
	s.CreateLine(back, front)
	rf := RootRadius - Overlap
	s.AddConstraint(
		sketch.NewHorizontalDistance(o, toe, 0), sketch.NewVerticalDistance(o, toe, ToeRadius),
		sketch.NewDistance(o, front, rf), sketch.NewAngle(toeToO, frontFace, -Rake/deg),
		sketch.NewDistance(o, back, rf), sketch.NewAngle(heelToO, backFace, -BackAngle/deg),
	)
	return b.extrude(s, 0, wheelThickness, what)
}

// buildEscapeWheel builds the escape wheel in its local frame about the origin: the root disc with five windows, 15
// club teeth with their roots rounded, the D press bore and the chamfers.
func (b *builder) buildEscapeWheel() (*decad.Body, error) {
	disc, err := b.escapeDisc()
	if err != nil {
		return nil, err
	}
	tooth, err := b.escapeTooth()
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(tooth, decad.CircularPattern{Center: r3.NewVec(0, 0, 0), Axis: zAxis, Count: Teeth}).Do(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("escape wheel: pattern teeth: %w", err)
	}
	wheel, err := b.union("escape wheel teeth", disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, err
	}
	selection := decad.Edges(decad.ParallelTo(zAxis), decad.Reflex()).Exactly(2 * Teeth)
	if wheel, err = decad.Fillet(wheel, selection, units.Millimeters(toothRootRound)).Do(b.ctx); err != nil {
		return nil, fmt.Errorf("escape wheel: root fillet: %w", err)
	}
	tool, err := b.dTool("escape wheel bore", 0, 0, wheelLow-wheelBoreReach, BodyBoreRadius, BodyBoreFlat,
		wheelBoreLength)
	if err != nil {
		return nil, err
	}
	if wheel, err = b.cut(wheel, tool, "escape wheel bore"); err != nil {
		return nil, err
	}
	// The five windows are chamfered on both faces; the teeth keep their edges.
	windows := func(l *decad.Loop, e []*decad.Edge) bool { return !l.IsOuter() && len(e) > 4 }
	wheel, err = b.finish(wheel, []float64{wheelLow, wheelHigh}, windows)
	if err != nil {
		return nil, fmt.Errorf("escape wheel windows: %w", err)
	}
	return wheel, nil
}
