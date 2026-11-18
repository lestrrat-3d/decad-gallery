package clamparm

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
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
	body, err := r.swivelBlank()
	if err != nil {
		return nil, nil, err
	}
	return body, r.sketches, nil
}

func (r *run) swivelBlank() (*decad.Body, error) {
	plan, err := r.swivelPlan()
	if err != nil {
		return nil, err
	}
	elevation, err := r.swivelElevation()
	if err != nil {
		return nil, err
	}
	body, err := decad.Intersect(r.ctx, plan, elevation)
	if err != nil {
		return nil, fmt.Errorf("clamparm: swivel intersection: %w", err)
	}
	if !body.IsSolid() {
		return nil, fmt.Errorf("clamparm: swivel is not a solid")
	}
	return body, nil
}

// buildSwivel builds the finished swivel in its local frame: the blank, its edge finishing, the J1 tongue and hole,
// the J2 fork slot, hole and head pocket, and the edge finishing of the tongue top and the fork faces. It returns the
// unplaced body and the sketches it used.
func buildSwivel(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	body, err := r.swivelBlank()
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.ChamferExcept(ctx, body, axisZ, []float64{0, Section}, edgefinish.Outer,
		skipParallel(axisY), edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: swivel outline chamfer: %w", err)
	}
	tongue, err := r.swivelTongueTool()
	if err != nil {
		return nil, nil, err
	}
	panHole, err := r.holeTool(panLocal)
	if err != nil {
		return nil, nil, err
	}
	slotTool, err := r.slotCutter(SwivelLength-(SlotDepth-SlotRound), SwivelLength+11, -1)
	if err != nil {
		return nil, nil, err
	}
	j2 := forkSite(SwivelLength, Section/2)
	forkHole, err := r.holeTool(j2)
	if err != nil {
		return nil, nil, err
	}
	forkPocket, err := r.pocketTool(j2)
	if err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, tongue, panHole, slotTool, forkHole, forkPocket); err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.ChamferExcept(ctx, body, axisZ, []float64{TongueThickness}, edgefinish.Outer,
		skipParallel(axisY), edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: swivel tongue chamfer: %w", err)
	}
	if body, _, err = edgefinish.ChamferExcept(ctx, body, axisY, []float64{-SlotRound, SlotRound},
		edgefinish.Outer, skipParallel(axisZ), edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: swivel fork chamfer: %w", err)
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: swivel is not a solid")
	}
	return body, r.sketches, nil
}

// swivelTongueTool removes the swivel's material above the J1 tongue: z 8 to 21 for x < 30, with a round of radius 4
// at the inside corner (30, 8). Its sketch is in the XZ plane at y = 12, extruded through y 12 to -12.
func (r *run) swivelTongueTool() (*decad.Body, error) {
	f, err := xz(12)
	if err != nil {
		return nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, err
	}
	// The tongue is 30 long from the J1 axis (steps.md 7.5); it is not the pan axis offset.
	pts := [][2]float64{{-12, TongueThickness}, {swivelTongueLength, TongueThickness}, {swivelTongueLength, 21}, {-12, 21}}
	if err := r.outline(s, pts, []float64{0, ShoulderRound, 0, 0}); err != nil {
		return nil, err
	}
	return r.extrude(s, 0, 24)
}

// buildHead builds the head in its local frame: the J4 axis at the origin, +X along the head, +Z up. It returns the
// unplaced body and the sketches it used.
func buildHead(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	f, err := xz(0)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	origin := s.Origin()
	top, bottom := s.CreatePoint(0, EndRadius), s.CreatePoint(0, -EndRadius)
	s.AddConstraint(sketch.NewHorizontalDistance(origin, top, 0), sketch.NewHorizontalDistance(origin, bottom, 0))
	arc := s.CreateArc(origin, top, bottom)
	s.AddConstraint(sketch.NewRadius(arc, EndRadius))
	c1, c2 := s.CreatePoint(HeadLength, EndRadius), s.CreatePoint(HeadLength, -EndRadius)
	side1, end, side2 := s.CreateLine(top, c1), s.CreateLine(c1, c2), s.CreateLine(c2, bottom)
	s.AddConstraint(sketch.NewHorizontal(side1), sketch.NewHorizontal(side2), sketch.NewVertical(end),
		sketch.NewDistancePointLine(origin, end, HeadLength))
	if _, err := sketch.CreateFillet(s, side1, end, c1, headRound); err != nil {
		return nil, nil, fmt.Errorf("clamparm: head corner round: %w", err)
	}
	if _, err := sketch.CreateFillet(s, end, side2, c2, headRound); err != nil {
		return nil, nil, fmt.Errorf("clamparm: head corner round: %w", err)
	}
	body, err := r.extrudeSymmetric(s, 0, Section/2)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisY, []float64{-Section / 2, Section / 2}, edgefinish.Outer,
		edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: head cap chamfer: %w", err)
	}
	bf, err := frame(r3.NewVec(StudOffset, 0, 9), axisZ, axisX)
	if err != nil {
		return nil, nil, err
	}
	boss, bs, err := fastenertool.RoundedRevolve(ctx, doc, bf, [][2]float64{{0, 0}, {0, 6}, {1, 6}, {3, 4}, {3, 0}})
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: head boss: %w", err)
	}
	r.sketches = append(r.sketches, bs)
	if body, err = decad.Union(ctx, body, boss); err != nil {
		return nil, nil, fmt.Errorf("clamparm: head boss union: %w", err)
	}
	tongueUp, err := r.tongueCutter(1)
	if err != nil {
		return nil, nil, err
	}
	tongueDown, err := r.tongueCutter(-1)
	if err != nil {
		return nil, nil, err
	}
	tongueHole, err := r.holeTool(tongueSite(0))
	if err != nil {
		return nil, nil, err
	}
	studHole, err := r.holeTool(studLocal)
	if err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, tongueUp, tongueDown, tongueHole, studHole); err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.ChamferExcept(ctx, body, axisY, []float64{-TongueThickness / 2,
		TongueThickness / 2}, edgefinish.Outer, skipParallel(axisZ), edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: head tongue chamfer: %w", err)
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: head is not a solid")
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

// headRound is the radius of the head's front corners.
const headRound = 2.0
