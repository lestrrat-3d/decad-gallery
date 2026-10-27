package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// Tessellation tolerances in millimeters. The cage is a faceted body whose mesh bound is 0.189 mm,
// so decad refuses anything finer; the ribbons use the helical gear's 0.1 mm.
const (
	screwGearRibbonTolerance = 0.1
	screwGearCageTolerance   = 0.2
)

// screwGearCameras look at the default mechanism. The sleeve sits at the origin with its axis along Z, and the
// two ribbons cross it at 80 degrees. fitCamera centers and sizes each view on the sleeve and the stretch of
// ribbon next to it (see screwGearFocus).
var screwGearCameras = map[string]solidlens.Camera{
	"hero": {
		Position: solidlens.Vec{X: 60, Y: -300, Z: 110},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
	"top": {
		Position: solidlens.Vec{Z: 400},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Y: 1},
		FOV:      30,
	},
	"low": {
		Position: solidlens.Vec{X: -40, Y: -300, Z: -60},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
}

// screwGearColors are ribbon A, ribbon B and the cage.
var screwGearColors = [3]solidlens.Color{
	solidlens.RGB(0.14, 0.4, 0.58),
	solidlens.RGB(0.72, 0.4, 0.14),
	solidlens.RGB(0.5, 0.56, 0.62),
}

// screwGearModels builds the default mechanism and returns ribbon A, ribbon B and the cage as three meshes.
// Ribbons cannot be boolean operands, so each body is its own model.
func screwGearModels(ctx context.Context) ([]solidlens.Model, error) {
	mechanism, err := screwgear.Build(ctx, screwgear.NewDocument(), screwgear.DefaultParams())
	if err != nil {
		return nil, fmt.Errorf("build screw gear: %w", err)
	}
	var models []solidlens.Model
	bodies := []*decad.Body{mechanism.GearA, mechanism.GearB, mechanism.Cage}
	tolerances := []float64{screwGearRibbonTolerance, screwGearRibbonTolerance, screwGearCageTolerance}
	for i, body := range bodies {
		mesh, err := body.Tessellate(ctx, units.Millimeters(tolerances[i]), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate screw gear body %d: %w", i, err)
		}
		models = append(models, solidlens.Model{
			Mesh:     mesh,
			Material: solidlens.Matte(screwGearColors[i]),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
	}
	return models, nil
}

// screwGearFocus is the part of the mechanism a view is framed on: the vertices within this distance of the
// origin. The ribbons are far longer than the sleeve, so framing on every vertex would shrink the sleeve to a speck.
const screwGearFocus = 70.0

// screwGearScene frames the camera on the sleeve and the ribbon next to it.
func screwGearScene(camera solidlens.Camera, models []solidlens.Model, aspect float64) solidlens.Scene {
	var near []r3.Vec
	for _, model := range models {
		for _, v := range model.Mesh.Vertices() {
			if v.Dot(v) <= screwGearFocus*screwGearFocus {
				near = append(near, v)
			}
		}
	}
	focus := []solidlens.Model{{Mesh: pointCloud(near)}}
	return bevelScene(fitCamera(camera, focus, aspect, bevelViewMargin), models)
}

// pointCloud is a vertex list that stands in for a mesh when only extents matter.
type pointCloud []r3.Vec

func (p pointCloud) Vertices() []r3.Vec { return p }

func (p pointCloud) Triangles() [][3]int { return nil }

func screwGearView(shot string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		models, err := screwGearModels(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return screwGearScene(screwGearCameras[shot], models, 16.0/9), nil
	}
}

// renderScrewGearReadme builds the mechanism once and writes index.png, hero.png, top.png and low.png.
func renderScrewGearReadme(ctx context.Context, outputDir string) error {
	models, err := screwGearModels(ctx)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name, shot    string
		width, height int
	}{
		{"index.png", "hero", 480, 270},
		{"hero.png", "hero", 1600, 900},
		{"top.png", "top", 1600, 900},
		{"low.png", "low", 1600, 900},
	} {
		scene := screwGearScene(screwGearCameras[image.shot], models, 16.0/9)
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), scene, image.width, image.height); err != nil {
			return fmt.Errorf("render screwgear %s: %w", image.name, err)
		}
	}
	return nil
}
