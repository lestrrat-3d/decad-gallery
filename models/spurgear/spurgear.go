package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one straight involute spur gear. Lengths are millimeters
// and angles are radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	// Placement gives the reference point and plane; nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
		Backlash:      0.05,
	}
}

// RejectPlacement returns an error naming model when p carries a placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// DefineParameters writes the gear parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// validate applies the input checks and limits.
func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("spurgear: module %g must be positive", p.Module)
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("spurgear: thickness %g must be positive", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("spurgear: pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("spurgear: bore diameter %g must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("spurgear: cap chamfer %g must not be negative", p.CapChamfer)
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) {
		return fmt.Errorf("spurgear: backlash %g must be finite", p.Backlash)
	}
	if math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("spurgear: backlash %g exceeds half the module %g", p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: placement is not a valid frame")
	}
	d := derive(p)
	if d == (dimensions{}) {
		return errors.New("spurgear: cannot derive gear dimensions")
	}
	if d.root <= 0 {
		return fmt.Errorf("spurgear: root radius %g must be positive", d.root)
	}
	if d.base <= 0 {
		return fmt.Errorf("spurgear: base radius %g must be positive", d.base)
	}
	if d.tip <= d.base {
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("spurgear: bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*d.root)
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("spurgear: root fillet radius %g must be finite", d.fillet)
	}
	return nil
}

// placement returns the placement frame of p.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted returns f moved dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a frame coordinate to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// Build builds the gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear into doc through the variant pipeline.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	if v.LocalBuild {
		return v.buildLocal(ctx, doc, p)
	}
	r, err := v.begin(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return v.assemble(ctx, r)
}

// assemble builds the tooth, the root disc, the pattern and join, then the
// finishing operations.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}

	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("root disc radius binding: %w", err)
	}
	if err := solveVerify(ctx, s); err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	discProfile, err := singleProfile(s)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, discProfile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root disc extrude: %w", err)
	}

	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("tooth pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("gear union: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("gear union: result is not a solid")
	}

	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
	} else if r.d.fillet > 0 {
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}

	if r.p.BoreDiameter > 0 {
		body, err = cutBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, fmt.Errorf("bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("parameter %s: %w", "cap_chamfer", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("gear: result is not a solid")
	}
	return body, nil
}

// solveVerify solves s and requires a fully constrained verification report.
func solveVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// singleProfile returns the one valid profile of s.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("got %d profiles, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is not valid")
	}
	return profiles[0], nil
}

// flankPoints holds the dimensioned sample points of one tooth.
type flankPoints struct {
	right, left []*sketch.Point
}

// drawFlankPoints creates the dimensioned flank sample points. Every point
// receives both coordinate dimensions except the left tip, which receives one.
func drawFlankPoints(s *sketch.Sketch, left, right []point) flankPoints {
	var fp flankPoints
	for _, q := range right {
		fp.right = append(fp.right, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	last := len(left) - 1
	for i, q := range left {
		if i == last {
			fp.left = append(fp.left, sketchcheck.DimensionArcEnd(s, q.x, q.y))
		} else {
			fp.left = append(fp.left, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
	}
	return fp
}

// rootPoints holds the root contacts of one ordinary tooth.
type rootPoints struct {
	right, left     *sketch.Point
	rightXY, leftXY point
}

// drawRoots creates the root points by scaling the first flank samples onto the
// root circle. The left root is created first and receives one dimension.
func drawRoots(s *sketch.Sketch, d dimensions, leftBase, rightBase point) rootPoints {
	var rp rootPoints
	rp.leftXY = rootContact(leftBase, d)
	rp.rightXY = rootContact(rightBase, d)
	rp.left = sketchcheck.DimensionArcEnd(s, rp.leftXY.x, rp.leftXY.y)
	rp.right = sketchcheck.DimensionPoint(s, rp.rightXY.x, rp.rightXY.y)
	return rp
}

// BuildProfile builds the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// BuildTwistedProfile builds the whole-gear outline with every tooth turned by
// twist, together with its 2N root contacts.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist %g must be finite", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	contacts := make([][2]float64, len(roots))
	for i, q := range roots {
		contacts[i] = [2]float64{q.x, q.y}
	}
	return s, profile, contacts, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return errors.New("spurgear: nil context")
	}
	if world == nil {
		return errors.New("spurgear: nil sketch world")
	}
	if plane == nil {
		return errors.New("spurgear: nil sketch plane")
	}
	return nil
}

// buildGearProfile builds the whole-gear outline without twist.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with every tooth angle
// increased by twist. It returns the 2N root contacts, right then left per tooth.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		last := len(left) - 1
		fp := drawFlankPoints(s, left, right)
		rp := drawRoots(s, d, left[0], right[0])
		roots = append(roots, rp.rightXY, rp.leftXY)

		if k == 0 {
			firstRight = rp.right
		} else {
			s.CreateArc(center, prevLeft, rp.right)
		}
		s.CreateLine(rp.right, fp.right[0])
		if _, err := s.CreateFitSpline(fp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, fp.right[last], fp.left[last])
		if _, err := s.CreateFitSpline(fp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(fp.left[0], rp.left)
		prevLeft = rp.left
	}
	s.CreateArc(center, prevLeft, firstRight)

	if err := solveVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// flankOwner names the tooth and side of a fit spline.
type flankOwner struct {
	tooth int
	right bool
}

// buildEmbeddedOutline builds the whole-gear outline when the base circle lies
// inside the root circle.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline sketch: %w", err)
	}
	center := s.Origin()
	owners := make(map[*sketch.FitSpline]flankOwner, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		last := len(left) - 1
		fp := drawFlankPoints(s, left, right)
		rs, err := s.CreateFitSpline(fp.right...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		owners[rs] = flankOwner{tooth: k, right: true}
		s.CreateArc(center, fp.right[last], fp.left[last])
		ls, err := s.CreateFitSpline(fp.left...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		owners[ls] = flankOwner{tooth: k}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, errors.New("embedded outline: no regions")
	}
	indices := make([]int, len(profiles))
	for i, region := range profiles {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline union: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		owner, known := owners[flank]
		if !known {
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root trim is not certified",
				owner.tooth)
		}
		slot := 2*owner.tooth + 1
		if owner.right {
			slot = 2 * owner.tooth
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has a duplicate root trim",
				owner.tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact is off the root circle",
				owner.tooth)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has a missing root trim", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// buildEmbeddedTooth builds the four-curve tooth section used when the base
// circle lies inside the root circle.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	last := len(left) - 1
	rightPts := make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == 0 || i == last {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])

	if err := solveVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d boundary curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// filletRoots rounds the ordinary root edges with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue rounds the axial edges that start at a saved root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, q := range roots {
		world[i] = toWorld(frame, q)
	}
	near := func(pos r3.Vec) bool {
		for _, w := range world {
			if pos.Sub(w).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, e := range edges {
		start := e.Start().Position().Value
		if near(start) {
			corners = append(corners, start)
			continue
		}
		end := e.End().Position().Value
		if near(end) {
			corners = append(corners, end)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return out, nil
}

// filletEmbedded rounds the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the sketch of the through-bore tool, margin below the
// lower cap plane.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore parameters: %w", err)
	}
	base, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore placement: %w", err)
	}
	frame, err := shifted(base, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore diameter binding: %w", err)
	}
	if err := solveVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts the through-bore from body using the document's existing
// parameter table.
func cutBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter %s: %w", "tool_margin", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter %s: %w", "bore_tool_length", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool extrude: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which spans z=0 through the thickness along the placement normal.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	if body == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops of gear with the given setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
