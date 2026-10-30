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
	world    *sketch.World
	scale    float64
	assembly *Assembly
}

type sectionHole struct{ x, y, diameter float64 }

func (b *construction) newSketch(name string, plane *sketch.Plane) (*sketch.Sketch, error) {
	s, err := b.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: create sketch: %w", name, err)
	}
	b.assembly.Sketches = append(b.assembly.Sketches, s)
	return s, nil
}

func (b *construction) circle(s *sketch.Sketch, h sectionHole) {
	center := s.Origin()
	if h.x != 0 || h.y != 0 {
		center = sketchcheck.DimensionPoint(s, h.x*b.scale, h.y*b.scale)
	}
	c := s.CreateCircle(center, h.diameter*b.scale/2)
	s.AddConstraint(sketch.NewDiameter(c, h.diameter*b.scale))
}

func (b *construction) rectangle(name string, x1, y1, x2, y2, length float64, holes []sectionHole) (*decad.Body, error) {
	s, err := b.newSketch(name, b.world.XY())
	if err != nil {
		return nil, err
	}
	k := b.scale
	r := s.CreateRectangle(x1*k, y1*k, x2*k, y2*k)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, x1*k),
		sketch.NewVerticalDistance(s.Origin(), r.A, y1*k),
		sketch.NewHorizontalDistance(r.A, r.B, (x2-x1)*k),
		sketch.NewVerticalDistance(r.B, r.C, (y2-y1)*k),
	)
	for _, h := range holes {
		b.circle(s, h)
	}
	return b.extrude(name, s, length, len(holes))
}

func (b *construction) round(name string, outer, inner, length float64) (*decad.Body, error) {
	s, err := b.newSketch(name, b.world.XY())
	if err != nil {
		return nil, err
	}
	c := s.CreateCircle(s.Origin(), outer*b.scale/2)
	s.AddConstraint(sketch.NewRadius(c, outer*b.scale/2))
	holes := 0
	if inner > 0 {
		b.circle(s, sectionHole{diameter: inner})
		holes = 1
	}
	return b.extrude(name, s, length, holes)
}

func (b *construction) profile(name string, s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(b.ctx); err != nil {
		return nil, fmt.Errorf("%s: solve sketch: %w", name, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify sketch: %w", name, err)
	}
	var chosen *sketch.Profile
	count := 0
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == holes {
			chosen = p
			count++
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("%s: select profile: got %d valid profiles with %d holes", name, count, holes)
	}
	return chosen, nil
}

func (b *construction) extrude(name string, s *sketch.Sketch, length float64, holes int) (*decad.Body, error) {
	p, err := b.profile(name, s, holes)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Distance{D: units.Millimeters(length * b.scale), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", name, err)
	}
	return body, nil
}

func (b *construction) place(name, group string, body *decad.Body, origin, u, v r3.Vec) error {
	f, err := r3.NewFrame(origin, u, v)
	if err != nil {
		return fmt.Errorf("%s: placement frame: %w", name, err)
	}
	t, err := r3.FromFrame(f)
	if err != nil {
		return fmt.Errorf("%s: placement transform: %w", name, err)
	}
	body, err = decad.Transform(b.ctx, body, t)
	if err != nil {
		return fmt.Errorf("%s: transform: %w", name, err)
	}
	if !body.IsSolid() {
		return fmt.Errorf("%s: transformed body is not solid", name)
	}
	b.assembly.Parts = append(b.assembly.Parts, Part{Name: name, Group: group, Body: body})
	return nil
}

func (b *construction) placeXZ(name, group string, body *decad.Body, x, y, z, angle float64) error {
	c, s := math.Cos(angle), math.Sin(angle)
	return b.place(name, group, body, r3.NewVec(x*b.scale, y*b.scale, z*b.scale), r3.NewVec(c, 0, s), r3.NewVec(-s, 0, c))
}

func (b *construction) box(name, group string, x1, x2, y1, y2, z1, z2 float64) error {
	body, err := b.rectangle(name, 0, 0, x2-x1, y2-y1, z2-z1, nil)
	if err != nil {
		return err
	}
	return b.place(name, group, body, r3.NewVec(x1*b.scale, y1*b.scale, z1*b.scale), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

func (b *construction) roundXZ(name, group string, outer, inner, x, z, y1, y2 float64) error {
	body, err := b.round(name, outer, inner, y2-y1)
	if err != nil {
		return err
	}
	return b.placeXZ(name, group, body, x, y2, z, 0)
}

func (b *construction) rectangleXZ(name, group string, x1, z1, x2, z2, x, y, z, angle, length float64, holes []sectionHole) error {
	body, err := b.rectangle(name, x1, z1, x2, z2, length, holes)
	if err != nil {
		return err
	}
	return b.placeXZ(name, group, body, x, y, z, angle)
}

func (b *construction) spindle() error {
	const name = "spindle"
	s, err := b.newSketch(name, b.world.XZ())
	if err != nil {
		return err
	}
	k := b.scale
	p := []*sketch.Point{s.Origin(), s.CreatePoint(5*k, 0), s.CreatePoint(5*k, 275*k), s.CreatePoint(.6*k, 300*k), s.CreatePoint(0, 300*k)}
	lines := make([]*sketch.Line, 5)
	for i := range lines {
		lines[i] = s.CreateLine(p[i], p[(i+1)%len(p)])
	}
	s.AddConstraint(sketch.NewHorizontal(lines[0]), sketch.NewVertical(lines[1]),
		sketch.NewHorizontal(lines[3]), sketch.NewVertical(lines[4]),
		sketch.NewHorizontalDistance(p[0], p[1], 5*k), sketch.NewVerticalDistance(p[1], p[2], 275*k),
		sketch.NewHorizontalDistance(p[0], p[3], .6*k), sketch.NewVerticalDistance(p[0], p[3], 300*k))
	profile, err := b.profile(name, s, 0)
	if err != nil {
		return err
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, profile, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.NewVec(0, 0, 1)}, decad.FullRevolution{})
	if err != nil {
		return fmt.Errorf("%s: revolve: %w", name, err)
	}
	return b.placeXZ(name, "steel", body, 12.3, 30, 482.3, 0)
}
