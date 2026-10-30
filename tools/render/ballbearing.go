package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

func bearing608Hero(ctx context.Context) (solidlens.Scene, error) {
	return bearingEnvelopeScene(ctx, ballbearing.DefaultParams())
}

func bearing605Hero(ctx context.Context) (solidlens.Scene, error) {
	return bearingEnvelopeScene(ctx, ballbearing.Params605ZZ())
}

func bearingEnvelopeScene(ctx context.Context, params ballbearing.Params) (solidlens.Scene, error) {
	m, err := ballbearing.Build(ctx, decad.New(), params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build bearing envelope: %w", err)
	}
	mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.03), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("mesh bearing envelope: %w", err)
	}
	models := []solidlens.Model{{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.45, .55, .62)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 28, Y: -36, Z: 30},
		Target: solidlens.Vec{Z: params.Width / 2}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

func renderBearingReadme(ctx context.Context, outputDir string) error {
	scene, err := bearing608Hero(ctx)
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		name          string
		width, height int
	}{
		{"hero.png", 1600, 900}, {"index.png", 480, 270},
	} {
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.name), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	other, err := bearing605Hero(ctx)
	if err != nil {
		return err
	}
	if err := renderPNG(ctx, filepath.Join(outputDir, "605zz.png"), other, 1600, 900); err != nil {
		return err
	}
	return nil
}
