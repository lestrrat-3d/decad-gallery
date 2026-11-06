package handcrankfan

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type hole struct{ u, v, r float64 }

func frame(origin, u, v r3.Vec) r3.Frame {
	f, _ := r3.NewFrame(origin, u, v)
	return f
}

// section starts a sketch on f; bodies extrude along f's normal U×V.
func (b *builder) section(f r3.Frame) *sketch.Sketch {
	if b.err != nil {
		return nil
	}
	if err := b.ctx.Err(); err != nil {
		b.fail("create section", err)
		return nil
	}
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		b.fail("create section plane", err)
		return nil
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		b.fail("create section", err)
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, s)
	return s
}

// profile solves s, requires it fully constrained and returns its one valid profile with holes holes.
func (b *builder) profile(s *sketch.Sketch, holes int) *sketch.Profile {
	if b.err != nil || s == nil {
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

// extrude extrudes the one valid profile of s with holes holes.
func (b *builder) extrude(s *sketch.Sketch, holes int, length float64) *decad.Body {
	selected := b.profile(s, holes)
	if selected == nil {
		return nil
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, selected, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	b.fail("extrude section", err)
	return body
}

// revolve turns the one valid profile of s a full turn about axis.
func (b *builder) revolve(s *sketch.Sketch, axis decad.ConstructionAxis) *decad.Body {
	selected := b.profile(s, 0)
	if selected == nil {
		return nil
	}
	body, err := decad.Revolve(b.ctx, b.doc, s, selected, axis, decad.FullRevolution{})
	b.fail("revolve section", err)
	return body
}

// roundCaps rounds the outer loops of both end caps of an extruded body.
func (b *builder) roundCaps(body *decad.Body, radius float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	loops := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Or(decad.OuterLoopOf(decad.CapEnd(body)))
	rounded, err := decad.Fillet(b.ctx, body, loops, units.Millimeters(radius))
	b.fail("round end caps", err)
	return rounded
}

// roundEnd rounds the outer loop of the end cap of an extruded body.
func (b *builder) roundEnd(body *decad.Body, radius float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	rounded, err := decad.Fillet(b.ctx, body, decad.Edges(decad.OuterLoopOf(decad.CapEnd(body))), units.Millimeters(radius))
	b.fail("round end", err)
	return rounded
}

func rectangle(s *sketch.Sketch, u0, v0, u1, v1 float64) {
	if s == nil {
		return
	}
	r := s.CreateRectangle(u0, v0, u1, v1)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, u0),
		sketch.NewVerticalDistance(s.Origin(), r.A, v0),
		sketch.NewHorizontalDistance(r.A, r.B, u1-u0),
		sketch.NewVerticalDistance(r.B, r.C, v1-v0),
	)
}

func circle(s *sketch.Sketch, u, v, r float64) {
	if s == nil {
		return
	}
	centre := s.Origin()
	if u != 0 || v != 0 {
		centre = s.CreatePoint(u, v)
		s.AddConstraint(
			sketch.NewHorizontalDistance(s.Origin(), centre, u),
			sketch.NewVerticalDistance(s.Origin(), centre, v),
		)
	}
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(centre, r), r))
}

// panel extrudes a rectangle with round holes along f's normal.
func (b *builder) panel(f r3.Frame, u0, v0, u1, v1, length float64, holes ...hole) *decad.Body {
	s := b.section(f)
	rectangle(s, u0, v0, u1, v1)
	for _, h := range holes {
		circle(s, h.u, h.v, h.r)
	}
	return b.extrude(s, len(holes), length)
}

// cylinder extrudes a disc of radius r centred on f's origin along f's normal.
func (b *builder) cylinder(f r3.Frame, r, length float64) *decad.Body {
	s := b.section(f)
	circle(s, 0, 0, r)
	return b.extrude(s, 0, length)
}

func (b *builder) union(parts ...*decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// Decad refuses a joint Union of three overlapping plates with shared faces, though it joins them in pairs:
	// fold left to right.
	var err error
	body := parts[0]
	for i, next := range parts[1:] {
		if body, err = decad.Union(b.ctx, body, next); err != nil {
			b.fail(fmt.Sprintf("union solid %d of %d", i+2, len(parts)), err)
			return nil
		}
	}
	return body
}

func (b *builder) cut(host, tool *decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	body, err := decad.Cut(b.ctx, host, tool)
	b.fail("cut solid", err)
	return body
}

// turn rotates body by angle about the line through point along axis.
func (b *builder) turn(body *decad.Body, point, axis r3.Vec, angle float64) *decad.Body {
	if b.err != nil || angle == 0 {
		return body
	}
	xf, err := r3.RotationAround(point, axis, units.Radians(angle))
	if err != nil {
		b.fail("create rotation", err)
		return nil
	}
	turned, err := decad.Transform(b.ctx, body, xf)
	b.fail("rotate body", err)
	return turned
}

