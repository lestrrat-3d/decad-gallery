// Package spurgear builds a parametric cylindrical gear with straight
// involute teeth.
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
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Params describes one spur gear. Lengths are millimeters and angles radians.
type Params struct {
	// Module is the tooth module.
	Module float64
	// Teeth is the tooth count.
	Teeth int
	// PressureAngle is the pressure angle.
	PressureAngle float64
	// Thickness is the axial thickness.
	Thickness float64
	// Samples is the number of fit points per flank.
	Samples int
	// BoreDiameter is the centered through-bore diameter; zero omits the bore.
	BoreDiameter float64
	// CapChamfer is the equal-distance outer cap setback; zero omits it.
	CapChamfer float64
	// Backlash is the circumferential tooth thinning at the pitch circle.
	// Zero keeps the full half-pitch tooth; a negative value thickens it.
	Backlash float64
	// Placement is the reference point and plane. Nil selects the world XY
	// frame at the origin.
	Placement *r3.Frame
}

// DefaultParams returns the default spur gear.
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

// RejectPlacement returns an error naming model when p carries a Placement.
func (p Params) RejectPlacement(model string) error {
	if p.Placement != nil {
		return fmt.Errorf("%s does not support Placement", model)
	}
	return nil
}

// placement resolves the gear frame; nil selects the world XY frame.
func placement(p Params) (r3.Frame, error) {
	if p.Placement != nil {
		return *p.Placement, nil
	}
	return r3.NewFrame(r3.NewVec(0, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// shifted moves f by dist along its normal.
func shifted(f r3.Frame, dist float64) (r3.Frame, error) {
	return r3.NewFrame(f.Origin().Add(f.N().Scale(dist)), f.U(), f.V())
}

// toWorld maps a sketch coordinate on the frame's plane to world space.
func toWorld(f r3.Frame, xy point) r3.Vec {
	return f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))
}

// validate applies the input checks and limits shared by the gear builders.
func validate(p Params) error {
	switch {
	case !(p.Module > 0):
		return fmt.Errorf("spurgear: module must be positive, got %v", p.Module)
	case !(p.Thickness > 0):
		return fmt.Errorf("spurgear: thickness must be positive, got %v", p.Thickness)
	case p.Teeth < 3:
		return fmt.Errorf("spurgear: teeth must be at least 3, got %d", p.Teeth)
	case p.Samples < 3:
		return fmt.Errorf("spurgear: samples must be at least 3, got %d", p.Samples)
	case !(p.PressureAngle > 0) || !(p.PressureAngle < math.Pi/2):
		return fmt.Errorf("spurgear: pressure angle must lie in (0, pi/2), got %v", p.PressureAngle)
	case p.BoreDiameter < 0:
		return fmt.Errorf("spurgear: bore diameter must not be negative, got %v", p.BoreDiameter)
	case p.CapChamfer < 0:
		return fmt.Errorf("spurgear: cap chamfer must not be negative, got %v", p.CapChamfer)
	case math.IsNaN(p.Backlash) || math.IsInf(p.Backlash, 0):
		return fmt.Errorf("spurgear: backlash must be finite, got %v", p.Backlash)
	case math.Abs(p.Backlash) > p.Module/2:
		return fmt.Errorf("spurgear: backlash magnitude %v exceeds half the module %v", p.Backlash, p.Module/2)
	case p.Placement != nil && !p.Placement.IsValid():
		return errors.New("spurgear: placement is not a valid frame")
	}
	m := p.Module
	n := float64(p.Teeth)
	pitch := m * n / 2
	base := pitch * math.Cos(p.PressureAngle)
	root := (m*n - 2.5*m) / 2
	tip := (m*n + 2*m) / 2
	inv := math.Tan(p.PressureAngle) - p.PressureAngle
	fillet := 0.9 * root * (math.Pi/n - 2*inv) / 2
	switch {
	case root <= 0:
		return fmt.Errorf("spurgear: root radius %v must be positive", root)
	case base <= 0:
		return fmt.Errorf("spurgear: base radius %v must be positive", base)
	case tip <= base:
		return fmt.Errorf("spurgear: tip radius %v must exceed base radius %v", tip, base)
	case p.BoreDiameter >= 2*root:
		return fmt.Errorf("spurgear: bore diameter %v must be below the root diameter %v", p.BoreDiameter, 2*root)
	case math.IsNaN(fillet) || math.IsInf(fillet, 0):
		return fmt.Errorf("spurgear: fillet radius %v is not finite", fillet)
	}
	return nil
}

// defineParameters writes the spur parameter table of p into t.
func defineParameters(t *param.Table, p Params) error {
	if t == nil {
		return errors.New("spurgear: define parameters: nil table")
	}
	if err := t.SetValue("module", units.Millimeters(p.Module)); err != nil {
		return fmt.Errorf("spurgear: define parameter module: %w", err)
	}
	if err := t.SetNumber("teeth", float64(p.Teeth)); err != nil {
		return fmt.Errorf("spurgear: define parameter teeth: %w", err)
	}
	if err := t.SetValue("pressure_angle", units.Radians(p.PressureAngle)); err != nil {
		return fmt.Errorf("spurgear: define parameter pressure_angle: %w", err)
	}
	if err := t.SetValue("thickness", units.Millimeters(p.Thickness)); err != nil {
		return fmt.Errorf("spurgear: define parameter thickness: %w", err)
	}
	if err := t.SetNumber("samples", float64(p.Samples)); err != nil {
		return fmt.Errorf("spurgear: define parameter samples: %w", err)
	}
	if err := t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter)); err != nil {
		return fmt.Errorf("spurgear: define parameter bore_diameter: %w", err)
	}
	if err := t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer)); err != nil {
		return fmt.Errorf("spurgear: define parameter cap_chamfer: %w", err)
	}
	if err := t.SetValue("backlash", units.Millimeters(p.Backlash)); err != nil {
		return fmt.Errorf("spurgear: define parameter backlash: %w", err)
	}
	if err := t.SetExpr("tool_margin", "1", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter tool_margin: %w", err)
	}
	if err := t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter pitch_radius: %w", err)
	}
	if err := t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter base_radius: %w", err)
	}
	if err := t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter root_radius: %w", err)
	}
	if err := t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter tip_radius: %w", err)
	}
	if err := t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One); err != nil {
		return fmt.Errorf("spurgear: define parameter involute_angle: %w", err)
	}
	if err := t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter fillet_radius: %w", err)
	}
	if err := t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter); err != nil {
		return fmt.Errorf("spurgear: define parameter bore_tool_length: %w", err)
	}
	return nil
}

