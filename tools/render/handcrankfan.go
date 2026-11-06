package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// handCrankFanHero builds the default fan and views it from the operator's side: the rotor faces the camera and
// the crank is on the right.
func handCrankFanHero(ctx context.Context) (solidlens.Scene, error) {
	return handCrankFanScene(ctx, false)
}

// handCrankFanExploded is the same fan pulled apart by Decad's parts.Set.Explode along the joints the model declares.
func handCrankFanExploded(ctx context.Context) (solidlens.Scene, error) {
	return handCrankFanScene(ctx, true)
}

func handCrankFanScene(ctx context.Context, exploded bool) (solidlens.Scene, error) {
	doc := handcrankfan.NewDocument()
	a, err := handcrankfan.Build(ctx, doc, handcrankfan.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build hand-crank fan: %w", err)
	}
	offsets := map[string]r3.Vec{}
	if exploded {
		set, err := a.Set(doc)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("set up hand-crank fan parts: %w", err)
		}
		layout, err := a.Explode(ctx, set)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("explode hand-crank fan: %w", err)
		}
		for _, p := range layout.Parts {
			offsets[p.Name] = p.Offset
		}
	}
	meshes, err := handCrankFanMeshes(ctx, a)
	if err != nil {
		return solidlens.Scene{}, err
	}
	models := make([]solidlens.Model, len(meshes))
	for i, mesh := range meshes {
		models[i] = mesh.model
		if exploded {
			models[i].Mesh = placedMesh{TriangleSource: mesh.model.Mesh, scale: 1, offset: offsets[mesh.part.Name]}
		}
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 330, Y: -560, Z: 330},
		Target: solidlens.Vec{X: 20, Y: -40, Z: 120}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	if exploded {
		camera = solidlens.Camera{Position: solidlens.Vec{X: 440, Y: -620, Z: 560},
			Target: solidlens.Vec{X: 60, Y: -60, Z: 120}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .06), models), nil
}

func renderHandCrankFanReadme(ctx context.Context, outputDir string) error {
	for _, shot := range []struct {
		name          string
		scene         func(context.Context) (solidlens.Scene, error)
		width, height int
	}{
		{"hero.png", handCrankFanHero, 1600, 900}, {"index.png", handCrankFanHero, 480, 270},
		{"exploded.png", handCrankFanExploded, 1600, 900},
	} {
		scene, err := shot.scene(ctx)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.name), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	return nil
}

// fanMesh is one part of the fan with its mesh, colour and outlines.
type fanMesh struct {
	part  handcrankfan.Part
	model solidlens.Model
}

// handCrankFanMeshes meshes every part of the assembly from its real Decad body.
func handCrankFanMeshes(ctx context.Context, a *handcrankfan.Assembly) ([]fanMesh, error) {
	meshes := make([]fanMesh, 0, len(a.Parts))
	for _, part := range a.Parts {
		// The crank shaft's faceted D stub is below Decad's minimum mesh bound at 0.1 mm.
		tolerance := .1
		if part.Name == "crank-shaft" {
			tolerance = .2
		}
		mesh, err := part.Body.Tessellate(ctx, units.Millimeters(tolerance), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("mesh hand-crank fan %s: %w", part.Name, err)
		}
		color := solidlens.RGB(.55, .58, .62)
		switch part.Group {
		case "stand", "housing":
			color = solidlens.RGB(.42, .47, .55)
		case "gear":
			color = solidlens.RGB(.80, .62, .20)
		case "rotor":
			color = solidlens.RGB(.16, .52, .70)
		case "blade":
			color = solidlens.RGB(.30, .68, .85)
		case "hardware":
			color = solidlens.RGB(.88, .90, .92)
		case "arm", "pin":
			color = solidlens.RGB(.70, .22, .12)
		case "grip":
			color = solidlens.RGB(.12, .12, .14)
		}
		meshes = append(meshes, fanMesh{part: part, model: solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1}}})
	}
	return meshes, nil
}
