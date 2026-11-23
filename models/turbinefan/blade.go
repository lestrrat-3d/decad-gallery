package turbinefan

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Blade and joint dimensions in mm. Other models build a hub and a blade joint from them.
const (
	// BladeSocketFloorRound rounds the outline of the hub socket's floor (the socket's size is in Size).
	BladeSocketFloorRound = 0.5
	// DefaultBladePitch is the blade angle at 75 % of the tip radius that the sections are drawn at. A blade's pitch lies in
	// [MinBladePitch, MaxBladePitch].
	DefaultBladePitch = refPitch
	MinBladePitch     = 10 * math.Pi / 180
	MaxBladePitch     = 40 * math.Pi / 180

	studChamfer = 0.3 // chamfer on the stud's inner end
)

// Blade section constants.
const (
	designAttack = 5 * math.Pi / 180  // design angle of attack of the cambered section
	refPitch     = 25 * math.Pi / 180 // the pitch the sections are drawn at; the finished blade is turned from it
	camberAt     = 0.40               // chord fraction of the maximum camber
	edgeThick    = 1.0                // trailing-edge thickness
	sidePoints   = 10                 // fit points per surface, besides the leading edge
)

// station is one blade section: radius, chord, thickness ratio, maximum camber as a fraction of the chord, and the
// weight of the ellipse in the shape (1 is an ellipse or the round root, 0 is the airfoil).
type station struct{ r, chord, ratio, camber, ellipse float64 }

// stationsFor returns the section table for sz. The first two stations are the round root and scale by the hub scale;
// the others are the airfoil and scale by the blade scale. Thickness ratio, camber and ellipse weight do not change.
func stationsFor(sz Size) [10]station {
	j, s := sz.HubScale, sz.Scale
	return [10]station{
		{sz.RootRadius, 8.0 * j, 1.00, 0.000, 1.00},
		{sz.RootRadius + 1.9*j, 8.0 * j, 1.00, 0.000, 1.00},
		{19.0 * s, 11.5 * s, 0.60, 0.010, 0.60},
		{21.5 * s, 15.5 * s, 0.38, 0.025, 0.25},
		{25.0 * s, 17.8 * s, 0.26, 0.035, 0},
		{30.0 * s, 16.5 * s, 0.22, 0.040, 0},
		{38.0 * s, 14.5 * s, 0.19, 0.040, 0},
		{46.0 * s, 12.5 * s, 0.18, 0.040, 0},
		{55.0 * s, 10.5 * s, 0.17, 0.035, 0},
		{sz.TipRadius, 8.0 * s, 0.22, 0.030, 0},
	}
}

