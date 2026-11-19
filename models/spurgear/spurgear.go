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
	PressureAngle float64
	Thickness     float64
	Teeth         int
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	Placement     *r3.Frame
}

func DefaultParams() Params {
	return Params{Module: 1, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Teeth: 17, Samples: 15, Backlash: 0.05}
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

func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("invalid module")
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("invalid thickness")
	}
	if p.Teeth < 3 {
		return fmt.Errorf("invalid teeth")
	}
	if p.Samples < 3 {
		return fmt.Errorf("invalid samples")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("invalid pressure angle")
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("invalid bore")
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("invalid chamfer")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("invalid backlash")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base || p.BoreDiameter >= 2*d.root || math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("invalid derived geometry")
	}
	return nil
}

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
	for _, item := range []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		v, err := doc.Param(item.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("%s: %w", item.name, err)
		}
		*item.dst = v.Base()
	}
	return d, nil
}

func oneProfile(ctx context.Context, s *sketch.Sketch, count int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("expected one valid profile, got %d", len(profiles))
	}
	if len(profiles[0].Outer) != count {
		return nil, fmt.Errorf("profile has %d edges, expected %d", len(profiles[0].Outer), count)
	}
	return profiles[0], nil
}

func BuildProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	s, profile, _, err := BuildTwistedProfile(ctx, w, plane, p, 0)
	return s, profile, err
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("nil profile argument")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("invalid twist")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, w, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, profile, out, nil
}
func buildGearProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, plane, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	embedded := d.base < d.root
	roots := make([]point, 2*p.Teeth)
	flanks := make(map[*sketch.FitSpline]int)
	var firstRight, prevLeft *sketch.Point
	for k := range p.Teeth {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lp, rp := flankPoints(s, l, r, false)
		if !embedded {
			lr, rr := rootContact(l[0], d), rootContact(r[0], d)
			roots[2*k], roots[2*k+1] = rr, lr
			leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
			rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
			if k == 0 {
				firstRight = rightRoot
			} else {
				s.CreateArc(s.Origin(), prevLeft, rightRoot)
			}
			s.CreateLine(rightRoot, rp[0])
			prevLeft = leftRoot
		}
		right, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		left, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		if embedded {
			flanks[right] = 2 * k
			flanks[left] = 2*k + 1
		} else {
			s.CreateLine(lp[0], prevLeft)
		}
	}
	if !embedded {
		s.CreateArc(s.Origin(), prevLeft, firstRight)
		profile, err := oneProfile(ctx, s, 6*p.Teeth)
		if err != nil {
			return nil, nil, nil, err
		}
		return s, profile, roots, nil
	}
	c := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(c, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, fmt.Errorf("no regions")
	}
	indices := make([]int, len(profiles))
	for i, profile := range profiles {
		if !profile.Valid {
			return nil, nil, nil, fmt.Errorf("invalid region %d", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union profiles: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("invalid embedded profile: %d edges, expected %d", len(profile.Outer), 4*p.Teeth)
	}
	seen := make([]bool, len(roots))
	for _, edge := range profile.Outer {
		f, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		idx, ok := flanks[f]
		if !ok || seen[idx] {
			return nil, nil, nil, fmt.Errorf("unknown or duplicate flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth %d uncertified trim", idx/2)
		}
		x, y := f.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d root contact", idx/2)
		}
		roots[idx] = point{x, y}
		seen[idx] = true
	}
	for i, ok := range seen {
		if !ok {
			return nil, nil, nil, fmt.Errorf("missing tooth %d flank", i/2)
		}
	}
	return s, profile, roots, nil
}

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("nil build argument")
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
		return nil, fmt.Errorf("tooth: %w", err)
	}
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root sketch: %w", err)
	}
	c := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(c, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	profile, err := oneProfile(ctx, s, 1)
	if err != nil {
		return nil, fmt.Errorf("root profile: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root extrusion: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("union: %w", err)
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
		setback, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, setback)
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
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		for _, vertex := range []*decad.Vertex{edge.Start(), edge.End()} {
			q := vertex.Position().Value
			matched := false
			for _, root := range roots {
				if q.Sub(toWorld(frame, root)).Len() <= 1e-6 {
					matched = true
					break
				}
			}
			if matched {
				corners = append(corners, q)
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
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(corner))
	}
	body, err = decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet: %w", err)
	}
	return body, nil
}
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	body, err := decad.Fillet(ctx, body, decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2*teeth), radius)
	if err != nil {
		return nil, fmt.Errorf("embedded fillet: %w", err)
	}
	return body, nil
}

func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("invalid bore arguments")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutThroughBore(ctx, doc, body, p)
}
func cutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("tool_margin: %w", err)
	}
	w := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, w, p, margin.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore extrusion: %w", err)
	}
	body, err = decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return body, nil
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
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, w.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore: %w", err)
	}
	profile, err := oneProfile(ctx, s, 1)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("invalid chamfer arguments")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	body, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return body, nil
}
