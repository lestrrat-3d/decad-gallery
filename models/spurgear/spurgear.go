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

// Params describes one straight involute spur gear. Lengths are in millimeters and angles
// in radians. A nil Placement selects the world XY frame at the origin.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	Placement     *r3.Frame
}

// DefaultParams returns the default 17-tooth, module 1 gear.
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

// DefineParameters writes the parameter table of p into t. It does not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

func finite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

// validate applies the input checks and limits shared by every entry point.
func validate(p Params) error {
	if !finite(p.Module) || p.Module <= 0 {
		return fmt.Errorf("spurgear: module must be positive and finite, got %v", p.Module)
	}
	if !finite(p.Thickness) || p.Thickness <= 0 {
		return fmt.Errorf("spurgear: thickness must be positive and finite, got %v", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	}
	if !finite(p.PressureAngle) || p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("spurgear: pressure angle must lie in (0, pi/2), got %v", p.PressureAngle)
	}
	if !finite(p.BoreDiameter) || p.BoreDiameter < 0 {
		return fmt.Errorf("spurgear: bore diameter must be nonnegative and finite, got %v", p.BoreDiameter)
	}
	if !finite(p.CapChamfer) || p.CapChamfer < 0 {
		return fmt.Errorf("spurgear: cap chamfer must be nonnegative and finite, got %v", p.CapChamfer)
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("spurgear: backlash must be finite with magnitude at most module/2, got %v", p.Backlash)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("spurgear: invalid placement frame")
	}
	n := float64(p.Teeth)
	root := (p.Module*n - 2.5*p.Module) / 2
	base := p.Module * n / 2 * math.Cos(p.PressureAngle)
	tip := (p.Module*n + 2*p.Module) / 2
	if root <= 0 {
		return fmt.Errorf("spurgear: root radius %v must be positive", root)
	}
	if base <= 0 {
		return fmt.Errorf("spurgear: base radius %v must be positive", base)
	}
	if tip <= base {
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", tip, base)
	}
	if p.BoreDiameter >= 2*root {
		return fmt.Errorf("spurgear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*root)
	}
	involute := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*involute) / 2
	if !finite(fillet) {
		return fmt.Errorf("spurgear: root fillet radius %v is not finite", fillet)
	}
	return nil
}

// placement returns the reference frame of p.
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

// toWorld maps frame coordinates onto the world.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// solveSingleProfile solves and verifies s, then returns its one valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, what string) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve %s: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify %s: %w", what, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s has %d profiles, expected 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s profile is not valid", what)
	}
	return profiles[0], nil
}

// flankSketchPoints creates the dimensioned points of one tooth's flanks. Every right point
// takes both dimensions. Every left point does too, except the tip and, when arcFirst is set,
// the first point, which take one dimension.
func flankSketchPoints(s *sketch.Sketch, left, right []point, arcFirst bool) ([]*sketch.Point, []*sketch.Point) {
	rightPts := make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 || (arcFirst && i == 0) {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	return rightPts, leftPts
}

// buildGearProfile draws the whole unfilleted outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole outline with every tooth turned by an extra twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeftRoot, firstRightRoot *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		angle := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
		rightPts, leftPts := flankSketchPoints(s, left, right, false)
		leftRootXY := rootContact(left[0], d)
		rightRootXY := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
		rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)
		roots = append(roots, rightRootXY, leftRootXY)

		if k == 0 {
			firstRightRoot = rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[last], leftPts[last])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeftRoot = leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)

	profile, err := solveSingleProfile(ctx, s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline has %d curves, expected %d", len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// flankOwner identifies the tooth and side of a flank spline.
type flankOwner struct {
	slot int
}

// buildEmbeddedOutline draws the whole outline of an embedded gear: the original flanks and
// tip arcs of every tooth plus one root circle, united into one profile. It returns the
// trimmed root contacts, right then left for each tooth.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	owners := make(map[*sketch.FitSpline]flankOwner, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		angle := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
		rightPts, leftPts := flankSketchPoints(s, left, right, false)
		rs, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[last], leftPts[last])
		ls, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		owners[rs] = flankOwner{slot: 2 * k}
		owners[ls] = flankOwner{slot: 2*k + 1}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded gear outline: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded gear outline: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded gear outline has no region")
	}
	indices := make([]int, len(regions))
	for i, r := range regions {
		if !r.Valid {
			return nil, nil, nil, fmt.Errorf("embedded gear outline region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded gear regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("embedded gear outline is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded gear outline has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded gear outline has %d curves, expected %d", len(profile.Outer), want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		owner, known := owners[spline]
		if !known {
			return nil, nil, nil, fmt.Errorf("embedded gear outline has an unknown flank edge")
		}
		tooth := owner.slot / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth %d flank trim is not certified", tooth)
		}
		if filled[owner.slot] {
			return nil, nil, nil, fmt.Errorf("tooth %d flank appears twice in the outline", tooth)
		}
		x, y := spline.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d flank trim lies at radius %v, expected %v",
				tooth, math.Hypot(x, y), d.root)
		}
		contacts[owner.slot] = point{x, y}
		filled[owner.slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("tooth %d flank has no trimmed root contact", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// embeddedHalfAngle returns the angle between the tooth axis and the left flank at radius r.
func embeddedHalfAngle(d dimensions, teeth float64, r float64) float64 {
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	ip := involutePoint(d.base, r)
	l := rotate(ip.x, ip.y, beta)
	return math.Atan2(l.y, l.x)
}

// embeddedClosingRadius returns the radius of the arc that closes an embedded tooth.
func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	if embeddedHalfAngle(d, teeth, d.base) < limit {
		return d.base, nil
	}
	if embeddedHalfAngle(d, teeth, d.root) >= limit {
		return 0, fmt.Errorf("no closing radius exists inside the root circle of radius %v", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if embeddedHalfAngle(d, teeth, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws one natural tooth of an embedded gear, closed by an arc just
// inside the root circle.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rightPts, leftPts := flankSketchPoints(s, left, right, true)
	last := p.Samples - 1
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])

	profile, err := solveSingleProfile(ctx, s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth has %d outer curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildProfile draws the whole unfilleted outline of the gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole outline with every tooth turned by twist about the
// sketch origin, and returns its root contacts, right then left for each tooth.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params,
	twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist must be finite")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, profile, out, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return fmt.Errorf("spurgear: nil sketch world")
	}
	if plane == nil {
		return fmt.Errorf("spurgear: nil sketch plane")
	}
	return validate(p)
}

// Build builds the gear as one solid body in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear with the variant's replacements.
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

// assemble builds the disc, joins the pattern of teeth and applies the finishing steps.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: joined gear is not solid")
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
			return nil, fmt.Errorf("spurgear: bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: finished gear is not solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root circle on the lower plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: bind root radius: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "root disc")
	if err != nil {
		return nil, fmt.Errorf("spurgear: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}
	return disc, nil
}

// filletRoots rounds the ordinary root edges with a radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue rounds the axial edges that start or end on a saved root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	const tol = 1e-6
	near := func(a, b r3.Vec) bool {
		return a.Sub(b).Len() <= tol
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		end := edge.End().Position().Value
		for _, root := range roots {
			w := toWorld(frame, root)
			if near(start, w) {
				corners = append(corners, start)
				break
			}
			if near(end, w) {
				corners = append(corners, end)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return filleted, nil
}

// filletEmbedded rounds the concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("define bore parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the parameters the document already holds.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read tool_margin: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from an assembled gear.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore diameter must be positive, got %v", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: define bore parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both caps with the given setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers both outer cap loops of an assembled gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance must be positive and finite, got %v", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
