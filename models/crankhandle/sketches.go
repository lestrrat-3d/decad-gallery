package crankhandle

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

// cylinder extrudes a circle of radius rad about (0, y) from z for length along +Z.
func (r *run) cylinder(y, z, rad, length float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(y, z))
	if err != nil {
		return nil, err
	}
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), rad), rad))
	return r.extrude(s, 0, length)
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

// armSketch sketches the arm on the back face: a stadium of radius ArmRadius about the origin and about the pin axis
// at (0, Throw), joined by two tangent lines, with the pin bolt's hole round the pin axis.
func (r *run) armSketch() (*sketch.Sketch, error) {
	s, err := r.sketchOn(across(0, 0))
	if err != nil {
		return nil, err
	}
	throw := r.p.Throw
	o := s.Origin()
	top := s.CreatePoint(0, throw)
	s.AddConstraint(sketch.NewVerticalPoints(o, top), sketch.NewVerticalDistance(o, top, throw))
	r1, l1 := s.CreatePoint(ArmRadius, 0), s.CreatePoint(-ArmRadius, 0)
	r2, l2 := s.CreatePoint(ArmRadius, throw), s.CreatePoint(-ArmRadius, throw)
	hubArc, pinArc := s.CreateArc(o, l1, r1), s.CreateArc(top, r2, l2)
	right, left := s.CreateLine(r1, r2), s.CreateLine(l2, l1)
	// Equal end radii make the line-arc joins exactly tangent, which the face chamfer needs.
	s.AddConstraint(sketch.NewRadius(hubArc, ArmRadius), sketch.NewEqualRadius(hubArc, pinArc),
		sketch.NewTangent(right, hubArc), sketch.NewTangent(right, pinArc),
		sketch.NewTangent(left, hubArc), sketch.NewTangent(left, pinArc))
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(top, PinHoleR), PinHoleR))
	return s, nil
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
