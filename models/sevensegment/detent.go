package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

const (
	ratchetBoreRadius = 3.25
	ratchetBoreFlat   = 2.75
	ratchetSamples    = 17 // samples of each flank spline
)

// rotate turns v counter-clockwise by deg degrees about the origin.
func rotate(v r3.Vec, deg float64) r3.Vec {
	s, c := math.Sincos(deg * math.Pi / 180)
	return r3.Vec{X: v.X*c - v.Y*s, Y: v.X*s + v.Y*c}
}

// rootRound solves the root round of a tooth whose lock face lies on the ray at angle 0: the circle of RatchetRootRound
// tangent to the lock face (centre above it) and to the spiral from outside. It returns the centre and the tangent point
// on the spiral.
func rootRound() (centre, onSpiral r3.Vec) {
	var spir []r3.Vec
	for i := 0; i <= 36000; i++ {
		rel := float64(i) * 0.001
		spir = append(spir, polar(flank(rel), rel))
	}
	dist := func(cx float64) float64 {
		d, _ := closest(spir, r3.Vec{X: cx, Y: RatchetRootRound})
		return d
	}
	lo, hi := RatchetRoot, 12.0
	for range 60 {
		mid := (lo + hi) / 2
		if dist(mid) < RatchetRootRound {
			lo = mid
		} else {
			hi = mid
		}
	}
	centre = r3.Vec{X: (lo + hi) / 2, Y: RatchetRootRound}
	_, onSpiral = closest(spir, centre)
	return centre, onSpiral
}

// ratchetSketch sketches the wheel's outline on its back face: ten teeth as one closed loop, plus the D bore as a hole.
func (r *run) ratchetSketch() (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(0))
	if err != nil {
		return nil, err
	}
	centre, bp := rootRound()
	bAngle := math.Atan2(bp.Y, bp.X) * 180 / math.Pi
	type tooth struct {
		tip, a, b *sketch.Point
		c         *sketch.Point
	}
	ps := newPolar(s)
	var teeth [RatchetTeeth]tooth
	for j := range teeth {
		phi := r.dm.LockPhase + 36*float64(j)
		at := func(v r3.Vec) r3.Vec { return rotate(v, phi) }
		t := &teeth[j]
		tp := at(r3.Vec{X: RatchetTip})
		t.tip = sketchcheck.DimensionPoint(s, tp.X, tp.Y)
		ap := at(r3.Vec{X: centre.X})
		t.a = s.CreatePoint(ap.X, ap.Y)
		cp := at(centre)
		t.c = s.CreatePoint(cp.X, cp.Y)
		bq := at(bp)
		t.b = sketchcheck.DimensionPoint(s, bq.X, bq.Y)
	}
	for j := range teeth {
		t, next := &teeth[j], &teeth[(j+1)%RatchetTeeth]
		phi := r.dm.LockPhase + 36*float64(j)
		// Lock face from the previous tooth's tip to this tooth's root round.
		lock := s.CreateLine(t.tip, t.a)
		arc := arcBetween(s, at0(phi, centre), t.c, at0(phi, r3.Vec{X: centre.X}), at0(phi, bp), t.a, t.b)
		s.AddConstraint(sketch.NewTangent(lock, arc), sketch.NewRadius(arc, RatchetRootRound),
			sketch.NewPointOnLine(t.a, ps.axis(phi)))
		// Spiral spline from the round's tangent point to the next tooth's tip.
		pts := []*sketch.Point{t.b}
		for i := 1; i < ratchetSamples-1; i++ {
			rel := bAngle + (36-bAngle)*float64(i)/float64(ratchetSamples-1)
			v := polar(flank(rel), phi+rel)
			pts = append(pts, sketchcheck.DimensionPoint(s, v.X, v.Y))
		}
		pts = append(pts, next.tip)
		if _, err := s.CreateFitSpline(pts...); err != nil {
			return nil, fmt.Errorf("spiral spline %d: %w", j+1, err)
		}
	}
	dBoreLoop(s, ratchetBoreRadius, ratchetBoreFlat)
	return s, nil
}

// at0 rotates v by phi degrees.
func at0(phi float64, v r3.Vec) r3.Vec { return rotate(v, phi) }

