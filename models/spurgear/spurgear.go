// Package spurgear builds a parametric straight involute spur gear.
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

// Params describes one spur gear. Lengths are in millimeters and angles are in
// radians. Zero fields are not defaults; use DefaultParams.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement is the reference point and plane. Nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth, module-1 gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
	}
}

// RejectPlacement returns an error naming model when Placement is set. Models
// that reuse Params without placement support call it.
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

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// validate applies the input checks and limits.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("spurgear: module %v must be positive", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("spurgear: thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("spurgear: pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return fmt.Errorf("spurgear: placement is not a valid frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return fmt.Errorf("spurgear: root radius %v must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("spurgear: base radius %v must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %v must be below root diameter %v", p.BoreDiameter, 2*d.root)
	case !finite(filletRadius(p, d)):
		return fmt.Errorf("spurgear: root fillet radius is not finite")
	}
	return nil
}

// solveVerify solves a sketch and requires it to be fully constrained.
func solveVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	return nil
}

// onlyProfile returns the single valid profile of a solved sketch.
func onlyProfile(s *sketch.Sketch, stage string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, expected 1", stage, len(profiles))
	}
	if !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is not valid", stage)
	}
	return profiles[0], nil
}

// flankPoints creates the dimensioned sample points of one tooth. Every right
// point carries both coordinates; every left point does too except its tip.
func flankPoints(s *sketch.Sketch, left, right []point) (lp, rp []*sketch.Point) {
	lp = make([]*sketch.Point, len(left))
	rp = make([]*sketch.Point, len(right))
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
	return lp, rp
}

// rootCoords scales a base sample to its root contact, multiplying before
// dividing.
func rootCoords(q point, d dimensions) point {
	return point{q.x * d.root / d.base, q.y * d.root / d.base}
}

func toothAngle(p Params, k int) float64 {
	return 2 * math.Pi * float64(k) / float64(p.Teeth)
}

// BuildProfile builds the unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil {
		return fmt.Errorf("spurgear: nil context")
	}
	if world == nil {
		return fmt.Errorf("spurgear: nil world")
	}
	if plane == nil {
		return fmt.Errorf("spurgear: nil plane")
	}
	return nil
}

// buildGearProfile builds the complete outline and returns the actual root
// contact coordinates.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create profile sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeft, firstRight *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		lp, rp := flankPoints(s, left, right)
		lr := rootCoords(left[0], d)
		rr := rootCoords(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		roots = append(roots, rr, lr)
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(lp[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	if err := solveVerify(ctx, s, "gear profile"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s, "gear profile")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile builds the outline for gears whose base circle lies
// inside the root circle: flanks are trimmed by a full root circle.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create embedded profile sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		lp, rp := flankPoints(s, left, right)
		_, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		_, err = s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveVerify(ctx, s, "embedded gear profile"); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	indices := make([]int, 0, len(regions))
	for i, r := range regions {
		if !r.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded gear profile: region %d of %d is not valid",
				i, len(regions))
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded gear profile: union of %d regions: %w",
			len(indices), err)
	}
	if !profile.Valid {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded gear profile: union is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded gear profile: union has %d holes, expected 0",
			len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded gear profile: got %d boundary curves, expected %d",
			len(profile.Outer), want)
	}
	roots := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf(
				"spurgear: embedded gear profile: edge %d spline trim is not certified (partial=%t exact=%t)",
				i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf(
				"spurgear: embedded gear profile: edge %d trim radius %v differs from root radius %v",
				i, math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if want := 2 * p.Teeth; len(roots) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded gear profile: got %d root contacts, expected %d",
			len(roots), want)
	}
	return s, profile, roots, nil
}

// Build lofts the gear between two identical profiles and applies the root
// fillet, bore and cap chamfer.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	if err := validate(p); err != nil {
		return nil, err
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: placement frame: %w", err)
	}
	d := derive(p)
	world := sketch.NewWorld()

	lowerPlane := world.XY()
	if p.Placement != nil {
		lowerPlane, err = world.CreatePlaneFromFrame(frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: lower plane: %w", err)
		}
	}
	upperFrame, err := shifted(frame, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper frame: %w", err)
	}
	upperPlane, err := world.CreatePlaneFromFrame(upperFrame)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper plane: %w", err)
	}
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lowerPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: lower profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upperPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper profile: %w", err)
	}
	body, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: loft: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: loft did not produce a solid body")
	}

	if rf := filletRadius(p, d); rf > 0 {
		body, err = filletRoots(ctx, body, frame, roots, rf)
		if err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		body, err = cutBore(ctx, doc, body, p)
		if err != nil {
			return nil, err
		}
	}
	if p.CapChamfer > 0 {
		body, err = ChamferOuterCaps(ctx, body, p.CapChamfer)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("spurgear: final body is not a solid")
	}
	return body, nil
}

// filletRoots fillets the axial edges that start at a saved root contact.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, w := range world {
			if start.Sub(w).Len() <= 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: root fillet: matched %d axial edges, expected %d",
			len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the circular bore tool sketch, starting margin below
// the lower cap along the placement normal.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: placement frame: %w", err)
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
	if err := solveVerify(ctx, s, "bore profile"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s, "bore profile")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore extrudes the bore tool through the gear and cuts it.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	const margin = 1.0
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D:   units.Millimeters(p.Thickness + 2*margin),
		Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from a gear
// body spanning the placement's lower cap through Thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	if body == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(p.BoreDiameter > 0) || !finite(p.BoreDiameter) {
		return nil, fmt.Errorf("spurgear: bore diameter %v must be positive", p.BoreDiameter)
	}
	return cutBore(ctx, doc, body, p)
}

// ChamferOuterCaps chamfers the outer loops of both caps by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if gear == nil {
		return nil, fmt.Errorf("spurgear: nil body")
	}
	if !(distance > 0) || !finite(distance) {
		return nil, fmt.Errorf("spurgear: chamfer distance %v must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}
