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

// Params holds the spur gear inputs. Lengths are in millimeters and angles in
// radians. Zero fields are not defaults; use DefaultParams for those.
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

// DefaultParams returns the default spur gear parameters.
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

// RejectPlacement returns an error naming model when p carries a placement.
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

// shifted returns f moved by dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a frame coordinate on the frame's plane to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// validate applies the input checks and limits.
func validate(p Params) error {
	switch {
	case !finite(p.Module) || p.Module <= 0:
		return fmt.Errorf("module %g must be positive and finite", p.Module)
	case !finite(p.Thickness) || p.Thickness <= 0:
		return fmt.Errorf("thickness %g must be positive and finite", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case !finite(p.PressureAngle) || p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	case !finite(p.BoreDiameter) || p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %g must be finite and not negative", p.BoreDiameter)
	case !finite(p.CapChamfer) || p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %g must be finite and not negative", p.CapChamfer)
	case !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("backlash %g must be finite with magnitude at most module/2 (%g)",
			p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("placement is not a valid frame")
	}
	d := derive(p)
	switch {
	case !(d.root > 0):
		return fmt.Errorf("root radius %g must be positive", d.root)
	case !(d.base > 0):
		return fmt.Errorf("base radius %g must be positive", d.base)
	case !(d.tip > d.base):
		return fmt.Errorf("tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("bore diameter %g must be smaller than the root diameter %g",
			p.BoreDiameter, 2*d.root)
	case !finite(d.fillet):
		return fmt.Errorf("root fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// solveAndVerify solves the sketch and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve %s sketch: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify %s sketch: %w", what, err)
	}
	return nil
}

// onlyProfile requires the sketch to hold exactly one valid profile.
func onlyProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s sketch: got %d profiles, expected 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s sketch: profile is not valid", what)
	}
	return profiles[0], nil
}

// toothPoints holds the dimensioned points of one ordinary tooth.
type toothPoints struct {
	right, left             []*sketch.Point
	rightRoot, leftRoot     *sketch.Point
	rightRootAt, leftRootAt point
}

