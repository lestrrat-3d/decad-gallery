package sevensegment

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// axisLength is the length of the construction lines that hold polar directions.
const axisLength = Scale * 30.0

// run builds the parts in their local frames. It records every sketch.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// sketchOn starts a sketch on the plane f; extrusions run along f's normal U x V.
func (r *run) sketchOn(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	r.sketches = append(r.sketches, s)
	return s, nil
}

// profile solves s, requires it fully constrained and returns its one valid profile with the given number of holes.
func (r *run) profile(s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(r.ctx); err != nil {
		return nil, fmt.Errorf("solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(r.ctx, s); err != nil {
		return nil, fmt.Errorf("verify sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != holes {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("several valid profiles with %d holes", holes)
		}
		chosen = pr
	}
	if chosen == nil {
		return nil, fmt.Errorf("no valid profile with %d holes", holes)
	}
	return chosen, nil
}

// extrude extrudes the profile of s with the given holes by length in direction dir.
func (r *run) extrude(s *sketch.Sketch, holes int, length float64, dir decad.Direction) (*decad.Body, error) {
	pr, err := r.profile(s, holes)
	if err != nil {
		return nil, err
	}
	return decad.Extrude(r.ctx, r.doc, s, pr, decad.Distance{D: units.Millimeters(length), Dir: dir})
}

// across is the plane z = const of a local frame, with U = +X and V = +Y.
func across(z float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	return f
}

// polarSketch holds a sketch's reference line, from which polar construction axes take their angles.
type polarSketch struct {
	s   *sketch.Sketch
	ref *sketch.Line
}

// newPolar adds the horizontal reference line from the sketch origin.
func newPolar(s *sketch.Sketch) *polarSketch {
	end := s.CreatePoint(axisLength, 0)
	ref := s.CreateLine(s.Origin(), end)
	ref.SetConstruction(true)
	s.AddConstraint(sketch.NewHorizontal(ref), sketch.NewHorizontalDistance(s.Origin(), end, axisLength))
	return &polarSketch{s: s, ref: ref}
}

// axis adds a construction line from the origin at angle theta degrees.
func (ps *polarSketch) axis(theta float64) *sketch.Line {
	a := wrap360(theta)
	if a > 180 {
		a -= 360
	}
	e := polar(axisLength, a)
	end := ps.s.CreatePoint(e.X, e.Y)
	l := ps.s.CreateLine(ps.s.Origin(), end)
	l.SetConstruction(true)
	ps.s.AddConstraint(sketch.NewAngle(ps.ref, l, a), sketch.NewDistance(ps.s.Origin(), end, axisLength))
	return l
}

// arcBetween creates the arc about center from a to b, whichever way round is the short one (at most 180 degrees).
func arcBetween(s *sketch.Sketch, center r3.Vec, cp *sketch.Point, a, b r3.Vec, pa, pb *sketch.Point) *sketch.Arc {
	u, v := a.Sub(center), b.Sub(center)
	if u.X*v.Y-u.Y*v.X >= 0 {
		return s.CreateArc(cp, pa, pb)
	}
	return s.CreateArc(cp, pb, pa)
}

// meridian sketches the closed polygon through the (r, z) vertices on the half plane holding the Z axis: U runs along +Z
// and V is the radius +X, so every side is held by its direction and its distance from an anchor (see polygon). It revolves the polygon a full turn about Z.
func (r *run) meridian(vertices [][2]float64) (*decad.Body, error) {
	return r.meridianAt(vertices, nil)
}

// meridianAt is meridian with the polygon's anchor placed at anchor, a (radius, z) point, when it is not nil: a part as long
// as the camshaft needs its anchor near its middle for the sketch to stay well conditioned.
func (r *run) meridianAt(vertices [][2]float64, anchor *[2]float64) (*decad.Body, error) {
	f, err := r3.NewFrame(r3.Vec{}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	uv := make([][2]float64, len(vertices))
	for i, v := range vertices {
		uv[i] = [2]float64{v[1], v[0]}
	}
	if anchor == nil {
		polygon(s, uv)
	} else {
		polygonAnchored(s, uv, anchor[1], anchor[0])
	}
	pr, err := r.profile(s, 0)
	if err != nil {
		return nil, err
	}
	return decad.Revolve(r.ctx, r.doc, s, pr, decad.ConstructionAxis{Origin: r3.Vec{}, Dir: r3.Vec{Z: 1}},
		decad.FullRevolution{})
}

// quarterRound returns n+1 (r, z) points of the round of radius rad about (cr, cz) from angle a0 to a1 degrees.
func quarterRound(cr, cz, rad, a0, a1 float64, n int) [][2]float64 {
	out := make([][2]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		a := (a0 + (a1-a0)*float64(i)/float64(n)) * math.Pi / 180
		out = append(out, [2]float64{cr + rad*math.Cos(a), cz + rad*math.Sin(a)})
	}
	return out
}

// dBoreLoop adds the D bore to s: an arc of radius rb the long way round through -V, closed by its flat at v = +fb.
func dBoreLoop(s *sketch.Sketch, rb, fb float64) {
	w := math.Sqrt(rb*rb - fb*fb)
	o := s.Origin()
	q1, q2 := s.CreatePoint(-w, fb), s.CreatePoint(w, fb)
	bore := s.CreateArc(o, q1, q2)
	flat := s.CreateLine(q2, q1)
	s.AddConstraint(sketch.NewRadius(bore, rb), sketch.NewHorizontal(flat), sketch.NewDistancePointLine(o, flat, fb))
}

// doubleDBoreLoop adds a double-D bore to s: two arcs of radius rb about the origin, closed by flats at v = +fb and
// v = -fb.
func doubleDBoreLoop(s *sketch.Sketch, rb, fb float64) {
	w := math.Sqrt(rb*rb - fb*fb)
	o := s.Origin()
	q1, q2 := s.CreatePoint(-w, fb), s.CreatePoint(w, fb)
	q3, q4 := s.CreatePoint(w, -fb), s.CreatePoint(-w, -fb)
	right := s.CreateArc(o, q3, q2)
	top := s.CreateLine(q2, q1)
	left := s.CreateArc(o, q1, q4)
	bottom := s.CreateLine(q4, q3)
	s.AddConstraint(sketch.NewRadius(right, rb), sketch.NewRadius(left, rb),
		sketch.NewHorizontal(top), sketch.NewDistancePointLine(o, top, fb),
		sketch.NewHorizontal(bottom), sketch.NewDistancePointLine(o, bottom, fb))
}

// circleSketch cuts a round hole: a sketch on across(z0) with one circle of radius rad centred at (x, y), the centre
// placed by horizontal and vertical distances from the origin. It extrudes the circle by length along +Z.
func (r *run) holeTool(z0, x, y, rad, length float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(z0))
	if err != nil {
		return nil, err
	}
	c := sketchcheck.DimensionPoint(s, x, y)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, rad), rad))
	return r.extrude(s, 0, length, decad.Along)
}