// ratchetWheel builds the ratchet wheel in its local frame and places it in the world at theta = 0.
func (r *run) ratchetWheel() (*decad.Body, error) {
	s, err := r.ratchetSketch()
	if err != nil {
		return nil, err
	}
	body, err := r.extrude(s, 1, RatchetThickness, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	f := frontFrame(-37.85)
	return placement.Place(r.ctx, body, &f)
}

const (
	pawlBackV     = 2.26  // v of the point where the back flank meets the arm's inner edge (pawl frame)
	pawlInnerSlot = 3.0   // width of the slot between the arm's inner edge and the plate at the root
	pawlOuterSide = 3.55  // u of the outer slot side at the root, past PawlArmInner (the slot is 1.65 wide there)
	pawlOuterOpen = 4.75  // u of the outer slot side where it meets the window circle, past PawlArmInner
	pawlRound     = 1.0   // back-flank and window rounds
	pawlRootIn    = 2.0   // root round on the arm's inner (tension) side
	pawlRootOut   = 1.0   // root round on the arm's outer side
	slotCorner    = 0.5   // rounds at the slot ends
	plateTop      = 21.0  // v of the plate's upper edge (z 49)
	plateBottom   = -28.0 // v of the plate's lower edge (z 0)
	plateBoltV    = -18.0 // v of the bolt holes (z 10)
	plateBoltU    = 21.0
	detentFace    = -37.25 // world Y of the plate's inner face
	chamfer       = 0.5    // outer-loop chamfer of the plates
)

// detentFrame is the plate's sketch frame: origin on the shaft axis in its inner face, U = +X, V = +Z, N = -Y.
func detentFrame() r3.Frame { return frontFrame(detentFace) }

// pawlGeom is one pawl in its own frame: seat on +u, arm along +v with its inner edge on u = PawlArmInner. Rotate by
// PawlAngle (pawl A) or PawlAngle + 180 (pawl B) into the sketch.
type pawlGeom struct {
	seat, foot, lk, tp, b  r3.Vec // tip round centre, lock-flank foot, lock flank on the outer edge, back-flank points
	ri, ro, pi, po         r3.Vec // root corners of the arm (inner, outer) and of the slot (inner, outer)
	jc, jo                 r3.Vec // where the inner and outer slot sides meet the window circle
	vr                     float64
	backDist, edgeO, edgeA float64 // back flank distance; outer edge distance and angle from the arm axis
	slotO, slotA           float64 // outer slot side's distance from the origin and angle from the arm axis
}

// lineDist is the distance from p to the line through a and b.
func lineDist(p, a, b r3.Vec) float64 {
	d := b.Sub(a)
	return math.Abs(d.X*(p.Y-a.Y)-d.Y*(p.X-a.X)) / d.Len()
}

// pawlGeometry computes pawl A's outline points and the distances and angles that constrain them, in the pawl frame.
func (r *run) pawlGeometry() pawlGeom {
	in := r.dm.PawlArmInner
	var g pawlGeom
	g.seat = polar(r.dm.SeatRadius, 0)
	lockDir := polar(1, -r.dm.SeatAngle)
	nl := r3.Vec{X: -lockDir.Y, Y: lockDir.X}
	g.foot = g.seat.Sub(nl.Scale(PawlTipRound))
	g.lk = g.foot.Add(lockDir.Scale((in + PawlArmTip - g.foot.X) / lockDir.X))
	g.b = r3.Vec{X: in, Y: pawlBackV}
	d := g.seat.Sub(g.b)
	dl := d.Len()
	alpha := math.Asin(PawlTipRound / dl)
	for _, sgn := range []float64{1, -1} {
		dir := rotate(d.Scale(1/dl), sgn*alpha*180/math.Pi)
		cand := g.b.Add(dir.Scale(dl * math.Cos(alpha)))
		if cr := g.b.Sub(cand); cr.X*(g.seat.Y-cand.Y)-cr.Y*(g.seat.X-cand.X) < 0 {
			g.tp = cand
		}
	}
	g.vr = pawlBackV + PawlArmLength
	g.ri = r3.Vec{X: in, Y: g.vr}
	g.ro = r3.Vec{X: in + PawlArmRoot, Y: g.vr}
	g.pi = r3.Vec{X: in - pawlInnerSlot, Y: g.vr}
	g.po = r3.Vec{X: in + pawlOuterSide, Y: g.vr}
	uc, uo := in-pawlInnerSlot, in+pawlOuterOpen
	g.jc = r3.Vec{X: uc, Y: math.Sqrt(WindowRadius*WindowRadius - uc*uc)}
	g.jo = r3.Vec{X: uo, Y: math.Sqrt(WindowRadius*WindowRadius - uo*uo)}
	o := r3.Vec{}
	g.backDist = lineDist(o, g.tp, g.b)
	// Signed angles from the arm axis (pawl frame +v) to each line's start-to-end direction, as sketch.NewAngle reads them.
	angleFromArm := func(a, b r3.Vec) float64 { return math.Atan2(b.Y-a.Y, b.X-a.X)*180/math.Pi - 90 }
	g.edgeO, g.edgeA = lineDist(o, g.lk, g.ro), angleFromArm(g.ro, g.lk)
	g.slotO, g.slotA = lineDist(o, g.po, g.jo), angleFromArm(g.jo, g.po)
	return g
}

// windowLoop adds the window to s: the circle WindowRadius with two slots, each holding a straight pawl arm whose root
// joins the plate at the slot's end.
func (r *run) windowLoop(s *sketch.Sketch, ps *polarSketch) error {
	o := s.Origin()
	g := r.pawlGeometry()
	type pawl struct {
		jo, po, ro, lk, foot, tp, b, ri, pi, jc, seat       *sketch.Point
		slotO, rootO, edge, lock, back, inner, rootI, slotI *sketch.Line
		tip                                                 *sketch.Arc
	}
	var ps2 [2]pawl
	for k := range ps2 {
		rot := PawlAngle + 180*float64(k)
		at := func(v r3.Vec) r3.Vec { return rotate(v, rot) }
		mk := func(v r3.Vec) *sketch.Point { w := at(v); return s.CreatePoint(w.X, w.Y) }
		p := &ps2[k]
		p.jo, p.po, p.ro, p.lk, p.foot, p.tp = mk(g.jo), mk(g.po), mk(g.ro), mk(g.lk), mk(g.foot), mk(g.tp)
		p.b, p.ri, p.pi, p.jc = mk(g.b), mk(g.ri), mk(g.pi), mk(g.jc)
		sc := at(g.seat)
		p.seat = sketchcheck.DimensionPoint(s, sc.X, sc.Y)
	}
	for k := range ps2 {
		rot := PawlAngle + 180*float64(k)
		at := func(v r3.Vec) r3.Vec { return rotate(v, rot) }
		p := &ps2[k]
		arm := ps.axis(rot + 90)
		p.slotO = s.CreateLine(p.jo, p.po)
		p.rootO = s.CreateLine(p.po, p.ro)
		p.edge = s.CreateLine(p.ro, p.lk)
		p.lock = s.CreateLine(p.lk, p.foot)
		p.tip = arcBetween(s, at(g.seat), p.seat, at(g.foot), at(g.tp), p.foot, p.tp)
		p.back = s.CreateLine(p.tp, p.b)
		p.inner = s.CreateLine(p.b, p.ri)
		p.rootI = s.CreateLine(p.ri, p.pi)
		p.slotI = s.CreateLine(p.pi, p.jc)
		s.AddConstraint(
			sketch.NewDistancePointLine(o, p.slotO, g.slotO), sketch.NewAngle(arm, p.slotO, g.slotA),
			sketch.NewPerpendicular(arm, p.rootO), sketch.NewDistancePointLine(o, p.rootO, g.vr),
			sketch.NewDistancePointLine(o, p.edge, g.edgeO), sketch.NewAngle(arm, p.edge, g.edgeA),
			sketch.NewParallel(ps.axis(r.dm.LockPhase+180*float64(k)), p.lock), sketch.NewTangent(p.lock, p.tip),
			sketch.NewRadius(p.tip, PawlTipRound), sketch.NewTangent(p.back, p.tip),
			sketch.NewDistancePointLine(o, p.back, g.backDist),
			sketch.NewParallel(arm, p.inner), sketch.NewDistancePointLine(o, p.inner, r.dm.PawlArmInner),
			sketch.NewPerpendicular(arm, p.rootI), sketch.NewDistancePointLine(o, p.rootI, g.vr),
			sketch.NewParallel(arm, p.slotI), sketch.NewDistancePointLine(o, p.slotI, r.dm.PawlArmInner-pawlInnerSlot),
		)
	}
	var window [2]*sketch.Arc
	for k := range ps2 {
		next := (k + 1) % 2
		rot, rotN := PawlAngle+180*float64(k), PawlAngle+180*float64(next)
		window[k] = arcBetween(s, r3.Vec{}, o, rotate(g.jc, rot), rotate(g.jo, rotN), ps2[k].jc, ps2[next].jo)
		s.AddConstraint(sketch.NewRadius(window[k], WindowRadius))
	}
	// The rounds. A fillet removes its corner point and the constraints on it.
	for k := range ps2 {
		prev := (k + 1) % 2
		p := &ps2[k]
		fail := func(what string, err error) error { return fmt.Errorf("round the %s, pawl %d: %w", what, k+1, err) }
		if _, err := sketch.CreateFillet(s, window[prev], p.slotO, p.jo, pawlRound); err != nil {
			return fail("window to the outer slot side", err)
		}
		if _, err := sketch.CreateFillet(s, p.slotO, p.rootO, p.po, slotCorner); err != nil {
			return fail("outer slot end", err)
		}
		if _, err := sketch.CreateFillet(s, p.rootO, p.edge, p.ro, pawlRootOut); err != nil {
			return fail("outer root", err)
		}
		if _, err := sketch.CreateFillet(s, p.back, p.inner, p.b, pawlRound); err != nil {
			return fail("back flank to the arm's inner edge", err)
		}
		if _, err := sketch.CreateFillet(s, p.inner, p.rootI, p.ri, pawlRootIn); err != nil {
			return fail("inner root", err)
		}
		if _, err := sketch.CreateFillet(s, p.rootI, p.slotI, p.pi, slotCorner); err != nil {
			return fail("inner slot end", err)
		}
		if _, err := sketch.CreateFillet(s, p.slotI, window[k], p.jc, pawlRound); err != nil {
			return fail("inner slot side to the window", err)
		}
	}
	return nil
}

// detentPlate builds the detent plate in the world: its outline with the window and the two bolt holes, the outer-loop
// chamfers and the two head pockets.
func (r *run) detentPlate() (*decad.Body, error) {
	s, err := r.sketchOn(detentFrame())
	if err != nil {
		return nil, err
	}
	ps := newPolar(s)
	c := [4]*sketch.Point{
		sketchcheck.DimensionPoint(s, -WallHalfWidth, plateBottom), sketchcheck.DimensionPoint(s, WallHalfWidth, plateBottom),
		sketchcheck.DimensionPoint(s, WallHalfWidth, plateTop), sketchcheck.DimensionPoint(s, -WallHalfWidth, plateTop),
	}
	for i := range c {
		s.CreateLine(c[i], c[(i+1)%4])
	}
	for _, x := range [2]float64{-plateBoltU, plateBoltU} {
		ctr := sketchcheck.DimensionPoint(s, x, plateBoltV)
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(ctr, m2fastener.HoleRadius), m2fastener.HoleRadius))
	}
	if err := r.windowLoop(s, ps); err != nil {
		return nil, err
	}
	body, err := r.extrude(s, 3, DetentThickness, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Y: 1}, []float64{detentFace, detentFace - DetentThickness},
		edgefinish.Long(edgefinish.Outer), chamfer); err != nil {
		return nil, fmt.Errorf("chamfer outer loops: %w", err)
	}
	for i, site := range r.dm.Sites[8:] {
		tool, sk, err := m2fastener.PocketTool(r.ctx, r.doc, site)
		if err != nil {
			return nil, fmt.Errorf("head pocket %d: %w", i+1, err)
		}
		r.sketches = append(r.sketches, sk)
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut head pocket %d: %w", i+1, err)
		}
	}
	return body, nil
}