// DefineParameters writes the spur parameter table of p into t. It does not
// validate p.
func DefineParameters(t *param.Table, p Params) error {
	return defineParameters(t, p)
}

// Build constructs the spur gear described by p in doc.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	return Variant{}.Build(ctx, doc, p)
}

// Build constructs the gear with this variant's hooks.
func (v Variant) Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: build: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: build: nil document")
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

// assemble builds tooth zero and the root disc, patterns and joins them, then
// applies the root fillet, bore and cap chamfer.
func (v Variant) assemble(ctx context.Context, r *run) (*decad.Body, error) {
	tooth, err := v.tooth(ctx, r)
	if err != nil {
		return nil, err
	}
	disc, err := buildRootDisc(ctx, r)
	if err != nil {
		return nil, err
	}
	copies, err := decad.PatternCopies(ctx, tooth, decad.CircularPattern{
		Center: r.frame.Origin(),
		Axis:   r.frame.N(),
		Count:  r.teeth,
	})
	if err != nil {
		return nil, fmt.Errorf("spurgear: pattern teeth: %w", err)
	}
	if got, want := len(copies), r.teeth-1; got != want {
		return nil, fmt.Errorf("spurgear: pattern teeth: got %d copies, want %d", got, want)
	}
	gear, err := decad.Union(ctx, disc, append([]*decad.Body{tooth}, copies...)...)
	if err != nil {
		return nil, fmt.Errorf("spurgear: join teeth and root disc: %w", err)
	}
	if !gear.IsSolid() {
		return nil, errors.New("spurgear: join teeth and root disc: result is not a solid")
	}

	switch {
	case v.RootFillet != nil:
		gear, err = v.RootFillet(ctx, r.doc, gear, r.frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	case r.d.fillet > 0:
		gear, err = filletSpurRoots(ctx, r.doc, gear, r, r.d.base < r.d.root)
		if err != nil {
			return nil, fmt.Errorf("spurgear: root fillet: %w", err)
		}
	}

	if r.p.BoreDiameter > 0 {
		gear, err = cutBore(ctx, r.doc, gear, r.p)
		if err != nil {
			return nil, err
		}
	}

	if r.p.CapChamfer > 0 {
		setback, err := r.doc.Param("cap_chamfer")
		if err != nil {
			return nil, fmt.Errorf("spurgear: read parameter cap_chamfer: %w", err)
		}
		gear, err = chamferCaps(ctx, gear, setback)
		if err != nil {
			return nil, err
		}
	}

	if !gear.IsSolid() {
		return nil, errors.New("spurgear: build: result is not a solid")
	}
	return gear, nil
}

// buildRootDisc draws the root circle on the lower plane, binds its radius to
// root_radius, and extrudes it through the gear thickness.
func buildRootDisc(ctx context.Context, r *run) (*decad.Body, error) {
	s, err := r.world.CreateSketch(r.lower)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), r.d.root)
	radius := sketch.NewRadius(circle, r.d.root)
	s.AddConstraint(radius)
	if err := s.Bind(radius, r.world.Params(), "root_radius"); err != nil {
		return nil, fmt.Errorf("spurgear: root disc: bind root_radius: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "root disc")
	if err != nil {
		return nil, err
	}
	disc, err := decad.Extrude(ctx, r.doc, s, profile, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root disc: extrude: %w", err)
	}
	return disc, nil
}

// BuildProfile draws the whole unfilleted gear outline on plane and returns
// the solved, verified sketch and its profile.
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params,
) (*sketch.Sketch, *sketch.Profile, error) {
	d, err := profileGuards(ctx, world, plane, p)
	if err != nil {
		return nil, nil, err
	}
	s, profile, _, err := buildGearProfile(ctx, world, plane, p, d)
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// BuildTwistedProfile draws the whole gear outline with every tooth turned by
// an extra twist about the sketch origin. It also returns the 2N root
// contacts: the right then the left root of each tooth in ascending order.
func BuildTwistedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, twist float64,
) (*sketch.Sketch, *sketch.Profile, [][2]float64, error) {
	d, err := profileGuards(ctx, world, plane, p)
	if err != nil {
		return nil, nil, nil, err
	}
	if math.IsNaN(twist) || math.IsInf(twist, 0) {
		return nil, nil, nil, fmt.Errorf("spurgear: twisted profile: twist must be finite, got %v", twist)
	}
	s, profile, roots, err := buildGearProfileTwisted(ctx, world, plane, p, d, twist)
	if err != nil {
		return nil, nil, nil, err
	}
	out := make([][2]float64, len(roots))
	for i, q := range roots {
		out[i] = [2]float64{q.x, q.y}
	}
	return s, profile, out, nil
}

