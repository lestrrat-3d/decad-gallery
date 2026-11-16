package spurgear

import (
	"context"
	"fmt"
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
	"math"
)

// Params specifies a straight-toothed involute gear. Lengths are in
// millimeters and angles are in radians. Zero values do not select defaults.
type Params struct {
	Module        float64   // Tooth module.
	Teeth         int       // Number of teeth.
	PressureAngle float64   // Pressure angle in radians.
	Thickness     float64   // Axial thickness in millimeters.
	Samples       int       // Fit points per flank.
	BoreDiameter  float64   // Through-bore diameter; zero omits the bore.
	CapChamfer    float64   // Outer cap setback; zero omits the chamfer.
	Backlash      float64   // Circumferential tooth thinning at the pitch circle.
	Placement     *r3.Frame // Reference plane and gear axis; nil selects world XY.
}

// DefaultParams returns the documented default gear parameters.
func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15, Backlash: 0.05}
}

// RejectPlacement reports that model cannot accept a non-nil Placement.
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
func validate(p Params) error {
	if p.Module <= 0 || p.Teeth < 3 || p.Thickness <= 0 || p.Samples < 3 || p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 || p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("invalid gear parameters")
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("invalid backlash")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base || p.BoreDiameter >= 2*d.root || !finite(d.fillet) {
		return fmt.Errorf("invalid gear dimensions")
	}
	return nil
}

// DefineParameters writes the spur gear inputs and derived expressions to t.
// It does not validate p. t must not be nil.
func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }
func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("bore_tool_length: %w", err)
	}
	return nil
}
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	vals := []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}}
	for _, v := range vals {
		x, err := doc.Param(v.name)
		if err != nil {
			return d, fmt.Errorf("%s: %w", v.name, err)
		}
		*v.dst = x.Base()
	}
	return d, nil
}

// Build constructs a single solid spur gear in doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}

// Build constructs a gear using the variant's tooth and finishing operations.
// ctx and doc must not be nil.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("nil context or document")
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

// BuildProfile returns the verified, unfilleted whole-gear outline on plane.
// ctx, w, and plane must not be nil.
func BuildProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	s, pr, _, err := BuildTwistedProfile(ctx, w, plane, p, 0)
	return s, pr, err
}

// BuildTwistedProfile returns the verified whole-gear outline rotated by twist
// in the sketch plane, and its root contacts in right-then-left tooth order.
// ctx, w, and plane must not be nil; twist must be finite.
func BuildTwistedProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("nil context, world or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("invalid twist")
	}
	s, pr, roots, err := buildGearProfileTwisted(ctx, w, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, pr, out, nil
}
func buildGearProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, plane, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear profile sketch: %w", err)
	}
	embedded := d.base < d.root
	roots := make([]point, 2*p.Teeth)
	splines := make(map[*sketch.FitSpline]int)
	var first, prev *sketch.Point
	for k := range p.Teeth {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lp, rp := flankPoints(s, l, r, false)
		var lr, rr *sketch.Point
		if !embedded {
			lroot, rroot := rootContact(l[0], d), rootContact(r[0], d)
			roots[2*k], roots[2*k+1] = rroot, lroot
			lr = sketchcheck.DimensionArcEnd(s, lroot.x, lroot.y)
			rr = sketchcheck.DimensionPoint(s, rroot.x, rroot.y)
			if k == 0 {
				first = rr
			} else {
				s.CreateArc(s.Origin(), prev, rr)
			}
			s.CreateLine(rr, rp[0])
		}
		rs, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right spline: %w", k, e)
		}
		splines[rs] = 2 * k
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		ls, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left spline: %w", k, e)
		}
		splines[ls] = 2*k + 1
		if !embedded {
			s.CreateLine(lp[0], lr)
			prev = lr
		}
	}
	if !embedded {
		s.CreateArc(s.Origin(), prev, first)
		pr, e := finishSketch(ctx, s, 6*p.Teeth)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("gear profile: %w", e)
		}
		return s, pr, roots, nil
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded outline: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, fmt.Errorf("no embedded regions")
	}
	indices := make([]int, len(profiles))
	for i, pr := range profiles {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("invalid embedded region %d", i)
		}
		indices[i] = i
	}
	pr, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union profiles: %w", err)
	}
	if !pr.Valid || len(pr.Holes) != 0 || len(pr.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded profile has %d curves, expected %d, holes %d", len(pr.Outer), 4*p.Teeth, len(pr.Holes))
	}
	seen := make([]bool, len(roots))
	for _, edge := range pr.Outer {
		sp, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		i, ok := splines[sp]
		if !ok || seen[i] {
			return nil, nil, nil, fmt.Errorf("unknown or duplicate flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth %d trim is not certified", i/2)
		}
		x, y := sp.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d trim is not on root circle", i/2)
		}
		roots[i] = point{x, y}
		seen[i] = true
	}
	for i, ok := range seen {
		if !ok {
			return nil, nil, nil, fmt.Errorf("tooth %d missing root contact", i/2)
		}
	}
	return s, pr, roots, nil
}
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err = s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	pr, err := finishSketch(ctx, s, 1)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, pr, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root extrusion: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join: %w", err)
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
		body, err = cutThroughBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, fmt.Errorf("bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		c, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, c)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("gear is not solid")
	}
	return body, nil
}
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		for _, end := range []r3.Vec{edge.Start().Position().Value, edge.End().Position().Value} {
			matched := false
			for _, root := range roots {
				delta := end.Sub(toWorld(frame, root))
				if math.Sqrt(delta.X*delta.X+delta.Y*delta.Y+delta.Z*delta.Z) <= 1e-6 {
					matched = true
					break
				}
			}
			if matched {
				corners = append(corners, end)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, fmt.Errorf("no root edges")
	}
	sel := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
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
		return nil, nil, fmt.Errorf("bore sketch parameters: %w", err)
	}
	f, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore placement: %w", err)
	}
	f, err = shifted(f, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("shift bore placement: %w", err)
	}
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diam := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(diam)
	if err = s.Bind(diam, w.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	pr, err := finishSketch(ctx, s, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	return s, pr, nil
}

// CutThroughBore cuts a centered through-bore from a body spanning the gear's
// thickness. ctx, doc, and body must not be nil; BoreDiameter must be positive.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("invalid bore arguments")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore parameters: %w", err)
	}
	return cutThroughBore(ctx, doc, body, p)
}
func cutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("tool_margin: %w", err)
	}
	w := sketch.NewWorld()
	s, pr, err := buildBoreProfile(ctx, w, p, margin.Base())
	if err != nil {
		return nil, fmt.Errorf("bore profile: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, pr, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore extrusion: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers the outer loops of both caps by distance, leaving
// bore rims sharp. ctx and body must not be nil; distance must be finite and positive.
func ChamferOuterCaps(ctx context.Context, body *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || body == nil || distance <= 0 || !finite(distance) {
		return nil, fmt.Errorf("invalid chamfer arguments")
	}
	return chamferCaps(ctx, body, units.Millimeters(distance))
}
func chamferCaps(ctx context.Context, body *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body)))
	out, err := decad.Chamfer(ctx, body, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return out, nil
}
