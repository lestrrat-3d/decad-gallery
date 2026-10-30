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

type hole struct{ x, y, diameter float64 }

func (c *construction) section(name string, axial bool) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane := w.XY()
	if axial {
		plane = w.XZ()
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("pedalspindle: %s: create sketch: %w", name, err)
	}
	c.assembly.Sketches = append(c.assembly.Sketches, s)
	return s, nil
}

func (c *construction) profile(name string, s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(c.ctx); err != nil {
		return nil, fmt.Errorf("pedalspindle: %s: solve sketch: %w", name, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(c.ctx, s); err != nil {
		return nil, fmt.Errorf("pedalspindle: %s: verify sketch: %w", name, err)
	}
	var selected *sketch.Profile
	for _, p := range s.Profiles() {
		if !p.Valid || len(p.Holes) != holes {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("pedalspindle: %s: multiple valid profiles with %d holes", name, holes)
		}
		selected = p
	}
	if selected == nil {
		return nil, fmt.Errorf("pedalspindle: %s: no valid profile with %d holes", name, holes)
	}
	return selected, nil
}

func addHoles(s *sketch.Sketch, holes []hole) {
	for _, h := range holes {
		center := s.Origin()
		if h.x != 0 || h.y != 0 {
			center = sketchcheck.DimensionPoint(s, h.x, h.y)
		}
		circle := s.CreateCircle(center, h.diameter/2)
		s.AddConstraint(sketch.NewDiameter(circle, h.diameter))
	}
}

func (c *construction) extrude(name string, s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	p, err := c.profile(name, s, holes)
	if err != nil {
		return nil, err
	}
	b, err := decad.Extrude(c.ctx, c.doc, s, p, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("pedalspindle: %s: extrude section: %w", name, err)
	}
	return b, nil
}

func (c *construction) rectangle(name string, x1, y1, x2, y2, length float64, holes []hole) (*decad.Body, error) {
	s, err := c.section(name, false)
	if err != nil {
		return nil, err
	}
	r := s.CreateRectangle(x1, y1, x2, y2)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, x1),
		sketch.NewVerticalDistance(s.Origin(), r.A, y1),
		sketch.NewHorizontalDistance(r.A, r.B, x2-x1),
		sketch.NewVerticalDistance(r.B, r.C, y2-y1),
	)
	addHoles(s, holes)
	return c.extrude(name, s, len(holes), length)
}

func (c *construction) round(name string, radius, bore, length float64) (*decad.Body, error) {
	s, err := c.section(name, false)
	if err != nil {
		return nil, err
	}
	outer := s.CreateCircle(s.Origin(), radius)
	s.AddConstraint(sketch.NewRadius(outer, radius))
	holes := 0
	if bore > 0 {
		addHoles(s, []hole{{diameter: bore}})
		holes = 1
	}
	return c.extrude(name, s, holes, length)
}

func xzFrame(origin r3.Vec, angle float64) (r3.Frame, error) {
	return r3.NewFrame(origin, r3.NewVec(math.Cos(angle), 0, math.Sin(angle)),
		r3.NewVec(-math.Sin(angle), 0, math.Cos(angle)))
}

func (c *construction) place(name string, body *decad.Body, frame r3.Frame) (*decad.Body, error) {
	t, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("pedalspindle: %s: create transform: %w", name, err)
	}
	body, err = decad.Transform(c.ctx, body, t)
	if err != nil {
		return nil, fmt.Errorf("pedalspindle: %s: transform body: %w", name, err)
	}
	return body, nil
}

func (c *construction) append(name, group string, body *decad.Body) error {
	if !body.IsSolid() {
		return fmt.Errorf("pedalspindle: %s: body is not solid", name)
	}
	for _, p := range c.assembly.Parts {
		if p.Name == name {
			return fmt.Errorf("pedalspindle: %s: duplicate part name", name)
		}
	}
	c.assembly.Parts = append(c.assembly.Parts, Part{Name: name, Group: group, Body: body})
	return nil
}

func (c *construction) box(name, group string, x1, x2, y1, y2, z1, z2, scale float64) error {
	b, err := c.rectangle(name, 0, 0, (x2-x1)*scale, (y2-y1)*scale, (z2-z1)*scale, nil)
	if err != nil {
		return err
	}
	f, err := r3.NewFrame(r3.NewVec(x1*scale, y1*scale, z1*scale), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create box frame: %w", name, err)
	}
	b, err = c.place(name, b, f)
	if err != nil {
		return err
	}
	return c.append(name, group, b)
}

func (c *construction) plate(name, group string, origin r3.Vec, angle, x1, z1, x2, z2, depth float64,
	holes []hole) error {
	b, err := c.rectangle(name, x1, z1, x2, z2, depth, holes)
	if err != nil {
		return err
	}
	f, err := xzFrame(origin, angle)
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create plate frame: %w", name, err)
	}
	b, err = c.place(name, b, f)
	if err != nil {
		return err
	}
	return c.append(name, group, b)
}

func (c *construction) cylinder(name, group string, center r3.Vec, y, radius, bore, length float64) error {
	b, err := c.round(name, radius, bore, length)
	if err != nil {
		return err
	}
	f, err := xzFrame(r3.NewVec(center.X, y, center.Z), 0)
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create cylinder frame: %w", name, err)
	}
	b, err = c.place(name, b, f)
	if err != nil {
		return err
	}
	return c.append(name, group, b)
}

func (c *construction) spindle(center r3.Vec, scale float64) error {
	const name = "spindle"
	s, err := c.section(name, true)
	if err != nil {
		return err
	}
	p0 := s.Origin()
	p1 := s.CreatePoint(2.5, 0)
	p2 := s.CreatePoint(2.5, 275*scale)
	p3 := s.CreatePoint(0.3, 300*scale)
	p4 := s.CreatePoint(0, 300*scale)
	l0 := s.CreateLine(p0, p1)
	l1 := s.CreateLine(p1, p2)
	s.CreateLine(p2, p3)
	l3 := s.CreateLine(p3, p4)
	l4 := s.CreateLine(p4, p0)
	s.AddConstraint(sketch.NewHorizontal(l0), sketch.NewVertical(l1), sketch.NewHorizontal(l3), sketch.NewVertical(l4),
		sketch.NewHorizontalDistance(p0, p1, 2.5), sketch.NewVerticalDistance(p1, p2, 275*scale),
		sketch.NewHorizontalDistance(p0, p3, 0.3), sketch.NewVerticalDistance(p0, p3, 300*scale))
	p, err := c.profile(name, s, 0)
	if err != nil {
		return err
	}
	b, err := decad.Revolve(c.ctx, c.doc, s, p, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.NewVec(0, 0, 1)},
		decad.FullRevolution{})
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: revolve section: %w", name, err)
	}
	f, err := xzFrame(r3.NewVec(center.X, 30*scale, center.Z), 0)
	if err != nil {
		return fmt.Errorf("pedalspindle: %s: create spindle frame: %w", name, err)
	}
	b, err = c.place(name, b, f)
	if err != nil {
		return err
	}
	return c.append(name, "steel", b)
}
