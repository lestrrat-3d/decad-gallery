package planetarygearbox

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

// boltAngles are the angles of the four bolt holes, between the cover spokes and the ring's tooth zero.
func boltAngles() []float64 {
	angles := make([]float64, 4)
	for i := range angles {
		angles[i] = math.Pi/4 + float64(i)*math.Pi/2
	}
	return angles
}

// spokeAngle is the angle of spoke k of a cover.
func spokeAngle(k int) float64 { return math.Pi/6 + 2*math.Pi*float64(k)/coverSpokes }

// addBoltHoles draws the four bolt-hole circles on s, each centered on the bolt circle and bound to the
// bolt hole diameter. They become inner loops of the profile.
func (r *run) addBoltHoles(s *sketch.Sketch) error {
	d := r.d
	for i, angle := range boltAngles() {
		center := sketchcheck.DimensionPoint(s, d.boltCircle*math.Cos(angle), d.boltCircle*math.Sin(angle))
		circle := s.CreateCircle(center, r.p.BoltHoleDiameter/2)
		dim := sketch.NewDiameter(circle, r.p.BoltHoleDiameter)
		s.AddConstraint(dim)
		if err := s.Bind(dim, r.world.Params(), "bolt_hole_diameter"); err != nil {
			return fmt.Errorf("bolt hole %d: bind diameter: %w", i, err)
		}
	}
	return nil
}

// addCircle draws a circle on the sketch origin bound to a table expression for its diameter.
func (r *run) addCircle(s *sketch.Sketch, diameter float64, expr string) error {
	circle := s.CreateCircle(s.Origin(), diameter/2)
	dim := sketch.NewDiameter(circle, diameter)
	s.AddConstraint(dim)
	if err := s.Bind(dim, r.world.Params(), expr); err != nil {
		return fmt.Errorf("bind %q: %w", expr, err)
	}
	return nil
}

// profileWithHoles solves and verifies s and returns the one valid profile with the given number of holes.
func profileWithHoles(ctx context.Context, s *sketch.Sketch, holes int, what string) (*sketch.Profile, error) {
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", what, err)
	}
	var found *sketch.Profile
	for _, candidate := range s.Profiles() {
		if candidate.Valid && len(candidate.Holes) == holes {
			if found != nil {
				return nil, fmt.Errorf("%s: more than one region has %d holes", what, holes)
			}
			found = candidate
		}
	}
	if found == nil {
		return nil, fmt.Errorf("%s: no valid region with %d holes among %d regions", what, holes, len(s.Profiles()))
	}
	return found, nil
}

// extrudeRingBlank extrudes the ring's outer cylinder from z=ringMargin up the ring length, with the four
// bolt holes and the bore of each mounting ear as inner loops of its section.
func (r *run) extrudeRingBlank(ctx context.Context) (*decad.Body, error) {
	d := r.d
	plane, err := r.planeAt(ringMargin)
	if err != nil {
		return nil, fmt.Errorf("ring blank: %w", err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("ring blank: sketch: %w", err)
	}
	if err := r.addRingOutline(s); err != nil {
		return nil, fmt.Errorf("ring blank: %w", err)
	}
	if err := r.addBoltHoles(s); err != nil {
		return nil, fmt.Errorf("ring blank: %w", err)
	}
	profile, err := profileWithHoles(ctx, s, 4+r.p.EarCount, "ring blank")
	if err != nil {
		return nil, err
	}
	blank, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(d.ringLength), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("ring blank: extrude: %w", err)
	}
	return blank, nil
}

// axisLength is the length of the construction lines that hold the cover spokes' directions.
const axisLength = 10

