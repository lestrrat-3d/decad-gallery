package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The hero image pulls the default two-stage gearbox apart along its axis: each stage rises by
// planetaryStageLift over the one below, each carrier lifts planetaryCarrierLift off its planets,
// and the ring floats planetaryRingLift above the stack's first stage.
const (
	planetaryStageLift   = 30
	planetaryCarrierLift = 17
	planetaryRingLift    = 78
)

var (
	planetaryRingColor    = solidlens.RGB(0.38, 0.45, 0.52)
	planetarySunColor     = solidlens.RGB(0.72, 0.4, 0.14)
	planetaryPlanetColor  = solidlens.RGB(0.17, 0.42, 0.6)
	planetaryCarrierColor = solidlens.RGB(0.55, 0.62, 0.3)
)

// planetaryModels builds the default gearbox with two stages and returns one model per body,
// moved to the exploded layout.
func planetaryModels(ctx context.Context) ([]solidlens.Model, error) {
	params := planetarygearbox.DefaultParams()
	params.Stages = 2
	box, err := planetarygearbox.Build(ctx, decad.New(), params)
	if err != nil {
		return nil, fmt.Errorf("build planetary gearbox: %w", err)
	}
	var models []solidlens.Model
	add := func(name string, body *decad.Body, color solidlens.Color, lift float64) error {
		mesh, err := body.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return fmt.Errorf("tessellate %s: %w", name, err)
		}
		models = append(models, solidlens.Model{
			Mesh:     placedMesh{TriangleSource: mesh, scale: 1, offset: r3.NewVec(0, 0, lift)},
			Material: solidlens.Matte(color),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
		return nil
	}
	if err := add(planetarygearbox.RingName, box.Ring, planetaryRingColor, planetaryRingLift); err != nil {
		return nil, err
	}
	for k, stage := range box.Stages {
		lift := float64(k) * planetaryStageLift
		if err := add(planetarygearbox.SunName(k), stage.Sun, planetarySunColor, lift); err != nil {
			return nil, err
		}
		for i, planet := range stage.Planets {
			if err := add(planetarygearbox.PlanetName(k, i), planet, planetaryPlanetColor, lift); err != nil {
				return nil, err
			}
		}
		if err := add(planetarygearbox.CarrierName(k), stage.Carrier, planetaryCarrierColor,
			lift+planetaryCarrierLift); err != nil {
			return nil, err
		}
	}
	return models, nil
}

// planetaryHeroCamera looks at the stack from the front and above; fitCamera centers it.
var planetaryHeroCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 90, Y: -340, Z: 120},
	Target:   solidlens.Vec{Z: 50},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

func planetaryGearboxHero(ctx context.Context) (solidlens.Scene, error) {
	models, err := planetaryModels(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(fitCamera(planetaryHeroCamera, models, 16.0/9, bevelViewMargin), models), nil
}

// renderPlanetaryGearboxReadme writes the index image and the hero from one build.
func renderPlanetaryGearboxReadme(ctx context.Context, outputDir string) error {
	scene, err := planetaryGearboxHero(ctx)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name          string
		width, height int
	}{{"index.png", 480, 270}, {"hero.png", 1600, 900}} {
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), scene, image.width, image.height); err != nil {
			return fmt.Errorf("render planetary gearbox %s: %w", image.name, err)
		}
	}
	return nil
}
