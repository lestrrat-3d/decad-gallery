package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/compressionspring"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

var compressionSpringEyes = map[string]solidlens.Vec{
	"hero":    {X: 24, Y: -38, Z: 27},
	"side":    {X: -55, Y: 24, Z: 3},
	"pressed": {X: 26, Y: -42, Z: 27},
}

func compressionSpringHero(ctx context.Context) (solidlens.Scene, error) {
	return compressionSpringView(ctx, "hero")
}

func compressionSpringViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return compressionSpringView(ctx, name) }
}

func compressionSpringView(ctx context.Context, name string) (solidlens.Scene, error) {
	eye, ok := compressionSpringEyes[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown compression spring view %q", name)
	}
	p := compressionspring.DefaultParams()
	if name == "pressed" {
		p.Spec = compressionspring.LeeLC026D16S()
		p.InstalledLength = 18.94
	}
	m, err := compressionspring.Build(ctx, decad.New(), p)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build compression spring: %w", err)
	}
	mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.04), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("mesh compression spring: %w", err)
	}
	models := []solidlens.Model{{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.72, .76, .79)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}}
	camera := solidlens.Camera{Position: eye, Target: solidlens.Vec{Y: p.InstalledLength / 2}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

func renderCompressionSpringReadme(ctx context.Context, outputDir string) error {
	return renderReadmeViews(ctx, outputDir, []readmeView{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"side", "side.png", 1600, 900}, {"pressed", "pressed.png", 1600, 900},
	}, compressionSpringView, renderPNG)
}
