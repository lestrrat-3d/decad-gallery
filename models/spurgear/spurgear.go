// Package spurgear builds a cylindrical gear with straight involute teeth.
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

// Params describes one spur gear. Lengths are millimeters, angles radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	// Placement is the reference point and plane; nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default gear.
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

func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// validate applies the input checks shared by every public entry point.
func validate(p Params) error {
	if !finite(p.Module, p.PressureAngle, p.Thickness, p.BoreDiameter, p.CapChamfer) {
		return fmt.Errorf("spurgear: module, pressure angle, thickness, bore and chamfer must be finite")
	}
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spurgear: module %g must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spurgear: thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spurgear: pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %g must not be negative", p.CapChamfer)
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("spurgear: backlash %g must be finite and within +/- module/2 (%g)",
			p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("spurgear: placement is not a valid frame")
	}
	n := float64(p.Teeth)
	pitch := p.Module * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (p.Module*n - 2.5*p.Module) / 2
	tip := (p.Module*n + 2*p.Module) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("spurgear: root radius %g must be positive", root)
	case base <= 0:
		return fmt.Errorf("spurgear: base radius %g must be positive", base)
	case tip <= base:
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("spurgear: bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*root)
	}
	involute := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*involute) / 2
	if !finite(fillet) {
		return fmt.Errorf("spurgear: root fillet radius %g is not finite", fillet)
	}
	return nil
}

// defineParameters writes the gear's parameter table into t.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("spurgear: parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("spurgear: parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("spurgear: parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("spurgear: parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("spurgear: parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("spurgear: parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("spurgear: parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("spurgear: parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("spurgear: parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter bore_tool_length: %w", err)
	}
	return nil
}

// DefineParameters writes the Parameter table of p into t without validating p.
func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }

// Build constructs the gear in doc and returns its solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear, delegating to the local pipeline when LocalBuild is set.
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

// assemble extrudes the root disc and one tooth, patterns the tooth, joins
// everything, and applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
	}
	disc, err := extrudeRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern tooth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join disc and %d teeth: %w", r.teeth, err)
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("spurgear: joined gear is not a solid")
	}
	switch {
	case v.RootFillet != nil:
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
	case r.d.fillet > 0:
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	if r.p.BoreDiameter > 0 {
		if gear, err = cutBore(ctx, r.doc, gear, r.p); err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read cap_chamfer: %w", err)
		}
		if gear, err = chamferCaps(ctx, gear, setback); err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("spurgear: finished gear is not a solid")
	}
	return gear, nil
}

// extrudeRootDisc extrudes the root-disc circle through the thickness.
func extrudeRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: bind root disc radius: %w", err)
	}
	if err := solveVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := onlyProfile(s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// BuildTwistedProfile draws the whole-gear outline with every tooth turned by
// twist and returns the 2N root contacts, right then left of each tooth.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist %g must be finite", twist)
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

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create outline sketch: %w", err)
	}
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, s, p, d, twist)
	}
	return buildOrdinaryOutline(ctx, s, p, d, twist)
}

// buildOrdinaryOutline draws all N teeth joined by root arcs.
func buildOrdinaryOutline(ctx context.Context, s *sketch.Sketch, p Params, d dimensions,
	twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var first, prev toothPoints
	for k := range p.Teeth {
		t := drawToothPoints(s, p, d, toothAngle(p, k)+twist)
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			first = t
		} else {
			s.CreateArc(center, prev.leftRoot, t.rightRoot)
		}
		if err := drawToothCurves(s, t); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d: %w", k, err)
		}
		prev = t
	}
	s.CreateArc(center, prev.leftRoot, first.rightRoot)
	if err := solveVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the original flanks and tip arcs with a full root
// circle, unions the bounded regions, and returns the trimmed root contacts.
func buildEmbeddedOutline(ctx context.Context, s *sketch.Sketch, p Params, d dimensions,
	twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	center := s.Origin()
	owner := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := range p.Teeth {
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
				continue
			}
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		owner[rs] = 2 * k
		owner[ls] = 2*k + 1
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	profiles := s.Profiles()
	indices := make([]int, len(profiles))
	for i, pr := range profiles {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline region %d is invalid", i)
		}
		indices[i] = i
	}
	var profile *sketch.Profile
	switch len(profiles) {
	case 0:
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has no regions")
	case 1:
		profile = profiles[0]
	default:
		var err error
		if profile, err = s.UnionProfiles(indices...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: union embedded outline: %w", err)
		}
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline is invalid or has %d holes", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d boundary curves, expected %d",
			len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 2*p.Teeth)
	seen := make([]bool, 2*p.Teeth)
	count := 0
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, ok := owner[flank]
		if !ok {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline holds an unknown spline")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: flank %d has no certified root trim (partial=%t exact=%t)",
				slot, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: flank %d trim radius %g differs from root radius %g",
				slot, math.Hypot(x, y), d.root)
		}
		if seen[slot] {
			return nil, nil, nil, fmt.Errorf("spurgear: flank %d appears twice in the outline", slot)
		}
		seen[slot] = true
		contacts[slot] = point{x, y}
		count++
	}
	if count != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d root contacts, expected %d",
			count, 2*p.Teeth)
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius Rc of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankBeta(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		m := leftFlankPoint(d.base, r, beta)
		return math.Atan2(m.y, m.x)
	}
	limit := math.Pi / teeth
	if half(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: adjacent flanks overlap at the root circle (radius %g)", d.root)
	}
	if half(d.base) < limit {
		return d.base, nil
	}
	lo, hi := d.base, d.root
	for range 200 {
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

// buildEmbeddedTooth draws one embedded tooth closed by an arc at radius Rc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rp := make([]*sketch.Point, len(right))
	lp := make([]*sketch.Point, len(left))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	last := len(rp) - 1
	center := s.Origin()
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded right flank: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded left flank: %w", err)
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
		return nil, nil, fmt.Errorf("spurgear: embedded tooth has %d outer curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges with a float radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start at the saved root contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, q := range roots {
		targets[i] = toWorld(frame, q)
	}
	near := func(a, b r3.Vec) bool { return a.Sub(b).Len() <= 1e-6 }
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		end := edge.End().Position().Value
		for _, target := range targets {
			if near(target, start) {
				corners = append(corners, start)
				break
			}
			if near(target, end) {
				corners = append(corners, end)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet %d root edges: %w", len(corners), err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet %d embedded root edges: %w", 2*teeth, err)
	}
	return out, nil
}

// buildBoreProfile draws the centered bore circle on a plane margin below the gear.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
	}
	low, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(low)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bind bore diameter: %w", err)
	}
	if err := solveVerify(ctx, s, "bore"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "bore")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore extrudes the bore tool and cuts it from gear. The document must
// already define the parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read tool_margin: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the gear thickness from the reference plane.
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
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("spurgear: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both cap faces.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("spurgear: chamfer distance %g must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
