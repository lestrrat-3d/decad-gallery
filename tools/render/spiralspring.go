package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// spiralSpringGap is the clear distance along X between springs in the size comparison.
const spiralSpringGap = 10.0

var spiralSpringCameras = map[string]solidlens.Camera{
	"hero":  {Position: solidlens.Vec{X: 30, Y: -85, Z: 75}, Target: solidlens.Vec{X: -1, Y: 1, Z: 3}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"top":   {Position: solidlens.Vec{X: -1, Y: 1, Z: 110}, Target: solidlens.Vec{X: -1, Y: 1}, Up: solidlens.Vec{Y: 1}, FOV: 35},
	"low":   {Position: solidlens.Vec{X: 40, Y: -70, Z: -35}, Target: solidlens.Vec{X: -1, Y: 1, Z: 3}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"root":  {Position: solidlens.Vec{X: -6, Y: -22, Z: 26}, Target: solidlens.Vec{X: -9, Y: -3, Z: 4}, Up: solidlens.Vec{Z: 1}, FOV: 28},
	"sizes": {Position: solidlens.Vec{X: 40, Y: -120, Z: 110}, Target: solidlens.Vec{X: 40, Y: 0, Z: 3}, Up: solidlens.Vec{Z: 1}, FOV: 35},
}

// spiralSpringSizes are the springs in the size comparison, left to right: the default, a small and a large one.
func spiralSpringSizes() []spiralspring.Params {
	small := spiralspring.DefaultParams()
	small.OuterDiameter, small.HubDiameter, small.Thickness, small.MinGap, small.Height = 40, 12, 1.5, 1.5, 5
	large := spiralspring.DefaultParams()
	large.OuterDiameter, large.HubDiameter, large.Thickness, large.MinGap, large.Height = 90, 16, 2, 2.5, 10
	return []spiralspring.Params{spiralspring.DefaultParams(), small, large}
}

func spiralSpringHero(ctx context.Context) (solidlens.Scene, error) {
	return spiralSpringView(ctx, "hero")
}

func spiralSpringViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return spiralSpringView(ctx, name) }
}

// spiralSpringView builds one view. Springs lie flat with the axis along +Z. Every view but "sizes" shows the default
// spring.
func spiralSpringView(ctx context.Context, name string) (solidlens.Scene, error) {
	camera, ok := spiralSpringCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown spiral spring view %q", name)
	}
	params := []spiralspring.Params{spiralspring.DefaultParams()}
	if name == "sizes" {
		params = spiralSpringSizes()
	}
	doc := decad.New()
	var models []solidlens.Model
	right := 0.0
	for i, p := range params {
		dm, err := p.Derive()
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("derive spring %d: %w", i, err)
		}
		// Slide each spring right until its left edge is spiralSpringGap clear of the one before.
		left := dm.EyeCenterX - dm.EyeRadius
		shift := 0.0
		if i > 0 {
			shift = right + spiralSpringGap - left
		}
		frame, err := r3.NewFrame(r3.Vec{X: shift}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create spring frame: %w", err)
		}
		p.Placement = &frame
		m, err := spiralspring.Build(ctx, doc, p)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build spring %d: %w", i, err)
		}
		right = shift + dm.CoilRadius
		mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh spring %d: %w", i, err)
		}
		models = append(models, solidlens.Model{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.80, .52, .22)),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}})
	}
	if name != "root" {
		camera = fitCamera(camera, models, 16.0/9, .1)
	}
	return bevelScene(camera, models), nil
}

// renderSpiralSpringReadme writes the index image and every README view.
func renderSpiralSpringReadme(ctx context.Context, outputDir string) error {
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"top", "top.png", 1600, 900}, {"low", "low.png", 1600, 900},
		{"root", "root.png", 1600, 900}, {"sizes", "sizes.png", 1600, 900},
	} {
		scene, err := spiralSpringView(ctx, shot.view)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	return nil
}
