package clamparm

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Clamp frame dimensions, in mm.
const (
	frameLeft    = -14.0 // the spine's outer face
	frameBottom  = -73.5 // the bottom jaw's underside
	frameRound   = 2.0   // round on every convex profile corner
	jawRootRound = 7.0   // round on the two jaw roots
	slotCentreX  = -0.74 // the J1 gap's round: centre x
	slotCentreZ  = 18.26 // the J1 gap's round: centre z
	padBottom    = -12.0 // the pad's underside
	padLeft      = -24.5 // the pad's -X face
	padRight     = 15.0  // the pad's +X face
	padHalfWidth = 15.0  // the pad's half width along Y
	padPlanRound = 2.0   // the pad's plan corner round
	screwHoleLow = -74.5 // the clamp screw hole cutter's lowest face
	screwHoleLen = 16.0  // the clamp screw hole cutter's length
	padSocketTop = -4.0  // the pad socket's floor
)

// buildClamp builds the clamp frame in the world frame before Placement: one XZ profile extruded along Y from -15 to
// 15. It returns the unplaced body and the sketches it used.
func buildClamp(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	dm, err := p.Derive()
	if err != nil {
		return nil, nil, err
	}
	r := &run{ctx: ctx, doc: doc, p: p, dm: dm}
	f, err := xz(0)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	if err := r.frameProfile(s); err != nil {
		return nil, nil, err
	}
	body, err := r.extrudeSymmetric(s, 0, FrameWidth/2)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisY, []float64{-FrameWidth / 2, FrameWidth / 2},
		edgefinish.Outer, edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: frame cap chamfer: %w", err)
	}

	panHole, err := r.holeTool(dm.PanSite)
	if err != nil {
		return nil, nil, err
	}
	panPocket, err := r.pocketTool(dm.PanSite)
	if err != nil {
		return nil, nil, err
	}
	hf, err := frame(r3.NewVec(ScrewX, 0, screwHoleLow), axisX, axisY)
	if err != nil {
		return nil, nil, err
	}
	screwHole, hs, err := fastenertool.Cylinder(ctx, doc, hf, m5fastener.HoleRadius, screwHoleLen)
	if screwHole, err = r.track(screwHole, hs, err); err != nil {
		return nil, nil, err
	}
	nf, err := frame(r3.NewVec(ScrewX, 0, jawNutFloor), axisY, r3.NewVec(-1, 0, 0))
	if err != nil {
		return nil, nil, err
	}
	nutPocket, ns, err := m5fastener.NutPocketTool(ctx, doc, nf, jawNutDepth)
	if nutPocket, err = r.track(nutPocket, ns, err); err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, panHole, panPocket, screwHole, nutPocket); err != nil {
		return nil, nil, err
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: frame is not a solid")
	}
	return body, r.sketches, nil
}

// frameProfile draws the frame's C profile on s, in (x, z): the spine, the two jaws and the upper fin with the J1 gap
// between the top jaw and the fin.
func (r *run) frameProfile(s *sketch.Sketch) error {
	origin := s.Origin()
	type vertex struct{ x, z, round float64 }
	vs := []vertex{
		{frameLeft, frameBottom, frameRound}, {45, frameBottom, frameRound}, {45, jawTop, frameRound},
		{0, jawTop, jawRootRound}, {0, 0, jawRootRound}, {50, 0, frameRound}, {50, gapLow, frameRound},
		{slotCentreX, gapLow, 0}, {slotCentreX, gapHigh, 0}, {gapEnd, gapHigh, frameRound},
		{gapEnd, finTop, frameRound}, {frameLeft, finTop, frameRound},
	}
	const slotFrom, slotTo = 7, 8 // S1 and S2, the gap's two ends
	pts := make([]*sketch.Point, len(vs))
	for i, v := range vs {
		pts[i] = s.CreatePoint(v.x, v.z)
	}
	centre := sketchcheck.DimensionPoint(s, slotCentreX, slotCentreZ)
	s.AddConstraint(sketch.NewHorizontalDistance(origin, pts[slotFrom], slotCentreX),
		sketch.NewHorizontalDistance(origin, pts[slotTo], slotCentreX))
	arc := s.CreateArc(centre, pts[slotTo], pts[slotFrom])
	s.AddConstraint(sketch.NewRadius(arc, SlotRound))

	lines := make([]*sketch.Line, len(vs))
	for i := range vs {
		j := (i + 1) % len(vs)
		if i == slotFrom {
			continue // the arc closes S1 to S2
		}
		line := s.CreateLine(pts[i], pts[j])
		lines[i] = line
		a, b := vs[i], vs[j]
		var c float64
		var horizontal bool
		switch {
		case a.z == b.z:
			s.AddConstraint(sketch.NewHorizontal(line))
			c, horizontal = a.z, true
		case a.x == b.x:
			s.AddConstraint(sketch.NewVertical(line))
			c = a.x
		default:
			return fmt.Errorf("clamparm: frame outline line %d is not axis-aligned", i)
		}
		// The two lines that end at the gap take their height from its round.
		if i == slotFrom-1 || i == slotTo {
			if !horizontal {
				return fmt.Errorf("clamparm: frame gap line %d is not horizontal", i)
			}
			continue
		}
		if c == 0 {
			s.AddConstraint(sketch.NewPointOnLine(origin, line))
		} else {
			s.AddConstraint(sketch.NewDistancePointLine(origin, line, math.Abs(c)))
		}
	}
	for i, v := range vs {
		if v.round == 0 {
			continue
		}
		prev := lines[(i+len(vs)-1)%len(vs)]
		if _, err := sketch.CreateFillet(s, prev, lines[i], pts[i], v.round); err != nil {
			return fmt.Errorf("clamparm: round frame corner %d: %w", i, err)
		}
	}
	return nil
}

// buildPad builds the pad in its local frame: the origin at the top face's centre on the screw axis, +Z up, the body
// spanning z -12 to 0. It returns the unplaced body and the sketches it used.
func buildPad(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	f, err := xy(padBottom)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	pts := [][2]float64{
		{padLeft, -padHalfWidth}, {padRight, -padHalfWidth}, {padRight, padHalfWidth}, {padLeft, padHalfWidth},
	}
	if err := r.outline(s, pts, []float64{padPlanRound, padPlanRound, padPlanRound, padPlanRound}); err != nil {
		return nil, nil, err
	}
	body, err := r.extrude(s, 0, -padBottom)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisZ, []float64{padBottom, 0}, edgefinish.Outer,
		edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: pad cap chamfer: %w", err)
	}
	sf, err := frame(r3.NewVec(0, 0, padSocketTop), axisX, r3.NewVec(0, -1, 0))
	if err != nil {
		return nil, nil, err
	}
	socket, ss, err := m5fastener.RoundedSocketTool(ctx, doc, sf, padSocketRadius, padSocketDepth, padSocketRound)
	if socket, err = r.track(socket, ss, err); err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, socket); err != nil {
		return nil, nil, err
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: pad is not a solid")
	}
	return body, r.sketches, nil
}
