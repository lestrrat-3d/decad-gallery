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

type builder struct {
	ctx      context.Context
	doc      *decad.Document
	assembly *Assembly
}

type hole struct{ x, y, diameter float64 }

func (b *builder) sketch(name string, axial bool) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane := w.XY()
	if axial {
		plane = w.XZ()
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: create sketch: %w", name, err)
	}
	b.assembly.Sketches = append(b.assembly.Sketches, s)
	return s, nil
}

func (b *builder) profile(name string, s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(b.ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", name, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", name, err)
	}
	var selected *sketch.Profile
	for _, p := range s.Profiles() {
		if !p.Valid || len(p.Holes) != holes {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("%s: multiple valid profiles with %d holes", name, holes)
		}
		selected = p
	}
	if selected == nil {
		return nil, fmt.Errorf("%s: no valid profile with %d holes", name, holes)
	}
	return selected, nil
}

func addHoles(s *sketch.Sketch, holes []hole) {
	for _, h := range holes {
		center := s.Origin()
		if h.x != 0 || h.y != 0 {
			center = sketchcheck.DimensionPoint(s, h.x, h.y)
		}
		c := s.CreateCircle(center, h.diameter/2)
		s.AddConstraint(sketch.NewDiameter(c, h.diameter))
	}
}

func (b *builder) extrude(name string, s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
	p, err := b.profile(name, s, holes)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", name, err)
	}
	return body, nil
}

func (b *builder) rectangle(name string, x0, y0, x1, y1, length float64, holes []hole) (*decad.Body, error) {
	s, err := b.sketch(name, false)
	if err != nil {
		return nil, err
	}
	r := s.CreateRectangle(x0, y0, x1, y1)
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), r.A, x0), sketch.NewVerticalDistance(s.Origin(), r.A, y0),
		sketch.NewHorizontalDistance(r.A, r.B, x1-x0), sketch.NewVerticalDistance(r.B, r.C, y1-y0))
	addHoles(s, holes)
	return b.extrude(name, s, len(holes), length)
}

func (b *builder) disc(name string, radius, diameter, length float64) (*decad.Body, error) {
	s, err := b.sketch(name, false)
	if err != nil {
		return nil, err
	}
	c := s.CreateCircle(s.Origin(), radius)
	s.AddConstraint(sketch.NewRadius(c, radius))
	holes := 0
	if diameter > 0 {
		addHoles(s, []hole{{diameter: diameter}})
		holes = 1
	}
	return b.extrude(name, s, holes, length)
}

func (b *builder) transform(name string, body *decad.Body, origin, u, v r3.Vec) (*decad.Body, error) {
	f, err := r3.NewFrame(origin, u, v)
	if err != nil {
		return nil, fmt.Errorf("%s: frame: %w", name, err)
	}
	t, err := r3.FromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("%s: placement: %w", name, err)
	}
	body, err = decad.Transform(b.ctx, body, t)
	if err != nil {
		return nil, fmt.Errorf("%s: transform: %w", name, err)
	}
	return body, nil
}

func (b *builder) place(name string, body *decad.Body, center r3.Vec, y, angle float64) (*decad.Body, error) {
	return b.transform(name, body, r3.NewVec(center.X, y, center.Z),
		r3.NewVec(math.Cos(angle), 0, math.Sin(angle)), r3.NewVec(-math.Sin(angle), 0, math.Cos(angle)))
}

func (b *builder) part(name, group string, body *decad.Body) error {
	if body == nil || !body.IsSolid() {
		return fmt.Errorf("%s: body is not solid", name)
	}
	b.assembly.Parts = append(b.assembly.Parts, Part{Name: name, Group: group, Body: body})
	return nil
}

func (b *builder) treadle(scale float64) (*decad.Body, error) {
	const name = "treadle"
	s, err := b.sketch(name, false)
	if err != nil {
		return nil, err
	}
	xy := [][2]float64{{-18, -18}, {18, -18}, {18, -12}, {212, -12}, {212, 12}, {18, 12}, {18, 18}, {-18, 18}}
	points := make([]*sketch.Point, len(xy))
	for i, p := range xy {
		points[i] = s.CreatePoint(p[0]*scale, p[1]*scale)
	}
	lines := make([]*sketch.Line, len(xy))
	for i := range points {
		lines[i] = s.CreateLine(points[i], points[(i+1)%len(points)])
		if i%2 == 0 {
			s.AddConstraint(sketch.NewHorizontal(lines[i]))
		} else {
			s.AddConstraint(sketch.NewVertical(lines[i]))
		}
	}
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), points[0], -18*scale),
		sketch.NewVerticalDistance(s.Origin(), points[0], -18*scale),
		sketch.NewHorizontalDistance(points[0], points[1], 36*scale),
		sketch.NewHorizontalDistance(points[2], points[3], 194*scale),
		sketch.NewHorizontalDistance(points[4], points[5], -194*scale),
		sketch.NewVerticalDistance(points[1], points[2], 6*scale),
		sketch.NewVerticalDistance(points[3], points[4], 24*scale),
		sketch.NewVerticalDistance(points[5], points[6], 6*scale))
	addHoles(s, []hole{{diameter: 22}, {x: 200 * scale, diameter: 5.2}})
	return b.extrude(name, s, 2, 70*scale)
}

func (b *builder) spindle(scale float64) (*decad.Body, error) {
	const name = "spindle"
	s, err := b.sketch(name, true)
	if err != nil {
		return nil, err
	}
	p := []*sketch.Point{s.Origin(), s.CreatePoint(2.5, 0), s.CreatePoint(2.5, 275*scale),
		s.CreatePoint(0.3, 300*scale), s.CreatePoint(0, 300*scale)}
	l := make([]*sketch.Line, len(p))
	for i := range p {
		l[i] = s.CreateLine(p[i], p[(i+1)%len(p)])
	}
	s.AddConstraint(sketch.NewHorizontal(l[0]), sketch.NewHorizontal(l[3]), sketch.NewVertical(l[1]),
		sketch.NewVertical(l[4]), sketch.NewHorizontalDistance(p[0], p[1], 2.5),
		sketch.NewVerticalDistance(p[1], p[2], 275*scale),
		sketch.NewHorizontalDistance(p[0], p[3], 0.3), sketch.NewVerticalDistance(p[0], p[3], 300*scale))
	profile, err := b.profile(name, s, 0)
	if err != nil {
		return nil, err
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, profile, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.NewVec(0, 0, 1)}, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("%s: revolve: %w", name, err)
	}
	return body, nil
}
