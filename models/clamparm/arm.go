package clamparm

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// buildLink builds one link in its local frame: the tongue's axis at the origin and the fork's axis at (L, 0, 0),
// both along Y. It returns the unplaced body and the sketches it used. Links A and B are identical copies, each built
// with its own sketches.
func buildLink(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	l := p.LinkLength
	f, err := xz(0)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	slot, err := s.CreateSlot(0, 0, l, 0, EndRadius)
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: link outline: %w", err)
	}
	s.AddConstraint(
		sketch.NewCoincident(slot.C1, s.Origin()),
		sketch.NewVerticalDistance(s.Origin(), slot.C2, 0),
		sketch.NewHorizontalDistance(s.Origin(), slot.C2, l),
		sketch.NewRadius(slot.A1, EndRadius),
	)
	body, err := r.extrudeSymmetric(s, 0, 10)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisY, []float64{-10, 10}, edgefinish.Outer,
		edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: link cap chamfer: %w", err)
	}

	tongueUp, err := r.tongueCutter(1)
	if err != nil {
		return nil, nil, err
	}
	tongueDown, err := r.tongueCutter(-1)
	if err != nil {
		return nil, nil, err
	}
	slotTool, err := r.slotCutter(l-(SlotDepth-SlotRound), l+11, -11)
	if err != nil {
		return nil, nil, err
	}
	tongueHole, err := r.holeTool(tongueSite(0))
	if err != nil {
		return nil, nil, err
	}
	forkHole, err := r.holeTool(forkSite(l, 0))
	if err != nil {
		return nil, nil, err
	}
	forkPocket, err := r.pocketTool(forkSite(l, 0))
	if err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, tongueUp, tongueDown, slotTool, tongueHole, forkHole, forkPocket); err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.ChamferExcept(ctx, body, axisY, []float64{-SlotRound, -TongueThickness / 2,
		TongueThickness / 2, SlotRound}, edgefinish.Outer, skipParallel(axisZ), edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: link tongue and fork chamfer: %w", err)
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: link is not a solid")
	}
	return body, r.sketches, nil
}

// swivelBlank builds the swivel's body in its local frame: the intersection of a plan prism and an elevation prism,
// 20 thick, round about J1 in plan and about J2 in elevation. The plan prism is taller (z -1 to 21) and longer (end at
// x = 81) than the swivel so that none of its faces lies in a plane of the elevation prism or touches its curve.
func swivelBlank(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	plan, err := r.swivelPlan()
	if err != nil {
		return nil, nil, err
	}
	elevation, err := r.swivelElevation()
	if err != nil {
		return nil, nil, err
	}
	body, err := decad.Intersect(ctx, plan, elevation)
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: swivel intersection: %w", err)
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: swivel is not a solid")
	}
	return body, r.sketches, nil
}

// swivelPlan is the plan prism: a half circle of radius 10 about the origin, sides y = +-10 to x = 81, z -1 to 21.
func (r *run) swivelPlan() (*decad.Body, error) {
	f, err := xy(-1)
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	origin := s.Origin()
	top, bottom := s.CreatePoint(0, EndRadius), s.CreatePoint(0, -EndRadius)
	s.AddConstraint(sketch.NewHorizontalDistance(origin, top, 0), sketch.NewHorizontalDistance(origin, bottom, 0))
	arc := s.CreateArc(origin, top, bottom)
	s.AddConstraint(sketch.NewRadius(arc, EndRadius))
	c1, c2 := s.CreatePoint(81, EndRadius), s.CreatePoint(81, -EndRadius)
	side1, end, side2 := s.CreateLine(top, c1), s.CreateLine(c1, c2), s.CreateLine(c2, bottom)
	s.AddConstraint(sketch.NewHorizontal(side1), sketch.NewHorizontal(side2), sketch.NewVertical(end),
		sketch.NewDistancePointLine(origin, end, 81))
	return r.extrude(s, 0, 22)
}

// swivelElevation is the elevation prism: x -11 to 70 at z 0 to 20 closed by a half circle of radius 10 about
// (70, 10), y 11 down to -11.
func (r *run) swivelElevation() (*decad.Body, error) {
	f, err := xz(11)
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	origin := s.Origin()
	a, b := s.CreatePoint(-11, 0), s.CreatePoint(70, 0)
	c, d := s.CreatePoint(70, 20), s.CreatePoint(-11, 20)
	s.AddConstraint(sketch.NewHorizontalDistance(origin, b, 70), sketch.NewHorizontalDistance(origin, c, 70))
	centre := sketchcheck.DimensionPoint(s, 70, 10)
	arc := s.CreateArc(centre, b, c)
	s.AddConstraint(sketch.NewRadius(arc, EndRadius))
	ab, cd, da := s.CreateLine(a, b), s.CreateLine(c, d), s.CreateLine(d, a)
	s.AddConstraint(sketch.NewHorizontal(ab), sketch.NewHorizontal(cd), sketch.NewVertical(da),
		sketch.NewDistancePointLine(origin, da, 11))
	return r.extrude(s, 0, 22)
}
