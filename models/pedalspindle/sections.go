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
	world    *sketch.World
}

type hole struct{ x, y, diameter float64 }

func (b *construction) newSection(name string, axial bool) (*sketch.Sketch, error) {
	plane := b.world.XY()
	if axial {
		plane = b.world.XZ()
	}
	s, err := b.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: create sketch: %w", name, err)
	}
	b.assembly.Sketches = append(b.assembly.Sketches, s)
	return s, nil
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

func (b *construction) profile(name string, s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(b.ctx); err != nil {
		return nil, fmt.Errorf("%s: solve sketch: %w", name, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify sketch: %w", name, err)
	}
	var selected *sketch.Profile
	count := 0
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == holes {
			selected = p
			count++
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("%s: select profile: want one valid profile with %d holes, got %d", name, holes, count)
	}
	return selected, nil
}

func (b *construction) extrude(name string, s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	p, err := b.profile(name, s, holes)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude section: %w", name, err)
	}
	return body, nil
}

func (b *construction) rectangle(name string, x1, y1, x2, y2, length float64, holes []hole) (*decad.Body, error) {
	s, err := b.newSection(name, false)
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
	return b.extrude(name, s, len(holes), length)
}

func (b *construction) round(name string, radius, bore, length float64) (*decad.Body, error) {
	s, err := b.newSection(name, false)
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
	return b.extrude(name, s, holes, length)
}

func xzFrame(origin r3.Vec, angle float64) (r3.Frame, error) {
	return r3.NewFrame(origin, r3.NewVec(math.Cos(angle), 0, math.Sin(angle)),
		r3.NewVec(-math.Sin(angle), 0, math.Cos(angle)))
}

func (b *construction) place(name string, body *decad.Body, frame r3.Frame) (*decad.Body, error) {
	t, err := r3.FromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("%s: placement transform: %w", name, err)
	}
	placed, err := decad.Transform(b.ctx, body, t)
	if err != nil {
		return nil, fmt.Errorf("%s: transform body: %w", name, err)
	}
	return placed, nil
}

func (b *construction) add(name, group string, body *decad.Body) error {
	if !body.IsSolid() {
		return fmt.Errorf("%s: body is not solid", name)
	}
	b.assembly.Parts = append(b.assembly.Parts, Part{Name: name, Group: group, Body: body})
	return nil
}

func (b *construction) addXZ(name, group string, body *decad.Body, origin r3.Vec, angle float64) error {
	f, err := xzFrame(origin, angle)
	if err != nil {
		return fmt.Errorf("%s: placement frame: %w", name, err)
	}
	body, err = b.place(name, body, f)
	if err != nil {
		return err
	}
	return b.add(name, group, body)
}

func (b *construction) box(name, group string, x1, x2, y1, y2, z1, z2 float64) error {
	body, err := b.rectangle(name, 0, 0, x2-x1, y2-y1, z2-z1, nil)
	if err != nil {
		return err
	}
	f, err := r3.NewFrame(r3.NewVec(x1, y1, z1), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return fmt.Errorf("%s: placement frame: %w", name, err)
	}
	body, err = b.place(name, body, f)
	if err != nil {
		return err
	}
	return b.add(name, group, body)
}

func (b *construction) treadle(scale float64) (*decad.Body, error) {
	const name = "treadle"
	s, err := b.newSection(name, false)
	if err != nil {
		return nil, err
	}
	coords := [8][2]float64{{-18, -18}, {18, -18}, {18, -12}, {212, -12}, {212, 12}, {18, 12}, {18, 18}, {-18, 18}}
	points := make([]*sketch.Point, len(coords))
	for i, p := range coords {
		points[i] = s.CreatePoint(p[0]*scale, p[1]*scale)
	}
	for i := range points {
		line := s.CreateLine(points[i], points[(i+1)%len(points)])
		if i%2 == 0 {
			s.AddConstraint(sketch.NewHorizontal(line))
		} else {
			s.AddConstraint(sketch.NewVertical(line))
		}
	}
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), points[0], -18*scale),
		sketch.NewVerticalDistance(s.Origin(), points[0], -18*scale),
		sketch.NewHorizontalDistance(points[0], points[1], 36*scale),
		sketch.NewHorizontalDistance(points[2], points[3], 194*scale),
		sketch.NewHorizontalDistance(points[4], points[5], -194*scale),
		sketch.NewVerticalDistance(points[1], points[2], 6*scale),
		sketch.NewVerticalDistance(points[3], points[4], 24*scale),
		sketch.NewVerticalDistance(points[5], points[6], 6*scale),
	)
	addHoles(s, []hole{{diameter: 22}, {x: 200 * scale, diameter: 5.2}})
	return b.extrude(name, s, 2, 70*scale)
}

func (b *construction) spindle(scale float64) (*decad.Body, error) {
	const name = "spindle"
	s, err := b.newSection(name, true)
	if err != nil {
		return nil, err
	}
	p := []*sketch.Point{s.Origin(), s.CreatePoint(2.5, 0), s.CreatePoint(2.5, 275*scale),
		s.CreatePoint(0.3, 300*scale), s.CreatePoint(0, 300*scale)}
	lines := make([]*sketch.Line, len(p))
	for i := range p {
		lines[i] = s.CreateLine(p[i], p[(i+1)%len(p)])
	}
	s.AddConstraint(sketch.NewHorizontal(lines[0]), sketch.NewVertical(lines[1]),
		sketch.NewHorizontal(lines[3]), sketch.NewVertical(lines[4]),
		sketch.NewHorizontalDistance(p[0], p[1], 2.5),
		sketch.NewVerticalDistance(p[1], p[2], 275*scale),
		sketch.NewHorizontalDistance(p[0], p[3], 0.3),
		sketch.NewVerticalDistance(p[0], p[3], 300*scale))
	profile, err := b.profile(name, s, 0)
	if err != nil {
		return nil, err
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, profile,
		decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.NewVec(0, 0, 1)}, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("%s: revolve axial section: %w", name, err)
	}
	return body, nil
}