type point2 struct{ u, v float64 }

// roundedPolygon draws an axis-aligned polygon whose edges are placed by their distance from the sketch origin, then
// rounds corner i with radii[i] (0 keeps it sharp). No edge may pass through the origin.
func roundedPolygon(s *sketch.Sketch, xy []point2, radii []float64) error {
	if s == nil {
		return nil
	}
	n := len(xy)
	points := make([]*sketch.Point, n)
	for i, p := range xy {
		points[i] = s.CreatePoint(p.u, p.v)
	}
	lines := make([]*sketch.Line, n)
	for i := range xy {
		j := (i + 1) % n
		lines[i] = s.CreateLine(points[i], points[j])
		if xy[i].u == xy[j].u {
			s.AddConstraint(sketch.NewVertical(lines[i]), sketch.NewDistancePointLine(s.Origin(), lines[i], math.Abs(xy[i].u)))
		} else {
			s.AddConstraint(sketch.NewHorizontal(lines[i]), sketch.NewDistancePointLine(s.Origin(), lines[i], math.Abs(xy[i].v)))
		}
	}
	for i := range xy {
		if radii[i] == 0 {
			continue
		}
		if _, err := sketch.CreateFillet(s, lines[(i+n-1)%n], lines[i], points[i], radii[i]); err != nil {
			return fmt.Errorf("round corner %d: %w", i, err)
		}
	}
	return nil
}

// roundedPrism extrudes an axis-aligned polygon with rounded corners along f's normal.
func (b *builder) roundedPrism(f r3.Frame, xy []point2, radii []float64, length float64) *decad.Body {
	s := b.section(f)
	b.fail("draw rounded outline", roundedPolygon(s, xy, radii))
	return b.extrude(s, 0, length)
}

// teardrop draws a round bore of radius r with a pointed top: the circle plus a cap of two lines tangent to it at 45°
// from the +U axis that meet at the point on U, r*sqrt(2) from the origin. A bore printed lying down has no flat roof
// to bridge, because no part of the roof is flatter than 45°.
func teardrop(s *sketch.Sketch, r float64) {
	if s == nil {
		return
	}
	k := r / math.Sqrt2
	start := sketchcheck.DimensionArcEnd(s, k, k)
	end := sketchcheck.DimensionArcEnd(s, k, -k)
	tip := sketchcheck.DimensionPoint(s, r*math.Sqrt2, 0)
	// The arc runs counterclockwise the long way round, away from the point.
	s.AddConstraint(sketch.NewRadius(s.CreateArc(s.Origin(), start, end), r))
	s.CreateLine(end, tip)
	s.CreateLine(tip, start)
}

// teardropPrism extrudes a teardrop bore along f's normal; the point is on f's U axis.
func (b *builder) teardropPrism(f r3.Frame, r, length float64) *decad.Body {
	s := b.section(f)
	teardrop(s, r)
	return b.extrude(s, 0, length)
}

// hexagon extrudes a regular hexagon of the given width across flats along f's normal, with its corners on f's U axis.
func (b *builder) hexagon(f r3.Frame, flats, length float64) *decad.Body {
	s := b.section(f)
	if s == nil {
		return nil
	}
	radius := flats / math.Sqrt(3)
	p, err := s.CreatePolygon(0, 0, 6, radius)
	if err != nil {
		b.fail("create hexagon", err)
		return nil
	}
	s.AddConstraint(
		sketch.NewCoincident(p.Center, s.Origin()),
		sketch.NewHorizontal(p.Sides[1]),
		sketch.NewDistance(p.Center, p.Vertices[0], radius),
	)
	return b.extrude(s, 0, length)
}

// screw builds an M2 socket head cap screw and its nut as one plain body with no thread. bearing is the centre of
// the head's bearing face, dir runs toward the tip, u is a unit vector perpendicular to dir that the nut's corners lie
// on, length is the length under the head, nutAt is the distance from the bearing face to the nut's near face (0 for
// no nut) and shank is the shank radius.
func (b *builder) screw(bearing, dir, u r3.Vec, length, nutAt, shank float64) *decad.Body {
	at := func(t float64) r3.Frame { return frame(bearing.Add(dir.Scale(t)), u, dir.Cross(u)) }
	head := b.cylinder(at(-headHeight), headRadius, headHeight)
	body := b.cylinder(at(-0.8), shank, length+0.8)
	parts := []*decad.Body{head, body}
	if nutAt > 0 {
		parts = append(parts, b.hexagon(at(nutAt), nutFlats, nutHeight))
	}
	return b.union(parts...)
}
