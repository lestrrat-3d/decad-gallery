package onewayclutch

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// axisLength is the length of the construction lines that hold polar directions.
const axisLength = 20.0

// run builds one clutch in the local frame. It records every sketch.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// sketchOn starts a sketch on the plane f; extrusions run along f's normal U × V.
func (r *run) sketchOn(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	r.sketches = append(r.sketches, s)
	return s, nil
}

// profile solves s, requires it fully constrained and returns its one valid profile with the given number of holes.
func (r *run) profile(s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(r.ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(r.ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != holes {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("several valid profiles with %d holes", holes)
		}
		chosen = pr
	}
	if chosen == nil {
		return nil, fmt.Errorf("no valid profile with %d holes", holes)
	}
	return chosen, nil
}

// extrude extrudes the profile of s with the given holes by length along its normal.
func (r *run) extrude(s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	pr, err := r.profile(s, holes)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(r.ctx, r.doc, s, pr, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
}

// across is the plane z = const of the local frame, with U = +X and V = +Y.
func across(z float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	return f
}

// mirror maps a point of the counter-clockwise layout to the built direction: (x, y) becomes (sign*x, y).
func (r *run) mirror(v r3.Vec) r3.Vec { return r3.Vec{X: r.dm.Sign * v.X, Y: v.Y} }

// angle maps an angle of the counter-clockwise layout to the built direction, in (-180, 180] degrees.
func (r *run) angle(theta float64) float64 {
	a := 90 + r.dm.Sign*(theta-90)
	return a - 360*math.Floor((a+180)/360)
}

// turn rotates v counter-clockwise by deg degrees about the origin.
func turn(v r3.Vec, deg float64) r3.Vec {
	s, c := math.Sincos(deg * math.Pi / 180)
	return r3.Vec{X: v.X*c - v.Y*s, Y: v.X*s + v.Y*c}
}

// polarSketch holds a sketch's reference line, from which polar construction axes take their angles.
type polarSketch struct {
	r   *run
	s   *sketch.Sketch
	ref *sketch.Line
}

// newPolar adds the horizontal reference line from the sketch origin.
func (r *run) newPolar(s *sketch.Sketch) *polarSketch {
	end := s.CreatePoint(axisLength, 0)
	ref := s.CreateLine(s.Origin(), end)
	ref.SetConstruction(true)
	s.AddConstraint(sketch.NewHorizontal(ref), sketch.NewHorizontalDistance(s.Origin(), end, axisLength))
	return &polarSketch{r: r, s: s, ref: ref}
}

// axis adds a construction line from the origin at angle theta of the counter-clockwise layout.
func (ps *polarSketch) axis(theta float64) *sketch.Line {
	e := ps.r.mirror(turn(r3.Vec{X: axisLength}, theta))
	end := ps.s.CreatePoint(e.X, e.Y)
	l := ps.s.CreateLine(ps.s.Origin(), end)
	l.SetConstruction(true)
	ps.s.AddConstraint(sketch.NewAngle(ps.ref, l, ps.r.angle(theta)), sketch.NewDistance(ps.s.Origin(), end, axisLength))
	return l
}

// arc creates the arc from a to b counter-clockwise in the counter-clockwise layout: the built direction's mirror
// image runs from b to a.
func (r *run) arc(s *sketch.Sketch, center, a, b *sketch.Point) *sketch.Arc {
	if r.dm.Sign < 0 {
		return s.CreateArc(center, b, a)
	}
	return s.CreateArc(center, a, b)
}

// boreLoop adds the D bore to s: an arc of radius r_b the long way round through -Y, closed by its flat at y = +f_b.
func (r *run) boreLoop(s *sketch.Sketch) {
	rb, fb := r.dm.BoreRadius, r.dm.BoreFlat
	w := math.Sqrt(rb*rb - fb*fb)
	o := s.Origin()
	q1, q2 := s.CreatePoint(-w, fb), s.CreatePoint(w, fb)
	bore := s.CreateArc(o, q1, q2)
	flat := s.CreateLine(q2, q1)
	s.AddConstraint(sketch.NewRadius(bore, rb), sketch.NewHorizontal(flat), sketch.NewDistancePointLine(o, flat, fb))
}

// boltHoles adds the three bolt holes: each centre on its axis at BoltCircle, radius m2fastener.HoleRadius.
func (r *run) boltHoles(s *sketch.Sketch, ps *polarSketch) {
	for _, a := range BoltAngles {
		pos := polar(BoltCircle, a)
		// The bolt angles are symmetric about x = 0, so they need no mirror.
		end := s.CreatePoint(axisLength*math.Cos(a*math.Pi/180), axisLength*math.Sin(a*math.Pi/180))
		axis := s.CreateLine(s.Origin(), end)
		axis.SetConstruction(true)
		s.AddConstraint(sketch.NewAngle(ps.ref, axis, a), sketch.NewDistance(s.Origin(), end, axisLength))
		center := s.CreatePoint(pos.X, pos.Y)
		s.AddConstraint(sketch.NewPointOnLine(center, axis), sketch.NewDistance(s.Origin(), center, BoltCircle))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(center, m2fastener.HoleRadius), m2fastener.HoleRadius))
	}
}

