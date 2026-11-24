package sevensegment

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
)

// Shaft and camshaft part dimensions that are not exported.
const (
	shaftCamRadius    = 4.8                                                   // cam section
	shaftCamFlat      = 4.05                                                  // cam section flat, from the axis
	shaftJournal      = bearingBoreR                                          // the journals: half the bearing's bore, so it is fixed by the hardware
	shaftStubRadius   = 2.5                                                   // knob stub: the click indexer's size, as the spring fits round it
	shaftStubStepFrom = frontInnerY - bearingWidth - 0.25                     // the stub's cone starts 0.25 outside the front bearing
	shaftStubStepTo   = shaftStubStepFrom - 2.6                               // 30 degrees or less over the 1.5 step
	shaftStubFlat     = 2.0                                                   // knob stub flat
	shaftCamFrom      = frontInnerY + 0.25                                    // the cam section starts 0.25 inside the front wall
	shaftCamEnd       = SegmentSpan + camFaceOffset + CamThickness + LooseFit // the last cam's back face and a Loose gap
	shaftRearFrom     = rearInnerY - 0.25                                     // the rear collar ends 0.25 inside the rear wall
	shaftFrontEnd     = frontWallOuterY - 14.7                                // the knob end
	shaftCamFlatFrom  = shaftCamFrom - 1.0
	shaftCamFlatTo    = shaftCamEnd - 0.5
	shaftStubFlatFrom = shaftFrontEnd - 1.0
	shaftStubFlatTo   = frontWallOuterY - 0.35
	boreStubRadius    = shaftStubRadius + PressFit // knob and collar ring bore
	boreStubFlat      = shaftStubFlat + PressFit
)

// camshaftFrame is a frame whose local Z runs along world +Y through the shaft axis at the given world Y, with local x
// along world +Z (the flats' normal) and local y along world +X.
func camshaftFrame(y float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Y: y, Z: ShaftZ}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	return f
}

// frontFrame is a frame at world Y y on the shaft axis looking along -Y: U = +X, V = +Z, N = -Y.
func frontFrame(y float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{Y: y, Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	return f
}

// flatTool is a box that cuts a flat: x from x0 to x1 across y +-4, over local z from z0 to z1.
func (r *run) flatTool(x0, x1, z0, z1 float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(z0))
	if err != nil {
		return nil, err
	}
	polygon(s, [][2]float64{{x0, -6.0}, {x1, -6.0}, {x1, 6.0}, {x0, 6.0}})
	return r.extrude(s, 0, z1-z0, decad.Along)
}

// shaft builds the printed camshaft in its local frame (axis on local Z = world +Y, flats toward local +x = world +Z) and
// places it in the world at theta = 0.
func (r *run) shaft() (*decad.Body, error) {
	// The journals are the bearings' size, 7 wide. The knob stub steps down to the
	// click indexer's size inside the front wall's lip hole.
	pts := [][2]float64{ // (r, z) in world Y
		{0, shaftFrontEnd}, {shaftStubRadius - 0.3, shaftFrontEnd}, {shaftStubRadius, shaftFrontEnd + 0.3},
		{shaftStubRadius, shaftStubStepTo}, {shaftJournal, shaftStubStepFrom}, {shaftJournal, shaftCamFrom},
		{shaftCamRadius, shaftCamFrom}, {shaftCamRadius, shaftCamEnd}, {RearCollarRadius, shaftCamEnd + 1.5585},
		{RearCollarRadius, shaftRearFrom}, {shaftJournal, shaftRearFrom}, {shaftJournal, rearInnerY + bearingWidth - 0.3},
		{shaftJournal - 0.3, rearInnerY + bearingWidth}, {0, rearInnerY + bearingWidth},
	}
	// The shaft is long, so its sketch is anchored at the display's centre plane (y 0), not at the shortest sides (a chamfer at
	// an end): from there the constraint system keeps a conditioning above the sketch checker's 4e-05. The anchor is off the
	// axis: on it, its distance to the axis line is a redundant constraint.
	body, err := r.meridianAt(pts, &[2]float64{2, 0})
	if err != nil {
		return nil, fmt.Errorf("revolve: %w", err)
	}
	for _, c := range []struct {
		name           string
		x0, x1, z0, z1 float64
	}{
		{"cam flat", shaftCamFlat, 12.0, shaftCamFlatFrom, shaftCamFlatTo},
		{"cam back flat", -12.0, -shaftCamFlat, shaftCamFlatFrom, shaftCamFlatTo},
		{"stub flat", shaftStubFlat, 12.0, shaftStubFlatFrom, shaftStubFlatTo},
	} {
		tool, err := r.flatTool(c.x0, c.x1, c.z0, c.z1)
		if err != nil {
			return nil, fmt.Errorf("%s tool: %w", c.name, err)
		}
		if body, err = decad.Cut(r.ctx, body, tool); err != nil {
			return nil, fmt.Errorf("cut %s: %w", c.name, err)
		}
	}
	f := camshaftFrame(0)
	return placement.Place(r.ctx, body, &f)
}
