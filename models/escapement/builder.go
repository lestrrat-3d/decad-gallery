package escapement

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// builder holds the state one Build shares: the document, the sketch world and every sketch drawn.
type builder struct {
	ctx      context.Context
	doc      *decad.Document
	world    *sketch.World
	g        Geometry
	sketches []*sketch.Sketch
	// plain skips the edge finishing, so a test can measure what the chamfers remove.
	plain bool
}

func newBuilder(ctx context.Context, doc *decad.Document, g Geometry) *builder {
	return &builder{ctx: ctx, doc: doc, world: sketch.NewWorld(), g: g}
}

// flat returns a sketch whose plane is parallel to XY at height z with its origin at (x, y, z).
func (b *builder) flat(what string, x, y, z float64) (*sketch.Sketch, error) {
	return b.sketchOn(what, r3.NewVec(x, y, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
}

// meridian returns a sketch in the plane through the vertical line at (x, y): U is +X and V is +Z.
func (b *builder) meridian(what string, x, y float64) (*sketch.Sketch, error) {
	return b.sketchOn(what, r3.NewVec(x, y, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
}

func (b *builder) sketchOn(what string, o, u, v r3.Vec) (*sketch.Sketch, error) {
	f, err := r3.NewFrame(o, u, v)
	if err != nil {
		return nil, fmt.Errorf("%s: frame: %w", what, err)
	}
	plane, err := b.world.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("%s: plane: %w", what, err)
	}
	s, err := b.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("%s: sketch: %w", what, err)
	}
	return s, nil
}

// profile solves s, requires it fully constrained, records it and returns its one valid profile with holes holes.
func (b *builder) profile(s *sketch.Sketch, holes int, what string) (*sketch.Profile, error) {
	if err := b.ctx.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if _, err := s.Solve(b.ctx); err != nil {
		return nil, fmt.Errorf("%s: solve: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, s); err != nil {
		return nil, fmt.Errorf("%s: verify: %w", what, err)
	}
	b.sketches = append(b.sketches, s)
	var found *sketch.Profile
	for _, p := range s.Profiles() {
		if !p.Valid || len(p.Holes) != holes {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("%s: more than one region has %d holes", what, holes)
		}
		found = p
	}
	if found == nil {
		return nil, fmt.Errorf("%s: no valid region with %d holes among %d regions", what, holes, len(s.Profiles()))
	}
	return found, nil
}

// extrude extrudes the profile of s with holes holes by length along the sketch normal.
func (b *builder) extrude(s *sketch.Sketch, holes int, length float64, what string) (*decad.Body, error) {
	p, err := b.profile(s, holes, what)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(b.ctx, b.doc, s, p, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("%s: extrude: %w", what, err)
	}
	return body, nil
}

// revolve revolves the profile of the meridian sketch s a full turn about its V axis.
func (b *builder) revolve(s *sketch.Sketch, what string) (*decad.Body, error) {
	p, err := b.profile(s, 0, what)
	if err != nil {
		return nil, err
	}
	axis := decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1}}
	body, err := decad.Revolve(b.ctx, b.doc, s, p, axis, decad.FullRevolution{})
	if err != nil {
		return nil, fmt.Errorf("%s: revolve: %w", what, err)
	}
	return body, nil
}

// cut subtracts tool from body.
func (b *builder) cut(body, tool *decad.Body, what string) (*decad.Body, error) {
	out, err := decad.Cut(b.ctx, body, tool)
	if err != nil {
		return nil, fmt.Errorf("%s: cut: %w", what, err)
	}
	return out, nil
}

// union joins the bodies.
func (b *builder) union(what string, first *decad.Body, rest ...*decad.Body) (*decad.Body, error) {
	out, err := decad.Union(b.ctx, first, rest...)
	if err != nil {
		return nil, fmt.Errorf("%s: union: %w", what, err)
	}
	return out, nil
}

// xRef adds the sketch's reference line: horizontal from the origin to a point 10 away. It anchors angles.
func xRef(s *sketch.Sketch) *sketch.Line {
	x1 := s.CreatePoint(10, 0)
	l := s.CreateLine(s.Origin(), x1)
	l.SetConstruction(true)
	s.AddConstraint(sketch.NewHorizontal(l), sketch.NewDistance(s.Origin(), x1, 10))
	return l
}

// polar creates the point (r at angle a) about the sketch origin, fixed by its distance and the angle of the
// construction line from the origin, measured from ref. A point on a radius-constrained arc passes r = 0 to skip
// the distance.
func polar(s *sketch.Sketch, ref *sketch.Line, r, a float64) *sketch.Point {
	q := s.CreatePoint(r*math.Cos(a), r*math.Sin(a))
	l := s.CreateLine(s.Origin(), q)
	l.SetConstruction(true)
	if r > 0 {
		s.AddConstraint(sketch.NewDistance(s.Origin(), q, r))
	}
	s.AddConstraint(sketch.NewAngle(ref, l, a/deg))
	return q
}

// circleAt draws a circle of radius r about the sketch origin.
func circleAt(s *sketch.Sketch, r float64) *sketch.Circle {
	c := s.CreateCircle(s.Origin(), r)
	s.AddConstraint(sketch.NewRadius(c, r))
	return c
}

// dLoop draws a D outline about the sketch origin: an arc of radius r and a vertical chord at x = flat.
func dLoop(s *sketch.Sketch, r, flat float64) {
	h := math.Sqrt(r*r - flat*flat)
	up := s.CreatePoint(flat, h)
	lo := s.CreatePoint(flat, -h)
	arc := s.CreateArc(s.Origin(), up, lo)
	chord := s.CreateLine(lo, up)
	s.AddConstraint(sketch.NewRadius(arc, r), sketch.NewVertical(chord),
		sketch.NewDistancePointLine(s.Origin(), chord, flat))
}

// dTool extrudes a D press bore tool: a D outline about (x, y) at height z, length long.
func (b *builder) dTool(what string, x, y, z, r, flat, length float64) (*decad.Body, error) {
	s, err := b.flat(what, x, y, z)
	if err != nil {
		return nil, err
	}
	dLoop(s, r, flat)
	return b.extrude(s, 0, length, what)
}

// discAt extrudes a circle of radius r about (x, y) at height z by length.
func (b *builder) discAt(what string, x, y, z, r, length float64) (*decad.Body, error) {
	s, err := b.flat(what, x, y, z)
	if err != nil {
		return nil, err
	}
	circleAt(s, r)
	return b.extrude(s, 0, length, what)
}

// finish chamfers body with edgefinish.Chamfer unless the builder is plain.
func (b *builder) finish(body *decad.Body, offsets []float64, keep edgefinish.LoopFilter) (*decad.Body, error) {
	if b.plain {
		return body, nil
	}
	out, _, err := edgefinish.Chamfer(b.ctx, body, zAxis, offsets, keep, edgefinish.MinChamfer)
	return out, err
}

// sketchFillet rounds the corner at where the line or arc first meets second, with a tangent arc of radius r.
func sketchFillet(s *sketch.Sketch, first, second sketch.Entity, at *sketch.Point, r float64) error {
	var err error
	switch a := first.(type) {
	case *sketch.Line:
		switch b := second.(type) {
		case *sketch.Line:
			_, err = sketch.CreateFillet(s, a, b, at, r)
		case *sketch.Arc:
			_, err = sketch.CreateFillet(s, a, b, at, r)
		}
	case *sketch.Arc:
		switch b := second.(type) {
		case *sketch.Line:
			_, err = sketch.CreateFillet(s, a, b, at, r)
		case *sketch.Arc:
			_, err = sketch.CreateFillet(s, a, b, at, r)
		}
	}
	return err
}
