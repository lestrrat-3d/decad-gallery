package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

var bevelCameras = map[string]solidlens.Camera{
	"hero": {
		Position: solidlens.Vec{X: -43, Y: -43, Z: 43},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Z: 1},
		FOV:      35,
	},
	"top": {
		Position: solidlens.Vec{X: 12, Y: 12, Z: 100},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Y: 1},
		FOV:      42,
	},
	"low": {
		Position: solidlens.Vec{X: -60, Y: -30, Z: 22},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Z: 1},
		FOV:      40,
	},
	"high": {
		Position: solidlens.Vec{X: -50, Y: -20, Z: 75},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Z: 1},
		FOV:      38,
	},
}

func bevelGearHero(ctx context.Context) (solidlens.Scene, error) {
	return bevelGearView(ctx, bevelCameras["hero"])
}

func bevelGearTop(ctx context.Context) (solidlens.Scene, error) {
	return bevelGearView(ctx, bevelCameras["top"])
}

func bevelGearLow(ctx context.Context) (solidlens.Scene, error) {
	return bevelGearView(ctx, bevelCameras["low"])
}

func bevelGearHigh(ctx context.Context) (solidlens.Scene, error) {
	return bevelGearView(ctx, bevelCameras["high"])
}

func bevelGearView(ctx context.Context, camera solidlens.Camera) (solidlens.Scene, error) {
	params := bevelgear.DefaultParams()
	pair, err := bevelgear.BuildPair(ctx, decad.New(), params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build bevel gear pair: %w", err)
	}
	models := make([]solidlens.Model, 0, 2)
	for _, member := range []struct {
		name string
		body *decad.Body
	}{
		{name: "pinion", body: pair.Pinion},
		{name: "driving", body: pair.Driving},
	} {
		mesh, err := member.body.Tessellate(ctx, units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("tessellate %s gear: %w", member.name, err)
		}
		models = append(models, solidlens.Model{
			Mesh:     mesh,
			Material: solidlens.Matte(solidlens.RGB(0.17, 0.42, 0.6)),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
	}
	return solidlens.Scene{
		Camera: camera,
		Models: models,
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
			{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
		},
		Background: solidlens.RGB(0.88, 0.92, 0.95),
	}, nil
}
