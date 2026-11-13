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

type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	Placement     *r3.Frame
}

func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15, Backlash: 0.05}
}
func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("invalid module, thickness, teeth, or samples")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("invalid pressure angle")
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("invalid bore diameter")
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("invalid cap chamfer")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("invalid backlash")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := dimensions{pitch: p.Module * float64(p.Teeth) / 2, base: p.Module * float64(p.Teeth) / 2 * math.Cos(p.PressureAngle), root: (p.Module*float64(p.Teeth) - 2.5*p.Module) / 2, tip: (p.Module*float64(p.Teeth) + 2*p.Module) / 2}
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("invalid derived radii")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter exceeds root circle")
	}
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	rf := 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*inv) / 2
	if math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("invalid fillet radius")
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
func toWorld(f r3.Frame, p point) r3.Vec {
	return f.Origin().Add(f.U().Scale(p.x)).Add(f.V().Scale(p.y))
}
func rootContact(p point, d dimensions) point {
	return point{p.x * d.root / d.base, p.y * d.root / d.base}
}
func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if doc == nil {
		return nil, fmt.Errorf("document is required")
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

func BuildProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	d := derive(p)
	s, prof, _, err := buildGearProfile(ctx, w, pl, p, d)
	return s, prof, err
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	d := derive(p)
	s, prof, roots, err := buildGearProfileTwisted(ctx, w, pl, p, d, twist)
	out := make([][2]float64, len(roots))
	for i, r := range roots {
		out[i] = [2]float64{r.x, r.y}
	}
	return s, prof, out, err
}
func buildGearProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, pl, p, d, 0)
}

func buildGearProfileTwisted(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, w, pl, p, d, twist)
	}
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear outline: %w", err)
	}
	var roots []point
	var first, previous *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rr, lr := rootContact(r[0], d), rootContact(l[0], d)
		roots = append(roots, rr, lr)
		rp, lp := make([]*sketch.Point, len(r)), make([]*sketch.Point, len(l))
		for i, q := range r {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range l {
			if i == len(l)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		if k == 0 {
			first = rightRoot
		} else {
			s.CreateArc(s.Origin(), previous, rightRoot)
		}
		s.CreateLine(rightRoot, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right spline: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left spline: %w", k, err)
		}
		s.CreateLine(lp[0], leftRoot)
		previous = leftRoot
	}
	s.CreateArc(s.Origin(), previous, first)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve gear outline: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify gear outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("gear outline has %d profiles, expected 1", len(profiles))
	}
	if n := len(profiles[0].Outer); n != 6*p.Teeth {
		return nil, nil, nil, fmt.Errorf("gear outline has %d curves, expected %d", n, 6*p.Teeth)
	}
	return s, profiles[0], roots, nil
}

func buildEmbeddedOutline(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded outline: %w", err)
	}
	rightFlanks := make([]*sketch.FitSpline, p.Teeth)
	leftFlanks := make([]*sketch.FitSpline, p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rp, lp := make([]*sketch.Point, len(r)), make([]*sketch.Point, len(l))
		for i, q := range r {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range l {
			if i == len(l)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		rightFlank, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, err
		}
		rightFlanks[k] = rightFlank
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		leftFlank, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, err
		}
		leftFlanks[k] = leftFlank
	}
	c := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(c, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded outline: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline has no regions")
	}
	indices := make([]int, len(ps))
	for i, q := range ps {
		if !q.Valid {
			return nil, nil, nil, fmt.Errorf("embedded region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline invalid topology: %d curves, expected %d", len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 2*p.Teeth)
	contactSeen := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		tooth, side := -1, -1
		for k := 0; k < p.Teeth; k++ {
			if flank == rightFlanks[k] {
				tooth, side = k, 0
				break
			}
			if flank == leftFlanks[k] {
				tooth, side = k, 1
				break
			}
		}
		if tooth < 0 {
			return nil, nil, nil, fmt.Errorf("embedded outline contains an unknown flank edge")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth %d flank trim is not certified", tooth)
		}
		slot := 2*tooth + side
		if contactSeen[slot] {
			return nil, nil, nil, fmt.Errorf("tooth %d has a duplicate flank trim", tooth)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d flank trim misses root circle", tooth)
		}
		contacts[slot] = point{x, y}
		contactSeen[slot] = true
	}
	for slot, seen := range contactSeen {
		if !seen {
			return nil, nil, nil, fmt.Errorf("embedded outline is missing tooth %d flank %d contact", slot/2, slot%2)
		}
	}
	return s, profile, contacts, nil
}

func buildEmbeddedTooth(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	angleAt := func(radius float64) float64 {
		l, _ := sampleFlanks(d.pitch, d.base, radius, radius, float64(p.Teeth), d.backlash, 2, 0)
		return math.Abs(math.Atan2(l[0].y, l[0].x))
	}
	if angleAt(d.root) >= math.Pi/float64(p.Teeth) {
		return nil, nil, fmt.Errorf("no closing radius exists inside root circle")
	}
	rc := d.base
	if angleAt(d.base) >= math.Pi/float64(p.Teeth) {
		lo, hi := d.base, d.root
		for i := 0; i < 200; i++ {
			mid := (lo + hi) / 2
			if angleAt(mid) >= math.Pi/float64(p.Teeth) {
				lo = mid
			} else {
				hi = mid
			}
		}
		rc = (hi + d.root) / 2
	}
	l, r := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	rp, lp := make([]*sketch.Point, len(r)), make([]*sketch.Point, len(l))
	for i, q := range r {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range l {
		if i == 0 || i == len(l)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve embedded tooth: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid || len(ps[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth topology invalid")
	}
	return s, ps[0], nil
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, e := range edges {
		a, b := e.Start().Position().Value, e.End().Position().Value
		for _, root := range roots {
			want := toWorld(frame, root)
			if a.Sub(want).Len() < 1e-6 {
				corners = append(corners, a)
				break
			}
			if b.Sub(want).Len() < 1e-6 {
				corners = append(corners, b)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d axial root edges, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return body, nil
	}
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return out, nil
}
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	out, err := decad.Fillet(ctx, body, decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2*teeth), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return out, nil
}

func buildBoreProfile(ctx context.Context, w *sketch.World, p Params, margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(w.Params(), p); err != nil {
		return nil, nil, err
	}
	f, err := placement(p)
	if err != nil {
		return nil, nil, err
	}
	f, err = shifted(f, -margin)
	if err != nil {
		return nil, nil, err
	}
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, err
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	dim := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(dim)
	if err := s.Bind(dim, w.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, nil, fmt.Errorf("bore profile has %d valid regions, expected 1", len(ps))
	}
	return s, ps[0], nil
}

func cutBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	w := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, w, p, 1)
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return out, nil
}
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, fmt.Errorf("context, document, and body are required")
	}
	if p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("bore diameter must be positive")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return out, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("context and gear are required")
	}
	if distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer distance must be positive and finite")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("build tooth: %w", err)
	}
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	dim := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(dim)
	if err := s.Bind(dim, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify root disc: %w", err)
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, fmt.Errorf("root disc has %d profiles, expected 1", len(ps))
	}
	disc, err := decad.Extrude(ctx, r.doc, s, ps[0], r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join gear: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("joined gear is not solid")
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
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		d, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, d)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("finished gear is not solid")
	}
	return body, nil
}