func profileGuards(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params,
) (dimensions, error) {
	if ctx == nil {
		return dimensions{}, errors.New("spurgear: profile: nil context")
	}
	if world == nil {
		return dimensions{}, errors.New("spurgear: profile: nil world")
	}
	if plane == nil {
		return dimensions{}, errors.New("spurgear: profile: nil plane")
	}
	if err := validate(p); err != nil {
		return dimensions{}, err
	}
	d, err := deriveDimensions(p)
	if err != nil {
		return dimensions{}, fmt.Errorf("spurgear: profile: %w", err)
	}
	return d, nil
}

// buildGearProfile draws the untwisted whole-gear outline.
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions,
) (*sketch.Sketch, *sketch.Profile, []point, error) {
	return buildGearProfileTwisted(ctx, world, plane, p, d, 0)
}

// buildGearProfileTwisted draws the whole-gear outline with each tooth at
// toothAngle(p, k) + twist and returns its root contacts.
func buildGearProfileTwisted(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64,
) (*sketch.Sketch, *sketch.Profile, []point, error) {
	if d.base < d.root {
		return buildEmbeddedProfile(ctx, world, plane, p, d, twist)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: gear outline: create sketch: %w", err)
	}
	center := s.Origin()
	rootPts := make([]*sketch.Point, 0, 2*p.Teeth)
	var firstRight, prevLeft *sketch.Point
	for k := range p.Teeth {
		angle := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
		rightPts, leftPts := flankPoints(s, left, right)
		n := len(rightPts)

		lr := rootContact(left[0], d)
		rr := rootContact(right[0], d)
		leftRoot := sketchcheck.DimensionArcEnd(s, lr.x, lr.y)
		rightRoot := sketchcheck.DimensionPoint(s, rr.x, rr.y)
		rootPts = append(rootPts, rightRoot, leftRoot)

		if k == 0 {
			firstRight = rightRoot
		} else {
			s.CreateArc(center, prevLeft, rightRoot)
		}
		s.CreateLine(rightRoot, rightPts[0])
		if _, err := s.CreateFitSpline(rightPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear outline: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPts[n-1], leftPts[n-1])
		if _, err := s.CreateFitSpline(leftPts...); err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: gear outline: tooth %d left flank: %w", k, err)
		}
		s.CreateLine(leftPts[0], leftRoot)
		prevLeft = leftRoot
	}
	s.CreateArc(center, prevLeft, firstRight)

	profile, err := solveSingleProfile(ctx, s, "gear outline")
	if err != nil {
		return nil, nil, nil, err
	}
	if got, want := len(profile.Outer), 6*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: gear outline: outer boundary has %d curves, want %d", got, want)
	}
	roots := make([]point, len(rootPts))
	for i, rp := range rootPts {
		roots[i] = point{x: rp.X(), y: rp.Y()}
	}
	return s, profile, roots, nil
}

