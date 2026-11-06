package handcrankfan

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Rotor dimensions in mm. The hub's back face is at Y = -hubBack and its front face at Y = -(hubBack + hubLength).
const (
	hubRadius = 14.0
	hubLength = 20.0
	hubBack   = -(fanWallY - wallThick) + fanGap
	hubMid    = hubBack + hubLength/2

	fanTip       = 26.0 // the fan shaft's rear tip, inside the pinion and clear of the drive gear's teeth
	studRadius   = 3.0  // the blade's root stud
	studStart    = 6.0  // radius of the stud's inner end
	studEnd      = 16.0 // radius of its outer end, inside the blade's root
	socketRadius = 3.15 // 0.15 sliding clearance around the stud
	socketStart  = 5.6  // radius of the socket floor
)

// The blade bolt is an M2 x 16 socket head cap screw with an M2 nut. It runs parallel to the fan axis at boltAt
// from the axis: the head sits in a counterbore on the front face and the nut in a hex trap in the back face.
const (
	boltAt            = 9.5
	holeRadius        = 1.15 // clearance hole, diameter 2.3
	headRadius        = 1.9
	headHeight        = 2.0
	counterboreRadius = 2.15
	counterboreDepth  = 2.2
	shankRadius       = 1.0
	boltLength        = 16.0
	nutFlats          = 4.0
	nutHeight         = 1.6
	trapFlats         = 4.3
	trapDepth         = 4.0
	hardwareGap       = 0.1 // the modelled bolt sits just clear of its seats, as it does before it is tightened
)

// Blade section constants.
const (
	designAttack = 5 * math.Pi / 180  // design angle of attack of the cambered section
	refRadius    = 48.0               // BladePitch is the blade angle here, 75 % of the tip radius
	refPitch     = 25 * math.Pi / 180 // the pitch the sections are drawn at; the finished blade is turned from it
	holdRadius   = 21.5               // inboard of this the blade angle stays constant, as on a turbine root
	camberAt     = 0.40               // chord fraction of the maximum camber
	edgeThick    = 0.5                // trailing-edge thickness
	sidePoints   = 10                 // fit points per surface, besides the leading edge
)

// station is one blade section: radius, chord, thickness ratio, maximum camber as a fraction of the chord, and the
// weight of the ellipse in the shape (1 is an ellipse or the round root, 0 is the airfoil).
type station struct{ r, chord, ratio, camber, ellipse float64 }

var stations = [...]station{
	{14.3, 8.0, 1.00, 0.000, 1.00},
	{16.3, 8.0, 1.00, 0.000, 1.00},
	{19.0, 11.5, 0.60, 0.010, 0.60},
	{21.5, 15.5, 0.38, 0.025, 0.25},
	{25.0, 17.8, 0.26, 0.035, 0},
	{30.0, 16.5, 0.22, 0.040, 0},
	{38.0, 14.5, 0.19, 0.040, 0},
	{46.0, 12.5, 0.18, 0.040, 0},
	{55.0, 10.5, 0.17, 0.035, 0},
	{64.0, 8.0, 0.18, 0.030, 0},
}

// beta is the blade angle at radius r, measured from the plane of rotation, for a blade angle of pitch at refRadius.
// The shape follows the angle of a uniform axial inflow plus designAttack, held constant across the root. The pitch
// turns the whole blade, as on a turbine with a pitch bearing.
func beta(r, pitch float64) float64 {
	return designAttack + math.Atan(math.Tan(refPitch-designAttack)*refRadius/math.Max(r, holdRadius)) + (pitch - refPitch)
}

// fan builds the fan shaft, the hub, the blades and the blade bolts.
func (b *builder) fan() {
	b.part("fan-shaft", "shaft", "fan", func() *decad.Body {
		shaft := b.cylinder(alongMinusY(-fanTip), shaftRadius, hubBack+hubLength+4-fanTip)
		// One set of grooves passes through the pinion and one through the hub, with the round shaft between them for the
		// bearings. The collar sits between the front bearing and the hub.
		shaft = b.grooves(shaft, fanSpline, fanTip-1, 36.5)
		shaft = b.grooves(shaft, fanSpline, hubBack-1, hubBack+hubLength+5)
		return b.union(shaft, b.cylinder(alongMinusY(-(fanBlockFace-0.2)), 6, hubBack-0.1-(fanBlockFace-0.2)))
	})
	b.part("fan-hub", "rotor", "fan", b.hub)
	var blades, bolts []*decad.Body
	b.part("fan-blade-1", "blade", "fan", func() *decad.Body {
		blades = b.blades()
		if len(blades) == 0 {
			return nil
		}
		return blades[0]
	})
	for k := 1; k < b.p.Blades; k++ {
		b.part(fmt.Sprintf("fan-blade-%d", k+1), "blade", "fan", func() *decad.Body { return blades[k] })
	}
	b.part("fan-bolt-1", "hardware", "fan", func() *decad.Body {
		bolts = b.bolts()
		if len(bolts) == 0 {
			return nil
		}
		return bolts[0]
	})
	for k := 1; k < b.p.Blades; k++ {
		b.part(fmt.Sprintf("fan-bolt-%d", k+1), "hardware", "fan", func() *decad.Body { return bolts[k] })
	}
}

