// Package spurgear builds a parametric cylindrical gear with straight involute
// teeth as a Decad solid.
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

// Params describes one spur gear. Lengths are millimeters and angles radians.
type Params struct {
	// Module is the tooth module m.
	Module float64
	// Teeth is the tooth count N.
	Teeth int
	// PressureAngle is the pressure angle α.
	PressureAngle float64
	// Thickness is the axial thickness h.
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

// DefaultParams returns the default gear inputs.
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

// placement resolves the placement frame of p.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted moves f by dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a sketch coordinate pair in f to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// DefineParameters writes the Parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	if t == nil {
		return errors.New("spurgear: define parameters: nil table")
	}
	return defineParameters(t, p)
}

// defineParameters writes the spur gear parameter table into t.
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
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2",
		units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: parameter bore_tool_length: %w", err)
	}
	return nil
}

// validate applies the input checks and limits to p.
func validate(p Params) error {
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spurgear: module must be positive, got %g", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spurgear: thickness must be positive, got %g", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spurgear: pressure angle must lie strictly between 0 and pi/2, got %g", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter must not be negative, got %g", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer must not be negative, got %g", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("spurgear: backlash must be finite, got %g", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash magnitude %g exceeds half the module %g", p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: placement is not a valid frame")
	}
	d, err := deriveDimensions(p)
	if err != nil {
		return fmt.Errorf("spurgear: derive dimensions: %w", err)
	}
	switch {
	case d.root <= 0:
		return fmt.Errorf("spurgear: root radius must be positive, got %g", d.root)
	case d.base <= 0:
		return fmt.Errorf("spurgear: base radius must be positive, got %g", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spurgear: fillet radius must be finite, got %g", d.fillet)
	}
	return nil
}

// filletRadius returns the derived root fillet radius.
func filletRadius(_ Params, d dimensions) float64 {
	return d.fillet
}

// Build constructs the spur gear described by p in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear with this variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: build: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: build: nil document")
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

// assemble builds tooth zero and the root disc, patterns and joins them, and
// applies the root fillet, bore and cap chamfer.
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
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join disc and teeth: %w", err)
	}
	if !gear.IsSolid() {
		return nil, errors.New("spurgear: join disc and teeth: result is not a solid")
	}

	switch {
	case v.RootFillet != nil:
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	case r.d.fillet > 0:
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
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
			return nil, fmt.Errorf("spurgear: read parameter cap_chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}

	if !gear.IsSolid() {
		return nil, errors.New("spurgear: build: result is not a solid")
	}
	return gear, nil
}

// buildRootDisc draws the root circle on the lower plane, bound to
// root_radius, and extrudes it through the gear thickness.
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
	if err := solveAndVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile returns the whole unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, "profile"); err != nil {
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

// BuildTwistedProfile returns the whole gear outline with every tooth turned
// by twist about the sketch origin, and the 2N root contacts: right root then
// left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, "twisted profile"); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twisted profile: twist must be finite, got %g", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, stage string) error {
	switch {
	case ctx == nil:
		return fmt.Errorf("spurgear: %s: nil context", stage)
	case world == nil:
		return fmt.Errorf("spurgear: %s: nil world", stage)
	case plane == nil:
		return fmt.Errorf("spurgear: %s: nil plane", stage)
	}
	return nil
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with each tooth k at
// toothAngle(p,k)+twist and returns its root contacts.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth
	roots := make([]point, 0, 2*n)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < n; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(n), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		last := len(right) - 1
		rightPts := dimensionFlank(s, right, false, false)
		leftPts := dimensionFlank(s, left, false, true)
		lr := rootContact(left[0], d)
		rr := rootContact(right[0], d)
		leftRoot := dimensionArcEnd(s, lr)
		rightRoot := dimensionPoint(s, rr)
		roots = append(roots, rr, lr)

		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[last], leftPts[last])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	if err := solveAndVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if got, want := len(profile.Outer), 6*n; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: gear outline: %d boundary curves, expected %d", got, want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile draws the whole-gear outline when the base circle lies
