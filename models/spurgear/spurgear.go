// Package spurgear builds a parametric spur gear with straight involute teeth.
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

// Params describes one cylindrical spur gear. Lengths are millimeters and
// angles are radians. Placement anchors the gear: its origin is the reference
// point, its U and V axes span the lower cap plane, and the gear axis runs
// along its normal. A nil Placement selects the world XY frame at the origin.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Placement     *r3.Frame
}

// DefaultParams returns the default 17-tooth, module-1 gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

// RejectPlacement returns an error naming model when Placement is set.
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

// validate applies the input checks shared by every public entry point.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("spur gear: module %g must be positive", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("spur gear: thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spur gear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spur gear: samples %d must be at least 3", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("spur gear: pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spur gear: bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spur gear: cap chamfer %g must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spur gear: invalid placement frame")
	}
	d := derive(p)
	switch {
	case !(d.root > 0):
		return fmt.Errorf("spur gear: root radius %g must be positive", d.root)
	case !(d.base > 0):
		return fmt.Errorf("spur gear: base radius %g must be positive", d.base)
	case !(d.tip > d.base):
		return fmt.Errorf("spur gear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spur gear: bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spur gear: root fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// Build constructs the gear body in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear body, delegating to the local pipeline when
// LocalBuild is set.
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

// assemble builds the tooth, the root disc, the pattern and the join, then
// applies the root fillet, the bore and the cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("spur gear tooth: %w", err)
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spur gear tooth pattern: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spur gear join: %w", err)
	}
	if !gear.IsSolid() {
		return nil, errors.New("spur gear join: result is not a solid")
	}
	embedded := r.d.base < r.d.root
	if v.RootFillet != nil {
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spur gear root fillet: %w", err)
		}
	} else if r.d.fillet > 0 {
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, embedded)
		if err != nil {
			return nil, fmt.Errorf("spur gear root fillet: %w", err)
		}
	}
	if r.p.BoreDiameter > 0 {
		gear, err = cutBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spur gear cap chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, errors.New("spur gear: final body is not a solid")
	}
	return gear, nil
}

// buildRootDisc extrudes the root-disc circle through the gear thickness.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spur gear root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spur gear root disc radius binding: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spur gear root disc extrusion: %w", err)
	}
	return disc, nil
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s sketch solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s sketch verification: %w", stage, err)
	}
	return nil
}

// solveSingleProfile solves and verifies s, then returns its one valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	if err := solveAndVerify(ctx, s, stage); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("%s sketch: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("%s sketch: profile is not valid", stage)
	}
	return profiles[0], nil
}

// flankPoints creates the dimensioned flank points of one tooth. Every right
// point and every left point except the last receives both coordinate
// dimensions; the last left point (the tip) receives one.
func flankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
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
	return lp, rp
}

// ordinaryTooth holds the points of one ordinary tooth section.
type ordinaryTooth struct {
	left, right         []*sketch.Point
	leftRoot, rightRoot *sketch.Point
	leftRootXY          point
	rightRootXY         point
}

// ordinaryToothPoints samples one tooth at angle and creates its points. The
// left root is created before the right root.
func ordinaryToothPoints(s *sketch.Sketch, p Params, d dimensions, angle float64) ordinaryTooth {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, angle)
	var t ordinaryTooth
	t.left, t.right = flankPoints(s, left, right)
	t.leftRootXY = rootContact(left[0], d)
	t.rightRootXY = rootContact(right[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftRootXY.x, t.leftRootXY.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightRootXY.x, t.rightRootXY.y)
	return t
}

// drawFlanks draws the right radial line, right spline, tip arc, left spline
// and left radial line of one ordinary tooth.
func drawFlanks(s *sketch.Sketch, t ordinaryTooth, label string) error {
	last := len(t.right) - 1
	s.CreateLine(t.rightRoot, t.right[0])
	if _, err := s.CreateFitSpline(t.right...); err != nil {
		return fmt.Errorf("%s right flank spline: %w", label, err)
	}
	s.CreateArc(s.Origin(), t.right[last], t.left[last])
	if _, err := s.CreateFitSpline(t.left...); err != nil {
		return fmt.Errorf("%s left flank spline: %w", label, err)
	}
	s.CreateLine(t.left[0], t.leftRoot)
	return nil
}

