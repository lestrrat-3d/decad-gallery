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

// Params describes one spur gear. Lengths are millimeters and angles are radians.
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
	// Placement is the reference point and plane; nil selects the world XY frame.
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
		Backlash:      0.05,
	}
}

// DefineParameters writes the spur parameter table of p into t. It does not validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// RejectPlacement returns an error naming model when p carries a placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// placement returns the gear frame of p.
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

// toWorld maps a frame-local planar coordinate to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits.
func validate(p Params) error {
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if !(p.Module > 0) || !finite(p.Module) {
		return fmt.Errorf("spurgear: module must be positive")
	}
	if !(p.Thickness > 0) || !finite(p.Thickness) {
		return fmt.Errorf("spurgear: thickness must be positive")
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth must be at least 3")
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples must be at least 3")
	}
	if !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2) {
		return fmt.Errorf("spurgear: pressure angle must be in (0, pi/2)")
	}
	if p.BoreDiameter < 0 || !finite(p.BoreDiameter) {
		return fmt.Errorf("spurgear: bore diameter must not be negative")
	}
	if p.CapChamfer < 0 || !finite(p.CapChamfer) {
		return fmt.Errorf("spurgear: cap chamfer must not be negative")
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("spurgear: backlash %v must be finite and within module/2", p.Backlash)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("spurgear: invalid placement frame")
	}
	m, n := p.Module, float64(p.Teeth)
	pitch := m * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (m*n - 2.5*m) / 2
	tip := (m*n + 2*m) / 2
	if root <= 0 {
		return fmt.Errorf("spurgear: root radius %v must be positive", root)
	}
	if base <= 0 {
		return fmt.Errorf("spurgear: base radius %v must be positive", base)
	}
	if tip <= base {
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", tip, base)
	}
	if p.BoreDiameter >= 2*root {
		return fmt.Errorf("spurgear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*root)
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	if !finite(fillet) {
		return fmt.Errorf("spurgear: root fillet radius %v is not finite", fillet)
	}
	return nil
}

// solveVerify solves s and requires full constraint.
func solveVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s: %w", what, err)
	}
	return nil
}

// solveSingleProfile solves and verifies s, then returns its only valid profile.
func solveSingleProfile(ctx context.Context, s *sketch.Sketch, what string) (*sketch.Profile, error) {
	if err := solveVerify(ctx, s, what); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s has %d profiles, want 1", what, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s profile is invalid", what)
	}
	return profiles[0], nil
}

// tooth holds the dimensioned sketch points of one tooth.
type tooth struct {
	rightPts, leftPts   []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rightXY, leftXY     point // root contact coordinates
}

// newTooth creates the dimensioned flank and root points of an ordinary tooth.
// The left root is created before the right root.
func newTooth(s *sketch.Sketch, d dimensions, left, right []point) *tooth {
	t := &tooth{}
	for _, q := range right {
		t.rightPts = append(t.rightPts, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	for i, q := range left {
		if i == len(left)-1 {
			t.leftPts = append(t.leftPts, sketchcheck.DimensionArcEnd(s, q.x, q.y))
		} else {
			t.leftPts = append(t.leftPts, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
	}
	t.leftXY = rootContact(left[0], d)
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.leftXY.x, t.leftXY.y)
	t.rightXY = rootContact(right[0], d)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rightXY.x, t.rightXY.y)
	return t
}

// BuildProfile builds the whole unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, err
	}
	s, prof, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, prof, err
}

