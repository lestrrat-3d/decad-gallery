// Package spurgear builds a straight involute spur gear.
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

// Params describes a spur gear. Lengths are millimeters and angles radians.
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
	// CapChamfer is the outer cap setback; zero omits the chamfer.
	CapChamfer float64
	// Backlash thins each tooth circumferentially at the pitch circle.
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

// RejectPlacement returns an error naming model when p carries a placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// DefineParameters writes the parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
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

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// validate applies the input checks and limits shared by every entry point.
func validate(p Params) error {
	if !(p.Module > 0) || !finite(p.Module) {
		return fmt.Errorf("module %g must be positive and finite", p.Module)
	}
	if !(p.Thickness > 0) || !finite(p.Thickness) {
		return fmt.Errorf("thickness %g must be positive and finite", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	}
	if !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2) {
		return fmt.Errorf("pressure angle %g must be in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 || !finite(p.BoreDiameter) {
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 || !finite(p.CapChamfer) {
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash %g must be finite and at most module/2 in magnitude", p.Backlash)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement frame")
	}
	d := derive(p)
	if d.root <= 0 {
		return fmt.Errorf("root radius %g must be positive", d.root)
	}
	if d.base <= 0 {
		return fmt.Errorf("base radius %g must be positive", d.base)
	}
	if d.tip <= d.base {
		return fmt.Errorf("tip radius %g must exceed base radius %g", d.tip, d.base)
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*d.root)
	}
	if !finite(d.fillet) {
		return fmt.Errorf("fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// solveVerify solves the sketch and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s verify: %w", stage, err)
	}
	return nil
}

// onlyProfile requires exactly one valid profile.
func onlyProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%d profiles, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("profile is not valid")
	}
	return profiles[0], nil
}

// flankPoints holds the sketch points of one ordinary tooth.
type flankPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootRight, rootLeft point
}

// createOrdinaryPoints creates the dimensioned flank and root points of one
// ordinary tooth. The left root is created before the right root.
func createOrdinaryPoints(s *sketch.Sketch, d dimensions, left, right []point) flankPoints {
	var fp flankPoints
	for _, pt := range right {
		fp.right = append(fp.right, sketchcheck.DimensionPoint(s, pt.x, pt.y))
	}
	for i, pt := range left {
		if i == len(left)-1 {
			fp.left = append(fp.left, sketchcheck.DimensionArcEnd(s, pt.x, pt.y))
		} else {
			fp.left = append(fp.left, sketchcheck.DimensionPoint(s, pt.x, pt.y))
		}
	}
	fp.rootLeft = rootContact(left[0], d)
	fp.leftRoot = sketchcheck.DimensionArcEnd(s, fp.rootLeft.x, fp.rootLeft.y)
	fp.rootRight = rootContact(right[0], d)
	fp.rightRoot = sketchcheck.DimensionPoint(s, fp.rootRight.x, fp.rootRight.y)
	return fp
}

