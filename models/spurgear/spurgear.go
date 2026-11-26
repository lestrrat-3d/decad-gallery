// Package spurgear builds a cylindrical gear with straight involute teeth.
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

// Params describes one spur gear. Lengths are millimetres and angles are
// radians.
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
	// Backlash is the circumferential tooth thinning at the pitch circle.
	Backlash float64
	// Placement is the reference point and plane; nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default spur gear.
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

// placement returns the gear's frame: the supplied placement, or the world
// XY frame at the origin.
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

// toWorld maps frame-plane coordinates to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// DefineParameters writes the spur gear parameter table of p into t. It does
// not validate p.
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

// validate applies the input checks and limits of the gear contract.
func validate(p Params) error {
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spur gear: module must be positive, got %v", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spur gear: thickness must be positive, got %v", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spur gear: teeth must be at least 3, got %d", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spur gear: samples must be at least 3, got %d", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spur gear: pressure angle must lie in (0, pi/2), got %v", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spur gear: bore diameter must not be negative, got %v", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spur gear: cap chamfer must not be negative, got %v", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("spur gear: backlash must be finite, got %v", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spur gear: backlash magnitude %v exceeds half the module %v", p.Backlash, p.Module)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spur gear: placement is not a valid frame")
	}
	d, err := deriveChecked(p)
	if err != nil {
		return fmt.Errorf("spur gear: %w", err)
	}
	switch {
	case d.root <= 0:
		return fmt.Errorf("spur gear: root radius must be positive, got %v", d.root)
	case d.base <= 0:
		return fmt.Errorf("spur gear: base radius must be positive, got %v", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spur gear: tip radius %v must exceed base radius %v", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spur gear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spur gear: root fillet radius must be finite, got %v", d.fillet)
	}
	return nil
}

// Build constructs the spur gear described by p in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear of this variant in doc.
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

// assemble builds tooth zero and the root disc, patterns and joins them, and
// applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
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
		return nil, fmt.Errorf("join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join teeth: result is not a solid")
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
			return nil, fmt.Errorf("read parameter cap_chamfer: %w", err)
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

