package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// The separate groove pin of the a, g and d sliders, and the keyed hole it slides into, in mm. These sliders print on
// their -X face, so an integral pin would lie sideways with a 3.2 mm wide overhang under it. The pin is one body in the
// model; the export cuts it into two identical halves on the plane x = 0 (normal +X), and each half prints flat on that
// face. The hole is a circle with 45 degree V roofs, closed on every side but the stem's +Y face.
const (
	groovePinClearance = 0.20 // radial clearance of the pin's round in the hole (Tight)
	groovePinShank     = 4.0  // the pin's length inside the stem
	groovePinFloorGap  = 0.20 // the gap from the pin's root end to the hole's floor
	groovePinChamfer   = 0.3  // the working part's tip chamfer
	groovePinRootCh    = 0.55 // the root's 45 degree chamfer: the floor round 0.75 less the floor gap
	groovePinWingIn    = 0.10 // the wing flank is pulled in this far from tangent to the round
	groovePinTaper     = 30.0 // the wings end in a cone of this half angle, degrees
	groovePinFloorR    = 0.75 // the round on the hole floor's edges
	groovePinHoleRoof  = 1.0  // how far the hole's cutter stands out past the stem's face
	// groovePinRaise is how far the slider settles below its groove pin, less the 0.05 that RaiseBelowDroop lowers the unlit
	// pose by: the hole's centre sits this far below the pin's axis on the body, and the body is placed this much higher.
	// The sag is 0.2 x (1 + 2 x 0.55 / 3.45) = 0.264 once the pin bears on the hole at the groove's mouth and at its root.
	groovePinRaise = 0.214
)

