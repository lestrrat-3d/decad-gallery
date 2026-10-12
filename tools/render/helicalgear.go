package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/helicalgear"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

var helicalCameras = map[string]solidlens.Camera{
	"hero": {
		Position: solidlens.Vec{X: 24, Y: -30, Z: 34},
		Target:   solidlens.Vec{Z: 5},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
	"top": {
		Position: solidlens.Vec{Z: 47},
		Target:   solidlens.Vec{Z: 5},
		Up:       solidlens.Vec{Y: 1},
		FOV:      30,
	},
	"low": {
		Position: solidlens.Vec{X: 25, Y: -32, Z: 15},
		Target:   solidlens.Vec{Z: 5},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
}

func helicalGearHero(ctx context.Context) (solidlens.Scene, error) {
	return helicalGearView(ctx, helicalCameras["hero"])
}

func helicalGearTop(ctx context.Context) (solidlens.Scene, error) {
	return helicalGearView(ctx, helicalCameras["top"])
}

func helicalGearLow(ctx context.Context) (solidlens.Scene, error) {
	return helicalGearView(ctx, helicalCameras["low"])
}

func helicalGearView(ctx context.Context, camera solidlens.Camera) (solidlens.Scene, error) {
	params := helicalgear.DefaultParams()
	params.BoreDiameter = 3
	params.CapChamfer = 0.1
	body, err := helicalgear.Build(ctx, decad.New(), params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build helical gear: %w", err)
	}
	mesh, err := body.Tessellate(ctx, units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("tessellate helical gear: %w", err)
	}
	return solidlens.Scene{
		Camera: camera,
		Models: []solidlens.Model{{
			Mesh:     mesh,
			Material: solidlens.Matte(solidlens.RGB(0.14, 0.4, 0.58)),
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
