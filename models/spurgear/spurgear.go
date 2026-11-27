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

// Params describes one cylindrical gear with straight involute teeth.
// Lengths are in millimeters and angles in radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	// Placement gives the reference point and plane; nil selects the world
	// XY frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default gear parameters.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
		BoreDiameter:  0,
		CapChamfer:    0,
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

// DefineParameters writes the parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
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

// toWorld maps a frame coordinate on the frame's plane to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits.
func validate(p Params) error {
	m, n, a := p.Module, p.Teeth, p.PressureAngle
	switch {
	case m <= 0:
		return fmt.Errorf("module %g must be positive", m)
	case p.Thickness <= 0:
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	case n < 3:
		return fmt.Errorf("teeth %d must be at least 3", n)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case a <= 0 || a >= math.Pi/2:
		return fmt.Errorf("pressure angle %g must be in (0, pi/2)", a)
	case p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > m/2:
		return fmt.Errorf("backlash %g must be finite and within +-module/2", p.Backlash)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("placement is not a valid frame")
	}
	tn := float64(n)
	rp := m * tn / 2
	rb := rp * math.Cos(a)
	rr := (m*tn - 2.5*m) / 2
	rt := (m*tn + 2*m) / 2
	rf := 0.9 * rr * (math.Pi/tn - 2*(math.Tan(a)-a)) / 2
	switch {
	case rr <= 0:
		return fmt.Errorf("root radius %g must be positive", rr)
	case rb <= 0:
		return fmt.Errorf("base radius %g must be positive", rb)
	case rt <= rb:
		return fmt.Errorf("tip radius %g must exceed base radius %g", rt, rb)
	case p.BoreDiameter >= 2*rr:
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*rr)
	case math.IsNaN(rf) || math.IsInf(rf, 0):
		return fmt.Errorf("root fillet radius %g must be finite", rf)
	}
	return nil
}

// solveAndVerify solves s and requires a fully constrained sketch.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
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
		return nil, fmt.Errorf("profile count %d, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is not valid")
	}
	return profiles[0], nil
}

// Build constructs the gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear in doc through the variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("build spur gear: nil context")
	}
	if doc == nil {
		return nil, errors.New("build spur gear: nil document")
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

// assemble extrudes the root disc, patterns the tooth, joins them and applies
// the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	copies, err := decad.PatternCopies(tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("tooth pattern: %w", err)
	}
	body, err := decad.Union(append([]*decad.Body{disc, tooth}, copies...)...).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("join: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join: result is not a solid")
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
			return nil, fmt.Errorf("read parameter cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, fmt.Errorf("cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("gear is not a solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root disc on the lower cap plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root_radius: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(r.doc, s, profile, r.extent).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile returns the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	s, profile, _, err := BuildTwistedProfile(ctx, world, plane, p, 0)
	return s, profile, err
}

// BuildTwistedProfile returns the whole-gear outline with every tooth turned
// by an extra twist, together with its 2N root contacts: the right root then
// the left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, errors.New("build profile: nil context, world or plane")
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("build profile: twist %g must be finite", twist)
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, fmt.Errorf("build profile: %w", err)
	}
	d := derive(p)
	if d == (dimensions{}) {
		return nil, nil, nil, errors.New("build profile: cannot derive dimensions")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	contacts := make([][2]float64, len(roots))
	for i, c := range roots {
		contacts[i] = [2]float64{c.x, c.y}
	}
	return s, profile, contacts, nil
}

// buildGearProfile builds the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with every tooth
// turned by an extra twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	teeth := float64(p.Teeth)
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, teeth, d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		lp, rp, lRoot, rRoot, lc, rc := drawFlankPoints(s, left, right, d)
		if k == 0 {
			firstRight = rRoot
		} else {
			s.CreateArc(center, lastLeft, rRoot)
		}
		s.CreateLine(rRoot, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], lRoot)
		lastLeft = lRoot
		roots = append(roots, rc, lc)
	}
	s.CreateArc(center, lastLeft, firstRight)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: boundary curves %d, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// drawFlankPoints creates the dimensioned flank and root points of one
// ordinary tooth. It returns the left and right flank points, the left and
// right roots and their coordinates.
func drawFlankPoints(s *sketch.Sketch, left, right []point, d dimensions) (
	lp, rp []*sketch.Point, lRoot, rRoot *sketch.Point, lc, rc point) {
	last := len(left) - 1
	rp = make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp = make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	lc = rootContact(left[0], d)
	rc = rootContact(right[0], d)
	lRoot = sketchcheck.DimensionArcEnd(s, lc.x, lc.y)
	rRoot = sketchcheck.DimensionPoint(s, rc.x, rc.y)
	return lp, rp, lRoot, rRoot, lc, rc
}

// buildEmbeddedOutline builds the whole-gear outline when the base circle lies
// inside the root circle. It returns the exact trimmed root contacts.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	teeth := float64(p.Teeth)
	rights := make([]*sketch.FitSpline, p.Teeth)
	lefts := make([]*sketch.FitSpline, p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, teeth, d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		last := len(left) - 1
		rp := make([]*sketch.Point, len(right))
		for i, q := range right {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		lp := make([]*sketch.Point, len(left))
		for i, q := range left {
			if i == last {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		if rights[k], err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[last], lp[last])
		if lefts[k], err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("embedded outline: no bounded region")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
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
		return nil, nil, nil, errors.New("embedded outline: union is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: boundary curves %d, expected %d",
			len(profile.Outer), want)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, tooth := -1, -1
		for k := 0; k < p.Teeth; k++ {
			switch flank {
			case rights[k]:
				slot, tooth = 2*k, k
			case lefts[k]:
				slot, tooth = 2*k+1, k
			}
		}
		if slot < 0 {
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root trim is not certified", tooth)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has a duplicate root trim", tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact is off the root circle", tooth)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact %d is missing", slot/2, slot)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius Rc of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	h := func(r float64) float64 { return halfAngle(d, teeth, r) }
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: neighbouring teeth overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if h(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth builds the four-curve section of the embedded tooth.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	last := len(left) - 1
	rp := make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == 0 || i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: boundary curves %d, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary-branch root edges with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end at a saved root
// contact on the lower cap.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	const tol = 1e-6
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	near := func(pos r3.Vec) bool {
		for _, t := range targets {
			if pos.Sub(t).Len() <= tol {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, e := range edges {
		if pos := e.Start().Position().Value; near(pos) {
			corners = append(corners, pos)
		} else if pos := e.End().Position().Value; near(pos) {
			corners = append(corners, pos)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root edges matched %d, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(body, selection.Exactly(len(corners)), radius).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(body, selection, radius).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile builds the bore tool sketch on a plane margin below the
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
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore_diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the document's existing parameters.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body,
	p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read parameter bore_tool_length: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(doc, s, profile, decad.Distance{D: length, Dir: decad.Along}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(gear, tool).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the lower cap plane through p.Thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, errors.New("cut through bore: nil context, document or body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("cut through bore: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops with setback. It checks nothing.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(gear, selection, setback).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, errors.New("chamfer outer caps: nil context or body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer outer caps: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
