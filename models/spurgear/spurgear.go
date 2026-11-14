// Package spurgear builds a cylindrical spur gear with straight involute teeth.
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

// Params describes one spur gear. Lengths are in millimeters and angles in radians.
type Params struct {
	Module        float64 // tooth module
	Teeth         int     // tooth count
	PressureAngle float64 // pressure angle
	Thickness     float64 // axial thickness
	Samples       int     // fit points per flank
	BoreDiameter  float64 // centered through-bore diameter; zero omits the bore
	CapChamfer    float64 // equal-distance outer cap setback; zero omits the chamfer
	Backlash      float64 // circumferential tooth thinning at the pitch circle
	Placement     *r3.Frame
}

// DefaultParams returns the default gear parameters.
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

// placement returns the placement frame of p, or the world XY frame.
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

// toWorld maps a frame coordinate on the reference plane to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits of instructions.md.
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
		return fmt.Errorf("pressure angle %g must be in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash %g must be finite and within half the module %g", p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("placement frame is not valid")
	}
	n := float64(p.Teeth)
	pitch := p.Module * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (p.Module*n - 2.5*p.Module) / 2
	tip := (p.Module*n + 2*p.Module) / 2
	if root <= 0 || base <= 0 || tip <= base {
		return fmt.Errorf("invalid radii: root %g, base %g, tip %g", root, base, tip)
	}
	if p.BoreDiameter >= 2*root {
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if math.IsNaN(fillet) || math.IsInf(fillet, 0) {
		return fmt.Errorf("root fillet radius %g is not finite", fillet)
	}
	return nil
}

// validatedDimensions validates p and returns its derived dimensions.
func validatedDimensions(p Params) (dimensions, error) {
	if err := validate(p); err != nil {
		return dimensions{}, err
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 {
		return dimensions{}, fmt.Errorf("derive dimensions: parameter table did not resolve")
	}
	return d, nil
}

// Build builds one spur gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds one spur gear into doc using the variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear build: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear build: nil document")
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

// assemble builds the tooth, root disc, pattern, join, fillet, bore and chamfer.
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
		return nil, fmt.Errorf("tooth join: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("tooth join: result is not a solid")
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
		setback, perr := r.doc.Param("cap_chamfer")
		if perr != nil {
			return nil, fmt.Errorf("parameter cap_chamfer: %w", perr)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// buildRootDisc extrudes the root-disc circle on the lower cap plane.
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
	if err := solveVerified(ctx, s, "root disc sketch"); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s, "root disc sketch")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	return disc, nil
}

// solveVerified solves s and requires it to be fully constrained.
func solveVerified(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", stage, err)
	}
	return nil
}

// singleProfile returns the only profile of s and requires it to be valid.
func singleProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s: got %d profiles, want 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// toothPoints holds the sketch points of one tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootRight, rootLeft point
}

