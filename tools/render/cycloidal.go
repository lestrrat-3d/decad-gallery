package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// cycloidalLayout is the exploded layout of the default two-disc drive: each part moves along a staircase,
// 34 mm along X and 16 mm along Z per step, so the parts sit close together and every part stays visible.
var cycloidalLayout = map[string]int{
	cycloidal.HousingName:     -2,
	cycloidal.DiscName(0):     -1,
	cycloidal.DiscName(1):     0,
	cycloidal.CamName:         1,
	cycloidal.OutputPlateName: 2,
}

// cycloidalStep is the offset of one layout step.
var cycloidalStep = r3.NewVec(34, 0, 16)

var cycloidalColors = map[string]solidlens.Color{
	cycloidal.HousingName:     solidlens.RGB(0.38, 0.45, 0.52),
	cycloidal.DiscName(0):     solidlens.RGB(0.17, 0.42, 0.6),
	cycloidal.DiscName(1):     solidlens.RGB(0.25, 0.52, 0.7),
	cycloidal.CamName:         solidlens.RGB(0.72, 0.4, 0.14),
	cycloidal.OutputPlateName: solidlens.RGB(0.55, 0.62, 0.3),
}

// cycloidalModels builds the default two-disc drive and returns one model per body, moved to the exploded layout.
// Each output pin travels with the plate.
func cycloidalModels(ctx context.Context) ([]solidlens.Model, error) {
	params := cycloidal.DefaultParams()
	params.DiscCount = 2
	drive, err := cycloidal.Build(ctx, decad.New(decad.WithChainTolerance(units.Millimeters(0.1))), params)
	if err != nil {
		return nil, fmt.Errorf("build cycloidal drive: %w", err)
	}
	var models []solidlens.Model
	for _, named := range drive.Named() {
		step, ok := cycloidalLayout[named.Name]
		if !ok {
			step = cycloidalLayout[cycloidal.OutputPlateName]
		}
		color, ok := cycloidalColors[named.Name]
		if !ok {
			color = cycloidalColors[cycloidal.OutputPlateName]
		}
		mesh, err := named.Body.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate %s: %w", named.Name, err)
		}
		models = append(models, solidlens.Model{
			Mesh:     placedMesh{TriangleSource: mesh, scale: 1, offset: cycloidalStep.Scale(float64(step))},
			Material: solidlens.Matte(color),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
	}
	return models, nil
}

// cycloidalHeroCamera looks at the staircase from the front and above; fitCamera centers it.
var cycloidalHeroCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 0, Y: -420, Z: 150},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

func cycloidalHero(ctx context.Context) (solidlens.Scene, error) {
	models, err := cycloidalModels(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(fitCamera(cycloidalHeroCamera, models, 16.0/9, bevelViewMargin), models), nil
}

// renderCycloidalReadme writes the index image and the hero.
func renderCycloidalReadme(ctx context.Context, outputDir string) error {
	scene, err := cycloidalHero(ctx)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name          string
		width, height int
	}{{"index.png", 480, 270}, {"hero.png", 1600, 900}} {
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), scene, image.width, image.height); err != nil {
			return fmt.Errorf("render cycloidal %s: %w", image.name, err)
		}
	}
	return nil
}
