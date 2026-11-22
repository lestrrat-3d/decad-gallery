package sevensegment

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
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

// flatTool is a box that cuts a flat: x from x0 to x1 across y +-4, over local z from z0 to z1.
func (r *run) flatTool(x0, x1, z0, z1 float64) (*decad.Body, error) {
	s, err := r.sketchOn(across(z0))
	if err != nil {
		return nil, err
	}
	polygon(s, [][2]float64{{x0, -4}, {x1, -4}, {x1, 4}, {x0, 4}})
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
		name           string
		x0, x1, z0, z1 float64
	}{
		{"cam flat", shaftCamFlat, 8, shaftCamFlatFrom, shaftCamFlatTo},
		{"cam back flat", -8, -shaftCamFlat, shaftCamFlatFrom, shaftCamFlatTo},
		{"stub flat", shaftStubFlat, 8, shaftStubFlatFrom, shaftStubFlatTo},
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
