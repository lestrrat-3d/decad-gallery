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

// Params contains the dimensional inputs for a spur gear.
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

func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// DefineParameters records p in a caller's parameter table.
func DefineParameters(t *param.Table, p Params) error {
	if t == nil {
		return fmt.Errorf("parameter table is required")
	}
	return defineParameters(t, p)
}

func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, item := range []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		v, err := doc.Param(item.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read %s: %w", item.name, err)
		}
		*item.dst = v.Base()
	}
	return d, nil
}

func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("module, thickness, teeth, or samples are outside their limits")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle is outside its limits")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer must not be negative")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash is outside its limits")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("derived radii are invalid")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter must be less than root diameter")
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("fillet radius is not finite")
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
func toWorld(f r3.Frame, p point) r3.Vec {
	return f.Origin().Add(f.U().Scale(p.x)).Add(f.V().Scale(p.y))
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileAtTwist(ctx, world, plane, p, d, 0)
}

func buildGearProfileAtTwist(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lr := rootContact(left[0], d)
		rr := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		roots = append(roots, rr, lr)
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, lastLeft, rightRoot)
		}
		rp := make([]*sketch.Point, len(right))
		lp := make([]*sketch.Point, len(left))
		for i, q := range right {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range left {
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		s.CreateLine(rightRoot, rp[0])
		if _, err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right spline: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		if _, err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left spline: %w", k, err)
		}
		s.CreateLine(lp[0], leftRoot)
		lastLeft = leftRoot
	}
	s.CreateArc(center, lastLeft, firstRight)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve outline: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("outline has %d valid profiles, expected 1", len(profiles))
	}
	if got, want := len(profiles[0].Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("outline has %d boundary curves, expected %d", got, want)
	}
	return s, profiles[0], roots, nil
}

func buildGearProfileTwisted(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	return buildGearProfileAtTwist(ctx, w, pl, p, d, twist)
}

func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded outline sketch: %w", err)
	}
	type flankRef struct {
		tooth int
		side  int
		curve *sketch.FitSpline
	}
	flanks := make(map[sketch.Entity]flankRef, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rp, lp := make([]*sketch.Point, len(right)), make([]*sketch.Point, len(left))
		for i, q := range right {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range left {
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		rf, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, e)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		lf, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, e)
		}
		flanks[rf] = flankRef{k, 0, rf}
		flanks[lf] = flankRef{k, 1, lf}
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
		return nil, nil, nil, fmt.Errorf("union embedded regions: %w", err)
	}
	want := 4 * p.Teeth
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline is invalid or has holes")
	}
	if got := len(profile.Outer); got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline has %d boundary curves, expected %d", got, want)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		ref, ok := flanks[flank]
		if !ok || !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth flank trim is not certified")
		}
		x, y := flank.Eval(edge.TStart)
		slot := 2*ref.tooth + ref.side
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("tooth %d has duplicate root trim", ref.tooth)
		}
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d root contact misses root circle", ref.tooth)
		}
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, nil, nil, fmt.Errorf("missing embedded root contact %d", i)
		}
	}
	return s, profile, contacts, nil
}
func BuildProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, pr, _, err := buildGearProfile(ctx, w, pl, p, derive(p))
	return s, pr, err
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, nil, fmt.Errorf("context, world, and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	s, pr, rr, err := buildGearProfileTwisted(ctx, w, pl, p, derive(p), twist)
	out := make([][2]float64, len(rr))
	for i, q := range rr {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, pr, out, err
}

// Build constructs and finishes a spur gear in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document are required")
	}
	return (Variant{}).Build(ctx, doc, p)
}

// Build dispatches to the local or shared assembly pipeline.
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
		return nil, fmt.Errorf("build tooth: %w", err)
	}
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err = s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	if _, err = s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify root disc: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("root disc has %d valid profiles, expected 1", len(profiles))
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profiles[0], r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern tooth: %w", err)
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
		body, err = cutThroughBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, fmt.Errorf("through bore: %w", err)
		}
	}
	if r.p.CapChamfer > 0 {
		setback, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("parameter cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, fmt.Errorf("cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("finished gear is not solid")
	}
	return body, nil
}

func buildBoreProfile(ctx context.Context, world *sketch.World, p Params, margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
	}
	f, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("placement: %w", err)
	}
	f, err = shifted(f, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("shift bore plane: %w", err)
	}
	pl, err := world.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore plane: %w", err)
	}
	s, err := world.CreateSketch(pl)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err = s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve bore sketch: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify bore sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("bore profile has %d valid regions, expected 1", len(profiles))
	}
	return s, profiles[0], nil
}

func cutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("parameter tool_margin: %w", err)
	}
	margin := marginValue.Base()
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin)
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	result, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	return result, nil
}

// CutThroughBore cuts a centered through-bore from a gear body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, fmt.Errorf("context, document, and body are required")
	}
	if p.BoreDiameter <= 0 || math.IsNaN(p.BoreDiameter) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("bore diameter must be positive")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutThroughBore(ctx, doc, body, p)
}

func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	body, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return body, nil
}

// ChamferOuterCaps chamfers the two outer cap loops of an assembled gear.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("context and gear body are required")
	}
	if distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("chamfer distance must be finite and positive")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range edges {
		start := edge.Start().Position().Value
		end := edge.End().Position().Value
		for _, root := range roots {
			target := toWorld(frame, root)
			if start.Sub(target).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
			if end.Sub(target).Len() <= 1e-6 {
				corners = append(corners, end)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, fmt.Errorf("no root edges to fillet")
	}
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(corner))
	}
	result, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet root edges: %w", err)
	}
	return result, nil
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	result, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded root edges: %w", err)
	}
	return result, nil
}
