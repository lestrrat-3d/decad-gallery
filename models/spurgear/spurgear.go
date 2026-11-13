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

func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		if !p.Placement.IsValid() {
			return r3.Frame{}, fmt.Errorf("invalid placement")
		}
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

func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("invalid module, thickness, teeth, or samples")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("invalid pressure angle")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer must be nonnegative")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("invalid backlash")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("invalid gear radii")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter exceeds root circle")
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("invalid fillet radius")
	}
	return nil
}

func defineParameters(t *param.Table, p Params) error {
	calls := []struct {
		name string
		err  error
	}{
		{"module", t.SetValue("module", units.Millimeters(p.Module))}, {"teeth", t.SetNumber("teeth", float64(p.Teeth))},
		{"pressure_angle", t.SetValue("pressure_angle", units.Radians(p.PressureAngle))}, {"thickness", t.SetValue("thickness", units.Millimeters(p.Thickness))}, {"samples", t.SetNumber("samples", float64(p.Samples))},
		{"bore_diameter", t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter))}, {"cap_chamfer", t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer))}, {"backlash", t.SetValue("backlash", units.Millimeters(p.Backlash))},
		{"tool_margin", t.SetExpr("tool_margin", "1", units.Millimeter)}, {"pitch_radius", t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter)}, {"base_radius", t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter)},
		{"root_radius", t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter)}, {"tip_radius", t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter)},
		{"involute_angle", t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One)}, {"fillet_radius", t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter)},
		{"bore_tool_length", t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter)},
	}
	for _, c := range calls {
		if c.err != nil {
			return fmt.Errorf("parameter %s: %w", c.name, c.err)
		}
	}
	return nil
}
func DefineParameters(t *param.Table, p Params) error {
	if t == nil {
		return fmt.Errorf("nil parameter table")
	}
	return defineParameters(t, p)
}
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, v := range []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		x, e := doc.Param(v.name)
		if e != nil {
			return dimensions{}, fmt.Errorf("parameter %s: %w", v.name, e)
		}
		*v.dst = x.Base()
	}
	return d, nil
}
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if v.LocalBuild {
		return v.buildLocal(ctx, doc, p)
	}
	r, e := v.begin(ctx, doc, p)
	if e != nil {
		return nil, e
	}
	return v.assemble(ctx, r)
}

func profile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, w, pl, p, d, twist)
	}
	s, e := w.CreateSketch(pl)
	if e != nil {
		return nil, nil, nil, fmt.Errorf("create outline sketch: %w", e)
	}
	var roots []point
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		ang := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, ang)
		rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
		for i := range right {
			rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		rr := point{right[0].x * d.root / d.base, right[0].y * d.root / d.base}
		lr := point{left[0].x * d.root / d.base, left[0].y * d.root / d.base}
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		roots = append(roots, rr, lr)
		if k > 0 {
			s.CreateArc(s.Origin(), lastLeft, rightRoot)
		} else {
			firstRight = rightRoot
		}
		s.CreateLine(rightRoot, rp[0])
		if _, e = s.CreateFitSpline(rp...); e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, e)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, e = s.CreateFitSpline(lp...); e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, e)
		}
		s.CreateLine(lp[0], leftRoot)
		lastLeft = leftRoot
	}
	s.CreateArc(s.Origin(), lastLeft, firstRight)
	if _, e = s.Solve(ctx); e != nil {
		return nil, nil, nil, fmt.Errorf("solve outline: %w", e)
	}
	if e = sketchcheck.VerifyFullyConstrained(ctx, s); e != nil {
		return nil, nil, nil, fmt.Errorf("verify outline: %w", e)
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, nil, nil, fmt.Errorf("outline profile count/validity: got %d, want one valid", len(ps))
	}
	if len(ps[0].Outer) != 6*p.Teeth {
		return nil, nil, nil, fmt.Errorf("outline has %d curves, want %d", len(ps[0].Outer), 6*p.Teeth)
	}
	return s, ps[0], roots, nil
}
func buildGearProfileTwisted(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return profile(ctx, w, pl, p, d, twist)
}
func buildGearProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, pl, p, d, 0)
}
func BuildProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if e := validate(p); e != nil {
		return nil, nil, e
	}
	s, pr, _, e := buildGearProfile(ctx, w, pl, p, derive(p))
	return s, pr, e
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if e := validate(p); e != nil {
		return nil, nil, nil, e
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	s, pr, rs, e := buildGearProfileTwisted(ctx, w, pl, p, derive(p), twist)
	out := make([][2]float64, len(rs))
	for i, r := range rs {
		out[i] = [2]float64{r.x, r.y}
	}
	return s, pr, out, e
}

func buildEmbeddedOutline(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, e := w.CreateSketch(pl)
	if e != nil {
		return nil, nil, nil, e
	}
	flanks := map[*sketch.FitSpline]struct{ tooth, side int }{}
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), d.root), d.root))
	for k := 0; k < p.Teeth; k++ {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
		for i := range r {
			rp[i] = sketchcheck.DimensionPoint(s, r[i].x, r[i].y)
			if i == len(l)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, l[i].x, l[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, l[i].x, l[i].y)
			}
		}
		rf, er := s.CreateFitSpline(rp...)
		if er != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, er)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		lf, er := s.CreateFitSpline(lp...)
		if er != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, er)
		}
		flanks[rf] = struct{ tooth, side int }{k, 0}
		flanks[lf] = struct{ tooth, side int }{k, 1}
	}
	if _, e = s.Solve(ctx); e != nil {
		return nil, nil, nil, e
	}
	if e = sketchcheck.VerifyFullyConstrained(ctx, s); e != nil {
		return nil, nil, nil, e
	}
	raw := s.Profiles()
	if len(raw) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline has no regions")
	}
	idx := make([]int, len(raw))
	for i, x := range raw {
		if !x.Valid {
			return nil, nil, nil, fmt.Errorf("embedded region %d invalid", i)
		}
		idx[i] = i
	}
	pr, e := s.UnionProfiles(idx...)
	if e != nil {
		return nil, nil, nil, e
	}
	if !pr.Valid || len(pr.Holes) != 0 || len(pr.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline has %d curves, want %d", len(pr.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range pr.Outer {
		f, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		meta, ok := flanks[f]
		if !ok || !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth flank trim is not certified")
		}
		x, y := f.Eval(edge.TStart)
		slot := 2*meta.tooth + meta.side
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for i, c := range contacts {
		if !filled[i] || math.Abs(math.Hypot(c.x, c.y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded tooth %d root contact invalid", i/2)
		}
	}
	return s, pr, contacts, nil
}

