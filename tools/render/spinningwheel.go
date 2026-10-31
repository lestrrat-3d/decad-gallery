package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

type wheelMesh struct {
	part  spinningwheel.Part
	model solidlens.Model
}

func spinningWheelModels(ctx context.Context) ([]wheelMesh, spinningwheel.State, error) {
	a, err := spinningwheel.Build(ctx, decad.New(), spinningwheel.DefaultParams())
	if err != nil {
		return nil, spinningwheel.State{}, err
	}
	meshes := make([]wheelMesh, 0, len(a.Parts))
	for _, part := range a.Parts {
		chord := .12
		if part.Name == "flyer-drive-whorl" {
			chord = .01
		}
		mesh, err := part.Body.Tessellate(ctx, units.Millimeters(chord), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, a.State, fmt.Errorf("tessellate spinning wheel %s: %w", part.Name, err)
		}
		color := solidlens.RGB(.65, .43, .23)
		switch part.Motion {
		case "wheel":
			color = solidlens.RGB(.62, .35, .16)
		case "flyer":
			color = solidlens.RGB(.69, .22, .08)
		case "bobbin":
			color = solidlens.RGB(.86, .69, .42)
		case "treadle":
			color = solidlens.RGB(.49, .27, .12)
		case "footman", "gimbal":
			color = solidlens.RGB(.75, .37, .13)
		}
		n := part.Name
		switch {
		case strings.Contains(n, "witness") || strings.Contains(n, "marker"):
			color = solidlens.RGB(.96, .44, .04)
		case strings.Contains(n, "pack"):
			color = solidlens.RGB(.07, .45, .51)
		case part.Motion == "flyer" && strings.Contains(n, "guide"):
			color = solidlens.RGB(.44, .49, .53)
		case strings.Contains(n, "leader") || part.Group == "yarn":
			color = solidlens.RGB(.02, .57, .62)
		case strings.Contains(n, "drive-band") || strings.Contains(n, "belt"):
			color = solidlens.RGB(.04, .10, .15)
		case part.Group == "brake-band" || strings.Contains(n, "brake-band") || strings.Contains(n, "brake-cord"):
			color = solidlens.RGB(.77, .04, .08)
		case part.Group == "bush" || strings.Contains(n, "bush"):
			color = solidlens.RGB(.9, .9, .84)
		case part.Group == "guide" || part.Group == "yarn-guide" || strings.Contains(n, "shaft") || strings.Contains(n, "bearing") || strings.Contains(n, "collar") ||
			strings.Contains(n, "pin") || strings.Contains(n, "spacer") || strings.Contains(n, "screw") ||
			strings.Contains(n, "bolt") || strings.Contains(n, "nut") || strings.Contains(n, "spring") ||
			strings.Contains(n, "eye") || strings.Contains(n, "stem"):
			color = solidlens.RGB(.44, .49, .53)
		}
		meshes = append(meshes, wheelMesh{part: part, model: solidlens.Model{Mesh: mesh,
			Material: solidlens.Matte(color), Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.12, .10, .08), Width: .6}}})
	}
	return meshes, a.State, nil
}

func spinningWheelScene(meshes []wheelMesh, shot string) solidlens.Scene {
	var models []solidlens.Model
	for _, mesh := range meshes {
		if shot == "flyer" || shot == "brake" || shot == "orifice" {
			box, err := mesh.part.Body.Bounds()
			if err != nil || box.Max.X > -25 || box.Min.Z < 520 {
				continue
			}
		}
		model := mesh.model
		if shot == "orifice" && strings.Contains(mesh.part.Name, "shaft") {
			// Clip the actual shaft mesh along its axis to expose the blind bore and radial outlet.
			model.Clip = &solidlens.ClipPlane{Origin: solidlens.Vec{X: -140}, Normal: solidlens.Vec{X: 1}}
		}
		models = append(models, model)
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 1000, Y: -1600, Z: 1050},
		Target: solidlens.Vec{X: 30, Y: 20, Z: 430}, Up: solidlens.Vec{Z: 1}, FOV: 32}
	switch shot {
	case "rear":
		camera.Position = solidlens.Vec{X: -900, Y: 1600, Z: 1000}
	case "side":
		camera.Position = solidlens.Vec{X: 1600, Y: 0, Z: 430}
	case "top":
		camera.Position = solidlens.Vec{X: 30, Y: 20, Z: 2000}
		camera.Up = solidlens.Vec{Y: 1}
	case "flyer":
		camera.Position = solidlens.Vec{X: -850, Y: -1100, Z: 1000}
		camera.Target = solidlens.Vec{X: -140, Y: 110, Z: 650}
	case "brake":
		camera.Position = solidlens.Vec{X: 250, Y: 800, Z: 940}
		camera.Target = solidlens.Vec{X: -140, Y: 90, Z: 640}
	case "orifice":
		camera.Position = solidlens.Vec{X: 300, Y: -1000, Z: 940}
		camera.Target = solidlens.Vec{X: -140, Y: 50, Z: 690}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models)
}

func spinningWheelView(shot string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		meshes, _, err := spinningWheelModels(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return spinningWheelScene(meshes, shot), nil
	}
}

func renderSpinningWheelReadme(ctx context.Context, out string) error {
	meshes, base, err := spinningWheelModels(ctx)
	if err != nil {
		return err
	}
	scene := spinningWheelScene(meshes, "hero")
	if err := renderPNG(ctx, filepath.Join(out, "hero.png"), scene, 1600, 900); err != nil {
		return err
	}
	if err := renderPNG(ctx, filepath.Join(out, "index.png"), scene, 480, 270); err != nil {
		return err
	}
	for _, shot := range []string{"front-back", "layout", "flyer", "brake", "whorl"} {
		if err := renderSpinningWheelSheet(ctx, meshes, shot, filepath.Join(out, shot+".png"), 2000, 1100); err != nil {
			return err
		}
	}
	return renderSpinningWheelAnimation(ctx, meshes, base, filepath.Join(out, "motion.gif"), 1200, 800, 96)
}