// extrudeCoverFrame extrudes a cover's frame from z=0 up the cover thickness: an outer rim with the four
// bolt holes, a hub with the lip bore through it, and spokes joining them. Between the spokes are three
// windows, so the section is one region with eight holes.
func (r *run) extrudeCoverFrame(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	half := d.arm / 2
	hubR, rimR := d.coverHub/2, d.rimInner
	hubX := math.Sqrt(hubR*hubR - half*half)
	rimX := math.Sqrt(rimR*rimR - half*half)
	s, err := r.world.CreateSketch(r.world.XY())
	if err != nil {
		return nil, fmt.Errorf("cover frame: sketch: %w", err)
	}
	if err := r.addCircle(s, 2*d.ringOuter, "2 * ring_outer_radius"); err != nil {
		return nil, fmt.Errorf("cover frame: %w", err)
	}
	if err := r.addCircle(s, d.coverLip, "cover_lip_bore_diameter"); err != nil {
		return nil, fmt.Errorf("cover frame: %w", err)
	}
	if err := r.addBoltHoles(s); err != nil {
		return nil, fmt.Errorf("cover frame: %w", err)
	}
	// A horizontal reference line and one construction axis per spoke hold the spoke edges' directions.
	refEnd := s.CreatePoint(axisLength, 0)
	ref := s.CreateLine(s.Origin(), refEnd)
	ref.SetConstruction(true)
	s.AddConstraint(sketch.NewHorizontal(ref), sketch.NewHorizontalDistance(s.Origin(), refEnd, axisLength))
	// at returns the point on spoke k's axis at distance x and sideways offset y.
	at := func(k int, x, y float64) (float64, float64) {
		sin, cos := math.Sincos(spokeAngle(k))
		return x*cos - y*sin, x*sin + y*cos
	}
	axes := make([]*sketch.Line, coverSpokes)
	for k := range axes {
		ex, ey := at(k, axisLength, 0)
		end := s.CreatePoint(ex, ey)
		axes[k] = s.CreateLine(s.Origin(), end)
		axes[k].SetConstruction(true)
		s.AddConstraint(
			sketch.NewAngle(ref, axes[k], spokeAngle(k)*180/math.Pi),
			sketch.NewDistance(s.Origin(), end, axisLength),
		)
	}
	// Each window's four corners take the spoke's own round, half the spoke width: the spokes carry the bearing's
	// load from the hub to the rim, so their roots are stressed roots.
	round := d.arm / 2
	for k := range coverSpokes {
		next := (k + 1) % coverSpokes
		// Window k lies between spoke k's upper edge and the next spoke's lower edge. The edges are parallel to
		// their spoke's axis and half the spoke width from it; the arcs are about the origin.
		ax, ay := at(k, hubX, half)
		bx, by := at(k, rimX, half)
		cx, cy := at(next, rimX, -half)
		dx, dy := at(next, hubX, -half)
		a, b, c, dd := s.CreatePoint(ax, ay), s.CreatePoint(bx, by), s.CreatePoint(cx, cy), s.CreatePoint(dx, dy)
		ab := s.CreateLine(a, b)
		rim := s.CreateArc(s.Origin(), b, c)
		cd := s.CreateLine(c, dd)
		hub := s.CreateArc(s.Origin(), a, dd)
		s.AddConstraint(
			sketch.NewParallel(axes[k], ab), sketch.NewDistancePointLine(s.Origin(), ab, half),
			sketch.NewParallel(axes[next], cd), sketch.NewDistancePointLine(s.Origin(), cd, half),
			sketch.NewRadius(rim, rimR), sketch.NewRadius(hub, hubR),
		)
		if _, err := sketch.CreateFillet(s, hub, ab, a, round); err != nil {
			return nil, fmt.Errorf("cover frame: window %d: round hub corner after spoke %d: %w", k+1, k+1, err)
		}
		if _, err := sketch.CreateFillet(s, ab, rim, b, round); err != nil {
			return nil, fmt.Errorf("cover frame: window %d: round rim corner after spoke %d: %w", k+1, k+1, err)
		}
		if _, err := sketch.CreateFillet(s, rim, cd, c, round); err != nil {
			return nil, fmt.Errorf("cover frame: window %d: round rim corner before spoke %d: %w", k+1, next+1, err)
		}
		if _, err := sketch.CreateFillet(s, cd, hub, dd, round); err != nil {
			return nil, fmt.Errorf("cover frame: window %d: round hub corner before spoke %d: %w", k+1, next+1, err)
		}
	}
	profile, err := profileWithHoles(ctx, s, coverSpokes+1+4, "cover frame")
	if err != nil {
		return nil, err
	}
	frame, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(p.CoverThickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cover frame: extrude: %w", err)
	}
	return frame, nil
}

// buildCover builds an end cover about the local origin: the frame from z=0 to the cover thickness, and on
// its +Z face a bearing boss, a ring of the boss diameter around a pocket of the bearing pocket diameter,
// as tall as the bearing is wide. The boss starts a millimeter inside the frame so the union overlaps.
func (r *run) buildCover(ctx context.Context) (*decad.Body, error) {
	p, d := r.p, r.d
	frame, err := r.extrudeCoverFrame(ctx)
	if err != nil {
		return nil, err
	}
	plane, err := r.planeAt(p.CoverThickness - 1)
	if err != nil {
		return nil, fmt.Errorf("cover boss: %w", err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("cover boss: sketch: %w", err)
	}
	if err := r.addCircle(s, d.coverBoss, "cover_boss_diameter"); err != nil {
		return nil, fmt.Errorf("cover boss: %w", err)
	}
	if err := r.addCircle(s, d.coverPocket, "cover_pocket_diameter"); err != nil {
		return nil, fmt.Errorf("cover boss: %w", err)
	}
	profile, err := profileWithHoles(ctx, s, 1, "cover boss")
	if err != nil {
		return nil, err
	}
	boss, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(1 + shaftBearing(p).Width), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("cover boss: extrude: %w", err)
	}
	cover, err := decad.Union(ctx, frame, boss)
	if err != nil {
		return nil, fmt.Errorf("cover: join frame and bearing boss: %w", err)
	}
	if !cover.IsSolid() {
		return nil, errors.New("cover: result is not a solid")
	}
	if cover, err = r.roundBossRoot(ctx, cover); err != nil {
		return nil, err
	}
	if cover, err = r.cutHeadPockets(ctx, cover); err != nil {
		return nil, err
	}
	if cover, err = r.chamferSeat(ctx, cover, "cover bearing pocket",
		p.CoverThickness+shaftBearing(p).Width, d.coverPocket); err != nil {
		return nil, err
	}
	// Both flat faces lose their outer rim. The inner face also loses the bearing lip's edge: the bearing rests
	// on the other side of the lip, so no bearing seat shrinks.
	lip := func(loop *decad.Loop, edges []*decad.Edge) bool {
		return loop.IsOuter() || len(edges) > 1 || (len(edges) == 1 && math.Abs(radiusAbout(edges[0])-d.coverLip/2) < 1e-3)
	}
	if cover, err = r.finishFaceEdges(ctx, cover, "cover inner face", 0, lip); err != nil {
		return nil, err
	}
	if cover, err = r.finishFaceEdges(ctx, cover, "cover outer face", p.CoverThickness, windowsAndRim); err != nil {
		return nil, err
	}
	return cover, nil
}

