// Package spurgear builds a cylindrical involute spur gear with Decad.
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
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters and angles radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement gives the reference point and plane; nil selects the world XY frame.
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
	}
}

// RejectPlacement reports an error when p carries a Placement, for models that
// reuse Params without placement support.
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
	if !(p.Module > 0) {
		return fmt.Errorf("spurgear: module must be positive, got %v", p.Module)
	}
	if !(p.Thickness > 0) {
		return fmt.Errorf("spurgear: thickness must be positive, got %v", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	}
	if !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2) {
		return fmt.Errorf("spurgear: pressure angle must lie in (0, pi/2), got %v", p.PressureAngle)
	}
	if p.BoreDiameter < 0 || math.IsNaN(p.BoreDiameter) {
		return fmt.Errorf("spurgear: bore diameter must not be negative, got %v", p.BoreDiameter)
	}
	if p.CapChamfer < 0 || math.IsNaN(p.CapChamfer) {
		return fmt.Errorf("spurgear: cap chamfer must not be negative, got %v", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: invalid placement frame")
	}
	n := float64(p.Teeth)
	rp := p.Module * n / 2
	rb := rp * math.Cos(p.PressureAngle)
	rr := (p.Module*n - 2.5*p.Module) / 2
	rt := (p.Module*n + 2*p.Module) / 2
	if rr <= 0 || rb <= 0 || rt <= rb {
		return fmt.Errorf("spurgear: radii are not admissible (root %v, base %v, tip %v)", rr, rb, rt)
	}
	if p.BoreDiameter >= 2*rr {
		return fmt.Errorf("spurgear: bore diameter %v must be smaller than the root diameter %v", p.BoreDiameter, 2*rr)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	rf := 0.9 * rr * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("spurgear: root fillet radius is not finite")
	}
	return nil
}

// Build constructs the gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear in doc using v's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
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

// assemble builds the gear body from a prepared run.
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
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join disc and teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: joined gear is not a solid")
	}
	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	} else if r.d.fillet > 0 {
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
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
		return nil, errors.New("spurgear: finished gear is not a solid")
	}
	return body, nil
}

// solveAndVerify solves s and requires a fully constrained report.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s sketch: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s sketch: %w", what, err)
	}
	return nil
}

// onlyProfile returns the single valid profile of s.
func onlyProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s sketch has %d profiles, want 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s profile is not valid", what)
	}
	return profiles[0], nil
}

// buildRootDisc extrudes the root-disc circle through the thickness.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: bind root radius: %w", err)
	}
	if err := solveAndVerify(ctx, s, "root disc"); err != nil {
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

// toothPoints are the sketch points of one ordinary tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootR, rootL        point
}

// addToothPoints creates the dimensioned points of one ordinary tooth.
func addToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	n := len(right)
	t := toothPoints{right: make([]*sketch.Point, n), left: make([]*sketch.Point, n)}
	for i, q := range right {
		t.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == n-1 {
			t.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			t.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	t.rootL = rootContact(left[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.rootL.x, t.rootL.y)
	t.rootR = rootContact(right[0], d)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rootR.x, t.rootR.y)
	return t
}

// drawFlanks draws, in order, the right line and spline, the tip arc, the
// left spline and the left line of one tooth.
func drawFlanks(s *sketch.Sketch, t toothPoints, tooth int) error {
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("spurgear: tooth %d right flank: %w", tooth, err)
	}
	s.CreateArc(s.Origin(), t.right[len(t.right)-1], t.left[len(t.left)-1])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("spurgear: tooth %d left flank: %w", tooth, err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// buildGearProfile draws the whole-gear outline on plane. For an ordinary gear
// it also returns the 2N root contacts, right then left for each tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeft, firstRight *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		t := addToothPoints(s, left, right, d)
		roots = append(roots, t.rootR, t.rootL)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(s.Origin(), prevLeft, t.rightRoot)
		}
		if err := drawFlanks(s, t, k); err != nil {
			return nil, nil, nil, err
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(s.Origin(), prevLeft, firstRight)
	if err := solveAndVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the whole-gear outline when the base circle lies
// inside the root circle, trimming the original flanks at the root circle.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline sketch: %w", err)
	}
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
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
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded outline regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, errors.New("spurgear: embedded outline is not a hole-free valid profile")
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d curves, want %d",
			len(profile.Outer), 4*p.Teeth)
	}
	var contacts []point
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, errors.New("spurgear: embedded flank trim is not certified at the root circle")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded flank trim lies at radius %v, want %v",
				math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x: x, y: y})
	}
	if len(contacts) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d root contacts, want %d",
			len(contacts), 2*p.Teeth)
	}
	return s, profile, contacts, nil
}

// closingRadius finds the radius of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth float64) (float64, error) {
	limit := math.Pi / teeth
	h := func(r float64) float64 { return flankHalfAngle(d.pitch, d.base, teeth, r) }
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, errors.New("spurgear: adjacent teeth overlap at the root circle")
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
	return ((lo+hi)/2 + d.root) / 2, nil
}

// buildEmbeddedTooth draws one tooth whose flanks cross the root circle.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	teeth := float64(p.Teeth)
	rc, err := closingRadius(d, teeth)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, teeth, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth sketch: %w", err)
	}
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
		return nil, nil, fmt.Errorf("spurgear: embedded right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if err := solveAndVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth has %d curves, want 4", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildProfile draws the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: BuildProfile requires a context, world and plane")
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

// InvoluteFlanks samples the left and right flanks of one tooth, each running
// base to tip, rotated by angle.
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
	angle float64) ([][2]float64, [][2]float64, error) {
	for _, v := range []float64{pitch, base, tip, effectiveTeeth, angle} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, nil, errors.New("spurgear: flank inputs must be finite")
		}
	}
	if base <= 0 || pitch <= base || tip <= pitch || effectiveTeeth <= 0 || samples < 3 {
		return nil, nil, fmt.Errorf("spurgear: invalid flank inputs (pitch %v, base %v, tip %v, teeth %v, samples %d)",
			pitch, base, tip, effectiveTeeth, samples)
	}
	left, right := sampleFlanks(pitch, base, base, tip, effectiveTeeth, samples, angle)
	pairs := func(in []point) [][2]float64 {
		out := make([][2]float64, len(in))
		for i, q := range in {
			out[i] = [2]float64{q.x, q.y}
		}
		return out
	}
	return pairs(left), pairs(right), nil
}

// buildBoreProfile draws the circular bore tool section, margin below the
// lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
	}
	frame, err = shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bind bore diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s, "bore"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "bore")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the through-bore tool from gear. doc must hold the parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, errors.New("spurgear: CutThroughBore requires a context, document and body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("spurgear: bore diameter must be positive, got %v", p.BoreDiameter)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, errors.New("spurgear: invalid placement frame")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops of gear.
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
	if ctx == nil || gear == nil {
		return nil, errors.New("spurgear: ChamferOuterCaps requires a context and body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance must be positive, got %v", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// filletRoots fillets the ordinary root edges with a float radius.
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
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, w := range world {
			if start.Sub(w).Len() <= 1e-6 {
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
