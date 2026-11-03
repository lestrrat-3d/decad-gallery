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

// Params are the inputs of one spur gear. Lengths are millimeters and angles
// are radians. Zero fields are not defaults; use DefaultParams.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement anchors the gear: its origin is the reference point, U and V
	// span the reference plane. Nil selects the world XY frame at the origin.
	Placement *r3.Frame
}

type point struct{ x, y float64 }

// DefaultParams returns the default 17-tooth, 1 mm module gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
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

// validate applies the input checks shared by Build, BuildProfile and
// BuildToothSection.
func validate(p Params) error {
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
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("spurgear: invalid placement frame")
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
		return fmt.Errorf("spurgear: bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(fillet) || math.IsInf(fillet, 0) {
		return fmt.Errorf("spurgear: root fillet radius is not finite")
	}
	return nil
}

// defineParameters writes the gear's parameter table.
func defineParameters(t *param.Table, p Params) error {
	wrap := func(name string, err error) error {
		if err != nil {
			return fmt.Errorf("spurgear: define parameter %s: %w", name, err)
		}
		return nil
	}
	if err := wrap("module", t.SetValue("module", units.Millimeters(p.Module))); err != nil {
		return err
	}
	if err := wrap("teeth", t.SetNumber("teeth", float64(p.Teeth))); err != nil {
		return err
	}
	if err := wrap("pressure_angle", t.SetValue("pressure_angle", units.Radians(p.PressureAngle))); err != nil {
		return err
	}
	if err := wrap("thickness", t.SetValue("thickness", units.Millimeters(p.Thickness))); err != nil {
		return err
	}
	if err := wrap("samples", t.SetNumber("samples", float64(p.Samples))); err != nil {
		return err
	}
	if err := wrap("bore_diameter", t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter))); err != nil {
		return err
	}
	if err := wrap("cap_chamfer", t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer))); err != nil {
		return err
	}
	if err := wrap("tool_margin", t.SetExpr("tool_margin", "1", units.Millimeter)); err != nil {
		return err
	}
	if err := wrap("pitch_radius", t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter)); err != nil {
		return err
	}
	if err := wrap("base_radius", t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter)); err != nil {
		return err
	}
	if err := wrap("root_radius", t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter)); err != nil {
		return err
	}
	if err := wrap("tip_radius", t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter)); err != nil {
		return err
	}
	if err := wrap("involute_angle", t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One)); err != nil {
		return err
	}
	if err := wrap("fillet_radius", t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter)); err != nil {
		return err
	}
	return wrap("bore_tool_length", t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter))
}

// Build constructs the gear in doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: build: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: build: nil document")
	}
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear, delegating to the local pipeline when requested.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
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
// the finishing features.
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
		gear, err = cutBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read cap_chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("spurgear: finished gear is not a solid")
	}
	return gear, nil
}

// buildRootDisc extrudes the root-radius disc on the lower plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: root disc: bind root_radius: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "root disc", 0)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: extrude: %w", err)
	}
	return disc, nil
}

// solveSingleProfile solves and verifies s, then returns its one valid
// profile. A positive outer requires exactly that many outer curves.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, stage string, outer int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, want 1", stage, len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	if outer > 0 && len(profile.Outer) != outer {
		return nil, fmt.Errorf("spurgear: %s: got %d outer curves, want %d", stage, len(profile.Outer), outer)
	}
	return profile, nil
}

// flankSketchPoints creates the dimensioned flank points: every right point
// with both coordinates, every left point with both except the tip.
func flankSketchPoints(s *sketch.Sketch, left, right []point) (rightPts, leftPts []*sketch.Point) {
	rightPts = make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts = make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rightPts, leftPts
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: profile: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: profile: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: profile: nil plane")
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

// buildGearProfile draws the whole-gear outline and returns the actual root
// contacts, right then left for each tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rightPts, leftPts := flankSketchPoints(s, left, right)
		leftContact := rootContact(left[0], d)
		rightContact := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, leftContact.x, leftContact.y)
		rightRoot := sketchcheck.DimensionPoint(s, rightContact.x, rightContact.y)
		roots = append(roots, rightContact, leftContact)
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	profile, err := solveSingleProfile(ctx, s, "outline", 0)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the full flanks and the root circle, then unions
// the regions into one hole-free outline trimmed at the root circle.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rightPts, leftPts := flankSketchPoints(s, left, right)
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: verify: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, 0, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: got %d boundary curves, want %d",
			len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: edge %d lacks a certified root trim "+
				"(partial %t, exact %t)", i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: edge %d trim at radius %g, want %g",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x: x, y: y})
	}
	if len(contacts) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: got %d root contacts, want %d",
			len(contacts), 2*p.Teeth)
	}
	return s, profile, contacts, nil
}

// closingRadius chooses the radius Rc of the arc that closes an embedded
// tooth section.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	h := func(r float64) float64 { return halfAngle(d.pitch, d.base, teeth, r) }
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: no closing radius: adjacent teeth overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
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

// buildEmbeddedTooth draws one embedded tooth: right spline, tip arc, left
// spline and the closing arc at radius Rc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	rightPts := make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	last := len(rightPts) - 1
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	profile, err := solveSingleProfile(ctx, s, "embedded tooth", 4)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges matched to roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges whose start vertex lies on a saved
// root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
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
		return nil, fmt.Errorf("spurgear: matched %d root edges, want %d", len(corners), len(roots))
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

// filletEmbedded fillets the 2*teeth concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet %d embedded root edges: %w", 2*teeth, err)
	}
	return out, nil
}

// buildBoreProfile draws the circular bore tool section on a plane set back
// from the gear's lower cap by margin.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: bind bore_diameter: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "bore tool", 0)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of diameter p.BoreDiameter from
// body, which spans the gear's lower cap to its upper cap.
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
		return nil, fmt.Errorf("spurgear: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool and cuts it from gear. The document's table
// already defines the gear parameters.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: read tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: read bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with setback
// distance. Bore rims stay sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: chamfer: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: chamfer: nil body")
	}
	if !(distance > 0) {
		return nil, fmt.Errorf("spurgear: chamfer setback %g must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}
