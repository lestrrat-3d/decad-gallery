package handcranksiren

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Shaft D-flat dimensions in mm: the shaft is 8 across, the bore of a 608 bearing, with a flat 3.25 from its axis toward +Z.
const (
	shaftRadius = 4.0
	flatOffset  = 3.25
	flatReach   = 3.0 // how far past the flat the cutting box runs
	boxHalf     = 5.0 // the box's half width across the flat
)

type point2 struct{ u, v float64 }

func frame(origin, u, v r3.Vec) r3.Frame {
	f, _ := r3.NewFrame(origin, u, v)
	return f
}

// shaftAxis is one axis along Y: the point on it at Y 0.
type shaftAxis struct{ point r3.Vec }

// across is the plane at Y = y across the axis with U = +X and V = −Z, so its normal is +Y.
func (a shaftAxis) across(y float64) r3.Frame {
	return frame(a.point.Add(r3.Vec{Y: y}), r3.Vec{X: 1}, r3.Vec{Z: -1})
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

// cylinder extrudes a disc of radius r centred on f's origin along f's normal.
func (b *builder) cylinder(f r3.Frame, r, length float64) *decad.Body {
	s := b.section(f)
	circle(s, 0, 0, r)
	return b.extrude(s, 0, length)
}

// box extrudes a rectangle u0..u1 by v0..v1 along f's normal. No edge may pass through the origin.
func (b *builder) box(f r3.Frame, u0, v0, u1, v1, length float64) *decad.Body {
	s := b.section(f)
	rectangle(s, u0, v0, u1, v1)
	return b.extrude(s, 0, length)
}

func (b *builder) union(parts ...*decad.Body) *decad.Body {
	if b.err != nil {
		return nil
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// Decad refuses a joint Union of overlapping bodies with shared faces, though it joins them in pairs: fold left to
	// right.
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

const axisTolerance = 1e-9

// segmentKind classifies a segment from a to b: 'h' horizontal, 'v' vertical, 's' slanted.
func segmentKind(a, b point2) byte {
	switch {
	case math.Abs(a.v-b.v) < axisTolerance:
		return 'h'
	case math.Abs(a.u-b.u) < axisTolerance:
		return 'v'
	}
	return 's'
}

// meridianPolygon draws the closed polygon pts in (u, v) and constrains it without fixing every point. A horizontal
// segment is horizontal at distance |v| from the origin, and a vertical one vertical at distance |u|. A vertex between
// a horizontal and a slanted segment is placed by its u, one between a vertical and a slanted segment by its v, and one
// between two slanted segments by both. A vertex between two axis-aligned segments needs nothing more. An
// axis-aligned segment through the origin is an error.
func meridianPolygon(s *sketch.Sketch, pts []point2) error {
	if s == nil {
		return nil
	}
	n := len(pts)
	if n < 3 {
		return fmt.Errorf("meridian polygon needs 3 points, got %d", n)
	}
	kinds := make([]byte, n)
	for i := range pts {
		kinds[i] = segmentKind(pts[i], pts[(i+1)%n])
		if (kinds[i] == 'h' && math.Abs(pts[i].v) < axisTolerance) || (kinds[i] == 'v' && math.Abs(pts[i].u) < axisTolerance) {
			return fmt.Errorf("segment %d passes through the sketch origin", i)
		}
	}
	points := make([]*sketch.Point, n)
	for i, p := range pts {
		points[i] = s.CreatePoint(p.u, p.v)
	}
	lines := make([]*sketch.Line, n)
	for i := range pts {
		lines[i] = s.CreateLine(points[i], points[(i+1)%n])
		switch kinds[i] {
		case 'h':
			s.AddConstraint(sketch.NewHorizontal(lines[i]),
				sketch.NewDistancePointLine(s.Origin(), lines[i], math.Abs(pts[i].v)))
		case 'v':
			s.AddConstraint(sketch.NewVertical(lines[i]),
				sketch.NewDistancePointLine(s.Origin(), lines[i], math.Abs(pts[i].u)))
		}
	}
	for i, p := range pts {
		before, after := kinds[(i+n-1)%n], kinds[i]
		switch {
		case before != 's' && after != 's':
		case before == 's' && after == 's':
			s.AddConstraint(
				sketch.NewHorizontalDistance(s.Origin(), points[i], p.u),
				sketch.NewVerticalDistance(s.Origin(), points[i], p.v),
			)
		case before == 'h' || after == 'h':
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), points[i], p.u))
		default:
			s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), points[i], p.v))
		}
	}
	return nil
}

// port draws the chord-sided port centred on +U: the points p1 (rIn cos h, −rIn sin h), p2 (rIn cos h, rIn sin h),
// p3 (rOut cos h, −rOut sin h) and p4 (rOut cos h, rOut sin h) joined by the lines p1–p2, p3–p4, p1–p3 and p2–p4. The
// lines p1–p2 and p3–p4 are vertical at their distances from the origin, the origin lies on p1–p3 and p2–p4, and p1
// and p2 are placed by their vertical distance. half is the half angle in radians.
func port(s *sketch.Sketch, rIn, rOut, half float64) {
	if s == nil {
		return
	}
	cos, sin := math.Cos(half), math.Sin(half)
	p1, p2 := s.CreatePoint(rIn*cos, -rIn*sin), s.CreatePoint(rIn*cos, rIn*sin)
	p3, p4 := s.CreatePoint(rOut*cos, -rOut*sin), s.CreatePoint(rOut*cos, rOut*sin)
	inner, outer := s.CreateLine(p1, p2), s.CreateLine(p3, p4)
	right, left := s.CreateLine(p1, p3), s.CreateLine(p2, p4)
	s.AddConstraint(
		sketch.NewVertical(inner), sketch.NewDistancePointLine(s.Origin(), inner, rIn*cos),
		sketch.NewVertical(outer), sketch.NewDistancePointLine(s.Origin(), outer, rOut*cos),
		sketch.NewPointOnLine(s.Origin(), right), sketch.NewPointOnLine(s.Origin(), left),
		sketch.NewVerticalDistance(s.Origin(), p1, -rIn*sin),
		sketch.NewVerticalDistance(s.Origin(), p2, rIn*sin),
	)
}

// dShape builds the D-shaped cylinder radius shaftRadius + clearance about a, from Y from for length, without the part
// beyond the flat, which lies flatOffset + clearance from the axis toward +Z. The box that removes it starts 1 before
// from and ends 1 after the cylinder, so the cut is clean at both ends.
func (b *builder) dShape(a shaftAxis, from, length, flat, radius float64) *decad.Body {
	round := b.cylinder(a.across(from), radius, length)
	// In the across frame V = −Z, so the flat side lies at v < −flat.
	slab := b.box(a.across(from-1), -boxHalf, -(flat + flatReach), boxHalf, -flat, length+2)
	return b.cut(round, slab)
}

// dBoreTool builds the cutter of a D-bore about a: the shaft's D-shape larger by clearance, from Y from for length.
func (b *builder) dBoreTool(a shaftAxis, from, length, clearance float64) *decad.Body {
	return b.dShape(a, from, length, flatOffset+clearance, shaftRadius+clearance)
}

// dShaft builds a D-flat shaft about a from Y from to Y to.
func (b *builder) dShaft(a shaftAxis, from, to float64) *decad.Body {
	return b.dShape(a, from, to-from, flatOffset, shaftRadius)
}
