package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pinring"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The ring stands on its lower face with its axis along +Z and spans z from 0 to 5.
var pinRingCameras = map[string]solidlens.Camera{
	"hero": {Position: solidlens.Vec{X: 22, Y: -28, Z: 20}, Target: solidlens.Vec{Z: 2.5}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"top":  {Position: solidlens.Vec{X: 0, Y: -.01, Z: 60}, Target: solidlens.Vec{Z: 2.5}, Up: solidlens.Vec{Y: 1}, FOV: 25},
	"side": {Position: solidlens.Vec{X: 0, Y: -60, Z: 2.5}, Target: solidlens.Vec{Z: 2.5}, Up: solidlens.Vec{Z: 1}, FOV: 20},
	"low":  {Position: solidlens.Vec{X: 22, Y: -28, Z: -15}, Target: solidlens.Vec{Z: 2.5}, Up: solidlens.Vec{Z: 1}, FOV: 30},
}

func pinRingHero(ctx context.Context) (solidlens.Scene, error) { return pinRingView(ctx, "hero") }

func pinRingViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return pinRingView(ctx, name) }
}

// pinRingView builds one view of the default pin ring.
func pinRingView(ctx context.Context, name string) (solidlens.Scene, error) {
	camera, ok := pinRingCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown pin ring view %q", name)
	}
	ring, err := pinring.Build(ctx, decad.New(), pinring.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build pin ring: %w", err)
	}
	mesh, err := ring.Body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("mesh pin ring %s view: %w", name, err)
	}
	models := []solidlens.Model{{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// renderPinRingReadme writes the index image and every README view.
func renderPinRingReadme(ctx context.Context, outputDir string) error {
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"top", "top.png", 1600, 900}, {"side", "side.png", 1600, 900}, {"low", "low.png", 1600, 900},
	} {
		scene, err := pinRingView(ctx, shot.view)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	return nil
}