// buildRootDisc extrudes the root circle through the gear thickness.
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
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the whole unfilleted gear outline on plane and returns
// the solved, verified sketch and its profile.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	d, err := profileGuards(ctx, world, plane, p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole gear outline with every tooth turned by
// twist about the sketch origin, and returns its 2N root contacts: the right
// then the left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("gear profile: twist must be finite, got %v", twist)
	}
	d, err := profileGuards(ctx, world, plane, p)
	if err != nil {
		return nil, nil, nil, err
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (dimensions, error) {
	if ctx == nil {
		return dimensions{}, errors.New("gear profile: nil context")
	}
	if world == nil {
		return dimensions{}, errors.New("gear profile: nil world")
	}
	if plane == nil {
		return dimensions{}, errors.New("gear profile: nil plane")
	}
	if err := validate(p); err != nil {
		return dimensions{}, err
	}
	d, err := deriveChecked(p)
	if err != nil {
		return dimensions{}, fmt.Errorf("gear profile: %w", err)
	}
	return d, nil
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with every tooth
// turned by an extra twist, for both branches.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	return buildOrdinaryProfile(ctx, world, plane, p, d, twist)
}

// buildOrdinaryProfile draws the six-curve-per-tooth outline of a gear whose
// base circle lies on or outside its root circle.
func buildOrdinaryProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRightRoot, previousLeftRoot *sketch.Point
	for k := range p.Teeth {
		leftSamples, rightSamples := sampleFlanks(d.pitch, d.base, d.base, d.tip,
			float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rightPoints, leftPoints := flankPoints(s, rightSamples, leftSamples)

		leftRootAt := rootContact(leftSamples[0], d)
		rightRootAt := rootContact(rightSamples[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, leftRootAt.x, leftRootAt.y)
		rightRoot := sketchcheck.DimensionPoint(s, rightRootAt.x, rightRootAt.y)
		roots = append(roots, rightRootAt, leftRootAt)

		if k == 0 {
			firstRightRoot = rightRoot
		} else {
			s.CreateArc(center, previousLeftRoot, rightRoot)
		}
		last := len(rightPoints) - 1
		s.CreateLine(rightRoot, rightPoints[0])
		if _, err := s.CreateFitSpline(rightPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear profile: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPoints[last], leftPoints[last])
		if _, err := s.CreateFitSpline(leftPoints...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear profile: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(leftPoints[0], leftRoot)
		previousLeftRoot = leftRoot
	}
	s.CreateArc(center, previousLeftRoot, firstRightRoot)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear profile: outline has %d boundary curves, want %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// flankRef records which tooth and side a fit spline belongs to.
type flankRef struct {
	tooth int
	left  bool
}

// buildEmbeddedProfile draws the outline of a gear whose base circle lies
// inside its root circle: original flanks and tip arcs over one full root
// circle, unioned into one hole-free region.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: create sketch: %w", err)
	}
	center := s.Origin()
	flanks := make(map[*sketch.FitSpline]flankRef, 2*p.Teeth)
	for k := range p.Teeth {
		leftSamples, rightSamples := sampleFlanks(d.pitch, d.base, d.base, d.tip,
			float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rightPoints, leftPoints := flankPoints(s, rightSamples, leftSamples)
		last := len(rightPoints) - 1
		right, err := s.CreateFitSpline(rightPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPoints[last], leftPoints[last])
		left, err := s.CreateFitSpline(leftPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d left flank: %w", k, err)
		}
		flanks[right] = flankRef{tooth: k}
		flanks[left] = flankRef{tooth: k, left: true}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("embedded profile: sketch has no regions")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded profile: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded profile: union is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded profile: union has %d holes, want 0", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded profile: outline has %d boundary curves, want %d",
			len(profile.Outer), want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		ref, ok := flanks[spline]
		if !ok {
			return nil, nil, nil, errors.New("embedded profile: boundary spline belongs to no tooth")
		}
		side := "right"
		slot := 2 * ref.tooth
		if ref.left {
			side = "left"
			slot++
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf(
				"embedded profile: tooth %d %s flank trim is not certified (partial %t, exact %t)",
				ref.tooth, side, edge.Partial, edge.TExact)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d %s flank appears twice", ref.tooth, side)
		}
		x, y := spline.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf(
				"embedded profile: tooth %d %s root contact radius %v, want %v",
				ref.tooth, side, math.Hypot(x, y), d.root)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			side := "right"
			if slot%2 == 1 {
				side = "left"
			}
			return nil, nil, nil, fmt.Errorf("embedded profile: tooth %d %s root contact missing", slot/2, side)
		}
	}
	return s, profile, contacts, nil
}

// buildEmbeddedTooth draws tooth zero of an embedded gear: its flanks run
// from the closing radius to the tip and an arc at the closing radius closes
// the four-curve section.
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
	leftSamples, rightSamples := sampleFlanks(d.pitch, d.base, rc, d.tip,
		float64(p.Teeth), d.backlash, p.Samples, 0)
	rightPoints := make([]*sketch.Point, len(rightSamples))
	for i, pt := range rightSamples {
		rightPoints[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	last := len(leftSamples) - 1
	leftPoints := make([]*sketch.Point, len(leftSamples))
	for i, pt := range leftSamples {
		if i == 0 || i == last {
			leftPoints[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		leftPoints[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank: %w", err)
	}
	s.CreateArc(center, rightPoints[last], leftPoints[last])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank: %w", err)
	}
	s.CreateArc(center, rightPoints[0], leftPoints[0])

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: profile has %d outer curves, want 4", len(profile.Outer))
	}
	return s, profile, nil
}

// filletRoots rounds the ordinary root corners with a radius in millimetres.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue rounds the axial edges of body that start or end at the
// saved root contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet: select axial edges: %w", err)
	}
	worldRoots := make([]r3.Vec, len(roots))
	for i, root := range roots {
		worldRoots[i] = toWorld(frame, root)
	}
	matches := func(pos r3.Vec) bool {
		for _, root := range worldRoots {
			if pos.Sub(root).Len() <= 1e-6 {
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
		return nil, fmt.Errorf("root fillet: matched %d axial root edges, want %d", len(corners), len(roots))
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

// filletEmbedded rounds the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return filleted, nil
}

// CutThroughBore cuts the centered through-bore of p from body, which spans
// the gear thickness along the placement normal. It defines the spur gear
// parameters on doc first.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("through-bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("through-bore: nil document")
	}
	if body == nil {
		return nil, errors.New("through-bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("through-bore: bore diameter must be positive, got %v", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("through-bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore cuts the bore using the document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read parameter tool_margin: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read parameter bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool: extrude: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return cut, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the lower
// cap and returns the solved, verified sketch and its profile.
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

// ChamferOuterCaps chamfers the outer loops of both caps of gear with a
// setback of distance millimetres. Inner loops such as a bore stay sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("cap chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("cap chamfer: distance must be positive and finite, got %v", distance)
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
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return chamfered, nil
}
