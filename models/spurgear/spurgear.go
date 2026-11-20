// Package spurgear builds a parametric cylindrical gear with straight involute
// teeth as a Decad solid.
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

// Params are the spur gear inputs. Lengths are millimetres and angles radians.
// Zero fields are not default requests; start from DefaultParams.
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
	// Backlash is the circumferential tooth thinning at the pitch circle; zero
	// keeps the full half-pitch tooth and a negative value thickens the tooth.
	Backlash float64
	// Placement is the reference point (origin) and reference plane (U, V).
	// The gear axis runs along U × V through the origin; the lower cap lies on
	// the plane. Nil selects the world XY frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default spur gear inputs.
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
// Gear models that reuse Params without placement support call it so that a
// placement is never silently ignored.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// placement resolves the gear frame; nil selects the world XY frame.
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

// toWorld maps a frame-plane coordinate to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// DefineParameters writes the spur gear parameter table of p into t. It does
// not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// defineParameters writes the spur gear parameter table of p into t.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("define module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("define teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("define pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("define thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("define samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("define bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("define cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("define backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("define tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("define base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("define involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("define bore_tool_length: %w", err)
	}
	return nil
}

// validate applies the input checks and limits to p.
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
		return fmt.Errorf("spurgear: pressure angle %g must lie strictly between 0 and pi/2", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %g must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("spurgear: backlash %g is not finite", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash %g exceeds half the module %g", p.Backlash, p.Module)
	case p.Placement != nil && !p.Placement.IsValid():
		return fmt.Errorf("spurgear: invalid placement frame")
	}
	d, err := deriveChecked(p)
	if err != nil {
		return fmt.Errorf("spurgear: derive dimensions: %w", err)
	}
	switch {
	case d.root <= 0:
		return fmt.Errorf("spurgear: root radius %g must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("spurgear: base radius %g must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spurgear: fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// Build builds the spur gear described by p in doc and returns its solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear through the spur pipeline with v's replacements.
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

// assemble extrudes tooth zero and the root disc, patterns and joins them,
// then applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
	}
	ds, dprof, err := buildRootDisc(ctx, r.world, r.lower, r.d)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, ds, dprof, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: join teeth: result is not a solid")
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
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: gear is not a solid")
	}
	return body, nil
}

// buildRootDisc draws the root circle on plane, its radius bound to root_radius.
func buildRootDisc(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	radius := sketch.NewRadius(circle, d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, world.Params(), "root_radius"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: root disc: bind root_radius: %w", err)
	}
	if err := solveAndVerify(ctx, s, "root disc"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "root disc", 0)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
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

// singleProfile requires exactly one valid profile in s and, when curves is
// positive, exactly that many outer boundary curves.
func singleProfile(s *sketch.Sketch, stage string, curves int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, want 1", stage, len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	if curves > 0 && len(profile.Outer) != curves {
		return nil, fmt.Errorf("spurgear: %s: got %d boundary curves, want %d", stage, len(profile.Outer), curves)
	}
	return profile, nil
}

// BuildProfile draws the whole unfilleted gear outline on plane and returns the
// solved, verified sketch and its profile.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	d, err := profileInputs(ctx, world, plane, p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole unfilleted gear outline with every tooth
// turned by twist about the sketch origin. It also returns the 2N root contacts,
// right then left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	d, err := profileInputs(ctx, world, plane, p)
	if err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist %g is not finite", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

// profileInputs applies the profile API guards and derives the dimensions.
func profileInputs(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) (dimensions, error) {
	if ctx == nil {
		return dimensions{}, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return dimensions{}, fmt.Errorf("spurgear: nil sketch world")
	}
	if plane == nil {
		return dimensions{}, fmt.Errorf("spurgear: nil sketch plane")
	}
	if err := validate(p); err != nil {
		return dimensions{}, err
	}
	d, err := deriveChecked(p)
	if err != nil {
		return dimensions{}, fmt.Errorf("spurgear: derive dimensions: %w", err)
	}
	return d, nil
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with every tooth turned
// by twist, for the ordinary or the embedded branch.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	return buildOrdinaryOutline(ctx, world, plane, p, d, twist)
}

