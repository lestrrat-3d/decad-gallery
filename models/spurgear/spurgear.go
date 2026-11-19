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
	Module, PressureAngle, Thickness   float64
	Teeth, Samples                     int
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
func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }

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
		return fmt.Errorf("pressure angle out of range")
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("bore diameter must be nonnegative")
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("cap chamfer must be nonnegative")
	}
	if math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash out of range")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base || p.BoreDiameter >= 2*d.root || math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("invalid derived gear dimensions")
	}
	return nil
}
func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, item := range []struct {
		name string
		dst  *float64
	}{{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root}, {"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash}} {
		v, e := doc.Param(item.name)
		if e != nil {
			return dimensions{}, fmt.Errorf("parameter %s: %w", item.name, e)
		}
		*item.dst = v.Base()
	}
	return d, nil
}
func derive(p Params) dimensions {
	doc := decad.New()
	if defineParameters(doc.Params(), p) != nil {
		return dimensions{}
	}
	d, e := readDimensions(doc)
	if e != nil {
		return dimensions{}
	}
	return d
}
func filletRadius(_ Params, d dimensions) float64 { return d.fillet }

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document are required")
	}
	if err := validate(p); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	c := circle.CreateCircle(circle.Origin(), r.d.root)
	dim := sketch.NewRadius(c, r.d.root)
	circle.AddConstraint(dim)
	if err := circle.Bind(dim, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	if _, err := circle.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, circle); err != nil {
		return nil, fmt.Errorf("verify root disc sketch: %w", err)
	}
	profiles := circle.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("root disc profile is invalid")
	}
	disc, err := decad.Extrude(ctx, r.doc, circle, profiles[0], r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth})
	if err != nil {
		return nil, fmt.Errorf("pattern teeth: %w", err)
	}
	bodies := append([]*decad.Body{tooth}, copies...)
	body, err := decad.Union(ctx, disc, bodies...)
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
			return nil, fmt.Errorf("parameter cap_chamfer: %w", e)
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

func BuildProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, fmt.Errorf("context, world and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, pr, _, e := buildGearProfileTwisted(ctx, w, pl, p, derive(p), 0)
	return s, pr, e
}
func BuildTwistedProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || w == nil || pl == nil {
		return nil, nil, nil, fmt.Errorf("context, world and plane are required")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("nonfinite twist")
	}
	s, pr, roots, e := buildGearProfileTwisted(ctx, w, pl, p, derive(p), twist)
	out := make([][2]float64, len(roots))
	for i, r := range roots {
		out[i] = [2]float64{r.x, r.y}
	}
	return s, pr, out, e
}

func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("context, document, body and positive bore diameter are required")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}
func cutBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	w := sketch.NewWorld()
	s, pr, err := buildBoreProfile(ctx, w, p, 1)
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("parameter bore_tool_length: %w", err)
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
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("invalid chamfer arguments")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	b, e := decad.Chamfer(ctx, gear, sel, setback)
	if e != nil {
		return nil, fmt.Errorf("cap chamfer: %w", e)
	}
	return b, nil
}

