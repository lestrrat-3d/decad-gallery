package sevensegment

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Frame dimensions in mm that are not exported.
const (
	wallTopZ       = flangeFrom + ribHeight + 1.5  // z of the wall's upper edge: ribs end 1.5 below it
	wallHalf       = WallHalfWidth + 1.5           // wall plate half width: ribs end 1.5 inside it ("Wall inset (decad constraint)")
	wallBottomZ    = ShaftZ - CamRadius - 4.5      // z of the wall's lower edge, 3.0 below the cams' rims
	ribInner       = 24.6                          // |x| where a rib starts, inside the nut pockets' flats
	ledgeFrom      = guideFrom - Seam - ribHeight  // z of a ledge rib's underside: the guide plate stands on it over a seam
	flangeFrom     = cradleFrom - Seam - ribHeight // z of a top flange rib's underside: the cradle plate stands on it over a seam
	ribHeight      = 6.0
	ribDepth       = 15.0
	frontLedgeDeep = -47.9 - frontInnerY // -51.3 + 3.4: front ledge ribs reach this far past the G1/G2 head pockets
	rearLedgeDeep  = rearInnerY - 44.8   // 48.0 - 3.2: rear ledge ribs reach y 44.8, past the G3/G4 head pockets
	ribRound       = 0.5
	wallChamfer    = 0.5
	frontInnerY    = -74.25
	rearInnerY     = 64.5
	markRadius     = 13.5 // look-alike hole circle on the walls: off the housing flange (1.5 x 10 is 15, under the flange)

	// The guide plate: z 54 to 58 in the original 1.0x design.
	guideFrom  = 81.0
	guideThick = 6.0
	guideTop   = guideFrom + guideThick
)

