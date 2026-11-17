package turbinefan

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// run builds one fan in the local frame. It records every sketch.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// across is the plane z = const with U = +X and V = +Y, through the point (0, 0, z) of the local frame.
func across(z float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	return f
}

// newSketch starts a sketch on the plane f; extrusions run along f's normal U × V.
func newSketch(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	return s, nil
}

// sketchOn starts a sketch on the plane f and records it.
func (r *run) sketchOn(f r3.Frame) (*sketch.Sketch, error) {
	s, err := newSketch(f)
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, s)
	return s, nil
}

// profile solves s, requires it fully constrained and returns its one valid profile with the given number of holes.
func (r *run) profile(s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(r.ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(r.ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != holes {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("several valid profiles with %d holes", holes)
		}
		chosen = pr
	}
	if chosen == nil {
		return nil, fmt.Errorf("no valid profile with %d holes", holes)
	}
	return chosen, nil
}

// extrudeOutline extrudes the one valid profile of s with the given holes by length along its normal.
func (r *run) extrudeOutline(s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	pr, err := r.profile(s, holes)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(r.ctx, r.doc, s, pr, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
}

// hubSketch sketches the hub's outline on the back face: the circle of radius Size.HubRadius, the D bore (an arc of radius
// r_b about the axis the long way round through +Y, closed by its flat at y = -f_b) and three circles of radius
// m2fastener.HoleRadius for the blade bolts. The bore runs through, so the profile has four holes.
func (r *run) hubSketch() (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(0))
	if err != nil {
		return nil, err
	}
	dm := r.dm
	o := s.Origin()
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(o, dm.Size.HubRadius), dm.Size.HubRadius))
	w := dm.BoreFlatWidth / 2
	qR, qL := s.CreatePoint(w, -dm.BoreFlat), s.CreatePoint(-w, -dm.BoreFlat)
	bore := s.CreateArc(o, qR, qL)
	flat := s.CreateLine(qL, qR)
	s.AddConstraint(sketch.NewRadius(bore, dm.BoreRadius), sketch.NewHorizontal(flat),
		sketch.NewDistancePointLine(o, flat, dm.BoreFlat))
	for k := 1; k <= BladeCount; k++ {
		u, v := dm.Size.BladeBoltAt*math.Cos(BladeAngle(k)), dm.Size.BladeBoltAt*math.Sin(BladeAngle(k))
		c := s.CreatePoint(u, v)
		s.AddConstraint(sketch.NewHorizontalDistance(o, c, u), sketch.NewVerticalDistance(o, c, v))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, m2fastener.HoleRadius), m2fastener.HoleRadius))
	}
	return s, nil
}

// nutSlotSketch sketches a hex nut slot on the plane f: a hexagon about f's origin with its corners on the circle of
// radius m2fastener.NutCavityRadius, one corner at -U, whose two sides toward +U run on as straight walls at
// v = ±NutCavityRadius*sqrt(3)/2 out to u = reach. Extruded along f's normal, it is a slot that opens where u = reach
// lies outside the part. Each side is held by its distance from the origin, and each next side of the hexagon turns 60
// degrees from the one before, as m2fastener.HexPocketTool draws a hexagon.
func (r *run) nutSlotSketch(f r3.Frame, reach float64) (*sketch.Sketch, error) {
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	rad := m2fastener.NutCavityRadius
	h := rad * math.Sqrt(3) / 2
	a, b := s.CreatePoint(reach, -h), s.CreatePoint(-rad/2, -h)
	c, d, e := s.CreatePoint(-rad, 0), s.CreatePoint(-rad/2, h), s.CreatePoint(reach, h)
	// The lines run counter-clockwise round the hexagon (D to C, C to B, B to A), each turning 60 degrees from the one before.
	cd, bc, ab := s.CreateLine(d, c), s.CreateLine(c, b), s.CreateLine(b, a)
	de, ea := s.CreateLine(d, e), s.CreateLine(e, a)
	o := s.Origin()
	s.AddConstraint(
		sketch.NewHorizontal(ab), sketch.NewHorizontal(de), sketch.NewVertical(ea),
		sketch.NewDistancePointLine(o, ab, h), sketch.NewDistancePointLine(o, bc, h),
		sketch.NewDistancePointLine(o, cd, h), sketch.NewDistancePointLine(o, de, h),
		sketch.NewDistancePointLine(o, ea, reach),
		sketch.NewAngle(cd, bc, 60), sketch.NewAngle(bc, ab, 60),
	)
	return s, nil
}