// plateSketch sketches a plate's profile on the plane z: the circle PlateRadius, the D bore and the three bolt holes.
func (r *run) plateSketch(z float64) (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(z))
	if err != nil {
		return nil, err
	}
	ps := r.newPolar(s)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), PlateRadius), PlateRadius))
	r.boreLoop(s)
	r.boltHoles(s, ps)
	return s, nil
}

// hubSketch sketches the hub's whole outline on its lower face: for each of the three pawls the core arc, the web root
// round, the web's back face, the arm's outer arc, the head's flank, tip round and lock face, the arm's inner arc and the
// slot's full round; with the D bore and the three bolt holes as holes. Section 6.2 of the design lists the outline.
func (r *run) hubSketch() (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(HubFrom))
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	ps := r.newPolar(s)
	dm := r.dm
	pt := func(v r3.Vec, deg float64) *sketch.Point {
		m := r.mirror(turn(v, deg))
		return s.CreatePoint(m.X, m.Y)
	}
	slotC := polar((CoreRadius+ArmInner)/2, SlotAngle)
	type pawl struct {
		a, w, hf, apex, hl, sIn, tCore, c *sketch.Point
		coreArc, outerArc, innerArc       *sketch.Arc
		slotArc                           *sketch.Arc
		webLine, flankLine, lockLine      *sketch.Line
	}
	var pw [Pawls]pawl
	for k := range pw {
		d := 120.0 * float64(k)
		pw[k] = pawl{
			a:     pt(polar(CoreRadius, WebAngle), d),
			w:     pt(polar(ArmOuter, WebAngle), d),
			hf:    pt(dm.FlankEnd, d),
			apex:  pt(dm.Apex, d),
			hl:    pt(dm.LockEnd, d),
			sIn:   pt(polar(ArmInner, SlotAngle), d),
			tCore: pt(polar(CoreRadius, SlotAngle), d),
			c:     pt(slotC, d),
		}
	}
	for k := range pw {
		prev := (k + Pawls - 1) % Pawls
		cur := &pw[k]
		cur.coreArc = r.arc(s, o, pw[prev].tCore, cur.a)
		cur.webLine = s.CreateLine(cur.a, cur.w)
		cur.outerArc = r.arc(s, o, cur.w, cur.hf)
		cur.flankLine = s.CreateLine(cur.hf, cur.apex)
		cur.lockLine = s.CreateLine(cur.apex, cur.hl)
		cur.innerArc = r.arc(s, o, cur.sIn, cur.hl)
		cur.slotArc = r.arc(s, cur.c, cur.tCore, cur.sIn)
	}
	for k := range pw {
		d := 120.0 * float64(k)
		next := (k + 1) % Pawls
		cur := &pw[k]
		webAxis, slotAxis, lockAxis := ps.axis(WebAngle+d), ps.axis(SlotAngle+d), ps.axis(LockAngle+d)
		s.AddConstraint(
			sketch.NewRadius(cur.coreArc, CoreRadius), sketch.NewRadius(cur.outerArc, ArmOuter),
			sketch.NewRadius(cur.innerArc, ArmInner),
			sketch.NewParallel(webAxis, cur.webLine), sketch.NewPointOnLine(cur.w, webAxis),
			sketch.NewAngle(ps.ref, cur.flankLine, r.angle(dm.RampAngle+d)),
			sketch.NewDistancePointLine(o, cur.flankLine, dm.FlankDistance),
			sketch.NewParallel(lockAxis, cur.lockLine), sketch.NewDistancePointLine(o, cur.lockLine, HeadGap),
			sketch.NewPointOnLine(cur.c, slotAxis),
			sketch.NewTangentCircles(cur.slotArc, cur.innerArc, true),
			sketch.NewTangentCircles(cur.slotArc, pw[next].coreArc, false),
		)
	}
	for k := range pw {
		cur := &pw[k]
		if _, err := sketch.CreateFillet(s, cur.coreArc, cur.webLine, cur.a, WebRound); err != nil {
			return nil, fmt.Errorf("round the web root of pawl %d: %w", k+1, err)
		}
		if _, err := sketch.CreateFillet(s, cur.flankLine, cur.lockLine, cur.apex, TipRound); err != nil {
			return nil, fmt.Errorf("round the head tip of pawl %d: %w", k+1, err)
		}
	}
	r.boreLoop(s)
	r.boltHoles(s, ps)
	return s, nil
}

