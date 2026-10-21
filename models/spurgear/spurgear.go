// Package spurgear builds a parametric straight involute spur gear.
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

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
)

// Params describes one spur gear. Zero fields are not defaults; use
// DefaultParams for the default gear. A nil Placement selects the world XY
// frame at the origin.
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

// RejectPlacement returns an error naming model when p has a Placement.
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

// validate applies the input checks shared by Build, BuildProfile, and
// BuildToothSection.
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
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: placement is not a valid frame")
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
		return fmt.Errorf("spurgear: bore diameter %g must be less than twice the root radius %g", p.BoreDiameter, d.root)
	}
	if rf := filletRadius(p, d); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("spurgear: root fillet radius %g is not finite", rf)
	}
	return nil
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch, stage string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("spurgear: %s: solve: %w", stage, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("spurgear: %s: verify: %w", stage, err)
	}
	return nil
}

// onlyProfile returns the single valid profile of s.
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

// toothPoints holds the sketch points of one tooth.
type toothPoints struct {
	right, left         []*sketch.Point
	rightRoot, leftRoot *sketch.Point
	rootR, rootL        point
}

// addFlankPoints creates the dimensioned flank points of one tooth.
func addFlankPoints(s *sketch.Sketch, left, right []point) toothPoints {
	var t toothPoints
	t.right = make([]*sketch.Point, len(right))
	t.left = make([]*sketch.Point, len(left))
	for i, q := range right {
		t.right[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == len(left)-1 {
			t.left[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			t.left[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return t
}

// addRootPoints creates the root contacts by scaling the base samples. The
// left root is created before the right root.
func addRootPoints(s *sketch.Sketch, t *toothPoints, left, right []point, d dimensions) {
	t.rootL = point{left[0].x * d.root / d.base, left[0].y * d.root / d.base}
	t.rootR = point{right[0].x * d.root / d.base, right[0].y * d.root / d.base}
	t.leftRoot = sketchcheck.DimensionArcEnd(s, t.rootL.x, t.rootL.y)
	t.rightRoot = sketchcheck.DimensionPoint(s, t.rootR.x, t.rootR.y)
}

// BuildProfile builds the unfilleted planar gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, err
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, prof, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, prof, err
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	switch {
	case ctx == nil:
		return errors.New("spurgear: nil context")
	case world == nil:
		return errors.New("spurgear: nil world")
	case plane == nil:
		return errors.New("spurgear: nil plane")
	}
	return nil
}

// buildGearProfile builds the whole-gear outline and returns the actual root
// contact coordinates, right root then left root for each tooth.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}

	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: ordinary profile: create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth
	roots := make([]point, 0, 2*n)
	var prevLeftRoot, firstRightRoot *sketch.Point

	for k := 0; k < n; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(n), p.Samples, 2*math.Pi*float64(k)/float64(n))
		t := addFlankPoints(s, left, right)
		addRootPoints(s, &t, left, right, d)
		roots = append(roots, t.rootR, t.rootL)

		if k == 0 {
			firstRightRoot = t.rightRoot
		} else {
			s.CreateArc(center, prevLeftRoot, t.rightRoot)
		}
		s.CreateLine(t.rightRoot, t.right[0])
		if _, err := s.CreateFitSpline(t.right...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: ordinary profile: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, t.right[p.Samples-1], t.left[p.Samples-1])
		if _, err := s.CreateFitSpline(t.left...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: ordinary profile: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(t.left[0], t.leftRoot)
		prevLeftRoot = t.leftRoot
	}
	s.CreateArc(center, prevLeftRoot, firstRightRoot)

	if err := solveAndVerify(ctx, s, "ordinary profile"); err != nil {
		return nil, nil, nil, err
	}
	prof, err := onlyProfile(s, "ordinary profile")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, prof, roots, nil
}

// flankRef identifies a drawn flank spline.
type flankRef struct {
	tooth int
	right bool
}

// buildEmbeddedProfile builds the outline for Rb < Rr: the original flanks and
// tip arcs plus a root circle, unioned into one region.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth
	splines := make(map[*sketch.FitSpline]flankRef, 2*n)

	for k := 0; k < n; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(n), p.Samples, 2*math.Pi*float64(k)/float64(n))
		t := addFlankPoints(s, left, right)
		rs, err := s.CreateFitSpline(t.right...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: tooth %d right flank: %w", k, err)
		}
		splines[rs] = flankRef{k, true}
		s.CreateArc(center, t.right[p.Samples-1], t.left[p.Samples-1])
		ls, err := s.CreateFitSpline(t.left...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: tooth %d left flank: %w", k, err)
		}
		splines[ls] = flankRef{k, false}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s, "embedded profile"); err != nil {
		return nil, nil, nil, err
	}

	var indices []int
	for i, region := range s.Profiles() {
		if region.Valid {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 {
		return nil, nil, nil, errors.New("spurgear: embedded profile: no valid regions")
	}
	prof, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: union regions: %w", err)
	}
	if !prof.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded profile: union is not valid")
	}
	if len(prof.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: union has %d holes", len(prof.Holes))
	}
	if got, want := len(prof.Outer), 4*n; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: got %d boundary curves, expected %d", got, want)
	}

	contacts := make([]point, 2*n)
	seen := make([]bool, 2*n)
	count := 0
	for _, edge := range prof.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		ref, ok := splines[spline]
		if !ok {
			return nil, nil, nil, errors.New("spurgear: embedded profile: boundary spline is not a drawn flank")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf(
				"spurgear: embedded profile: tooth %d spline trim is not certified (partial=%t exact=%t)",
				ref.tooth, edge.Partial, edge.TExact)
		}
		x, y := spline.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf(
				"spurgear: embedded profile: tooth %d root contact radius %g differs from root radius %g",
				ref.tooth, math.Hypot(x, y), d.root)
		}
		idx := 2 * ref.tooth
		if !ref.right {
			idx++
		}
		if seen[idx] {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: tooth %d flank appears twice", ref.tooth)
		}
		seen[idx] = true
		contacts[idx] = point{x, y}
		count++
	}
	if count != 2*n {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: got %d root contacts, expected %d", count, 2*n)
	}
	return s, prof, contacts, nil
}