// polygon draws a closed polygon whose vertices are the meeting points of its lines. Each line is held by its direction
// (horizontal, vertical, or an angle from a reference line) and by its distance from an anchor point, so no computed
// vertex is fixed and the sketch is fully constrained. The anchor is the polygon's middle, placed from the sketch origin;
// measuring from it and not from the origin keeps the system well conditioned for a polygon far from the origin. It
// returns the points and the lines, edge i from point i to point i+1.
func polygon(s *sketch.Sketch, v [][2]float64) ([]*sketch.Point, []*sketch.Line) {
	mx, my := polygonAnchor(v)
	return polygonAnchored(s, v, mx, my)
}

// polygonAnchor returns the anchor polygon uses: the average of the midpoints of the four shortest sides, to 0.01.
func polygonAnchor(v [][2]float64) (float64, float64) { // The anchor lies among the shortest sides, where the sketch has its finest features.
	type side struct{ mx, my, length float64 }
	sides := make([]side, len(v))
	for i := range v {
		a, b := v[i], v[(i+1)%len(v)]
		sides[i] = side{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, math.Hypot(b[0]-a[0], b[1]-a[1])}
	}
	sort.Slice(sides, func(i, j int) bool { return sides[i].length < sides[j].length })
	mx, my := 0.0, 0.0
	n := min(4, len(sides))
	for _, sd := range sides[:n] {
		mx += sd.mx
		my += sd.my
	}
	return math.Round(mx/float64(n)*100) / 100, math.Round(my/float64(n)*100) / 100
}

// polygonAnchored is polygon with its anchor at (mx, my).
func polygonAnchored(s *sketch.Sketch, v [][2]float64, mx, my float64) ([]*sketch.Point, []*sketch.Line) {
	var ref *sketch.Line
	anchor := sketchcheck.DimensionPoint(s, mx, my)
	// A run of short slanted sides is a faceted arc (the kernel refuses true arcs where the model uses these chords): its
	// vertices are the arc's sample points and are placed from the anchor. Sides that touch one keep only their direction.
	faceted := make([]bool, len(v))
	short := func(i int) bool {
		a, b := v[i], v[(i+1)%len(v)]
		return math.Hypot(b[0]-a[0], b[1]-a[1]) < 1.0 && math.Abs(b[0]-a[0]) > 1e-9 && math.Abs(b[1]-a[1]) > 1e-9
	}
	for i := range v {
		if short(i) && short((i+1)%len(v)) {
			faceted[i], faceted[(i+1)%len(v)], faceted[(i+2)%len(v)] = true, true, true
		}
	}
	pts := make([]*sketch.Point, len(v))
	for i, q := range v {
		if faceted[i] {
			pts[i] = s.CreatePoint(q[0], q[1])
			s.AddConstraint(sketch.NewHorizontalDistance(anchor, pts[i], q[0]-mx), sketch.NewVerticalDistance(anchor, pts[i], q[1]-my))
			continue
		}
		pts[i] = s.CreatePoint(q[0], q[1])
	}
	lines := make([]*sketch.Line, len(v))
	for i := range v {
		a, b := v[i], v[(i+1)%len(v)]
		line := s.CreateLine(pts[i], pts[(i+1)%len(v)])
		lines[i] = line
		if faceted[i] && faceted[(i+1)%len(v)] {
			continue
		}
		dx, dy := b[0]-a[0], b[1]-a[1]
		dist := math.Abs((a[0]-mx)*dy-(a[1]-my)*dx) / math.Hypot(dx, dy)
		anchored := faceted[i] || faceted[(i+1)%len(v)]
		switch {
		case math.Abs(dy) < 1e-9:
			s.AddConstraint(sketch.NewHorizontal(line))
		case math.Abs(dx) < 1e-9:
			s.AddConstraint(sketch.NewVertical(line))
		default:
			if ref == nil {
				ref = newPolar(s).ref
			}
			s.AddConstraint(sketch.NewAngle(ref, line, math.Atan2(dy, dx)*180/math.Pi))
		}
		if !anchored {
			s.AddConstraint(sketch.NewDistancePointLine(anchor, line, dist))
		}
	}
	return pts, lines
}
