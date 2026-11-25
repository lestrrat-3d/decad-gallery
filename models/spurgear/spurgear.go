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
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one cylindrical spur gear. Lengths are in millimeters and
// angles in radians. Zero fields are not defaults; use DefaultParams.
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
	// CapChamfer is the equal-distance outer cap setback; zero omits it.
	CapChamfer float64
	// Backlash is the circumferential tooth thinning at the pitch circle.
	Backlash float64
	// Placement is the reference point and plane; nil selects the world XY
	// frame at the origin.
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
		BoreDiameter:  0,
		CapChamfer:    0,
		Backlash:      0.05,
		Placement:     nil,
	}
}

// RejectPlacement returns an error naming model when p carries a placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// validate applies the input checks and limits of the gear contract.
func validate(p Params) error {
	switch {
	case !finite(p.Module) || p.Module <= 0:
		return fmt.Errorf("module %g must be positive", p.Module)
	case !finite(p.Thickness) || p.Thickness <= 0:
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case !finite(p.PressureAngle) || p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("pressure angle %g must lie in (0, π/2)", p.PressureAngle)
	case !finite(p.BoreDiameter) || p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case !finite(p.CapChamfer) || p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	case !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("backlash %g must be finite and at most module/2 = %g in magnitude",
			p.Backlash, p.Module/2)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("placement is not a valid frame")
	}
	m, n, a := p.Module, float64(p.Teeth), p.PressureAngle
	rp := m * n / 2
	rb := rp * math.Cos(a)
	rr := (m*n - 2.5*m) / 2
	rt := (m*n + 2*m) / 2
	switch {
	case rr <= 0:
		return fmt.Errorf("root radius %g must be positive", rr)
	case rb <= 0:
		return fmt.Errorf("base radius %g must be positive", rb)
	case rt <= rb:
		return fmt.Errorf("tip radius %g must exceed base radius %g", rt, rb)
	case p.BoreDiameter >= 2*rr:
		return fmt.Errorf("bore diameter %g must be less than the root diameter %g", p.BoreDiameter, 2*rr)
	}
	rf := 0.9 * rr * (math.Pi/n - 2*(math.Tan(a)-a)) / 2
	if !finite(rf) {
		return fmt.Errorf("root fillet radius %g is not finite", rf)
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

// toWorld maps a frame-local planar point to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func checkProfileArgs(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	switch {
	case ctx == nil:
		return errors.New("context is required")
	case world == nil:
		return errors.New("sketch world is required")
	case plane == nil:
		return errors.New("sketch plane is required")
	}
	return nil
}

// DefineParameters writes the gear parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// flankPoints creates the dimensioned flank points of one tooth. Every
// right point gets both coordinates; every left point does too except its
// tip, which gets one.
func flankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
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
	return rightPts, leftPts
}

// rootPoints creates the left then right root points of one ordinary tooth.
func rootPoints(s *sketch.Sketch, leftBase, rightBase point, d dimensions) (*sketch.Point, *sketch.Point) {
	lr := rootContact(leftBase, d)
	leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rr := rootContact(rightBase, d)
	rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
	return leftRoot, rightRoot
}

// Build builds one spur gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds one spur gear into doc using the variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if doc == nil {
		return nil, errors.New("document is required")
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

// assemble builds the gear of a prepared run: tooth zero, the root disc, the
// pattern and join, the root fillet, the bore and the cap chamfer.
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
			return nil, fmt.Errorf("read parameter cap_chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, fmt.Errorf("cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("gear is not a solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root-disc circle on the lower plane.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return disc, nil
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
	d := derive(p)
	if d.pitch <= 0 {
		return nil, nil, errors.New("profile: cannot derive dimensions")
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole-gear outline with every tooth turned by
// the extra angle twist about the sketch origin. It also returns the 2N root
// contacts, right then left of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := checkProfileArgs(ctx, world, plane); err != nil {
		return nil, nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, errors.New("profile: twist must be finite")
	}
	d := derive(p)
	if d.pitch <= 0 {
		return nil, nil, nil, errors.New("profile: cannot derive dimensions")
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

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with every tooth
// turned by the extra angle twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	return buildOrdinaryOutline(ctx, world, plane, p, d, twist)
}

func buildOrdinaryOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPts, leftPts := flankPoints(s, left, right)
		leftRoot, rightRoot := rootPoints(s, left[0], right[0], d)
		roots = append(roots, rootContact(right[0], d), rootContact(left[0], d))

		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: left flank spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	profile, err := solveSingleProfile(ctx, s, "outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("outline: boundary has %d curves, expected %d", got, want)
	}
	return s, profile, roots, nil
}

func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	flanks := make([]*sketch.FitSpline, 0, 2*p.Teeth)
	slots := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rightPts, leftPts := flankPoints(s, left, right)
		rightSpline, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		leftSpline, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: left flank spline: %w", k, err)
		}
		slots[rightSpline] = 2 * k
		slots[leftSpline] = 2*k + 1
		flanks = append(flanks, rightSpline, leftSpline)
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: verify: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("embedded outline: sketch has no region")
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
		return nil, nil, nil, errors.New("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: boundary has %d curves, expected %d", got, want)
	}

	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, known := slots[spline]
		if !known {
			return nil, nil, nil, errors.New("embedded outline: boundary spline matches no tooth")
		}
		tooth := slot / 2
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d: root trim is not certified", tooth)
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d: duplicate flank edge", tooth)
		}
		x, y := flanks[slot].Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d: root contact radius %g, expected %g",
				tooth, math.Hypot(x, y), d.root)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for slot, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d: missing root contact", slot/2)
		}
	}
	return s, profile, contacts, nil
}

// closingRadius returns the radius Rc of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth float64) (float64, error) {
	limit := math.Pi / teeth
	h := func(r float64) float64 { return halfAngle(d.pitch, d.base, teeth, d.backlash, r) }
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: neighbouring teeth overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if h(mid) > limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws one embedded tooth closed by an arc at radius Rc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	teeth := float64(p.Teeth)
	rc, err := closingRadius(d, teeth)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, teeth, d.backlash, p.Samples, 0)
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

	profile, err := solveSingleProfile(ctx, s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: boundary has %d curves, expected 4", got)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges that start at the saved roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges whose lower-cap end lies at one of
// the saved root contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet: select axial edges: %w", err)
	}
	want := make([]r3.Vec, len(roots))
	for i, q := range roots {
		want[i] = toWorld(frame, q)
	}
	near := func(pos r3.Vec) bool {
		for _, w := range want {
			if pos.Sub(w).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, e := range edges {
		if pos := e.Start().Position().Value; near(pos) {
			corners = append(corners, pos)
		} else if pos := e.End().Position().Value; near(pos) {
			corners = append(corners, pos)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
// body must span z=0 through the gear thickness along the placement normal.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("bore: context is required")
	case doc == nil:
		return nil, errors.New("bore: document is required")
	case body == nil:
		return nil, errors.New("bore: body is required")
	case !finite(p.BoreDiameter) || p.BoreDiameter <= 0:
		return nil, fmt.Errorf("bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore cuts the bore using the document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read parameter bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool: extrude: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore tool: cut: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the gear's
// lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: shift placement: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
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
	profile, err := solveSingleProfile(ctx, s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("chamfer: context is required")
	case gear == nil:
		return nil, errors.New("chamfer: body is required")
	case !finite(distance) || distance <= 0:
		return nil, fmt.Errorf("chamfer: distance %g must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps with the given setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}
