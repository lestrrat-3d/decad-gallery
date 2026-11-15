package crankhandle

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

// run builds one crank in the local frame. It records every sketch and keeps the first error.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// sketchOn starts a sketch on the plane f; extrusions run along f's normal U × V.
func (r *run) sketchOn(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
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

// extrude extrudes the profile of s with the given holes by length along its normal.
func (r *run) extrude(s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	pr, err := r.profile(s, holes)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(r.ctx, r.doc, s, pr, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
}

// across is the plane z = const through the point (0, y) of the local frame, with U = +X and V = +Y.
func across(y, z float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Y: y, Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	return f
}

// hexagon extrudes a regular hexagon flats across, its corners on X, about (0, y) from z for length along +Z.
func (r *run) hexagon(y, z, flats, length float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(y, z))
	if err != nil {
		return nil, err
	}
	radius := flats / math.Sqrt(3)
	poly, err := s.CreatePolygon(0, 0, 6, radius)
	if err != nil {
		return nil, fmt.Errorf("create hexagon: %w", err)
	}
	s.AddConstraint(
		sketch.NewCoincident(poly.Center, s.Origin()),
		sketch.NewHorizontal(poly.Sides[1]),
		sketch.NewDistance(poly.Center, poly.Vertices[0], radius),
	)
	return r.extrude(s, 0, length)
}

// armSketch sketches the arm on the back face: an arc of radius ArmRadius about the pin axis at (0, Throw), the two
// straight sides at x = ±ArmRadius tangent to it, which run down to y = SideEnd and turn into ClampCornerR corner arcs,
// and the ClampEndR end arc tangent to both corners from inside, with the pin bolt's hole round the pin axis. Every join
// of the outline is tangent, so it has no inside corner and no step.
func (r *run) armSketch() (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(0, 0))
	if err != nil {
		return nil, err
	}
	throw, dm := r.p.Throw, r.dm
	o := s.Origin()
	top := s.CreatePoint(0, throw)
	s.AddConstraint(sketch.NewVerticalPoints(o, top), sketch.NewVerticalDistance(o, top, throw))
	r2, l2 := s.CreatePoint(ArmRadius, throw), s.CreatePoint(-ArmRadius, throw)
	sR, sL := s.CreatePoint(ArmRadius, dm.SideEnd), s.CreatePoint(-ArmRadius, dm.SideEnd)
	pinArc := s.CreateArc(top, r2, l2)
	right, left := s.CreateLine(sR, r2), s.CreateLine(l2, sL)
	s.AddConstraint(sketch.NewRadius(pinArc, ArmRadius), sketch.NewTangent(right, pinArc), sketch.NewTangent(left, pinArc),
		sketch.NewVertical(right), sketch.NewVertical(left),
		sketch.NewVerticalDistance(o, sR, dm.SideEnd), sketch.NewVerticalDistance(o, sL, dm.SideEnd))
	cR, cB := dm.CornerCenter(), dm.EndCenter()
	cL := r3.Vec{X: -cR.X, Y: cR.Y}
	// The corner arc meets the end arc on the line through their centres, ClampEndR from the end arc's centre.
	pR := cB.Add(cR.Sub(cB).Scale(ClampEndR / cR.Sub(cB).Len()))
	pL := r3.Vec{X: -pR.X, Y: pR.Y}
	cRp, cLp, cBp := s.CreatePoint(cR.X, cR.Y), s.CreatePoint(cL.X, cL.Y), s.CreatePoint(cB.X, cB.Y)
	pRp, pLp := s.CreatePoint(pR.X, pR.Y), s.CreatePoint(pL.X, pL.Y)
	arcR := s.CreateArc(cRp, pRp, sR) // counter-clockwise from the end arc up to the right side
	arcL := s.CreateArc(cLp, sL, pLp) // from the left side down to the end arc
	arcB := s.CreateArc(cBp, pLp, pRp)
	s.AddConstraint(
		sketch.NewRadius(arcR, ClampCornerR), sketch.NewTangent(right, arcR),
		sketch.NewRadius(arcL, ClampCornerR), sketch.NewTangent(left, arcL),
		sketch.NewRadius(arcB, ClampEndR), sketch.NewTangentCircles(arcB, arcR, true),
		sketch.NewTangentCircles(arcB, arcL, true),
	)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, PinHoleR), PinHoleR))
	return s, nil
}

