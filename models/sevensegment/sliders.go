package sevensegment

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Groove pin and stem dimensions that are not exported, in mm.
const (
	sliderStemFrom = sliderPinZ - 5.25  // z where the stem starts
	sliderPinZ     = ShaftZ + OffRadius // z of the groove pin's axis at the unlit pose: on the groove's off radius
	sliderPinR     = PinRadius
	sliderPinRound = 0.5
	sliderPinChamf = 0.3
	sliderHalfWide = 5.25 // half the stem's width along the rotor axis of a, g and d, and along x of b to f
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
	v := [][2]float64{{-sliderPinRound, 0}, {-sliderPinRound, sliderPinR + sliderPinRound}} // (s, r)
	for _, q := range quarterRound(sliderPinRound, sliderPinR+sliderPinRound, sliderPinRound, 180, 270, 8) {
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
	sliderHeadTop   = pinAxisZ - 12.075      // head or arm top at the build pose (unlit)
	tongueRootDip   = 0.03                   // how far the tongue's root rounds bottom out inside the head
	bladeRidgeZ     = sliderHeadTop + 16.125 // top of the blade's gable
	bladeShoulderZ  = sliderHeadTop + 15.225 //
	bladeExtOverlap = 0.1                    //
	stemChamfer     = wallChamfer            //
	tongueEndGap    = 1.95                   // the end cutter's lower line lies this far below the head top
	pinStationAGD   = 2.25                   // the drive axis's distance from the stem along s, a, g, d
	pinStationBC    = 22.5                   // along x, b and c
	pinStationEF    = 27.0                   // along x, e and f
	tongueWidth     = 2.1                    // the tongue strip's width along s
	pinAxisZ        = RotorAxisZ - 2.625     // the drive axis at the unlit build pose of a, g and d
)

// The drive cone's dimensions, in mm. The cone is the rule "Thin pins" form: a base 4.2 across on the tongue's end disc, and
// a section that thins to a small tip. Its base lies off the drive axis, toward the rotor axis and down, where the notch has
// room. The cone itself stands on the drive axis the whole way down: a buttress, the hull of the junction circle and the
// end disc, joins them, so its front is a vertical wall and its back a slope, and no face points down.
const (
	coneDiscR     = 2.10  // the end disc that carries the base
	coneTau       = 0.97  // the buttress's top section is the hull of the junction circle and this fraction of the way to the disc
	coneOffset    = 1.90  // the base centre's distance from the drive axis
	coneOffsetDeg = 235.0 // its direction in the sweep frame: 180 is toward the rotor axis, 270 down
	coneDiscRound = 0.5   // the inside rounds where the disc meets the strip's sides
	coneHalfAngle = 8.0   // the right cone's half angle, degrees
	conePlay      = 1.0   // the slider's play along the rotor axis, either way: blade 0.35, registration 0.05, yaw 0.36, rotor 0.25
	coneInsertR   = 0.95  // the section at the pocket mouth at the full insertion (+conePlay)
	coneFit       = 0.20  // the pocket's clearance round that section
	coneSpan      = 2.55  // from the tongue face to the pocket mouth
	coneJoin      = 1.30  // the junction of the buttress and the right cone lies this far outside the pocket mouth
	coneEngage    = 2.85  // from the pocket mouth to the tip
	coneChamfer   = 0.15  // the tip's chamfer
)

var (
	coneSlope   = math.Tan(coneHalfAngle * math.Pi / 180)
	coneMouthR  = coneInsertR - conePlay*coneSlope                  // the section at the mouth at the nominal pose: 0.809
	coneJoinR   = coneMouthR + coneJoin*coneSlope                   // the section at the junction: 0.992
	coneTipR    = coneMouthR - coneEngage*coneSlope                 // the section at the tip, before its chamfer: 0.409
	coneLength  = coneJoin + coneEngage                             // the right cone's length: 4.15
	coneOffsetW = -coneOffset * math.Cos(coneOffsetDeg*math.Pi/180) // the base centre's distance toward the rotor axis along w: 1.090
	coneOffsetZ = coneOffset * math.Sin(coneOffsetDeg*math.Pi/180)  // its height from the drive axis: -1.556
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

// The tongue's root rounds and the feet where they meet the head.
const (
	// Root r 1.05 and feet 1.2, not a root r 1.2 with feet 1.35: decad refuses the 0.5 chamfer of the stem faces
	// ("a partial chamfer's cutter meets or comes within bounds of another face") on the head-top run next to a root round of
	// about 1.15 or more, and builds it at 1.1 and under.
	tongueRootR = 1.05
	tongueFoot  = 1.2
)

// toAxis is +1 when the rotor axis lies toward +w from the drive axis along the tongue sketch's w coordinate, and -1 when
// it lies toward -w. A slider of a, g or d has its drive axis at yi + 2.25 and the rotor axis at yi; b and c have x 22.5 and
// 24.75, e and f have x -27 and -24.75.
func toAxis(uIsX bool) float64 {
	if uIsX {
		return -1
	}
	return 1
}

// hullLoop draws, on s in its own (x, y), the hull of circle 1 (centre c1, radius r1) and circle 2 (c2, r2), r1 >= r2: arc 1 on
// the far side of circle 1, the lower tangent, arc 2 on the far side of circle 2, the upper tangent, all counter-clockwise.
// Both circles' centres are dimensioned, both arcs carry their radius, and each tangent line touches both arcs.
func hullLoop(s *sketch.Sketch, c1 [2]float64, r1 float64, c2 [2]float64, r2 float64) {
	th := math.Atan2(c2[1]-c1[1], c2[0]-c1[0])
	bt := math.Acos((r1 - r2) / math.Hypot(c2[0]-c1[0], c2[1]-c1[1]))
	pt := func(c [2]float64, r, a float64) *sketch.Point {
		return s.CreatePoint(c[0]+r*math.Cos(a), c[1]+r*math.Sin(a))
	}
	t1p, t1m := pt(c1, r1, th+bt), pt(c1, r1, th-bt)
	t2p, t2m := pt(c2, r2, th+bt), pt(c2, r2, th-bt)
	k1, k2 := sketchcheck.DimensionPoint(s, c1[0], c1[1]), sketchcheck.DimensionPoint(s, c2[0], c2[1])
	a1 := s.CreateArc(k1, t1p, t1m)
	l1 := s.CreateLine(t1m, t2m)
	a2 := s.CreateArc(k2, t2m, t2p)
	l2 := s.CreateLine(t2p, t1p)
	s.AddConstraint(sketch.NewRadius(a1, r1), sketch.NewRadius(a2, r2), sketch.NewTangent(l1, a1), sketch.NewTangent(l1, a2),
		sketch.NewTangent(l2, a2), sketch.NewTangent(l2, a1))
}

// tongueOutline draws the tongue in (w, z): the strip [sL, sR] with its feet and r 1.05 root rounds at the head, its two sides
// up to the end of radius coneDiscR about (dc, zc), and the two coneDiscRound inside rounds that join each side to it. The end
// is the hull of that disc and the junction circle of radius coneJoinR about the drive axis (jw, jz): the disc's arc round the
// back, two tangent lines and the junction circle's arc at the front.
func tongueOutline(s *sketch.Sketch, sL, sR, dc, zc, jw, jz float64) {
	dp := func(x, y float64) *sketch.Point { return sketchcheck.DimensionPoint(s, x, y) }
	zh := sliderHeadTop
	zr := zh + tongueRootR - tongueRootDip
	big, rf := coneDiscR, coneDiscRound
	// The inside rounds' centres lie outside the strip's side and outside the disc, rf from the side and big + rf from the disc centre.
	fR := [2]float64{sR + rf, zc - math.Sqrt((big+rf)*(big+rf)-(sR+rf-dc)*(sR+rf-dc))}
	fL := [2]float64{sL - rf, zc - math.Sqrt((big+rf)*(big+rf)-(sL-rf-dc)*(sL-rf-dc))}
	onDisc := func(f [2]float64) [2]float64 {
		dx, dy := f[0]-dc, f[1]-zc
		d := math.Hypot(dx, dy)
		return [2]float64{dc + big*dx/d, zc + big*dy/d}
	}
	tR, tL := onDisc(fR), onDisc(fL)
	var right, left []*sketch.Point
	right = append(right, dp(sR+tongueFoot, zh-tongueFoot), dp(sR+tongueFoot, zr-tongueRootR))
	for _, q := range quarterRound(sR+tongueRootR, zr, tongueRootR, 270, 180, 8) {
		right = append(right, dp(q[0], q[1]))
	}
	for _, q := range quarterRound(sL-tongueRootR, zr, tongueRootR, 0, -90, 8) {
		left = append(left, dp(q[0], q[1]))
	}
	left = append(left, dp(sL-tongueFoot, zr-tongueRootR), dp(sL-tongueFoot, zh-tongueFoot))
	sideR := s.CreatePoint(sR, fR[1])
	sideL := s.CreatePoint(sL, fL[1])
	pR, pL := s.CreatePoint(tR[0], tR[1]), s.CreatePoint(tL[0], tL[1])
	cR, cL, cD := s.CreatePoint(fR[0], fR[1]), s.CreatePoint(fL[0], fL[1]), dp(dc, zc)
	for i := 0; i+1 < len(right); i++ {
		s.CreateLine(right[i], right[i+1])
	}
	lR := s.CreateLine(right[len(right)-1], sideR)
	aR := s.CreateArc(cR, pR, sideR) // counter-clockwise from the disc to the side: the round turns outward
	// The two external tangents of the disc and the junction circle: normals at th +- bt, th the direction from the disc to it.
	rJ := coneJoinR
	th := math.Atan2(jz-zc, jw-dc)
	bt := math.Acos((big - rJ) / math.Hypot(jw-dc, jz-zc))
	aR0 := math.Atan2(tR[1]-zc, tR[0]-dc)
	norm := func(a float64) float64 {
		for a < aR0 {
			a += 2 * math.Pi
		}
		for a >= aR0+2*math.Pi {
			a -= 2 * math.Pi
		}
		return a
	}
	n1, n2 := norm(th-bt), norm(th+bt)
	if n2 < n1 {
		n1, n2 = n2, n1
	}
	pt := func(cw, cz, r, a float64) *sketch.Point { return s.CreatePoint(cw+r*math.Cos(a), cz+r*math.Sin(a)) }
	tD1, tJ1, tJ2, tD2 := pt(dc, zc, big, n1), pt(jw, jz, rJ, n1), pt(jw, jz, rJ, n2), pt(dc, zc, big, n2)
	cJ := dp(jw, jz)
	disc := s.CreateArc(cD, pR, tD1)
	l1 := s.CreateLine(tD1, tJ1)
	arcJ := s.CreateArc(cJ, tJ1, tJ2)
	l2 := s.CreateLine(tJ2, tD2)
	discB := s.CreateArc(cD, tD2, pL)
	aL := s.CreateArc(cL, sideL, pL)
	lL := s.CreateLine(sideL, left[0])
	for i := 0; i+1 < len(left); i++ {
		s.CreateLine(left[i], left[i+1])
	}
	s.CreateLine(left[len(left)-1], right[0])
	s.AddConstraint(sketch.NewVertical(lR), sketch.NewVertical(lL),
		sketch.NewRadius(disc, big), sketch.NewRadius(aR, rf), sketch.NewRadius(aL, rf),
		sketch.NewTangent(lR, aR), sketch.NewTangent(lL, aL),
		sketch.NewTangentCircles(aR, disc, false), sketch.NewTangentCircles(aL, discB, false),
		sketch.NewRadius(discB, big), sketch.NewRadius(arcJ, rJ), sketch.NewTangent(l1, disc),
		sketch.NewTangent(l1, arcJ), sketch.NewTangent(l2, arcJ), sketch.NewTangent(l2, discB))
}

// rootedTongue builds the tongue and its drive cone. u is the rotor axis direction (world X when uIsX, else world Y); the
// tongue runs along +u from ua, the print face, to ue, the cone face; its strip spans [sL, sR] in the other horizontal axis;
// the drive axis is at (sc, zp). The result has r 1.05 roots on three sides, the end hull, and the cone: a buttress lofted from
// the end hull to the junction plane, and a right cone on the drive axis from there to the tip. The buttress is flush with the
// tongue's end outline, so the cone has no root ring on the tongue face, and no face of the cone points down.
func (r *run) rootedTongue(uIsX bool, ua, ue, sL, sR, sc, zp float64) (*decad.Body, error) {
	dc, zc := sc+toAxis(uIsX)*coneOffsetW, zp+coneOffsetZ
	// 1. The hull prism: the outline in (w, z), extruded along u from ua to ue + 1.6. Its chord runs make its end cap a faceted
	// face that decad places 0.0027 off the plane it was drawn on, so the prism starts 1.0 early and a box cuts it back to
	// ua: the end face is then the exact plane of the stem's print face, and the two merge.
	f1, d1 := slFrame(uIsX, ua-1.0)
	s1, err := r.sketchOn(f1)
	if err != nil {
		return nil, err
	}
	tongueOutline(s1, sL, sR, dc, zc, sc, zp)
	p1, err := r.extrude(s1, 0, ue+1.6-ua+1.0, d1)
	if err != nil {
		return nil, fmt.Errorf("tongue hull prism: %w", err)
	}
	fb, _ := slFrame(uIsX, ua-3.0)
	cutBack, err := r.polyBody(fb, [][2]float64{{sL - 6, sliderStemFrom - 1}, {sR + 6, sliderStemFrom - 1}, {sR + 6, zp + 6}, {sL - 6, zp + 6}},
		decad.Distance{D: units.Millimeters(3.0), Dir: d1})
	if err != nil {
		return nil, fmt.Errorf("tongue hull cut-back box: %w", err)
	}
	if p1, err = decad.Cut(r.ctx, p1, cutBack); err != nil {
		return nil, fmt.Errorf("tongue hull cut-back: %w", err)
	}
	// 2. The end cutter in (u, z), extruded along w over the tongue's width: it removes everything past ue except the root
	// rounds to the head.
	wLo := math.Min(sL, dc-coneDiscR)
	var f2 r3.Frame
	var d2 decad.Direction
	if uIsX {
		f2, _ = r3.NewFrame(r3.Vec{Y: wLo - 2.25}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		d2 = decad.Against
	} else {
		f2, _ = r3.NewFrame(r3.Vec{X: wLo - 2.25}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		d2 = decad.Along
	}
	ztop := zp + 4.5
	zcE := sliderHeadTop + tongueRootR - tongueRootDip
	cv := [][2]float64{{ue, ztop}}
	cv = append(cv, quarterRound(ue+tongueRootR, zcE, tongueRootR, 180, 270, 8)...)
	cv = append(cv, [2]float64{ue + tongueFoot, zcE - tongueRootR}, [2]float64{ue + tongueFoot, sliderHeadTop - tongueEndGap},
		[2]float64{ue + 2.0, sliderHeadTop - tongueEndGap}, [2]float64{ue + 2.0, ztop})
	s2, err := r.sketchOn(f2)
	if err != nil {
		return nil, err
	}
	polyLoop(s2, cv)
	cut, err := r.extrude(s2, 0, math.Max(sR, dc+coneDiscR)-wLo+4.5, d2)
	if err != nil {
		return nil, fmt.Errorf("tongue end cutter: %w", err)
	}
	tongue, err := decad.Cut(r.ctx, p1, cut)
	if err != nil {
		return nil, fmt.Errorf("tongue end cut: %w", err)
	}
	// 3. The buttress: a ruled loft from the end hull on u = ue, flush with the tongue's end outline, to the junction plane
	// u = uj.
	uj := ue + coneSpan - coneJoin
	plane := func(u float64) r3.Frame {
		if uIsX {
			f, _ := r3.NewFrame(r3.Vec{X: u}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
			return f
		}
		f, _ := r3.NewFrame(r3.Vec{Y: u}, r3.Vec{Z: 1}, r3.Vec{X: 1}) // U = Z, V = X: normal +Y
		return f
	}
	// hull is the section of the buttress at u: the hull of the junction circle and the circle at fraction t of the way from
	// the end disc to the junction circle (centre and radius interpolated). Every such circle touches the same two tangent
	// lines as the end disc does, so the loft's walls are two vertical planes and the junction circle's vertical cylinder.
	hull := func(u, t float64) (*sketch.Sketch, *sketch.Profile, error) {
		s, err := r.sketchOn(plane(u))
		if err != nil {
			return nil, nil, err
		}
		cw, cz := sc+(1-t)*(dc-sc), zp+(1-t)*(zc-zp)
		rr := (1-t)*coneDiscR + t*coneJoinR
		p := func(w, z float64) [2]float64 {
			if uIsX {
				return [2]float64{w, z}
			}
			return [2]float64{z, w}
		}
		hullLoop(s, p(cw, cz), rr, p(sc, zp), coneJoinR)
		pr, err := r.profile(s, 0)
		return s, pr, err
	}
	hb, pb, err := hull(ue, 0)
	if err != nil {
		return nil, fmt.Errorf("boss base: %w", err)
	}
	ht, pt, err := hull(uj, coneTau)
	if err != nil {
		return nil, fmt.Errorf("boss top: %w", err)
	}
	boss, err := decad.Loft(r.ctx, r.doc, hb, pb, ht, pt)
	if err != nil {
		return nil, fmt.Errorf("boss loft: %w", err)
	}
	// 4. The right cone: a meridian (along u from uj, radius) revolved about the drive axis. Its first side is the junction
	// circle's disc, which lies in the buttress's top cap.
	o, axis := r3.Vec{X: uj, Y: sc, Z: zp}, r3.Vec{X: 1}
	if !uIsX {
		o, axis = r3.Vec{X: sc, Y: uj, Z: zp}, r3.Vec{Y: 1}
	}
	fm, err := r3.NewFrame(o, axis, r3.Vec{Z: 1})
	if err != nil {
		return nil, err
	}
	sm, err := r.sketchOn(fm)
	if err != nil {
		return nil, err
	}
	rEnd := coneTipR + coneChamfer*coneSlope // the radius where the chamfer starts, coneChamfer back from the tip
	polygon(sm, [][2]float64{{0, 0}, {0, coneJoinR}, {coneLength - coneChamfer, rEnd}, {coneLength, rEnd - coneChamfer}, {coneLength, 0}})
	prm, err := r.profile(sm, 0)
	if err != nil {
		return nil, fmt.Errorf("cone meridian: %w", err)
	}
	cone, err := decad.Revolve(r.ctx, r.doc, sm, prm, decad.ConstructionAxis{Origin: o, Dir: axis}, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("cone revolve: %w", err)
	}
	drive, err := decad.Union(r.ctx, boss, cone)
	if err != nil {
		return nil, fmt.Errorf("boss and right cone: %w", err)
	}
	if tongue, err = decad.Union(r.ctx, tongue, drive); err != nil {
		return nil, fmt.Errorf("tongue and cone: %w", err)
	}
	return tongue, nil
}

// bfProfile draws the b-f slider's main profile (stem, arm, head and blade) in (x, z) on y = yi - 1.8 and extrudes it 3.6. The
// arm points to +x for b and c and to -x for e and f.
func (r *run) bfProfile(ef bool, yi float64) (*decad.Body, float64, float64, error) {
	f, d := slFrame(false, yi-StemThickness/2)
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, 0, 0, err
	}
	bL, bR := 13.05, 14.7 // the blade is 1.65 thick
	zh, za := sliderHeadTop, sliderHeadTop-3.75
	h := sliderHalfWide
	var v [][2]float64
	if !ef {
		v = [][2]float64{{-h, sliderStemFrom}, {h, sliderStemFrom}, {h, za}, {26.85, za}, {26.85, zh},
			{bR, zh}, {bR, bladeShoulderZ}, {(bL + bR) / 2, bladeRidgeZ}, {bL, bladeShoulderZ}, {bL, zh}, {-h, zh}}
	} else {
		bL, bR = -14.7, -13.05
		v = [][2]float64{{-h, sliderStemFrom}, {h, sliderStemFrom}, {h, zh}, {bR, zh}, {bR, bladeShoulderZ}, {(bL + bR) / 2, bladeRidgeZ},
			{bL, bladeShoulderZ}, {bL, zh}, {-29.25, zh}, {-29.25, za}, {-h, za}}
	}
	ps, ls := polygon(s, v)
	n := len(v)
	type round struct {
		i   int
		rad float64
	}
	rounds := []round{{2, 1.875}, {5, 0.9}, {9, 0.9}}
	if ef {
		rounds = []round{{3, 0.9}, {7, 0.9}, {10, 1.875}}
	}
	for _, q := range rounds {
		prev := ls[(q.i-1+n)%n]
		if _, err := sketch.CreateFillet(s, prev, ls[q.i], ps[q.i], q.rad); err != nil {
			return nil, 0, 0, fmt.Errorf("profile round %d: %w", q.i, err)
		}
	}
	body, err := r.extrude(s, 0, StemThickness, d)
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
	ef2, ed2 := slFrame(false, yi+StemThickness/2-bladeExtOverlap)
	es, err := r.sketchOn(ef2)
	if err != nil {
		return nil, err
	}
	polyLoop(es, [][2]float64{{bL, zh}, {bR, zh}, {bR, bladeShoulderZ}, {(bL + bR) / 2, bladeRidgeZ}, {bL, bladeShoulderZ}})
	ext, err := r.extrude(es, 0, 1.8+bladeExtOverlap, ed2)
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
	tg, err := r.rootedTongue(false, yi-StemThickness/2, yi-0.3, sL, sR, sc, zp)
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
	_, d := slFrame(true, -sliderHalfWide)
	f, err := r3.NewFrame(r3.Vec{X: -sliderHalfWide, Y: yi}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	zh := sliderHeadTop
	sh := StemThickness / 2
	v := [][2]float64{{-sh, sliderStemFrom}, {sh, sliderStemFrom}, {sh, zh - 2.475}, {3.6, zh - 1.575}, {3.6, zh},
		{-10.05, zh}, {-10.05, bladeShoulderZ}, {-10.875, bladeRidgeZ}, {-11.7, bladeShoulderZ},
		{-11.7, zh}, {-12.9, zh}, {-12.9, zh - 3.0}, {-sh, zh - 3.0}}
	ps, ls := polygon(s, v)
	for _, c := range []struct {
		a, b int
		rad  float64
	}{{4, 5, 0.9}, {8, 9, 0.9}, {11, 12, 1.5}} {
		if _, err := sketch.CreateFillet(s, ls[c.a], ls[c.b], ps[c.b], c.rad); err != nil {
			return nil, fmt.Errorf("profile round %d: %w", c.b, err)
		}
	}
	body, err := r.extrude(s, 0, 2*sliderHalfWide, d)
	if err != nil {
		return nil, fmt.Errorf("main profile: %w", err)
	}
	tg, err := r.rootedTongue(true, -sliderHalfWide, -2.85, yi+pinStationAGD-tongueWidth, yi+pinStationAGD, yi+pinStationAGD, pinAxisZ)
	if err != nil {
		return nil, err
	}
	if body, err = decad.Union(r.ctx, body, tg); err != nil {
		return nil, fmt.Errorf("union tongue: %w", err)
	}
	return r.finishSlider(body, seg, yi)
}

// finishSlider chamfers the stem's vertical edges and adds the integral groove pin of b to f, or cuts the keyed hole of the
// separate pin of a, g and d.
func (r *run) finishSlider(body *decad.Body, seg Segment, yi float64) (*decad.Body, error) {
	// The 0.5 chamfers of "Surface edge finishing" on the stem's two large faces: every long run of their outer loops, the stem's
	// vertical edges included, so the stem takes these chamfers and no smaller bevel (Fusion cannot chamfer an
	// edge that already carries a smaller bevel).
	body, _, err := edgefinish.ChamferRuns(r.ctx, body, r3.Vec{Y: 1}, []float64{yi - StemThickness/2, yi + StemThickness/2},
		edgefinish.Outer, func(*decad.Loop, []*decad.Edge) bool { return false }, wallChamfer)
	if err != nil {
		return nil, fmt.Errorf("stem face chamfers: %w", err)
	}
	if _, _, horizontal := segmentCentre(seg); horizontal {
		// A, g and d take a separate pin: a keyed hole whose centre is raise below the pin's axis, and the body then rises by
		// raise, so the pin on the groove's centreline sits in the middle of the hole's play.
		if body, err = r.groovePinHole(body, yi+StemThickness/2, sliderPinZ-raise(seg)); err != nil {
			return nil, err
		}
		lift, err := r3.Translation(r3.Vec{Z: raise(seg)})
		if err != nil {
			return nil, err
		}
		return decad.Transform(r.ctx, body, lift)
	}
	gp, err := r.pin(yi + StemThickness/2)
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
