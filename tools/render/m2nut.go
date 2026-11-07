package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The nut stands on its lower face with its axis along +Z, except in "withbolt", where an M2x10 bolt lies along X.
var m2NutCameras = map[string]solidlens.Camera{
	"hero":     {Position: solidlens.Vec{X: 7, Y: -9, Z: 7}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"top":      {Position: solidlens.Vec{X: 1, Y: -1.5, Z: 20}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Y: 1}, FOV: 25},
	"side":     {Position: solidlens.Vec{X: 0, Y: -30, Z: .8}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Z: 1}, FOV: 20},
	"low":      {Position: solidlens.Vec{X: 7, Y: -9, Z: -7}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"withbolt": {Position: solidlens.Vec{X: 9, Y: -12, Z: 9}, Target: solidlens.Vec{X: -4}, Up: solidlens.Vec{Z: 1}, FOV: 35},
}

func m2NutHero(ctx context.Context) (solidlens.Scene, error) { return m2NutView(ctx, "hero") }

func m2NutViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return m2NutView(ctx, name) }
}

func m2NutModel(mesh *decad.Mesh) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// m2NutView builds one view of the nut, or of the nut on a bolt for "withbolt".
func m2NutView(ctx context.Context, name string) (solidlens.Scene, error) {
	camera, ok := m2NutCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown M2 nut view %q", name)
	}
	doc := decad.New()
	params := m2nut.DefaultParams()
	var bodies []*decad.Body
	if name == "withbolt" {
		// Both frames map their +Z to world +X. The bolt's head underside is at x = 0 and its tip at x = -10; the nut sits 1 mm from the tip.
		boltFrame, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
		}
		bolt, err := m2bolt.Build(ctx, doc, m2bolt.Params{Length: 10, Placement: &boltFrame})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build M2 bolt: %w", err)
		}
		bodies = append(bodies, bolt.Body)
		nutFrame, err := r3.NewFrame(r3.Vec{X: -9}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create nut frame: %w", err)
		}
		params.Placement = &nutFrame
	}
	nut, err := m2nut.Build(ctx, doc, params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build M2 nut: %w", err)
	}
	bodies = append(bodies, nut.Body)
	var models []solidlens.Model
	for _, body := range bodies {
		mesh, err := body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh %s view: %w", name, err)
		}
		models = append(models, m2NutModel(mesh))
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// renderM2NutReadme writes the index image and every README view.
func renderM2NutReadme(ctx context.Context, outputDir string) error {
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"top", "top.png", 1600, 900}, {"side", "side.png", 1600, 900},
		{"low", "low.png", 1600, 900}, {"withbolt", "withbolt.png", 1600, 900},
	} {
		scene, err := m2NutView(ctx, shot.view)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	return nil
}
