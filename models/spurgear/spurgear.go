package spurgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params specifies a straight involute gear in millimeters and radians.
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

// DefaultParams returns the 17-tooth, module-one gear parameters.
func DefaultParams() Params {
	return Params{Module: 1, Teeth: 17, PressureAngle: 20 * math.Pi / 180, Thickness: 10, Samples: 15}
}

// RejectPlacement returns an error naming model when a placement was supplied.
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
	if p.Module <= 0 || p.Thickness <= 0 || p.Teeth < 3 || p.Samples < 3 {
		return fmt.Errorf("module and thickness must be positive; teeth and samples must be at least 3")
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		return fmt.Errorf("pressure angle must lie strictly between 0 and pi/2")
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
		return fmt.Errorf("bore diameter must be smaller than the root diameter")
	}
	if math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0) {
		return fmt.Errorf("root fillet radius must be finite")
	}
	return nil
}

func defineParameters(t *param.Table, p Params) error {
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("define module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("define teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("define pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("define thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("define samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("define bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("define cap_chamfer: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("define tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("define base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("define involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("define fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("define bore_tool_length: %w", err)
	}
	return nil
}

// Build constructs a gear in doc. The context and document must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return (Variant{}).Build(ctx, doc, p)
}

// Build constructs the variant's tooth pattern, root disc, and finishing features.
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
		return nil, fmt.Errorf("root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("bind root disc radius: %w", err)
	}
	profile, err := solvedProfile(ctx, s, 1)
	if err != nil {
		return nil, fmt.Errorf("root disc profile: %w", err)
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
		return nil, fmt.Errorf("joined gear is not solid")
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
		distance, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("read cap_chamfer: %w", err)
		}
		gear, err = chamferOuterCaps(ctx, gear, distance)
		if err != nil {
			return nil, err
		}
	}
	if !gear.IsSolid() {
		return nil, fmt.Errorf("finished gear is not solid")
	}
	return gear, nil
}

func profileArguments(ctx context.Context, world *sketch.World, plane *sketch.Plane) error {
	if ctx == nil || world == nil || plane == nil {
		return fmt.Errorf("profile requires nonnil context, world, and plane")
	}
	return nil
}

func solveVerified(ctx context.Context, s *sketch.Sketch) error {
	if _, err := s.Solve(ctx); err != nil {
		return fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fmt.Errorf("verify sketch: %w", err)
	}
	return nil
}

func solvedProfile(ctx context.Context, s *sketch.Sketch, curves int) (*sketch.Profile, error) {
	if err := solveVerified(ctx, s); err != nil {
		return nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 {
		return nil, fmt.Errorf("profile count %d, expected 1", len(profiles))
	}
	profile := profiles[0]
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, fmt.Errorf("profile is invalid or has %d holes, expected 0", len(profile.Holes))
	}
	if len(profile.Outer) != curves {
		return nil, fmt.Errorf("profile outer curve count %d, expected %d", len(profile.Outer), curves)
	}
	return profile, nil
}

// BuildProfile constructs the unfilleted whole-gear outline on plane.
// The context, world, and plane must not be nil.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if err := profileArguments(ctx, world, plane); err != nil {
		return nil, nil, err
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
		return nil, nil, nil, fmt.Errorf("gear outline sketch: %w", err)
	}
	var firstRight, previousLeft *sketch.Point
	roots := make([]point, 0, 2*p.Teeth)
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		l, r := dimensionFlanks(s, left, right, false)
		lr, rr := rootContact(left[0], d), rootContact(right[0], d)
		lp := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rp := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		roots = append(roots, rr, lr)
		if k == 0 {
			firstRight = rp
		} else {
			s.CreateArc(s.Origin(), previousLeft, rp)
		}
		s.CreateLine(rp, r[0])
		if err := drawFlanks(s, l, r, k); err != nil {
			return nil, nil, nil, err
		}
		s.CreateLine(l[0], lp)
		previousLeft = lp
	}
	s.CreateArc(s.Origin(), previousLeft, firstRight)
	profile, err := solvedProfile(ctx, s, 6*p.Teeth)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gear outline: %w", err)
	}
	return s, profile, roots, nil
}

func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline sketch: %w", err)
	}
	for k := range p.Teeth {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), p.Samples, toothAngle(p, k))
		l, r := dimensionFlanks(s, left, right, false)
		if err := drawFlanks(s, l, r, k); err != nil {
			return nil, nil, nil, err
		}
	}
	circle := s.CreateCircle(s.Origin(), d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveVerified(ctx, s); err != nil {
		return nil, nil, nil, fmt.Errorf("embedded outline: %w", err)
	}
	regions := s.Profiles()
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("embedded outline region %d is invalid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("union embedded outline regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("embedded outline is invalid or has %d holes, expected 0", len(profile.Holes))
	}
	if len(profile.Outer) != 4*p.Teeth {
		return nil, nil, nil, fmt.Errorf("embedded outer curve count %d, expected %d", len(profile.Outer), 4*p.Teeth)
	}
	roots := make([]point, 0, 2*p.Teeth)
	for i, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("embedded flank edge %d lacks certified root-circle trimming", i)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("embedded flank edge %d trim is not on the root circle", i)
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

func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius units.Value) (*decad.Body, error) {
	axial, err := decad.Edges(decad.ParallelTo(frame.N())).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select axial root candidates: %w", err)
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range axial {
		start := edge.Start().Position().Value
		for _, root := range roots {
			if start.Sub(toWorld(frame, root)).Len() <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) || len(corners) == 0 {
		return nil, fmt.Errorf("root edge count %d, expected %d", len(corners), len(roots))
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

func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame, teeth int,
	radius units.Value) (*decad.Body, error) {
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
		return nil, nil, fmt.Errorf("bore sketch parameters: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, err
	}
	frame, err = shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("bind bore diameter: %w", err)
	}
	profile, err := solvedProfile(ctx, s, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("bore tool profile: %w", err)
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore along the placement normal.
// The context, document, and body must not be nil. The diameter must be positive.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, fmt.Errorf("bore requires nonnil context, document, and body")
	}
	if p.BoreDiameter <= 0 {
		return nil, fmt.Errorf("bore diameter must be positive")
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("bore parameters: %w", err)
	}
	return cutThroughBore(ctx, doc, body, p)
}

func cutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body, p Params) (*decad.Body, error) {
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

// ChamferOuterCaps chamfers the two outer cap loops, leaving bore rims sharp.
// The context and gear must not be nil. The distance must be positive.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, fmt.Errorf("cap chamfer requires nonnil context and gear")
	}
	if distance <= 0 {
		return nil, fmt.Errorf("cap chamfer distance must be positive")
	}
	return chamferOuterCaps(ctx, gear, units.Millimeters(distance))
}

func chamferOuterCaps(ctx context.Context, gear *decad.Body, distance units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(decad.OuterLoopOf(decad.CapEnd(gear)))
	result, err := decad.Chamfer(ctx, gear, selection, distance)
	if err != nil {
		return nil, fmt.Errorf("chamfer outer caps: %w", err)
	}
	return result, nil
}