func buildBoreProfile(ctx context.Context, w *sketch.World, p Params, margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if e := defineParameters(w.Params(), p); e != nil {
		return nil, nil, e
	}
	f, e := placement(p)
	if e != nil {
		return nil, nil, e
	}
	f, e = shifted(f, -margin)
	if e != nil {
		return nil, nil, e
	}
	pl, e := w.CreatePlaneFromFrame(f)
	if e != nil {
		return nil, nil, e
	}
	s, e := w.CreateSketch(pl)
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
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, nil, fmt.Errorf("bore profile must be one valid region")
	}
	return s, ps[0], nil
}

func boreCut(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	w := sketch.NewWorld()
	if _, e := doc.Param("tool_margin"); e != nil {
		return nil, fmt.Errorf("tool_margin: %w", e)
	}
	s, pr, e := buildBoreProfile(ctx, w, p, 1)
	if e != nil {
		return nil, fmt.Errorf("bore profile: %w", e)
	}
	length, e := doc.Param("bore_tool_length")
	if e != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", e)
	}
	tool, e := decad.Extrude(ctx, doc, s, pr, decad.Distance{D: length, Dir: decad.Along})
	if e != nil {
		return nil, fmt.Errorf("bore tool extrusion: %w", e)
	}
	out, e := decad.Cut(ctx, gear, tool)
	if e != nil {
		return nil, fmt.Errorf("bore cut: %w", e)
	}
	return out, nil
}
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("context, document, body, and positive bore diameter are required")
	}
	if e := defineParameters(doc.Params(), p); e != nil {
		return nil, e
	}
	return boreCut(ctx, doc, body, p)
}
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, e := decad.Chamfer(ctx, gear, sel, setback)
	if e != nil {
		return nil, fmt.Errorf("cap chamfer: %w", e)
	}
	return out, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("context, body, and positive finite distance are required")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	edges, e := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if e != nil {
		return nil, e
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, root := range roots {
		target := toWorld(frame, root)
		for _, edge := range edges {
			start := edge.Start().Position().Value
			end := edge.End().Position().Value
			if start.Sub(target).Len() < 1e-6 {
				corners = append(corners, start)
				break
			}
			if end.Sub(target).Len() < 1e-6 {
				corners = append(corners, end)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, want %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, fmt.Errorf("no root corners")
	}
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(c))
	}
	return decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
}
func filletRoots(ctx context.Context, b *decad.Body, f r3.Frame, roots []point, r float64) (*decad.Body, error) {
	return filletRootsValue(ctx, b, f, roots, units.Millimeters(r))
}
func filletEmbedded(ctx context.Context, b *decad.Body, f r3.Frame, n int, r units.Value) (*decad.Body, error) {
	return decad.Fillet(ctx, b, decad.Edges(decad.ParallelTo(f.N()), decad.Concave()).Exactly(2*n), r)
}
func filletRadius(_ Params, d dimensions) float64 { return d.fillet }
func rootContact(p point, d dimensions) point {
	return point{p.x * d.root / d.base, p.y * d.root / d.base}
}
func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, e := v.tooth(ctx, r)
	if e != nil {
		return nil, fmt.Errorf("tooth: %w", e)
	}
	s, e := r.world.CreateSketch(r.lower)
	if e != nil {
		return nil, e
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	dim := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(dim)
	if e = s.Bind(dim, r.world.Params(), "root_radius"); e != nil {
		return nil, e
	}
	if _, e = s.Solve(ctx); e != nil {
		return nil, fmt.Errorf("root disc solve: %w", e)
	}
	if e = sketchcheck.VerifyFullyConstrained(ctx, s); e != nil {
		return nil, fmt.Errorf("root disc verify: %w", e)
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, fmt.Errorf("root disc profile invalid")
	}
	disc, e := decad.Extrude(ctx, r.doc, s, ps[0], r.extent)
	if e != nil {
		return nil, fmt.Errorf("root disc extrusion: %w", e)
	}
	copies, e := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if e != nil {
		return nil, fmt.Errorf("tooth pattern: %w", e)
	}
	body, e := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if e != nil {
		return nil, fmt.Errorf("gear union: %w", e)
	}
	if v.RootFillet != nil {
		body, e = v.RootFillet(ctx, r.doc, body, r.frame)
	} else if r.d.fillet > 0 {
		body, e = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
	}
	if e != nil {
		return nil, fmt.Errorf("root fillet: %w", e)
	}
	if r.p.BoreDiameter > 0 {
		body, e = boreCut(ctx, r.doc, body, r.p)
		if e != nil {
			return nil, e
		}
	}
	if r.p.CapChamfer > 0 {
		c, er := r.doc.Param("cap_chamfer")
		if er != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", er)
		}
		body, e = chamferCaps(ctx, body, c)
		if e != nil {
			return nil, e
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("gear is not solid")
	}
	return body, nil
}
