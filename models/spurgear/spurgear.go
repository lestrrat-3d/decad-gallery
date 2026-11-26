package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// Params describes one cylindrical spur gear. Lengths are millimeters and
// angles are radians. The zero value is not a default request; use
// DefaultParams.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	// Placement gives the reference point and plane. Nil selects the world
	// XY frame at the origin.
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

// DefineParameters writes the parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// validate applies the input checks and limits.
func validate(p Params) error {
	for name, v := range map[string]float64{
		"module": p.Module, "pressure angle": p.PressureAngle, "thickness": p.Thickness,
		"bore diameter": p.BoreDiameter, "cap chamfer": p.CapChamfer,
	} {
		if !finite(v) {
			return fmt.Errorf("spur gear: nonfinite %s %v", name, v)
		}
	}
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spur gear: module %v must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spur gear: thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spur gear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spur gear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spur gear: pressure angle %v must lie in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spur gear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spur gear: cap chamfer %v must not be negative", p.CapChamfer)
	case !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spur gear: backlash %v must be finite with magnitude at most module/2", p.Backlash)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spur gear: placement is not a valid frame")
	}
	n := float64(p.Teeth)
	pitch := p.Module * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (p.Module*n - 2.5*p.Module) / 2
	tip := (p.Module*n + 2*p.Module) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("spur gear: root radius %v must be positive", root)
	case base <= 0:
		return fmt.Errorf("spur gear: base radius %v must be positive", base)
	case tip <= base:
		return fmt.Errorf("spur gear: tip radius %v must exceed base radius %v", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("spur gear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if !finite(fillet) {
		return fmt.Errorf("spur gear: root fillet radius %v is not finite", fillet)
	}
	return nil
}

// placement returns the placement frame of p, the world XY frame by default.
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

// toWorld maps sketch coordinates in frame f to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// Build builds a spur gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds a spur gear in doc through the variant pipeline.
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

// assemble builds the gear from a prepared run: tooth, root disc, pattern,
// join, root fillet, bore and cap chamfer.
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
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("tooth pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("join: result is not a solid")
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
			return nil, fmt.Errorf("parameter cap_chamfer: %w", err)
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

// buildRootDisc extrudes the root-disc circle on the lower plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root_radius: %w", err)
	}
	if err := solveVerified(ctx, s); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s, 0)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return body, nil
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// BuildTwistedProfile draws the whole-gear outline with every tooth turned by
// twist about the sketch origin. It also returns the 2N root contacts, right
// then left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("gear outline: nonfinite twist %v", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, pairs(roots), nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return errors.New("gear outline: nil context")
	}
	if world == nil {
		return errors.New("gear outline: nil world")
	}
	if plane == nil {
		return errors.New("gear outline: nil plane")
	}
	return validate(p)
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with each tooth turned
// by an extra twist, and returns the root contacts.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		t := addToothPoints(s, left, right, d)
		roots = append(roots, t.rootRight, t.rootLeft)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(s.Origin(), prevLeft, t.rightRoot)
		}
		if _, _, err := t.toothCurves(s, k); err != nil {
			return nil, nil, nil, err
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(s.Origin(), prevLeft, firstRight)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	profile, err := singleProfile(s, 6*p.Teeth)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the whole-gear outline when the base circle lies
// inside the root circle. It trims every flank at the root circle and returns
// the exact trimmed root contacts.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	slots := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPts, leftPts := addFlankPoints(s, left, right)
		last := p.Samples - 1
		rightSpline, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rightPts[last], leftPts[last])
		leftSpline, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		slots[rightSpline] = 2 * k
		slots[leftSpline] = 2*k + 1
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded gear outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, errors.New("embedded gear outline: no regions")
	}
	indices := make([]int, 0, len(profiles))
	for i, region := range profiles {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded gear outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded gear outline union: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, errors.New("embedded gear outline: union is not a valid hole-free region")
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded gear outline: got %d outer curves, expected %d",
			len(profile.Outer), 4*p.Teeth)
	}
	contacts, err := embeddedContacts(profile, slots, p.Teeth, d.root)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, contacts, nil
}

