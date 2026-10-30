package pedalspindle

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

type construction struct {
	ctx      context.Context
	doc      *decad.Document
	assembly *Assembly
}

type circularHole struct {
	x, y, diameter float64
}

func newSection() (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	return w.CreateSketch(w.XY())
}

func addHole(s *sketch.Sketch, hole circularHole) {
	center := s.Origin()
	if hole.x != 0 || hole.y != 0 {
		center = sketchcheck.DimensionPoint(s, hole.x, hole.y)
	}
	c := s.CreateCircle(center, hole.diameter/2)
	s.AddConstraint(sketch.NewDiameter(c, hole.diameter))
}

func (b *construction) sectionProfile(s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(b.ctx); err != nil {
		return nil, fmt.Errorf("solve section: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		return nil, fmt.Errorf("verify section: %w", err)
	}
	var selected *sketch.Profile
	for _, profile := range s.Profiles() {
		if !profile.Valid || len(profile.Holes) != holes {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("select section: multiple valid profiles with %d holes", holes)
		}
		selected = profile
	}
	if selected == nil {
		return nil, fmt.Errorf("select section: no valid profile with %d holes", holes)
	}
	b.assembly.Sketches = append(b.assembly.Sketches, s)
	return selected, nil
}

func (b *construction) extrude(s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	profile, err := b.sectionProfile(s, holes)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, profile,
		decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude section: %w", err)
	}
	return body, nil
}

func (b *construction) rectangle(x1, y1, x2, y2, length float64, holes []circularHole) (*decad.Body, error) {
	s, err := newSection()
	if err != nil {
		return nil, fmt.Errorf("create rectangle sketch: %w", err)
	}
	r := s.CreateRectangle(x1, y1, x2, y2)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, x1),
		sketch.NewVerticalDistance(s.Origin(), r.A, y1),
		sketch.NewHorizontalDistance(r.A, r.B, x2-x1),
		sketch.NewVerticalDistance(r.B, r.C, y2-y1),
	)
	for _, hole := range holes {
		addHole(s, hole)
	}
	return b.extrude(s, len(holes), length)
}

func (b *construction) ring(outerRadius, innerDiameter, length float64) (*decad.Body, error) {
	s, err := newSection()
	if err != nil {
		return nil, fmt.Errorf("create circular sketch: %w", err)
	}
	outer := s.CreateCircle(s.Origin(), outerRadius)
	s.AddConstraint(sketch.NewRadius(outer, outerRadius))
	holes := 0
	if innerDiameter > 0 {
		addHole(s, circularHole{diameter: innerDiameter})
		holes = 1
	}
	return b.extrude(s, holes, length)
}

func xzFrame(origin r3.Vec, angle float64) (r3.Frame, error) {
	c, s := math.Cos(angle), math.Sin(angle)
	return r3.NewFrame(origin, r3.NewVec(c, 0, s), r3.NewVec(-s, 0, c))
}

func (b *construction) place(body *decad.Body, frame r3.Frame) (*decad.Body, error) {
	t, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create placement transform: %w", err)
	}
	placed, err := decad.Transform(b.ctx, body, t)
	if err != nil {
		return nil, fmt.Errorf("transform body: %w", err)
	}
	return placed, nil
}

func (b *construction) appendPart(name, group string, body *decad.Body, err error, frame r3.Frame) error {
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: %w", name, err)
	}
	placed, err := b.place(body, frame)
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: %w", name, err)
	}
	if !placed.IsSolid() {
		return fmt.Errorf("pedalspindle: %s: placed body is not solid", name)
	}
	b.assembly.Parts = append(b.assembly.Parts, Part{Name: name, Group: group, Body: placed})
	return nil
}

func (b *construction) box(name, group string, x1, x2, y1, y2, z1, z2 float64) error {
	frame, err := r3.NewFrame(r3.NewVec(x1, y1, z1), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create box frame: %w", name, err)
	}
	body, err := b.rectangle(0, 0, x2-x1, y2-y1, z2-z1, nil)
	return b.appendPart(name, group, body, err, frame)
}

func (b *construction) xzRectangle(name, group string, origin r3.Vec, angle,
	x1, y1, x2, y2, length float64, holes []circularHole) error {
	frame, err := xzFrame(origin, angle)
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create rectangle frame: %w", name, err)
	}
	body, err := b.rectangle(x1, y1, x2, y2, length, holes)
	return b.appendPart(name, group, body, err, frame)
}

func (b *construction) xzRing(name, group string, center r3.Vec, backY, outerRadius,
	innerDiameter, length float64) error {
	frame, err := xzFrame(r3.NewVec(center.X, backY, center.Z), 0)
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create circular frame: %w", name, err)
	}
	body, err := b.ring(outerRadius, innerDiameter, length)
	return b.appendPart(name, group, body, err, frame)
}

func (b *construction) spindle(scale float64, center r3.Vec) error {
	w := sketch.NewWorld()
	s, err := w.CreateSketch(w.XZ())
	if err != nil {
		return fmt.Errorf("pedalspindle: spindle: create axial sketch: %w", err)
	}
	points := []*sketch.Point{
		s.Origin(), s.CreatePoint(2.5, 0), s.CreatePoint(2.5, 275*scale),
		s.CreatePoint(0.3, 300*scale), s.CreatePoint(0, 300*scale),
	}
	lines := make([]*sketch.Line, len(points))
	for i, point := range points {
		lines[i] = s.CreateLine(point, points[(i+1)%len(points)])
	}
	s.AddConstraint(
		sketch.NewHorizontal(lines[0]), sketch.NewVertical(lines[1]),
		sketch.NewHorizontal(lines[3]), sketch.NewVertical(lines[4]),
		sketch.NewHorizontalDistance(points[0], points[1], 2.5),
		sketch.NewVerticalDistance(points[1], points[2], 275*scale),
		sketch.NewHorizontalDistance(points[0], points[3], 0.3),
		sketch.NewVerticalDistance(points[0], points[3], 300*scale),
	)
	profile, err := b.sectionProfile(s, 0)
	if err != nil {
		return fmt.Errorf("pedalspindle: spindle: %w", err)
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, profile,
		decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.NewVec(0, 0, 1)}, decad.FullRevolution{})
	if err != nil {
		return fmt.Errorf("pedalspindle: spindle: revolve axial profile: %w", err)
	}
	frame, err := xzFrame(r3.NewVec(center.X, 30*scale, center.Z), 0)
	if err != nil {
		return fmt.Errorf("pedalspindle: spindle: create axial frame: %w", err)
	}
	return b.appendPart("spindle", "steel", body, nil, frame)
}