// bladeBody lofts blade 1 through the sections drawn at refPitch, turns it to pitch and joins a drilled root stud to
// it, with a flat on the stud's inner end for printing. The body is in the blade's local frame: Z is its radial line, X
// the direction of motion and Y downstream.
func (b *bladeBuilder) bladeBody(pitch float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	w := sketch.NewWorld()
	stations := stationsFor(b.sz)
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
	if turn := pitch - refPitch; math.Abs(turn) > 1e-12 {
		if b.mirror {
			turn = -turn
		}
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
	stud := b.hexPrism(frame(r3.Vec{Z: b.sz.BladeStudStart}, r3.Vec{X: 1}, r3.Vec{Y: 1}), b.sz.BladeStudRadius,
		b.sz.BladeStudEnd-b.sz.BladeStudStart)
	// A 0.3 chamfer on the inner end keeps the stud's corner clear of the socket floor's round.
	stud = b.chamferStart(stud, studChamfer)
	reach := b.sz.BladeStudRadius + 2 // the bolt hole's cutter starts and ends this far from the stud's axis
	stud = b.cut(stud, b.cylinder(frame(r3.Vec{Y: -reach, Z: b.sz.BladeBoltAt}, r3.Vec{Z: 1}, r3.Vec{X: 1}),
		m2fastener.HoleRadius, 2*reach))
	// The flat is cut before the union, so the stud stays exact and the blade's held mesh bound stays small.
	stud = b.studFlat(stud, loft, pitch)
	return b.union(loft, stud)
}

// studFlat cuts the flat on the stud's inner end: the stud material under the bed in the print pose at the rest tilt
// that restTiltFitted finds from the turned loft, where the stud end corner lies the fitted depth (StudFlatDepth or less) under the bed. In the local
// frame the bed is the plane n·p = c, with n = sin th Z + cos th u the pose's up direction and c the loft's lowest pose
// height. The cutter is a square of side 4 BladeStudRadius on that plane, centred on the stud end's projection,
// extruded 2 BladeStudRadius along -n.
func (b *bladeBuilder) studFlat(stud, loft *decad.Body, pitch float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	mesh, err := loft.Tessellate(b.ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		b.fail("mesh loft for the stud flat", err)
		return nil
	}
	hand := RightHand
	if b.mirror {
		hand = LeftHand
	}
	tilt, c, _, err := restTiltFitted(mesh.Vertices(), b.sz, pitch, hand)
	if err != nil {
		b.fail("stud flat", err)
		return nil
	}
	n := r3.Vec{Z: math.Sin(tilt)}.Add(poseUp(b.sz, pitch, hand).Scale(math.Cos(tilt)))
	end := r3.Vec{Z: b.sz.BladeStudStart}
	origin := end.Add(n.Scale(c - n.Dot(end)))
	// v is local Z's direction in the plane; u = n × v makes the frame's normal u × v = -n.
	v, ok := r3.Vec{Z: 1}.Sub(n.Scale(n.Z)).Normalize()
	if !ok {
		b.fail("stud flat", errors.New("the flat's plane is normal to the radial line"))
		return nil
	}
	half := 2 * b.sz.BladeStudRadius
	square := [][2]float64{{-half, -half}, {half, -half}, {half, half}, {-half, half}}
	cutter := b.extrudePolygon(frame(origin, n.Cross(v), v), square, b.sz.BladeStudRadius*2, "stud flat")
	return b.cut(stud, cutter)
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
	local := func(x, eta float64) [2]float64 { return sectionLocal(st, angle, x, eta) }
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

// sectionLocal returns the point of station st at chord fraction x, offset eta (mm) toward the suction side, in the
// plane local to the stacking point and turned to the blade angle, before the right hand's X negation.
func sectionLocal(st station, angle, x, eta float64) [2]float64 {
	pivot := pivotAt(st)
	etaPivot := st.chord * camberLine(pivot, st.camber)
	cosB, sinB := math.Cos(angle), math.Sin(angle)
	du, dv := pivot*st.chord-x*st.chord, eta-etaPivot
	return [2]float64{du*cosB - dv*sinB, -du*sinB - dv*cosB}
}

// bladeSection sketches one station on the world XY plane at height r: a fit spline through the section's points
// closed by two trailing-edge lines that meet at the midpoint of the edge. The sketch origin is the stacking point.
func (b *bladeBuilder) bladeSection(w *sketch.World, st station) decad.LoftSection {
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
	b.sketches = append(b.sketches, sk)
	pts := sectionPoints(st, b.sz.Beta(st.r, refPitch))
	// The blades are mirrored: the spur pair reverses the crank's turn, so the fan turns the other way round and its
	// blades must lead with the other face.
	if !b.mirror {
		for i := range pts {
			pts[i][0] = -pts[i][0]
		}
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

// Blade is one blade in its local frame: the origin on the fan axis in the hub's mid plane, +Z the blade's radial line,
// +Y downstream along the fan axis, and the blade moving toward -X. Sketches are every sketch it was built from, each
// fully constrained.
type Blade struct {
	Body     *decad.Body
	Sketches []*sketch.Sketch
}

// BuildBlade lofts the blade at pitch (MinBladePitch to MaxBladePitch), joins its drilled root stud and returns it
// unplaced. sz comes from SizeFor. doc needs a chain tolerance of 0.1 mm (NewDocument): the stud union refuses a plain document. ctx and doc
// must not be nil.
func BuildBlade(ctx context.Context, doc *decad.Document, pitch float64, sz Size) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, sz, false)
}

// BuildBladeMirrored builds the blade of the opposite hand: BuildBlade's blade reflected in the plane x = 0 of the
// blade's local frame, so it moves toward +X and its concave face still faces the way the air goes. Its stud and bolt
// hole are on that plane and do not change. The arguments are BuildBlade's.
func BuildBladeMirrored(ctx context.Context, doc *decad.Document, pitch float64, sz Size) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, sz, true)
}

func buildBlade(ctx context.Context, doc *decad.Document, pitch float64, sz Size, mirror bool) (*Blade, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("turbinefan: context and document must not be nil")
	}
	if math.IsNaN(pitch) || pitch < MinBladePitch || pitch > MaxBladePitch {
		return nil, errors.New("turbinefan: pitch must lie in [10°, 40°] radians")
	}
	if !sz.valid() {
		return nil, errors.New("turbinefan: sz must come from SizeFor")
	}
	b := &bladeBuilder{ctx: ctx, doc: doc, sz: sz, mirror: mirror}
	body := b.bladeBody(pitch)
	if b.err != nil {
		return nil, b.err
	}
	if body == nil || !body.IsSolid() {
		return nil, errors.New("turbinefan: blade is not a solid")
	}
	return &Blade{Body: body, Sketches: b.sketches}, nil
}

