package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// m2BoltHero lays the default M2x8 bolt on its side, head toward the camera, so the hex socket shows.
func m2BoltHero(ctx context.Context) (solidlens.Scene, error) {
	// The frame maps the bolt's +Z, from underside to head, to world +X.
	frame, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
	}
	params := m2bolt.DefaultParams()
	params.Placement = &frame
	m, err := m2bolt.Build(ctx, decad.New(), params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build M2 bolt: %w", err)
	}
	mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("mesh M2 bolt: %w", err)
	}
	models := []solidlens.Model{{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 9, Y: -9, Z: 7},
		Target: solidlens.Vec{X: -2}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}
