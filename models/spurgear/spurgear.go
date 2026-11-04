// Package spurgear builds a cylindrical spur gear with straight involute
// teeth as a parametric decad model.
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

// Params are the spur gear inputs. Lengths are in millimeters and angles in
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
	// CapChamfer is the outer cap setback; zero omits the chamfer.
	CapChamfer float64
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

// placement returns the placement frame, or the world XY frame.
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

// toWorld maps a point in frame f to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// validate applies the input checks shared by Build, BuildProfile and
// BuildToothSection.
func validate(p Params) error {
	for name, v := range map[string]float64{
		"Module": p.Module, "PressureAngle": p.PressureAngle, "Thickness": p.Thickness,
		"BoreDiameter": p.BoreDiameter, "CapChamfer": p.CapChamfer,
	} {
		if !finite(v) {
			return fmt.Errorf("%s must be finite, got %v", name, v)
		}
	}
	switch {
	case p.Module <= 0:
		return fmt.Errorf("module %v must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %v must not be negative", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("invalid placement frame")
	}
	m, n := p.Module, float64(p.Teeth)
	pitch := m * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (m*n - 2.5*m) / 2
	tip := (m*n + 2*m) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("root radius %v must be positive", root)
	case base <= 0:
		return fmt.Errorf("base radius %v must be positive", base)
	case tip <= base:
		return fmt.Errorf("tip radius %v must exceed base radius %v", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if !finite(fillet) {
		return fmt.Errorf("root fillet radius %v is not finite", fillet)
	}
	return nil
}

// Build constructs the gear in doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear in doc, honoring the variant hooks.
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

// assemble builds the root disc and one tooth, patterns and joins them, then
// applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
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
		return nil, fmt.Errorf("join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join teeth: result is not a solid")
	}
	switch {
	case v.RootFillet != nil:
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
	case r.d.fillet > 0:
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	if r.p.BoreDiameter > 0 {
		if body, err = cutBore(ctx, r.doc, body, r.p); err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("read parameter cap_chamfer: %w", err)
		}
		if body, err = chamferCaps(ctx, body, setback); err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: result is not a solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root circle between the cap planes.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("root disc: bind radius: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	profile, err := singleProfile(s, 0)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root disc: extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("gear profile: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("gear profile: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("gear profile: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	d := derive(p)
	if d.root <= 0 {
		return nil, nil, errors.New("gear profile: cannot derive gear dimensions")
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// buildGearProfile draws the whole-gear outline and returns the root contacts
// of each tooth, right root then left root.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		tp := newToothPoints(s, d, left, right)
		roots = append(roots, tp.rightRootAt, tp.leftRootAt)
		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, prevLeft, tp.rightRoot)
		}
		if err := drawToothFlanks(s, tp); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d: %w", k, err)
		}
		prevLeft = tp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	profile, err := singleProfile(s, 0)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws every original flank and tip arc plus the full
// root circle, and unions all bounded regions into one outline.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	var flanks []*sketch.FitSpline
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rp := make([]*sketch.Point, len(right))
		for i, q := range right {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		lp := make([]*sketch.Point, len(left))
		for i, q := range left {
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank: %w", k, err)
		}
		flanks = append(flanks, rs, ls)
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d of %d is not valid", i, len(regions))
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
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: expected %d boundary curves, got %d",
			want, len(profile.Outer))
	}
	known := make(map[*sketch.FitSpline]bool, len(flanks))
	for _, f := range flanks {
		known[f] = true
	}
	var contacts []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !known[flank] {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d lies on an unknown spline", i)
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d has no certified root trim (partial %t, exact %t)",
				i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d trim radius %v differs from root radius %v",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if want := 2 * p.Teeth; len(contacts) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: expected %d root contacts, got %d", want, len(contacts))
	}
	return s, profile, contacts, nil
}

// embeddedClosingRadius returns the radius Rc of the arc that closes an
// embedded tooth section.
func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	beta := flankOffset(d.pitch, d.base, float64(p.Teeth))
	half := func(r float64) float64 {
		x, y := involutePoint(d.base, r)
		q := rotate(x, -y, beta)
		return math.Atan2(q.y, q.x)
	}
	limit := math.Pi / float64(p.Teeth)
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

// buildEmbeddedTooth draws one embedded tooth: right flank, tip arc, left
// flank and the closing arc at radius Rc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	rp := make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	last := len(rp) - 1
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	return s, profile, nil
}

// filletRoots rounds the ordinary root edges named by roots with radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue rounds the axial edges whose start matches a saved root
// contact. Selecting every axial edge would also pick flank joints.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if toWorld(frame, root).Equal(start, 1e-6) {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("match root edges: matched %d of %d root contacts", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, errors.New("match root edges: no root contacts")
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

// filletEmbedded rounds the 2*teeth concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the
// placed lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: shift frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
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
		return nil, nil, fmt.Errorf("bore tool: bind diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	profile, err := singleProfile(s, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the placed lower cap through the upper cap.
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
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("through bore: diameter %v must be positive", p.BoreDiameter)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, errors.New("through bore: invalid placement frame")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("through bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool and cuts it from the gear. The document
// must already define the parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("through bore: read parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("through bore: read parameter bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("through bore: extrude tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("through bore: cut: %w", err)
	}
	return cut, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with setback
// distance, leaving any bore rims sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("cap chamfer: nil body")
	}
	if !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("cap chamfer: setback %v must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps of gear.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return chamfered, nil
}
