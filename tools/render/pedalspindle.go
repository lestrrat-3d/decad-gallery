package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

type pedalSpindleMesh struct {
	name  string
	group string
	body  *decad.Body
	model solidlens.Model
}

// pedalSpindleModels builds the actual assembly, including its removable cover.
func pedalSpindleModels(ctx context.Context) ([]pedalSpindleMesh, error) {
	a, err := pedalspindle.Build(ctx, decad.New(), pedalspindle.DefaultParams())
	if err != nil {
		return nil, fmt.Errorf("build pedal spindle: %w", err)
	}
	colors := map[string]solidlens.Color{
		"frame": solidlens.RGB(0.56, 0.44, 0.28),
		"drive": solidlens.RGB(0.08, 0.32, 0.48),
		"gear":  solidlens.RGB(0.85, 0.39, 0.08),
		"steel": solidlens.RGB(0.48, 0.56, 0.62),
		"bush":  solidlens.RGB(0.68, 0.56, 0.23),
		"pedal": solidlens.RGB(0.48, 0.25, 0.10),
		"guard": solidlens.RGB(0.68, 0.75, 0.80),
	}
	models := make([]pedalSpindleMesh, 0, len(a.Parts))
	for _, part := range a.Parts {
		mesh, err := part.Body.Tessellate(ctx, units.Millimeters(0.15), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate pedal spindle %s: %w", part.Name, err)
		}
		color := colors[part.Group]
		switch part.Name {
		case "input-gear":
			color = colors["drive"]
		case "intermediate-pinion", "intermediate-gear":
			color = solidlens.RGB(0.90, 0.61, 0.16)
		}
		models = append(models, pedalSpindleMesh{name: part.Name, group: part.Group, body: part.Body, model: solidlens.Model{
			Mesh: mesh, Material: solidlens.Matte(color),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(0.04, 0.06, 0.08), Width: 1},
		}})
	}
	return models, nil
}

func pedalSpindleScene(meshes []pedalSpindleMesh, shot string) solidlens.Scene {
	var models []solidlens.Model
	for _, mesh := range meshes {
		if mesh.group == "guard" && shot != "guarded" {
			continue
		}
		if shot == "drive" && (mesh.group == "frame" || mesh.group == "pedal") {
			continue
		}
		if shot == "drive" && (mesh.name == "heel-pin" || mesh.name == "toe-pin") {
			continue
		}
		models = append(models, mesh.model)
	}
	camera := solidlens.Camera{
		Position: solidlens.Vec{X: 820, Y: -1400, Z: 880},
		Target:   solidlens.Vec{X: -60, Y: -50, Z: 270},
		Up:       solidlens.Vec{Z: 1}, FOV: 35,
	}
	if shot == "drive" {
		camera.Position = solidlens.Vec{X: 420, Y: -1100, Z: 690}
		camera.Target = solidlens.Vec{X: -50, Y: -30, Z: 390}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, 0.06), models)
}

func pedalSpindleView(shot string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		meshes, err := pedalSpindleModels(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return pedalSpindleScene(meshes, shot), nil
	}
}

// renderPedalSpindleReadme shares one build across all documented views.
func renderPedalSpindleReadme(ctx context.Context, outputDir string) error {
	meshes, err := pedalSpindleModels(ctx)
	if err != nil {
		return err
	}
	for _, shot := range []string{"hero", "guarded", "drive"} {
		scene := pedalSpindleScene(meshes, shot)
		if err := renderPNG(ctx, filepath.Join(outputDir, shot+".png"), scene, 1600, 900); err != nil {
			return fmt.Errorf("render pedal spindle %s: %w", shot, err)
		}
		if shot == "hero" {
			if err := renderPNG(ctx, filepath.Join(outputDir, "index.png"), scene, 480, 270); err != nil {
				return fmt.Errorf("render pedal spindle index: %w", err)
			}
		}
	}
	for _, shot := range pedalSpindleSheetShots {
		width, height := pedalSpindleSheetSize(shot)
		if err := renderPedalSpindleSheet(ctx, meshes, shot, filepath.Join(outputDir, shot+".png"), width, height); err != nil {
			return err
		}
	}
	return nil
}
