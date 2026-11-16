package turbinefan

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// run builds one fan in the local frame. It records every sketch.
type run struct {
	ctx      context.Context
	doc      *decad.Document
	p        Params
	dm       Dimensions
	sketches []*sketch.Sketch
}

// across is the plane z = const with U = +X and V = +Y, through the point (0, 0, z) of the local frame.
func across(z float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	return f
}

// hubSketch sketches the hub's outline on the back face: the disc of radius HubRadius with the clamp tab below it, every
// join tangent, so the outline has no inside corner except the two root rounds of radius TabRootR where the tab leaves
// the disc, and no step. The tab's sides run at x = ±TabHalfWidth down to SideEnd and end in crankhandle's tail (corner
// arcs and end arc). Three circles of radius m2fastener.HoleRadius are the blade bolts' holes.
func (r *run) hubSketch() (*sketch.Sketch, error) {
	s, err := newSketch(across(0))
	if err != nil {
		return nil, err
	}
	r.sketches = append(r.sketches, s)
	dm := r.dm
	o := s.Origin()
	w := float64(TabHalfWidth)
	pR := s.CreatePoint(dm.RootDisc.X, dm.RootDisc.Y)
	pL := s.CreatePoint(-dm.RootDisc.X, dm.RootDisc.Y)
	tR := s.CreatePoint(w, dm.RootSide)
	tL := s.CreatePoint(-w, dm.RootSide)
	fR := s.CreatePoint(dm.RootCenter.X, dm.RootCenter.Y)
	fL := s.CreatePoint(-dm.RootCenter.X, dm.RootCenter.Y)
	sR := s.CreatePoint(w, dm.SideEnd)
	sL := s.CreatePoint(-w, dm.SideEnd)
	disc := s.CreateArc(o, pR, pL) // counter-clockwise from the right root over +Y to the left root
	filR := s.CreateArc(fR, pR, tR)
	filL := s.CreateArc(fL, tL, pL)
	right, left := s.CreateLine(sR, tR), s.CreateLine(tL, sL)
	s.AddConstraint(
		sketch.NewRadius(disc, HubRadius),
		sketch.NewVertical(right), sketch.NewHorizontalDistance(o, sR, w), sketch.NewVerticalDistance(o, sR, dm.SideEnd),
		sketch.NewVertical(left), sketch.NewHorizontalDistance(o, sL, -w), sketch.NewVerticalDistance(o, sL, dm.SideEnd),
		sketch.NewRadius(filR, TabRootR), sketch.NewTangent(right, filR), sketch.NewTangentCircles(disc, filR, false),
		sketch.NewRadius(filL, TabRootR), sketch.NewTangent(left, filL), sketch.NewTangentCircles(disc, filL, false),
	)
	crankhandle.SketchTail(s, right, left, sR, sL, w, dm.SideEnd)
	for k := 1; k <= BladeCount; k++ {
		u, v := handcrankfan.BladeBoltAt*math.Cos(BladeAngle(k)), handcrankfan.BladeBoltAt*math.Sin(BladeAngle(k))
		c := s.CreatePoint(u, v)
		s.AddConstraint(sketch.NewHorizontalDistance(o, c, u), sketch.NewVerticalDistance(o, c, v))
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, m2fastener.HoleRadius), m2fastener.HoleRadius))
	}
	return s, nil
}

// newSketch starts a sketch on the plane f; extrusions run along f's normal U × V.
func newSketch(f r3.Frame) (*sketch.Sketch, error) {
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	if err != nil {
		return nil, fmt.Errorf("create sketch plane: %w", err)
	}
	s, err := w.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("create sketch: %w", err)
	}
	return s, nil
}

// extrudeOutline solves s, requires it fully constrained and extrudes its one valid profile with the given holes by
// length along its normal.
func (r *run) extrudeOutline(s *sketch.Sketch, holes int, length float64) (*decad.Body, error) {
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
	return decad.Extrude(r.ctx, r.doc, s, chosen, decad.Distance{D: units.Millimeters(length), Dir: decad.Along})
}