// inside the root circle and returns the exact trimmed root contacts.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth

	type flankRef struct{ tooth, side int }
	refs := make(map[*sketch.FitSpline]flankRef, 2*n)
	for k := 0; k < n; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(n), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		last := len(right) - 1
		rightPts := dimensionFlank(s, right, false, false)
		leftPts := dimensionFlank(s, left, false, true)
		rs, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[last], leftPts[last])
		ls, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d left flank spline: %w", k, err)
		}
		refs[rs] = flankRef{k, 0}
		refs[ls] = flankRef{k, 1}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}

	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("spurgear: embedded outline: no bounded region")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded outline: union profile is not valid")
	}
	if got := len(profile.Holes); got != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union profile has %d holes, expected 0", got)
	}
	if got, want := len(profile.Outer), 4*n; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: %d boundary curves, expected %d", got, want)
	}

	contacts := make([]point, 2*n)
	seen := make([]bool, 2*n)
	found := 0
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		ref, ok := refs[flank]
		if !ok {
			return nil, nil, nil, errors.New("spurgear: embedded outline: boundary spline is not a tooth flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank root trim is not certified "+
				"(partial %t, exact %t)", ref.tooth, sideName(ref.side), edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if diff := math.Abs(math.Hypot(x, y) - d.root); diff > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank root contact is %g "+
				"off the root circle", ref.tooth, sideName(ref.side), diff)
		}
		idx := 2*ref.tooth + ref.side
		if seen[idx] {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank appears twice",
				ref.tooth, sideName(ref.side))
		}
		seen[idx] = true
		contacts[idx] = point{x, y}
		found++
	}
	if found != 2*n {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: %d root contacts, expected %d", found, 2*n)
	}
	return s, profile, contacts, nil
}

func sideName(side int) string {
	if side == 0 {
		return "right"
	}
	return "left"
}

// buildEmbeddedTooth draws the four-curve embedded tooth section at angle 0.
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
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	last := len(right) - 1
	rightPts := dimensionFlank(s, right, false, false)
	leftPts := dimensionFlank(s, left, true, true)

	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])

	if err := solveAndVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: %d outer curves, expected 4", got)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges matched to roots with a radius
// in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges of body that start or end at the
// saved root contacts on the lower cap.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("spurgear: root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: select axial edges: %w", err)
	}
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	matches := func(pos r3.Vec) bool {
		for _, t := range targets {
			if pos.Sub(t).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	corners := make([]r3.Vec, 0, len(roots))
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
		return nil, fmt.Errorf("spurgear: root fillet: matched %d axial root edges, expected %d",
			len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: embedded root fillet: %w", err)
	}
	return filleted, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through p.Thickness in p's placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: through bore: nil context")
	case doc == nil:
		return nil, errors.New("spurgear: through bore: nil document")
	case body == nil:
		return nil, errors.New("spurgear: through bore: nil body")
	case !(p.BoreDiameter > 0):
		return nil, fmt.Errorf("spurgear: through bore: bore diameter must be positive, got %g", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore cuts the through-bore using doc's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: through bore: read parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: through bore: read parameter bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: through bore: extrude tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: through bore: cut: %w", err)
	}
	return cut, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if !(p.BoreDiameter > 0) {
		return nil, nil, fmt.Errorf("spurgear: bore tool: bore diameter must be positive, got %g", p.BoreDiameter)
	}
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: shift placement: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: create plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: bind bore_diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with setback
// distance in millimeters. Inner loops such as a bore stay sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: cap chamfer: nil context")
	case gear == nil:
		return nil, errors.New("spurgear: cap chamfer: nil body")
	case math.IsNaN(distance) || math.IsInf(distance, 0) || distance <= 0:
		return nil, fmt.Errorf("spurgear: cap chamfer: distance must be positive and finite, got %g", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps of gear. It checks no
// arguments.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return chamfered, nil
}

// dimensionFlank creates the flank sample points with signed driving
// dimensions from the sketch origin. singleFirst and singleLast give the first
// and last points one coordinate dimension for an arc end.
func dimensionFlank(s *sketch.Sketch, pts []point, singleFirst, singleLast bool) []*sketch.Point {
	out := make([]*sketch.Point, len(pts))
	last := len(pts) - 1
	for i, pt := range pts {
		if (i == 0 && singleFirst) || (i == last && singleLast) {
			out[i] = dimensionArcEnd(s, pt)
			continue
		}
		out[i] = dimensionPoint(s, pt)
	}
	return out
}

func dimensionPoint(s *sketch.Sketch, pt point) *sketch.Point {
	return sketchcheck.DimensionPoint(s, pt.x, pt.y)
}

func dimensionArcEnd(s *sketch.Sketch, pt point) *sketch.Point {
	return sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	return nil
}

// singleProfile requires exactly one valid profile in s.
func singleProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	return profiles[0], nil
}