// BuildTwistedProfile builds the whole unfilleted outline with every tooth turned by
// an extra twist about the sketch origin. It also returns the 2N root contacts: the
// right root then the left root of each tooth in ascending tooth order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if err := profileGuards(ctx, world, plane, p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist must be finite")
	}
	s, prof, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, prof, out, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) error {
	if ctx == nil {
		return fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return fmt.Errorf("spurgear: nil world")
	}
	if plane == nil {
		return fmt.Errorf("spurgear: nil plane")
	}
	return validate(p)
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws every tooth at toothAngle(p,k)+twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create outline sketch: %w", err)
	}
	center := s.Origin()
	var roots []point
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		t := newTooth(s, d, left, right)
		roots = append(roots, t.rightXY, t.leftXY)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(center, prevLeft, t.rightRoot)
		}
		s.CreateLine(t.rightRoot, t.rightPts[0])
		if _, err := s.CreateFitSpline(t.rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, t.rightPts[len(t.rightPts)-1], t.leftPts[len(t.leftPts)-1])
		if _, err := s.CreateFitSpline(t.leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(t.leftPts[0], t.leftRoot)
		prevLeft = t.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	prof, err := solveSingleProfile(ctx, s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, prof, roots, nil
}

// flankSide names which flank of which tooth a spline is.
type flankSide struct {
	tooth int
	left  bool
}

// buildEmbeddedOutline draws the original flanks and tip arcs of every tooth and one
// root circle, then unions the regions into one outline.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create embedded outline sketch: %w", err)
	}
	center := s.Origin()
	splines := map[*sketch.FitSpline]flankSide{}
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		var rightPts, leftPts []*sketch.Point
		for _, q := range right {
			rightPts = append(rightPts, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
		for i, q := range left {
			if i == len(left)-1 {
				leftPts = append(leftPts, sketchcheck.DimensionArcEnd(s, q.x, q.y))
			} else {
				leftPts = append(leftPts, sketchcheck.DimensionPoint(s, q.x, q.y))
			}
		}
		rs, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		ls, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		splines[rs] = flankSide{tooth: k}
		splines[ls] = flankSide{tooth: k, left: true}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, prof := range s.Profiles() {
		if !prof.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	prof, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded outline regions: %w", err)
	}
	if !prof.Valid {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline profile is invalid")
	}
	if len(prof.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d holes", len(prof.Holes))
	}
	if n := len(prof.Outer); n != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d boundary curves, want %d", n, 4*p.Teeth)
	}
	roots := make([]point, 2*p.Teeth)
	seen := make([]bool, 2*p.Teeth)
	for _, edge := range prof.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded flank trim is not certified (partial %t, exact %t)",
				edge.Partial, edge.TExact)
		}
		side, ok := splines[flank]
		if !ok {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has an unknown flank spline")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded root contact at radius %v, want %v", math.Hypot(x, y), d.root)
		}
		idx := 2 * side.tooth
		if side.left {
			idx++
		}
		if seen[idx] {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d flank trimmed more than once", side.tooth)
		}
		seen[idx] = true
		roots[idx] = point{x, y}
	}
	for _, ok := range seen {
		if !ok {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d root contacts, want %d",
				countTrue(seen), 2*p.Teeth)
		}
	}
	return s, prof, roots, nil
}

func countTrue(v []bool) int {
	n := 0
	for _, b := range v {
		if b {
			n++
		}
	}
	return n
}

// embeddedClosingRadius finds the radius Rc at which the embedded tooth is closed.
func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	beta := flankRotation(d.pitch, d.base, teeth, d.backlash)
	half := func(r float64) float64 {
		m := involutePoint(d.base, r)
		l := rotate(m.x, m.y, beta)
		return math.Atan2(l.y, l.x)
	}
	limit := math.Pi / teeth
	if half(d.base) < limit {
		return d.base, nil
	}
	if half(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: no closing radius exists inside the root circle")
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

// buildEmbeddedTooth draws the natural tooth of an embedded gear, closed by an arc
// at the closing radius.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	var rightPts, leftPts []*sketch.Point
	for _, q := range right {
		rightPts = append(rightPts, sketchcheck.DimensionPoint(s, q.x, q.y))
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			leftPts = append(leftPts, sketchcheck.DimensionArcEnd(s, q.x, q.y))
		} else {
			leftPts = append(leftPts, sketchcheck.DimensionPoint(s, q.x, q.y))
		}
	}
	center := s.Origin()
	last := len(rightPts) - 1
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded right flank: %w", err)
	}
	s.CreateArc(center, rightPts[last], leftPts[last])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded left flank: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])
	prof, err := solveSingleProfile(ctx, s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if n := len(prof.Outer); n != 4 {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth has %d outer curves, want 4", n)
	}
	return s, prof, nil
}

// Build constructs the gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear with the variant's replacements.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if v.LocalBuild {
		return v.buildLocal(ctx, doc, p)
	}
	r, err := v.begin(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return v.assemble(ctx, r)
}

// assemble extrudes the root disc and tooth, patterns and joins them, then
// applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join disc and teeth: %w", err)
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("spurgear: joined gear is not a solid")
	}
	if v.RootFillet != nil {
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	} else if r.d.fillet > 0 {
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
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
			return nil, fmt.Errorf("spurgear: read cap chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("spurgear: finished gear is not a solid")
	}
	return gear, nil
}

// buildRootDisc extrudes the root circle between the cap planes.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: bind root radius: %w", err)
	}
	prof, err := solveSingleProfile(ctx, s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, prof, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}
	return disc, nil
}

// filletRoots rounds the ordinary root corners with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue rounds the axial edges that start at the saved root contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, q := range roots {
		world[i] = toWorld(frame, q)
	}
	var corners []r3.Vec
	for _, e := range edges {
		start := e.Start().Position().Value
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
	sel := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet root edges: %w", err)
	}
	return out, nil
}

// filletEmbedded rounds the 2*teeth concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, sel, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet embedded root edges: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body, which spans
// z=0 through the thickness in the gear frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore diameter must be positive")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: define bore parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore cuts the bore using the document's existing parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read tool margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read bore tool length: %w", err)
	}
	world := sketch.NewWorld()
	s, prof, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, prof, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore circle on a plane margin below the gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: define bore sketch parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
	}
	low, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(low)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bind bore diameter: %w", err)
	}
	prof, err := solveSingleProfile(ctx, s, "bore")
	if err != nil {
		return nil, nil, err
	}
	return s, prof, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance must be positive and finite")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loop of each cap, leaving bore rims sharp.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}
