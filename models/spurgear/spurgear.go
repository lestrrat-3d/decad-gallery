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
	d.fillet = 0.9 * d.root * (math.Pi/float64(p.Teeth) - 2*(math.Tan(p.PressureAngle)-p.PressureAngle)) / 2
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base || p.BoreDiameter >= 2*d.root || math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("invalid gear dimensions")
	}
	return nil
}

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

func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }
func defineParameters(t *param.Table, p Params) error {
	set := func(name string, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if err := set("module", t.SetValue("module", units.Millimeters(p.Module))); err != nil {
		return err
	}
	if err := set("teeth", t.SetNumber("teeth", float64(p.Teeth))); err != nil {
		return err
	}
	if err := set("pressure_angle", t.SetValue("pressure_angle", units.Radians(p.PressureAngle))); err != nil {
		return err
	}
	if err := set("thickness", t.SetValue("thickness", units.Millimeters(p.Thickness))); err != nil {
		return err
	}
	if err := set("samples", t.SetNumber("samples", float64(p.Samples))); err != nil {
		return err
	}
	if err := set("bore_diameter", t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter))); err != nil {
		return err
	}
	if err := set("cap_chamfer", t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer))); err != nil {
		return err
	}
	if err := set("backlash", t.SetValue("backlash", units.Millimeters(p.Backlash))); err != nil {
		return err
	}
	if err := set("tool_margin", t.SetExpr("tool_margin", "1", units.Millimeter)); err != nil {
		return err
	}
	if err := set("pitch_radius", t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter)); err != nil {
		return err
	}
	if err := set("base_radius", t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter)); err != nil {
		return err
	}
	if err := set("root_radius", t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter)); err != nil {
		return err
	}
	if err := set("tip_radius", t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter)); err != nil {
		return err
	}
	if err := set("involute_angle", t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One)); err != nil {
		return err
	}
	if err := set("fillet_radius", t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter)); err != nil {
		return err
	}
	return set("bore_tool_length", t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter))
}
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	names := []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}}
	for _, v := range names {
		x, e := doc.Param(v.name)
		if e != nil {
			return d, fmt.Errorf("%s: %w", v.name, e)
		}
		*v.dst = x.Base()
	}
	return d, nil
}
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("build requires context and document")
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

func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("build tooth: %w", err)
	}
	circle, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root disc sketch: %w", err)
	}
	c := circle.CreateCircle(circle.Origin(), r.d.root)
	dim := sketch.NewRadius(c, r.d.root)
	circle.AddConstraint(dim)
	if err = circle.Bind(dim, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	if _, err = circle.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, circle); err != nil {
		return nil, fmt.Errorf("verify root disc: %w", err)
	}
	profiles := circle.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("root disc profile count %d, expected one valid profile", len(profiles))
	}
	disc, err := decad.Extrude(ctx, r.doc, circle, profiles[0], r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join teeth and root disc: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("gear union is not solid")
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
		distance, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, distance)
		if err != nil {
			return nil, fmt.Errorf("chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("finished gear is not solid")
	}
	return body, nil
}
func BuildProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || plane == nil {
		return nil, nil, fmt.Errorf("profile requires context, world and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, pr, _, e := buildGearProfile(ctx, w, plane, p, derive(p))
	return s, pr, e
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("profile requires context, world and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	s, pr, roots, e := buildGearProfileTwisted(ctx, w, plane, p, derive(p), twist)
	out := make([][2]float64, len(roots))
	for i, v := range roots {
		out[i] = [2]float64{v.x, v.y}
	}
	return s, pr, out, e
}

func buildGearProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, plane, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, w, plane, p, d, twist)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create outline sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		a := toothAngle(p, k) + twist
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, a)
		rr, lr := rootContact(r[0], d), rootContact(l[0], d)
		roots = append(roots, rr, lr)
		rp, lp := make([]*sketch.Point, len(r)), make([]*sketch.Point, len(l))
		for i, v := range r {
			rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
		for i, v := range l {
			if i == len(l)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
			}
		}
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(s.Origin(), lastLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rp[0])
		if _, err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], leftRoot)
		lastLeft = leftRoot
	}
	s.CreateArc(s.Origin(), lastLeft, firstRight)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve outline: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("outline profile count %d, expected one valid profile", len(profiles))
	}
	if got, want := len(profiles[0].Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("outline has %d curves, expected %d", got, want)
	}
	return s, profiles[0], roots, nil
}

