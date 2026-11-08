// Package spurgear builds a cylindrical gear with straight involute teeth.
package spurgear

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are in millimeters and angles in
// radians. Zero fields are not defaults; use DefaultParams for those.
type Params struct {
	Module        float64
	Teeth         int
	PressureAngle float64
	Thickness     float64
	Samples       int
	BoreDiameter  float64
	CapChamfer    float64
	Backlash      float64
	// Placement is the reference point and plane. Nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default 17-tooth, module 1 gear.
func DefaultParams() Params {
	return Params{
		Module:        1,
		Teeth:         17,
		PressureAngle: 20 * math.Pi / 180,
		Thickness:     10,
		Samples:       15,
		BoreDiameter:  0,
		CapChamfer:    0,
		Backlash:      0.05,
	}
}

// RejectPlacement returns an error naming model when p carries a placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// DefineParameters writes the gear's parameter table into t without
// validating p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// placement returns the placement frame, or the world XY frame.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted returns f moved dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a point of the frame's plane to world coordinates.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits shared by every entry point.
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
		return fmt.Errorf("spurgear: pressure angle %v must lie in (0, pi/2)", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter %v must not be negative", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer %v must not be negative", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0) || math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash %v must be finite and within module/2 (%v)", p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: placement is not a valid frame")
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
	case math.IsNaN(d.fillet) || math.IsInf(d.fillet, 0):
		return fmt.Errorf("spurgear: root fillet radius %v must be finite", d.fillet)
	}
	return nil
}

// Build constructs the gear in doc and returns its single solid body.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear described by v and p in doc.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("spurgear: build requires a context and a document")
	}
	if v.LocalBuild {
		return v.buildLocal(ctx, doc, p)
	}
	r, err := v.begin(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return v.assemble(ctx, r)
}

// assemble builds the root disc and the tooth, joins the patterned teeth,
// and applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth: %w", err)
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	body, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join teeth: %w", err)
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: joined gear is not a solid")
	}
	if v.RootFillet != nil {
		body, err = v.RootFillet(ctx, r.doc, body, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	} else if r.d.fillet > 0 {
		body, err = filletSpurRoots(ctx, r.doc, body, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	}
	if r.p.BoreDiameter > 0 {
		body, err = cutBore(ctx, r.doc, body, r.p)
		if err != nil {
			return nil, err
		}
	}
	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read cap chamfer: %w", err)
		}
		body, err = chamferCaps(ctx, body, setback)
		if err != nil {
			return nil, err
		}
	}
	if !body.IsSolid() {
		return nil, errors.New("spurgear: finished gear is not a solid")
	}
	return body, nil
}

// buildRootDisc extrudes the root-disc circle through the gear thickness.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: root disc radius: %w", err)
	}
	if err := solveAndVerify(ctx, s, "root disc"); err != nil {
		return nil, err
	}
	profile, err := singleProfile(s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude root disc: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the unfilleted outline of the whole gear on plane.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params) (*sketch.Sketch, *sketch.Profile, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, errors.New("spurgear: profile requires a context, world and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, derive(p))
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole-gear outline with every tooth turned by
// an extra twist about the sketch origin. It also returns the 2N root
// contacts: the right root then the left root of each tooth.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	if ctx == nil || world == nil || plane == nil {
		return nil, nil, nil, errors.New("spurgear: profile requires a context, world and plane")
	}
	if err := validate(p); err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twist %v must be finite", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, derive(p), twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, root := range roots {
		out[i] = [2]float64{root.x, root.y}
	}
	return s, profile, out, nil
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline of either branch with
// every tooth turned by an extra twist.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedOutline(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: outline sketch: %w", err)
	}
	center := s.Origin()
	roots := make([]point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := 0; k < p.Teeth; k++ {
		t := addToothPoints(s, p, d, toothAngle(p, k)+twist)
		roots = append(roots, t.rightRootXY, t.leftRootXY)
		if k == 0 {
			firstRight = t.rightRoot
		} else {
			s.CreateArc(center, prevLeft, t.rightRoot)
		}
		if _, _, err := drawToothSides(s, t, k); err != nil {
			return nil, nil, nil, err
		}
		prevLeft = t.leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)
	if err := solveAndVerify(ctx, s, "gear outline"); err != nil {
		return nil, nil, nil, err
	}
	profile, err := singleProfile(s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	return s, profile, roots, nil
}

// buildEmbeddedOutline draws the whole-gear outline when the base circle
// lies inside the root circle. Each flank crosses the root circle, so the
// outline is the union of the regions bounded by the flanks and the root
// circle, and the root contacts are the certified trim points.
func buildEmbeddedOutline(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline sketch: %w", err)
	}
	center := s.Origin()
	slot := make(map[*sketch.FitSpline]int, 2*p.Teeth)
	for k := 0; k < p.Teeth; k++ {
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash,
			p.Samples, toothAngle(p, k)+twist)
		rp, lp := dimensionFlanks(s, left, right, false)
		last := p.Samples - 1
		rs, err := s.CreateFitSpline(rp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d: right flank: %w", k, err)
		}
		s.CreateArc(center, rp[last], lp[last])
		ls, err := s.CreateFitSpline(lp...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: tooth %d: left flank: %w", k, err)
		}
		slot[rs] = 2 * k
		slot[ls] = 2*k + 1
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))
	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	var indices []int
	for i, region := range s.Profiles() {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline region %d is not valid", i)
		}
		indices = append(indices, i)
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: union embedded outline regions: %w", err)
	}
	if !profile.Valid || len(profile.Holes) != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline is not a hole-free valid profile (%d holes)",
			len(profile.Holes))
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline has %d curves, expected %d", got, want)
	}
	contacts := make([]point, 2*p.Teeth)
	seen := make([]bool, 2*p.Teeth)
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		idx, ok := slot[flank]
		if !ok {
			return nil, nil, nil, errors.New("spurgear: embedded outline holds an unknown spline")
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: flank %d root trim is not certified", idx)
		}
		x, y := flank.Eval(edge.TStart)
		if math.Abs(math.Hypot(x, y)-d.root) > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: flank %d trim lies at radius %v, expected %v",
				idx, math.Hypot(x, y), d.root)
		}
		contacts[idx] = point{x, y}
		seen[idx] = true
	}
	for idx, ok := range seen {
		if !ok {
			return nil, nil, nil, fmt.Errorf("spurgear: got %d root contacts, expected %d", idx, 2*p.Teeth)
		}
	}
	return s, profile, contacts, nil
}

