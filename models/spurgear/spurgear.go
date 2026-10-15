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

// Params describes one spur gear. Lengths are millimeters; angles are radians.
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

// RejectPlacement returns an error naming model when Placement is set.
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
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spurgear: module %g must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spurgear: thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spurgear: pressure angle %g must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %g must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: invalid placement frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return fmt.Errorf("spurgear: root radius %g must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("spurgear: base radius %g must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("spurgear: tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("spurgear: bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*d.root)
	}
	if rf := filletRadius(p, d); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("spurgear: root fillet radius %g is not finite", rf)
	}
	return nil
}

// solveAndVerify solves a sketch and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: solve %s: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: verify %s: %w", what, err)
	}
	return nil
}

// singleProfile requires exactly one valid profile without holes.
func singleProfile(s *sketch.Sketch, what string) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("spurgear: %s: got %d profiles, expected 1", what, len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid {
		return nil, fmt.Errorf("spurgear: %s: profile is invalid", what)
	}
	return profile, nil
}

// flankPoints creates the dimensioned sample points of one tooth. Every right
// point receives both coordinate dimensions; the left tip receives one.
func flankPoints(s *sketch.Sketch, left, right []point) ([]*sketch.Point, []*sketch.Point) {
	rightPts := make([]*sketch.Point, len(right))
	for i, pt := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	leftPts := make([]*sketch.Point, len(left))
	for i, pt := range left {
		if i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, pt.x, pt.y)
			continue
		}
		leftPts[i] = sketchcheck.DimensionPoint(s, pt.x, pt.y)
	}
	return rightPts, leftPts
}

// rootPoints creates the left then right root points by scaling the base samples.
func rootPoints(s *sketch.Sketch, d dimensions, leftBase, rightBase point) (
	*sketch.Point, *sketch.Point, point, point) {
	leftXY := point{leftBase.x * d.root / d.base, leftBase.y * d.root / d.base}
	rightXY := point{rightBase.x * d.root / d.base, rightBase.y * d.root / d.base}
	leftRoot := sketchcheck.DimensionArcEnd(s, leftXY.x, leftXY.y)
	rightRoot := sketchcheck.DimensionPoint(s, rightXY.x, rightXY.y)
	return leftRoot, rightRoot, leftXY, rightXY
}

// toothCurves draws the flank splines and tip arc of one tooth.
func toothCurves(s *sketch.Sketch, tooth, samples int, rightPts, leftPts []*sketch.Point) error {
	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return fmt.Errorf("spurgear: tooth %d right flank: %w", tooth, err)
	}
	s.CreateArc(s.Origin(), rightPts[samples-1], leftPts[samples-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return fmt.Errorf("spurgear: tooth %d left flank: %w", tooth, err)
	}
	return nil
}

// buildGearProfile draws the complete unfilleted outline on plane. It returns
// the actual root contact coordinates (right then left per tooth).
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeftRoot, firstRightRoot *sketch.Point
	for k := range p.Teeth {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		rightPts, leftPts := flankPoints(s, left, right)
		leftRoot, rightRoot, leftXY, rightXY := rootPoints(s, d, left[0], right[0])
		roots = append(roots, rightXY, leftXY)

		if k == 0 {
			firstRightRoot = rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if err := toothCurves(s, k, p.Samples, rightPts, leftPts); err != nil {
			return nil, nil, nil, err
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeftRoot = leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)

	if err := solveAndVerify(ctx, s, "gear profile"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "gear profile")
	if err != nil {
		return nil, nil, nil, err
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile has %d holes, expected 0", len(profile.Holes))
	}
	if want := 6 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile has %d curves, expected %d",
			len(profile.Outer), want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile draws the original flanks and tip arcs, adds a root
// circle, and unions every bounded region.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create sketch: %w", err)
	}
	for k := range p.Teeth {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		rightPts, leftPts := flankPoints(s, left, right)
		if err := toothCurves(s, k, p.Samples, rightPts, leftPts); err != nil {
			return nil, nil, nil, err
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded gear profile"); err != nil {
		return nil, nil, nil, err
	}

	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	if len(indices) == 0 {
		return nil, nil, nil, errors.New("spurgear: embedded profile has no regions")
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded union profile is invalid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile has %d holes, expected 0",
			len(profile.Holes))
	}
	if want := 4 * p.Teeth; len(profile.Outer) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile has %d curves, expected %d",
			len(profile.Outer), want)
	}

	roots := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded spline edge %d lacks a certified trim", i)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded spline edge %d trim radius %g, expected %g",
				i, math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if want := 2 * p.Teeth; len(roots) != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile has %d root contacts, expected %d",
			len(roots), want)
	}
	return s, profile, roots, nil
}

// BuildProfile builds the unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: context, world and plane must not be nil")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// Build lofts the gear into doc and applies fillets, bore and chamfer.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("spurgear: context and document must not be nil")
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
	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lower, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: lower profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upper, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper profile: %w", err)
	}
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: loft: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: lofted gear is not a solid")
	}

	if radius := filletRadius(p, d); radius > 0 {
		if body, err = filletRoots(ctx, body, frame, roots, radius); err != nil {
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
		return nil, errors.New("spurgear: gear is not a solid")
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
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if toWorld(frame, root).Sub(start).Len() <= 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet root edges: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the circular bore tool sketch on a plane offset by
// -margin along the placement normal.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: placement: %w", err)
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
		return nil, nil, fmt.Errorf("spurgear: create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s, "bore profile"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "bore profile")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from a gear
// body spanning the placement's lower plane through Thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, errors.New("spurgear: context, document and body must not be nil")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore diameter %g must be positive", p.BoreDiameter)
	}
	const margin = 1.0
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops with the given setback.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, errors.New("spurgear: context and gear must not be nil")
	}
	if !(distance > 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance %g must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}
