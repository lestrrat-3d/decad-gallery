// Package spurgear builds a cylindrical spur gear with straight involute teeth.
package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters and angles are
// radians. A nil Placement selects the world XY frame at the origin.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Placement     *r3.Frame
}

// DefaultParams returns the default gear inputs.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

// RejectPlacement returns an error naming model when p carries a Placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// placement returns the gear frame.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted returns f moved dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps frame coordinates to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks shared by Build, BuildProfile and
// BuildToothSection.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("module %g must be positive", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case !(p.PressureAngle > 0 && p.PressureAngle < math.Pi/2):
		return fmt.Errorf("pressure angle %g must lie in (0, pi/2)", p.PressureAngle)
	case !(p.BoreDiameter >= 0):
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case !(p.CapChamfer >= 0):
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return fmt.Errorf("placement is not a valid frame")
	}
	d := derive(p)
	switch {
	case !(d.root > 0):
		return fmt.Errorf("root radius %g must be positive", d.root)
	case !(d.base > 0):
		return fmt.Errorf("base radius %g must be positive", d.base)
	case !(d.tip > d.base):
		return fmt.Errorf("tip radius %g must exceed base radius %g", d.tip, d.base)
	case !(p.BoreDiameter < 2*d.root):
		return fmt.Errorf("bore diameter %g must be below the root diameter %g", p.BoreDiameter, 2*d.root)
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("root fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// defineParameters writes the gear's parameter table.
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
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2",
		units.Millimeter); err != nil {
		return fmt.Errorf("parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("parameter bore_tool_length: %w", err)
	}
	return nil
}

// Build builds the gear into doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build builds the gear through the variant pipeline.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spur gear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spur gear: nil document")
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

// assemble extrudes the root disc and tooth zero, patterns and joins them,
// then applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("spur gear: tooth: %w", err)
	}
	disc, err := extrudeRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spur gear: pattern teeth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spur gear: join teeth: %w", err)
	}
	if !gear.IsSolid() {
		return nil, errors.New("spur gear: joined body is not solid")
	}
	if v.RootFillet != nil {
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spur gear: root fillet: %w", err)
		}
	} else if r.d.fillet > 0 {
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("spur gear: root fillet: %w", err)
		}
	}
	if r.p.BoreDiameter > 0 {
		gear, err = throughBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spur gear: read cap_chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, errors.New("spur gear: final body is not solid")
	}
	return gear, nil
}

// extrudeRootDisc extrudes the root disc on the lower plane.
func extrudeRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("root disc: bind radius: %w", err)
	}
	if err := solveVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := onlyProfile(s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("root disc: extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile builds the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("gear profile: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("gear profile: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("gear profile: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// buildGearProfile draws the whole-gear outline and returns the actual root
// contacts of every tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeft, firstRight *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		t := makeToothPoints(s, p, d, toothAngle(p, k))
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(center, prevLeft, t.rightRoot)
		}
		if err := drawToothCurves(s, t); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d: %w", k, err)
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the original flanks and tip arcs with a full
// root circle, and unions the bounded regions into one outline.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rp := make([]*sketch.Point, p.Samples)
		lp := make([]*sketch.Point, p.Samples)
		for i := 0; i < p.Samples; i++ {
			rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
			if i == p.Samples-1 {
				lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		_, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		_, err = s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, errors.New("embedded outline: union is not a valid hole-free outline")
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, want %d", got, want)
	}
	var contacts []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: boundary curve %d lacks a certified spline trim", i)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: boundary curve %d trims at radius %g, want %g",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if got, want := len(contacts), 2*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d root contacts, want %d", got, want)
	}
	return s, profile, contacts, nil
}

// flankHalfAngle returns the angle between the tooth axis and the left flank
// where the flank meets radius r.
func flankHalfAngle(d dimensions, teeth float64, r float64) float64 {
	beta := flankShift(d.pitch, d.base, teeth)
	m := mirroredInvolute(d.base, r)
	l := rotate(m.x, m.y, beta)
	return math.Atan2(l.y, l.x)
}

// closingRadius returns the radius of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth int) (float64, error) {
	n := float64(teeth)
	limit := math.Pi / n
	if flankHalfAngle(d, n, d.base) < limit {
		return d.base, nil
	}
	if flankHalfAngle(d, n, d.root) >= limit {
		return 0, fmt.Errorf("embedded tooth: adjacent flanks overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if flankHalfAngle(d, n, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws the four-curve tooth whose flanks cross the root
// circle, closed by an arc at the closing radius.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(d, p.Teeth)
	if err != nil {
		return nil, nil, err
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	rp := make([]*sketch.Point, p.Samples)
	lp := make([]*sketch.Point, p.Samples)
	for i := 0; i < p.Samples; i++ {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		if i == 0 || i == p.Samples-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	last := p.Samples - 1
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if n := len(profile.Outer); n != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d outer curves, want 4", n)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges with a float radius.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start at a saved root contact.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet: select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, w := range world {
			if start.Sub(w).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d root edges, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("embedded root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the
// gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	low, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: shift frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(low)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bore tool: bind diameter: %w", err)
	}
	if err := solveVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("through bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("through bore: nil document")
	}
	if body == nil {
		return nil, errors.New("through bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("through bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("through bore: %w", err)
	}
	return throughBore(ctx, doc, body, p)
}

// throughBore builds the bore tool and cuts it from gear.
func throughBore(ctx context.Context, doc *decad.Document, gear *decad.Body,
	p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("through bore: read tool_margin: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, marginValue.Base())
	if err != nil {
		return nil, err
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("through bore: read bore_tool_length: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("through bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("through bore: cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("cap chamfer: nil body")
	}
	if !(distance > 0) {
		return nil, fmt.Errorf("cap chamfer: setback %g must be positive", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loops of both caps by setback.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}