// buildEmbeddedTooth draws one embedded tooth: the right flank, the tip arc,
// the left flank and the arc at the closing radius.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(d, p.Teeth)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth sketch: %w", err)
	}
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	rp, lp := dimensionFlanks(s, left, right, true)
	center := s.Origin()
	last := p.Samples - 1
	if _, err := s.CreateFitSpline(rp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth right flank: %w", err)
	}
	s.CreateArc(center, rp[last], lp[last])
	if _, err := s.CreateFitSpline(lp...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth left flank: %w", err)
	}
	s.CreateArc(center, rp[0], lp[0])
	if err := solveAndVerify(ctx, s, "embedded tooth"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if got := len(profile.Outer); got != 4 {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth has %d outer curves, expected 4", got)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges with a radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges that start at a saved root contact.
// Selecting every axial edge would also select the flank joints.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value) (*decad.Body, error) {
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
			if math.Abs(w.X-start.X) <= 1e-6 && math.Abs(w.Y-start.Y) <= 1e-6 && math.Abs(w.Z-start.Z) <= 1e-6 {
				corners = append(corners, start)
				break
			}
		}
	}
	if len(corners) != len(roots) {
		return nil, fmt.Errorf("spurgear: matched %d root edges, expected %d", len(corners), len(roots))
	}
	if len(corners) == 0 {
		return nil, errors.New("spurgear: no root edges to fillet")
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	out, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet %d root edges: %w", len(corners), err)
	}
	return out, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	out, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: fillet %d concave root edges: %w", 2*teeth, err)
	}
	return out, nil
}

// buildBoreProfile draws the circular bore tool section on a plane that lies
// margin below the placed gear's lower cap.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, err
	}
	base, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore placement: %w", err)
	}
	frame, err := shifted(base, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore frame: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore diameter: %w", err)
	}
	if err := solveAndVerify(ctx, s, "bore"); err != nil {
		return nil, nil, err
	}
	profile, err := singleProfile(s, "bore")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from a gear
// body that spans the lower cap through p.Thickness above it.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil || body == nil {
		return nil, errors.New("spurgear: bore requires a context, document and body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: bore diameter %v must be positive", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, err
	}
	return cutBore(ctx, doc, body, p)
}

// cutBore extrudes the bore tool and cuts it from the gear. The document's
// table must already hold the gear parameters.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read tool margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read bore tool length: %w", err)
	}
	s, profile, err := buildBoreProfile(ctx, sketch.NewWorld(), p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude bore tool: %w", err)
	}
	out, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cut bore: %w", err)
	}
	return out, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear by distance
// millimeters, leaving any bore rims sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil || gear == nil {
		return nil, errors.New("spurgear: chamfer requires a context and a body")
	}
	if !(distance > 0) || math.IsInf(distance, 0) {
		return nil, fmt.Errorf("spurgear: chamfer distance %v must be positive and finite", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}

// chamferCaps chamfers the outer loop of each cap.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	out, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: chamfer outer caps: %w", err)
	}
	return out, nil
}