// groovePin builds the keyed groove pin as one body (the split is not modeled): a round of PinRadius from its root to its
// tip with a 0.3 tip chamfer, and inside the stem two wings, flats at x = +-PinRadius with 45 degree flanks, that end in a
// 30 degree cone 0.1 inside the stem face; a 45 degree chamfer trims the root. The axis runs along +Y at x = 0, z = zc, and
// yFace is the stem's +Y face. The seam plane has normal +X: it holds the pin's axis and the main load direction Z.
func (r *run) groovePin(yFace, zc float64) (*decad.Body, error) {
	yRoot := yFace - groovePinShank
	length := groovePinShank + PinLength
	frame, err := r3.NewFrame(r3.Vec{Y: yRoot, Z: zc}, r3.Vec{Y: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	axis := decad.ConstructionAxis{Origin: frame.Origin(), Dir: r3.Vec{Y: 1}}
	revolve := func(v [][2]float64) (*decad.Body, error) {
		s, err := r.sketchOn(frame)
		if err != nil {
			return nil, err
		}
		polygon(s, v)
		pr, err := r.profile(s, 0)
		if err != nil {
			return nil, err
		}
		return decad.Revolve(r.ctx, r.doc, s, pr, axis, decad.FullRevolution{})
	}
	rad := PinRadius
	round, err := revolve([][2]float64{{0, 0}, {0, rad - groovePinRootCh}, {groovePinRootCh, rad}, {length - groovePinChamfer, rad},
		{length, rad - groovePinChamfer}, {length, 0}})
	if err != nil {
		return nil, fmt.Errorf("pin round: %w", err)
	}
	// The wing prism: an octagon in (x, z) with flats at |x| = rad and flanks |x| + |z - zc| = k, closed inside the round.
	k := rad*math.Sqrt2 - groovePinWingIn
	zf := k - rad
	const xi = 1.5
	zi := k - xi
	f, d := slFrame(false, yRoot)
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	polygon(s, [][2]float64{{rad, zc - zf}, {rad, zc + zf}, {xi, zc + zi}, {-xi, zc + zi}, {-rad, zc + zf}, {-rad, zc - zf},
		{-xi, zc - zi}, {xi, zc - zi}})
	wings, err := r.extrude(s, 0, groovePinShank, d)
	if err != nil {
		return nil, fmt.Errorf("pin wings: %w", err)
	}
	// The envelope: r 2.6 over the shank, the 45 degree root cone, a 30 degree cone that crosses the round 0.1 inside the
	// face, then r 2.2 past it.
	tn := math.Tan(groovePinTaper * math.Pi / 180)
	sx := groovePinShank - 0.1
	rootR := rad - groovePinRootCh - 0.1
	env, err := revolve([][2]float64{{0, 0}, {0, rootR}, {2.6 - rootR, 2.6}, {sx - (2.6-rad)/tn, 2.6}, {sx + 0.05/tn, rad - 0.05},
		{groovePinShank + 0.05, rad - 0.05}, {groovePinShank + 0.05, 0}})
	if err != nil {
		return nil, fmt.Errorf("pin wing envelope: %w", err)
	}
	if wings, err = decad.Intersect(r.ctx, wings, env); err != nil {
		return nil, fmt.Errorf("pin wing taper: %w", err)
	}
	pin, err := decad.Union(r.ctx, round, wings)
	if err != nil {
		return nil, fmt.Errorf("pin round and wings: %w", err)
	}
	return pin, nil
}

// groovePinHole cuts the blind keyed hole into body: a circle 0.2 larger in radius than the pin with 45 degree V faces
// tangent to it, meeting in sharp apexes at x = +-r sqrt 2 (in print the +X apex is the roof and the -X apex the floor), from
// the floor 4.2 inside the stem's +Y face yFace out past it, the floor edges rounded 0.75. Its axis is at x = 0, z = zc.
func (r *run) groovePinHole(body *decad.Body, yFace, zc float64) (*decad.Body, error) {
	rh := PinRadius + groovePinClearance
	depth := groovePinShank + groovePinFloorGap
	yFloor := yFace - depth
	f, d := slFrame(false, yFloor)
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	c := sketchcheck.DimensionPoint(s, 0, zc)
	q := rh / math.Sqrt2
	p1, p2 := s.CreatePoint(q, zc+q), s.CreatePoint(-q, zc+q)
	p3, p4 := s.CreatePoint(-q, zc-q), s.CreatePoint(q, zc-q)
	e, w := s.CreatePoint(rh*math.Sqrt2, zc), s.CreatePoint(-rh*math.Sqrt2, zc)
	top := s.CreateArc(c, p1, p2)
	bot := s.CreateArc(c, p3, p4)
	l1, l2 := s.CreateLine(p4, e), s.CreateLine(e, p1)
	l3, l4 := s.CreateLine(p2, w), s.CreateLine(w, p3)
	ref := newPolar(s).ref
	s.AddConstraint(sketch.NewRadius(top, rh), sketch.NewEqualRadius(top, bot),
		sketch.NewTangent(l1, bot), sketch.NewTangent(l2, top), sketch.NewTangent(l3, top), sketch.NewTangent(l4, bot),
		sketch.NewAngle(ref, l1, 45), sketch.NewAngle(ref, l2, 135), sketch.NewAngle(ref, l3, -135), sketch.NewAngle(ref, l4, -45))
	tool, err := r.extrude(s, 0, depth+groovePinHoleRoof, d)
	if err != nil {
		return nil, fmt.Errorf("hole prism: %w", err)
	}
	// Round every convex edge of the floor loop, as the rotor's pocket does: a 45 degree cone floor reads 59 to 70 degree
	// turns in the print sections, a 0.5 round 52 degrees, and 0.75 none past 45.
	convex, err := decad.Edges(decad.Convex()).SelectEdges(tool)
	if err != nil {
		return nil, fmt.Errorf("hole floor edges: %w", err)
	}
	var sel *decad.EdgeQuery
	n := 0
	for _, e := range convex {
		a, b := e.Start().Position().Value, e.End().Position().Value
		if math.Abs(a.Y-yFloor) > 1e-6 || math.Abs(b.Y-yFloor) > 1e-6 {
			continue
		}
		var preds []decad.EdgePredicate
		switch e.Curve().(type) {
		case decad.Circle3, decad.Arc3:
			preds = []decad.EdgePredicate{decad.EndpointAt(a), decad.Circular()}
		default:
			preds = []decad.EdgePredicate{decad.EndpointAt(a), decad.ParallelTo(b.Sub(a))}
		}
		if sel == nil {
			sel = decad.Edges(preds...)
		} else {
			sel = sel.Or(preds...)
		}
		n++
	}
	if n == 0 {
		return nil, fmt.Errorf("hole floor edges: none found")
	}
	if tool, err = decad.Fillet(r.ctx, tool, sel.Exactly(n), units.Millimeters(groovePinFloorR)); err != nil {
		return nil, fmt.Errorf("round the hole floor (%d edges): %w", n, err)
	}
	if body, err = decad.Cut(r.ctx, body, tool); err != nil {
		return nil, fmt.Errorf("cut the groove pin hole: %w", err)
	}
	return body, nil
}

// groovePinFor builds segment seg's groove pin at its modeled pose: on the groove's centreline at x = 0, z = sliderPinZ,
// from the stem's +Y face yFace. Only a, g and d have one; b to f keep the integral pin.
func (r *run) groovePinFor(seg Segment) (*decad.Body, error) {
	return r.groovePin(r.dm.Station(seg)+StemThickness/2, sliderPinZ)
}
