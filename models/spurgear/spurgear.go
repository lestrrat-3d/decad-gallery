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

// Params describes one spur gear. Lengths are millimeters, angles radians.
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

// RejectPlacement returns an error naming model when a placement is set.
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

func validate(p Params) error {
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("spurgear: invalid module %g, thickness %g, teeth %d, or samples %d",
			p.Module, p.Thickness, p.Teeth, p.Samples)
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("spurgear: pressure angle %g out of range", p.PressureAngle)
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("spurgear: negative bore diameter %g or chamfer %g", p.BoreDiameter, p.CapChamfer)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return errors.New("spurgear: invalid placement frame")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("spurgear: invalid radii root %g, base %g, tip %g", d.root, d.base, d.tip)
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("spurgear: bore diameter %g must be below root diameter %g", p.BoreDiameter, 2*d.root)
	}
	if !finite(filletRadius(p)) {
		return errors.New("spurgear: nonfinite root fillet radius")
	}
	return nil
}

func solveVerify(ctx context.Context, s *sketch.Sketch, what string) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("%s: verify: %w", what, err)
	}
	return nil
}

// BuildProfile builds the unfilleted outline sketch on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: nil context, world, or plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

// dimensionFlanks creates the dimensioned flank sample points.
func dimensionFlanks(s *sketch.Sketch, left, right []point) (lp, rp []*sketch.Point) {
	rp = make([]*sketch.Point, len(right))
	for i, q := range right {
		rp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	lp = make([]*sketch.Point, len(left))
	for i, q := range left {
		if i == len(left)-1 {
			lp[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
		} else {
			lp[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
		}
	}
	return lp, rp
}

func rootContact(d dimensions, q point) point {
	return point{q.x * d.root / d.base, q.y * d.root / d.base}
}

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
	n := p.Teeth
	roots := make([]point, 0, 2*n)
	var firstRight, prevLeft *sketch.Point
	for k := range n {
		theta := 2 * math.Pi * float64(k) / float64(n)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(n), p.Samples, theta)
		lp, rp := dimensionFlanks(s, left, right)
		lrc := rootContact(d, left[0])
		rrc := rootContact(d, right[0])
		leftRoot := sketchcheck.DimensionArcEnd(s, lrc.x, lrc.y)
		rightRoot := sketchcheck.DimensionPoint(s, rrc.x, rrc.y)
		roots = append(roots, rrc, lrc)
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(lp[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveVerify(ctx, s, "spurgear profile"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

func onlyProfile(s *sketch.Sketch) (*sketch.Profile, error) {
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("spurgear: expected exactly one valid profile, got %d", len(profiles))
	}
	return profiles[0], nil
}

func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth
	splines := map[sketch.Entity]bool{}
	for k := range n {
		theta := 2 * math.Pi * float64(k) / float64(n)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(n), p.Samples, theta)
		lp, rp := dimensionFlanks(s, left, right)
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d left flank: %w", k, err)
		}
		splines[rs] = true
		splines[ls] = true
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerify(ctx, s, "spurgear embedded profile"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 || len(profile.Outer) != 4*n {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline valid=%t holes=%d curves=%d, want %d curves",
			profile.Valid, len(profile.Holes), len(profile.Outer), 4*n)
	}
	var contacts []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded edge %d lacks certified trim (partial=%t exact=%t)",
				i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded edge %d trim radius %g, want %g",
				i, math.Hypot(x, y), d.root)
		}
		contacts = append(contacts, point{x, y})
	}
	if len(contacts) != 2*n {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded root contacts: got %d, want %d", len(contacts), 2*n)
	}
	return s, profile, contacts, nil
}

// Build lofts the complete gear into doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("spurgear: nil context or document")
	}
	if err := validate(p); err != nil {
		return nil, err
	}
	d := derive(p)
	world := sketch.NewWorld()
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: placement: %w", err)
	}
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
	body, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: loft: %w", err)
	}
	if radius := filletRadius(p); radius > 0 {
		body, err = filletRoots(ctx, body, frame, roots, radius)
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
		return nil, errors.New("spurgear: result is not a solid")
	}
	return body, nil
}

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
		return nil, fmt.Errorf("spurgear: root edge matches: got %d, want %d", len(corners), len(roots))
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
	if err := solveVerify(ctx, s, "spurgear bore"); err != nil {
		return nil, nil, err
	}
	profile, err := onlyProfile(s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore from a gear spanning the lower
// plane through Thickness above it.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, errors.New("spurgear: nil context, document, or body")
	}
	if !finite(p.BoreDiameter) || p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("spurgear: bore diameter %g must be positive", p.BoreDiameter)
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
	tool, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore tool: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore cut: %w", err)
	}
	return cut, nil
}

// ChamferOuterCaps chamfers both outer cap loops with the given setback.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, errors.New("spurgear: nil context or body")
	}
	if !finite(distance) || distance <= 0 {
		return nil, fmt.Errorf("spurgear: chamfer distance %g must be positive", distance)
	}
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, units.Millimeters(distance))
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return out, nil
}
