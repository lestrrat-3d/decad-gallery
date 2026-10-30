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
	scale    float64
	err      error
	reported bool
}

type hole struct{ x, y, diameter float64 }

func (b *builder) section(axial bool, offset float64) *sketch.Sketch {
	if b.err != nil {
		return nil
	}
	w := sketch.NewWorld()
	plane := w.XY()
	if axial {
		plane = w.XZ()
	}
	if offset != 0 {
		plane, b.err = w.CreateOffsetPlane(plane, offset)
		if b.err != nil {
			b.err = fmt.Errorf("create offset plane: %w", b.err)
			return nil
		}
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		b.err = fmt.Errorf("create section: %w", err)
		return nil
	}
	b.assembly.Sketches = append(b.assembly.Sketches, s)
	return s
}

func rectangle(s *sketch.Sketch, x0, y0, x1, y1 float64) {
	r := s.CreateRectangle(x0, y0, x1, y1)
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), r.A, x0),
		sketch.NewVerticalDistance(s.Origin(), r.A, y0),
		sketch.NewHorizontalDistance(r.A, r.B, x1-x0), sketch.NewVerticalDistance(r.B, r.C, y1-y0))
}

func holes(s *sketch.Sketch, list []hole) {
	for _, h := range list {
		p := s.Origin()
		if h.x != 0 || h.y != 0 {
			p = s.CreatePoint(h.x, h.y)
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), p, h.x), sketch.NewVerticalDistance(s.Origin(), p, h.y))
		}
		c := s.CreateCircle(p, h.diameter/2)
		s.AddConstraint(sketch.NewDiameter(c, h.diameter))
	}
}

func (b *builder) profile(s *sketch.Sketch, holeCount int) *sketch.Profile {
	if b.err != nil {
		return nil
	}
	if _, err := s.Solve(b.ctx); err != nil {
		b.err = fmt.Errorf("solve section: %w", err)
		return nil
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		b.err = fmt.Errorf("verify section: %w", err)
		return nil
	}
	var selected *sketch.Profile
	for _, p := range s.Profiles() {
		if !p.Valid || len(p.Holes) != holeCount {
			continue
		}
		if selected != nil {
			b.err = fmt.Errorf("select section: multiple valid profiles with %d holes", holeCount)
			return nil
		}
		selected = p
	}
	if selected == nil {
		b.err = fmt.Errorf("select section: no valid profile with %d holes", holeCount)
	}
	return selected
}

func (b *builder) extrude(s *sketch.Sketch, n int, length float64) *decad.Body {
	p := b.profile(s, n)
	if b.err != nil {
		return nil
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		b.err = fmt.Errorf("extrude section: %w", err)
	}
	return body
}

func (b *builder) rect(x0, y0, x1, y1, length, offset float64, hs ...hole) *decad.Body {
	s := b.section(false, offset)
	if b.err != nil {
		return nil
	}
	rectangle(s, x0, y0, x1, y1)
	holes(s, hs)
	return b.extrude(s, len(hs), length)
}

func (b *builder) disc(radius, bore, length, offset float64) *decad.Body {
	s := b.section(false, offset)
	if b.err != nil {
		return nil
	}
	c := s.CreateCircle(s.Origin(), radius)
	s.AddConstraint(sketch.NewRadius(c, radius))
	n := 0
	if bore > 0 {
		holes(s, []hole{{0, 0, bore}})
		n = 1
	}
	return b.extrude(s, n, length)
}

func (b *builder) place(body *decad.Body, x, y, z, t float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	f, err := r3.NewFrame(r3.NewVec(x, y, z), r3.NewVec(math.Cos(t), 0, math.Sin(t)), r3.NewVec(-math.Sin(t), 0, math.Cos(t)))
	if err != nil {
		b.err = fmt.Errorf("create placement frame: %w", err)
		return nil
	}
	xf, err := r3.FromFrame(f)
	if err != nil {
		b.err = fmt.Errorf("create placement transform: %w", err)
		return nil
	}
	placed, err := decad.Transform(b.ctx, body, xf)
	if err != nil {
		b.err = fmt.Errorf("transform body: %w", err)
	}
	return placed
}

func (b *builder) add(name, group string, body *decad.Body) {
	if b.err != nil {
		if !b.reported {
			b.err = fmt.Errorf("pedalspindle: %s: %w", name, b.err)
			b.reported = true
		}
		return
	}
	if body == nil || !body.IsSolid() {
		b.err = fmt.Errorf("pedalspindle: %s: body is not solid", name)
		b.reported = true
		return
	}
	b.assembly.Parts = append(b.assembly.Parts, Part{Name: name, Group: group, Body: body})
}

func (b *builder) box(name, group string, x0, x1, y0, y1, z0, z1 float64) {
	s := b.scale
	body := b.rect(x0*s, z0*s, x1*s, z1*s, (y1-y0)*s, 0)
	b.add(name, group, b.place(body, 0, y1*s, 0, 0))
}

func (b *builder) ring(name, group string, x, z, y0, y1, od, id, t float64) {
	body := b.disc(od/2, id, y1-y0, 0)
	b.add(name, group, b.place(body, x, y1, z, t))
}

func polygon(s *sketch.Sketch, coords [][2]float64, origin bool) ([]*sketch.Point, []*sketch.Line) {
	ps := make([]*sketch.Point, len(coords))
	ls := make([]*sketch.Line, len(coords))
	for i, p := range coords {
		if i == 0 && origin {
			ps[i] = s.Origin()
		} else {
			ps[i] = s.CreatePoint(p[0], p[1])
		}
	}
	for i := range ps {
		ls[i] = s.CreateLine(ps[i], ps[(i+1)%len(ps)])
	}
	return ps, ls
}

func (b *builder) revolve(s *sketch.Sketch) *decad.Body {
	p := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, p, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.NewVec(0, 0, 1)}, decad.FullRevolution{})
	if err != nil {
		b.err = fmt.Errorf("revolve section: %w", err)
	}
	return body
}

func (b *builder) spindle() *decad.Body {
	s := b.section(true, 0)
	if b.err != nil {
		return nil
	}
	k := b.scale
	ps, ls := polygon(s, [][2]float64{{0, 0}, {2.5, 0}, {2.5, 275 * k}, {.3, 300 * k}, {0, 300 * k}}, true)
	s.AddConstraint(sketch.NewHorizontal(ls[0]), sketch.NewVertical(ls[1]), sketch.NewHorizontal(ls[3]),
		sketch.NewVertical(ls[4]), sketch.NewHorizontalDistance(ps[0], ps[1], 2.5),
		sketch.NewVerticalDistance(ps[1], ps[2], 275*k), sketch.NewHorizontalDistance(ps[0], ps[3], .3),
		sketch.NewVerticalDistance(ps[0], ps[3], 300*k))
	return b.revolve(s)
}