// toothSketch sketches the ring's tooth tool on the plane z: the whole void inside the teeth, one closed loop. Lock face j
// lies on the radial axis at LockAngle + 15 degrees * j from Q_j (ToothRoot) in to P_j (ToothTip); the ramp of tooth j
// runs from P_j to Q_{j+1}; each Q_j is rounded by RootRound.
func (r *run) toothSketch(z float64) (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(z))
	if err != nil {
		return nil, err
	}
	o := s.Origin()
	ps := r.newPolar(s)
	pitch := 360.0 / Teeth
	rampDistance := r.dm.RampDistance
	at := func(radius float64, j int) *sketch.Point {
		m := r.mirror(polar(radius, LockAngle+pitch*float64(j)))
		return s.CreatePoint(m.X, m.Y)
	}
	var tip, root [Teeth]*sketch.Point
	for j := range tip {
		tip[j], root[j] = at(ToothTip, j), at(ToothRoot, j)
	}
	var lock, ramp [Teeth]*sketch.Line
	for j := range lock {
		lock[j] = s.CreateLine(root[j], tip[j])
		ramp[j] = s.CreateLine(tip[j], root[(j+1)%Teeth])
	}
	for j := range lock {
		axis := ps.axis(LockAngle + pitch*float64(j))
		s.AddConstraint(
			sketch.NewPointOnLine(tip[j], axis), sketch.NewDistance(o, tip[j], ToothTip),
			sketch.NewParallel(axis, lock[j]), sketch.NewDistancePointLine(o, ramp[j], rampDistance),
		)
	}
	for j := range root {
		prev := (j + Teeth - 1) % Teeth
		if _, err := sketch.CreateFillet(s, ramp[prev], lock[j], root[j], RootRound); err != nil {
			return nil, fmt.Errorf("round the root of tooth %d: %w", j+1, err)
		}
	}
	return s, nil
}

// meridian sketches the closed polygon through the (r, z) vertices on the half plane holding the Z axis: U runs along +Z
// and V is the radius +X, so every vertex is dimensioned from the origin. It revolves the polygon a full turn about Z.
func (r *run) meridian(vertices [][2]float64) (*decad.Body, error) {
	f, err := r3.NewFrame(r3.Vec{}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	points := make([]*sketch.Point, len(vertices))
	for i, v := range vertices {
		points[i] = sketchcheck.DimensionPoint(s, v[1], v[0])
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(r.ctx, r.doc, s, pr, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.Vec{Z: 1}},
		decad.FullRevolution{})
}