// createToothPoints creates the dimensioned flank and root points of one tooth.
func createToothPoints(s *sketch.Sketch, left, right []point, d dimensions) toothPoints {
	var t toothPoints
	for _, q := range right {
		t.right = append(t.right, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	for i, q := range left {
		if i == len(left)-1 {
			t.left = append(t.left, sketchcheck.DimensionArcEnd(s, q.x, q.y))
		} else {
			t.left = append(t.left, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
	}
	t.rootLeft = rootContact(left[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.rootLeft.x, t.rootLeft.y)
	t.rootRight = rootContact(right[0], d)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rootRight.x, t.rootRight.y)
	return t
}

// drawToothCurves draws the two radial lines, the two flank splines and the tip arc.
func drawToothCurves(s *sketch.Sketch, t toothPoints) error {
	last := len(t.right) - 1
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), t.right[last], t.left[last])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("left flank spline: %w", err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// buildGearProfile builds the unfilleted whole-gear outline on plane.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with every tooth turned
// by an extra twist. It returns the 2N root contacts, right then left per tooth.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	teeth := float64(p.Teeth)
	roots := make([]point, 0, 2*p.Teeth)
	var firstRightRoot, prevLeftRoot *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, teeth, d.backlash, p.Samples,
			toothAngle(p, k)+twist)
		t := createToothPoints(s, left, right, d)
		roots = append(roots, t.rootRight, t.rootLeft)
		if k == 0 {
			firstRightRoot = t.rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, t.rightRoot)
		}
		if err := drawToothCurves(s, t); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d: %w", k, err)
		}
		prevLeftRoot = t.leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)
	if err := solveVerified(ctx, s, "gear outline sketch"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "gear outline sketch")
	if err != nil {
		return nil, nil, nil, err
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: got %d boundary curves, want %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// flankKey identifies one flank spline of the embedded outline.
type flankKey struct {
	tooth int
	right bool
}

// buildEmbeddedOutline builds the whole-gear outline for the embedded branch.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	teeth := float64(p.Teeth)
	flanks := make(map[*sketch.FitSpline]flankKey, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, teeth, d.backlash, p.Samples,
			toothAngle(p, k)+twist)
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
		rs, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		ls, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank spline: %w", k, err)
		}
		flanks[rs] = flankKey{tooth: k, right: true}
		flanks[ls] = flankKey{tooth: k, right: false}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerified(ctx, s, "embedded outline sketch"); err != nil {
		return nil, nil, nil, err
	}

	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: no bounded region")
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
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, want %d",
			len(profile.Outer), want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		key, ok := flanks[spline]
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: boundary spline belongs to no tooth")
		}
		side, slot := "left", 2*key.tooth+1
		if key.right {
			side, slot = "right", 2*key.tooth
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d %s flank trim is not certified",
				key.tooth, side)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d %s flank has two trimmed edges",
				key.tooth, side)
		}
		x, y := spline.Eval(edge.TStart)
		if dev := math.Abs(math.Hypot(x, y) - d.root); dev > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d %s root contact is %g off the root circle",
				key.tooth, side, dev)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has no trimmed root contact in slot %d",
				slot/2, slot)
		}
	}
	return s, profile, contacts, nil
}

// buildEmbeddedTooth builds the four-curve tooth section of the embedded branch.
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
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	if err := solveVerified(ctx, s, "embedded tooth sketch"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "embedded tooth sketch")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d boundary curves, want 4", len(profile.Outer))
	}
	return s, profile, nil
}

// closingRadius returns the radius Rc of the arc that closes the embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	if halfAngle(d, teeth, d.base) < limit {
		return d.base, nil
	}
	if halfAngle(d, teeth, d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: flanks of adjacent teeth overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if halfAngle(d, teeth, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// BuildProfile builds the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	d, err := validatedDimensions(p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile builds the whole-gear outline with every tooth turned by
// twist about the sketch origin, and returns its 2N root contacts.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist %g must be finite", twist)
	}
	d, err := validatedDimensions(p)
	if err != nil {
		return nil, nil, nil, err
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

// checkProfileArgs rejects nil context, world or plane.
func checkProfileArgs(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	if world == nil {
		return fmt.Errorf("nil sketch world")
	}
	if plane == nil {
		return fmt.Errorf("nil sketch plane")
	}
	return nil
}

// DefineParameters writes the parameter table of p into t. It does not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// filletRoots fillets the ordinary-branch root edges at the saved roots.
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
	worldRoots := make([]r3.Vec, len(roots))
	for i, root := range roots {
		worldRoots[i] = toWorld(frame, root)
	}
	near := func(pos r3.Vec) bool {
		for _, w := range worldRoots {
			if pos.Sub(w).Len() <= 1e-6 {
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
	if len(corners) != len(roots) || len(corners) == 0 {
		return nil, fmt.Errorf("matched %d axial root edges, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of the embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile builds the bore tool sketch on a plane margin below the
// placed lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore sketch: bind bore_diameter: %w", err)
	}
	if err := solveVerified(ctx, s, "bore sketch"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "bore sketch")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the document's existing parameter table.
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
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body, which
// must span z=0 through the gear thickness in the placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("cut through bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("cut through bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("cut through bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("cut through bore: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops of gear by setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("chamfer outer caps: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("chamfer outer caps: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer outer caps: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