// wallFrame is a wall's sketch frame: origin on the shaft axis in its inner face, V = +Z and N outward.
func wallFrame(front bool) r3.Frame {
	if front {
		f, _ := r3.NewFrame(r3.Vec{Y: frontInnerY, Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		return f
	}
	f, _ := r3.NewFrame(r3.Vec{Y: rearInnerY, Z: ShaftZ}, r3.Vec{X: -1}, r3.Vec{Z: 1})
	return f
}

// rectOn adds the rectangle u in [u0, u1], v in [v0, v1] to s, its sides held at their distances from the origin.
func rectOn(s *sketch.Sketch, u0, u1, v0, v1 float64) {
	polygon(s, [][2]float64{{u0, v0}, {u1, v0}, {u1, v1}, {u0, v1}})
}

// box extrudes the rectangle in the plane of f against its normal by depth.
func (r *run) box(f r3.Frame, u0, u1, v0, v1, depth float64) (*decad.Body, error) {
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	rectOn(s, u0, u1, v0, v1)
	return r.extrude(s, 0, depth, decad.Against)
}

// wall builds the front or rear wall in the world: the plate, four ribs, chamfers, rib root rounds, the bearing seat and
// lip, the fastener tools and the look-alike holes.
func (r *run) wall(front bool) (*decad.Body, error) {
	f := wallFrame(front)
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	rectOn(s, -wallHalf, wallHalf, wallBottomZ-ShaftZ, wallTopZ-ShaftZ)
	body, err := r.extrude(s, 0, WallThickness, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	if body, err = r.addRibs(body, f, front); err != nil {
		return nil, err
	}
	innerY := rearInnerY
	outerY := rearInnerY + WallThickness
	if front {
		innerY, outerY = frontInnerY, frontInnerY-WallThickness
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Y: 1}, []float64{outerY}, edgefinish.Long(edgefinish.Outer),
		wallChamfer); err != nil {
		return nil, fmt.Errorf("chamfer outer face: %w", err)
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Y: 1}, []float64{innerY}, edgefinish.Long(edgefinish.Outer),
		wallChamfer); err != nil {
		return nil, fmt.Errorf("chamfer inner face: %w", err)
	}
	// Each rib's root is one closed loop; decad rounds one loop per call.
	for _, rib := range []struct{ side, level float64 }{{-1, 0}, {1, 0}, {-1, 1}, {1, 1}} {
		if body, err = roundRoot(r.ctx, body, rib.side, rib.level, innerY); err != nil {
			return nil, fmt.Errorf("rib root: %w", err)
		}
	}
	seat, err := r.seatTool()
	if err != nil {
		return nil, fmt.Errorf("seat tool: %w", err)
	}
	seat, err = placement.Place(r.ctx, seat, &f)
	if err != nil {
		return nil, fmt.Errorf("place seat tool: %w", err)
	}
	// The fastener tools go first: cut after the seat, the front wall's G pockets meet a held mesh coarser than
	// their chord tolerance.
	if body, err = r.wallFasteners(body, front); err != nil {
		return nil, err
	}
	if body, err = decad.Cut(r.ctx, body, seat); err != nil {
		return nil, fmt.Errorf("cut seat: %w", err)
	}
	return r.wallMarks(body, front, f)
}

func depth(front, ledge bool) float64 {
	if front && ledge {
		return frontLedgeDeep
	}
	if ledge {
		return rearLedgeDeep
	}
	return ribDepth
}

// seatTool is the revolved cutter for the 608ZZ seat and the lip hole, in the wall's local frame (axis Z, depth d along
// +Z from the inner face).
func (r *run) seatTool() (*decad.Body, error) {
	// The seat, the lip and the lead-ins belong to the bearing, which is purchased: they keep their size. The lip
	// hole stands 0.5 outside the inner ring's face and ends at the outer face, WallThickness from the inner one.
	return r.meridian([][2]float64{{0, -1}, {bearingSeatR + 0.5, -1}, {bearingSeatR + 0.5, 0}, {bearingSeatR, 0.5},
		{bearingSeatR, bearingWidth}, {bearingRaceR + 0.5, bearingWidth}, {bearingRaceR + 0.5, WallThickness - 0.5},
		{bearingRaceR + 1.0, WallThickness}, {bearingRaceR + 1.0, WallThickness + 1}, {0, WallThickness + 1}})
}

// wallFasteners cuts the fastener tools of the wall's sites: front wall T1, T2, G1, G2, D1, D2; rear wall T3, T4, G3, G4.
func (r *run) wallFasteners(body *decad.Body, front bool) (*decad.Body, error) {
	type job struct {
		site  m2fastener.Site
		kind  string // "T" nut pocket and hole, "G" head pocket and hole, "D" nut pocket and hole
		label string
	}
	var jobs []job
	sites := r.dm.Sites
	if front {
		jobs = []job{{sites[0], "T", "T1"}, {sites[1], "T", "T2"}, {sites[4], "G", "G1"}, {sites[5], "G", "G2"},
			{sites[8], "D", "D1"}, {sites[9], "D", "D2"}}
	} else {
		jobs = []job{{sites[2], "T", "T3"}, {sites[3], "T", "T4"}, {sites[6], "G", "G3"}, {sites[7], "G", "G4"}}
	}
	for _, j := range jobs {
		var tools []*decad.Body
		hole, sk, err := m2fastener.HoleTool(r.ctx, r.doc, j.site)
		if err != nil {
			return nil, fmt.Errorf("%s hole: %w", j.label, err)
		}
		r.sketches = append(r.sketches, sk)
		tools = append(tools, hole)
		var pocket *decad.Body
		if j.kind == "G" {
			pocket, sk, err = m2fastener.PocketTool(r.ctx, r.doc, j.site)
		} else {
			pocket, sk, err = m2fastener.HexPocketTool(r.ctx, r.doc, j.site, m2fastener.NutCavityHeight+m2fastener.Stock)
		}
		if err != nil {
			return nil, fmt.Errorf("%s pocket: %w", j.label, err)
		}
		r.sketches = append(r.sketches, sk)
		tools = append(tools, pocket)
		for _, tool := range tools {
			if body, err = decad.Cut(r.ctx, body, tool); err != nil {
				return nil, fmt.Errorf("cut %s: %w", j.label, err)
			}
		}
	}
	return body, nil
}

// wallMarks cuts the look-alike holes: the front wall one, the rear wall two, on a circle of markRadius about the shaft
// axis from 270 degrees, 25 degrees apart.
func (r *run) wallMarks(body *decad.Body, front bool, f r3.Frame) (*decad.Body, error) {
	n := 2
	if front {
		n = 1
	}
	for i := range n {
		a := (camMarkFirst + camMarkStep*float64(i)) * math.Pi / 180
		x, v := markRadius*math.Cos(a), markRadius*math.Sin(a) // front-view coordinates
		u := x
		if !front {
			u = -x
		}
		s, err := r.sketchOn(shift(f, -1))
		if err != nil {
			return nil, err
		}
		c := sketchcheck.DimensionPoint(s, u, v)
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, camMarkRadius), camMarkRadius))
		hole, err := r.extrude(s, 0, WallThickness+2, decad.Along)
		if err != nil {
			return nil, fmt.Errorf("mark %d: %w", i+1, err)
		}
		if body, err = decad.Cut(r.ctx, body, hole); err != nil {
			return nil, fmt.Errorf("cut mark %d: %w", i+1, err)
		}
	}
	return body, nil
}