// slitTool extrudes the slit's cutter from z = -1 through the arm: x within ±SlitWidth/2 from 2 mm past the end arc up
// to SlitInBore past the bore's wall, closed there by a half circle of radius SlitWidth/2. The round end lies inside the
// bore's outline, so it shows only in the bore's floor.
func (r *run) slitTool() (*decad.Body, error) {
	s, err := r.sketchOn(across(0, -1))
	if err != nil {
		return nil, err
	}
	half := SlitWidth / 2
	y0, y1 := -r.dm.Reach-2, -r.dm.BoreRadius+SlitInBore
	yc := y1 - half
	a := sketchcheck.DimensionPoint(s, half, y0)
	// The arc's ends are placed by height: placing their x at ±half would meet the circle at a tangent, a singular solve.
	b, c := s.CreatePoint(half, yc), s.CreatePoint(-half, yc)
	s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), b, yc), sketch.NewVerticalDistance(s.Origin(), c, yc))
	d := sketchcheck.DimensionPoint(s, -half, y0)
	center := sketchcheck.DimensionPoint(s, 0, yc)
	arc := s.CreateArc(center, b, c)
	s.AddConstraint(sketch.NewRadius(arc, half))
	s.CreateLine(a, b)
	s.CreateLine(c, d)
	s.CreateLine(d, a)
	return r.extrude(s, 0, ArmThickness+2)
}

// clampPocketTool extrudes the clamp nut's hexagonal pocket from its floor, the site's Cavity, out through the -X side
// face: NutCavityHeight plus m2fastener.Stock along the bolt. The hexagon is centred on the bolt's axis with its corners
// on the circle of radius m2fastener.NutCavityRadius along Corner (±Y), so two flats lie parallel to the print bed. Each
// side is held by its distance from the axis, the first flat is horizontal and each next side turns 60 degrees from
// the one before, so the corners lie where the sides meet. The corners stay sharp, matching the nut's sharp corners.
// The sketch lies on site.Across(Cavity): U is Corner (+Y) and V is Dir × Corner (-Z), so it extrudes along -X.
func (r *run) clampPocketTool() (*decad.Body, error) {
	site := r.dm.ClampSite
	f, err := site.Across(site.Cavity)
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	rad := m2fastener.NutCavityRadius
	flat := rad * math.Sqrt(3) / 2
	var corners [6]*sketch.Point
	for i := range corners {
		a := float64(i) * math.Pi / 3
		corners[i] = s.CreatePoint(rad*math.Cos(a), rad*math.Sin(a))
	}
	var sides [6]*sketch.Line
	for i := range sides {
		sides[i] = s.CreateLine(corners[i], corners[(i+1)%6])
		s.AddConstraint(sketch.NewDistancePointLine(s.Origin(), sides[i], flat))
	}
	// sides[1] runs from the corner at 60 degrees to the one at 120 degrees: a flat.
	s.AddConstraint(sketch.NewHorizontal(sides[1]))
	for i := range 5 {
		s.AddConstraint(sketch.NewAngle(sides[i], sides[i+1], 60))
	}
	return r.extrude(s, 0, m2fastener.NutCavityHeight+m2fastener.Stock)
}

