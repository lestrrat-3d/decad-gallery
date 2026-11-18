package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m5nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The nut stands on its lower face with its axis along +Z, except in "withbolt", where an M5x25 bolt lies along X.
var m5NutCameras = map[string]solidlens.Camera{
	"hero":     {Position: solidlens.Vec{X: 11, Y: -14, Z: 11}, Target: solidlens.Vec{Z: 2.3}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"top":      {Position: solidlens.Vec{X: 1, Y: -2, Z: 30}, Target: solidlens.Vec{Z: 2.3}, Up: solidlens.Vec{Y: 1}, FOV: 25},
	"side":     {Position: solidlens.Vec{X: 0, Y: -45, Z: 2.3}, Target: solidlens.Vec{Z: 2.3}, Up: solidlens.Vec{Z: 1}, FOV: 20},
	"withbolt": {Position: solidlens.Vec{X: 22, Y: -28, Z: 20}, Target: solidlens.Vec{X: -12}, Up: solidlens.Vec{Z: 1}, FOV: 35},
}

func m5NutHero(ctx context.Context) (solidlens.Scene, error) { return m5NutView(ctx, "hero") }

func m5NutViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return m5NutView(ctx, name) }
}

func m5NutModel(mesh *decad.Mesh) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// m5NutView builds one view of the nut, or of the nut on a bolt for "withbolt".
func m5NutView(ctx context.Context, name string) (solidlens.Scene, error) {
	camera, ok := m5NutCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown M5 nut view %q", name)
	}
	doc := decad.New()
	params := m5nut.DefaultParams()
	var bodies []*decad.Body
	if name == "withbolt" {
		// Both frames map their +Z to world +X. The bolt's head underside is at x = 0 and its tip at x = -25; the nut sits 1 mm from the tip.
		boltFrame, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
		}
		bolt, err := m5bolt.Build(ctx, doc, m5bolt.Params{Length: 25, Placement: &boltFrame})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build M5 bolt: %w", err)
		}
		bodies = append(bodies, bolt.Body)
		nutFrame, err := r3.NewFrame(r3.Vec{X: -24}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create nut frame: %w", err)
		}
		params.Placement = &nutFrame
	}
	nut, err := m5nut.Build(ctx, doc, params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build M5 nut: %w", err)
	}
	bodies = append(bodies, nut.Body)
	var models []solidlens.Model
	for _, body := range bodies {
		mesh, err := body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh %s view: %w", name, err)
		}
		models = append(models, m5NutModel(mesh))
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// renderM5NutReadme writes the index image and every README view.
func renderM5NutReadme(ctx context.Context, outputDir string) error {
	return renderReadmeViews(ctx, outputDir, []readmeView{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"top", "top.png", 1600, 900}, {"side", "side.png", 1600, 900},
		{"withbolt", "withbolt.png", 1600, 900},
	}, m5NutView, renderPNG)
}