// buildGearProfile builds the whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with every tooth
// turned by an extra twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("outline sketch: %w", err)
	}
	center := s.Origin()
	n := p.Samples
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		angle := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
		fp := createOrdinaryPoints(s, d, left, right)
		roots = append(roots, fp.rootRight, fp.rootLeft)
		if k > 0 {
			s.CreateArc(center, prevLeft, fp.rightRoot)
		} else {
			firstRight = fp.rightRoot
		}
		s.CreateLine(fp.rightRoot, fp.right[0])
		if _, err := s.CreateFitSpline(fp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, fp.right[n-1], fp.left[n-1])
		if _, err := s.CreateFitSpline(fp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(fp.left[0], fp.leftRoot)
		prevLeft = fp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveVerify(ctx, s, "outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("outline: %w", err)
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("outline: %d boundary curves, expected %d", len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// splineSlot names one flank spline of the embedded outline.
type splineSlot struct{ tooth, side int }

// buildEmbeddedOutline builds the whole-gear outline when the base circle is
// inside the root circle.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline sketch: %w", err)
	}
	center := s.Origin()
	n := p.Samples
	slots := make(map[*sketch.FitSpline]splineSlot, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		angle := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
		var rp, lp []*sketch.Point
		for _, pt := range right {
			rp = append(rp, sketchcheck.DimensionPoint(s, pt.x, pt.y))
		}
		for i, pt := range left {
			if i == len(left)-1 {
				lp = append(lp, sketchcheck.DimensionArcEnd(s, pt.x, pt.y))
			} else {
				lp = append(lp, sketchcheck.DimensionPoint(s, pt.x, pt.y))
			}
		}
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[n-1], lp[n-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		slots[rs] = splineSlot{k, 0}
		slots[ls] = splineSlot{k, 1}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: no bounded region")
	}
	indices := make([]int, 0, len(regions))
	for i, r := range regions {
		if !r.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline union: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: %d boundary curves, expected %d",
			len(profile.Outer), want)
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
			return nil, nil, nil, fmt.Errorf("embedded outline: unknown flank spline edge")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d flank trim is not certified", slot.tooth)
		}
		idx := 2*slot.tooth + slot.side
		if filled[idx] {
			return nil, nil, nil, fmt.Errorf("embedded outline: duplicate flank edge for tooth %d", slot.tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d root contact is off the root circle",
				slot.tooth)
		}
		contacts[idx] = point{x, y}
		filled[idx] = true
	}
	for idx, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d has no root contact", idx/2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		m := mirroredInvolute(d.base, r)
		q := rotate(m.x, m.y, beta)
		return math.Atan2(q.y, q.x)
	}
	limit := math.Pi / teeth
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: flanks of adjacent teeth overlap at the root circle")
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

// buildEmbeddedTooth draws one embedded tooth at angle zero.
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
	n := p.Samples
	var rp, lp []*sketch.Point
	for _, pt := range right {
		rp = append(rp, sketchcheck.DimensionPoint(s, pt.x, pt.y))
	}
	for i, pt := range left {
		if i == 0 || i == n-1 {
			lp = append(lp, sketchcheck.DimensionArcEnd(s, pt.x, pt.y))
		} else {
			lp = append(lp, sketchcheck.DimensionPoint(s, pt.x, pt.y))
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(center, rp[n-1], lp[n-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: %d boundary curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// buildBoreProfile draws the through-bore tool section on a plane margin
// below the gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore placement: %w", err)
	}
	lowered, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(lowered)
	if err != nil {
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore diameter binding: %w", err)
	}
	if err := solveVerify(ctx, s, "bore"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("bore: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using the document's existing parameters.
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
		return nil, fmt.Errorf("bore tool extrude: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore from a gear body spanning z=0
// through the thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("bore: nil body")
	}
	if !(p.BoreDiameter > 0) || !finite(p.BoreDiameter) {
		return nil, fmt.Errorf("bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of a gear body.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("chamfer: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("chamfer: nil body")
	}
	if !(distance > 0) || !finite(distance) {
		return nil, fmt.Errorf("chamfer: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
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
	targets := make([]r3.Vec, len(roots))
	for i, root := range roots {
		targets[i] = toWorld(frame, root)
	}
	near := func(pos r3.Vec) bool {
		for _, t := range targets {
			if pos.Sub(t).Len() <= 1e-6 {
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
		return nil, fmt.Errorf("root edges: matched %d, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return out, nil
}

// BuildProfile builds the unfilleted whole-gear outline on plane.
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

// BuildTwistedProfile builds the whole-gear outline with every tooth turned by
// twist, and returns its root contacts.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("twisted profile: nonfinite twist")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, r := range roots {
		out[i] = [2]float64{r.x, r.y}
	}
	return s, profile, out, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return fmt.Errorf("profile: nil context")
	}
	if world == nil {
		return fmt.Errorf("profile: nil world")
	}
	if plane == nil {
		return fmt.Errorf("profile: nil plane")
	}
	if err := validate(p); err != nil {
		return fmt.Errorf("profile: %w", err)
	}
	return nil
}

// Build builds the gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear in doc using the variant's hooks.
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

// assemble extrudes the root disc, patterns and joins the teeth, and applies
// the finishing features.
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
	if err := solveVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root disc extrude: %w", err)
	}

	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
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
		return nil, fmt.Errorf("spur gear: result is not a solid")
	}
	return body, nil
}
