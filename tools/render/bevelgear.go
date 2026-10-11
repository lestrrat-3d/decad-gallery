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
		Camera: solidlens.Camera{
			Position: solidlens.Vec{X: -11, Y: -11, Z: 11},
			Target:   solidlens.Vec{X: 3, Y: 3},
			Up:       solidlens.Vec{Z: 1},
			FOV:      35,
		},
		Models: models,
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
			{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
		},
		Background: solidlens.RGB(0.88, 0.92, 0.95),
	}, nil
}
