package crankhandle

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// run builds one crank in the local frame. It records every sketch and keeps the first error.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// sketchOn starts a sketch on the plane f and records it; extrusions run along f's normal U × V.
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
	return selectProfile(r.ctx, s, holes)
}

// extrude extrudes the profile of s with the given holes by length along its normal.
func (r *run) extrude(s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	return extrudeProfile(r.ctx, r.doc, s, holes, length)
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
	SketchTail(s, right, left, sR, sL, ArmRadius, dm.SideEnd)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, PinHoleR), PinHoleR))
	return s, nil
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
