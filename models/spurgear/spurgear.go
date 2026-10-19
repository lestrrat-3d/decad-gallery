// Package spurgear builds a cylindrical straight-tooth involute spur gear.
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
	// Placement is the reference point (origin) and reference plane (U, V).
	// Nil selects the world XY frame at the origin.
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

// validate applies the input checks shared by Build, BuildProfile and
// BuildToothSection.
func validate(p Params) error {
	switch {
	case p.Module <= 0:
		return fmt.Errorf("module %g must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("thickness %g must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("pressure angle %g must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("bore diameter %g must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("cap chamfer %g must not be negative", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("invalid placement frame")
	}
	d := derive(p)
	switch {
	case d.root <= 0:
		return fmt.Errorf("root radius %g must be positive", d.root)
	case d.base <= 0:
		return fmt.Errorf("base radius %g must be positive", d.base)
	case d.tip <= d.base:
		return fmt.Errorf("tip radius %g must exceed base radius %g", d.tip, d.base)
	case p.BoreDiameter >= 2*d.root:
		return fmt.Errorf("bore diameter %g must be less than root diameter %g", p.BoreDiameter, 2*d.root)
	}
	if rf := filletRadius(p, d); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("root fillet radius %g is not finite", rf)
	}
	return nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	switch {
	case ctx == nil:
		return errors.New("nil context")
	case world == nil:
		return errors.New("nil world")
	case plane == nil:
		return errors.New("nil plane")
	}
	return nil
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// singleProfile returns the only profile of s, which must be valid.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profiles: got %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if profile == nil || !profile.Valid {
		return nil, errors.New("profile is not valid")
	}
	return profile, nil
}

// flankPoints creates the dimensioned flank sample points. Every right point
// receives both dimensions. Every left point does too, except the tip, which
// receives one.
func flankPoints(s *sketch.Sketch, left, right [][2]float64) (rightPts, leftPts []*sketch.Point) {
	rightPts = make([]*sketch.Point, len(right))
	for i, c := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, c[0], c[1])
	}
	leftPts = make([]*sketch.Point, len(left))
	for i, c := range left {
		if i == len(left)-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, c[0], c[1])
		} else {
			leftPts[i] = sketchcheck.DimensionPoint(s, c[0], c[1])
		}
	}
	return rightPts, leftPts
}

// rootPoints creates the left then right root points by scaling the flank base
// samples out to the root radius (multiply, then divide).
func rootPoints(s *sketch.Sketch, left, right [][2]float64, d dimensions) (
	leftRoot, rightRoot *sketch.Point, leftXY, rightXY point) {
	leftXY = point{left[0][0] * d.root / d.base, left[0][1] * d.root / d.base}
	rightXY = point{right[0][0] * d.root / d.base, right[0][1] * d.root / d.base}
	leftRoot = sketchcheck.DimensionArcEnd(s, leftXY.x, leftXY.y)
	rightRoot = sketchcheck.DimensionPoint(s, rightXY.x, rightXY.y)
	return leftRoot, rightRoot, leftXY, rightXY
}

