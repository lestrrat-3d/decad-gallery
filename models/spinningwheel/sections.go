package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type point2 struct{ x, y float64 }
type hole struct{ x, y, r float64 }

func (b *builder) section(axial bool) *sketch.Sketch {
	if b.err != nil {
		return nil
	}
	if err := b.ctx.Err(); err != nil {
		b.fail("create section", err)
		return nil
	}
	w := sketch.NewWorld()
	plane := w.XY()
	if axial {
		plane = w.XZ()
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		b.fail("create section", err)
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, s)
	return s
}

func (b *builder) profile(s *sketch.Sketch, holes int) *sketch.Profile {
	if b.err != nil {
		return nil
	}
	if _, err := s.Solve(b.ctx); err != nil {
		b.fail("solve section", err)
		return nil
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		b.fail("verify section", err)
		return nil
	}
	var selected *sketch.Profile
	for _, candidate := range s.Profiles() {
		if !candidate.Valid || len(candidate.Holes) != holes {
			continue
		}
		if selected != nil {
			b.fail("select section profile", fmt.Errorf("multiple valid profiles with %d holes", holes))
			return nil
		}
		selected = candidate
	}
	if selected == nil {
		b.fail("select section profile", fmt.Errorf("no valid profile with %d holes", holes))
	}
	return selected
}

func rectangle(s *sketch.Sketch, x0, y0, x1, y1 float64) {
	if s == nil {
		return
	}
	r := s.CreateRectangle(x0, y0, x1, y1)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, x0),
		sketch.NewVerticalDistance(s.Origin(), r.A, y0),
		sketch.NewHorizontalDistance(r.A, r.B, x1-x0),
		sketch.NewVerticalDistance(r.B, r.C, y1-y0),
	)
}

func circle(s *sketch.Sketch, x, y, r float64) {
	if s == nil {
		return
	}
	centre := s.Origin()
	if x != 0 || y != 0 {
		centre = s.CreatePoint(x, y)
		s.AddConstraint(
			sketch.NewHorizontalDistance(s.Origin(), centre, x),
			sketch.NewVerticalDistance(s.Origin(), centre, y),
		)
	}
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(centre, r), r))
}

func dOutline(s *sketch.Sketch, r, flat float64) {
	if s == nil {
		return
	}
	y := math.Sqrt(r*r - flat*flat)
	upper, lower := s.CreatePoint(flat, y), s.CreatePoint(flat, -y)
	arc := s.CreateArc(s.Origin(), upper, lower)
	chord := s.CreateLine(lower, upper)
	s.AddConstraint(
		sketch.NewRadius(arc, r),
		sketch.NewHorizontalDistance(s.Origin(), upper, flat),
		sketch.NewVertical(chord),
	)
}

func polygon(s *sketch.Sketch, xy []point2) {
	if s == nil || len(xy) < 3 {
		return
	}
	points := make([]*sketch.Point, len(xy))
	xParents, yParents := make([]int, len(xy)), make([]int, len(xy))
	for i, p := range xy {
		points[i] = s.CreatePoint(p.x, p.y)
		xParents[i], yParents[i] = i, i
	}
	root := func(parents []int, index int) int {
		for parents[index] != index {
			index = parents[index]
		}
		return index
	}
	for i, p := range xy {
		j := (i + 1) % len(xy)
		line := s.CreateLine(points[i], points[j])
		if p.x == xy[j].x {
			s.AddConstraint(sketch.NewVertical(line))
			xParents[root(xParents, j)] = root(xParents, i)
		}
		if p.y == xy[j].y {
			s.AddConstraint(sketch.NewHorizontal(line))
			yParents[root(yParents, j)] = root(yParents, i)
		}
	}
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), points[0], xy[0].x),
		sketch.NewVerticalDistance(s.Origin(), points[0], xy[0].y),
	)
	// Dimension only independent coordinate differences. Closing edges already
	// constrain equal coordinates, so their dimensions must not be repeated.
	for i := 0; i < len(xy)-1; i++ {
		j := i + 1
		if root(xParents, i) != root(xParents, j) {
			s.AddConstraint(sketch.NewHorizontalDistance(points[i], points[j], xy[j].x-xy[i].x))
			xParents[root(xParents, j)] = root(xParents, i)
		}
		if root(yParents, i) != root(yParents, j) {
			s.AddConstraint(sketch.NewVerticalDistance(points[i], points[j], xy[j].y-xy[i].y))
			yParents[root(yParents, j)] = root(yParents, i)
		}
	}
}

func scaled(scale float64, xy ...point2) []point2 {
	result := make([]point2, len(xy))
	for i, p := range xy {
		result[i] = point2{p.x * scale, p.y * scale}
	}
	return result
}

func frame(origin, u, v r3.Vec) r3.Frame {
	f, _ := r3.NewFrame(origin, u, v)
	return f
}

func shaftFrame(x, y, z, angle float64) r3.Frame {
	angle = math.Remainder(angle, 2*math.Pi)
	c, s := math.Cos(angle), math.Sin(angle)
	return frame(r3.Vec{X: x, Y: y, Z: z}, r3.Vec{X: c, Z: s}, r3.Vec{X: -s, Z: c})
}

