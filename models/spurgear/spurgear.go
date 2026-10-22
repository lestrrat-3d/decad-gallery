// Package spurgear builds a straight involute spur gear from Decad operations.
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

// validate applies the contract's input checks and limits.
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
	if rf := filletRadius(p); math.IsNaN(rf) || math.IsInf(rf, 0) {
		return fmt.Errorf("root fillet radius %g is not finite", rf)
	}
	return nil
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
		return nil, fmt.Errorf("spurgear: build: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: placement frame: %w", err)
	}
	d := derive(p)

	world := sketch.NewWorld()
	lowerPlane := world.XY()
	if p.Placement != nil {
		lowerPlane, err = world.CreatePlaneFromFrame(frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: build: lower plane: %w", err)
		}
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
	body, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("spurgear: build: loft: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: build: loft did not produce a solid")
	}

	if filletRadius(p) > 0 {
		body, err = filletRoots(ctx, body, frame, roots, filletRadius(p))
		if err != nil {
			return nil, fmt.Errorf("spurgear: build: %w", err)
		}
	}
	if p.BoreDiameter > 0 {
		body, err = CutThroughBore(ctx, doc, body, p)
		if err != nil {
			return nil, fmt.Errorf("spurgear: build: %w", err)
		}
	}
	if p.CapChamfer > 0 {
		body, err = ChamferOuterCaps(ctx, body, p.CapChamfer)
		if err != nil {
			return nil, fmt.Errorf("spurgear: build: %w", err)
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: build: final body is not a solid")
	}
	return body, nil
}

// BuildProfile builds the unfilleted planar gear outline on plane.
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
		return nil, nil, fmt.Errorf("spurgear: profile: %w", err)
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: profile: %w", err)
	}
	return s, profile, nil
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
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth
	roots := make([]point, 0, 2*n)
	var firstRight, prevLeft, lastLeft *sketch.Point
	for k := range n {
		angle := 2 * math.Pi * float64(k) / float64(n)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(n), p.Samples, angle)
		rp, lp := dimensionFlanks(s, left, right)
		leftRoot, rightRoot := rootPoints(s, d, left[0], right[0])
		roots = append(roots,
			point{right[0].x * d.root / d.base, right[0].y * d.root / d.base},
			point{left[0].x * d.root / d.base, left[0].y * d.root / d.base})
		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rp[0])
		if _, err := s.CreateFitSpline(rp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		if _, err := s.CreateFitSpline(lp...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		s.CreateLine(lp[0], leftRoot)
		prevLeft = leftRoot
		lastLeft = leftRoot
	}
	s.CreateArc(center, lastLeft, firstRight)

	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedProfile builds the outline for gears whose base circle lies
// inside the root circle by unioning all regions around a root circle.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	center := s.Origin()
	n := p.Teeth
	flanks := make([]*sketch.FitSpline, 0, 2*n)
	for k := range n {
		angle := 2 * math.Pi * float64(k) / float64(n)
		left, right := sampleFlanks(d.pitch, d.base, d.tip, float64(n), p.Samples, angle)
		rp, lp := dimensionFlanks(s, left, right)
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank spline: %w", k, err)
		}
		s.CreateArc(center, rp[p.Samples-1], lp[p.Samples-1])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank spline: %w", k, err)
		}
		flanks = append(flanks, rs, ls)
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("solve: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("verify: %w", err)
	}
	var indices []int
	for i, pr := range s.Profiles() {
		if !pr.Valid {
			return nil, nil, nil, fmt.Errorf("region %d is invalid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union profiles: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("union profile is invalid")
	}
	if len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("union profile has %d holes, want 0", len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*n; got != want {
		return nil, nil, nil, fmt.Errorf("union profile has %d boundary curves, want %d", got, want)
	}
	known := make(map[*sketch.FitSpline]bool, len(flanks))
	for _, f := range flanks {
		known[f] = true
	}
	var roots []point
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !known[flank] {
			return nil, nil, nil, fmt.Errorf("boundary edge %d lies on an unknown spline", i)
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("boundary edge %d has no certified root trim (partial %t, exact %t)",
				i, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("boundary edge %d trims at radius %g, want %g",
				i, math.Hypot(x, y), d.root)
		}
		roots = append(roots, point{x, y})
	}
	if got, want := len(roots), 2*n; got != want {
		return nil, nil, nil, fmt.Errorf("got %d root contacts, want %d", got, want)
	}
	return s, profile, roots, nil
}

// filletRoots fillets the axial edges that start at a saved root contact.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial edges: %w", err)
	}
	var corners []r3.Vec
	for _, e := range edges {
		start := e.Start().Position().Value
		for _, root := range roots {
			w := toWorld(frame, root)
			if math.Abs(w.X-start.X) <= 1e-8 && math.Abs(w.Y-start.Y) <= 1e-8 &&
				math.Abs(w.Z-start.Z) <= 1e-8 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("root fillet: matched %d edges, want %d", len(corners), len(roots))
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

// buildBoreProfile builds the sketch of the through-bore tool, starting margin
// below the gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("placement frame: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	s.AddConstraint(sketch.NewDiameter(circle, p.BoreDiameter))
	profile, err := solveSingleProfile(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered cylindrical bore through a body spanning
// z=0 through Thickness.
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
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore: diameter %g must be positive", p.BoreDiameter)
	}
	const margin = 1.0
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: tool profile: %w", err)
	}
	tool, err := decad.Extrude(ctx, doc, s, profile,
		decad.Distance{D: units.Millimeters(p.Thickness + 2*margin), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: extrude tool: %w", err)
	}
	out, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore: cut: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops by distance.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: chamfer: nil body")
	}
	if !(distance > 0) {
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
