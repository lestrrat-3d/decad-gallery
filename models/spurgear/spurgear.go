package spurgear

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

type Params struct {
	Module                             float64
	Teeth                              int
	PressureAngle, Thickness           float64
	Samples                            int
	BoreDiameter, CapChamfer, Backlash float64
	Placement                          *r3.Frame
}

func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15, Backlash: 0.05}
}
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
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
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 || p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 || p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("invalid gear parameters")
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("invalid backlash")
	}
	if _, err := placement(p); err != nil {
		return err
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base || p.BoreDiameter >= 2*d.root || !finite(d.fillet) {
		return fmt.Errorf("invalid gear dimensions")
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
	for _, v := range []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		x, err := doc.Param(v.name)
		if err != nil {
			return d, fmt.Errorf("%s: %w", v.name, err)
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
		return nil, fmt.Errorf("disc sketch: %w", err)
	}
	c := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(c, r.d.root)
	s.AddConstraint(radius)
	if err = s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("disc radius: %w", err)
	}
	profile, err := sectionProfile(ctx, s, 1)
	if err != nil {
		return nil, fmt.Errorf("disc profile: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("disc extrude: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join: %w", err)
	}
	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
	} else if r.d.fillet > 0 {
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("fillet: %w", err)
	}
	if r.p.BoreDiameter > 0 {
		body, err = cutThroughBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, fmt.Errorf("bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		var distance units.Value
		distance, err = r.doc.Param("cap_chamfer")
		if err == nil {
			body, err = chamferOuterCaps(ctx, body, distance)
		}
		if err != nil {
			return nil, fmt.Errorf("chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("gear is not solid")
	}
	return body, nil
}

func BuildProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	s, prof, _, err := BuildTwistedProfile(ctx, w, plane, p, 0)
	return s, prof, err
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("nil profile argument")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("invalid twist")
	}
	s, prof, roots, err := buildGearProfileTwisted(ctx, w, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, v := range roots {
		out[i] = [2]float64{v.x, v.y}
	}
	return s, prof, out, nil
}
func buildGearProfile(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, plane, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, w *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("profile sketch: %w", err)
	}
	embedded := d.base < d.root
	var first, previous *sketch.Point
	roots := make([]point, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lp, rp := flankPoints(s, l, r, false)
		var lr, rr *sketch.Point
		if !embedded {
			lroot, rroot := rootContact(l[0], d), rootContact(r[0], d)
			lr = sketchcheck.DimensionArcEnd(s, lroot.x, lroot.y)
			rr = sketchcheck.DimensionPoint(s, rroot.x, rroot.y)
			roots = append(roots, rroot, lroot)
			if k == 0 {
				first = rr
			} else {
				s.CreateArc(s.Origin(), previous, rr)
			}
			s.CreateLine(rr, rp[0])
		}
		if _, err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		if !embedded {
			s.CreateLine(lp[0], lr)
			previous = lr
		}
	}
	if !embedded {
		s.CreateArc(s.Origin(), previous, first)
		prof, e := sectionProfile(ctx, s, 0)
		return s, prof, roots, e
	}
	c := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(c, d.root))
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded profile: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded profile: %w", err)
	}
	profiles := s.Profiles()
	indices := make([]int, len(profiles))
	for i, prof := range profiles {
		if !prof.Valid {
			return nil, nil, nil, fmt.Errorf("invalid embedded region %d", i)
		}
		indices[i] = i
	}
	prof, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union profiles: %w", err)
	}
	if !prof.Valid || len(prof.Holes) != 0 || len(prof.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded boundary curves %d, expected %d and no holes", len(prof.Outer), 4*p.Teeth)
	}
	for _, edge := range prof.Outer {
		if flank, ok := edge.Entity.(*sketch.FitSpline); ok {
			if !edge.Partial || !edge.TExact {
				return nil, nil, nil, fmt.Errorf("uncertified spline trim")
			}
			x, y := flank.Eval(edge.TStart)
			if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
				return nil, nil, nil, fmt.Errorf("spline contact not on root circle")
			}
			roots = append(roots, point{x, y})
		}
	}
	if len(roots) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("root contacts %d, expected %d", len(roots), 2*p.Teeth)
	}
	// Return right then left for each tooth, independent of boundary traversal.
	slices.SortFunc(roots, func(left, right point) int {
		a := math.Mod(math.Mod(math.Atan2(left.y, left.x)-twist, 2*math.Pi)+2*math.Pi, 2*math.Pi)
		b := math.Mod(math.Mod(math.Atan2(right.y, right.x)-twist, 2*math.Pi)+2*math.Pi, 2*math.Pi)
		return cmp.Compare(a, b)
	})
	last := roots[len(roots)-1]
	roots = append([]point{last}, roots[:len(roots)-1]...)
	return s, prof, roots, nil
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
		pos := edge.Start().Position().Value
		for _, root := range roots {
			if pos.Sub(toWorld(frame, root)).Len() < 1e-6 {
				corners = append(corners, pos)
				break
			}
		}
	}
	if len(corners) != len(roots) || len(corners) == 0 {
		return nil, fmt.Errorf("root edges %d, expected %d", len(corners), len(roots))
	}
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, p := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(p))
	}
	return decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
}
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	return decad.Fillet(ctx, body, decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2*teeth), radius)
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
	pl, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, fmt.Errorf("bore plane: %w", err)
	}
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	dim := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(dim)
	if err = s.Bind(dim, w.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore diameter: %w", err)
	}
	prof, err := sectionProfile(ctx, s, 1)
	return s, prof, err
}
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || !(p.BoreDiameter > 0) {
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
		return nil, fmt.Errorf("tool margin: %w", err)
	}
	w := sketch.NewWorld()
	s, prof, err := buildBoreProfile(ctx, w, p, margin.Base())
	if err != nil {
		return nil, fmt.Errorf("bore profile: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore tool length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, prof, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore extrusion: %w", err)
	}
	cut, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return cut, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("invalid chamfer arguments")
	}
	return chamferOuterCaps(ctx, gear, units.Millimeters(distance))
}
func chamferOuterCaps(ctx context.Context, gear *decad.Body, distance units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	result, err := decad.Chamfer(ctx, gear, sel, distance)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return result, nil
}
