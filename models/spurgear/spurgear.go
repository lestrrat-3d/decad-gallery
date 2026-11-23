// Package spurgear builds a cylindrical spur gear with straight involute teeth.
package spurgear

import (
	"context"
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
	// Backlash is the circumferential tooth thinning at the pitch circle.
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

// placement returns the gear's reference frame.
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

// toWorld maps a frame coordinate to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits shared by every entry point.
func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("module %g must be positive", p.Module)
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) {
		return fmt.Errorf("backlash %g must be finite", p.Backlash)
	}
	if math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash %g exceeds half the module %g", p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement frame")
	}
	n := float64(p.Teeth)
	rp := p.Module * n / 2
	rb := rp * math.Cos(p.PressureAngle)
	rr := (p.Module*n - 2.5*p.Module) / 2
	rt := (p.Module*n + 2*p.Module) / 2
	if rr <= 0 {
		return fmt.Errorf("root radius %g must be positive", rr)
	}
	if rb <= 0 {
		return fmt.Errorf("base radius %g must be positive", rb)
	}
	if rt <= rb {
		return fmt.Errorf("tip radius %g must exceed base radius %g", rt, rb)
	}
	if p.BoreDiameter >= 2*rr {
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*rr)
	}
	rf := 0.9 * rr * (math.Pi/n - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
	if math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("root fillet radius %g must be finite", rf)
	}
	return nil
}

// defineParameters writes the gear's parameter table.
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("parameter %q: %w", "module", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("parameter %q: %w", "teeth", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("parameter %q: %w", "pressure_angle", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("parameter %q: %w", "thickness", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("parameter %q: %w", "samples", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("parameter %q: %w", "bore_diameter", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("parameter %q: %w", "cap_chamfer", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("parameter %q: %w", "backlash", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "tool_margin", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "pitch_radius", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "base_radius", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "root_radius", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "tip_radius", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("parameter %q: %w", "involute_angle", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "fillet_radius", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter %q: %w", "bore_tool_length", err)
	}
	return nil
}

// DefineParameters writes the gear's parameter table into t without validating p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// Build builds the gear into doc and returns its single solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear through the variant's hooks.
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

// emitSolveVerify solves a sketch and requires it to be fully constrained.
func emitSolveVerify(ctx context.Context, s *sketch.Sketch, name string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve %s: %w", name, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify %s: %w", name, err)
	}
	return nil
}

// emitSingleProfile requires exactly one valid profile.
func emitSingleProfile(s *sketch.Sketch, name string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("%s: got %d profiles, want exactly one valid profile", name, len(profiles))
	}
	return profiles[0], nil
}

// assemble builds the gear from a prepared run: tooth, root disc, pattern,
// join, root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}

	disc, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := disc.CreateCircle(disc.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	disc.AddConstraint(radius)
	if err := disc.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root disc radius: %w", err)
	}
	if err := emitSolveVerify(ctx, disc, "root disc sketch"); err != nil {
		return nil, err
	}
	discProfile, err := emitSingleProfile(disc, "root disc sketch")
	if err != nil {
		return nil, err
	}
	discBody, err := decad.Extrude(ctx, r.doc, disc, discProfile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}

	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, discBody, append([]*decad.Body{tooth}, copies...)...)
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
			return nil, fmt.Errorf("read parameter %q: %w", "cap_chamfer", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spur gear: result is not a solid")
	}
	return body, nil
}

// BuildProfile builds the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := emitProfileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	d := derive(p)
	if d.base == 0 {
		return nil, nil, fmt.Errorf("gear profile: derive dimensions failed")
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile builds the whole-gear outline with every tooth turned by
// an extra twist about the sketch origin. It also returns the 2N root contacts.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := emitProfileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("gear profile: nonfinite twist %g", twist)
	}
	d := derive(p)
	if d.base == 0 {
		return nil, nil, nil, fmt.Errorf("gear profile: derive dimensions failed")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, profile, out, nil
}

func emitProfileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return fmt.Errorf("gear profile: nil context")
	}
	if world == nil {
		return fmt.Errorf("gear profile: nil sketch world")
	}
	if plane == nil {
		return fmt.Errorf("gear profile: nil plane")
	}
	return validate(p)
}

