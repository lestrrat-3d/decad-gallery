// Package spurgear builds a cylindrical gear with straight involute teeth.
//
// The gear is one root disc and one tooth repeated about the gear axis and
// joined into one solid, finished with root fillets, an optional through-bore
// and optional cap chamfers.
package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimetres and angles radians.
type Params struct {
	// Module is the tooth module.
	Module float64
	// Teeth is the tooth count.
	Teeth int
	// PressureAngle is the pressure angle.
	PressureAngle float64
	// Thickness is the axial thickness.
	Thickness float64
	// Samples is the number of fit points per flank.
	Samples int
	// BoreDiameter is the centered through-bore diameter; zero omits the bore.
	BoreDiameter float64
	// CapChamfer is the equal-distance outer cap setback; zero omits the chamfer.
	CapChamfer float64
	// Backlash is the circumferential tooth thinning at the pitch circle.
	Backlash float64
	// Placement is the reference point and plane; nil selects the world XY
	// frame at the origin.
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

// RejectPlacement returns an error naming model when p carries a Placement.
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

func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// validate applies the input checks and limits.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("spurgear: module %v must be positive", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("spurgear: thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("spurgear: pressure angle %v must lie in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	case !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash %v must be finite with magnitude at most module/2 (%v)",
			p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return fmt.Errorf("spurgear: placement is not a valid frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return fmt.Errorf("spurgear: root radius %v must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("spurgear: base radius %v must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*d.root)
	case !finite(d.fillet):
		return fmt.Errorf("spurgear: root fillet radius %v is not finite", d.fillet)
	}
	return nil
}

// solveVerify solves s and requires it fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// onlyProfile requires s to hold exactly one valid profile and returns it.
func onlyProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// Build builds the gear into doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear described by p into doc.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
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

// assemble builds tooth zero, the root disc, the pattern and the join, then
// applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: joined gear is not one solid")
	}

	embedded := r.d.base < r.d.root
	switch {
	case v.RootFillet != nil:
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	case r.d.fillet > 0:
		body, err = filletSpurRoots(ctx, r.doc, body, r, embedded)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	}

	if r.p.BoreDiameter > 0 {
		body, err = cutBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read parameter %q: %w", "cap_chamfer", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: finished gear is not one solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root disc on the lower cap plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: root disc: bind radius: %w", err)
	}
	if err := solveVerify(ctx, s, "spurgear: root disc"); err != nil {
		return nil, err
	}
	profile, err := onlyProfile(s, "spurgear: root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	d := derive(p)
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole-gear outline with every tooth turned by
// twist about the sketch origin. It also returns the 2N root contacts, right
// root then left root of each tooth.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist %v is not finite", twist)
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	d := derive(p)
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, rt := range roots {
		out[i] = [2]float64{rt.x, rt.y}
	}
	return s, profile, out, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return fmt.Errorf("spurgear: nil world")
	}
	if plane == nil {
		return fmt.Errorf("spurgear: nil plane")
	}
	return nil
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	roots := make([]point, 0, 2*p.Teeth)
	var firstRightRoot, prevLeftRoot *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		tp := addToothPoints(s, left, right, d)
		roots = append(roots, tp.rightRootAt, tp.leftRootAt)
		if k == 0 {
			firstRightRoot = tp.rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, tp.rightRoot)
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, tp.right[last], tp.left[last])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		prevLeftRoot = tp.leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)

	if err := solveVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// flankAngle is the angle between the tooth axis and the left flank where it
// meets radius r.
func flankAngle(r float64, p Params, d dimensions) float64 {
	beta := flankBeta(d.pitch, d.base, float64(p.Teeth), d.backlash)
	m := mirroredInvolute(d.base, r)
	q := rotate(m.x, m.y, beta)
	return math.Atan2(q.y, q.x)
}

// closingRadius chooses the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	limit := math.Pi / float64(p.Teeth)
	if flankAngle(d.root, p, d) >= limit {
		return 0, fmt.Errorf("embedded tooth: flanks of adjacent teeth still meet at the root circle radius %g",
			d.root)
	}
	if flankAngle(d.base, p, d) < limit {
		return d.base, nil
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if flankAngle(mid, p, d) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedOutline draws the whole-gear outline of an embedded gear: the
// original flanks and tip arcs plus one root circle, with all regions united.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	type flankRef struct {
		tooth int
		right bool
	}
	flanks := make(map[*sketch.FitSpline]flankRef, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		rp := make([]*sketch.Point, p.Samples)
		lp := make([]*sketch.Point, p.Samples)
		for i, pt := range right {
			rp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
		for i, pt := range left {
			if i == last {
				lp[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
				continue
			}
			lp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[last], lp[last])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank spline: %w", k, err)
		}
		flanks[rs] = flankRef{k, true}
		flanks[ls] = flankRef{k, false}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: sketch has no region")
	}
	indices := make([]int, len(regions))
	for i, reg := range regions {
		if !reg.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}

	roots := make([]point, 2*p.Teeth)
	found := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		ref, ok := flanks[flank]
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: boundary spline is not a drawn flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root trim is not certified", ref.tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d trim lies off the root circle", ref.tooth)
		}
		idx := 2 * ref.tooth
		if !ref.right {
			idx++
		}
		roots[idx] = point{x, y}
		found[idx] = true
	}
	n := 0
	for _, ok := range found {
		if ok {
			n++
		}
	}
	if n != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d root contacts, expected %d", n, 2*p.Teeth)
	}
	return s, profile, roots, nil
}

// buildEmbeddedTooth draws one embedded tooth: right flank, tip arc, left
// flank and the arc about the gear axis at the closing radius.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rp := make([]*sketch.Point, p.Samples)
	lp := make([]*sketch.Point, p.Samples)
	for i, pt := range right {
		rp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	for i, pt := range left {
		if i == 0 || i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		lp[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])

	if err := solveVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d boundary curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges matching roots, with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges whose lower-cap end lies at one of
// roots.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	const tol = 1e-6
	targets := make([]r3.Vec, len(roots))
	for i, rt := range roots {
		targets[i] = toWorld(frame, rt)
	}
	matches := func(pos r3.Vec) bool {
		for _, t := range targets {
			if pos.Sub(t).Len() <= tol {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, e := range edges {
		if pos := e.Start().Position().Value; matches(pos) {
			corners = append(corners, pos)
		} else if pos := e.End().Position().Value; matches(pos) {
			corners = append(corners, pos)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root edges: matched %d, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, fmt.Errorf("root edges: no root contacts to fillet")
	}
	sel := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2*teeth concave axial edges.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, sel, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the circular bore tool section on a plane margin
// below the gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: shift placement: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore tool: bind diameter: %w", err)
	}
	if err := solveVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore extrudes the bore tool and cuts it from gear. It reads the
// document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("bore: read parameter %q: %w", "tool_margin", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore: read parameter %q: %w", "bore_tool_length", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, marginValue.Base())
	if err != nil {
		return nil, fmt.Errorf("bore: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore: cut tool: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through the gear thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("spurgear: bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore: diameter %v must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both cap faces by setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer caps: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers the outer loops of both cap faces of gear.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: chamfer: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: chamfer: nil body")
	}
	if !(distance > 0) || !finite(distance) {
		return nil, fmt.Errorf("spurgear: chamfer: distance %v must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