// BladeSocketTool builds one blade socket's cutter: a cylinder of radius sz.BladeSocketRadius from f's origin, the socket
// floor's centre, along f's normal (the radial line, outward) for sz.HubRadius + 1 - sz.BladeSocketStart, its start cap's
// outer loop rounded by BladeSocketFloorRound. It returns the cutter and its sketch, fully constrained.
func BladeSocketTool(ctx context.Context, doc *decad.Document, f r3.Frame, sz Size) (*decad.Body, *sketch.Sketch, error) {
	if !sz.valid() {
		return nil, nil, errors.New("turbinefan: sz must come from SizeFor")
	}
	b := &bladeBuilder{ctx: ctx, doc: doc, sz: sz}
	// The hexagon's corners lie on the shaft's direction U of f and its flats face the tangent V, as the stud's do.
	g := frame(f.Origin(), f.V(), f.U().Scale(-1))
	body := b.roundStart(b.hexPrism(g, sz.BladeSocketRadius, sz.HubRadius+1-sz.BladeSocketStart), BladeSocketFloorRound)
	if b.err != nil {
		return nil, nil, b.err
	}
	if len(b.sketches) != 1 {
		return nil, nil, fmt.Errorf("turbinefan: socket tool made %d sketches", len(b.sketches))
	}
	return body, b.sketches[0], nil
}

// bladeBuilder builds a blade or a socket tool. The first failure stops every later step.
type bladeBuilder struct {
	ctx      context.Context
	doc      *decad.Document
	sz       Size
	sketches []*sketch.Sketch
	err      error
	// mirror builds the blade of the opposite hand: its sections are not flipped and its pitch turn is reversed.
	mirror bool
	// what names the part in error messages: "blade" when empty.
	what string
}

func (b *bladeBuilder) fail(operation string, err error) {
	if err != nil && b.err == nil {
		what := b.what
		if what == "" {
			what = "blade"
		}
		b.err = fmt.Errorf("turbinefan: %s %s: %w", what, operation, err)
	}
}

// cylinder extrudes a disc of radius r centred on f's origin along f's normal.
func (b *bladeBuilder) cylinder(f r3.Frame, r, length float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	s, err := newSketch(f)
	if err != nil {
		b.fail("create section", err)
		return nil
	}
	b.sketches = append(b.sketches, s)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), r), r))
	return b.extrudeSection(s, length)
}

// hexPrism extrudes a regular hexagon about f's origin along f's normal. Its corners lie on the circle of radius r, two
// of them on f's V axis, so its flats stand at +-r sqrt(3)/2 on U. Every flat is held at that distance from the origin
// and each next flat turns 60 degrees from the one before, as nutSlotSketch draws a hexagon.
func (b *bladeBuilder) hexPrism(f r3.Frame, r, length float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	s, err := newSketch(f)
	if err != nil {
		b.fail("create section", err)
		return nil
	}
	b.sketches = append(b.sketches, s)
	a := r * math.Sqrt(3) / 2
	var corners [6]*sketch.Point
	for i := range corners {
		t := math.Pi/6 + float64(i)*math.Pi/3
		corners[i] = s.CreatePoint(r*math.Cos(t), r*math.Sin(t))
	}
	var sides [6]*sketch.Line
	for i := range sides {
		sides[i] = s.CreateLine(corners[i], corners[(i+1)%6])
	}
	s.AddConstraint(sketch.NewVertical(sides[5]))
	for i := range sides {
		s.AddConstraint(sketch.NewDistancePointLine(s.Origin(), sides[i], a))
	}
	for i := 0; i < 5; i++ {
		s.AddConstraint(sketch.NewAngle(sides[i], sides[i+1], 60))
	}
	return b.extrudeSection(s, length)
}

// extrudeSection solves s, requires it fully constrained and extrudes its one hole-free profile by length.
func (b *bladeBuilder) extrudeSection(s *sketch.Sketch, length float64) *decad.Body {
	if _, err := s.Solve(b.ctx); err != nil {
		b.fail("solve section", err)
		return nil
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		b.fail("verify section", err)
		return nil
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != 0 {
			continue
		}
		if chosen != nil {
			b.fail("select section profile", errors.New("several valid profiles"))
			return nil
		}
		chosen = pr
	}
	if chosen == nil {
		b.fail("select section profile", errors.New("no valid profile"))
		return nil
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, chosen, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	b.fail("extrude section", err)
	return body
}

// roundStart rounds the outer loop of the start cap of an extruded body.
func (b *bladeBuilder) roundStart(body *decad.Body, radius float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	rounded, err := decad.Fillet(b.ctx, body, decad.Edges(decad.OuterLoopOf(decad.CapStart(body))), units.Millimeters(radius))
	b.fail("round start", err)
	return rounded
}

// chamferStart chamfers the outer loop of the start cap of an extruded body.
func (b *bladeBuilder) chamferStart(body *decad.Body, size float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	cut, err := decad.Chamfer(b.ctx, body, decad.Edges(decad.OuterLoopOf(decad.CapStart(body))), units.Millimeters(size))
	b.fail("chamfer start", err)
	return cut
}

func (b *bladeBuilder) cut(host, tool *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	body, err := decad.Cut(b.ctx, host, tool)
	b.fail("cut solid", err)
	return body
}

func (b *bladeBuilder) union(a, c *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	body, err := decad.Union(b.ctx, a, c)
	b.fail("union solid", err)
	return body
}

// frame returns the frame at origin with axes u and v, which are unit and perpendicular.
func frame(origin, u, v r3.Vec) r3.Frame {
	f, _ := r3.NewFrame(origin, u, v)
	return f
}