// buildGearProfile builds the whole-gear outline with no twist.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with each tooth turned
// by toothAngle(p, k) + twist. It dispatches to the embedded outline when the
// base circle lies inside the root circle.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return emitBuildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeft, firstRight *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		tooth, err := emitDrawOrdinaryTooth(s, p, d, toothAngle(p, k)+twist, prevLeft, k)
		if err != nil {
			return nil, nil, nil, err
		}
		roots = append(roots, tooth.rightRootXY, tooth.leftRootXY)
		if k == 0 {
			firstRight = tooth.rightRoot
		}
		prevLeft = tooth.leftRoot
	}
	s.CreateArc(s.Origin(), prevLeft, firstRight)
	if err := emitSolveVerify(ctx, s, "gear sketch"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := emitSingleProfile(s, "gear sketch")
	if err != nil {
		return nil, nil, nil, err
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline has %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// emitFlankSpline records a drawn flank spline and the tooth side it belongs to.
type emitFlankSpline struct {
	tooth int
	left  bool
}

// emitBuildEmbeddedOutline builds the whole-gear outline when the base circle
// lies inside the root circle. It returns the trimmed root contacts.
func emitBuildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear sketch: %w", err)
	}
	center := s.Origin()
	splines := make(map[*sketch.FitSpline]emitFlankSpline, 2*p.Teeth)
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
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPoints[n-1], leftPoints[n-1])
		ls, err := s.CreateFitSpline(leftPoints...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		splines[rs] = emitFlankSpline{tooth: k}
		splines[ls] = emitFlankSpline{tooth: k, left: true}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := emitSolveVerify(ctx, s, "embedded gear sketch"); err != nil {
		return nil, nil, nil, err
	}

	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded gear sketch has no bounded region")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded gear sketch region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded gear regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("embedded gear outline is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded gear outline has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded gear outline has %d boundary curves, expected %d",
			len(profile.Outer), want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		info, known := splines[flank]
		if !known {
			return nil, nil, nil, fmt.Errorf("embedded gear outline has an unknown flank spline")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded gear tooth %d: flank trim at the root circle is not certified",
				info.tooth)
		}
		slot := 2 * info.tooth
		if info.left {
			slot++
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded gear tooth %d: duplicate flank edge", info.tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded gear tooth %d: root contact radius %g differs from root radius %g",
				info.tooth, math.Hypot(x, y), d.root)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded gear tooth %d: missing root contact", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// emitFlankHalfAngle returns the angle between the tooth axis and the left
// flank where it meets radius r.
func emitFlankHalfAngle(d dimensions, teeth float64, r float64) float64 {
	beta := flankShift(d.pitch, d.base, teeth, d.backlash)
	ip := involutePoint(d.base, r)
	l := rotate(ip.x, -ip.y, beta)
	return math.Atan2(l.y, l.x)
}

// emitClosingRadius returns the radius of the arc that closes an embedded tooth.
func emitClosingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	if emitFlankHalfAngle(d, teeth, d.base) < limit {
		return d.base, nil
	}
	if emitFlankHalfAngle(d, teeth, d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: adjacent flanks overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if emitFlankHalfAngle(d, teeth, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + d.root) / 2, nil
}

// buildEmbeddedTooth builds the four-curve section of one embedded tooth at
// angle zero on plane.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := emitClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
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
	center := s.Origin()
	if _, err := s.CreateFitSpline(rightPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[n-1], leftPoints[n-1])
	if _, err := s.CreateFitSpline(leftPoints...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank spline: %w", err)
	}
	s.CreateArc(center, rightPoints[0], leftPoints[0])
	if err := emitSolveVerify(ctx, s, "embedded tooth sketch"); err != nil {
		return nil, nil, err
	}
	profile, err := emitSingleProfile(s, "embedded tooth sketch")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth section has %d boundary curves, expected 4",
			len(profile.Outer))
	}
	return s, profile, nil
}

// buildBoreProfile builds the bore tool's circle on a plane margin below the
// placed gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("define bore parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore placement: %w", err)
	}
	low, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(low)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool plane: %w", err)
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
	if err := emitSolveVerify(ctx, s, "bore sketch"); err != nil {
		return nil, nil, err
	}
	profile, err := emitSingleProfile(s, "bore sketch")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read parameter %q: %w", "tool_margin", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read parameter %q: %w", "bore_tool_length", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cut bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("cut bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("cut bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("cut bore: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both caps by setback. It checks no
// arguments.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers the outer loops of both caps of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("chamfer caps: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("chamfer caps: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer caps: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// filletRoots fillets the ordinary gear's root edges with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end at a saved root
// contact on the lower cap.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	near := func(v r3.Vec) bool {
		for _, w := range world {
			if v.Sub(w).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if near(start) {
			corners = append(corners, start)
			continue
		}
		end := edge.End().Position().Value
		if near(end) {
			corners = append(corners, end)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d axial edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return out, nil
}
