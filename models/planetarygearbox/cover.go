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
// bolt holes as inner loops of its section.
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
	if err := r.addCircle(s, 2*d.ringOuter, "2 * ring_outer_radius"); err != nil {
		return nil, fmt.Errorf("ring blank: %w", err)
	}
	if err := r.addBoltHoles(s); err != nil {
		return nil, fmt.Errorf("ring blank: %w", err)
	}
	profile, err := profileWithHoles(ctx, s, 4, "ring blank")
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
	if err := r.addCircle(s, d.lip, "lip_bore_diameter"); err != nil {
		return nil, fmt.Errorf("cover frame: %w", err)
	}
	if err := r.addBoltHoles(s); err != nil {
		return nil, fmt.Errorf("cover frame: %w", err)
	}
	// at returns the point on spoke k's axis at distance x and sideways offset y.
	at := func(k int, x, y float64) (float64, float64) {
		sin, cos := math.Sincos(spokeAngle(k))
		return x*cos - y*sin, x*sin + y*cos
	}
	for k := range coverSpokes {
		next := (k + 1) % coverSpokes
		// Window k lies between spoke k's upper edge and the next spoke's lower edge. Its hub arc
		// and rim arc each have one fully dimensioned end and one end the arc's radius relation fixes.
		ax, ay := at(k, hubX, half)
		bx, by := at(k, rimX, half)
		cx, cy := at(next, rimX, -half)
		dx, dy := at(next, hubX, -half)
		a := sketchcheck.DimensionPoint(s, ax, ay)
		b := sketchcheck.DimensionPoint(s, bx, by)
		c := sketchcheck.DimensionArcEnd(s, cx, cy)
		dd := sketchcheck.DimensionArcEnd(s, dx, dy)
		s.CreateLine(a, b)
		s.CreateArc(s.Origin(), b, c)
		s.CreateLine(c, dd)
		s.CreateArc(s.Origin(), a, dd)
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
	if err := r.addCircle(s, d.pocket, "pocket_diameter"); err != nil {
		return nil, fmt.Errorf("cover boss: %w", err)
	}
	profile, err := profileWithHoles(ctx, s, 1, "cover boss")
	if err != nil {
		return nil, err
	}
	boss, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(1 + planetBearing().Width), Dir: decad.Along})
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
	if cover, err = r.cutHeadPockets(ctx, cover); err != nil {
		return nil, err
	}
	return r.chamferSeat(ctx, cover, "cover bearing pocket", p.CoverThickness+planetBearing().Width)
}

// buildInputShaft builds the input shaft about the local origin: a D-shaped rod that starts InputShaftLength
// below the front cover's bearing boss and ends flush with the top of the first sun, so it fills the sun's
// bore. A collar between the front bearing's inner race and the sun's lower face keeps the sun off the cover, with
// collarClearance to spare.
func (r *run) buildInputShaft(ctx context.Context) (*decad.Body, error) {
	p := r.p
	start := -(p.StageGap + p.CoverThickness + planetBearing().Width + p.InputShaftLength)
	inner := -(p.StageGap + p.CoverThickness)
	// The groove runs from the sun's lower face to past the shaft's top, through the sun.
	return r.buildShaft(ctx, "input shaft", start, p.GearThickness, inner, -collarClearance, span{0, p.GearThickness + 1})
}

// chamferSeat chamfers the entrance of a bearing seat by SeatChamfer, so a bearing can be pressed into the tight
// printed pocket. The entrance is the pocket's circular rim in the plane z. A boolean moves a circle's seam, so
// the rim is found by its radius and height and selected by its stored start point. It returns body unchanged
// when SeatChamfer is zero.
func (r *run) chamferSeat(ctx context.Context, body *decad.Body, what string, z float64) (*decad.Body, error) {
	if r.p.SeatChamfer == 0 {
		return body, nil
	}
	size, err := r.doc.Param("seat_chamfer")
	if err != nil {
		return nil, fmt.Errorf("%s: read chamfer size: %w", what, err)
	}
	var seam *r3.Vec
	for _, e := range body.Edges() {
		at := e.Start().Position().Value
		if math.Abs(at.Z-z) < 1e-6 && math.Abs(math.Hypot(at.X, at.Y)-r.d.pocket/2) < 1e-3 {
			if seam != nil {
				return nil, fmt.Errorf("%s: more than one edge starts on the pocket's rim", what)
			}
			seam = &at
		}
	}
	if seam == nil {
		return nil, fmt.Errorf("%s: no edge starts on the pocket's rim", what)
	}
	rim := decad.Edges(decad.Circular(), decad.EndpointAt(*seam)).Exactly(1)
	out, err := decad.Chamfer(ctx, body, rim, size)
	if err != nil {
		return nil, fmt.Errorf("%s: chamfer entrance: %w", what, err)
	}
	if !out.IsSolid() {
		return nil, fmt.Errorf("%s: chamfered result is not a solid", what)
	}
	return out, nil
}
