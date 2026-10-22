// Package spurgear builds a parametric straight-tooth involute spur gear.
package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters, angles radians.
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

// RejectPlacement returns an error naming model when a Placement is set.
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

// validate applies the input checks and limits.
func validate(p Params) error {
	if !(p.Module > 0) || math.IsInf(p.Module, 0) {
		return fmt.Errorf("spurgear: module must be positive and finite, got %v", p.Module)
	}
	if !(p.Thickness > 0) || math.IsInf(p.Thickness, 0) {
		return fmt.Errorf("spurgear: thickness must be positive and finite, got %v", p.Thickness)
	}
	if p.Teeth < 3 {
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	}
	if p.Samples < 3 {
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	}
	if !(p.PressureAngle > 0 && p.PressureAngle < math.Pi/2) {
		return fmt.Errorf("spurgear: pressure angle must be in (0, pi/2), got %v", p.PressureAngle)
	}
	if !(p.BoreDiameter >= 0) || math.IsInf(p.BoreDiameter, 0) {
		return fmt.Errorf("spurgear: bore diameter must be nonnegative and finite, got %v", p.BoreDiameter)
	}
	if !(p.CapChamfer >= 0) || math.IsInf(p.CapChamfer, 0) {
		return fmt.Errorf("spurgear: cap chamfer must be nonnegative and finite, got %v", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: invalid placement frame")
	}
	d := derive(p)
	if !(d.root > 0) {
		return fmt.Errorf("spurgear: root radius %v must be positive", d.root)
	}
	if !(d.base > 0) {
		return fmt.Errorf("spurgear: base radius %v must be positive", d.base)
	}
	if !(d.tip > d.base) {
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", d.tip, d.base)
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("spurgear: bore diameter %v must be below root diameter %v", p.BoreDiameter, 2*d.root)
	}
	if rf := filletRadius(p, d); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("spurgear: nonfinite root fillet radius %v", rf)
	}
	return nil
}

// BuildProfile draws the unfilleted whole-gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// buildGearProfile draws the whole outline and returns the actual root contacts.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("outline: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, lastLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		tp := createToothPoints(s, left, right, d)
		roots = append(roots, tp.rightRootXY, tp.leftRootXY)

		if k == 0 {
			firstRight = tp.rightRoot
		} else {
			s.CreateArc(center, lastLeft, tp.rightRoot)
		}
		s.CreateLine(tp.rightRoot, tp.right[0])
		if _, err := s.CreateFitSpline(tp.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, tp.right[len(tp.right)-1], tp.left[len(tp.left)-1])
		if _, err := s.CreateFitSpline(tp.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("outline: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(tp.left[0], tp.leftRoot)
		lastLeft = tp.leftRoot
	}
	s.CreateArc(center, lastLeft, firstRight)

	if err := solveAndVerify(ctx, s, "outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile draws the flanks and tip arcs over a full root circle and
// unions the bounded regions into one outline.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		rightPts, leftPts := flankPoints(s, left, right)
		_, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("embedded outline: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		_, err = s.CreateFitSpline(leftPts...)
		if err != nil {
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
			return nil, nil, nil, fmt.Errorf("embedded outline: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("embedded outline: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline: union profile has %d holes", len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	var contacts []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d spline trim is not certified", i)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded outline: edge %d root contact radius %v, expected %v",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if want := 2 * p.Teeth; len(contacts) != want {
		return nil, nil, nil, fmt.Errorf("embedded outline: got %d root contacts, expected %d",
			len(contacts), want)
	}
	return s, profile, contacts, nil
}

// Build constructs the gear in doc and returns its body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: nil document")
	}
	if err := validate(p); err != nil {
		return nil, err
	}
	d := derive(p)
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

	var body *decad.Body
	var roots []point
	if d.base >= d.root {
		body, roots, err = buildOrdinaryBody(ctx, doc, world, lower, upper, p, d)
	} else {
		body, roots, err = buildEmbeddedBody(ctx, doc, world, lower, upper, p, d)
	}
	if err != nil {
		return nil, err
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: gear body is not solid")
	}

	if rf := filletRadius(p, d); rf > 0 {
		if body, err = filletRoots(ctx, body, frame, roots, rf); err != nil {
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
	return body, nil
}

func buildOrdinaryBody(ctx context.Context, doc *decad.Document, world *sketch.World,
	lower, upper *sketch.Plane, p Params, d dimensions) (*decad.Body, []point, error) {
	ds, err := world.CreateSketch(lower)
	if err != nil {
		return nil, nil, fmt.Errorf("root disc: create sketch: %w", err)
	}
	circle := ds.CreateCircle(ds.Origin(), d.root)
	ds.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, ds, "root disc"); err != nil {
		return nil, nil, err
	}
	dp, err := singleProfile(ds, "root disc")
	if err != nil {
		return nil, nil, err
	}
	disc, err := decad.Extrude(ctx, doc, ds, dp,
		decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, nil, fmt.Errorf("root disc: extrude: %w", err)
	}

	teeth := make([]*decad.Body, 0, p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		angle := toothAngle(p, k)
		ls, lp, err := buildToothSection(ctx, world, lower, p, d, angle)
		if err != nil {
			return nil, nil, fmt.Errorf("tooth %d: lower section: %w", k, err)
		}
		us, up, err := buildToothSection(ctx, world, upper, p, d, angle)
		if err != nil {
			return nil, nil, fmt.Errorf("tooth %d: upper section: %w", k, err)
		}
		tooth, err := decad.Loft(ctx, doc, ls, lp, us, up)
		if err != nil {
			return nil, nil, fmt.Errorf("tooth %d: loft: %w", k, err)
		}
		teeth = append(teeth, tooth)
	}
	body, err := decad.Union(ctx, disc, teeth...)
	if err != nil {
		return nil, nil, fmt.Errorf("join disc and %d teeth: %w", len(teeth), err)
	}

	left0, right0 := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, 0)
	rr := scaleToRoot(right0[0], d)
	lr := scaleToRoot(left0[0], d)
	roots := make([]point, 0, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		angle := toothAngle(p, k)
		roots = append(roots, rotate(rr.x, rr.y, angle), rotate(lr.x, lr.y, angle))
	}
	return body, roots, nil
}

func buildEmbeddedBody(ctx context.Context, doc *decad.Document, world *sketch.World,
	lower, upper *sketch.Plane, p Params, d dimensions) (*decad.Body, []point, error) {
	ls, lp, roots, err := buildGearProfile(ctx, world, lower, p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("lower outline: %w", err)
	}
	us, up, _, err := buildGearProfile(ctx, world, upper, p, d)
	if err != nil {
		return nil, nil, fmt.Errorf("upper outline: %w", err)
	}
	body, err := decad.Loft(ctx, doc, ls, lp, us, up)
	if err != nil {
		return nil, nil, fmt.Errorf("outline loft: %w", err)
	}
	return body, roots, nil
}

// filletRoots fillets the axial edges that start at a saved root contact.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("root fillet: select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, r := range roots {
		world[i] = toWorld(frame, r)
	}
	var corners []r3.Vec
	for _, e := range edges {
		start := e.Start().Position().Value
		for _, w := range world {
			if w.Sub(start).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile draws the through-bore tool circle on a plane margin below
// the lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of diameter p.BoreDiameter from
// body, which must span z=0 through p.Thickness in the placement frame.
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
		return nil, fmt.Errorf("spurgear: bore diameter must be positive, got %v", p.BoreDiameter)
	}
	const margin = 1.0
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("bore tool: extrude: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("bore: cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance must be positive, got %v", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("cap chamfer: %w", err)
	}
	return out, nil
}