// BuildProfile builds the unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileGuards(ctx, world, plane); err != nil {
		return nil, nil, fmt.Errorf("spurgear: profile: %w", err)
	}
	if err := validate(p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: profile: %w", err)
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// buildGearProfile draws the complete outline and returns the actual root
// contact coordinates.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile: create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := range p.Teeth {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		rightPts, leftPts := flankPoints(s, left, right)
		leftRoot, rightRoot, leftXY, rightXY := rootPoints(s, left, right, d)
		roots = append(roots, rightXY, leftXY)

		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear profile: tooth %d right spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear profile: tooth %d left spline: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile: %w", err)
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile: holes: got %d, want 0", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: gear profile: outer curves: got %d, want %d", got, want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile draws every original flank and tip arc plus a root
// circle, then unions the bounded regions into one outline.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: create sketch: %w", err)
	}
	center := s.Origin()
	for k := range p.Teeth {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		rightPts, leftPts := flankPoints(s, left, right)
		_, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: tooth %d right spline: %w", k, err)
		}
		s.CreateArc(center, rightPts[len(rightPts)-1], leftPts[len(leftPts)-1])
		_, err = s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: tooth %d left spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, 0, len(regions))
	for i, r := range regions {
		if r == nil || !r.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded profile: union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: holes: got %d, want 0", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: boundary curves: got %d, want %d", got, want)
	}

	roots := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		spline, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf(
				"spurgear: embedded profile: boundary edge %d spline trim is not certified (partial=%t exact=%t)",
				i, edge.Partial, edge.TExact)
		}
		x, y := spline.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf(
				"spurgear: embedded profile: boundary edge %d root contact radius %g, want %g",
				i, math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if got, want := len(roots), 2*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: root contacts: got %d, want %d", got, want)
	}
	return s, profile, roots, nil
}

// Build lofts the gear into doc and applies the root fillet, bore and cap
// chamfer in that order.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: build: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: build: nil document")
	}
	if err := validate(p); err != nil {
		return nil, fmt.Errorf("spurgear: build: %w", err)
	}
	d := derive(p)
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: placement: %w", err)
	}

	world := sketch.NewWorld()
	var lowerPlane *sketch.Plane
	if p.Placement == nil {
		lowerPlane = world.XY()
	} else if lowerPlane, err = world.CreatePlaneFromFrame(frame); err != nil {
		return nil, fmt.Errorf("spurgear: build: lower plane: %w", err)
	}
	upperFrame, err := shifted(frame, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: upper frame: %w", err)
	}
	upperPlane, err := world.CreatePlaneFromFrame(upperFrame)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: upper plane: %w", err)
	}

	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lowerPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: lower profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upperPlane, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: upper profile: %w", err)
	}
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: loft: %w", err)
	}

	if rf := filletRadius(p, d); rf > 0 {
		if body, err = filletRoots(ctx, body, frame, roots, rf); err != nil {
			return nil, fmt.Errorf("spurgear: build: root fillet: %w", err)
		}
	}
	if p.BoreDiameter > 0 {
		if body, err = cutBore(ctx, doc, body, p); err != nil {
			return nil, fmt.Errorf("spurgear: build: bore: %w", err)
		}
	}
	if p.CapChamfer > 0 {
		if body, err = ChamferOuterCaps(ctx, body, p.CapChamfer); err != nil {
			return nil, fmt.Errorf("spurgear: build: cap chamfer: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: build: result is not a solid")
	}
	return body, nil
}

// filletRoots fillets the axial edges that start at the saved root contacts.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
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
		return nil, fmt.Errorf("root edges: matched %d, want %d", len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("fillet: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the
// placed lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: %w", err)
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the placed lower cap through Thickness.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: bore: nil context")
	case doc == nil:
		return nil, errors.New("spurgear: bore: nil document")
	case body == nil:
		return nil, errors.New("spurgear: bore: nil body")
	case !(p.BoreDiameter > 0):
		return nil, fmt.Errorf("spurgear: bore: diameter %g must be positive", p.BoreDiameter)
	case !(p.Thickness > 0):
		return nil, fmt.Errorf("spurgear: bore: thickness %g must be positive", p.Thickness)
	}
	return cutBore(ctx, doc, body, p)
}

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
		return nil, fmt.Errorf("spurgear: bore: extrude tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: cut: %w", err)
	}
	return cut, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	switch {
	case ctx == nil:
		return nil, errors.New("spurgear: chamfer: nil context")
	case gear == nil:
		return nil, errors.New("spurgear: chamfer: nil body")
	case !(distance > 0) || math.IsInf(distance, 0):
		return nil, fmt.Errorf("spurgear: chamfer: distance %g must be positive and finite", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer: %w", err)
	}
	return chamfered, nil
}
