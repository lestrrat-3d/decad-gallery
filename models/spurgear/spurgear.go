package spurgear

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params are the spur gear inputs. Lengths are millimeters, angles radians.
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

// DefaultParams returns the default gear inputs.
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

// RejectPlacement returns an error naming model when p carries a Placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
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

// validate applies the contract's input checks and limits.
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
	}
	j := p.Backlash
	if math.IsNaN(j) || math.IsInf(j, 0) || math.Abs(j) > m/2 {
		return fmt.Errorf("backlash %g must be finite and within module/2 (%g)", j, m/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("placement is not a valid frame")
	}
	N := float64(n)
	rp := m * N / 2
	rb := rp * math.Cos(a)
	rr := (m*N - 2.5*m) / 2
	rt := (m*N + 2*m) / 2
	switch {
	case rr <= 0:
		return fmt.Errorf("root radius %g must be positive", rr)
	case rb <= 0:
		return fmt.Errorf("base radius %g must be positive", rb)
	case rt <= rb:
		return fmt.Errorf("tip radius %g must exceed base radius %g", rt, rb)
	case p.BoreDiameter >= 2*rr:
		return fmt.Errorf("bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*rr)
	}
	rf := 0.9 * rr * (math.Pi/N - 2*(math.Tan(a)-a)) / 2
	if math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("root fillet radius %g is not finite", rf)
	}
	return nil
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify sketch: %w", err)
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
		return nil, fmt.Errorf("profile is not valid")
	}
	return profiles[0], nil
}

// flankPoints creates the dimensioned flank points. The right flank gets both
// coordinates everywhere. The left flank gets both except its tip; when
// closed is true its first point also gets one coordinate.
func flankPoints(s *sketch.Sketch, left, right []point, closed bool) ([]*sketch.Point, []*sketch.Point) {
	rightPts := make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 || (closed && i == 0) {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return leftPts, rightPts
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
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

// BuildTwistedProfile draws the gear outline with every tooth turned by an
// extra twist, and returns its 2N root contacts as sketch coordinates.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist %g must be finite", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, r := range roots {
		out[i] = [2]float64{r.x, r.y}
	}
	return s, profile, out, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return fmt.Errorf("profile: nil context")
	}
	if world == nil {
		return fmt.Errorf("profile: nil world")
	}
	if plane == nil {
		return fmt.Errorf("profile: nil plane")
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
	roots := make([]point, 0, 2*p.Teeth)
	var firstRightRoot, prevLeftRoot *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		leftPts, rightPts := flankPoints(s, left, right, false)
		leftRoot, rightRoot := rootPoints(s, left[0], right[0], d)
		roots = append(roots, rootContact(right[0], d), rootContact(left[0], d))
		if k == 0 {
			firstRightRoot = rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[p.Samples-1], leftPts[p.Samples-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeftRoot = leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	return s, profile, roots, nil
}

func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		leftPts, rightPts := flankPoints(s, left, right, false)
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[p.Samples-1], leftPts[p.Samples-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, len(regions))
	for i, r := range regions {
		if !r.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d of %d is not valid", i, len(regions))
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
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, expected %d",
			len(profile.Outer), 4*p.Teeth)
	}
	var contacts []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: boundary curve %d lacks a certified root trim "+
				"(partial %v, exact %v)", i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: boundary curve %d trims at radius %g, expected %g",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if len(contacts) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d root contacts, expected %d",
			len(contacts), 2*p.Teeth)
	}
	// The profile walk may start anywhere. Order contacts by angle from just
	// before tooth zero's right flank so they read right, left per tooth.
	lo := -math.Pi/float64(p.Teeth) + twist
	angleOf := func(c point) float64 {
		a := math.Atan2(c.y, c.x) - lo
		a = math.Mod(a, 2*math.Pi)
		if a < 0 {
			a += 2 * math.Pi
		}
		return a
	}
	sort.SliceStable(contacts, func(i, j int) bool { return angleOf(contacts[i]) < angleOf(contacts[j]) })
	return s, profile, contacts, nil
}

// closingRadius chooses the radius at which the embedded tooth is closed.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankBeta(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		m := mirroredInvolute(d.base, r)
		l := rotate(m.x, m.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / teeth
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: adjacent flanks overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) > limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws the four-curve natural tooth of the embedded branch.
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
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	leftPts, rightPts := flankPoints(s, left, right, true)
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[p.Samples-1], leftPts[p.Samples-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d outer curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// Build constructs the gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear into doc, using the variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("build gear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("build gear: nil document")
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

// assemble builds the tooth, root disc, pattern, join, fillet, bore and chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("build gear: tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("build gear: root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("build gear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("build gear: join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("build gear: joined body is not solid")
	}
	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
		if err != nil {
			return nil, fmt.Errorf("build gear: root fillet: %w", err)
		}
	} else if filletRadius(r.p, r.d) > 0 {
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("build gear: root fillet: %w", err)
		}
	}
	if r.p.BoreDiameter > 0 {
		body, err = cutBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, fmt.Errorf("build gear: bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("build gear: read cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, fmt.Errorf("build gear: cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("build gear: final body is not solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root-disc circle bound to the root_radius entry.
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
	body, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return body, nil
}

// filletRoots fillets the ordinary root edges matched to roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, target := range targets {
			if start.Sub(target).Len() <= 1e-6 {
				corners = append(corners, start)
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
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return out, nil
}

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the gear.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	low, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: shift frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(low)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore tool: bind bore_diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cut bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("cut bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("cut bore: nil body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("cut bore: bore diameter %g must be positive and finite", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool and cuts it from gear. doc must hold the
// parameter table of p.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read bore_tool_length: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore tool: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("chamfer caps: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("chamfer caps: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer caps: setback %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return out, nil
}
