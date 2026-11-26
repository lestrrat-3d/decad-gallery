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

// Params describes one straight involute spur gear. Lengths are in
// millimeters and angles in radians.
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

// placement returns the placement frame, or the world XY frame when none is set.
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

// toWorld maps a frame-local point to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits shared by every entry point.
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
		return fmt.Errorf("pressure angle %g must be between 0 and pi/2", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("backlash %g must be finite and within module/2 (%g)", p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("placement is not a valid frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return fmt.Errorf("root radius %g must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("base radius %g must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("root fillet radius %g is not finite", d.fillet)
	}
	return nil
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

// DefineParameters writes the Parameter table of p into t. It does not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// Build constructs the gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear in doc, applying the variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear build: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear build: nil document")
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

// assemble builds the gear from a prepared run: tooth zero, the root disc, the
// circular pattern, one join, then root fillet, bore and cap chamfer.
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
		return nil, fmt.Errorf("join teeth and root disc: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join teeth and root disc: result is not a solid")
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
			return nil, fmt.Errorf("through-bore: %w", err)
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

// buildRootDisc extrudes the root-disc circle on the lower plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(r.doc, s, profile, r.extent).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return disc, nil
}

// checkProfileArgs rejects nil arguments shared by the profile builders.
func checkProfileArgs(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return errors.New("nil context")
	}
	if world == nil {
		return errors.New("nil sketch world")
	}
	if plane == nil {
		return errors.New("nil sketch plane")
	}
	return nil
}

// solveSingleProfile solves and verifies s, then requires exactly one valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("sketch has %d profiles, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("sketch profile is not valid")
	}
	return profiles[0], nil
}

// flankPoints creates the dimensioned flank points of one tooth. Every right
// point and every left point except its tip receives both coordinate
// dimensions; the left tip receives one.
func flankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
	rightPoints := make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPoints := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rightPoints, leftPoints
}

// BuildProfile draws the whole unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
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

// BuildTwistedProfile draws the whole unfilleted outline with every tooth
// turned by twist about the sketch origin. It also returns the 2N root contacts.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist %g is not finite", twist)
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

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with each tooth turned
// by an extra twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPoints, leftPoints := flankPoints(s, left, right)
		last := len(rightPoints) - 1

		leftRootXY := rootContact(left[0], d)
		rightRootXY := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, leftRootXY.x, leftRootXY.y)
		rightRoot := sketchcheck.DimensionPoint(s, rightRootXY.x, rightRootXY.y)
		roots = append(roots, rightRootXY, leftRootXY)

		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rightPoints[0])
		if _, err := s.CreateFitSpline(rightPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPoints[last], leftPoints[last])
		if _, err := s.CreateFitSpline(leftPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(leftPoints[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("gear outline: profile has %d boundary curves, expected %d", got, want)
	}
	return s, profile, roots, nil
}

// flankSplines records the two fit splines of one embedded tooth.
type flankSplines struct{ right, left *sketch.FitSpline }

// buildEmbeddedProfile draws the whole-gear outline of an embedded gear: the
// original flanks and tip arcs plus one root circle, joined into one region.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	splines := make([]flankSplines, p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPoints, leftPoints := flankPoints(s, left, right)
		last := len(rightPoints) - 1

		rightSpline, err := s.CreateFitSpline(rightPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPoints[last], leftPoints[last])
		leftSpline, err := s.CreateFitSpline(leftPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank spline: %w", k, err)
		}
		splines[k] = flankSplines{right: rightSpline, left: leftSpline}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: verify sketch: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("embedded outline: sketch has no region")
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
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: profile has %d boundary curves, expected %d", got, want)
	}

	type slot struct{ index int }
	owner := make(map[*sketch.FitSpline]slot, 2*p.Teeth)
	for k, sp := range splines {
		owner[sp.right] = slot{2 * k}
		owner[sp.left] = slot{2*k + 1}
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		sl, known := owner[spline]
		if !known {
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth flank")
		}
		tooth := sl.index / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank trim at the root circle is not certified", tooth)
		}
		if filled[sl.index] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank appears twice in the boundary", tooth)
		}
		x, y := spline.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact radius %g differs from root radius %g",
				tooth, math.Hypot(x, y), d.root)
		}
		contacts[sl.index] = point{x, y}
		filled[sl.index] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has no trimmed root contact", i/2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius Rc of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankShift(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		m := mirroredInvolute(d.base, r)
		l := rotate(m.x, m.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / teeth
	if half(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: neighbouring teeth overlap at the root circle (radius %g)", d.root)
	}
	if half(d.base) < limit {
		return d.base, nil
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
	return ((lo+hi)/2 + d.root) / 2, nil
}

// buildEmbeddedTooth draws one embedded tooth section, closed by an arc about
// the gear axis at the closing radius.
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
	rightPoints := make([]*sketch.Point, len(right))
	for i, q := range right {
		rightPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	leftPoints := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	last := len(rightPoints) - 1

	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[0], leftPoints[0])

	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: profile has %d boundary curves, expected 4", got)
	}
	return s, profile, nil
}

// filletRoots rounds the ordinary root edges with a radius given in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue rounds the axial edges that start or end at a saved root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, q := range roots {
		targets[i] = toWorld(frame, q)
	}
	matches := func(pos r3.Vec) bool {
		for _, target := range targets {
			if pos.Sub(target).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, edge := range edges {
		if pos := edge.Start().Position().Value; matches(pos) {
			corners = append(corners, pos)
		} else if pos := edge.End().Position().Value; matches(pos) {
			corners = append(corners, pos)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(body, selection.Exactly(len(corners)), radius).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return filleted, nil
}

// filletEmbedded rounds the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(body, selection, radius).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the circular bore tool sketch on a plane margin
// below the placement plane.
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
		return nil, nil, fmt.Errorf("bore tool: bind diameter: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts a centered through-bore using the document's existing parameters.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
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

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cut through bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("cut through bore: nil document")
	}
	if body == nil {
		return nil, errors.New("cut through bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("cut through bore: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both cap faces.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(gear, selection, setback).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers the outer loops of both cap faces by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("chamfer outer caps: nil context")
	}
	if gear == nil {
		return nil, errors.New("chamfer outer caps: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer outer caps: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
