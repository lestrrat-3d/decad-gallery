// Package spurgear builds a cylindrical spur gear with straight involute teeth.
package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters and angles radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement gives the reference point and plane. Nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default gear parameters.
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

// validate applies the input checks shared by Build, BuildProfile and
// BuildToothSection.
func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("spurgear: module %g must be positive", p.Module)
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("spurgear: thickness %g must be positive", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("spurgear: pressure angle %g must be in (0, pi/2)", p.PressureAngle)
	}
	if p.BoreDiameter < 0 {
		return fmt.Errorf("spurgear: bore diameter %g must not be negative", p.BoreDiameter)
	}
	if p.CapChamfer < 0 {
		return fmt.Errorf("spurgear: cap chamfer %g must not be negative", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("spurgear: invalid placement frame")
	}
	d := derive(p)
	if d.root <= 0 {
		return fmt.Errorf("spurgear: root radius %g must be positive", d.root)
	}
	if d.base <= 0 {
		return fmt.Errorf("spurgear: base radius %g must be positive", d.base)
	}
	if d.tip <= d.base {
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("spurgear: bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*d.root)
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("spurgear: root fillet radius %g is not finite", d.fillet)
	}
	return nil
}

// singleProfile returns the only profile of s, requiring it to be valid.
func singleProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("%s: want 1 valid profile, got %d", stage, len(profiles))
	}
	return profiles[0], nil
}

// Build constructs the gear in doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear in doc with the variant's changes to the spur
// pipeline and returns its body.
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

// assemble builds the tooth, disc, pattern and finishing of one prepared run.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	doc, p := r.doc, r.p
	d, frame, lower, extent := r.d, r.frame, r.lower, r.extent
	world, teeth := r.world, r.teeth
	// Tooth zero.
	embedded := d.base < d.root
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
	}
	// Root disc.
	ds, err := world.CreateSketch(lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc sketch: %w", err)
	}
	circle := ds.CreateCircle(ds.Origin(), d.root)
	radius := sketch.NewRadius(circle, d.root)
	ds.AddConstraint(radius)
	if err := ds.Bind(radius, world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: bind root radius: %w", err)
	}
	if err := solveAndVerify(ctx, ds, "spurgear: root disc"); err != nil {
		return nil, err
	}
	dp, err := singleProfile(ds, "spurgear: root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, doc, ds, dp, extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}

	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: frame.Origin(), Axis: frame.N(), Count: teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: joined gear is not a solid")
	}

	// Step 7: root fillet.
	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, doc, body, frame)
		if err != nil {
			return nil, err
		}
	} else if d.fillet > 0 {
		body, err = filletSpurRoots(ctx, doc, body, r, embedded)
		if err != nil {
			return nil, err
		}
	}

	// Step 8: bore.
	if p.BoreDiameter > 0 {
		body, err = cutBore(ctx, doc, body, p)
		if err != nil {
			return nil, err
		}
	}

	// Step 9: cap chamfer.
	if p.CapChamfer > 0 {
		setback, err := doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read cap chamfer: %w", err)
		}
		body, err = chamferOuterCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// BuildProfile draws the unfilleted outline of the whole gear on plane and
// returns its verified sketch and profile.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, fmt.Errorf("spurgear: nil world")
	}
	if plane == nil {
		return nil, nil, fmt.Errorf("spurgear: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, prof, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, prof, err
}

// buildGearProfile draws the whole-gear outline and returns the actual root
// contact coordinates, two per tooth.
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
	last := p.Samples - 1
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		tp := addToothPoints(s, p, d, toothAngle(p, k))
		roots = append(roots, tp.rightRootXY, tp.leftRootXY)
		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, prevLeft, tp.rightRoot)
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, tp.right[last], tp.left[last])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("gear outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		prevLeft = tp.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	prof, err := singleProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, prof, roots, nil
}

// buildEmbeddedOutline draws every original flank and tip arc plus the root
// circle, then unions the bounded regions into one outline.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	last := p.Samples - 1
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rp := make([]*sketch.Point, len(right))
		lp := make([]*sketch.Point, len(left))
		for i, q := range right {
			rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
		for i, q := range left {
			if i == last {
				lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			} else {
				lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
			}
		}
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[last], lp[last])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, pr := range s.Profiles() {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	prof, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !prof.Valid || len(prof.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union is not a valid hole-free profile (valid %t, %d holes)",
			prof.Valid, len(prof.Holes))
	}
	if want := 4 * p.Teeth; len(prof.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, want %d", len(prof.Outer), want)
	}
	var contacts []point
	for _, edge := range prof.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: flank trim is not certified (partial %t, exact %t)",
				edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: trim contact radius %g differs from root radius %g",
				math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if want := 2 * p.Teeth; len(contacts) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d root contacts, want %d", len(contacts), want)
	}
	return s, prof, contacts, nil
}

// halfAngle returns the angle between the tooth axis and the left flank where
// the flank meets radius r.
func halfAngle(d dimensions, teeth float64, r float64) float64 {
	beta := flankBeta(d.pitch, d.base, teeth)
	m := involutePoint(d.base, r)
	l := rotate(m.x, m.y, beta)
	return math.Atan2(l.y, l.x)
}

// closingRadius chooses the radius of the arc that closes an embedded tooth.
func closingRadius(d dimensions, teeth int) (float64, error) {
	n := float64(teeth)
	limit := math.Pi / n
	if halfAngle(d, n, d.base) < limit {
		return d.base, nil
	}
	if halfAngle(d, n, d.root) >= limit {
		return 0, fmt.Errorf("no closing radius: adjacent flanks still overlap at the root circle (radius %g)", d.root)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if halfAngle(d, n, mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws one natural tooth of an embedded gear: two flank
// splines, the tip arc and a closing arc at radius Rc.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(d, p.Teeth)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	last := p.Samples - 1
	rp := make([]*sketch.Point, len(right))
	lp := make([]*sketch.Point, len(left))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: right flank spline: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("embedded tooth: left flank spline: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveAndVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	prof, err := singleProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if n := len(prof.Outer); n != 4 {
		return nil, nil, fmt.Errorf("embedded tooth: got %d outer curves, want 4", n)
	}
	return s, prof, nil
}

// filletRoots fillets the ordinary-branch root corners matching roots.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, e := range edges {
		start := e.Start().Position().Value
		for _, root := range roots {
			if start.Sub(toWorld(frame, root)).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: matched %d root edges, want %d", len(corners), len(roots))
	}
	sel := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		sel = sel.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, sel.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet root edges: %w", err)
	}
	return out, nil
}

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, sel, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet embedded root edges: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the centered bore tool section on a plane one margin
// below the placed lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("bore tool: define parameters: %w", err)
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
		return nil, nil, fmt.Errorf("bore tool: plane: %w", err)
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
	if err := solveAndVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	prof, err := singleProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, prof, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the lower cap through p.Thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("bore: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("bore: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("bore: nil body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("bore: diameter %g must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore: define parameters: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	marginValue, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("bore: read tool margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("bore: read tool length: %w", err)
	}
	world := sketch.NewWorld()
	s, prof, err := buildBoreProfile(ctx, world, p, marginValue.Base())
	if err != nil {
		return nil, fmt.Errorf("bore: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, prof, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("bore: cut tool: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with the given setback.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	return chamferOuterCaps(ctx, gear, units.Millimeters(distance))
}

func chamferOuterCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	sel := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, sel, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return out, nil
}