// flankSlot identifies one flank spline of the embedded outline.
type flankSlot struct {
	tooth int
	right bool
}

// buildEmbeddedProfile draws the whole embedded outline: every original flank
// and tip arc plus one full root circle, unioned into one hole-free region.
func buildEmbeddedProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions, twist float64,
) (*sketch.Sketch, *sketch.Profile, []point, error) {
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: create sketch: %w", err)
	}
	center := s.Origin()
	slots := make(map[*sketch.FitSpline]flankSlot, 2*p.Teeth)
	for k := range p.Teeth {
		angle := toothAngle(p, k) + twist
		left, right := sampleFlanks(d.pitch, d.base, d.base, d.tip, float64(p.Teeth), d.backlash, p.Samples, angle)
		rightPts, leftPts := flankPoints(s, left, right)
		n := len(rightPts)
		rs, err := s.CreateFitSpline(rightPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d right flank: %w", k, err)
		}
		s.CreateArc(center, rightPts[n-1], leftPts[n-1])
		ls, err := s.CreateFitSpline(leftPts...)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d left flank: %w", k, err)
		}
		slots[rs] = flankSlot{tooth: k, right: true}
		slots[ls] = flankSlot{tooth: k, right: false}
	}
	circle := s.CreateCircle(center, d.root)
	s.AddConstraint(sketch.NewRadius(circle, d.root))

	if err := solveAndVerify(ctx, s, "embedded outline"); err != nil {
		return nil, nil, nil, err
	}
	regions := s.Profiles()
	if len(regions) == 0 {
		return nil, nil, nil, errors.New("spurgear: embedded outline: sketch has no region")
	}
	indices := make([]int, len(regions))
	for i, region := range regions {
		if !region.Valid {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: region %d is not valid", i)
		}
		indices[i] = i
	}
	profile, err := s.UnionProfiles(indices...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union regions: %w", err)
	}
	if !profile.Valid {
		return nil, nil, nil, errors.New("spurgear: embedded outline: union profile is not valid")
	}
	if got := len(profile.Holes); got != 0 {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: union profile has %d holes, want 0", got)
	}
	if got, want := len(profile.Outer), 4*p.Teeth; got != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: outer boundary has %d curves, want %d", got, want)
	}

	contacts := make([]point, 2*p.Teeth)
	seen := make([]bool, 2*p.Teeth)
	found := 0
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		slot, ok := slots[flank]
		if !ok {
			return nil, nil, nil, errors.New("spurgear: embedded outline: boundary names an unknown flank")
		}
		side := "left"
		idx := 2*slot.tooth + 1
		if slot.right {
			side = "right"
			idx = 2 * slot.tooth
		}
		if !edge.Partial || !edge.TExact {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank trim is not certified (partial %t, exact %t)",
				slot.tooth, side, edge.Partial, edge.TExact)
		}
		x, y := flank.Eval(edge.TStart)
		if dr := math.Abs(math.Hypot(x, y) - d.root); dr > 1e-8 {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank trim lies %g from the root circle",
				slot.tooth, side, dr)
		}
		if seen[idx] {
			return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: tooth %d %s flank appears twice on the boundary",
				slot.tooth, side)
		}
		seen[idx] = true
		contacts[idx] = point{x: x, y: y}
		found++
	}
	if want := 2 * p.Teeth; found != want {
		return nil, nil, nil, fmt.Errorf("spurgear: embedded outline: found %d root contacts, want %d", found, want)
	}
	return s, profile, contacts, nil
}

