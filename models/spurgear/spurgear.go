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
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are in millimeters and angles in
// radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement is the reference point and plane. Nil selects the world XY
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
	}
}

// RejectPlacement returns an error naming model when p carries a Placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// validate applies the input checks and limits.
func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("spurgear: module %v must be positive", p.Module)
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("spurgear: thickness %v must be positive", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("spurgear: pressure angle %v must lie in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: invalid placement frame")
	}
	n := float64(p.Teeth)
	rp := p.Module * n / 2
	rb := rp * math.Cos(p.PressureAngle)
	rr := (p.Module*n - 2.5*p.Module) / 2
	rt := (p.Module*n + 2*p.Module) / 2
	if rr <= 0 {
		return fmt.Errorf("spurgear: root radius %v must be positive", rr)
	}
	if rb <= 0 {
		return fmt.Errorf("spurgear: base radius %v must be positive", rb)
	}
	if rt <= rb {
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", rt, rb)
	}
	if p.BoreDiameter >= 2*rr {
		return fmt.Errorf("spurgear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*rr)
	}
	rf := 0.9 * rr * (math.Pi/n - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
	if math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("spurgear: root fillet radius %v is not finite", rf)
	}
	return nil
}

// placement returns the reference frame of p.
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

// Build builds the gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear into doc, using v's hooks.
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

// assemble builds the root disc and teeth, joins them, and applies the root
// fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	thickness, err := r.doc.Param("thickness")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read thickness: %w", err)
	}
	extent := decad.Distance{D: thickness, Dir: decad.Along}
	embedded := r.d.base < r.d.root

	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth: %w", err)
	}

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
	if err := solveVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s, "root disc", 0)
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, extent)
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
		return nil, errors.New("spurgear: joined gear is not a solid")
	}

	switch {
	case v.RootFillet != nil:
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
	case r.d.fillet > 0:
		body, err = filletSpurRoots(ctx, r.doc, body, r, embedded)
	}
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
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
			return nil, fmt.Errorf("spurgear: read cap chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// guardProfile rejects nil arguments to the profile APIs.
func guardProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return errors.New("spurgear: nil context")
	}
	if world == nil {
		return errors.New("spurgear: nil world")
	}
	if plane == nil {
		return errors.New("spurgear: nil plane")
	}
	return nil
}

// solveVerify solves s and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	return nil
}

// singleProfile requires exactly one valid profile with outer outer curves
// (zero skips the curve count).
func singleProfile(s *sketch.Sketch, stage string, outer int) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, want 1", stage, len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	if outer > 0 && len(profile.Outer) != outer {
		return nil, fmt.Errorf("spurgear: %s: got %d outer curves, want %d", stage, len(profile.Outer), outer)
	}
	return profile, nil
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := guardProfile(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// buildGearProfile draws the whole-gear outline and returns the actual root
// contacts, right then left for each tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		t, err := drawTooth(s, d, left, right, k, lastLeft)
		if err != nil {
			return nil, nil, nil, err
		}
		roots = append(roots, rootContact(right[0], d), rootContact(left[0], d))
		if k == 0 {
			firstRight = t.rightRoot
		}
		lastLeft = t.leftRoot
	}
	s.CreateArc(center, lastLeft, firstRight)
	if err := solveVerify(ctx, s, "outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "outline", 0)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws every original flank and tip arc with a full
// root circle, unions the regions and records the exact root trims.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		n := len(right)
		rightPts := make([]*sketch.Point, n)
		leftPts := make([]*sketch.Point, n)
		for i := 0; i < n; i++ {
			rightPts[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		}
		for i := 0; i < n; i++ {
			if i == n-1 {
				leftPts[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				leftPts[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		_, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d: right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[n-1], leftPts[n-1])
		_, err = s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d: left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union has %d holes", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: got %d boundary curves, want %d",
			len(profile.Outer), 4*p.Teeth)
	}
	var roots []point
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, errors.New("spurgear: embedded outline: flank trim is not certified")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: root contact radius %v differs from root radius %v",
				math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if len(roots) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: got %d root contacts, want %d",
			len(roots), 2*p.Teeth)
	}
	return s, profile, roots, nil
}

// halfAngle returns the angle between the tooth axis and the left flank at
// radius r.
func halfAngle(d dimensions, teeth float64, r float64) float64 {
	beta := flankPhase(d.pitch, d.base, teeth)
	q := mirroredInvolute(d.base, r)
	l := rotate(q.x, q.y, beta)
	return math.Atan2(l.y, l.x)
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	n := float64(p.Teeth)
	limit := math.Pi / n
	if halfAngle(d, n, d.base) < limit {
		return d.base, nil
	}
	if halfAngle(d, n, d.root) >= limit {
		return 0, fmt.Errorf("spurgear: embedded tooth flanks overlap neighbours at the root circle (radius %v)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if halfAngle(d, n, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return ((lo+hi)/2 + d.root) / 2, nil
}

// buildEmbeddedTooth draws one embedded tooth closed by an arc about the gear
// axis.
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
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	n := len(right)
	rightPts := make([]*sketch.Point, n)
	leftPts := make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rightPts[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == 0 || i == n-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[n-1], leftPts[n-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	if err := solveVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "embedded tooth", 4)
	if err != nil {
		return nil, nil, err
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
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			w := toWorld(frame, root)
			if math.Abs(start.X-w.X) <= 1e-6 && math.Abs(start.Y-w.Y) <= 1e-6 && math.Abs(start.Z-w.Z) <= 1e-6 {
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
		return nil, fmt.Errorf("spurgear: fillet root edges: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet embedded root edges: %w", err)
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
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore plane frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
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
	if err := solveVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "bore tool", 0)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through the thickness in the gear frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	if body == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("spurgear: bore diameter %v must be positive", p.BoreDiameter)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, errors.New("spurgear: invalid placement frame")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool and cuts it from body. The document must
// already define the spur parameters.
func cutBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read tool margin: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read bore tool length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if distance <= 0 {
		return nil, fmt.Errorf("spurgear: chamfer distance %v must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps, leaving bore rims sharp.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}