// onY is the plane at y whose normal is -Y and whose coordinates are measured from the bolt line of blade 1.
func onY(y float64) r3.Frame {
	return frame(r3.Vec{X: boltAt, Y: y, Z: axisHeight}, r3.Vec{X: 1}, r3.Vec{Z: 1})
}

// ring returns body and its copies about the fan axis, one for each blade.
func (b *builder) ring(body *decad.Body) []*decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	out := []*decad.Body{body}
	if b.p.Blades > 1 {
		// Transform retires its input, so the other copies are made from the finished one.
		copies, err := decad.PatternCopies(b.ctx, body, decad.CircularPattern{
			Center: b.a.State.Apex, Axis: fanAxis, Count: b.p.Blades})
		if err != nil {
			b.fail("copy about the fan axis", err)
			return nil
		}
		out = append(out, copies...)
	}
	return out
}

// hub builds the disc with the shaft bore and one bolt hole for each blade, then cuts a socket, a head counterbore
// and a nut trap for each blade. The tools run past the faces they open, so none shares a face with the hub.
func (b *builder) hub() *decad.Body {
	n := b.p.Blades
	s := b.section(frame(r3.Vec{Y: -hubBack, Z: axisHeight}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
	circle(s, 0, 0, hubRadius)
	for k := 0; k < n; k++ {
		theta := 2 * math.Pi * float64(k) / float64(n)
		circle(s, boltAt*math.Cos(theta), boltAt*math.Sin(theta), holeRadius)
	}
	hub := b.roundCaps(b.extrude(s, n, hubLength), 1)
	hub = b.splineCut(hub, fanSpline, hubBack-1, hubLength+2)
	front := -(hubBack + hubLength)
	// The hub prints front face down, so a socket lies on its side: its point is toward the back face, which is up.
	socket := b.teardropPrism(frame(r3.Vec{X: socketStart, Y: -hubMid, Z: axisHeight}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
		socketRadius, hubRadius+1-socketStart)
	counterbore := b.cylinder(onY(front+counterboreDepth), counterboreRadius, counterboreDepth+1)
	trap := b.hexagon(onY(-hubBack+1), trapFlats, trapDepth+1)
	for _, tool := range []*decad.Body{socket, counterbore, trap} {
		for _, copy := range b.ring(tool) {
			hub = b.cut(hub, copy)
		}
	}
	return hub
}

// blades lofts blade 1 through the sections drawn at refPitch, turns it to the pitch, joins a drilled root stud to
// it, places it along +X in the hub's mid plane and copies it about the fan axis.
func (b *builder) blades() []*decad.Body {
	if b.err != nil {
		return nil
	}
	w := sketch.NewWorld()
	sections := make([]decad.LoftSection, len(stations))
	for i, st := range stations {
		sections[i] = b.bladeSection(w, st)
	}
	if b.err != nil {
		return nil
	}
	// Decad's twisted loft panics for some section tables when the pitch changes the sections, so every pitch
	// lofts the same sections and the finished loft is turned about the stacking line.
	loft, err := decad.LoftSections(b.ctx, b.doc, sections...)
	if err != nil {
		b.fail("loft blade", err)
		return nil
	}
	if turn := b.p.BladePitch - refPitch; math.Abs(turn) > 1e-12 {
		xf, err := r3.RotationAround(r3.Vec{}, r3.Vec{Z: 1}, units.Radians(turn))
		if err != nil {
			b.fail("create pitch rotation", err)
			return nil
		}
		if loft, err = decad.Transform(b.ctx, loft, xf); err != nil {
			b.fail("turn blade to pitch", err)
			return nil
		}
	}
	// Local Z is the blade's radial line, local X the direction of motion and local Y downstream.
	stud := b.cylinder(frame(r3.Vec{Z: studStart}, r3.Vec{X: 1}, r3.Vec{Y: 1}), studRadius, studEnd-studStart)
	stud = b.cut(stud, b.cylinder(frame(r3.Vec{Y: -5, Z: boltAt}, r3.Vec{Z: 1}, r3.Vec{X: 1}), holeRadius, 10))
	blade := b.union(loft, stud)
	if b.err != nil {
		return nil
	}
	// Local Z becomes +X, local X the direction of motion (+Z) and local Y downstream (-Y).
	xf, err := r3.FromFrame(frame(r3.Vec{Y: -hubMid, Z: axisHeight}, r3.Vec{Z: 1}, r3.Vec{Y: -1}))
	if err != nil {
		b.fail("create blade placement", err)
		return nil
	}
	if blade, err = decad.Transform(b.ctx, blade, xf); err != nil {
		b.fail("place blade", err)
		return nil
	}
	return b.ring(blade)
}

// bolts builds one bolt with its nut as a single plain body (no thread) for each blade, sitting hardwareGap clear
// of its seats.
func (b *builder) bolts() []*decad.Body {
	front := -(hubBack + hubLength)
	seat := front + counterboreDepth - hardwareGap
	head := b.cylinder(onY(seat), headRadius, headHeight)
	shank := b.cylinder(onY(seat+boltLength), shankRadius, boltLength+0.8)
	nut := b.hexagon(onY(-hubBack-trapDepth+nutHeight+hardwareGap), nutFlats, nutHeight)
	return b.ring(b.union(head, shank, nut))
}

func camberLine(x, m float64) float64 {
	if x < camberAt {
		return m / (camberAt * camberAt) * (2*camberAt*x - x*x)
	}
	return m / ((1 - camberAt) * (1 - camberAt)) * ((1 - 2*camberAt) + 2*camberAt*x - x*x)
}

// pivotAt is the chord fraction of the stacking point: 0.4 for an airfoil, 0.5 for the round root.
func pivotAt(st station) float64 { return camberAt + st.ellipse*(0.5-camberAt) }

// halfThickness is the half thickness at chord fraction x: the NACA 4-digit shape blended with an ellipse, plus a
// linear trailing-edge ramp.
func halfThickness(x float64, st station) float64 {
	thick := st.ratio * st.chord
	n := 0.2969*math.Sqrt(x) - 0.1260*x - 0.3516*x*x + 0.2843*x*x*x - 0.1036*x*x*x*x
	naca := (thick/2-0.3*edgeThick/2)/0.10002*n + edgeThick/2*x
	ellipse := thick*math.Sqrt(x*(1-x)) + edgeThick/2*x
	return (1-st.ellipse)*naca + st.ellipse*ellipse
}

// sectionPoints returns the section's fit points from the trailing edge along the suction side, round the leading
// edge and back along the pressure side, in the plane local to the stacking point and turned to the blade angle.
func sectionPoints(st station, angle float64) [][2]float64 {
	pivot := pivotAt(st)
	etaPivot := st.chord * camberLine(pivot, st.camber)
	cosB, sinB := math.Cos(angle), math.Sin(angle)
	local := func(x, eta float64) [2]float64 {
		du, dv := pivot*st.chord-x*st.chord, eta-etaPivot
		return [2]float64{du*cosB - dv*sinB, -du*sinB - dv*cosB}
	}
	chordFraction := func(k int) float64 { return (1 - math.Cos(math.Pi*float64(k)/sidePoints)) / 2 }
	var pts [][2]float64
	for k := sidePoints; k >= 1; k-- {
		x := chordFraction(k)
		pts = append(pts, local(x, st.chord*camberLine(x, st.camber)+halfThickness(x, st)))
	}
	pts = append(pts, local(0, 0))
	for k := 1; k <= sidePoints; k++ {
		x := chordFraction(k)
		pts = append(pts, local(x, st.chord*camberLine(x, st.camber)-halfThickness(x, st)))
	}
	return pts
}

// bladeSection sketches one station on the world XY plane at height r: a fit spline through the section's points
// closed by two trailing-edge lines that meet at the midpoint of the edge. The sketch origin is the stacking point.
func (b *builder) bladeSection(w *sketch.World, st station) decad.LoftSection {
	if b.err != nil {
		return decad.LoftSection{}
	}
	plane, err := w.CreateOffsetPlane(w.XY(), st.r)
	if err != nil {
		b.fail("create blade plane", err)
		return decad.LoftSection{}
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		b.fail("create blade section", err)
		return decad.LoftSection{}
	}
	b.a.Sketches = append(b.a.Sketches, sk)
	pts := sectionPoints(st, beta(st.r, refPitch))
	// The blades are mirrored: the spur pair reverses the crank's turn, so the fan turns the other way round and its
	// blades must lead with the other face.
	for i := range pts {
		pts[i][0] = -pts[i][0]
	}
	fit := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		fit[i] = sketchcheck.DimensionPoint(sk, p[0], p[1])
	}
	first, last := fit[0], fit[len(fit)-1]
	tail := sk.CreatePoint((pts[0][0]+pts[len(pts)-1][0])/2, (pts[0][1]+pts[len(pts)-1][1])/2)
	sk.AddConstraint(sketch.NewMidpointOf(tail, first, last))
	if _, err := sk.CreateFitSpline(fit...); err != nil {
		b.fail("create blade spline", err)
		return decad.LoftSection{}
	}
	sk.CreateLine(last, tail)
	sk.CreateLine(tail, first)
	if _, err := sk.Solve(b.ctx); err != nil {
		b.fail("solve blade section", err)
		return decad.LoftSection{}
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, sk); err != nil {
		b.fail("verify blade section", fmt.Errorf("radius %g: %w", st.r, err))
		return decad.LoftSection{}
	}
	for _, profile := range sk.Profiles() {
		if profile.Valid && len(profile.Holes) == 0 && len(profile.Outer) == 3 {
			return decad.LoftSection{Sketch: sk, Profile: profile}
		}
	}
	b.fail("select blade profile", fmt.Errorf("radius %g: no valid profile of three edges", st.r))
	return decad.LoftSection{}
}
