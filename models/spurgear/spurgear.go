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
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("module, thickness, teeth, or samples out of range")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle out of range")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer must be nonnegative")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash out of range")
	}
	f, err := placement(p)
	if err != nil {
		return fmt.Errorf("placement: %w", err)
	}
	if !f.IsValid() {
		return fmt.Errorf("placement is invalid")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("derived radii out of range")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter must be less than root diameter")
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("fillet radius is nonfinite")
	}
	return nil
}
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

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

func DefineParameters(t *param.Table, p Params) error {
	if t == nil {
		return fmt.Errorf("parameter table is required")
	}
	return defineParameters(t, p)
}
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, v := range []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		x, err := doc.Param(v.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("%s: %w", v.name, err)
		}
		*v.dst = x.Base()
	}
	return d, nil
}
func decadNewForDerive(p Params) (*decad.Document, error) {
	d := decad.New()
	if err := defineParameters(d.Params(), p); err != nil {
		return nil, err
	}
	return d, nil
}
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear outline: %w", err)
	}
	contacts := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		a := 2*math.Pi*float64(k)/float64(p.Teeth) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, a)
		rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
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
		rootL := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rootR := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		contacts = append(contacts, rr, lr)
		if k > 0 {
			s.CreateArc(s.Origin(), lastLeft, rootR)
		} else {
			firstRight = rootR
		}
		s.CreateLine(rootR, rp[0])
		if _, err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right spline: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left spline: %w", k, err)
		}
		s.CreateLine(lp[0], rootL)
		lastLeft = rootL
	}
	s.CreateArc(s.Origin(), lastLeft, firstRight)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve gear outline: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify gear outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("gear outline profile count %d or invalid", len(profiles))
	}
	want := 6 * p.Teeth
	if len(profiles[0].Outer) != want {
		return nil, nil, nil, fmt.Errorf("gear outline has %d curves, want %d", len(profiles[0].Outer), want)
	}
	return s, profiles[0], contacts, nil
}

func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded outline: %w", err)
	}
	rightFlanks := make([]*sketch.FitSpline, p.Teeth)
	leftFlanks := make([]*sketch.FitSpline, p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		a := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, a)
		rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
		for i := range rp {
			rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
			if i == len(lp)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		rightFlanks[k], err = s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right spline: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[p.Samples-1], lp[p.Samples-1])
		leftFlanks[k], err = s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left spline: %w", k, err)
		}
	}
	c := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(c, d.root))
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded outline: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded outline: %w", err)
	}
	regions := s.Profiles()
	if len(regions) < 2 {
		return nil, nil, nil, fmt.Errorf("embedded outline found %d regions, want at least 2", len(regions))
	}
	indices := make([]int, len(regions))
	for i, r := range regions {
		if !r.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded outline regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline union is invalid or has holes")
	}
	want := 4 * p.Teeth
	if len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline has %d curves, want %d", len(profile.Outer), want)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, len(contacts))
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot := -1
		for k := range rightFlanks {
			if flank == rightFlanks[k] {
				slot = 2 * k
				break
			}
			if flank == leftFlanks[k] {
				slot = 2*k + 1
				break
			}
		}
		if slot < 0 || filled[slot] {
			return nil, nil, nil, fmt.Errorf("embedded outline has unknown or duplicate spline trim")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth %d embedded outline spline trim is not certified", slot/2)
		}
		x, y := flank.Eval(edge.TStart)
		contacts[slot] = point{x, y}
		filled[slot] = true
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline contact %d misses root circle", slot)
		}
	}
	for i, v := range filled {
		if !v {
			return nil, nil, nil, fmt.Errorf("embedded outline contact %d is missing", i)
		}
	}
	return s, profile, contacts, nil
}
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, pr, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, pr, err
}
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	s, pr, pts, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	out := make([][2]float64, len(pts))
	for i, v := range pts {
		out[i] = [2]float64{v.x, v.y}
	}
	return s, pr, out, err
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
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return body, nil
	}
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, err
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, root := range roots {
		target := toWorld(frame, root)
		for _, edge := range edges {
			for _, v := range []r3.Vec{edge.Start().Position().Value, edge.End().Position().Value} {
				if v.Sub(target).Len() < 1e-6 {
					corners = append(corners, v)
					goto matched
				}
			}
		}
	matched:
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, want %d", len(corners), len(roots))
	}
	q := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		q.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(c))
	}
	return decad.Fillet(ctx, body, q.Exactly(len(corners)), radius)
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
	pl, err := world.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(pl)
	if err != nil {
		return nil, nil, err
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	dim := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(dim)
	if err = s.Bind(dim, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, err
	}
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, nil, fmt.Errorf("invalid bore profile")
	}
	return s, ps[0], nil
}
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	w := sketch.NewWorld()
	sketch, profile, err := buildBoreProfile(ctx, w, p, 1)
	if err != nil {
		return nil, fmt.Errorf("bore profile: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, sketch, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool: %w", err)
	}
	body, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore cut: %w", err)
	}
	return body, nil
}
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || p.BoreDiameter <= 0 || math.IsNaN(p.BoreDiameter) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("context, document, body, and positive bore diameter are required")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	q := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	return decad.Chamfer(ctx, gear, q, setback)
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("context, gear, and positive finite distance are required")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("tooth: %w", err)
	}
	discSketch, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root disc sketch: %w", err)
	}
	c := discSketch.CreateCircle(discSketch.Origin(), r.d.root)
	radius := sketch.NewRadius(c, r.d.root)
	discSketch.AddConstraint(radius)
	if err = discSketch.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, err
	}
	if _, err = discSketch.Solve(ctx); err != nil {
		return nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, discSketch); err != nil {
		return nil, err
	}
	ps := discSketch.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, fmt.Errorf("invalid root disc profile")
	}
	disc, err := decad.Extrude(ctx, r.doc, discSketch, ps[0], r.extent)
	if err != nil {
		return nil, fmt.Errorf("root disc extrusion: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("tooth pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("gear union: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("gear union did not produce a solid")
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
		value, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, value)
		if err != nil {
			return nil, fmt.Errorf("cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("finished gear is not solid")
	}
	return body, nil
}
