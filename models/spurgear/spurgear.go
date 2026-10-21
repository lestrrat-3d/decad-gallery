// Package spurgear builds a cylindrical straight involute spur gear.
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

// Params describes a spur gear. Lengths are millimeters and angles radians.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	// Placement gives the reference point (origin) and plane (U, V). Nil
	// selects the world XY frame at the origin.
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

// RejectPlacement returns an error naming model when a placement is set.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

func isFinite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// validate applies the input checks shared by Build, BuildProfile, and
// BuildToothSection.
func validate(p Params) error {
	switch {
	case p.Module <= 0:
		return fmt.Errorf("spurgear: module must be positive, got %v", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spurgear: thickness must be positive, got %v", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spurgear: pressure angle must be in (0, pi/2), got %v", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter must not be negative, got %v", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer must not be negative, got %v", p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: invalid placement frame")
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
		return fmt.Errorf("spurgear: bore diameter %v must be less than root diameter %v",
			p.BoreDiameter, 2*d.root)
	}
	if rf := filletRadius(p); !isFinite(rf) {
		return fmt.Errorf("spurgear: root fillet radius is not finite: %v", rf)
	}
	return nil
}

// placement returns the placement frame, defaulting to the world XY frame.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted returns f translated by dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a frame-plane coordinate to world space on f's plane.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// solveAndVerify solves s and requires a fully constrained verification.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify: %w", err)
	}
	return nil
}

// onlyValidProfile returns the sketch's single profile, requiring it be valid.
func onlyValidProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("sketch has %d profiles, expected 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("sketch profile is not valid")
	}
	return profiles[0], nil
}

// flankSketchPoints creates dimensioned sample points for both flanks. Every
// right point and every left point except the left tip receives both
// coordinate dimensions; the left tip receives one.
func flankSketchPoints(s *sketch.Sketch, left, right []point) (rp, lp []*sketch.Point) {
	rp = make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp = make([]*sketch.Point, len(left))
	last := len(left) - 1
	for i, q := range left {
		if i == last {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return rp, lp
}

// rootSketchPoints creates the radial-line root points by scaling each base
// sample to the root radius. The left root is created first.
func rootSketchPoints(s *sketch.Sketch, d dimensions, leftBase, rightBase point) (
	rootR, rootL *sketch.Point, rr, lr point) {
	lr = point{leftBase.x * d.root / d.base, leftBase.y * d.root / d.base}
	rr = point{rightBase.x * d.root / d.base, rightBase.y * d.root / d.base}
	rootL = sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
	rootR = sketchcheck.DimensionPoint(s, rr.x, rr.y)
	return rootR, rootL, rr, lr
}

// BuildProfile builds the unfilleted whole-gear outline on plane. Placement is
// not applied; plane locates the sketch.
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

// buildGearProfile builds the ordinary or embedded outline and returns the
// actual root contact coordinates.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: profile sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		rp, lp := flankSketchPoints(s, left, right)
		rootR, rootL, rr, lr := rootSketchPoints(s, d, left[0], right[0])
		roots = append(roots, rr, lr)

		if k == 0 {
			firstRight = rootR
		} else {
			s.CreateArc(center, prevLeft, rootR)
		}
		s.CreateLine(rootR, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], rootL)
		prevLeft = rootL
	}
	s.CreateArc(center, prevLeft, firstRight)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: ordinary profile: %w", err)
	}
	profile, err := onlyValidProfile(s)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: ordinary profile: %w", err)
	}
	if want, got := 6*p.Teeth, len(profile.Outer); got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: ordinary profile has %d outer curves, expected %d", got, want)
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile builds the outline for Rb < Rr by drawing the original
// flanks and a root circle, then uniting the bounded regions.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile sketch: %w", err)
	}
	center := s.Origin()
	for k := 0; k < p.Teeth; k++ {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		rp, lp := flankSketchPoints(s, left, right)
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[len(rp)-1], lp[len(lp)-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile: %w", err)
	}
	all := s.Profiles()
	indices := make([]int, 0, len(all))
	for i, pr := range all {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded profile region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile union: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded union profile has %d holes", len(profile.Holes))
	}
	if want, got := 4*p.Teeth, len(profile.Outer); got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile has %d outer curves, expected %d", got, want)
	}
	var contacts []point
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, errors.New("spurgear: embedded flank trim is not certified")
		}
		x, y := flank.Eval(edge.TStart)
		if r := math.Hypot(x, y); math.Abs(r-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded root contact radius %v, expected %v", r, d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if want, got := 2*p.Teeth, len(contacts); got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded profile has %d root contacts, expected %d", got, want)
	}
	return s, profile, contacts, nil
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
	lowerPlane := world.XY()
	if p.Placement != nil {
		if lowerPlane, err = world.CreatePlaneFromFrame(frame); err != nil {
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
		return nil, errors.New("spurgear: lofted body is not a solid")
	}
	if rf := filletRadius(p); rf > 0 {
		if body, err = filletRoots(ctx, body, frame, roots, rf); err != nil {
			return nil, err
		}
	}
	if p.BoreDiameter > 0 {
		if body, err = cutBore(ctx, doc, body, p); err != nil {
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

// filletRoots fillets the axial edges that start at a saved root contact.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: select axial edges: %w", err)
	}
	const tol = 1e-8
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if start.Sub(toWorld(frame, root)).Len() <= tol {
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
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the sketch of the centered bore tool on a plane
// offset margin below the gear's reference plane.
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
		return nil, nil, fmt.Errorf("spurgear: bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: %w", err)
	}
	profile, err := onlyValidProfile(s)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore profile: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span the reference plane through Thickness above it.
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
	if !(p.BoreDiameter > 0) || !isFinite(p.BoreDiameter) {
		return nil, fmt.Errorf("spurgear: bore diameter must be positive, got %v", p.BoreDiameter)
	}
	return cutBore(ctx, doc, body, p)
}

func cutBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	const margin = 1.0
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin)
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D:   units.Millimeters(p.Thickness + 2*margin),
		Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore tool extrude: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers the outer loop of both cap faces of gear by
// distance, leaving any bore rims sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: nil body")
	}
	if !(distance > 0) || !isFinite(distance) {
		return nil, fmt.Errorf("spurgear: chamfer distance must be positive, got %v", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return out, nil
}
