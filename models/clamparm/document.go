package clamparm

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// NewDocument returns a document with the chain tolerance every cut after a union or an intersection needs. opts
// follow the tolerance, so they may override it.
func NewDocument(opts ...decad.DocumentOption) *decad.Document {
	return decad.New(append([]decad.DocumentOption{decad.WithChainTolerance(units.Millimeters(0.01))}, opts...)...)
}

// run builds one part. It collects every sketch the part uses.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// frame wraps r3.NewFrame with the package's error prefix.
func frame(origin, u, v r3.Vec) (r3.Frame, error) {
	f, err := r3.NewFrame(origin, u, v)
	if err != nil {
		return r3.Frame{}, fmt.Errorf("clamparm: create frame: %w", err)
	}
	return f, nil
}

var (
	axisX = r3.NewVec(1, 0, 0)
	axisY = r3.NewVec(0, 1, 0)
	axisZ = r3.NewVec(0, 0, 1)
)

// xy is the frame with X and Y axes at height z: normal +Z.
func xy(z float64) (r3.Frame, error) { return frame(r3.NewVec(0, 0, z), axisX, axisY) }

// xz is the frame with X and Z axes at y: normal -Y.
func xz(y float64) (r3.Frame, error) { return frame(r3.NewVec(0, y, 0), axisX, axisZ) }

// sketchOn creates a sketch on a fresh plane through f and records it.
func (r *run) sketchOn(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("clamparm: create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("clamparm: create sketch: %w", err)
	}
	r.sketches = append(r.sketches, s)
	return s, nil
}

// profile solves s, requires it fully constrained and returns its one valid profile with exactly holes holes.
func (r *run) profile(s *sketch.Sketch, holes int) (*sketch.Profile, error) {
	if _, err := s.Solve(r.ctx); err != nil {
		return nil, fmt.Errorf("clamparm: solve sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(r.ctx, s); err != nil {
		return nil, fmt.Errorf("clamparm: verify sketch: %w", err)
	}
	var chosen *sketch.Profile
	for _, pr := range s.Profiles() {
		if !pr.Valid || len(pr.Holes) != holes {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("clamparm: select profile: several valid profiles with %d holes", holes)
		}
		chosen = pr
	}
	if chosen == nil {
		return nil, fmt.Errorf("clamparm: select profile: no valid profile with %d holes", holes)
	}
	return chosen, nil
}

func (r *run) extrudeWith(s *sketch.Sketch, holes int, e decad.Extent) (*decad.Body, error) {
	pr, err := r.profile(s, holes)
	if err != nil {
		return nil, err
	}
	body, err := decad.Extrude(r.ctx, r.doc, s, pr, e)
	if err != nil {
		return nil, fmt.Errorf("clamparm: extrude: %w", err)
	}
	return body, nil
}

// extrude extrudes the profile of s with holes holes by d along the sketch's normal.
func (r *run) extrude(s *sketch.Sketch, holes int, d float64) (*decad.Body, error) {
	return r.extrudeWith(s, holes, decad.Distance{D: units.Millimeters(d), Dir: decad.Along})
}

// extrudeAgainst extrudes by d against the sketch's normal.
func (r *run) extrudeAgainst(s *sketch.Sketch, holes int, d float64) (*decad.Body, error) {
	return r.extrudeWith(s, holes, decad.Distance{D: units.Millimeters(d), Dir: decad.Against})
}

// extrudeSymmetric extrudes by half to each side of the sketch's plane.
func (r *run) extrudeSymmetric(s *sketch.Sketch, holes int, half float64) (*decad.Body, error) {
	return r.extrudeWith(s, holes, decad.Symmetric{D: units.Millimeters(half)})
}

// outline draws the closed axis-aligned outline through pts and rounds each corner i with radii[i] greater than 0. A
// line along X or Y is held by its distance from the origin; a line through the origin is held by a point on it.
func (r *run) outline(s *sketch.Sketch, pts [][2]float64, radii []float64) error {
	origin := s.Origin()
	points := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		points[i] = s.CreatePoint(p[0], p[1])
	}
	lines := make([]*sketch.Line, len(pts))
	for i := range pts {
		j := (i + 1) % len(pts)
		line := s.CreateLine(points[i], points[j])
		lines[i] = line
		var c float64
		switch {
		case pts[i][1] == pts[j][1]:
			s.AddConstraint(sketch.NewHorizontal(line))
			c = pts[i][1]
		case pts[i][0] == pts[j][0]:
			s.AddConstraint(sketch.NewVertical(line))
			c = pts[i][0]
		default:
			return fmt.Errorf("clamparm: outline line %d is not axis-aligned", i)
		}
		if c == 0 {
			s.AddConstraint(sketch.NewPointOnLine(origin, line))
		} else {
			s.AddConstraint(sketch.NewDistancePointLine(origin, line, math.Abs(c)))
		}
	}
	for i, radius := range radii {
		if radius == 0 {
			continue
		}
		prev := lines[(i+len(pts)-1)%len(pts)]
		if _, err := sketch.CreateFillet(s, prev, lines[i], points[i], radius); err != nil {
			return fmt.Errorf("clamparm: round outline corner %d: %w", i, err)
		}
	}
	return nil
}