// closingRadius chooses the radius of the embedded tooth's closing arc.
func closingRadius(p Params, d dimensions) (float64, error) {
	teeth := float64(p.Teeth)
	limit := math.Pi / teeth
	h := func(r float64) float64 {
		return halfAngle(d.pitch, d.base, teeth, d.backlash, r)
	}
	if h(d.base) < limit {
		return d.base, nil
	}
	if h(d.root) >= limit {
		return 0, fmt.Errorf("spurgear: embedded tooth: adjacent teeth meet at the root circle (radius %v); no closing radius exists",
			d.root)
	}
	lo, hi := d.base, d.root
	for range 200 {
		mid := (lo + hi) / 2
		if h(mid) >= limit {
			lo = mid
		} else {
			hi = mid
		}
	}
	rq := (lo + hi) / 2
	return (rq + d.root) / 2, nil
}

// buildEmbeddedTooth draws the four-curve embedded tooth section at angle 0.
func buildEmbeddedTooth(ctx context.Context, world *sketch.World, plane *sketch.Plane,
	p Params, d dimensions,
) (*sketch.Sketch, *sketch.Profile, error) {
	rc, err := closingRadius(p, d)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: create sketch: %w", err)
	}
	center := s.Origin()
	left, right := sampleFlanks(d.pitch, d.base, rc, d.tip, float64(p.Teeth), d.backlash, p.Samples, 0)
	n := len(right)
	rightPts := make([]*sketch.Point, n)
	leftPts := make([]*sketch.Point, n)
	for i, q := range right {
		rightPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}
	for i, q := range left {
		if i == 0 || i == n-1 {
			leftPts[i] = sketchcheck.DimensionArcEnd(s, q.x, q.y)
			continue
		}
		leftPts[i] = sketchcheck.DimensionPoint(s, q.x, q.y)
	}

	if _, err := s.CreateFitSpline(rightPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: tooth 0 right flank: %w", err)
	}
	s.CreateArc(center, rightPts[n-1], leftPts[n-1])
	if _, err := s.CreateFitSpline(leftPts...); err != nil {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: tooth 0 left flank: %w", err)
	}
	s.CreateArc(center, rightPts[0], leftPts[0])

	profile, err := solveSingleProfile(ctx, s, "embedded tooth")
	if err != nil {
		return nil, nil, err
	}
	if got, want := len(profile.Outer), 4; got != want {
		return nil, nil, fmt.Errorf("spurgear: embedded tooth: outer boundary has %d curves, want %d", got, want)
	}
	return s, profile, nil
}

// filletRoots fillets the ordinary root edges matched to roots with a
// radius in millimeters.
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
	radius float64,
) (*decad.Body, error) {
	return filletRootsValue(ctx, body, frame, roots, units.Millimeters(radius))
}

// filletRootsValue fillets the axial edges whose lower endpoints lie on the
// saved ordinary root contacts.
func filletRootsValue(ctx context.Context, body *decad.Body, frame r3.Frame,
	roots []point, radius units.Value,
) (*decad.Body, error) {
	if len(roots) == 0 {
		return nil, errors.New("spurgear: root fillet: no root contacts")
	}
	axis := frame.N()
	edges, err := decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: select axial edges: %w", err)
	}
	world := make([]r3.Vec, len(roots))
	for i, root := range roots {
		world[i] = toWorld(frame, root)
	}
	matches := func(pos r3.Vec) bool {
		for _, w := range world {
			if pos.Sub(w).Len() <= 1e-6 {
				return true
			}
		}
		return false
	}
	corners := make([]r3.Vec, 0, len(roots))
	for _, edge := range edges {
		start := edge.Start().Position().Value
		if matches(start) {
			corners = append(corners, start)
			continue
		}
		end := edge.End().Position().Value
		if matches(end) {
			corners = append(corners, end)
		}
	}
	if got, want := len(corners), len(roots); got != want {
		return nil, fmt.Errorf("spurgear: root fillet: matched %d root edges, want %d", got, want)
	}
	selection := decad.Edges(decad.ParallelTo(axis), decad.EndpointAt(corners[0]))
	for _, corner := range corners[1:] {
		selection = selection.Or(decad.ParallelTo(axis), decad.EndpointAt(corner))
	}
	filleted, err := decad.Fillet(ctx, body, selection.Exactly(len(corners)), radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: root fillet: %w", err)
	}
	return filleted, nil
}

