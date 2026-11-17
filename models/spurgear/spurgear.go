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
		return fmt.Errorf("bore diameter must not be negative")
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("cap chamfer must not be negative")
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
		return fmt.Errorf("bore diameter must be smaller than root diameter")
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("invalid fillet radius")
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
func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }

func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear profile: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var first, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		a := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, a)
		rc, lc := rootContact(right[0], d), rootContact(left[0], d)
		roots = append(roots, rc, lc)
		rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
		for i, v := range right {
			rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
		for i, v := range left {
			if i == p.Samples-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
			}
		}
		pl := sketchcheck.DimensionArcEnd(s, lc.x, lc.y)
		pr := sketchcheck.DimensionPoint(s, rc.x, rc.y)
		if k == 0 {
			first = pr
		} else {
			s.CreateArc(center, lastLeft, pr)
		}
		s.CreateLine(pr, rp[0])
		if _, err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		if _, err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], pl)
		lastLeft = pl
	}
	s.CreateArc(center, lastLeft, first)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve gear profile: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify gear profile: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("ordinary profile count %d, expected one valid profile", len(profiles))
	}
	if len(profiles[0].Outer) != 6*p.Teeth {
		return nil, nil, nil, fmt.Errorf("ordinary profile has %d curves, expected %d", len(profiles[0].Outer), 6*p.Teeth)
	}
	return s, profiles[0], roots, nil
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}
func BuildProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, fmt.Errorf("context, world and plane are required")
	}
	if e := validate(p); e != nil {
		return nil, nil, e
	}
	s, pr, _, e := buildGearProfile(ctx, w, pl, p, derive(p))
	return s, pr, e
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, nil, fmt.Errorf("context, world and plane are required")
	}
	if e := validate(p); e != nil {
		return nil, nil, nil, e
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	s, pr, r, e := buildGearProfileTwisted(ctx, w, pl, p, derive(p), twist)
	out := make([][2]float64, len(r))
	for i, v := range r {
		out[i] = [2]float64{v.x, v.y}
	}
	return s, pr, out, e
}

func buildEmbeddedProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded profile: %w", err)
	}
	center := s.Origin()
	type flankRef struct {
		k    int
		side int
		f    *sketch.FitSpline
	}
	refs := make([]flankRef, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
		for i, v := range right {
			rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
		for i, v := range left {
			if i == p.Samples-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
			}
		}
		rs, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, e)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		ls, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, e)
		}
		refs = append(refs, flankRef{k, 0, rs}, flankRef{k, 1, ls})
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded profile: %w", err)
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded profile: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded profile has no regions")
	}
	indices := make([]int, len(profiles))
	for i, pr := range profiles {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("embedded region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded boundary has %d curves, expected %d and no holes", len(profile.Outer), 4*p.Teeth)
	}
	roots := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		f, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded flank trim is not certified")
		}
		found := false
		for _, ref := range refs {
			if ref.f == f {
				idx := 2*ref.k + ref.side
				if filled[idx] {
					return nil, nil, nil, fmt.Errorf("duplicate embedded trim for tooth %d", ref.k)
				}
				x, y := f.Eval(edge.TStart)
				roots[idx] = point{x, y}
				filled[idx] = true
				found = true
				break
			}
		}
		if !found {
			return nil, nil, nil, fmt.Errorf("unknown embedded spline edge")
		}
	}
	for i, r := range roots {
		if !filled[i] || math.Abs(math.Hypot(r.x, r.y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("invalid embedded root contact %d", i)
		}
	}
	return s, profile, roots, nil
}

func closingRadius(p Params, d dimensions) (float64, error) {
	half := func(r float64) float64 {
		v := involute(d.base, r)
		v.y = -v.y
		ip := involute(d.base, d.pitch)
		ip.y = -ip.y
		beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(ip.y, ip.x)
		v = rotate(v.x, v.y, beta)
		return math.Abs(math.Atan2(v.y, v.x))
	}
	target := math.Pi / float64(p.Teeth)
	if half(d.root) >= target {
		return 0, fmt.Errorf("no closing radius inside root circle")
	}
	if half(d.base) < target {
		return d.base, nil
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if half(mid) > target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (hi + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, e := closingRadius(p, d)
	if e != nil {
		return nil, nil, e
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, e := w.CreateSketch(pl)
	if e != nil {
		return nil, nil, e
	}
	rp, lp := make([]*sketch.Point, p.Samples), make([]*sketch.Point, p.Samples)
	for i, v := range right {
		rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
	}
	for i, v := range left {
		if i == 0 || i == p.Samples-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
	}
	c := s.Origin()
	if _, e = s.CreateFitSpline(rp...); e != nil {
		return nil, nil, e
	}
	s.CreateArc(c, rp[p.Samples-1], lp[p.Samples-1])
	if _, e = s.CreateFitSpline(lp...); e != nil {
		return nil, nil, e
	}
	s.CreateArc(c, rp[0], lp[0])
	if _, e = s.Solve(ctx); e != nil {
		return nil, nil, e
	}
	if e = sketchcheck.VerifyFullyConstrained(ctx, s); e != nil {
		return nil, nil, e
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid || len(ps[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth profile invalid or has wrong curve count")
	}
	return s, ps[0], nil
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
	r, e := v.begin(ctx, doc, p)
	if e != nil {
		return nil, e
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
		return nil, fmt.Errorf("root disc sketch: %w", err)
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
		return nil, fmt.Errorf("root disc profile count %d, expected one valid profile", len(profiles))
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profiles[0], r.extent)
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
		return nil, fmt.Errorf("finished gear is not solid")
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
	pl, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, nil, err
	}
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, err
	}
	c := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	d := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(d)
	if err = s.Bind(d, w.Params(), "bore_diameter"); err != nil {
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
		return nil, nil, fmt.Errorf("bore profile is invalid")
	}
	return s, ps[0], nil
}
func cutBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	w := sketch.NewWorld()
	s, pr, err := buildBoreProfile(ctx, w, p, 1)
	if err != nil {
		return nil, fmt.Errorf("build bore profile: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, pr, decad.Distance{D: length, Dir: decad.Along})
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
		return nil, fmt.Errorf("context, document and body are required")
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
	b, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return b, nil
}
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("context and gear are required")
	}
	if distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("distance must be positive and finite")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, err
	}
	corners := make([]r3.Vec, 0, len(roots))
	near := func(a, b r3.Vec) bool { d := a.Sub(b); return math.Sqrt(d.X*d.X+d.Y*d.Y+d.Z*d.Z) < 1e-6 }
	for _, e := range edges {
		a := e.Start().Position().Value
		b := e.End().Position().Value
		for _, r := range roots {
			target := toWorld(frame, r)
			if near(a, target) {
				corners = append(corners, a)
				break
			}
			if near(b, target) {
				corners = append(corners, b)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		sel.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(c))
	}
	sel.Exactly(len(corners))
	out, err := decad.Fillet(ctx, body, sel, radius)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, sel, radius)
	if err != nil {
		return nil, err
	}
	return out, nil
}