// cuts cuts every tool from body, in order, and requires a solid after each.
func (r *run) cuts(body *decad.Body, tools ...*decad.Body) (*decad.Body, error) {
	for i, tool := range tools {
		var err error
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("clamparm: cut %d: %w", i, err)
		}
		if !body.IsSolid() {
			return nil, fmt.Errorf("clamparm: cut %d: body is not solid", i)
		}
	}
	return body, nil
}

// track records the sketch of a fastener tool and passes its body on.
func (r *run) track(body *decad.Body, s *sketch.Sketch, err error) (*decad.Body, error) {
	if err != nil {
		return nil, fmt.Errorf("clamparm: fastener tool: %w", err)
	}
	r.sketches = append(r.sketches, s)
	return body, nil
}

// holeTool is the clearance hole of site.
func (r *run) holeTool(site m5fastener.Site) (*decad.Body, error) {
	body, s, err := m5fastener.HoleTool(r.ctx, r.doc, site)
	return r.track(body, s, err)
}

// pocketTool is the head pocket of site.
func (r *run) pocketTool(site m5fastener.Site) (*decad.Body, error) {
	f, err := pocketFrame(site)
	if err != nil {
		return nil, fmt.Errorf("clamparm: head pocket frame: %w", err)
	}
	body, s, err := m5fastener.HeadPocketTool(r.ctx, r.doc, f)
	return r.track(body, s, err)
}

// slotCutter cuts the fork slot |y| < 4.26 from xEnd back to a round of radius 4.26 centred at cx, through z0 to
// z0 + 22.
func (r *run) slotCutter(cx, xEnd, z0 float64) (*decad.Body, error) {
	f, err := xy(z0)
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	origin := s.Origin()
	a := sketchcheck.DimensionPoint(s, xEnd, SlotRound)
	d := sketchcheck.DimensionPoint(s, xEnd, -SlotRound)
	b := s.CreatePoint(cx, SlotRound)
	c := s.CreatePoint(cx, -SlotRound)
	s.AddConstraint(sketch.NewHorizontalDistance(origin, b, cx), sketch.NewHorizontalDistance(origin, c, cx))
	centre := sketchcheck.DimensionPoint(s, cx, 0)
	arc := s.CreateArc(centre, b, c)
	s.AddConstraint(sketch.NewRadius(arc, SlotRound))
	s.CreateLine(a, b)
	s.CreateLine(c, d)
	s.CreateLine(d, a)
	return r.extrude(s, 0, 22)
}

// tongueCutter removes the material beyond the tongue on side (+1 or -1): |y| > 4 for x < 20, with a round of
// radius 4 at the shoulder.
func (r *run) tongueCutter(side float64) (*decad.Body, error) {
	f, err := xy(-11)
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	pts := [][2]float64{{-11, 4 * side}, {TongueShoulder, 4 * side}, {TongueShoulder, 11 * side}, {-11, 11 * side}}
	if err := r.outline(s, pts, []float64{0, ShoulderRound, 0, 0}); err != nil {
		return nil, err
	}
	return r.extrude(s, 0, 22)
}

// mark builds one part-mark cutter per angle (degrees, counter-clockwise seen from the outer face at height h): a
// hole of diameter 1.5 on radius MarkRadius, through the part.
func (r *run) mark(h float64, angles ...float64) ([]*decad.Body, error) {
	var tools []*decad.Body
	for _, a := range angles {
		f, err := xy(h + 1)
		if err != nil {
			return nil, err
		}
		s, err := r.sketchOn(f)
		if err != nil {
			return nil, err
		}
		rad := a * math.Pi / 180
		centre := sketchcheck.DimensionPoint(s, MarkRadius*math.Cos(rad), MarkRadius*math.Sin(rad))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(centre, markDiameter/2), markDiameter/2))
		tool, err := r.extrudeAgainst(s, 0, h+2)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// skipParallel returns the edge filter that is true for a straight edge whose end points differ along axis only.
func skipParallel(axis r3.Vec) func(*decad.Edge) bool {
	return func(e *decad.Edge) bool {
		if _, ok := e.Curve().(decad.Line3); !ok {
			return false
		}
		d := e.End().Position().Value.Sub(e.Start().Position().Value)
		return d.Sub(axis.Scale(d.Dot(axis))).Len() < 1e-6
	}
}
