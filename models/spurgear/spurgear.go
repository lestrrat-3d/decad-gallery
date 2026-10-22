// Package spurgear builds a cylindrical gear with straight involute teeth.
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
	// Placement is the reference point and plane; nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth gear.
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

// validate applies the input checks and limits and returns the dimensions.
func validate(p Params) (dimensions, error) {
	switch {
	case p.Module <= 0:
		return dimensions{}, fmt.Errorf("spurgear: module %v must be positive", p.Module)
	case p.Thickness <= 0:
		return dimensions{}, fmt.Errorf("spurgear: thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return dimensions{}, fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return dimensions{}, fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return dimensions{}, fmt.Errorf("spurgear: pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return dimensions{}, fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return dimensions{}, fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return dimensions{}, errors.New("spurgear: invalid placement frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return dimensions{}, fmt.Errorf("spurgear: root radius %v must be positive", d.root)
	case d.base <= 0:
		return dimensions{}, fmt.Errorf("spurgear: base radius %v must be positive", d.base)
	case d.tip <= d.base:
		return dimensions{}, fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return dimensions{}, fmt.Errorf("spurgear: bore diameter %v must be below root diameter %v",
			p.BoreDiameter, 2*d.root)
	}
	if rf := filletRadius(p, d); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return dimensions{}, fmt.Errorf("spurgear: root fillet radius %v is not finite", rf)
	}
	return d, nil
}

// BuildProfile draws the whole-gear outline of both branches on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil sketch world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: nil sketch plane")
	}
	d, err := validate(p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	return s, profile, err
}

// flankPoints creates the dimensioned flank points: every right point with
// both coordinates, every left point likewise except the tip.
func flankPoints(s *sketch.Sketch, left, right []point) (rp, lp []*sketch.Point) {
	n := len(right)
	rp = make([]*sketch.Point, n)
	lp = make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n-1; i++ {
		lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
	}
	lp[n-1] = sketchcheck.DimensionArcEnd(s, left[n-1].x, left[n-1].y)
	return rp, lp
}

// buildGearProfile draws the whole-gear outline and returns the actual root
// contacts, right then left for each tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		t := makeToothPoints(s, d, left, right)
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(center, prevLeft, t.rightRoot)
		}
		if err := drawToothSide(s, t); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d: %w", k, err)
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline: %w", err)
	}
	if want := 4*p.Teeth + 2*p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: outline has %d outer curves, want %d", len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the original flanks and tip arcs plus one root
// circle and unions the bounded regions into one outline.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rp, lp := flankPoints(s, left, right)
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: %w", err)
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline union: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded outline union is invalid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d boundary curves, want %d",
			len(profile.Outer), want)
	}
	roots := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline edge %d spline trim is not certified "+
				"(partial %v, exact %v)", i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline edge %d trim radius %v differs from root %v",
				i, math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if len(roots) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d root contacts, want %d",
			len(roots), 2*p.Teeth)
	}
	return s, profile, roots, nil
}

// embeddedClosingRadius returns Rc for the embedded tooth.
func embeddedClosingRadius(p Params, d dimensions) (float64, error) {
	n := float64(p.Teeth)
	h := func(r float64) float64 { return halfAngle(d.pitch, d.base, n, r) }
	limit := math.Pi / n
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: no closing radius: the flank half-angle at the root circle "+
			"(radius %v) is %v, not below pi/N = %v", d.root, h(d.root), limit)
	}
	lo, hi := d.base, d.root
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if h(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws the four-curve embedded tooth section.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := embeddedClosingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), p.Samples, 0)
	n := p.Samples
	rp := make([]*sketch.Point, n)
	lp := make([]*sketch.Point, n)
	for i := 0; i < n; i++ {
		rp[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
	}
	for i := 0; i < n; i++ {
		if i == 0 || i == n-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
		}
	}
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth right flank spline: %w", err)
	}
	s.CreateArc(center, rp[n-1], lp[n-1])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth left flank spline: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: %w", err)
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth has %d outer curves, want 4", got)
	}
	return s, profile, nil
}

// buildRootDisc extrudes the root circle from plane through the thickness.
func buildRootDisc(ctx context.Context, doc *decad.Document, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*decad.Body, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	body, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("root disc extrude: %w", err)
	}
	return body, nil
}

// Build constructs the gear in doc and returns its single solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	d, err := validate(p)
	if err != nil {
		return nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: placement: %w", err)
	}
	world := sketch.NewWorld()
	lower := world.XY()
	if p.Placement != nil {
		if lower, err = world.CreatePlaneFromFrame(frame); err != nil {
			return nil, fmt.Errorf("spurgear: lower plane: %w", err)
		}
	}
	upperFrame, err := shifted(frame, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(upperFrame)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper plane: %w", err)
	}

	disc, err := buildRootDisc(ctx, doc, world, lower, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %w", err)
	}
	embedded := d.base < d.root
	var s0, s1 *sketch.Sketch
	var p0, p1 *sketch.Profile
	if embedded {
		if s0, p0, err = buildEmbeddedTooth(ctx, world, lower, p, d); err != nil {
			return nil, fmt.Errorf("spurgear: lower tooth section: %w", err)
		}
		if s1, p1, err = buildEmbeddedTooth(ctx, world, upper, p, d); err != nil {
			return nil, fmt.Errorf("spurgear: upper tooth section: %w", err)
		}
	} else {
		if s0, p0, err = buildToothSection(ctx, world, lower, p, d, 0); err != nil {
			return nil, fmt.Errorf("spurgear: lower tooth section: %w", err)
		}
		if s1, p1, err = buildToothSection(ctx, world, upper, p, d, 0); err != nil {
			return nil, fmt.Errorf("spurgear: upper tooth section: %w", err)
		}
	}
	tooth, err := decad.Loft(ctx, doc, s0, p0, s1, p1)
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth loft: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: frame.Origin(), Axis: frame.N(), Count: p.Teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth pattern: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth union: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: joined gear is not a solid")
	}

	if radius := filletRadius(p, d); radius > 0 {
		if embedded {
			body, err = filletEmbeddedRoots(ctx, body, frame, p.Teeth, radius)
		} else {
			body, err = filletRoots(ctx, body, frame, orderedRoots(p, d), radius)
		}
		if err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		if body, err = CutThroughBore(ctx, doc, body, p); err != nil {
			return nil, err
		}
	}
	if p.CapChamfer > 0 {
		if body, err = ChamferOuterCaps(ctx, body, p.CapChamfer); err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: finished gear is not a solid")
	}
	return body, nil
}

// orderedRoots lists the root contacts of every tooth, right then left.
func orderedRoots(p Params, d dimensions) []point {
	left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, 0)
	rr, lr := scaleRoot(right[0], d), scaleRoot(left[0], d)
	roots := make([]point, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		a := toothAngle(p, k)
		roots = append(roots, rotate(a, rr.x, rr.y), rotate(a, lr.x, lr.y))
	}
	return roots
}

// filletRoots fillets the ordinary-branch axial edges that start at a saved
// root contact on the lower cap.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
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
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return filleted, nil
}

// filletEmbeddedRoots fillets the 2N concave axial edges of an embedded gear.
func filletEmbeddedRoots(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius float64) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: embedded root fillet: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the centered bore circle on a plane margin below the
// gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	if body == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("spurgear: bore diameter %v must be positive", p.BoreDiameter)
	}
	const margin = 1.0
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore tool extrude: %w", err)
	}
	cut, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore cut: %w", err)
	}
	return cut, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with setback distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance %v must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return chamfered, nil
}