// embeddedContacts reads each flank's exact trim point at the root circle.
func embeddedContacts(profile *sketch.Profile, slots map[*sketch.FitSpline]int, teeth int,
	root float64) ([]point, error) {
	contacts := make([]point, 2*teeth)
	filled := make([]bool, 2*teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, known := slots[flank]
		if !known {
			return nil, errors.New("embedded gear outline: unknown flank edge")
		}
		if filled[slot] {
			return nil, fmt.Errorf("embedded gear outline: duplicate flank edge on tooth %d", slot/2)
		}
		if !edge.Partial || !edge.TExact {
			return nil, fmt.Errorf("embedded gear outline: root trim of tooth %d is not certified", slot/2)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-root) > 1e-8 {
			return nil, fmt.Errorf("embedded gear outline: tooth %d root contact is off the root circle", slot/2)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, fmt.Errorf("embedded gear outline: tooth %d has no root contact", slot/2)
		}
	}
	return contacts, nil
}

// closingRadius returns the radius Rc of the arc that closes an embedded
// tooth.
func closingRadius(d dimensions, teeth float64) (float64, error) {
	limit := math.Pi / teeth
	h := func(r float64) float64 { return flankAngle(d.pitch, d.base, teeth, d.backlash, r) }
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, errors.New("embedded tooth: flanks of adjacent teeth meet inside the root circle")
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if h(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + d.root) / 2, nil
}

// buildEmbeddedTooth draws one embedded tooth at angle 0 on plane. Its
// flanks start on a closing arc at radius Rc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	teeth := float64(p.Teeth)
	rc, err := closingRadius(d, teeth)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, teeth, d.backlash, p.Samples, 0)
	last := p.Samples - 1
	rightPts := make([]*sketch.Point, 0, p.Samples)
	for _, pt := range right {
		rightPts = append(rightPts, sketchcheck.DimensionPoint(s, pt.x, pt.y))
	}
	leftPts := make([]*sketch.Point, 0, p.Samples)
	for i, pt := range left {
		if i == 0 || i == last {
			leftPts = append(leftPts, sketchcheck.DimensionArcEnd(s, pt.x, pt.y))
			continue
		}
		leftPts = append(leftPts, sketchcheck.DimensionPoint(s, pt.x, pt.y))
	}
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth right flank: %w", err)
	}
	s.CreateArc(s.Origin(), rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth left flank: %w", err)
	}
	s.CreateArc(s.Origin(), rightPts[0], leftPts[0])
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	profile, err := singleProfile(s, 4)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges with a radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges whose endpoint lies on a saved
// root contact. Selecting every axial edge would also select flank joints.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet edges: %w", err)
	}
	want := make([]r3.Vec, len(roots))
	for i, root := range roots {
		want[i] = toWorld(frame, root)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		if corner, ok := matchRoot(edge, want); ok {
			corners = append(corners, corner)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d root edges, expected %d", len(corners), len(roots))
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

// matchRoot returns the endpoint of edge that lies within 1e-6 of a root
// position. It tests the start first, then the end.
func matchRoot(edge *decad.Edge, want []r3.Vec) (r3.Vec, bool) {
	for _, end := range []*decad.Vertex{edge.Start(), edge.End()} {
		pos := end.Position().Value
		for _, w := range want {
			if pos.Sub(w).Len() <= 1e-6 {
				return pos, true
			}
		}
	}
	return r3.Vec{}, false
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the
// placed lower cap.
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
		return nil, nil, fmt.Errorf("bore plane frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
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
		return nil, nil, fmt.Errorf("bind bore_diameter: %w", err)
	}
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore: %w", err)
	}
	profile, err := singleProfile(s, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("bore: %w", err)
	}
	return s, profile, nil
}

// cutBore cuts the through-bore using doc's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool extrude: %w", err)
	}
	body, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return body, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which spans z=0 through the thickness. It defines the spur parameters on
// doc first.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("bore: nil document")
	}
	if body == nil {
		return nil, errors.New("bore: nil body")
	}
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("bore: diameter %v must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers the outer loops of both caps. The bore loop stays
// sharp. Its callers check the arguments.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	body, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return body, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("cap chamfer: nil body")
	}
	if !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("cap chamfer: distance %v must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
