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

// Params describes one cylindrical gear with straight involute teeth.
// Lengths are millimeters and angles are radians.
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
	// Backlash thins each tooth circumferentially at the pitch circle.
	Backlash float64
	// Placement is the reference point and plane; nil selects the world XY frame.
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

// validate applies the input checks and limits.
func validate(p Params) error {
	switch {
	case p.Module <= 0:
		return fmt.Errorf("module %g must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("pressure angle %g must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("backlash %g must be finite", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("backlash %g exceeds half the module %g", p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("placement is not a valid frame")
	}
	n := float64(p.Teeth)
	pitch := p.Module * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (p.Module*n - 2.5*p.Module) / 2
	tip := (p.Module*n + 2*p.Module) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("root radius %g must be positive", root)
	case base <= 0:
		return fmt.Errorf("base radius %g must be positive", base)
	case tip <= base:
		return fmt.Errorf("tip radius %g must exceed base radius %g", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(fillet) || math.IsInf(fillet, 0) {
		return fmt.Errorf("root fillet radius %g must be finite", fillet)
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

// shifted returns f moved by dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld converts a frame coordinate to a world position.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// DefineParameters writes the parameter table of p into t. It does not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// solveSingleProfile solves and verifies s and returns its one valid profile.
// A positive wantOuter requires that many outer boundary curves.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, stage string,
	wantOuter int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", stage, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, want 1", stage, len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	if wantOuter > 0 && len(profile.Outer) != wantOuter {
		return nil, fmt.Errorf("%s: got %d boundary curves, want %d", stage, len(profile.Outer), wantOuter)
	}
	return profile, nil
}

// Build builds one gear into doc and returns its solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds one gear into doc according to the variant.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spur gear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spur gear: nil document")
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

// assemble builds the gear described by the prepared run.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
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
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("join teeth: result is not a solid")
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
			return nil, err
		}
	}
	return body, nil
}

// buildRootDisc extrudes the root disc on the lower plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root disc radius: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "root disc", 0)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	return disc, nil
}

// BuildProfile builds the whole unfilleted gear outline on plane. ctx, world and
// plane must not be nil.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// BuildTwistedProfile builds the whole unfilleted gear outline with every tooth
// turned by an extra finite twist about the sketch origin. It also returns the
// 2N root contacts: the right root then the left root of each tooth in
// ascending tooth order. ctx, world and plane must not be nil.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("gear profile: nonfinite twist %g", twist)
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
		return fmt.Errorf("gear profile: nil context")
	}
	if world == nil {
		return fmt.Errorf("gear profile: nil world")
	}
	if plane == nil {
		return fmt.Errorf("gear profile: nil plane")
	}
	if err := validate(p); err != nil {
		return fmt.Errorf("gear profile: %w", err)
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
		return nil, nil, nil, fmt.Errorf("create gear outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		tp := addToothPoints(s, left, right, d)
		roots = append(roots, tp.rightRootXY, tp.leftRootXY)
		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, prevLeft, tp.rightRoot)
		}
		if err := drawToothCurves(s, tp); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: %w", k, err)
		}
		prevLeft = tp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	profile, err := solveSingleProfile(ctx, s, "gear outline", 6*p.Teeth)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline builds the whole outline of a gear whose flanks start
// below the root circle. It returns the exact trimmed root contacts.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded outline sketch: %w", err)
	}
	center := s.Origin()
	slots := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPoints := make([]*sketch.Point, len(right))
		leftPoints := make([]*sketch.Point, len(left))
		for i, q := range right {
			rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range left {
			if i == len(left)-1 {
				leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
				continue
			}
			leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		rightSpline, err := s.CreateFitSpline(rightPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPoints[len(rightPoints)-1], leftPoints[len(leftPoints)-1])
		leftSpline, err := s.CreateFitSpline(leftPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		slots[rightSpline] = 2 * k
		slots[leftSpline] = 2*k + 1
	}
	rootCircle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(rootCircle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: verify: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: no bounded region")
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
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union is not a valid hole-free outline")
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, want %d",
			len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, ok := slots[flank]
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: unknown flank spline in boundary")
		}
		tooth := slot / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root trim is not certified", tooth)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank appears twice", tooth)
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
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has no root contact", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius Rc at which an embedded tooth closes.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		ip := involutePoint(d.base, r)
		l := rotate(ip.x, -ip.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / teeth
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("adjacent teeth overlap at the root circle of radius %g", d.root)
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

// buildEmbeddedTooth builds the four-curve section of one embedded tooth.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rightPoints := make([]*sketch.Point, len(right))
	leftPoints := make([]*sketch.Point, len(left))
	for i, q := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	center := s.Origin()
	last := len(rightPoints) - 1
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[0], leftPoints[0])
	profile, err := solveSingleProfile(ctx, s, "embedded tooth", 4)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// filletRoots rounds the ordinary root edges that start at the saved contacts.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// endpointNear returns the position of the first end of e within 1e-6 of one of want.
func endpointNear(e *decad.Edge, want []r3.Vec) (r3.Vec, bool) {
	for _, v := range []*decad.Vertex{e.Start(), e.End()} {
		pos := v.Position().Value
		for _, w := range want {
			if pos.Equal(w, 1e-6) {
				return pos, true
			}
		}
	}
	return r3.Vec{}, false
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	want := make([]r3.Vec, len(roots))
	for i, q := range roots {
		want[i] = toWorld(frame, q)
	}
	var corners []r3.Vec
	for _, e := range edges {
		if pos, ok := endpointNear(e, want); ok {
			corners = append(corners, pos)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("match root edges: got %d, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
	}
	return filleted, nil
}

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile builds the sketch of the bore tool, margin below the lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore parameters: %w", err)
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
	profile, err := solveSingleProfile(ctx, s, "bore tool", 0)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the through-bore from gear using the document's parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
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

// CutThroughBore cuts the through-bore of p from body. body must span z=0
// through the gear thickness along the placement normal. ctx, doc and body must
// not be nil, and p.BoreDiameter must be positive.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cut through bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("cut through bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("cut through bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("cut through bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both caps of gear.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers the outer loops of both caps of gear by distance.
// ctx and gear must not be nil, and distance must be positive and finite.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("chamfer outer caps: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("chamfer outer caps: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer outer caps: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