func buildEmbeddedOutline(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded outline sketch: %w", err)
	}
	type flankRef struct {
		tooth int
		side  int
		curve *sketch.FitSpline
	}
	flanks := make([]flankRef, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
		for i, v := range right {
			rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
		for i, v := range left {
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
			}
		}
		rightSpline, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, e)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		leftSpline, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, e)
		}
		flanks = append(flanks, flankRef{tooth: k, side: 0, curve: rightSpline}, flankRef{tooth: k, side: 1, curve: leftSpline})
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded outline: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded outline: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline has no regions")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded outline regions: %w", err)
	}
	want := 4 * p.Teeth
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline has valid=%t, %d holes and %d curves; expected valid, no holes and %d curves", profile.Valid, len(profile.Holes), len(profile.Outer), want)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, len(contacts))
	for _, edge := range profile.Outer {
		curve, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline contains an uncertified spline trim")
		}
		match := -1
		for i, ref := range flanks {
			if ref.curve == curve {
				match = i
				break
			}
		}
		if match < 0 {
			return nil, nil, nil, fmt.Errorf("embedded outline contains an unknown spline edge")
		}
		ref := flanks[match]
		index := 2*ref.tooth + ref.side
		if filled[index] {
			return nil, nil, nil, fmt.Errorf("tooth %d has duplicate embedded root trim", ref.tooth)
		}
		x, y := curve.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d spline trim misses the root circle", ref.tooth)
		}
		contacts[index] = point{x, y}
		filled[index] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded root contact %d was not certified", i)
		}
	}
	return s, profile, contacts, nil
}

func buildBoreProfile(ctx context.Context, w *sketch.World, p Params, margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(w.Params(), p); err != nil {
		return nil, nil, err
	}
	f, e := placement(p)
	if e != nil {
		return nil, nil, e
	}
	f, e = shifted(f, -margin)
	if e != nil {
		return nil, nil, e
	}
	plane, e := w.CreatePlaneFromFrame(f)
	if e != nil {
		return nil, nil, e
	}
	s, e := w.CreateSketch(plane)
	if e != nil {
		return nil, nil, e
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	dim := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(dim)
	if e = s.Bind(dim, w.Params(), "bore_diameter"); e != nil {
		return nil, nil, e
	}
	if _, e = s.Solve(ctx); e != nil {
		return nil, nil, e
	}
	if e = sketchcheck.VerifyFullyConstrained(ctx, s); e != nil {
		return nil, nil, e
	}
	pr := s.Profiles()
	if len(pr) != 1 || !pr[0].Valid {
		return nil, nil, fmt.Errorf("bore profile count %d, expected one valid profile", len(pr))
	}
	return s, pr[0], nil
}
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin := 1.0
	w := sketch.NewWorld()
	s, pr, e := buildBoreProfile(ctx, w, p, margin)
	if e != nil {
		return nil, e
	}
	length, e := doc.Param("bore_tool_length")
	if e != nil {
		return nil, e
	}
	tool, e := decad.Extrude(ctx, doc, s, pr, decad.Distance{D: length, Dir: decad.Along})
	if e != nil {
		return nil, e
	}
	return decad.Cut(ctx, gear, tool)
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("no ordinary root contacts")
	}
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, root := range roots {
		want := toWorld(frame, root)
		found := false
		for _, edge := range edges {
			for _, vertex := range []*decad.Vertex{edge.Start(), edge.End()} {
				got := vertex.Position().Value
				if got.Equal(want, 1e-6) {
					corners = append(corners, got)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("root contact (%g,%g) has no axial edge", root.x, root.y)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(corner))
	}
	result, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet ordinary roots: %w", err)
	}
	return result, nil
}

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	result, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return result, nil
}

func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("bore requires context, document, body and positive diameter")
	}
	if e := defineParameters(doc.Params(), p); e != nil {
		return nil, e
	}
	return cutBore(ctx, doc, body, p)
}
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	q := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	b, e := decad.Chamfer(ctx, gear, q, setback)
	if e != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", e)
	}
	return b, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer requires context, body and positive finite distance")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
