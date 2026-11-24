// Package spurgear builds a cylindrical gear with straight involute teeth.
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

// Params describes one spur gear. Lengths are millimeters and angles radians.
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
	// CapChamfer is the equal-distance outer cap setback; zero omits the chamfer.
	CapChamfer float64
	// Backlash thins each tooth by this arc at the pitch circle.
	Backlash float64
	// Placement gives the reference point and plane; nil selects the world XY frame.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth, module-1 gear.
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

// placement returns the frame of p, or the world XY frame.
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

// validate applies the input checks and limits of the gear contract.
func validate(p Params) error {
	if !(p.Module > 0) {
		return fmt.Errorf("module %v must be positive", p.Module)
	}
	if !(p.Thickness > 0) {
		return fmt.Errorf("thickness %v must be positive", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	}
	if !(p.PressureAngle > 0 && p.PressureAngle < math.Pi/2) {
		return fmt.Errorf("pressure angle %v must lie in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("bore diameter %v must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("cap chamfer %v must not be negative", p.CapChamfer)
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash %v must be finite with magnitude at most module/2 (%v)",
			p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("placement is not a valid frame")
	}

	n := float64(p.Teeth)
	root := (p.Module*n - 2.5*p.Module) / 2
	pitch := p.Module * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	tip := (p.Module*n + 2*p.Module) / 2
	if root <= 0 {
		return fmt.Errorf("root radius %v must be positive", root)
	}
	if base <= 0 {
		return fmt.Errorf("base radius %v must be positive", base)
	}
	if tip <= base {
		return fmt.Errorf("tip radius %v must exceed base radius %v", tip, base)
	}
	if p.BoreDiameter >= 2*root {
		return fmt.Errorf("bore diameter %v must be less than the root diameter %v",
			p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(fillet) || math.IsInf(fillet, 0) {
		return fmt.Errorf("root fillet radius %v is not finite", fillet)
	}
	return nil
}

// defineParameters writes the gear parameter table into t.
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

// DefineParameters writes the parameter table of p into t without validating p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// Build builds the gear described by p into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear described by p into doc through the variant hooks.
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

// assemble builds the gear from a prepared run.
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
		return nil, fmt.Errorf("root disc extrusion: %w", err)
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
			return nil, fmt.Errorf("through bore: %w", err)
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
		return nil, errors.New("spur gear: result is not a solid")
	}
	return body, nil
}

// solveSingleProfile solves and verifies s, then returns its only valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// BuildProfile draws the unfilleted whole-gear outline of p on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// BuildTwistedProfile draws the whole-gear outline with every tooth turned by an
// extra twist about the sketch origin. It also returns the 2N root contacts,
// right then left per tooth, as sketch coordinates.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twisted profile: nonfinite twist %v", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, root := range roots {
		out[i] = [2]float64{root.x, root.y}
	}
	return s, profile, out, nil
}

// profileGuards checks the arguments shared by the profile APIs.
func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return errors.New("spur profile: nil context")
	}
	if world == nil {
		return errors.New("spur profile: nil sketch world")
	}
	if plane == nil {
		return errors.New("spur profile: nil plane")
	}
	return validate(p)
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline of either branch with
// each tooth turned by an extra twist. It returns the root contacts.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		t, err := drawOrdinaryTooth(s, d, left, right, prevLeft)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: %w", k, err)
		}
		if k == 0 {
			firstRight = t.rightRoot
		}
		prevLeft = t.leftRoot
		roots = append(roots, t.rightRootXY, t.leftRootXY)
	}
	s.CreateArc(s.Origin(), prevLeft, firstRight)

	profile, err := solveSingleProfile(ctx, s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("gear outline: %d boundary curves, expected %d", got, want)
	}
	return s, profile, roots, nil
}

// flankSplines pairs the flank splines drawn for one embedded-outline tooth.
type flankSplines struct{ right, left *sketch.FitSpline }

// buildEmbeddedOutline draws every original flank and tip arc plus one root
// circle, unions the bounded regions, and reads the trimmed root contacts.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	center := s.Origin()
	splines := make([]flankSplines, p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		n := len(right)
		rightPoints := make([]*sketch.Point, n)
		leftPoints := make([]*sketch.Point, n)
		for i := 0; i < n; i++ {
			rightPoints[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		}
		for i := 0; i < n; i++ {
			if i == n-1 {
				leftPoints[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				leftPoints[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		rs, err := s.CreateFitSpline(rightPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPoints[n-1], leftPoints[n-1])
		ls, err := s.CreateFitSpline(leftPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: left flank spline: %w", k, err)
		}
		splines[k] = flankSplines{right: rs, left: ls}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
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
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union has %d holes", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: %d boundary curves, expected %d", got, want)
	}

	type slot struct{ tooth, side int }
	owners := make(map[*sketch.FitSpline]slot, 2*p.Teeth)
	for k, sp := range splines {
		owners[sp.right] = slot{k, 0}
		owners[sp.left] = slot{k, 1}
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		owner, known := owners[flank]
		if !known {
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root trim is not certified", owner.tooth)
		}
		idx := 2*owner.tooth + owner.side
		if filled[idx] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has a duplicate flank edge", owner.tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact is off the root circle",
				owner.tooth)
		}
		contacts[idx] = point{x, y}
		filled[idx] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has no root contact", i/2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	half := func(r float64) float64 {
		pt := leftFlankAt(d.pitch, d.base, teeth, d.backlash, r)
		return math.Atan2(pt.y, pt.x)
	}
	limit := math.Pi / teeth
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: neighbouring flanks overlap at the root circle (radius %v)", d.root)
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

// buildEmbeddedTooth draws one embedded tooth whose flanks cross the root circle.
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
	n := len(right)
	rightPoints := make([]*sketch.Point, n)
	leftPoints := make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rightPoints[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == 0 || i == n-1 {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			leftPoints[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[n-1], leftPoints[n-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[0], leftPoints[0])

	profile, err := solveSingleProfile(ctx, s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: %d boundary curves, expected 4", got)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges with a radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start at a saved root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	matches := func(pos r3.Vec) bool {
		for _, root := range roots {
			if toWorld(frame, root).Sub(pos).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if matches(start) {
			corners = append(corners, start)
			continue
		}
		end := edge.End().Position().Value
		if matches(end) {
			corners = append(corners, end)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the gear.
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

// cutBore cuts the centered through-bore using the document's parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool extrusion: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body. The
// body must span the placed gear's lower cap through its thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("through bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("through bore: nil document")
	}
	if body == nil {
		return nil, errors.New("through bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("through bore: diameter %v must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops of gear by setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers the outer loops of both caps of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("cap chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("cap chamfer: distance %v must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