// buildOrdinaryOutline draws the six-curve-per-tooth whole-gear outline.
func buildOrdinaryOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		rightPts, leftPts := flankPoints(s, left, right, false)
		lr := rootContact(left[0], d)
		rr := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		roots = append(roots, rr, lr)
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		if err := ordinaryToothCurves(s, k, rightRoot, rightPts, leftPts, leftRoot); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear outline: %w", err)
		}
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "gear outline", 6*p.Teeth)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// flankSlot identifies one tooth flank in the embedded outline.
type flankSlot struct {
	tooth int
	left  bool
}

// buildEmbeddedOutline draws the original flanks, tip arcs and one root circle,
// unions every region into one outline, and reads the exact root trims.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	flanks := make(map[*sketch.FitSpline]flankSlot, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		rightPts, leftPts := flankPoints(s, left, right, false)
		n := len(rightPts)
		rs, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPts[n-1], leftPts[n-1])
		ls, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d left flank: %w", k, err)
		}
		flanks[rs] = flankSlot{tooth: k}
		flanks[ls] = flankSlot{tooth: k, left: true}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: no regions")
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
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union profile has %d holes, want 0", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: got %d boundary curves, want %d",
			len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, known := flanks[flank]
		if !known {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: boundary spline is not a known flank")
		}
		side := "right"
		idx := 2 * slot.tooth
		if slot.left {
			side = "left"
			idx++
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank trim is not certified (partial %t, exact %t)",
				slot.tooth, side, edge.Partial, edge.TExact)
		}
		if filled[idx] {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank appears twice", slot.tooth, side)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s root contact radius %g, want %g",
				slot.tooth, side, math.Hypot(x, y), d.root)
		}
		contacts[idx] = point{x, y}
		filled[idx] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d root contact %d missing", i/2, i%2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius chooses the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	h := func(r float64) float64 {
		return flankHalfAngle(d.pitch, d.base, teeth, d.backlash, r)
	}
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: embedded tooth: adjacent flanks meet at or outside the root circle (radius %g)", d.root)
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

// buildEmbeddedTooth draws the four-curve natural tooth closed by an arc at the
// closing radius, at angle zero.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	rightPts, leftPts := flankPoints(s, left, right, true)
	n := len(rightPts)
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: tooth 0 right flank: %w", err)
	}
	s.CreateArc(center, rightPts[n-1], leftPts[n-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: tooth 0 left flank: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	if err := solveAndVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "embedded tooth", 4)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// filletRadius returns the derived root fillet radius.
func filletRadius(p Params, d dimensions) float64 {
	return d.fillet
}

// filletRoots fillets the ordinary root edges with a millimetre radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end on the saved root
// contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("spurgear: root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if onRoot(frame, roots, start) {
			corners = append(corners, start)
			continue
		}
		end := edge.End().Position().Value
		if onRoot(frame, roots, end) {
			corners = append(corners, end)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: root fillet: matched %d root edges, want %d", len(corners), len(roots))
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

// onRoot reports whether pos lies within 1e-6 of a saved root contact.
func onRoot(frame r3.Frame, roots []point, pos r3.Vec) bool {
	for _, root := range roots {
		if toWorld(frame, root).Sub(pos).Len() <= 1e-6 {
			return true
		}
	}
	return false
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

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body, which
// spans z=0 through p.Thickness in p's frame. It defines the spur parameters on
// doc first.
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
		return nil, fmt.Errorf("spurgear: bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool and cuts it from gear, using the document's
// existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: read tool_margin: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: read bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: extrude tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: cut: %w", err)
	}
	return cut, nil
}

// buildBoreProfile draws the bore circle on the plane margin below the lower
// cap, its diameter bound to bore_diameter.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: placement: %w", err)
	}
	lowered, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: shift frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(lowered)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: plane: %w", err)
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
	profile, err := singleProfile(s, "bore tool", 0)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// ChamferOuterCaps chamfers the outer loops of gear's two cap faces with the
// given setback in millimetres. Bore rims stay sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: cap chamfer: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps with setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return chamfered, nil
}