// Build constructs the gear body in doc.
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

	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lower, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: lower profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upper, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: upper profile: %w", err)
	}

	body, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: loft: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: loft did not produce a solid")
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
	return body, nil
}

// filletRoots fillets the axial edges that start at the saved root contacts.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			w := toWorld(frame, root)
			if math.Abs(start.X-w.X) <= 1e-8 && math.Abs(start.Y-w.Y) <= 1e-8 &&
				math.Abs(start.Z-w.Z) <= 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: root fillet: matched %d root edges, expected %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the circular bore tool sketch on a plane margin
// below the gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: nil world")
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s, "bore tool"); err != nil {
		return nil, nil, err
	}
	prof, err := onlyProfile(s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, prof, nil
}

// CutThroughBore cuts a centered cylindrical bore of p.BoreDiameter through
// body, which must span z=0 through p.Thickness in the placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: bore: nil context")
	case doc == nil:
		return nil, errors.New("spurgear: bore: nil document")
	case body == nil:
		return nil, errors.New("spurgear: bore: nil body")
	case !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0):
		return nil, fmt.Errorf("spurgear: bore: diameter %g must be positive", p.BoreDiameter)
	}
	const margin = 1.0
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D:   units.Millimeters(p.Thickness + 2*margin),
		Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: chamfer: nil context")
	case gear == nil:
		return nil, errors.New("spurgear: chamfer: nil body")
	case !(distance > 0) || math.IsInf(distance, 0):
		return nil, fmt.Errorf("spurgear: chamfer: distance %g must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer: %w", err)
	}
	return out, nil
}