// shift moves frame f by d along its normal.
func shift(f r3.Frame, d float64) r3.Frame {
	g, _ := r3.NewFrame(f.Origin().Add(f.N().Scale(d)), f.U(), f.V())
	return g
}

// addRibs unions the four ribs of a wall onto its plate.
func (r *run) addRibs(body *decad.Body, f r3.Frame, front bool) (*decad.Body, error) {
	for _, rib := range []struct{ v0, depth float64 }{
		{ledgeFrom - ShaftZ, depth(front, true)}, {flangeFrom - ShaftZ, ribDepth},
	} {
		for _, sgn := range []float64{-1, 1} {
			u0, u1 := ribInner, WallHalfWidth
			if sgn < 0 {
				u0, u1 = -WallHalfWidth, -ribInner
			}
			b, err := r.box(f, u0, u1, rib.v0, rib.v0+ribHeight, rib.depth)
			if err != nil {
				return nil, fmt.Errorf("rib: %w", err)
			}
			if body, err = decad.Union(r.ctx, body, b); err != nil {
				return nil, fmt.Errorf("union rib: %w", err)
			}
		}
	}
	return body, nil
}

// roundRoot rounds the four root edges of one rib on the inner face at y = innerY with ribRound: side -1 or +1 is the
// sign of x, level 0 the ledge rib and 1 the top flange rib. The edges are the reflex edges of the body on that plane
// whose middle lies in the rib's footprint.
func roundRoot(ctx context.Context, body *decad.Body, side, level, innerY float64) (*decad.Body, error) {
	edges, err := decad.Edges(decad.Reflex()).SelectEdges(body)
	if err != nil {
		return nil, fmt.Errorf("select root edges: %w", err)
	}
	z0 := ledgeFrom
	if level == 1 {
		z0 = flangeFrom
	}
	var query *decad.EdgeQuery
	n := 0
	for _, e := range edges {
		a, b := e.Start().Position().Value, e.End().Position().Value
		if math.Abs(a.Y-innerY) > 1e-6 || math.Abs(b.Y-innerY) > 1e-6 {
			continue
		}
		mid := a.Add(b).Scale(0.5)
		if mid.X*side < ribInner-1e-6 || mid.X*side > WallHalfWidth+1e-6 || mid.Z < z0-1e-6 || mid.Z > z0+ribHeight+1e-6 {
			continue
		}
		var preds []decad.EdgePredicate
		switch e.Curve().(type) {
		case decad.Circle3, decad.Arc3:
			preds = []decad.EdgePredicate{decad.EndpointAt(a), decad.Circular()}
		default:
			preds = []decad.EdgePredicate{decad.EndpointAt(a), decad.ParallelTo(b.Sub(a))}
		}
		if query == nil {
			query = decad.Edges(preds...)
		} else {
			query = query.Or(preds...)
		}
		n++
	}
	if n != 4 {
		return nil, fmt.Errorf("rib root has %d edges, want 4", n)
	}
	out, err := decad.Fillet(ctx, body, query.Exactly(n), units.Millimeters(ribRound))
	if err != nil {
		return nil, fmt.Errorf("round the root: %w", err)
	}
	if !out.IsSolid() {
		return nil, fmt.Errorf("rounded wall is not a solid")
	}
	return out, nil
}