func axisFrame(origin, axis r3.Vec) r3.Frame {
	n, ok := axis.Normalize()
	if !ok {
		return r3.Frame{}
	}
	ref := r3.Vec{Z: 1}
	if math.Abs(n.Z) > 0.9 {
		ref = r3.Vec{X: 1}
	}
	u, ok := ref.Cross(n).Normalize()
	if !ok {
		return r3.Frame{}
	}
	return frame(origin, u, n.Cross(u))
}

func (b *builder) extrude(s *sketch.Sketch, holes int, length float64, f r3.Frame) *decad.Body {
	p := b.profile(s, holes)
	if b.err != nil {
		return nil
	}
	xf, err := r3.FromFrame(f)
	if err != nil {
		b.fail("create extrusion placement", err)
		return nil
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		b.fail("extrude section", err)
		return nil
	}
	body, err = decad.Transform(b.ctx, body, xf)
	b.fail("place extrusion", err)
	return body
}

func (b *builder) panel(x0, z0, x1, z1, length float64, f r3.Frame, holes ...hole) *decad.Body {
	s := b.section(false)
	rectangle(s, x0, z0, x1, z1)
	for _, h := range holes {
		circle(s, h.x, h.y, h.r)
	}
	return b.extrude(s, len(holes), length, f)
}

func (b *builder) box(x0, x1, y0, y1, z0, z1 float64) *decad.Body {
	return b.panel(x0, y0, x1, y1, z1-z0,
		frame(r3.Vec{Z: z0}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
}

func (b *builder) cylinder(radius, length float64, f r3.Frame) *decad.Body {
	s := b.section(false)
	circle(s, 0, 0, radius)
	return b.extrude(s, 0, length, f)
}

func (b *builder) annulus(inner, outer, length float64, f r3.Frame) *decad.Body {
	if b.err != nil {
		return nil
	}
	m, err := ballbearing.Build(b.ctx, b.doc, ballbearing.Params{
		BoreDiameter: 2 * inner, OuterDiameter: 2 * outer, Width: length, Placement: &f,
	})
	if err != nil {
		b.fail("build annular envelope", err)
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Sketch)
	return m.Body
}

func (b *builder) dTool(radius, flat, length float64, f r3.Frame) *decad.Body {
	s := b.section(false)
	dOutline(s, radius, flat)
	return b.extrude(s, 0, length, f)
}

func (b *builder) dAnnulus(inner, flat, outer, length float64, f r3.Frame) *decad.Body {
	s := b.section(false)
	circle(s, 0, 0, outer)
	dOutline(s, inner, flat)
	return b.extrude(s, 1, length, f)
}

func (b *builder) cut(host, tool *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	body, err := decad.Cut(b.ctx, host, tool)
	b.fail("cut solid", err)
	return body
}

func (b *builder) union(parts ...*decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	if len(parts) == 0 {
		b.fail("union solids", fmt.Errorf("no bodies supplied"))
		return nil
	}
	if len(parts) == 1 {
		return parts[0]
	}
	body, err := decad.Union(b.ctx, parts[0], parts[1:]...)
	b.fail("union solids", err)
	return body
}

func (b *builder) flatShaft(body *decad.Body, caps ...*decad.Body) *decad.Body {
	if len(caps) == 0 {
		return body
	}
	return b.cut(body, b.union(caps...))
}

func (b *builder) span(a, c r3.Vec, radius float64) *decad.Body {
	d := c.Sub(a)
	return b.cylinder(radius, d.Len(), axisFrame(a, d))
}

func (b *builder) revolve(xy []point2, f r3.Frame) *decad.Body {
	s := b.section(true)
	polygon(s, xy)
	p := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	xf, err := r3.FromFrame(f)
	if err != nil {
		b.fail("create revolved placement", err)
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, p,
		decad.ConstructionAxis{Dir: r3.Vec{Z: 1}}, decad.FullRevolution{})
	if err != nil {
		b.fail("revolve axial section", err)
		return nil
	}
	body, err = decad.Transform(b.ctx, body, xf)
	b.fail("place revolved section", err)
	return body
}

func (b *builder) torus(radius, wire, start, angle float64, centre r3.Vec) *decad.Body {
	s := b.section(true)
	circle(s, radius, 0, wire)
	p := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, p,
		decad.ConstructionAxis{Dir: r3.Vec{Z: 1}},
		decad.AngleExtent{A: units.Radians(angle), Dir: decad.Along})
	if err != nil {
		b.fail("revolve circular wrap", err)
		return nil
	}
	// The completed revolve's +Z axis maps to -Y, and its initial +X
	// radial ray maps to (cos(start), 0, sin(start)).
	xf, err := r3.FromFrame(shaftFrame(centre.X, centre.Y, centre.Z, start))
	if err != nil {
		b.fail("create circular wrap placement", err)
		return nil
	}
	body, err = decad.Transform(b.ctx, body, xf)
	b.fail("place circular wrap", err)
	return body
}
