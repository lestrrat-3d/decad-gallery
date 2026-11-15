package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The clutch lies with its shaft axis along +Z, the back plate at the bottom and the cover on top. The hero view cuts
// every body by the plane z = clutchMidPlane, the hub's mid-height, and removes the +Z side, so the pawl heads in the
// ring's teeth, the arms, the back plate and the nuts below show.
const clutchMidPlane = onewayclutch.HubFrom + onewayclutch.HubHeight/2

var (
	clutchHubColor   = solidlens.RGB(.70, .22, .12)
	clutchRingColor  = solidlens.RGB(.20, .22, .25)
	clutchPlateColor = solidlens.RGB(.30, .45, .62)
	clutchBoltColor  = solidlens.RGB(.55, .57, .60)
)

// clutchPart is one body's name, mesh and colour, in drawing order.
type clutchPart struct {
	name  string
	mesh  *decad.Mesh
	color solidlens.Color
}

// buildClutchParts builds the default clutch once and tessellates its ten bodies.
func buildClutchParts(ctx context.Context, p onewayclutch.Params) ([]clutchPart, error) {
	m, err := onewayclutch.Build(ctx, decad.New(), p)
	if err != nil {
		return nil, fmt.Errorf("build one-way clutch: %w", err)
	}
	parts := []clutchPart{
		{"hub", nil, clutchHubColor}, {"ring", nil, clutchRingColor},
		{"back plate", nil, clutchPlateColor}, {"cover", nil, clutchPlateColor},
	}
	bodies := []*decad.Body{m.Hub, m.Ring, m.BackPlate, m.Cover}
	for i := range m.Bolts {
		parts = append(parts, clutchPart{fmt.Sprintf("bolt %d", i+1), nil, clutchBoltColor})
		bodies = append(bodies, m.Bolts[i].Body)
	}
	for i := range m.Nuts {
		parts = append(parts, clutchPart{fmt.Sprintf("nut %d", i+1), nil, clutchBoltColor})
		bodies = append(bodies, m.Nuts[i].Body)
	}
	for i := range parts {
		tol := .02
		if i >= 4 {
			tol = .01
		}
		mesh, err := bodies[i].Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return nil, fmt.Errorf("mesh clutch %s: %w", parts[i].name, err)
		}
		parts[i].mesh = mesh
	}
	return parts, nil
}

// oneWayClutchHero draws the default clutch cut at the hub's mid-height, seen from the front and above.
func oneWayClutchHero(ctx context.Context) (solidlens.Scene, error) {
	parts, err := buildClutchParts(ctx, onewayclutch.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, err
	}
	var models []solidlens.Model
	for _, part := range parts {
		// The cover lies wholly in the removed half.
		if low, _ := meshBounds(part.mesh); low.Z >= clutchMidPlane {
			continue
		}
		c := part.color
		model := solidlens.Model{Mesh: part.mesh, Material: solidlens.Matte(c),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2},
			Clip: &solidlens.ClipPlane{Origin: solidlens.Vec{Z: clutchMidPlane}, Normal: solidlens.Vec{Z: 1},
				Cap: &solidlens.Material{Color: solidlens.RGB(c.R+(1-c.R)*.35, c.G+(1-c.G)*.35, c.B+(1-c.B)*.35),
					Unlit: true}}}
		models = append(models, model)
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 40, Y: -60, Z: 85}, Target: solidlens.Vec{Z: 8},
		Up: solidlens.Vec{Z: 1}, FOV: 35}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}