// filletEmbedded fillets the 2N concave axial edges of an embedded gear.
func filletEmbedded(ctx context.Context, body *decad.Body, frame r3.Frame,
	teeth int, radius units.Value,
) (*decad.Body, error) {
	selection := decad.Edges(decad.ParallelTo(frame.N()), decad.Concave()).Exactly(2 * teeth)
	filleted, err := decad.Fillet(ctx, body, selection, radius)
	if err != nil {
		return nil, fmt.Errorf("spurgear: embedded root fillet: %w", err)
	}
	return filleted, nil
}

// buildBoreProfile draws the bore tool circle on a plane margin below the
// lower cap, binding its diameter to bore_diameter.
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
	margin float64,
) (*sketch.Sketch, *sketch.Profile, error) {
	if err := defineParameters(world.Params(), p); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: placement: %w", err)
	}
	toolFrame, err := shifted(frame, -margin)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: shift placement: %w", err)
	}
	plane, err := world.CreatePlaneFromFrame(toolFrame)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: create plane: %w", err)
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: create sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), p.BoreDiameter/2)
	diameter := sketch.NewDiameter(circle, p.BoreDiameter)
	s.AddConstraint(diameter)
	if err := s.Bind(diameter, world.Params(), "bore_diameter"); err != nil {
		return nil, nil, fmt.Errorf("spurgear: bore tool: bind bore_diameter: %w", err)
	}
	profile, err := solveSingleProfile(ctx, s, "bore tool")
	if err != nil {
		return nil, nil, err
	}
	return s, profile, nil
}

// cutBore cuts the centered through-bore using the document's existing
// parameter table.
func cutBore(ctx context.Context, doc *decad.Document, gear *decad.Body, p Params) (*decad.Body, error) {
	margin, err := doc.Param("tool_margin")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read parameter tool_margin: %w", err)
	}
	length, err := doc.Param("bore_tool_length")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read parameter bore_tool_length: %w", err)
	}
	world := sketch.NewWorld()
	s, profile, err := buildBoreProfile(ctx, world, p, margin.Base())
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, doc, s, profile, decad.Distance{D: length, Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore tool: extrude: %w", err)
	}
	cut, err := decad.Cut(ctx, gear, tool)
	if err != nil {
		return nil, fmt.Errorf("spurgear: bore cut: %w", err)
	}
	return cut, nil
}

// CutThroughBore cuts a centered through-bore of p.BoreDiameter from a body
// spanning the gear thickness. It defines the spur parameters on doc first.
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
	p Params,
) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: through-bore: nil context")
	}
	if doc == nil {
		return nil, errors.New("spurgear: through-bore: nil document")
	}
	if body == nil {
		return nil, errors.New("spurgear: through-bore: nil body")
	}
	if !(p.BoreDiameter > 0) {
		return nil, fmt.Errorf("spurgear: through-bore: bore diameter must be positive, got %v", p.BoreDiameter)
	}
	if err := defineParameters(doc.Params(), p); err != nil {
		return nil, fmt.Errorf("spurgear: through-bore: %w", err)
	}
	return cutBore(ctx, doc, body, p)
}

// chamferCaps chamfers both outer cap loops of gear with setback. It checks
// no arguments; its callers do.
func chamferCaps(ctx context.Context, gear *decad.Body, setback units.Value) (*decad.Body, error) {
	selection := decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
		decad.OuterLoopOf(decad.CapEnd(gear)))
	chamfered, err := decad.Chamfer(ctx, gear, selection, setback)
	if err != nil {
		return nil, fmt.Errorf("spurgear: cap chamfer: %w", err)
	}
	return chamfered, nil
}

// ChamferOuterCaps chamfers both outer cap loops of gear with distance,
// leaving bore rims sharp.
func ChamferOuterCaps(ctx context.Context, gear *decad.Body, distance float64) (*decad.Body, error) {
	if ctx == nil {
		return nil, errors.New("spurgear: cap chamfer: nil context")
	}
	if gear == nil {
		return nil, errors.New("spurgear: cap chamfer: nil body")
	}
	if math.IsNaN(distance) || math.IsInf(distance, 0) || distance <= 0 {
		return nil, fmt.Errorf("spurgear: cap chamfer: distance must be positive and finite, got %v", distance)
	}
	return chamferCaps(ctx, gear, units.Millimeters(distance))
}