// addToothPoints creates the flank and root points of one ordinary tooth. Every
// right point takes both dimensions. Every left point takes both except its tip.
// The left root is created before the right root, and takes one dimension.
func addToothPoints(s *sketch.Sketch, d dimensions, left, right []point) toothPoints {
	tp := toothPoints{
		right: make([]*sketch.Point, len(right)),
		left:  make([]*sketch.Point, len(left)),
	}
	for i, pt := range right {
		tp.right[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	for i, pt := range left {
		if i == len(left)-1 {
			tp.left[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
		} else {
			tp.left[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
	}
	tp.leftRootAt = rootContact(left[0], d)
	tp.leftRoot = sketchcheck.DimensionArcEnd(s, tp.leftRootAt.x, tp.leftRootAt.y)
	tp.rightRootAt = rootContact(right[0], d)
	tp.rightRoot = sketchcheck.DimensionPoint(s, tp.rightRootAt.x, tp.rightRootAt.y)
	return tp
}

// BuildProfile builds the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("build profile: context, world and plane must not be nil")
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("build profile: %w", err)
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile builds the whole-gear outline with every tooth turned by
// an extra twist, and returns its 2N root contacts: the right root then the
// left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, errors.New("build twisted profile: context, world and plane must not be nil")
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("build twisted profile: twist %g must be finite", twist)
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, fmt.Errorf("build twisted profile: %w", err)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

// buildGearProfile builds the whole-gear outline without twist.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted builds the whole-gear outline with every tooth turned
// by an extra twist, and returns the root contacts the profile saved.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		tp := addToothPoints(s, d, left, right)
		roots = append(roots, tp.rightRootAt, tp.leftRootAt)
		if k > 0 {
			s.CreateArc(center, prevLeft, tp.rightRoot)
		} else {
			firstRight = tp.rightRoot
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, tp.right[p.Samples-1], tp.left[p.Samples-1])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		prevLeft = tp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// flankID names one flank of one tooth in an embedded outline.
type flankID struct {
	tooth int
	left  bool
}

// buildEmbeddedOutline builds the whole-gear outline of the embedded branch.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear outline sketch: %w", err)
	}
	center := s.Origin()
	flanks := make(map[*sketch.FitSpline]flankID, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPts := make([]*sketch.Point, len(right))
		for i, pt := range right {
			rightPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
		leftPts := make([]*sketch.Point, len(left))
		for i, pt := range left {
			if i == len(left)-1 {
				leftPts[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			} else {
				leftPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
			}
		}
		rightSpline, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		flanks[rightSpline] = flankID{tooth: k}
		s.CreateArc(center, rightPts[p.Samples-1], leftPts[p.Samples-1])
		leftSpline, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		flanks[leftSpline] = flankID{tooth: k, left: true}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, errors.New("embedded gear outline: no region")
	}
	indices := make([]int, len(profiles))
	for i, region := range profiles {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded gear outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded gear outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded gear outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded gear outline: union profile has %d holes",
			len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded gear outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		id, ok := flanks[spline]
		if !ok {
			return nil, nil, nil, errors.New("embedded gear outline: boundary spline is not a tooth flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf(
				"embedded gear outline: tooth %d root trim is not certified", id.tooth)
		}
		slot := 2 * id.tooth
		if id.left {
			slot++
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf(
				"embedded gear outline: tooth %d flank trimmed more than once", id.tooth)
		}
		x, y := spline.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf(
				"embedded gear outline: tooth %d root contact lies at radius %g, expected %g",
				id.tooth, math.Hypot(x, y), d.root)
		}
		contacts[slot] = point{x: x, y: y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded gear outline: tooth %d has no root contact", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		m := mirroredInvolute(d.base, r)
		l := rotate(m.x, m.y, beta)
		return math.Atan2(l.y, l.x)
	}
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: adjacent flanks meet at or inside the root circle (radius %g)",
			d.root)
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

// buildEmbeddedTooth builds the four-curve section of one embedded tooth at
// angle zero on plane.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rightPts := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == 0 || i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
		}
	}
	last := p.Samples - 1
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	if err := solveAndVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d boundary curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// Build builds the gear into doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear into doc, through one shared pipeline.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("build spur gear: nil context")
	}
	if doc == nil {
		return nil, errors.New("build spur gear: nil document")
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

// assemble joins one tooth, repeated around the axis, to the root disc and
// applies the finishing features.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root disc radius: %w", err)
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
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join disc and teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join disc and teeth: result is not a solid")
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
			return nil, fmt.Errorf("read parameter %q: %w", "cap_chamfer", err)
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

// filletRoots fillets the ordinary root edges named by roots with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start or end on a saved root contact.
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
	matches := func(pos r3.Vec) bool {
		for _, w := range world {
			if pos.Sub(w).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	var corners []r3.Vec
	for _, edge := range edges {
		if start := edge.Start().Position().Value; matches(start) {
			corners = append(corners, start)
		} else if end := edge.End().Position().Value; matches(end) {
			corners = append(corners, end)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root edges: matched %d, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, errors.New("root edges: no root contacts")
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile builds the sketch of the bore tool on a plane margin below
// the placed gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
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

// cutBore cuts the through-bore tool from gear. The document already holds the
// parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read parameter %q: %w", "tool_margin", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read parameter %q: %w", "bore_tool_length", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
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

// CutThroughBore cuts a centered through-bore of p's diameter from body, which
// must span z=0 through the thickness of p in the placement frame.
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
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("cut through bore: bore diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loop of both caps of gear. The bore loop stays sharp.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with the given setback.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("chamfer outer caps: nil context")
	}
	if gear == nil {
		return nil, errors.New("chamfer outer caps: nil body")
	}
	if !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("chamfer outer caps: distance %g must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
