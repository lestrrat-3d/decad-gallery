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

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("module, thickness, teeth, or samples are outside their limits")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle is outside its limits")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer must be nonnegative")
	}
	if !finite(p.Backlash) || math.Abs(p.Backlash) > p.Module/2 {
		return fmt.Errorf("backlash is outside its limits")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("derived radii are outside their limits")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter must be smaller than the root circle")
	}
	if !finite(d.fillet) {
		return fmt.Errorf("fillet radius must be finite")
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

func DefineParameters(t *param.Table, p Params) error { return defineParameters(t, p) }

func readDimensions(doc *decad.Document) (dimensions, error) {
	var d dimensions
	for _, item := range []struct {
		name string
		dst  *float64
	}{
		{"pitch_radius", &d.pitch}, {"base_radius", &d.base}, {"root_radius", &d.root},
		{"tip_radius", &d.tip}, {"fillet_radius", &d.fillet}, {"backlash", &d.backlash},
	} {
		v, err := doc.Param(item.name)
		if err != nil {
			return dimensions{}, fmt.Errorf("read %s: %w", item.name, err)
		}
		*item.dst = v.Base()
	}
	return d, nil
}

func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear profile sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lp, rp := make([]*sketch.Point, len(left)), make([]*sketch.Point, len(right))
		for i := range right {
			rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		}
		for i := range left {
			if i == len(left)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		rc, lc := rootContact(right[0], d), rootContact(left[0], d)
		rp0, lp0 := sketchcheck.DimensionPoint(s, rc.x, rc.y), sketchcheck.DimensionArcEnd(s, lc.x, lc.y)
		roots = append(roots, rc, lc)
		if k == 0 {
			firstRight = rp0
		} else {
			s.CreateArc(s.Origin(), lastLeft, rp0)
		}
		s.CreateLine(rp0, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], lp0)
		lastLeft = lp0
	}
	s.CreateArc(s.Origin(), lastLeft, firstRight)
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve gear profile: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify gear profile: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, nil, fmt.Errorf("gear profile has %d valid profiles, expected 1", len(profiles))
	}
	if len(profiles[0].Outer) != 6*p.Teeth {
		return nil, nil, nil, fmt.Errorf("gear profile has %d curves, expected %d", len(profiles[0].Outer), 6*p.Teeth)
	}
	return s, profiles[0], roots, nil
}

func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded profile sketch: %w", err)
	}
	type source struct {
		tooth       int
		right, left *sketch.FitSpline
	}
	sources := make(map[*sketch.FitSpline]source, 2*p.Teeth)
	for k := range p.Teeth {
		l, r := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, toothAngle(p, k)+twist)
		lp, rp := make([]*sketch.Point, len(l)), make([]*sketch.Point, len(r))
		for i := range r {
			rp[i] = sketchcheck.DimensionPoint(s, r[i].x, r[i].y)
		}
		for i := range l {
			if i == len(l)-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, l[i].x, l[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, l[i].x, l[i].y)
			}
		}
		rs, e := s.CreateFitSpline(rp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, e)
		}
		s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
		ls, e := s.CreateFitSpline(lp...)
		if e != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, e)
		}
		sources[rs] = source{tooth: k, right: rs}
		sources[ls] = source{tooth: k, left: ls}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve embedded profile: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify embedded profile: %w", err)
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, fmt.Errorf("embedded profile has no regions")
	}
	indices := make([]int, len(regions))
	for i, pr := range regions {
		indices[i] = i
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("embedded region %d is invalid", i)
		}
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded profile regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded profile is invalid or has holes")
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded profile has %d curves, expected %d", len(profile.Outer), 4*p.Teeth)
	}
	contacts := make([]point, 2*p.Teeth)
	filled := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("tooth %d spline trim is not certified", sources[flank].tooth)
		}
		src, ok := sources[flank]
		if !ok {
			return nil, nil, nil, fmt.Errorf("embedded profile has an unknown spline edge")
		}
		slot := 2 * src.tooth
		if flank == src.left {
			slot++
		}
		if filled[slot] {
			return nil, nil, nil, fmt.Errorf("tooth %d has a duplicate root contact", src.tooth)
		}
		x, y := flank.Eval(edge.TStart)
		contacts[slot] = point{x, y}
		filled[slot] = true
	}
	for i, c := range contacts {
		if !filled[i] {
			return nil, nil, nil, fmt.Errorf("root contact %d is missing", i)
		}
		if math.Abs(math.Hypot(c.x, c.y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("root contact %d is off the root circle", i)
		}
	}
	return s, profile, contacts, nil
}

