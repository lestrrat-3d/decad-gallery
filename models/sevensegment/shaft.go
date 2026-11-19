package sevensegment

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Shaft and camshaft part dimensions that are not exported.
const (
	shaftCamRadius    = 3.2 // cam section
	shaftCamFlat      = 2.7 // cam section flat, from the axis
	shaftStubRadius   = 2.5 // knob stub, journals
	shaftStubFlat     = 2.0 // knob stub flat
	shaftCamFlatFrom  = -50.25
	shaftCamFlatTo    = 39.2
	shaftStubFlatFrom = -73.2
	shaftStubFlatTo   = -57.85
	boreStubRadius    = 2.55 // knob and collar ring bore
	boreStubFlat      = 2.05
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

// flatTool is a box that cuts a flat: x from `from` to 8 across y +-4, over local z from z0 to z1.
func (r *run) flatTool(from, z0, z1 float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(z0))
	if err != nil {
		return nil, err
	}
	polygon(s, [][2]float64{{from, -4}, {8, -4}, {8, 4}, {from, 4}})
	return r.extrude(s, 0, z1-z0, decad.Along)
}

// shaft builds the printed camshaft in its local frame (axis on local Z = world +Y, flats toward local +x = world +Z) and
// places it in the world at theta = 0.
func (r *run) shaft() (*decad.Body, error) {
	pts := [][2]float64{ // (r, z) in world Y
		{0, -72.2}, {2.2, -72.2}, {2.5, -71.9}, {2.5, -49.25}, {3.2, -49.25}, {3.2, 39.7}, {3.8, 40.739}, {3.8, 42.75},
		{2.5, 42.75}, {2.5, 47.7}, {2.2, 48.0}, {0, 48.0},
	}
	body, err := r.meridian(pts)
	if err != nil {
		return nil, fmt.Errorf("revolve: %w", err)
	}
	for _, c := range []struct {
		name         string
		flat, z0, z1 float64
	}{
		{"cam flat", shaftCamFlat, shaftCamFlatFrom, shaftCamFlatTo},
		{"stub flat", shaftStubFlat, shaftStubFlatFrom, shaftStubFlatTo},
	} {
		tool, err := r.flatTool(c.flat, c.z0, c.z1)
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

// collarRing builds the separate collar ring pressed on the knob stub: a disc of CollarRingRadius with a D hole,
// CollarRingLength long from world Y -57.9 toward -Y.
func (r *run) collarRing() (*decad.Body, error) {
	f := frontFrame(-57.9)
	s, err := r.sketchOn(across(0))
	if err != nil {
		return nil, err
	}
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), CollarRingRadius), CollarRingRadius))
	dBoreLoop(s, boreStubRadius, boreStubFlat)
	body, err := r.extrude(s, 1, CollarRingLength, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("extrude: %w", err)
	}
	return placement.Place(r.ctx, body, &f)
}

// knob builds the knob: a revolved disc with a blind D bore from its back face.
func (r *run) knob() (*decad.Body, error) {
	pts := [][2]float64{{0, 0}, {KnobRadius - 0.5, 0}, {KnobRadius, 0.5}, {KnobRadius, KnobThickness - 0.5},
		{KnobRadius - 0.5, KnobThickness}, {0, KnobThickness}}
	body, err := r.meridian(pts)
	if err != nil {
		return nil, fmt.Errorf("revolve: %w", err)
	}
	s, err := r.sketchOn(across(-1))
	if err != nil {
		return nil, err
	}
	dBoreLoop(s, boreStubRadius, boreStubFlat)
	bore, err := r.extrude(s, 0, 11.8, decad.Along)
	if err != nil {
		return nil, fmt.Errorf("bore tool: %w", err)
	}
	if body, err = decad.Cut(r.ctx, body, bore); err != nil {
		return nil, fmt.Errorf("cut bore: %w", err)
	}
	f := frontFrame(-61.45)
	return placement.Place(r.ctx, body, &f)
}
