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
	// Placement anchors the gear; nil selects the world XY frame at the origin.
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

// validate applies the input checks and limits.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("module %g must be positive", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	case !(p.BoreDiameter >= 0):
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case !(p.CapChamfer >= 0):
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("backlash %g must be finite and within module/2", p.Backlash)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("invalid placement frame")
	}
	n := float64(p.Teeth)
	rp := p.Module * n / 2
	rb := rp * math.Cos(p.PressureAngle)
	rr := (p.Module*n - 2.5*p.Module) / 2
	rt := (p.Module*n + 2*p.Module) / 2
	switch {
	case rr <= 0:
		return fmt.Errorf("root radius %g must be positive", rr)
	case rb <= 0:
		return fmt.Errorf("base radius %g must be positive", rb)
	case rt <= rb:
		return fmt.Errorf("tip radius %g must exceed base radius %g", rt, rb)
	case p.BoreDiameter >= 2*rr:
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*rr)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	rf := 0.9 * rr * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("root fillet radius %g must be finite", rf)
	}
	return nil
}

// DefineParameters writes the parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// readDimensions reads the derived dimensions from the document's table.
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	targets := []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch},
		{"base_radius", &d.base},
		{"root_radius", &d.root},
		{"tip_radius", &d.tip},
		{"fillet_radius", &d.fillet},
		{"backlash", &d.backlash},
	}
	for _, t := range targets {
		v, err := doc.Param(t.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read parameter %s: %w", t.name, err)
		}
		*t.dst = v.Base()
	}
	return d, nil
}

// placement returns the placement frame of p, or the world XY frame.
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

// toWorld maps a frame coordinate to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// solveSingleProfile solves and verifies s, then returns its one valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, what string) (*sketch.Profile, error) {
	if err := solveVerified(ctx, s, what); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, expected 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", what)
	}
	return profiles[0], nil
}

// solveVerified solves s and requires it to be fully constrained.
func solveVerified(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", what, err)
	}
	return nil
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
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile builds the whole-gear outline with every tooth turned by
// twist, and returns its 2N root contacts (right then left of each tooth).
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, errors.New("twisted profile: twist must be finite")
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
		return errors.New("profile: nil context")
	}
	if world == nil {
		return errors.New("profile: nil world")
	}
	if plane == nil {
		return errors.New("profile: nil plane")
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
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRightRoot, prevLeftRoot *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		g := newToothGeometry(s, d, p.Teeth, p.Samples, toothAngle(p, k)+twist)
		roots = append(roots, g.rootR, g.rootL)
		if k == 0 {
			firstRightRoot = g.rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, g.rightRoot)
		}
		if err := g.drawFlanks(s); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: %w", k, err)
		}
		prevLeftRoot = g.leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)
	profile, err := solveSingleProfile(ctx, s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline builds the whole-gear outline when the base circle lies
// below the root circle: original flanks and tip arcs, trimmed by the root circle.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline sketch: %w", err)
	}
	center := s.Origin()
	slots := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rp := make([]*sketch.Point, len(right))
		lp := make([]*sketch.Point, len(left))
		for i, q := range right {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range left {
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		slots[rs] = 2 * k
		slots[ls] = 2*k + 1
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerified(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, errors.New("embedded outline: no bounded region")
	}
	indices := make([]int, len(profiles))
	for i, pr := range profiles {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, errors.New("embedded outline: union is not a valid hole-free region")
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
		slot, known := slots[flank]
		if !known {
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth")
		}
		tooth := slot / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank trim is not certified", tooth)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank appears twice", tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact radius %g, expected %g",
				tooth, math.Hypot(x, y), d.root)
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

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	beta := flankRotation(d.pitch, d.base, float64(p.Teeth), d.backlash)
	half := func(r float64) float64 {
		m := involutePoint(d.base, r)
		l := rotate(m.x, m.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / float64(p.Teeth)
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, errors.New("embedded tooth: adjacent flanks overlap at the root circle")
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) >= limit {
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
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rp := make([]*sketch.Point, len(right))
	lp := make([]*sketch.Point, len(left))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank spline: %w", err)
	}
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank spline: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	profile, err := solveSingleProfile(ctx, s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d boundary curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// Build builds one gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds one gear into doc, with the variant's hooks applied.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spur gear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spur gear: nil document")
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

// assemble builds the tooth, the root disc, the pattern and join, and the
// finishing features of a prepared run.
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
	profile, err := solveSingleProfile(ctx, s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
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
		return nil, fmt.Errorf("tooth union: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("tooth union: result is not a solid")
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
			return nil, fmt.Errorf("parameter cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// filletRoots fillets the ordinary-branch root edges that match roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end at a saved root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	near := func(pos r3.Vec) bool {
		for _, root := range roots {
			if toWorld(frame, root).Sub(pos).Len() <= 1e-6 {
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
	if len(corners) != len(roots) || len(corners) == 0 {
		return nil, fmt.Errorf("root fillet: matched %d root edges, expected %d", len(corners), len(roots))
	}
	sel := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, sel, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the verified circular bore tool sketch, starting
// margin below the gear's lower cap.
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
		return nil, nil, fmt.Errorf("bore tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore diameter binding: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter tool_margin: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool extrude: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter in body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("bore: nil document")
	}
	if body == nil {
		return nil, errors.New("bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// ChamferOuterCaps chamfers the outer loops of both caps of gear.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loop of each cap. The caller checks arguments.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}
