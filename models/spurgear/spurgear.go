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
	Module                             float64
	Teeth                              int
	PressureAngle, Thickness           float64
	Samples                            int
	BoreDiameter, CapChamfer, Backlash float64
	Placement                          *r3.Frame
}

func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15, Backlash: .05}
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
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}
func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("invalid gear dimensions")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("invalid pressure angle")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("invalid bore or chamfer")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("invalid backlash")
	}
	if _, err := placement(p); err != nil {
		return err
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base || p.BoreDiameter >= 2*d.root || math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("invalid derived dimensions")
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
	for _, entry := range []struct {
		name   string
		target *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		v, err := doc.Param(entry.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("%s: %w", entry.name, err)
		}
		*entry.target = v.Base()
	}
	return d, nil
}

func solvedProfile(ctx context.Context, s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid || (curves > 0 && len(ps[0].Outer) != curves) {
		return nil, fmt.Errorf("invalid profile")
	}
	return ps[0], nil
}

func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	s, profile, _, err := BuildTwistedProfile(ctx, world, plane, p, 0)
	return s, profile, err
}
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("context, world and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("invalid twist")
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	coords := make([][2]float64, len(roots))
	for i, r := range roots {
		coords[i] = [2]float64{r.x, r.y}
	}
	return s, profile, coords, nil
}
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, err
	}
	embedded := d.base < d.root
	roots := make([]point, 0, 2*p.Teeth)
	flanks := make([]*sketch.FitSpline, 0, 2*p.Teeth)
	var firstRight, previousLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lp, rp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
		for i := range p.Samples {
			rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
			if i == p.Samples-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		var lr, rr *sketch.Point
		if !embedded {
			l, r := rootContact(left[0], d), rootContact(right[0], d)
			lr = sketchcheck.DimensionArcEnd(s, l.x, l.y)
			rr = sketchcheck.DimensionPoint(s, r.x, r.y)
			roots = append(roots, r, l)
			if k == 0 {
				firstRight = rr
			} else {
				s.CreateArc(s.Origin(), previousLeft, rr)
			}
			s.CreateLine(rr, rp[0])
		}
		rf, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, e
		}
		s.CreateArc(s.Origin(), rp[p.Samples-1], lp[p.Samples-1])
		lf, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, e
		}
		flanks = append(flanks, rf, lf)
		if !embedded {
			s.CreateLine(lp[0], lr)
			previousLeft = lr
		}
	}
	if !embedded {
		s.CreateArc(s.Origin(), previousLeft, firstRight)
		profile, e := solvedProfile(ctx, s, 0)
		return s, profile, roots, e
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	ps := s.Profiles()
	indices := make([]int, len(ps))
	for i, profile := range ps {
		if !profile.Valid {
			return nil, nil, nil, fmt.Errorf("invalid embedded region")
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, err
	}
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("invalid embedded outline")
	}
	contacts := make(map[*sketch.FitSpline]point, 2*p.Teeth)
	for _, edge := range profile.Outer {
		if f, ok := edge.Entity.(*sketch.FitSpline); ok {
			if !edge.Partial || !edge.TExact {
				return nil, nil, nil, fmt.Errorf("uncertified root trim")
			}
			x, y := f.Eval(edge.TStart)
			if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
				return nil, nil, nil, fmt.Errorf("invalid root contact")
			}
			contacts[f] = point{x, y}
		}
	}
	for _, f := range flanks {
		contact, ok := contacts[f]
		if !ok {
			return nil, nil, nil, fmt.Errorf("missing root contact")
		}
		roots = append(roots, contact)
	}
	return s, profile, roots, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	halfAngle := func(radius float64) float64 {
		q := involute(d.base, radius)
		pitch := involute(d.base, d.pitch)
		beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(-pitch.y, pitch.x)
		q = rotate(q.x, -q.y, beta)
		return math.Atan2(q.y, q.x)
	}
	limit := math.Pi / float64(p.Teeth)
	if halfAngle(d.root) >= limit {
		return nil, nil, fmt.Errorf("flanks overlap at root circle")
	}
	start := d.base
	if halfAngle(d.base) >= limit {
		lo, hi := d.base, d.root
		for range 200 {
			mid := (lo + hi) / 2
			if halfAngle(mid) >= limit {
				lo = mid
			} else {
				hi = mid
			}
		}
		start = (hi + d.root) / 2
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, start, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	lp, rp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
	for i := range p.Samples {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == 0 || i == p.Samples-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[p.Samples-1], lp[p.Samples-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	profile, err := solvedProfile(ctx, s, 4)
	return s, profile, err
}

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document are required")
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
		return nil, err
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, err
	}
	profile, err := solvedProfile(ctx, s, 0)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join teeth: %w", err)
	}
	if v.RootFillet != nil {
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
	} else if r.d.fillet > 0 {
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("fillet roots: %w", err)
	}
	if r.p.BoreDiameter > 0 {
		gear, err = cutThroughBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, e
		}
		gear, err = chamferOuterCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("gear is not solid")
	}
	return gear, nil
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, err
	}
	var corners []r3.Vec
	for _, edge := range edges {
		v := edge.Start().Position().Value
		for _, root := range roots {
			if v.Sub(toWorld(frame, root)).Len() <= 1e-6 {
				corners = append(corners, v)
				break
			}
		}
	}
	if len(corners) != len(roots) || len(corners) == 0 {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	query := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		query = query.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	return decad.Fillet(ctx, body, query.Exactly(len(corners)), radius)
}
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	return decad.Fillet(ctx, body, decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2*teeth), radius)
}

func buildBoreProfile(ctx context.Context, world *sketch.World, p Params, margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
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
	plane, err := world.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, err
	}
	profile, err := solvedProfile(ctx, s, 0)
	return s, profile, err
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
		return nil, err
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, fmt.Errorf("bore profile: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool: %w", err)
	}
	result, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return result, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("invalid chamfer arguments")
	}
	return chamferOuterCaps(ctx, gear, units.Millimeters(distance))
}
func chamferOuterCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	result, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return result, nil
}
