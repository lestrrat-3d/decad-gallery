package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// Params describes one cylindrical gear with straight involute teeth.
// Lengths are in millimeters and angles in radians. Zero fields are not
// defaults; use DefaultParams. A nil Placement selects the world XY frame at
// the origin.
type Params struct {
	// Module is the tooth module.
	Module float64
	// Teeth is the tooth count.
	Teeth int
	// PressureAngle is the pressure angle in radians.
	PressureAngle float64
	// Thickness is the axial thickness.
	Thickness float64
	// Samples is the number of fit points per flank.
	Samples int
	// BoreDiameter is the centered through-bore diameter; zero omits the bore.
	BoreDiameter float64
	// CapChamfer is the equal-distance outer cap setback; zero omits it.
	CapChamfer float64
	// Backlash thins each tooth by this much arc at the pitch circle.
	Backlash float64
	// Placement is the reference point and plane. Its origin is the reference
	// point, its U and V axes span the reference plane and its normal U x V is
	// the gear axis.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth, module 1 gear.
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
// Gear models that reuse Params without placement support call it so a
// placement is never silently ignored.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// validate applies the input checks and limits to p.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("module %v must be positive", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("pressure angle %v must lie in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %v must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("backlash %v must be finite with magnitude at most module/2 (%v)",
			p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("invalid placement frame")
	}
	n := float64(p.Teeth)
	pitch := p.Module * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (p.Module*n - 2.5*p.Module) / 2
	tip := (p.Module*n + 2*p.Module) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("root radius %v must be positive", root)
	case base <= 0:
		return fmt.Errorf("base radius %v must be positive", base)
	case tip <= base:
		return fmt.Errorf("tip radius %v must exceed base radius %v", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("bore diameter %v must be less than the root diameter %v", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(fillet) || math.IsInf(fillet, 0) {
		return fmt.Errorf("root fillet radius %v is not finite", fillet)
	}
	return nil
}

// placement returns the frame of p, or the world XY frame when it has none.
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

// toWorld maps a point of the frame's plane to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// defineParameters writes the gear's parameter table into t.
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

// DefineParameters writes the gear's parameter table of p into t, so a model
// that builds its gears without Build still records them. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// Build constructs the gear of p in doc and returns its one solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear of p in doc and returns its one solid body.
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

// assemble builds the tooth and the root disc, patterns and joins them,
// then fillets, bores and chamfers.
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
	bodies := append([]*decad.Body{disc, tooth}, copies...)
	body, err := decad.Union(bodies...).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("join disc and teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join disc and teeth: result is not a solid")
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
	if !body.IsSolid() {
		return nil, errors.New("gear is not a solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root-circle disc on the lower plane.
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
	if err := solveVerified(ctx, s); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(r.doc, s, profile, r.extent).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return body, nil
}

// buildGearProfile draws the whole unfilleted gear outline on plane.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole outline with every tooth turned by
// an extra twist about the sketch origin. It returns the 2N root contacts,
// right then left of each tooth in ascending tooth order.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var previousLeftRoot, firstRightRoot *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		tp := newToothPoints(s, d, left, right)
		roots = append(roots, tp.rightRootXY, tp.leftRootXY)
		if k == 0 {
			firstRightRoot = tp.rightRoot
		} else {
			s.CreateArc(center, previousLeftRoot, tp.rightRoot)
		}
		if err := drawFlanks(s, k, tp); err != nil {
			return nil, nil, nil, err
		}
		previousLeftRoot = tp.leftRoot
	}
	s.CreateArc(center, previousLeftRoot, firstRightRoot)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("gear outline has %d boundary curves, expected %d", got, want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the outline of a gear whose flanks cross the
// root circle. The union of the regions has 4N boundary curves. The roots are
// the exact trimmed contacts of each flank with the root circle.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create outline sketch: %w", err)
	}
	center := s.Origin()
	slots := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPts := make([]*sketch.Point, len(right))
		for i, q := range right {
			rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		leftPts := make([]*sketch.Point, len(left))
		for i, q := range left {
			if i == len(left)-1 {
				leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		rightSpline, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		slots[rightSpline] = 2 * k
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		leftSpline, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		slots[leftSpline] = 2*k + 1
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, errors.New("embedded outline: no bounded region")
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
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, errors.New("embedded outline: union is not a valid hole-free profile")
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline has %d boundary curves, expected %d", got, want)
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
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth flank")
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

// halfAngle returns the angle between the tooth axis and the left flank at
// radius r.
func halfAngle(d dimensions, beta, r float64) float64 {
	q := leftFlankPoint(d.base, r, beta)
	return math.Atan2(q.y, q.x)
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	beta := flankRotation(d.pitch, d.base, float64(p.Teeth), d.backlash)
	limit := math.Pi / float64(p.Teeth)
	if halfAngle(d, beta, d.base) < limit {
		return d.base, nil
	}
	if halfAngle(d, beta, d.root) >= limit {
		return 0, fmt.Errorf("no closing radius: adjacent teeth overlap at the root circle (radius %v)", d.root)
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if halfAngle(d, beta, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws one natural tooth of an embedded gear: the right
// flank from the closing radius to the tip, the tip arc, the left flank back,
// and the closing arc about the gear axis.
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
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rightPts[0], leftPts[0])
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("embedded tooth has %d boundary curves, expected 4", got)
	}
	return s, profile, nil
}

// BuildProfile draws the whole unfilleted gear outline of p on plane, for
// both the ordinary and the embedded branch. It returns the solved, verified
// sketch and its profile.
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

// BuildTwistedProfile draws the whole outline with every tooth turned by an
// extra finite twist about the sketch origin, in sketch coordinates. It also
// returns the 2N root contacts: the right then the left root of each tooth in
// ascending tooth order. A twist of zero gives the outline of BuildProfile.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("gear outline: nonfinite twist %v", twist)
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

// profileGuards checks the arguments shared by both profile APIs.
func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return errors.New("gear outline: nil context")
	}
	if world == nil {
		return errors.New("gear outline: nil sketch world")
	}
	if plane == nil {
		return errors.New("gear outline: nil sketch plane")
	}
	return validate(p)
}

// filletRoots fillets the ordinary root edges that match the saved roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end at a saved root.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	near := func(a, b r3.Vec) bool { return a.Sub(b).Len() <= 1e-6 }
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		end := edge.End().Position().Value
		for _, target := range targets {
			if near(start, target) {
				corners = append(corners, start)
				break
			}
			if near(end, target) {
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
		selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(body, selection.Exactly(len(corners)), radius).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(body, selection, radius).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the circular bore tool section on a plane margin
// below the gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("placement: %w", err)
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
		return nil, nil, fmt.Errorf("bind bore_diameter: %w", err)
	}
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts a centered through-bore using the document's existing
// parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter tool_margin: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(doc, s, profile, decad.Distance{D: length, Dir: decad.Along}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	out, err := decad.Cut(gear, tool).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through the thickness of p in the placement frame. It
// defines the standard spur parameters on doc first.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cut bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("cut bore: nil document")
	}
	if body == nil {
		return nil, errors.New("cut bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("cut bore: bore diameter %v must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both caps by setback. The bore loop
// stays sharp.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(gear, selection, setback).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("chamfer caps: nil context")
	}
	if gear == nil {
		return nil, errors.New("chamfer caps: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer caps: distance %v must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
