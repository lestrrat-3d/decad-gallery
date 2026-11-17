package swisslever

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Balance wheel, hairspring and retainer heights and dimensions in mm.
const (
	balanceLow        = 13.40
	balanceThickness  = 5.0
	balanceHigh       = balanceLow + balanceThickness
	balanceRimOuter   = 42.0
	balanceRimInner   = 36.0
	balanceHubRadius  = 7.0
	balanceArmHalf    = 2.0 // half the width of an arm
	balanceWindowRnd  = 2.0
	balanceFirstArm   = 45 * deg
	balanceArms       = 4
	balanceBossRadius = 7.0
	balanceBossHeight = 1.0
	balanceBoreReach  = 0.75
	balanceBoreLength = 7.5
	springLow         = 19.65
	retainerLow       = 22.90
	retainerHigh      = 25.90
	retainerRadius    = 5.0
	retainerBoreReach = 0.75
	retainerBoreLong  = 4.5
)

// balanceDisc draws the rim disc with its four windows between the arms and extrudes it.
func (b *builder) balanceDisc() (*decad.Body, error) {
	const what = "balance wheel"
	s, err := b.flat(what, 0, 0, balanceLow)
	if err != nil {
		return nil, err
	}
	ref := xRef(s)
	circleAt(s, balanceRimOuter)
	o := s.Origin()
	step := 2 * math.Pi / balanceArms
	for k := 0; k < balanceArms; k++ {
		s1 := balanceFirstArm + step*float64(k)
		s2 := s1 + step
		di, do := math.Asin(balanceArmHalf/balanceHubRadius), math.Asin(balanceArmHalf/balanceRimInner)
		a := s.CreatePoint(balanceHubRadius*math.Cos(s1+di), balanceHubRadius*math.Sin(s1+di))
		bb := s.CreatePoint(balanceHubRadius*math.Cos(s2-di), balanceHubRadius*math.Sin(s2-di))
		c := s.CreatePoint(balanceRimInner*math.Cos(s2-do), balanceRimInner*math.Sin(s2-do))
		d := s.CreatePoint(balanceRimInner*math.Cos(s1+do), balanceRimInner*math.Sin(s1+do))
		inner := s.CreateArc(o, a, bb)
		outer := s.CreateArc(o, d, c)
		side1 := s.CreateLine(a, d)
		side2 := s.CreateLine(bb, c)
		s.AddConstraint(
			sketch.NewRadius(inner, balanceHubRadius), sketch.NewRadius(outer, balanceRimInner),
			sketch.NewAngle(ref, side1, s1/deg), sketch.NewDistancePointLine(o, side1, balanceArmHalf),
			sketch.NewAngle(ref, side2, s2/deg), sketch.NewDistancePointLine(o, side2, balanceArmHalf),
		)
		for i, c := range []struct {
			before, after sketch.Entity
			at            *sketch.Point
		}{{inner, side1, a}, {side2, inner, bb}, {outer, side2, c}, {side1, outer, d}} {
			if err := sketchFillet(s, c.before, c.after, c.at, balanceWindowRnd); err != nil {
				return nil, fmt.Errorf("%s: round window %d corner %d: %w", what, k+1, i+1, err)
			}
		}
	}
	return b.extrude(s, balanceArms, balanceThickness, what)
}

// buildBalanceWheel builds the balance wheel about the origin with its D flat on +X: the rim, four arms, the hub boss
// on the front, the D press bore and the chamfers on the rim and the windows.
func (b *builder) buildBalanceWheel() (*decad.Body, error) {
	wheel, err := b.balanceDisc()
	if err != nil {
		return nil, err
	}
	face := decad.Faces(decad.FaceCreatedBy(decad.CapEnd(wheel))).Exactly(1)
	if wheel, err = b.boss(wheel, face, r3.NewVec(0, 0, balanceHigh), 2*balanceBossRadius, balanceBossHeight,
		"balance boss"); err != nil {
		return nil, err
	}
	tool, err := b.dTool("balance bore", 0, 0, balanceLow-balanceBoreReach, BodyBoreRadius, BodyBoreFlat,
		balanceBoreLength)
	if err != nil {
		return nil, err
	}
	if wheel, err = b.cut(wheel, tool, "balance bore"); err != nil {
		return nil, err
	}
	if b.skipBalanceChamfer {
		return wheel, nil
	}
	keep := func(l *decad.Loop, e []*decad.Edge) bool { return l.IsOuter() || len(e) > 4 }
	out, err := b.finishExcept(wheel, []float64{balanceLow, balanceHigh}, keep, bossRootEdge)
	if err != nil {
		return nil, fmt.Errorf("balance wheel edges: %w", err)
	}
	return out, nil
}

// bossRootEdge reports whether e is an edge where the hub boss's root round meets the wheel's front face: both its
// ends lie on the circle of radius balanceBossRadius + rootRound about the staff. The round is tangent to the face
// there, so the edge cannot be chamfered and is not a corner of the face.
func bossRootEdge(e *decad.Edge) bool {
	const want = balanceBossRadius + rootRound
	for _, p := range []r3.Vec{e.Start().Position().Value, e.End().Position().Value} {
		if math.Abs(math.Hypot(p.X, p.Y)-want) > 1e-3 || math.Abs(p.Z-balanceHigh) > 1e-3 {
			return false
		}
	}
	return true
}

// finishWith is finish with the error naming the step.
func (b *builder) finishWith(body *decad.Body, offsets []float64, keep edgefinish.LoopFilter, what string) (*decad.Body, error) {
	out, err := b.finish(body, offsets, keep)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return out, nil
}

// buildHairspring builds the hairspring with spiralspring.Build on the balance staff, its bore flat facing the staff's
// flat, and records the item's profile sketch.
func (b *builder) buildHairspring() (*decad.Body, error) {
	g := &b.g
	sp := SpringParams()
	frame, err := r3.NewFrame(r3.NewVec(g.B.X, g.B.Y, springLow),
		r3.NewVec(math.Cos(g.GammaS), math.Sin(g.GammaS), 0),
		r3.NewVec(-math.Sin(g.GammaS), math.Cos(g.GammaS), 0))
	if err != nil {
		return nil, fmt.Errorf("hairspring: placement: %w", err)
	}
	sp.Placement = &frame
	m, err := spiralspring.Build(b.ctx, b.doc, sp)
	if err != nil {
		return nil, fmt.Errorf("hairspring: %w", err)
	}
	b.sketches = append(b.sketches, m.Profile)
	return m.Body, nil
}

// buildRetainer builds the ring that stops the hairspring hub on the front of the balance staff, about the origin.
func (b *builder) buildRetainer() (*decad.Body, error) {
	ring, err := b.discAt("hairspring retainer", 0, 0, retainerLow, retainerRadius, retainerHigh-retainerLow)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("retainer bore", 0, 0, retainerLow-retainerBoreReach, BodyBoreRadius, BodyBoreFlat,
		retainerBoreLong)
	if err != nil {
		return nil, err
	}
	return b.cut(ring, bore, "hairspring retainer bore")
}
