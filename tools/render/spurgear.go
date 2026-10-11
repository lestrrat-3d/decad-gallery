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
	return spurGearView(ctx, solidlens.Camera{
		Position: solidlens.Vec{X: 20, Y: -27, Z: 29.5},
		Target:   solidlens.Vec{Z: 2.5},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	})
}

func spurGearTop(ctx context.Context) (solidlens.Scene, error) {
	return spurGearView(ctx, solidlens.Camera{
		Position: solidlens.Vec{Z: 45},
		Target:   solidlens.Vec{Z: 2.5},
		Up:       solidlens.Vec{Y: 1},
		FOV:      30,
	})
}

func spurGearLow(ctx context.Context) (solidlens.Scene, error) {
	return spurGearView(ctx, solidlens.Camera{
		Position: solidlens.Vec{X: 22, Y: -25, Z: 10},
		Target:   solidlens.Vec{Z: 2.5},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	})
}

func spurGearBack(ctx context.Context) (solidlens.Scene, error) {
	return spurGearView(ctx, solidlens.Camera{
		Position: solidlens.Vec{X: -20, Y: 25, Z: 30},
		Target:   solidlens.Vec{Z: 2.5},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	})
}

func spurGearBottom(ctx context.Context) (solidlens.Scene, error) {
	return spurGearView(ctx, solidlens.Camera{
		Position: solidlens.Vec{X: 22, Y: -27, Z: -22},
		Target:   solidlens.Vec{Z: 2.5},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	})
}

func spurGearView(ctx context.Context, camera solidlens.Camera) (solidlens.Scene, error) {
	params := spurgear.DefaultParams()
	params.Thickness = 5
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
	lights := []solidlens.DirectionalLight{
		{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
		{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
	}
	if camera.Position.Z < 0 {
		lights = append(lights, solidlens.DirectionalLight{
			Direction: solidlens.Vec{X: 0.5, Y: -0.3, Z: 1},
			Color:     solidlens.RGB(1, 1, 1),
			Intensity: 0.9,
		})
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
		DirectionalLights: lights,
		Background: solidlens.RGB(0.88, 0.92, 0.95),
	}, nil
}
