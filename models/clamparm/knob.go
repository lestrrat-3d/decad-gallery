package clamparm

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
)

// lobedOutline draws the knobs' shared outline on s: six convex lobes of radius 4 centred on radius 11 at 0, 60, ...
// degrees and six concave valleys of radius 3 centred on radius 13.857 at 30, 90, ... degrees, each valley tangent to
// its two neighbouring lobes. s's origin is the outline's centre.
func lobedOutline(s *sketch.Sketch) {
	const n = 6
	polar := func(radius, degrees float64) (float64, float64) {
		a := degrees * math.Pi / 180
		return radius * math.Cos(a), radius * math.Sin(a)
	}
	var lobeAt, valleyAt [n][2]float64
	var lobeC [n]*sketch.Point
	var valleyC [n]*sketch.Point
	for k := range n {
		lobeAt[k][0], lobeAt[k][1] = polar(lobeCentre, 60*float64(k))
		valleyAt[k][0], valleyAt[k][1] = polar(valleyCentre, 60*float64(k)+30)
		lobeC[k] = sketchcheck.DimensionPoint(s, lobeAt[k][0], lobeAt[k][1])
		valleyC[k] = s.CreatePoint(valleyAt[k][0], valleyAt[k][1])
	}
	toward := func(from, to [2]float64, d float64) (float64, float64) {
		dx, dy := to[0]-from[0], to[1]-from[1]
		l := math.Hypot(dx, dy)
		return from[0] + dx/l*d, from[1] + dy/l*d
	}
	var lobes, valleys [n]*sketch.Arc
	var p, q [n]*sketch.Point
	for k := range n {
		next := (k + 1) % n
		px, py := toward(lobeAt[k], valleyAt[k], lobeRadius)
		qx, qy := toward(lobeAt[next], valleyAt[k], lobeRadius)
		p[k] = s.CreatePoint(px, py)
		q[next] = s.CreatePoint(qx, qy)
	}
	for k := range n {
		next := (k + 1) % n
		lobes[k] = s.CreateArc(lobeC[k], q[k], p[k])
		valleys[k] = s.CreateArc(valleyC[k], q[next], p[k])
		s.AddConstraint(sketch.NewRadius(lobes[k], lobeRadius), sketch.NewRadius(valleys[k], valleyRadius))
	}
	for k := range n {
		next := (k + 1) % n
		s.AddConstraint(
			sketch.NewTangentCircles(lobes[k], valleys[k], false),
			sketch.NewTangentCircles(valleys[k], lobes[next], false),
		)
	}
}

// buildKnob builds one joint knob (J1 to J4) in its local frame: the origin at the inner face's centre on the bolt
// axis, +Z outward and the hex corner on +X. It returns the unplaced body and the sketches it used.
func buildKnob(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	f, err := xy(0)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	lobedOutline(s)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), m5fastener.HoleRadius), m5fastener.HoleRadius))
	body, err := r.extrude(s, 1, KnobHeight)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisZ, []float64{0, KnobHeight}, edgefinish.Outer,
		edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: knob cap chamfer: %w", err)
	}
	floor, err := xy(knobNutFloor)
	if err != nil {
		return nil, nil, err
	}
	pocket, ps, err := m5fastener.NutPocketTool(ctx, doc, floor, knobNutDepth)
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: knob nut pocket: %w", err)
	}
	r.sketches = append(r.sketches, ps)
	marks, err := r.mark(KnobHeight, markFirst)
	if err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, append([]*decad.Body{pocket}, marks...)...); err != nil {
		return nil, nil, err
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: knob is not a solid")
	}
	return body, r.sketches, nil
}

// buildScrewKnob builds one screw knob (the clamp screw's and the stud's) in its local frame: the origin at the inner
// face's centre on the screw axis, +Z outward and the hex corner on +X. The bolt head presses into the outer face's
// pocket and ends flush. It returns the unplaced body and the sketches it used.
func buildScrewKnob(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	f, err := xy(0)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	lobedOutline(s)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), m5fastener.HoleRadius), m5fastener.HoleRadius))
	body, err := r.extrude(s, 1, ScrewKnobHeight)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisZ, []float64{0, ScrewKnobHeight}, edgefinish.Outer,
		edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: screw knob cap chamfer: %w", err)
	}
	floor, err := xy(pressFloor)
	if err != nil {
		return nil, nil, err
	}
	pocket, ps, err := m5fastener.PressPocketTool(ctx, doc, floor)
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: screw knob head pocket: %w", err)
	}
	r.sketches = append(r.sketches, ps)
	marks, err := r.mark(ScrewKnobHeight, markFirst, markFirst+markStep)
	if err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, append([]*decad.Body{pocket}, marks...)...); err != nil {
		return nil, nil, err
	}
	if !body.IsSolid() {
		return nil, nil, fmt.Errorf("clamparm: screw knob is not a solid")
	}
	return body, r.sketches, nil
}
