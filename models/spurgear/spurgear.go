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

// Params specifies a straight involute gear in millimeters and radians.
// A nil Placement selects the world XY plane at the origin.
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

// DefaultParams returns the standard 17-tooth gear inputs.
func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15}
}

// RejectPlacement prevents models without placement support from ignoring it.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		if !p.Placement.IsValid() {
			return r3.Frame{}, fmt.Errorf("invalid placement frame")
		}
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

func validate(p Params) error {
	if p.Module <= 0 {
		return fmt.Errorf("module must be positive")
	}
	if p.Teeth < 3 {
		return fmt.Errorf("teeth must be at least 3")
	}
	if p.Thickness <= 0 {
		return fmt.Errorf("thickness must be positive")
	}
	if p.Samples < 3 {
		return fmt.Errorf("samples must be at least 3")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle must be between zero and pi/2")
	}
	if p.BoreDiameter < 0 || p.CapChamfer < 0 {
		return fmt.Errorf("bore diameter and cap chamfer must be nonnegative")
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return fmt.Errorf("invalid placement frame")
	}
	d := derive(p)
	if d.root <= 0 || d.base <= 0 || d.tip <= d.base {
		return fmt.Errorf("invalid root, base, or tip radius")
	}
	if p.BoreDiameter >= 2*d.root {
		return fmt.Errorf("bore diameter must be smaller than root diameter")
	}
	if !finite(d.fillet) {
		return fmt.Errorf("fillet radius must be finite")
	}
	return nil
}

// Build constructs one solid gear. The context and document must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}

// Build constructs the gear using this variant's construction hooks.
// The context and document must not be nil.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if v.LocalBuild {
		return v.buildLocal(ctx, doc, p)
	}
	r, err := v.begin(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return v.assemble(ctx, r)
}

func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("build tooth zero: %w", err)
	}
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("create root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root disc radius: %w", err)
	}
	if err := solveVerified(ctx, s); err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	profile, err := oneProfile(s, 1)
	if err != nil {
		return nil, fmt.Errorf("root disc: %w", err)
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(), Axis: r.frame.N(), Count: r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("pattern tooth: %w", err)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("join root disc and teeth: %w", err)
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("joined gear is not one solid")
	}
	if v.RootFillet != nil {
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
	} else if r.d.fillet > 0 {
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
	}
	if err != nil {
		return nil, fmt.Errorf("fillet gear roots: %w", err)
	}
	if r.p.BoreDiameter > 0 {
		gear, err = cutThroughBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("read cap_chamfer: %w", err)
		}
		gear, err = chamferOuterCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("finished gear is not one solid")
	}
	return gear, nil
}

// BuildProfile returns the unfilleted whole-gear outline on plane.
// The context, world, and plane must not be nil.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, fmt.Errorf("gear profile requires context, world, and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	return s, profile, err
}

func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create gear profile sketch: %w", err)
	}
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, previousLeft *sketch.Point
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		l, r := flankPoints(s, left, right, false)
		lc, rc := rootContact(left[0], d), rootContact(right[0], d)
		lroot := sketchcheck.DimensionArcEnd(s, lc.x, lc.y)
		rroot := sketchcheck.DimensionPoint(s, rc.x, rc.y)
		roots = append(roots, rc, lc)
		if k == 0 {
			firstRight = rroot
		} else {
			s.CreateArc(s.Origin(), previousLeft, rroot)
		}
		s.CreateLine(rroot, r[0])
		if _, err := s.CreateFitSpline(r...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), r[len(r)-1], l[len(l)-1])
		if _, err := s.CreateFitSpline(l...); err != nil {
			return nil, nil, nil, fmt.Errorf("tooth %d left flank: %w", k, err)
		}
		s.CreateLine(l[0], lroot)
		previousLeft = lroot
	}
	s.CreateArc(s.Origin(), previousLeft, firstRight)
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	profile, err := oneProfile(s, 6*p.Teeth)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear profile: %w", err)
	}
	return s, profile, roots, nil
}

func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create embedded profile sketch: %w", err)
	}
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		l, r := flankPoints(s, left, right, false)
		if _, err := s.CreateFitSpline(r...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded tooth %d right flank: %w", k, err)
		}
		s.CreateArc(s.Origin(), r[len(r)-1], l[len(l)-1])
		if _, err := s.CreateFitSpline(l...); err != nil {
			return nil, nil, nil, fmt.Errorf("embedded tooth %d left flank: %w", k, err)
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded profile: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded profile region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded union must be valid and hole-free")
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded boundary curve count %d, expected %d", len(profile.Outer), 4*p.Teeth)
	}
	roots := make([]point, 0, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded flank lacks certified trim parameters")
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded flank trim misses root circle")
		}
		roots = append(roots, point{x, y})
	}
	if len(roots) != 2*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded root contact count %d, expected %d", len(roots), 2*p.Teeth)
	}
	return s, profile, roots, nil
}

func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
	edges, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial root edges: %w", err)
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range edges {
		start := edge.Start().Position().Value
		for _, root := range roots {
			delta := start.Sub(toWorld(frame, root))
			if math.Sqrt(delta.X*delta.X+delta.Y*delta.Y+delta.Z*delta.Z) <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("axial root edge count %d, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, fmt.Errorf("root fillet requires at least one root contact")
	}
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(frame.N()), decad.EndpointAt(corner))
	}
	result, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("fillet ordinary roots: %w", err)
	}
	return result, nil
}

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	result, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("fillet embedded roots: %w", err)
	}
	return result, nil
}

func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("define bore sketch parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, err
	}
	frame, err = shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("shift bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("create bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	profile, err := oneProfile(s, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("bore sketch: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore removes a centered through-bore along p's placement normal.
// The context, document, and body must not be nil. Diameter must be positive.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, fmt.Errorf("through-bore requires context, document, and body")
	}
	if p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("through-bore diameter must be positive")
	}
	return cutThroughBore(ctx, doc, body, p)
}

func cutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("define bore parameters: %w", err)
	}
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("read tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("read bore_tool_length: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude bore tool: %w", err)
	}
	result, err := decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("cut through-bore: %w", err)
	}
	return result, nil
}

// ChamferOuterCaps chamfers both outer cap loops and preserves bore rims.
// The context and gear must not be nil, and distance must be positive.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("cap chamfer requires context and body")
	}
	if distance <= 0 {
		return nil, fmt.Errorf("cap chamfer distance must be positive")
	}
	return chamferOuterCaps(ctx, gear, units.Millimeters(distance))
}

func chamferOuterCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	result, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return result, nil
}