// counterboreTool is the set screw's counterbore: a revolved meridian about the screw's line, from the floor at rho_f a
// quarter round of radius CounterboreRound, drawn in eight chords, out to the head pocket's radius, and on out through
// the round face. The plane's U runs along the screw's line outward from the floor and V is the radius.
func (r *run) counterboreTool() (*decad.Body, error) {
	dm := r.dm
	origin := r3.Vec{Y: -dm.CounterboreFloor, Z: SetScrewZ}
	f, err := r3.NewFrame(origin, r3.Vec{Y: -1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	rr, rc := CounterboreRound, m2fastener.HeadPocketRadius
	vertices := [][2]float64{{0, 0}}
	for i := 0; i <= 8; i++ {
		t := math.Pi - (math.Pi/2)*float64(i)/8
		vertices = append(vertices, [2]float64{rr + rr*math.Cos(t), rc - rr + rr*math.Sin(t)})
	}
	out := dm.Size.HubRadius + m2fastener.Stock - dm.CounterboreFloor
	vertices = append(vertices, [2]float64{out, rc}, [2]float64{out, 0})
	points := make([]*sketch.Point, len(vertices))
	for i, v := range vertices {
		points[i] = sketchcheck.DimensionPoint(s, v[0], v[1])
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(r.ctx, r.doc, s, pr, decad.ConstructionAxis{Origin: origin, Dir: r3.Vec{Y: -1}},
		decad.FullRevolution{})
}

// setScrewSlotTool is the set screw's nut slot: a hex slot from y = -(f_b + PressWall) to -(f_b + PressWall + 2), open to
// the back face, with every edge of both slot faces rounded by NutSlotRound.
func (r *run) setScrewSlotTool() (*decad.Body, error) {
	f, err := r3.NewFrame(r3.Vec{Y: -(r.dm.BoreFlat + PressWall), Z: SetScrewZ}, r3.Vec{Z: -1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.nutSlotSketch(f, SetScrewZ+m2fastener.Stock)
	if err != nil {
		return nil, err
	}
	tool, err := r.extrudeOutline(s, 0, m2fastener.NutCavityHeight)
	if err != nil {
		return nil, err
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(tool))).Or(decad.OuterLoopOf(decad.CapEnd(tool)))
	if tool, err = decad.Fillet(r.ctx, tool, loops, units.Millimeters(NutSlotRound)); err != nil {
		return nil, fmt.Errorf("round slot faces: %w", err)
	}
	return tool, nil
}

// slotTool is a nut slot for site: a hex slot from the cavity's far face up to its near face, along u (the radial line
// outward from the axis through the site's head) from the corner at 6.99 behind the bolt line out to reach beyond it,
// with its three hexagon corners rounded by NutSlotRound. The slot's mouth opens where reach lies outside the part,
// through the hub's rim. A site with Dir -Z (a blade bolt) puts the slot's far face CavityFar behind its head; a site
// with Dir +Z (an adapter bolt) puts the slot Cavity in front of its head.
func (r *run) slotTool(site m2fastener.Site, u r3.Vec, reach float64) (*decad.Body, error) {
	v := r3.Vec{X: -u.Y, Y: u.X}
	z := site.Head.Z - site.CavityFar()
	if site.Dir.Z > 0 {
		z = site.Head.Z + site.Cavity
	}
	f, err := r3.NewFrame(r3.Vec{X: site.Head.X, Y: site.Head.Y, Z: z}, u, v)
	if err != nil {
		return nil, err
	}
	s, err := r.nutSlotSketch(f, reach)
	if err != nil {
		return nil, err
	}
	tool, err := r.extrudeOutline(s, 0, m2fastener.NutCavityHeight)
	if err != nil {
		return nil, err
	}
	// The hexagon's three corners B, C and D each turn 60 degrees, so their edges along Z are rounded (a layer reads each
	// as an inside corner). Decad matches a stored position exactly, so find each edge's stored vertex.
	rad := m2fastener.NutCavityRadius
	h := rad * math.Sqrt(3) / 2
	var query *decad.EdgeQuery
	for _, c := range [][2]float64{{-rad / 2, -h}, {-rad, 0}, {-rad / 2, h}} {
		corner := r3.Vec{X: site.Head.X + c[0]*u.X + c[1]*v.X, Y: site.Head.Y + c[0]*u.Y + c[1]*v.Y}
		var at r3.Vec
		found := 0
		for _, e := range tool.Edges() {
			a, b := e.Start().Position().Value, e.End().Position().Value
			if math.Abs(a.X-b.X) > 1e-6 || math.Abs(a.Y-b.Y) > 1e-6 || math.Abs(a.Z-b.Z) < 1 {
				continue
			}
			if math.Hypot(a.X-corner.X, a.Y-corner.Y) > 1e-6 {
				continue
			}
			at, found = a, found+1
		}
		if found != 1 {
			return nil, fmt.Errorf("nut slot: found %d corner edges at %v, want 1", found, corner)
		}
		preds := []decad.EdgePredicate{decad.ParallelTo(r3.Vec{Z: 1}), decad.EndpointAt(at)}
		if query == nil {
			query = decad.Edges(preds...)
		} else {
			query = query.Or(preds...)
		}
	}
	if tool, err = decad.Fillet(r.ctx, tool, query.Exactly(3), units.Millimeters(NutSlotRound)); err != nil {
		return nil, fmt.Errorf("round slot corners: %w", err)
	}
	return tool, nil
}