// buildGearProfile draws the whole-gear outline on plane. It returns the
// sketch, its profile and the actual root contacts of every tooth, right root
// then left root.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spur gear outline sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		t := ordinaryToothPoints(s, p, d, toothAngle(p, k))
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(s.Origin(), prevLeft, t.rightRoot)
		}
		if err := drawFlanks(s, t, fmt.Sprintf("spur gear outline tooth %d", k)); err != nil {
			return nil, nil, nil, err
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(s.Origin(), prevLeft, firstRight)
	profile, err := solveSingleProfile(ctx, s, "spur gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the whole-gear outline for Rb < Rr: every
// original flank and tip arc plus one root circle, joined into one profile.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spur gear embedded outline sketch: %w", err)
	}
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		lp, rp := flankPoints(s, left, right)
		last := p.Samples - 1
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spur gear embedded outline tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[last], lp[last])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spur gear embedded outline tooth %d left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "spur gear embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("spur gear embedded outline: sketch has no regions")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spur gear embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spur gear embedded outline union: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spur gear embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spur gear embedded outline: union has %d holes, expected 0", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spur gear embedded outline: got %d boundary curves, expected %d",
			len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spur gear embedded outline: flank edge %d has no certified root trim", i)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spur gear embedded outline: flank edge %d trim radius %g, expected root radius %g",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x: x, y: y})
	}
	if len(contacts) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spur gear embedded outline: got %d root contacts, expected %d",
			len(contacts), 2*p.Teeth)
	}
	return s, profile, contacts, nil
}

// halfAngle returns the angle between the tooth axis and the left flank where
// the flank meets radius r.
func halfAngle(d dimensions, teeth int, r float64) float64 {
	beta := flankShift(d.pitch, d.base, float64(teeth))
	ip := involuteAt(d.base, r)
	q := rotate(ip.x, -ip.y, beta)
	return math.Atan2(q.y, q.x)
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth int) (float64, error) {
	limit := math.Pi / float64(teeth)
	if halfAngle(d, teeth, d.base) < limit {
		return d.base, nil
	}
	if halfAngle(d, teeth, d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: adjacent flanks overlap at the root circle (radius %g)", d.root)
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

// buildEmbeddedTooth draws one embedded tooth: both flanks from the closing
// radius to the tip, the tip arc and the closing arc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(d, p.Teeth)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spur gear embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	last := len(right) - 1
	rp := make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp := make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == 0 || i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spur gear embedded tooth right flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spur gear embedded tooth left flank spline: %w", err)
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	profile, err := solveSingleProfile(ctx, s, "spur gear embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if len(profile.Outer) != 4 {
		return nil, nil, fmt.Errorf("spur gear embedded tooth: got %d boundary curves, expected 4", len(profile.Outer))
	}
	return s, profile, nil
}

// BuildProfile draws the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, err
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

func checkProfileArgs(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return errors.New("spur gear: nil context")
	}
	if world == nil {
		return errors.New("spur gear: nil sketch world")
	}
	if plane == nil {
		return errors.New("spur gear: nil sketch plane")
	}
	return nil
}

// filletRoots fillets the ordinary root edges matching roots, with a float
// radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges whose start matches a saved root.
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
			w := toWorld(frame, root)
			if math.Abs(w.X-start.X) <= 1e-6 && math.Abs(w.Y-start.Y) <= 1e-6 && math.Abs(w.Z-start.Z) <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("match root edges: got %d, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2*teeth concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the gear's
// lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("spur gear bore parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spur gear bore placement: %w", err)
	}
	low, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spur gear bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(low)
	if err != nil {
		return nil, nil, fmt.Errorf("spur gear bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spur gear bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spur gear bore diameter binding: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "spur gear bore")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore extrudes the bore tool and cuts it from gear. The document must
// already define the parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spur gear bore margin: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spur gear bore length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spur gear bore tool extrusion: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spur gear bore cut: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from a gear
// body that spans the placed cap planes.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spur gear bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("spur gear bore: nil document")
	}
	if body == nil {
		return nil, errors.New("spur gear bore: nil body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("spur gear bore: diameter %g must be positive", p.BoreDiameter)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, errors.New("spur gear bore: invalid placement frame")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spur gear bore parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops with the given setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spur gear cap chamfer: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with setback distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spur gear chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("spur gear chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spur gear chamfer: distance %g must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
