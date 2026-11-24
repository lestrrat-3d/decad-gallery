package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Flag is one printed two-colour rotor: a yellow body (the face slab and the tips: the slab runs onto the 45 degree
// facets, 0.8 inside their edges) and the black body that holds everything else. Each body has one filament colour;
// they share their boundary faces.
type Flag struct{ Yellow, Black *decad.Body }

// Rotor dimensions that are not exported, in mm. The rotor is built in a local frame: local x is the rotor's x (the face
// normal at turn 0 is along it), local y is up, and local z is the axis u.
//
// The shells under the faces and the yellow's inset (0.8), the chamfer legs (0.5, 0.7, 0.8) and the pin pocket's play (0.20)
// are print limits and fits: they keep their size in mm.
const (
	rotorEdgeRound  = 0.75                                   // long edges; swept radius 4.875*sqrt2 + 0.75 = 7.644
	rotorSquareHalf = 15.0                                   // the section is square for |u| <= 10
	rotorTipB       = 18.525                                 // prism B's end points, so the pyramid edges stay sharp
	rotorFlat       = -3.45                                  // the black body keeps y >= -2.3
	notchFloor      = RotorHalfWidth - 0.8                   // shells under both faces are 0.8
	notchEnd        = 0.5                                    // notch end chamfer leg
	notchApex       = 0.8                                    // apex chamfer leg
	slotStart       = 1.5                                    //
	slotEnd         = 4.05                                   //
	slotRadius      = flagPinRadius + 0.2                    // the pin pocket's radius: the pin's 0.5 plus 0.20
	pocketFloorR    = 0.9                                    // the pin pocket's floor round
	notchMouth      = 0.7                                    // the mouth chamfer of a notch end within 0.8 of the tip slant, on hidden face 315
	yellowDepth     = 0.8                                    //
	yellowHalfWidth = RotorHalfWidth - 0.8                   //
	yellowTipK      = RotorHalfWidth + rotorSquareHalf - 0.8 // the yellow's chamfer |b| + |u| <= yellowTipK runs 0.8 inside the pyramid edges (19.825)
	yellowTipInner  = yellowTipK                             // the yellow's inner face a >= yellowTipInner - |u| runs 0.8 under the facet (19.825 - |u|)
	yellowTipEnd    = 17.4                                   // the yellow ends at |u| = yellowTipEnd, clear of the pivot bore's mouth
)

// rotorNotches lists a rotor type's notches as (u0, u1, pocket end) in the rotor's local u. Type A (segments a, g, d) has one
// notch; type B (b, c, e, f) has two, the first used by b and e and the second by c and f.
func rotorNotches(s Segment) [][3]float64 {
	switch s {
	case SegA, SegG, SegD:
		return [][3]float64{{-8.4, -0.3, 4.2}}
	}
	return [][3]float64{{-14.1, -6.0, -1.5}, {2.4, 10.5, 15.0}}
}

// rotorFrame is the rotor's local frame in the world at turn 0: origin on the axis at the rotor's middle, local x along s,
// local y up and local z along u. a, g, d have u = +X and s = +Y; b, c, e, f have u = +Y and s = -X.
func rotorFrame(s Segment) r3.Frame {
	cx, cy, _ := segmentCentre(s)
	u := r3.Vec{Y: 1}
	if _, _, horizontal := segmentCentre(s); horizontal {
		u = r3.Vec{X: 1}
	}
	sAxis := r3.Vec{Z: 1}.Cross(u) // s = Z x u
	f, _ := r3.NewFrame(r3.Vec{X: cx, Y: cy, Z: RotorAxisZ}, sAxis, r3.Vec{Z: 1})
	return f
}

func local(o, u, v r3.Vec) r3.Frame {
	f, err := r3.NewFrame(o, u, v)
	if err != nil {
		panic(err)
	}
	return f
}

// polyBody extrudes a closed polygon on frame f, each side held by its direction and its distance from the origin.
func (r *run) polyBody(f r3.Frame, pts [][2]float64, ext decad.Extent) (*decad.Body, error) {
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	polygon(s, pts)
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(r.ctx, r.doc, s, pr, ext)
}

func symmetric(d float64) decad.Extent {
	return decad.Symmetric{D: units.Millimeters(d), FullLength: true}
}

var (
	axisA = r3.Vec{X: 1 / math.Sqrt2, Y: 1 / math.Sqrt2}
	axisB = r3.Vec{X: -1 / math.Sqrt2, Y: 1 / math.Sqrt2}
	axisZ = r3.Vec{Z: 1}
)

// octagon is a prism outline in the plane of a diagonal and u: slanted edges at 45 degrees through (3.75, +-10), overshoot to
// c, and end points (+-cend, +-uend).
func octagon(c, cend, uend float64) [][2]float64 {
	u0 := rotorSquareHalf - (c - RotorHalfWidth)
	return [][2]float64{{c, -u0}, {c, u0}, {cend, uend}, {-cend, uend}, {-c, u0}, {-c, -u0}, {-cend, -uend}, {cend, -uend}}
}

// rotorSplit builds the rotor and splits it into its yellow slab and the rest, in the local frame and at turn 0, and
// returns the two bodies with the rotor's volume before the split.
func (r *run) rotorSplit() (yellow, black *decad.Body, rotorVolume float64, err error) {
	wrap := func(what string, e error) error { return fmt.Errorf("rotor %s: %w", what, e) }
	// 1. The pyramid tips: two prisms intersected.
	pa, err := r.polyBody(local(r3.Vec{}, axisA, axisZ), octagon(6.375, 2.25, RotorHalfLength), symmetric(21.0))
	if err != nil {
		return nil, nil, 0, wrap("prism A", err)
	}
	pb, err := r.polyBody(local(r3.Vec{}, axisB, axisZ), octagon(6.375, 2.1, rotorTipB), symmetric(21.0))
	if err != nil {
		return nil, nil, 0, wrap("prism B", err)
	}
	tips, err := decad.Intersect(r.ctx, pa, pb)
	if err != nil {
		return nil, nil, 0, wrap("tips", err)
	}
	// 2. The rounded square section: four lines held by their distance from the axis and their angle, filleted.
	s, err := r.sketchOn(local(r3.Vec{Z: -19.5}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	if err != nil {
		return nil, nil, 0, wrap("section sketch", err)
	}
	k := RotorHalfWidth * math.Sqrt2
	v := [][2]float64{{k, 0}, {0, k}, {-k, 0}, {0, -k}}
	pts := make([]*sketch.Point, 4)
	for i, q := range v {
		pts[i] = s.CreatePoint(q[0], q[1])
	}
	lines := make([]*sketch.Line, 4)
	o := s.Origin()
	ps := newPolar(s)
	for i := range v {
		lines[i] = s.CreateLine(pts[i], pts[(i+1)%4])
		d := v[(i+1)%4]
		a := math.Atan2(d[1]-v[i][1], d[0]-v[i][0]) * 180 / math.Pi
		s.AddConstraint(sketch.NewDistancePointLine(o, lines[i], RotorHalfWidth), sketch.NewAngle(ps.ref, lines[i], a))
	}
	for i := range pts {
		if _, err := sketch.CreateFillet(s, lines[(i+3)%4], lines[i], pts[i], rotorEdgeRound); err != nil {
			return nil, nil, 0, wrap(fmt.Sprintf("section round %d", i+1), err)
		}
	}
	section, err := r.extrude(s, 0, 39.0, decad.Along)
	if err != nil {
		return nil, nil, 0, wrap("section", err)
	}
	rotor, err := decad.Intersect(r.ctx, tips, section)
	if err != nil {
		return nil, nil, 0, wrap("rotor", err)
	}
	rv, err := rotor.Volume()
	if err != nil {
		return nil, nil, 0, wrap("volume", err)
	}
	// 3. The colour split: the slab T = P1 x P2.
	// P1 in (a, u): the slab floor, the inner face under the facet, and the end at |u| = yellowTipEnd.
	e, k := yellowTipEnd, yellowTipInner-yellowTipEnd
	p1, err := r.polyBody(local(r3.Vec{}, axisA, axisZ), [][2]float64{{notchFloor, -(yellowTipInner - notchFloor)},
		{notchFloor, yellowTipInner - notchFloor}, {k, e}, {9.0, e}, {9.0, -e}, {k, -e}}, symmetric(21.0))
	if err != nil {
		return nil, nil, 0, wrap("P1", err)
	}
	// P2 in (b, u): the slab's width and the chamfer |b| + |u| <= yellowTipK, an octagon.
	w := yellowHalfWidth
	cu, cb := yellowTipK-yellowHalfWidth, yellowTipK-yellowTipEnd-0.5 // chamfer ends: (w, cu) and (cb, yellowTipK - cb)
	p2, err := r.polyBody(local(r3.Vec{}, axisB, axisZ), [][2]float64{{w, -cu}, {w, cu}, {cb, yellowTipK - cb},
		{-cb, yellowTipK - cb}, {-w, cu}, {-w, -cu}, {-cb, -(yellowTipK - cb)}, {cb, -(yellowTipK - cb)}}, symmetric(21.0))
	if err != nil {
		return nil, nil, 0, wrap("P2", err)
	}
	slab, err := decad.Intersect(r.ctx, p1, p2)
	if err != nil {
		return nil, nil, 0, wrap("slab", err)
	}
	rotorCopy, err := decad.Copy(r.ctx, rotor)
	if err != nil {
		return nil, nil, 0, wrap("copy rotor", err)
	}
	slabCopy, err := decad.Copy(r.ctx, slab)
	if err != nil {
		return nil, nil, 0, wrap("copy slab", err)
	}
	if yellow, err = decad.Intersect(r.ctx, rotorCopy, slabCopy); err != nil {
		return nil, nil, 0, wrap("yellow", err)
	}
	if black, err = decad.Cut(r.ctx, rotor, slab); err != nil {
		return nil, nil, 0, wrap("black", err)
	}
	return yellow, black, rv.Value.Base(), nil
}

// notchTool builds the cutter of one notch [u0, u1]: the two slanted profiles intersected, trimmed by the apex box.
func (r *run) notchTool(u0, u1 float64) (*decad.Body, error) {
	prof := [][2]float64{{notchFloor, u0 + notchEnd}, {notchFloor, u1 - notchEnd}, {notchFloor - notchEnd, u1}, {-9.0, u1},
		{-9.0, u0}, {notchFloor - notchEnd, u0}}
	sa, err := r.polyBody(local(r3.Vec{}, axisA, axisZ), prof, symmetric(21.0))
	if err != nil {
		return nil, fmt.Errorf("slant A: %w", err)
	}
	sb, err := r.polyBody(local(r3.Vec{}, axisB, axisZ), prof, symmetric(21.0))
	if err != nil {
		return nil, fmt.Errorf("slant B: %w", err)
	}
	tool, err := decad.Intersect(r.ctx, sa, sb)
	if err != nil {
		return nil, fmt.Errorf("slant A x B: %w", err)
	}
	ya := (2*notchFloor - notchApex) / math.Sqrt2
	box, err := r.polyBody(local(r3.Vec{Z: u0 - 1}, r3.Vec{X: 1}, r3.Vec{Y: 1}),
		[][2]float64{{-10.5, -10.5}, {10.5, -10.5}, {10.5, ya}, {-10.5, ya}},
		decad.Distance{D: units.Millimeters(u1 - u0 + 2), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("apex box: %w", err)
	}
	if tool, err = decad.Intersect(r.ctx, tool, box); err != nil {
		return nil, fmt.Errorf("apex box: %w", err)
	}
	if u0 > -rotorSquareHalf && u0+rotorSquareHalf < 0.8 { // the notch ends within the 0.8 print limit of the tip slant
		m, err := r.mouthTool(u0)
		if err != nil {
			return nil, err
		}
		if tool, err = decad.Union(r.ctx, tool, m); err != nil {
			return nil, fmt.Errorf("mouth: %w", err)
		}
	}
	return tool, nil
}

// mouthTool chamfers the edge where a notch's u0 end meets hidden face 315 (b = -3.75), leg notchMouth along u.
func (r *run) mouthTool(u0 float64) (*decad.Body, error) {
	k := 1 / math.Sqrt2
	kk := u0 - notchMouth + RotorHalfWidth // the chamfer plane u - b = kk
	aLo, top := -1.0, u0+1.0
	p1, err := r.polyBody(local(axisA.Scale(aLo), axisB, axisZ), [][2]float64{{-6.75, -6.75 + kk}, {-6.75, top}, {top - kk, top}},
		decad.Distance{D: units.Millimeters(notchFloor - aLo), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("mouth prism: %w", err)
	}
	tdir := axisB.Scale(-k).Add(axisZ.Scale(k))
	o := axisB.Scale(-RotorHalfWidth).Add(axisZ.Scale(-RotorHalfWidth + kk))
	af := notchFloor - notchEnd
	p3, err := r.polyBody(local(o, axisA, tdir), [][2]float64{{aLo - 1, -1}, {af - 1, -1}, {af + 2, 2}, {aLo - 1, 2}},
		symmetric(45.0))
	if err != nil {
		return nil, fmt.Errorf("mouth taper: %w", err)
	}
	return decad.Intersect(r.ctx, p1, p3)
}

// pocketTool builds a pin pocket's cutter: a stadium along local x from slotStart to slotEnd, radius slotRadius, from local
// z z0 for the given length, its floor rounded 0.5.
func (r *run) pocketTool(z0, length float64) (*decad.Body, error) {
	s, err := r.sketchOn(local(r3.Vec{Z: z0}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	if err != nil {
		return nil, err
	}
	c0, c1 := sketchcheck.DimensionPoint(s, slotStart, 0), sketchcheck.DimensionPoint(s, slotEnd, 0)
	a1, a2 := s.CreatePoint(slotStart, slotRadius), s.CreatePoint(slotStart, -slotRadius)
	b1, b2 := s.CreatePoint(slotEnd, -slotRadius), s.CreatePoint(slotEnd, slotRadius)
	arc0, arc1 := s.CreateArc(c0, a1, a2), s.CreateArc(c1, b1, b2)
	l1, l2 := s.CreateLine(a2, b1), s.CreateLine(b2, a1)
	s.AddConstraint(sketch.NewRadius(arc0, slotRadius), sketch.NewEqualRadius(arc0, arc1), sketch.NewTangent(l1, arc0),
		sketch.NewTangent(l1, arc1), sketch.NewTangent(l2, arc0), sketch.NewTangent(l2, arc1))
	b, err := r.extrude(s, 0, length, decad.Along)
	if err != nil {
		return nil, err
	}
	convex, err := decad.Edges(decad.Convex()).SelectEdges(b)
	if err != nil {
		return nil, fmt.Errorf("select pocket floor: %w", err)
	}
	var sel *decad.EdgeQuery
	n := 0
	for _, e := range convex {
		a, c := e.Start().Position().Value, e.End().Position().Value
		if math.Abs(a.Z-(z0+length)) > 1e-6 || math.Abs(c.Z-(z0+length)) > 1e-6 {
			continue
		}
		var preds []decad.EdgePredicate
		switch e.Curve().(type) {
		case decad.Circle3, decad.Arc3:
			preds = []decad.EdgePredicate{decad.EndpointAt(a), decad.Circular()}
		default:
			preds = []decad.EdgePredicate{decad.EndpointAt(a), decad.ParallelTo(c.Sub(a))}
		}
		if sel == nil {
			sel = decad.Edges(preds...)
		} else {
			sel = sel.Or(preds...)
		}
		n++
	}
	if n == 0 {
		return nil, fmt.Errorf("no pocket floor edge")
	}
	if b, err = decad.Fillet(r.ctx, b, sel.Exactly(n), units.Millimeters(pocketFloorR)); err != nil {
		return nil, fmt.Errorf("round the pocket floor (%d edges): %w", n, err)
	}
	return b, nil
}

// flagBodies builds one rotor in the local frame at turn 0: the yellow slab and the black body.
func (r *run) flagBodies(s Segment) (Flag, error) {
	yellow, black, _, err := r.rotorSplit()
	if err != nil {
		return Flag{}, err
	}
	wrap := func(what string, e error) error { return fmt.Errorf("rotor %s: %w", what, e) }
	// 4. The flat under the bottom: black only.
	flat, err := r.polyBody(local(r3.Vec{Z: -21.0}, r3.Vec{X: 1}, r3.Vec{Y: 1}),
		[][2]float64{{-10.5, -10.5}, {10.5, -10.5}, {10.5, rotorFlat}, {-10.5, rotorFlat}},
		decad.Distance{D: units.Millimeters(42.0), Dir: decad.Along})
	if err != nil {
		return Flag{}, wrap("flat tool", err)
	}
	if black, err = decad.Cut(r.ctx, black, flat); err != nil {
		return Flag{}, wrap("cut flat", err)
	}
	// 5 and 6. Each notch, then its pin pocket.
	for i, n := range rotorNotches(s) {
		tool, err := r.notchTool(n[0], n[1])
		if err != nil {
			return Flag{}, wrap(fmt.Sprintf("notch %d tool", i+1), err)
		}
		if black, err = decad.Cut(r.ctx, black, tool); err != nil {
			return Flag{}, wrap(fmt.Sprintf("cut notch %d", i+1), err)
		}
		pocket, err := r.pocketTool(n[1]-0.9, n[2]-n[1]+0.9)
		if err != nil {
			return Flag{}, wrap(fmt.Sprintf("pocket %d tool", i+1), err)
		}
		if black, err = decad.Cut(r.ctx, black, pocket); err != nil {
			return Flag{}, wrap(fmt.Sprintf("cut pocket %d", i+1), err)
		}
	}
	// 7. Each end's pivot bore: the revolved bore with its floor round and 45 degree mouth, then the teardrop roof.
	for _, sgn := range []float64{1, -1} {
		bore, err := r.pivotBoreTool(sgn)
		if err != nil {
			return Flag{}, wrap("pivot bore tool", err)
		}
		if black, err = decad.Cut(r.ctx, black, bore); err != nil {
			return Flag{}, wrap("cut pivot bore", err)
		}
		roof, err := r.pivotRoofTool(sgn)
		if err != nil {
			return Flag{}, wrap("pivot roof tool", err)
		}
		if black, err = decad.Cut(r.ctx, black, roof); err != nil {
			return Flag{}, wrap("cut pivot roof", err)
		}
	}
	return Flag{Yellow: yellow, Black: black}, nil
}

// flag builds segment s's rotor in the world at turn 0, unplaced by the model's Placement.
func (r *run) flag(s Segment) (Flag, error) {
	f, err := r.flagBodies(s)
	if err != nil {
		return Flag{}, err
	}
	frame := rotorFrame(s)
	// Place both bodies in two steps: translate by the origin's coordinates in the local frame, then rotate about the world
	// origin. The result is the single placement's rigid motion; a single placement works too (decad item D3).
	o := frame.Origin()
	shift, err := r3.Translation(r3.Vec{X: o.Dot(frame.U()), Y: o.Dot(frame.V()), Z: o.Dot(frame.N())})
	if err != nil {
		return Flag{}, fmt.Errorf("rotor %s: shift: %w", s, err)
	}
	turn, err := r3.NewFrame(r3.Vec{}, frame.U(), frame.V())
	if err != nil {
		return Flag{}, fmt.Errorf("rotor %s: turn frame: %w", s, err)
	}
	for _, b := range []struct {
		name string
		dst  **decad.Body
	}{{"yellow", &f.Yellow}, {"black", &f.Black}} {
		if *b.dst, err = decad.Transform(r.ctx, *b.dst, shift); err != nil {
			return Flag{}, fmt.Errorf("rotor %s: shift %s: %w", s, b.name, err)
		}
		if *b.dst, err = placement.Place(r.ctx, *b.dst, &turn); err != nil {
			return Flag{}, fmt.Errorf("rotor %s: place %s: %w", s, b.name, err)
		}
	}
	return f, nil
}

// fromLocal returns the world point of the rotor-local point (x, y, u) of segment s.
func (Flag) fromLocal(s Segment, x, y, u float64) r3.Vec {
	f := rotorFrame(s)
	return f.Origin().Add(f.U().Scale(x)).Add(f.V().Scale(y)).Add(f.N().Scale(u))
}
