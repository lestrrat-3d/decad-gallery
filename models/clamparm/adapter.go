package clamparm

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Unexported dimensions of the placeholder adapter, in mm.
const (
	adapterRadius    = 20.0 // pad disc radius
	adapterThickness = 15.0 // pad disc thickness
	adapterNutFloor  = 4.7  // the nut pocket's floor over the mating face
	adapterNutDepth  = 10.3 // the nut pocket's depth, floor to the top face
	recessLow        = 1.0  // the recess cutter's overshoot below the mating face
	recessCone       = 3.695
	recessTop        = 16.0 // the stud hole cutter's top, over the mating face
	studHoleRadius   = 2.8
	recessLowRadius  = 7.495
)

// InterfaceRecessTool builds the cutter of the mount interface's recess and stud hole for an adapter whose mating
// face lies on f, with f's normal pointing out of the arm's interface face (the arm's N). The cutter is a revolve about
// f's normal: a 45 degree cone from RecessRadius at the face to the stud hole radius 2.8 at 3.695 above it, then the
// hole up to 16. It also runs 1 mm below the face so that the cut leaves no skin. It returns the cutter and its
// fully constrained sketch.
func InterfaceRecessTool(ctx context.Context, doc *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch,
	error) {
	m, err := r3.NewFrame(f.Origin(), f.N(), f.U())
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: recess frame: %w", err)
	}
	body, s, err := fastenertool.RoundedRevolve(ctx, doc, m, [][2]float64{
		{-recessLow, 0},
		{-recessLow, recessLowRadius},
		{0, RecessRadius},
		{recessCone, studHoleRadius},
		{recessTop, studHoleRadius},
		{recessTop, 0},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("clamparm: recess tool: %w", err)
	}
	return body, s, nil
}

// buildAdapter builds the placeholder pad adapter in the interface frame I: the mating face at z = 0, the disc up to
// z = 15. It returns the unplaced body and the sketches it used.
func buildAdapter(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, []*sketch.Sketch, error) {
	r := &run{ctx: ctx, doc: doc, p: p}
	f, err := xy(0)
	if err != nil {
		return nil, nil, err
	}
	s, err := r.sketchOn(f)
	if err != nil {
		return nil, nil, err
	}
	circle := s.CreateCircle(s.Origin(), adapterRadius)
	s.AddConstraint(sketch.NewRadius(circle, adapterRadius))
	body, err := r.extrude(s, 0, adapterThickness)
	if err != nil {
		return nil, nil, err
	}
	if body, _, err = edgefinish.Chamfer(ctx, body, axisZ, []float64{0, adapterThickness}, edgefinish.Outer,
		edgefinish.MinChamfer); err != nil {
		return nil, nil, fmt.Errorf("clamparm: adapter rim chamfer: %w", err)
	}

	recess, rs, err := InterfaceRecessTool(ctx, doc, f)
	if _, err = r.track(recess, rs, err); err != nil {
		return nil, nil, err
	}
	nf, err := frame(r3.NewVec(0, 0, adapterNutFloor), axisX, axisY)
	if err != nil {
		return nil, nil, err
	}
	pocket, ps, err := m5fastener.NutPocketTool(ctx, doc, nf, adapterNutDepth)
	if _, err = r.track(pocket, ps, err); err != nil {
		return nil, nil, err
	}
	if body, err = r.cuts(body, recess, pocket); err != nil {
		return nil, nil, err
	}
	return body, r.sketches, nil
}