// slotLoop adds the guide slot of the station at y to s as one closed loop: the slot x +-3.75, y -2.05 to +2.05 with its
// 0.55 corner rounds and the pin notch x +-1.9 up to y + 4.25 with 0.5 rounds at its far corners. Every edge is held
// horizontal or vertical at its distance from the origin.
func slotLoop(s *sketch.Sketch, y float64) error {
	o := s.Origin()
	// The slot is the stem's section (7.0 by 3.6) with the Tight fit round it; the notch passes the groove pin: the stem's
	// narrow part (3.6) with 0.1 each side, and the pin's length with 0.4.
	sx, sy := 5.25+TightFit, StemThickness/2+TightFit
	nx, ny := 2.8, StemThickness/2+PinLength+0.4 // 2.8 is 2.7 + 0.1
	v := [][2]float64{{-sx, y - sy}, {sx, y - sy}, {sx, y + sy}, {nx, y + sy}, {nx, y + ny},
		{-nx, y + ny}, {-nx, y + sy}, {-sx, y + sy}}
	pts := make([]*sketch.Point, len(v))
	for i, q := range v {
		pts[i] = s.CreatePoint(q[0], q[1])
	}
	lines := make([]*sketch.Line, len(v))
	for i := range v {
		lines[i] = s.CreateLine(pts[i], pts[(i+1)%len(v)])
		if i%2 == 0 { // edges 0, 2, 4, 6 are horizontal
			s.AddConstraint(sketch.NewHorizontal(lines[i]), sketch.NewDistancePointLine(o, lines[i], math.Abs(v[i][1])))
		} else {
			s.AddConstraint(sketch.NewVertical(lines[i]), sketch.NewDistancePointLine(o, lines[i], math.Abs(v[i][0])))
		}
	}
	for _, c := range []struct {
		corner int
		radius float64
	}{{0, 0.55}, {1, 0.55}, {2, 0.55}, {4, 0.5}, {5, 0.5}, {7, 0.55}} {
		prev := (c.corner + len(v) - 1) % len(v)
		if _, err := sketch.CreateFillet(s, lines[prev], lines[c.corner], pts[c.corner], c.radius); err != nil {
			return fmt.Errorf("round slot corner %d: %w", c.corner, err)
		}
	}
	return nil
}

// plateBox is the guide and top plates' y range: between the walls' inner faces with a seam each side.
const (
	plateFrom = frontInnerY + Seam // -49.25
	plateTo   = rearInnerY - Seam  // 42.75
)

// guidePlate builds the guide plate: seven slots with pin notches, four bolt holes, outer-loop chamfers and nut pockets
// open to the top face at the G sites.
func (r *run) guidePlate() (*decad.Body, error) {
	s, err := r.sketchOn(xyFrame(guideFrom))
	if err != nil {
		return nil, err
	}
	rectOn(s, -WallHalfWidth, WallHalfWidth, plateFrom, plateTo)
	for _, seg := range Segments() {
		if err := slotLoop(s, r.dm.Station(seg)); err != nil {
			return nil, fmt.Errorf("slot %s: %w", seg, err)
		}
	}
	for _, site := range r.dm.Sites[4:8] {
		c := sketchcheck.DimensionPoint(s, site.Head.X, site.Head.Y)
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, m2fastener.HoleRadius), m2fastener.HoleRadius))
	}
	body, err := r.extrude(s, 11, guideThick, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	if body, _, err = edgefinish.Chamfer(r.ctx, body, r3.Vec{Z: 1}, []float64{guideFrom, guideTop}, edgefinish.Long(edgefinish.Outer),
		wallChamfer); err != nil {
		return nil, fmt.Errorf("chamfer outer loops: %w", err)
	}
	for i, site := range r.dm.Sites[4:8] {
		tool, sk, err := m2fastener.HexPocketTool(r.ctx, r.doc, site, m2fastener.NutCavityHeight+m2fastener.Stock)
		if err != nil {
			return nil, fmt.Errorf("nut pocket %d: %w", i+1, err)
		}
		r.sketches = append(r.sketches, sk)
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut nut pocket %d: %w", i+1, err)
		}
	}
	return body, nil
}
