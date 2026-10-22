// Package spurgear builds a straight-toothed involute spur gear as a
// parametric Decad body.
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
	// Placement gives the reference point and plane; nil selects the world
	// XY frame at the origin.
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
		return fmt.Errorf("spurgear: module %v must be positive", p.Module)
	case p.Thickness <= 0:
		return fmt.Errorf("spurgear: thickness %v must be positive", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth %d must be at least 3", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples %d must be at least 3", p.Samples)
	case p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2:
		return fmt.Errorf("spurgear: pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	case p.Placement != nil && !p.Placement.IsValid():
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
	if rf := filletRadius(p); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("spurgear: root fillet radius %v is not finite", rf)
	}
	return nil
}

// solveAndVerify solves s and requires it to be fully constrained.
func solveAndVerify(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify constraints: %w", err)
	}
	return nil
}

// singleProfile returns the only profile of s, requiring it to be valid.
func singleProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("got %d profiles, want 1", len(profiles))
	}
	if !profiles[0].Valid {
		return nil, errors.New("profile is not valid")
	}
	return profiles[0], nil
}

// Build constructs the gear body in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: build: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: build: nil document")
	}
	if err := validate(p); err != nil {
		return nil, err
	}
	d := derive(p)
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: placement: %w", err)
	}

	world := sketch.NewWorld()
	var lower *sketch.Plane
	if p.Placement == nil {
		lower = world.XY()
	} else if lower, err = world.CreatePlaneFromFrame(frame); err != nil {
		return nil, fmt.Errorf("spurgear: build: lower plane: %w", err)
	}
	upperFrame, err := shifted(frame, p.Thickness)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: upper frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(upperFrame)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: upper plane: %w", err)
	}

	lowerSketch, lowerProfile, roots, err := buildGearProfile(ctx, world, lower, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: lower profile: %w", err)
	}
	upperSketch, upperProfile, _, err := buildGearProfile(ctx, world, upper, p, d)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: upper profile: %w", err)
	}
	body, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: loft: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: build: loft did not produce a solid")
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

// BuildProfile builds the unfilleted gear outline on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil {
		return nil, nil, errors.New("spurgear: profile: nil context")
	}
	if world == nil {
		return nil, nil, errors.New("spurgear: profile: nil world")
	}
	if plane == nil {
		return nil, nil, errors.New("spurgear: profile: nil plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// buildGearProfile draws, solves, and verifies the whole-gear outline and
// returns the actual root contact coordinates.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var prevLeft, firstRight *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		t, err := drawTooth(s, left, right, d, prevLeft)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: %w", k, err)
		}
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			firstRight = t.rightRoot
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile draws the flanks and tip arcs plus a full root circle
// and unions every bounded region into one outline.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	for k := 0; k < p.Teeth; k++ {
		theta := 2 * math.Pi * float64(k) / float64(p.Teeth)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(p.Teeth), p.Samples, theta)
		n := p.Samples
		rightPts := make([]*sketch.Point, n)
		leftPts := make([]*sketch.Point, n)
		for i := 0; i < n; i++ {
			rightPts[i] = sketchcheck.DimensionPoint(s, right[i].x, right[i].y)
		}
		for i := 0; i < n; i++ {
			if i == n-1 {
				leftPts[i] = sketchcheck.DimensionArcEnd(s, left[i].x, left[i].y)
			} else {
				leftPts[i] = sketchcheck.DimensionPoint(s, left[i].x, left[i].y)
			}
		}
		_, err = s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: right flank spline: %w", k, err)
		}
		s.CreateArc(s.Origin(), rightPts[n-1], leftPts[n-1])
		_, err = s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d: left flank spline: %w", k, err)
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("union profile is not valid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("union profile has %d holes, want 0", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("union profile has %d boundary curves, want %d", got, want)
	}

	var contacts []point
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, errors.New("flank trim at the root circle is not certified exact")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("flank trim radius %v differs from root radius %v",
				math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if got, want := len(contacts), 2*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("got %d root contacts, want %d", got, want)
	}
	return s, profile, contacts, nil
}

// filletRoots fillets the axial edges that start at the saved root contacts.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet: select axial edges: %w", err)
	}
	worldRoots := make([]r3.Vec, len(roots))
	for i, rt := range roots {
		worldRoots[i] = toWorld(frame, rt)
	}
	var corners []r3.Vec
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, wr := range worldRoots {
			if start.Sub(wr).Len() <= 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: fillet: matched %d root edges, want %d",
			len(corners), len(roots))
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, c := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(c))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), units.Millimeters(radius))
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet: %w", err)
	}
	return out, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from body,
// which must span z=0 through p.Thickness in the placement frame.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: bore: nil document")
	}
	if body == nil {
		return nil, errors.New("spurgear: bore: nil body")
	}
	if !(p.BoreDiameter > 0) || math.IsInf(p.BoreDiameter, 0) {
		return nil, fmt.Errorf("spurgear: bore: diameter %v must be positive", p.BoreDiameter)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, errors.New("spurgear: bore: invalid placement frame")
	}
	return cutBore(ctx, doc, body, p)
}

// boreMargin is how far the bore tool extends past each cap.
const boreMargin = 1.0

func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body,
	p Params) (*decad.Body, error) {
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, boreMargin)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{
		D:   units.Millimeters(p.Thickness + 2*boreMargin),
		Dir: decad.Along,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: cut: %w", err)
	}
	return out, nil
}

// buildBoreProfile builds the bore tool circle on a plane margin below the
// placement plane.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("placement: %w", err)
	}
	below, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(below)
	if err != nil {
		return nil, nil, fmt.Errorf("tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("tool sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	if err := solveAndVerify(ctx, s); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body,
	distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: chamfer: nil body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer: distance %v must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer: %w", err)
	}
	return out, nil
}
