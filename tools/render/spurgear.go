package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

func spurGearHero(ctx context.Context) (solidlens.Scene, error) {
	params := spurgear.DefaultParams()
	params.BoreDiameter = 3
	params.CapChamfer = 0.1
	body, err := spurgear.Build(ctx, decad.New(), params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build spur gear: %w", err)
	}
	mesh, err := body.Tessellate(ctx, units.Millimeters(0.02), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("tessellate spur gear: %w", err)
	}
	return solidlens.Scene{
		Camera: solidlens.Camera{
			Position: solidlens.Vec{X: 20, Y: -27, Z: 32},
			Target:   solidlens.Vec{Z: 5},
			Up:       solidlens.Vec{Z: 1},
			FOV:      30,
		},
		Models: []solidlens.Model{{
			Mesh:     mesh,
			Material: solidlens.Matte(solidlens.RGB(0.26, 0.64, 0.8)),
		}},
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
			{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
		},
		Background: solidlens.RGB(0.012, 0.025, 0.055),
	}, nil
}