func buildGearProfile(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, w, pl, p, d, 0)
}
func buildGearProfileTwisted(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, w, pl, p, d, twist)
	}
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, nil, err
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		a := toothAngle(p, k) + twist
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, a)
		rrxy := rootContact(r[0], d)
		llxy := rootContact(l[0], d)
		rr := sketchcheck.DimensionPoint(s, rrxy.x, rrxy.y)
		lr := sketchcheck.DimensionArcEnd(s, llxy.x, llxy.y)
		rp := make([]*sketch.Point, len(r))
		lp := make([]*sketch.Point, len(l))
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
		if k > 0 {
			s.CreateArc(s.Origin(), lastLeft, rr)
		} else {
			firstRight = rr
		}
		s.CreateLine(rr, rp[0])
		if _, err = s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, err
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err = s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, err
		}
		s.CreateLine(lp[0], lr)
		roots = append(roots, rrxy, llxy)
		lastLeft = lr
	}
	s.CreateArc(s.Origin(), lastLeft, firstRight)
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid || len(ps[0].Outer) != 6*p.Teeth {
		return nil, nil, nil, fmt.Errorf("gear outline has %d curves, expected %d", profileCurves(ps), 6*p.Teeth)
	}
	return s, ps[0], roots, nil
}
func profileCurves(p []*sketch.Profile) int {
	if len(p) == 1 {
		return len(p[0].Outer)
	}
	return 0
}
func buildEmbeddedOutline(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded gear sketch: %w", err)
	}
	type flankRef struct {
		tooth int
		side  string
		flank *sketch.FitSpline
	}
	flanks := make([]flankRef, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip,
			float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		rp := make([]*sketch.Point, len(right))
		lp := make([]*sketch.Point, len(left))
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
		rightFlank, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		leftFlank, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		flanks = append(flanks, flankRef{tooth: k, side: "right", flank: rightFlank},
			flankRef{tooth: k, side: "left", flank: leftFlank})
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded gear sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded gear sketch: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded gear sketch has no regions")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded gear region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded gear regions: %w", err)
	}
	expected := 4 * p.Teeth
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != expected {
		return nil, nil, nil, fmt.Errorf("embedded gear topology is invalid: valid=%t holes=%d edges=%d, expected valid profile without holes and %d edges",
			profile.Valid, len(profile.Holes), len(profile.Outer), expected)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, len(contacts))
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			for _, ref := range flanks {
				if ref.flank == flank {
					return nil, nil, nil, fmt.Errorf("tooth %d %s trim is not certified", ref.tooth, ref.side)
				}
			}
			return nil, nil, nil, fmt.Errorf("embedded gear has an unknown uncertified spline edge")
		}
		matched := false
		for _, ref := range flanks {
			if ref.flank != flank {
				continue
			}
			matched = true
			slot := 2 * ref.tooth
			if ref.side == "left" {
				slot++
			}
			if filled[slot] {
				return nil, nil, nil, fmt.Errorf("tooth %d %s trim appears more than once", ref.tooth, ref.side)
			}
			x, y := flank.Eval(edge.TStart)
			contacts[slot] = point{x, y}
			filled[slot] = true
			break
		}
		if !matched {
			return nil, nil, nil, fmt.Errorf("embedded gear boundary contains an unknown spline edge")
		}
	}
	for i, contact := range contacts {
		if !filled[i] {
			return nil, nil, nil, fmt.Errorf("tooth %d %s root contact is missing", i/2, []string{"right", "left"}[i%2])
		}
		if math.Abs(math.Hypot(contact.x, contact.y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("tooth %d %s root contact is off root radius", i/2, []string{"right", "left"}[i%2])
		}
	}
	return s, profile, contacts, nil
}

func buildEmbeddedTooth(ctx context.Context, w *sketch.World, pl *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	baseInvolute := involute(d.pitch, d.base)
	beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(-baseInvolute.y, baseInvolute.x)
	halfAngle := func(radius float64) float64 {
		q := involute(radius, d.base)
		a := rotate(q.x, -q.y, beta)
		return math.Abs(math.Atan2(a.y, a.x))
	}
	rc := d.base
	if halfAngle(d.base) >= math.Pi/float64(p.Teeth) {
		if halfAngle(d.root) >= math.Pi/float64(p.Teeth) {
			return nil, nil, fmt.Errorf("no embedded tooth closing radius inside root circle")
		}
		lo, hi := d.base, d.root
		for i := 0; i < 200; i++ {
			mid := (lo + hi) / 2
			if halfAngle(mid) >= math.Pi/float64(p.Teeth) {
				lo = mid
			} else {
				hi = mid
			}
		}
		rc = (lo + hi) / 2
		rc = (rc + d.root) / 2
	}
	s, err := w.CreateSketch(pl)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rp := make([]*sketch.Point, len(right))
	lp := make([]*sketch.Point, len(left))
	for i, v := range right {
		rp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
	}
	for i, v := range left {
		if i == 0 || i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, v.x, v.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, v.x, v.y)
		}
	}
	if _, err = s.CreateFitSpline(rp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err = s.CreateFitSpline(lp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if _, err = s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err = sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid || len(ps[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth profile must have four curves")
	}
	return s, ps[0], nil
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
	dim := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(dim)
	if err = s.Bind(dim, w.Params(), "bore_diameter"); err != nil {
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
func filletRoots(ctx context.Context, b *decad.Body, f r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, b, f, roots, units.Millimeters(radius))
}
func filletRootsValue(ctx context.Context, b *decad.Body, f r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, fmt.Errorf("no root corners")
	}
	edges, err := decad.Edges(decad.ParallelTo(f.N())).SelectEdges(b)
	if err != nil {
		return nil, err
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, e := range edges {
		for _, v := range []r3.Vec{e.Start().Position().Value, e.End().Position().Value} {
			matched := false
			for _, q := range roots {
				if toWorld(f, q).Sub(v).Len() < 1e-6 {
					corners = append(corners, v)
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d of %d root edges", len(corners), len(roots))
	}
	sel := decad.Edges(decad.ParallelTo(f.N()), decad.EndpointAt(corners[0]))
	for _, v := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(f.N()), decad.EndpointAt(v))
	}
	out, err := decad.Fillet(ctx, b, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, err
	}
	return out, nil
}
func filletEmbedded(ctx context.Context, b *decad.Body, f r3.Frame, teeth int, radius units.Value) (*decad.Body, error) {
	out, err := decad.Fillet(ctx, b, decad.Edges(decad.ParallelTo(f.N()), decad.Concave()).Exactly(2*teeth), radius)
	if err != nil {
		return nil, err
	}
	return out, nil
}
