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

// Groove pin and stem dimensions that are not exported, in mm.
const (
	sliderStemFrom = 42.5 // z where the stem starts
	sliderPinZ     = 46.0 // z of the groove pin's axis at the unlit pose
	sliderPinR     = 1.5
	sliderPinRound = 0.5
	sliderPinChamf = 0.3
)

// segmentCentre returns the segment's centre in the XY plane and whether its long axis is X.
func segmentCentre(s Segment) (x, y float64, horizontal bool) {
	switch s {
	case SegA:
		return 0, SegmentSpan, true
	case SegG:
		return 0, 0, true
	case SegD:
		return 0, -SegmentSpan, true
	case SegB:
		return SegmentSpan / 2, SegmentSpan / 2, false
	case SegC:
		return SegmentSpan / 2, -SegmentSpan / 2, false
	case SegE:
		return -SegmentSpan / 2, -SegmentSpan / 2, false
	default:
		return -SegmentSpan / 2, SegmentSpan / 2, false
	}
}

// xyFrame is the horizontal plane z = z0 of the world with U = +X and V = +Y.
func xyFrame(z0 float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Z: z0}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	return f
}

// pin revolves the pin about the line x = 0, z = 46 along +Y from the stem's +Y face at y0.
func (r *run) pin(y0 float64) (*decad.Body, error) {
	f, err := r3.NewFrame(r3.Vec{Y: y0, Z: sliderPinZ}, r3.Vec{Y: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	v := [][2]float64{{-0.5, 0}, {-0.5, 2.0}} // (s, r)
	for _, q := range quarterRound(0.5, 2.0, sliderPinRound, 180, 270, 8) {
		v = append(v, [2]float64{q[0], q[1]})
	}
	v = append(v, [2]float64{PinLength - sliderPinChamf, sliderPinR}, [2]float64{PinLength, sliderPinR - sliderPinChamf},
		[2]float64{PinLength, 0})
	polygon(s, v)
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(r.ctx, r.doc, s, pr, decad.ConstructionAxis{Origin: f.Origin(), Dir: r3.Vec{Y: 1}},
		decad.FullRevolution{})
}

// Flag slider dimensions that are not exported, in mm.
const (
	sliderHeadTop   = 72.8 // head or arm top at the build pose (unlit)
	tongueRootDip   = 0.03 // how far the tongue's root rounds bottom out inside the head
	tongueHullR     = 0.6  // radius of the hull round about the pin axis
	flagPinRadius   = 0.50 // the rotor pin, Ø1.0
	flagPinLength   = 3.6
	bladeRidgeZ     = 83.55 // top of the blade's gable
	bladeShoulderZ  = 82.95
	bladeExtOverlap = 0.1
	stemChamfer     = 0.3
	tongueEndGap    = 1.3   // the end cutter's lower line lies this far below the head top
	pinStationAGD   = 1.5   // the rotor pin's distance from the stem along s, a, g, d
	pinStationBC    = 15.0  // along x, b and c
	pinStationEF    = 18.0  // along x, e and f
	tongueWidth     = 1.4   // the tongue strip's width along s
	tongueTopGap    = 1.2   // the tongue's top edge lies this far below the pin axis
	pinAxisZ        = 80.85 // the rotor pin's axis at the unlit build pose of a, g and d
)

// slFrame returns the sketch frame and extrusion direction for a profile drawn in (w, z): w is world Y with the
// profile on x = at extruded along +X (wIsY), or world X on y = at extruded along +Y (against the normal -Y).
func slFrame(wIsY bool, at float64) (r3.Frame, decad.Direction) {
	if wIsY {
		f, _ := r3.NewFrame(r3.Vec{X: at}, r3.Vec{Y: 1}, r3.Vec{Z: 1}) // normal +X
		return f, decad.Along
	}
	f, _ := r3.NewFrame(r3.Vec{Y: at}, r3.Vec{X: 1}, r3.Vec{Z: 1}) // normal -Y
	return f, decad.Against
}

// polyLoop draws a closed polygon whose sides are held by direction and distance from the origin (see polygon).
func polyLoop(s *sketch.Sketch, v [][2]float64) {
	polygon(s, v)
}

// tangentPoint returns the point where a line from p touches the circle (c, rad); side +1 or -1 picks the tangent.
func tangentPoint(p, c [2]float64, rad, side float64) [2]float64 {
	dx, dy := p[0]-c[0], p[1]-c[1]
	d := math.Hypot(dx, dy)
	a := math.Atan2(dy, dx) + side*math.Acos(rad/d)
	return [2]float64{c[0] + rad*math.Cos(a), c[1] + rad*math.Sin(a)}
}

// tongueOutline draws the tongue's hull in (s, z): feet at the head, 8-chord root rounds of r 0.7 that bottom out inside
// the head, the strip sides up to the pin, and two hull lines tangent to the circle of radius 0.6 about the pin axis.
func tongueOutline(s *sketch.Sketch, sL, sR, sc, zp float64) {
	dp := func(x, y float64) *sketch.Point { return sketchcheck.DimensionPoint(s, x, y) }
	zt := zp - tongueTopGap
	zh := sliderHeadTop
	zc := zh + 0.7 - tongueRootDip
	var right, left []*sketch.Point
	right = append(right, dp(sR+0.8, zh-0.8), dp(sR+0.8, zc-0.7))
	for _, q := range quarterRound(sR+0.7, zc, 0.7, 270, 180, 8) {
		right = append(right, dp(q[0], q[1]))
	}
	right = append(right, dp(sR, zt))
	for _, q := range quarterRound(sL-0.7, zc, 0.7, 0, -90, 8) {
		left = append(left, dp(q[0], q[1]))
	}
	left = append([]*sketch.Point{dp(sL, zt)}, left...)
	left = append(left, dp(sL-0.8, zc-0.7), dp(sL-0.8, zh-0.8))
	c := dp(sc, zp)
	tr1 := tangentPoint([2]float64{sR, zt}, [2]float64{sc, zp}, tongueHullR, 1)
	tr2 := tangentPoint([2]float64{sR, zt}, [2]float64{sc, zp}, tongueHullR, -1)
	tR := tr1
	if tr2[0] > tr1[0] {
		tR = tr2
	}
	tl1 := tangentPoint([2]float64{sL, zt}, [2]float64{sc, zp}, tongueHullR, 1)
	tl2 := tangentPoint([2]float64{sL, zt}, [2]float64{sc, zp}, tongueHullR, -1)
	tL := tl1
	if tl2[0] < tl1[0] {
		tL = tl2
	}
	pR, pL := s.CreatePoint(tR[0], tR[1]), s.CreatePoint(tL[0], tL[1])
	for i := 0; i+1 < len(right); i++ {
		s.CreateLine(right[i], right[i+1])
	}
	lR := s.CreateLine(right[len(right)-1], pR)
	arc := s.CreateArc(c, pR, pL)
	lL := s.CreateLine(pL, left[0])
	for i := 0; i+1 < len(left); i++ {
		s.CreateLine(left[i], left[i+1])
	}
	s.CreateLine(left[len(left)-1], right[0])
	s.AddConstraint(sketch.NewRadius(arc, tongueHullR), sketch.NewTangent(lR, arc), sketch.NewTangent(lL, arc))
}

// rootedTongue builds the tongue and its pin. u is the rotor axis direction (world X when uIsX, else world Y); the tongue runs
// along +u from ua, the print face, to ue, the pin face; its strip spans [sL, sR] in the other horizontal axis; the pin
// axis is at (sc, zp). The result has r 0.7 roots on three sides, a sharp pin root (decad item D2) and a Ø1.0 pin 3.6
// long with a 0.2 tip chamfer.
func (r *run) rootedTongue(uIsX bool, ua, ue, sL, sR, sc, zp float64) (*decad.Body, error) {
	// 1. The hull prism: the outline in (s, z), extruded along u from ua to ue + 3.8.
	f1, d1 := slFrame(uIsX, ua)
	s1, err := r.sketchOn(f1)
	if err != nil {
		return nil, err
	}
	tongueOutline(s1, sL, sR, sc, zp)
	p1, err := r.extrude(s1, 0, ue+3.8-ua, d1)
	if err != nil {
		return nil, fmt.Errorf("tongue hull prism: %w", err)
	}
	// 2. The end cutter in (u, z), extruded along s over [sL - 1.5, sR + 1.5].
	var f2 r3.Frame
	var d2 decad.Direction
	if uIsX {
		f2, _ = r3.NewFrame(r3.Vec{Y: sL - 1.5}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		d2 = decad.Against
	} else {
		f2, _ = r3.NewFrame(r3.Vec{X: sL - 1.5}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		d2 = decad.Along
	}
	ztop := zp + 1.5
	zcE := sliderHeadTop + 0.7 - tongueRootDip
	cv := [][2]float64{{ue, ztop}}
	cv = append(cv, quarterRound(ue+0.7, zcE, 0.7, 180, 270, 8)...)
	cv = append(cv, [2]float64{ue + 0.8, zcE - 0.7}, [2]float64{ue + 0.8, sliderHeadTop - tongueEndGap},
		[2]float64{ue + 4.0, sliderHeadTop - tongueEndGap}, [2]float64{ue + 4.0, ztop})
	s2, err := r.sketchOn(f2)
	if err != nil {
		return nil, err
	}
	polyLoop(s2, cv)
	cut, err := r.extrude(s2, 0, sR-sL+3.0, d2)
	if err != nil {
		return nil, fmt.Errorf("tongue end cutter: %w", err)
	}
	// 3. The pin: a circle of r 0.50 on the plane u = ue - 0.1 extruded 3.7, its far edge chamfered 0.2.
	var f r3.Frame
	if uIsX {
		f, _ = r3.NewFrame(r3.Vec{X: ue - 0.1}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	} else {
		f, _ = r3.NewFrame(r3.Vec{Y: ue - 0.1}, r3.Vec{Z: 1}, r3.Vec{X: 1}) // U = Z, V = X: normal +Y
	}
	cs, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	var pc *sketch.Point
	if uIsX {
		pc = sketchcheck.DimensionPoint(cs, sc, zp)
	} else {
		pc = sketchcheck.DimensionPoint(cs, zp, sc)
	}
	cs.AddConstraint(sketch.NewRadius(cs.CreateCircle(pc, flagPinRadius), flagPinRadius))
	pin, err := r.extrude(cs, 0, 3.7, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("rotor pin: %w", err)
	}
	var tipAt r3.Vec
	found := false
	circles, err := decad.Edges(decad.Circular()).SelectEdges(pin)
	if err != nil {
		return nil, fmt.Errorf("rotor pin edges: %w", err)
	}
	for _, e := range circles {
		a := e.Start().Position().Value
		uu := a.X
		if !uIsX {
			uu = a.Y
		}
		if math.Abs(uu-(ue+flagPinLength)) < 1e-6 {
			tipAt, found = a, true
		}
	}
	if !found {
		return nil, fmt.Errorf("rotor pin: no tip edge at u = %g", ue+flagPinLength)
	}
	if pin, err = decad.Chamfer(r.ctx, pin, decad.Edges(decad.Circular(), decad.EndpointAt(tipAt)).Exactly(1),
		units.Millimeters(0.2)); err != nil {
		return nil, fmt.Errorf("rotor pin tip chamfer: %w", err)
	}
	// 4. cutter = cutter - pin; tongue = hull - cutter.
	if cut, err = decad.Cut(r.ctx, cut, pin); err != nil {
		return nil, fmt.Errorf("tongue cutter minus pin: %w", err)
	}
	tongue, err := decad.Cut(r.ctx, p1, cut)
	if err != nil {
		return nil, fmt.Errorf("tongue end cut: %w", err)
	}
	return tongue, nil
}

// stemVerticals chamfers 0.3 the stem's four convex vertical edges (the ones that start at z 42.5).
func (r *run) stemVerticals(body *decad.Body) (*decad.Body, error) {
	edges, err := decad.Edges(decad.Convex(), decad.ParallelTo(r3.Vec{Z: 1})).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select stem edges: %w", err)
	}
	var q *decad.EdgeQuery
	n := 0
	for _, e := range edges {
		a, b := e.Start().Position().Value, e.End().Position().Value
		if math.Abs(math.Min(a.Z, b.Z)-sliderStemFrom) > 1e-6 {
			continue
		}
		p := []decad.EdgePredicate{decad.Convex(), decad.EndpointAt(a), decad.ParallelTo(r3.Vec{Z: 1})}
		if q == nil {
			q = decad.Edges(p...)
		} else {
			q = q.Or(p...)
		}
		n++
	}
	if n != 4 {
		return nil, fmt.Errorf("stem has %d convex vertical edges at the foot, want 4", n)
	}
	return decad.Chamfer(r.ctx, body, q.Exactly(n), units.Millimeters(stemChamfer))
}

// bfProfile draws the b-f slider's main profile (stem, arm, head and blade) in (x, z) on y = yi - 1.8 and extrudes it 3.6. The
// arm points to +x for b and c and to -x for e and f.
func (r *run) bfProfile(ef bool, yi float64) (*decad.Body, float64, float64, error) {
	f, d := slFrame(false, yi-1.8)
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, 0, 0, err
	}
	bL, bR := 8.7, 9.8 // the blade is 1.1 thick: its rotor-side face stands 0.1 back from the old 9.9
	zh, za := sliderHeadTop, sliderHeadTop-2.5
	var v [][2]float64
	if !ef {
		v = [][2]float64{{-3.5, 42.5}, {3.5, 42.5}, {3.5, za}, {17.9, za}, {17.9, zh},
			{bR, zh}, {bR, bladeShoulderZ}, {(bL + bR) / 2, bladeRidgeZ}, {bL, bladeShoulderZ}, {bL, zh}, {-3.5, zh}}
	} else {
		bL, bR = -9.8, -8.7
		v = [][2]float64{{-3.5, 42.5}, {3.5, 42.5}, {3.5, zh}, {bR, zh}, {bR, bladeShoulderZ}, {(bL + bR) / 2, bladeRidgeZ},
			{bL, bladeShoulderZ}, {bL, zh}, {-19.5, zh}, {-19.5, za}, {-3.5, za}}
	}
	ps, ls := polygon(s, v)
	n := len(v)
	type round struct {
		i   int
		rad float64
	}
	rounds := []round{{2, 1.25}, {5, 0.6}, {9, 0.6}}
	if ef {
		rounds = []round{{3, 0.6}, {7, 0.6}, {10, 1.25}}
	}
	for _, q := range rounds {
		prev := ls[(q.i-1+n)%n]
		if _, err := sketch.CreateFillet(s, prev, ls[q.i], ps[q.i], q.rad); err != nil {
			return nil, 0, 0, fmt.Errorf("profile round %d: %w", q.i, err)
		}
	}
	body, err := r.extrude(s, 0, 3.6, d)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("stem, arm and blade: %w", err)
	}
	return body, bL, bR, nil
}

// bfSlider builds a b, c, e or f slider at the unlit build pose.
func (r *run) bfSlider(seg Segment) (*decad.Body, error) {
	ef := seg == SegE || seg == SegF
	yi := r.dm.Station(seg)
	body, bL, bR, err := r.bfProfile(ef, yi)
	if err != nil {
		return nil, err
	}
	zh := sliderHeadTop
	// The blade extension beyond the arm: a polygon in (x, z) on y = yi + 1.7, extruded +Y 1.3.
	ef2, ed2 := slFrame(false, yi+1.8-bladeExtOverlap)
	es, err := r.sketchOn(ef2)
	if err != nil {
		return nil, err
	}
	polyLoop(es, [][2]float64{{bL, zh}, {bR, zh}, {bR, bladeShoulderZ}, {(bL + bR) / 2, bladeRidgeZ}, {bL, bladeShoulderZ}})
	ext, err := r.extrude(es, 0, 1.2+bladeExtOverlap, ed2)
	if err != nil {
		return nil, fmt.Errorf("blade extension: %w", err)
	}
	if body, err = decad.Union(r.ctx, body, ext); err != nil {
		return nil, fmt.Errorf("union blade extension: %w", err)
	}
	// The tongue and pin.
	sL, sR, sc, zp := pinStationBC, pinStationBC+tongueWidth, pinStationBC, pinAxisZ+raise(seg)
	if ef {
		sL, sR, sc = -pinStationEF, -pinStationEF+tongueWidth, -pinStationEF
	}
	tg, err := r.rootedTongue(false, yi-1.8, yi-0.2, sL, sR, sc, zp)
	if err != nil {
		return nil, err
	}
	if body, err = decad.Union(r.ctx, body, tg); err != nil {
		return nil, fmt.Errorf("union tongue: %w", err)
	}
	return r.finishSlider(body, seg, yi)
}

// agdSlider builds an a, g or d slider at the unlit build pose.
func (r *run) agdSlider(seg Segment) (*decad.Body, error) {
	yi := r.dm.Station(seg)
	// The sketch's origin sits at this slider's station, so every coordinate is relative to it.
	_, d := slFrame(true, -3.5)
	f, err := r3.NewFrame(r3.Vec{X: -3.5, Y: yi}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	zh := sliderHeadTop
	v := [][2]float64{{-1.8, 42.5}, {1.8, 42.5}, {1.8, zh - 1.65}, {2.4, zh - 1.05}, {2.4, zh},
		{-6.7, zh}, {-6.7, bladeShoulderZ}, {-7.25, bladeRidgeZ}, {-7.8, bladeShoulderZ},
		{-7.8, zh}, {-8.6, zh}, {-8.6, zh - 2.0}, {-1.8, zh - 2.0}}
	ps, ls := polygon(s, v)
	for _, c := range []struct {
		a, b int
		rad  float64
	}{{4, 5, 0.6}, {8, 9, 0.6}, {11, 12, 1.0}} {
		if _, err := sketch.CreateFillet(s, ls[c.a], ls[c.b], ps[c.b], c.rad); err != nil {
			return nil, fmt.Errorf("profile round %d: %w", c.b, err)
		}
	}
	body, err := r.extrude(s, 0, 7.0, d)
	if err != nil {
		return nil, fmt.Errorf("main profile: %w", err)
	}
	tg, err := r.rootedTongue(true, -3.5, -1.9, yi+pinStationAGD-tongueWidth, yi+pinStationAGD, yi+pinStationAGD, pinAxisZ)
	if err != nil {
		return nil, err
	}
	if body, err = decad.Union(r.ctx, body, tg); err != nil {
		return nil, fmt.Errorf("union tongue: %w", err)
	}
	return r.finishSlider(body, seg, yi)
}

// finishSlider chamfers the stem's vertical edges and adds the groove pin.
func (r *run) finishSlider(body *decad.Body, seg Segment, yi float64) (*decad.Body, error) {
	body, err := r.stemVerticals(body)
	if err != nil {
		return nil, fmt.Errorf("stem vertical chamfers: %w", err)
	}
	gp, err := r.pin(yi + 1.8)
	if err != nil {
		return nil, fmt.Errorf("groove pin: %w", err)
	}
	if body, err = decad.Union(r.ctx, body, gp); err != nil {
		return nil, fmt.Errorf("union groove pin: %w", err)
	}
	return body, nil
}

// flagSlider builds segment s's slider for the flag mechanism at the unlit build pose: a/g/d, b/c or e/f, one of three types.
func (r *run) flagSlider(s Segment) (*decad.Body, error) {
	if _, _, horizontal := segmentCentre(s); horizontal {
		return r.agdSlider(s)
	}
	return r.bfSlider(s)
}
