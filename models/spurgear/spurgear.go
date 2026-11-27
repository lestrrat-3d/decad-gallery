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
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15, Backlash: .05}
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

func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("module must be positive")
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("thickness must be positive")
	}
	if p.Teeth < 3 {
		return fmt.Errorf("teeth must be at least 3")
	}
	if p.Samples < 3 {
		return fmt.Errorf("samples must be at least 3")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle must be between zero and pi/2")
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("bore diameter must be nonnegative")
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("cap chamfer must be nonnegative")
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash must be finite and no greater than half the module")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("placement is invalid")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("derived gear radii are invalid")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter must be smaller than the root diameter")
	}
	if !finite(d.fillet) {
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

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}

func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must be nonnil")
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
	discSketch, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := discSketch.CreateCircle(discSketch.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	discSketch.AddConstraint(radius)
	if err := discSketch.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	if _, err := discSketch.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, discSketch); err != nil {
		return nil, fmt.Errorf("verify root disc: %w", err)
	}
	profiles := discSketch.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("root disc has %d valid profiles, expected one", len(profiles))
	}
	disc, err := decad.Extrude(r.doc, discSketch, profiles[0], r.extent).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	body, err := decad.Union(append([]*decad.Body{disc, tooth}, copies...)...).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("join root disc and teeth: %w", err)
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
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		distance, e := r.doc.Param("cap_chamfer")
		if e != nil {
			return nil, fmt.Errorf("cap_chamfer: %w", e)
		}
		body, err = chamferCaps(ctx, body, distance)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("finished gear is not solid")
	}
	return body, nil
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear outline: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rroot, lroot := rootContact(right[0], d), rootContact(left[0], d)
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
		lr := sketchcheck.DimensionArcEnd(s, lroot.x, lroot.y)
		rr := sketchcheck.DimensionPoint(s, rroot.x, rroot.y)
		if k == 0 {
			firstRight = rr
		} else {
			s.CreateArc(center, lastLeft, rr)
		}
		s.CreateLine(rr, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], lr)
		lastLeft = lr
		roots = append(roots, rroot, lroot)
	}
	s.CreateArc(center, lastLeft, firstRight)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve ordinary gear outline: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify ordinary gear outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("ordinary gear outline has %d valid profiles, expected one", len(profiles))
	}
	if len(profiles[0].Outer) != 6*p.Teeth {
		return nil, nil, nil, fmt.Errorf("ordinary outline has %d curves, expected %d", len(profiles[0].Outer), 6*p.Teeth)
	}
	return s, profiles[0], roots, nil
}

func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded outline: %w", err)
	}
	type pair struct{ right, left *sketch.FitSpline }
	splines := make([]pair, p.Teeth)
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
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
		rs, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, e)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		ls, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, e)
		}
		splines[k] = pair{rs, ls}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded outline: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded outline: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline has no regions")
	}
	ids := make([]int, len(profiles))
	for i, pr := range profiles {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("embedded region %d is invalid", i)
		}
		ids[i] = i
	}
	pr, err := s.UnionProfiles(ids...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded regions: %w", err)
	}
	if !pr.Valid || len(pr.Holes) != 0 || len(pr.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outline has %d curves, expected %d, holes=%d valid=%t", len(pr.Outer), 4*p.Teeth, len(pr.Holes), pr.Valid)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range pr.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded flank trim is not certified")
		}
		matched := false
		for k, pair := range splines {
			for side, curve := range []*sketch.FitSpline{pair.right, pair.left} {
				if flank == curve {
					idx := 2*k + side
					x, y := curve.Eval(edge.TStart)
					contacts[idx] = point{x, y}
					filled[idx] = true
					matched = true
				}
			}
		}
		if !matched {
			return nil, nil, nil, fmt.Errorf("unknown embedded flank trim")
		}
	}
	for i, q := range contacts {
		if !filled[i] || math.Abs(math.Hypot(q.x, q.y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded root contact %d missing or off root circle", i)
		}
	}
	return s, pr, contacts, nil
}

func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("context, world, and plane must be nonnil")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, pr, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, pr, err
}
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, fmt.Errorf("context, world, and plane must be nonnil")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	s, pr, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, pr, out, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	angleAt := func(r float64) float64 {
		l, _ := sampleFlanks(d.pitch, d.base, d.base, r, float64(p.Teeth), d.backlash, 3, 0)
		return math.Abs(math.Atan2(l[len(l)-1].y, l[len(l)-1].x))
	}
	rc := d.base
	if angleAt(d.base) >= math.Pi/float64(p.Teeth) {
		if angleAt(d.root) >= math.Pi/float64(p.Teeth) {
			return nil, nil, fmt.Errorf("root circle admits no embedded tooth closing radius")
		}
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
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	center := s.Origin()
	rp := make([]*sketch.Point, len(right))
	lp := make([]*sketch.Point, len(left))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded right flank: %w", err)
	}
	s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded left flank: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid || len(profiles[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth profile does not have one valid four-curve region")
	}
	return s, profiles[0], nil
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
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	d := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(d)
	if err := s.Bind(d, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, err
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("bore sketch did not produce one valid profile")
	}
	return s, profiles[0], nil
}

func cutThroughBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, pr, err := buildBoreProfile(ctx, world, p, 1)
	if err != nil {
		return nil, fmt.Errorf("bore profile: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(doc, s, pr, decad.Distance{D: length, Dir: decad.Along}).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	result, err := decad.Cut(gear, tool).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return result, nil
}
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, fmt.Errorf("context, document, and body must be nonnil")
	}
	if p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("bore diameter must be positive")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutThroughBore(ctx, doc, body, p)
}

func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	result, err := decad.Chamfer(gear, sel, setback).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return result, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("context and gear must be nonnil")
	}
	if !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("chamfer distance must be positive and finite")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, err
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range edges {
		for _, v := range []*decad.Vertex{edge.Start(), edge.End()} {
			pos := v.Position().Value
			for _, root := range roots {
				want := toWorld(frame, root)
				delta := pos.Sub(want)
				if delta.Len() < 1e-6 {
					corners = append(corners, pos)
					break
				}
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(corner))
	}
	result, err := decad.Fillet(body, sel.Exactly(len(corners)), radius).Do(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	result, err := decad.Fillet(body, sel, radius).Do(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}