func embeddedHalfAngle(r float64, d dimensions, p Params) float64 {
	tPitch := math.Tan(math.Acos(d.base / d.pitch))
	px := d.base * (math.Cos(tPitch) + tPitch*math.Sin(tPitch))
	py := -d.base * (math.Sin(tPitch) - tPitch*math.Cos(tPitch))
	beta := math.Pi/(2*float64(p.Teeth)) - d.backlash/(2*d.pitch) - math.Atan2(py, px)
	t := math.Tan(math.Acos(d.base / r))
	x := d.base * (math.Cos(t) + t*math.Sin(t))
	y := -d.base * (math.Sin(t) - t*math.Cos(t))
	q := rotate(x, y, beta)
	return math.Abs(math.Atan2(q.y, q.x))
}

func embeddedClosingRadius(d dimensions, p Params) (float64, error) {
	half := math.Pi / float64(p.Teeth)
	if embeddedHalfAngle(d.base, d, p) < half {
		return d.base, nil
	}
	if embeddedHalfAngle(d.root, d, p) >= half {
		return 0, fmt.Errorf("no embedded tooth closing radius before the root circle")
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if embeddedHalfAngle(mid, d, p) > half {
			lo = mid
		} else {
			hi = mid
		}
	}
	return ((lo+hi)/2 + d.root) / 2, nil
}

func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(d, p)
	if err != nil {
		return nil, nil, err
	}
	l, r := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create embedded tooth sketch: %w", err)
	}
	lp, rp := make([]*sketch.Point, len(l)), make([]*sketch.Point, len(r))
	for i := range r {
		rp[i] = sketchcheck.DimensionPoint(s, r[i].x, r[i].y)
	}
	for i := range l {
		if i == 0 || i == len(l)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, l[i].x, l[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, l[i].x, l[i].y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[len(rp)-1], lp[len(lp)-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, err
	}
	s.CreateArc(s.Origin(), rp[0], lp[0])
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("solve embedded tooth: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("verify embedded tooth: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("embedded tooth has %d valid profiles, expected 1", len(profiles))
	}
	if len(profiles[0].Outer) != 4 {
		return nil, nil, fmt.Errorf("embedded tooth has %d curves, expected 4", len(profiles[0].Outer))
	}
	return s, profiles[0], nil
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane, p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
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
	if !finite(twist) {
		return nil, nil, nil, fmt.Errorf("twist must be finite")
	}
	s, pr, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	out := make([][2]float64, len(roots))
	for i, r := range roots {
		out[i] = [2]float64{r.x, r.y}
	}
	return s, pr, out, err
}

func toothAngle(p Params, k int) float64 { return 2 * math.Pi * float64(k) / float64(p.Teeth) }

func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}

func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
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
	c := s.CreateCircle(s.Origin(), r.d.root)
	rad := sketch.NewRadius(c, r.d.root)
	s.AddConstraint(rad)
	if err := s.Bind(rad, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root radius: %w", err)
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify root disc sketch: %w", err)
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
		body, err = cutThroughBore(ctx, r.doc, body, r.p)
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

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point, radius units.Value) (*decad.Body, error) {
	axial, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, err
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, root := range roots {
		target := toWorld(frame, root)
		found := false
		for _, edge := range axial {
			for _, vertex := range []*decad.Vertex{edge.Start(), edge.End()} {
				got := vertex.Position().Value
				if got.Sub(target).Len() <= 1e-6 {
					corners = append(corners, got)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("root contact at %.9g, %.9g was not found", root.x, root.y)
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet ordinary roots: %w", err)
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
	dia := sketch.NewDiameter(c, p.BoreDiameter)
	s.AddConstraint(dia)
	if err := s.Bind(dia, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, err
	}
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	ps := s.Profiles()
	if len(ps) != 1 || !ps[0].Valid {
		return nil, nil, fmt.Errorf("bore profile has %d profiles, expected 1", len(ps))
	}
	return s, ps[0], nil
}

func cutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	w := sketch.NewWorld()
	if err := defineParameters(w.Params(), p); err != nil {
		return nil, err
	}
	s, pr, err := buildBoreProfile(ctx, w, p, 1)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("cut through bore: %w", err)
	}
	return out, nil
}

func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("context, document, body, and positive bore diameter are required")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutThroughBore(ctx, doc, body, p)
}

func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer caps: %w", err)
	}
	return out, nil
}

func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil || distance <= 0 || !finite(distance) {
		return nil, fmt.Errorf("context, body, and positive finite chamfer distance are required")
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
