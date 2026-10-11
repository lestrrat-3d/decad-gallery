package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

func bevelGearHero(ctx context.Context) (solidlens.Scene, error) {
	params := bevelgear.DefaultParams()
	params.Teeth = 8
	params.MateTeeth = 8
	params.Samples = 5
	params.BoreDiameter = 1
	body, err := bevelgear.Build(ctx, decad.New(), params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build bevel gear: %w", err)
	}
	mesh, err := body.Tessellate(ctx, units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("tessellate bevel gear: %w", err)
	}
	return solidlens.Scene{
		Camera: solidlens.Camera{
			Position: solidlens.Vec{X: -9, Y: -13, Z: 10},
			Target:   solidlens.Vec{X: 4},
			Up:       solidlens.Vec{Z: 1},
			FOV:      27,
		},
		Models: []solidlens.Model{{
			Mesh:     mesh,
			Material: solidlens.Matte(solidlens.RGB(0.17, 0.42, 0.6)),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		}},
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
			{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
		},
		Background: solidlens.RGB(0.88, 0.92, 0.95),
	}, nil
}