// buildInputShaft builds the input shaft about the local origin: a D-shaped rod that starts InputShaftLength
// below the front cover's bearing boss and ends flush with the top of the first sun, so it fills the sun's
// bore. A collar between the front bearing's inner race and the sun's lower face keeps the sun off the cover, with
// collarBearingGap to the bearing and collarClearance to the sun.
func (r *run) buildInputShaft(ctx context.Context) (*decad.Body, error) {
	p := r.p
	start := -(p.StageGap + p.CoverThickness + shaftBearing(p).Width + p.InputShaftLength)
	inner := -(p.StageGap + p.CoverThickness) + collarBearingGap
	// The flat runs from the sun's lower face to past the shaft's top, through the sun.
	// A second flat runs along the end that reaches out of the cover, where a coupling or a set screw grips.
	return r.buildShaft(ctx, "input shaft", start, p.GearThickness, inner, -collarClearance,
		span{0, p.GearThickness + 1}, span{start - 1, start + p.InputShaftLength - endFlatMargin})
}

// circleSeam returns the start of the one edge of body that starts at height z on a circle of the given radius about
// the axis. A boolean moves a circle's seam, so a circular edge is found by its radius and height and selected by its
// stored start point. what names the feature in errors.
func circleSeam(body *decad.Body, what string, z, radius float64) (r3.Vec, error) {
	var seam *r3.Vec
	for _, e := range body.Edges() {
		at := e.Start().Position().Value
		if math.Abs(at.Z-z) < 1e-6 && math.Abs(math.Hypot(at.X, at.Y)-radius) < 1e-3 {
			if seam != nil {
				return r3.Vec{}, fmt.Errorf("%s: more than one edge starts on the circle of radius %v", what, radius)
			}
			seam = &at
		}
	}
	if seam == nil {
		return r3.Vec{}, fmt.Errorf("%s: no edge starts on the circle of radius %v", what, radius)
	}
	return *seam, nil
}

// bossRootRound is the radius of the round where the bearing boss stands on the frame: half the thinner of the boss
// wall and the frame (1.5 mm by default). The corner opens upward in the print orientation, so it takes a round.
func bossRootRound(p Params) float64 { return math.Min(p.CarrierWall, p.CoverThickness) / 2 }

// roundBossRoot rounds the concave circular edge where the bearing boss's outside wall stands on the frame's +Z face.
func (r *run) roundBossRoot(ctx context.Context, body *decad.Body) (*decad.Body, error) {
	seam, err := circleSeam(body, "cover boss root", r.p.CoverThickness, r.d.coverBoss/2)
	if err != nil {
		return nil, err
	}
	root := decad.Edges(decad.Circular(), decad.EndpointAt(seam)).Exactly(1)
	out, err := decad.Fillet(ctx, body, root, units.Millimeters(bossRootRound(r.p)))
	if err != nil {
		return nil, fmt.Errorf("cover boss root: round: %w", err)
	}
	if !out.IsSolid() {
		return nil, errors.New("cover boss root: rounded result is not a solid")
	}
	return out, nil
}

// chamferSeat chamfers the entrance of a bearing seat by SeatChamfer, so a bearing can be pressed into the tight
// printed pocket. The entrance is the pocket's circular rim in the plane z. A boolean moves a circle's seam, so
// the rim is found by its radius and height and selected by its stored start point. It returns body unchanged
// when SeatChamfer is zero. pocket is the pocket's diameter.
func (r *run) chamferSeat(ctx context.Context, body *decad.Body, what string, z, pocket float64) (*decad.Body, error) {
	if r.p.SeatChamfer == 0 {
		return body, nil
	}
	size, err := r.doc.Param("seat_chamfer")
	if err != nil {
		return nil, fmt.Errorf("%s: read chamfer size: %w", what, err)
	}
	seam, err := circleSeam(body, what, z, pocket/2)
	if err != nil {
		return nil, err
	}
	rim := decad.Edges(decad.Circular(), decad.EndpointAt(seam)).Exactly(1)
	out, err := decad.Chamfer(ctx, body, rim, size)
	if err != nil {
		return nil, fmt.Errorf("%s: chamfer entrance: %w", what, err)
	}
	if !out.IsSolid() {
		return nil, fmt.Errorf("%s: chamfered result is not a solid", what)
	}
	return out, nil
}