// boreTool extrudes the D bore's cutter from 1 before the back face to the bore's floor: an arc of radius r_b the
// long way round through -Y, closed by its flat at y = +f_b, toward the pin.
func (r *run) boreTool() (*decad.Body, error) {
	s, err := r.sketchOn(across(0, -1))
	if err != nil {
		return nil, err
	}
	rb, fb := r.dm.BoreRadius, r.dm.BoreFlat
	w := math.Sqrt(rb*rb - fb*fb)
	o := s.Origin()
	q1, q2 := s.CreatePoint(-w, fb), s.CreatePoint(w, fb)
	bore := s.CreateArc(o, q1, q2)
	flat := s.CreateLine(q2, q1)
	s.AddConstraint(sketch.NewRadius(bore, rb), sketch.NewHorizontal(flat), sketch.NewDistancePointLine(o, flat, fb))
	return r.extrude(s, 0, 1+ArmThickness-BoreFloor)
}

// handle revolves HandleProfile about the pin axis. The sketch plane holds the axis: U runs along it (+Z) and V is the
// radius (+X), so every vertex is dimensioned from the origin at (0, Throw, HandleFrom).
func (r *run) handle() (*decad.Body, error) {
	origin := r3.Vec{Y: r.p.Throw, Z: HandleFrom}
	f, err := r3.NewFrame(origin, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	points := make([]*sketch.Point, len(HandleProfile))
	for i, pt := range HandleProfile {
		points[i] = sketchcheck.DimensionPoint(s, pt[0], pt[1])
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(r.ctx, r.doc, s, pr, decad.ConstructionAxis{Origin: origin, Dir: r3.Vec{Z: 1}},
		decad.FullRevolution{})
}

// turnedPin revolves the pin's turned part about the pin axis at (0, y): a cone of half angle ConeAngle from radius
// RootR at ShankFrom out to the shank (radius ShankR), the shank to ShankTo, a cone out to the head (radius HeadR) at
// HeadFrom, and the head to HeadFrom + HeadLength. The sketch plane holds the axis: U runs along it (+Z) and V is the
// radius (+X), so every vertex is dimensioned from the origin at (0, y, 0).
func (r *run) turnedPin(y float64) (*decad.Body, error) {
	origin := r3.Vec{Y: y}
	f, err := r3.NewFrame(origin, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	slope := math.Tan(ConeAngle * math.Pi / 180)
	top := HeadFrom + HeadLength
	meridian := [][2]float64{
		{ShankFrom, 0}, {ShankFrom, RootR}, {ShankFrom + (ShankR-RootR)/slope, ShankR}, {ShankTo, ShankR},
		{HeadFrom, HeadR}, {top, HeadR}, {top, 0},
	}
	points := make([]*sketch.Point, len(meridian))
	for i, pt := range meridian {
		points[i] = sketchcheck.DimensionPoint(s, pt[0], pt[1])
	}
	for i := range points {
		s.CreateLine(points[i], points[(i+1)%len(points)])
	}
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(r.ctx, r.doc, s, pr, decad.ConstructionAxis{Origin: origin, Dir: r3.Vec{Z: 1}},
		decad.FullRevolution{})
}

// flatTool cuts one of a cross bolt's double flats: everything beyond CrossFlat from the pin axis at (0, y) on the
// side (+1 for +Y, -1 for -Y), over CrossFlatHalf either side of z. Its sketch lies on the plane across the pin at the
// flat's lower end, with its origin on the pin axis.
func (r *run) flatTool(y, z, side float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(y, z-CrossFlatHalf))
	if err != nil {
		return nil, err
	}
	const reach = 3.0 // past the shank's surface
	half := ShankR + 1
	v0, v1 := CrossFlat, CrossFlat+reach
	if side < 0 {
		v0, v1 = -v1, -v0
	}
	rect := s.CreateRectangle(-half, v0, half, v1)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), rect.A, -half),
		sketch.NewVerticalDistance(s.Origin(), rect.A, v0),
		sketch.NewHorizontalDistance(rect.A, rect.B, 2*half),
		sketch.NewVerticalDistance(rect.B, rect.C, v1-v0),
	)
	return r.extrude(s, 0, 2*CrossFlatHalf)
}
