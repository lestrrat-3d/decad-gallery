package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// m2BoltPitch is the spacing between bolts in the length comparison.
const m2BoltPitch = 6.0

var m2BoltCameras = map[string]solidlens.Camera{
	"hero":      {Position: solidlens.Vec{X: 9, Y: -9, Z: 7}, Target: solidlens.Vec{X: -2}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"head":      {Position: solidlens.Vec{X: 30, Y: -8, Z: 6}, Target: solidlens.Vec{}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"side":      {Position: solidlens.Vec{X: -4, Y: -40, Z: 2}, Target: solidlens.Vec{X: -4}, Up: solidlens.Vec{Z: 1}, FOV: 20},
	"underside": {Position: solidlens.Vec{X: -22, Y: -14, Z: -10}, Target: solidlens.Vec{X: -2}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"lengths":   {Position: solidlens.Vec{X: -6, Y: 8, Z: 45}, Target: solidlens.Vec{X: -6, Y: 12}, Up: solidlens.Vec{Y: 1}, FOV: 30},
}

func m2BoltHero(ctx context.Context) (solidlens.Scene, error) { return m2BoltView(ctx, "hero") }

func m2BoltViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return m2BoltView(ctx, name) }
}

// m2BoltView builds one view. Bolts lie on their sides along X with the head toward +X.
// Every view but "lengths" shows the default length.
func m2BoltView(ctx context.Context, name string) (solidlens.Scene, error) {
	lengths := []float64{m2bolt.DefaultParams().Length}
	if name == "lengths" {
		lengths = m2bolt.StandardLengths()
	}
	doc := decad.New()
	var models []solidlens.Model
	for i, length := range lengths {
		// The frame maps the bolt's +Z, from underside to head, to world +X.
		frame, err := r3.NewFrame(r3.Vec{Y: float64(i) * m2BoltPitch}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
		}
		m, err := m2bolt.Build(ctx, doc, m2bolt.Params{Length: length, Placement: &frame})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build M2x%g bolt: %w", length, err)
		}
		mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh M2x%g bolt: %w", length, err)
		}
		models = append(models, solidlens.Model{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}})
	}
	camera, ok := m2BoltCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown M2 bolt view %q", name)
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// renderM2BoltReadme writes the index image and every README view.
func renderM2BoltReadme(ctx context.Context, outputDir string) error {
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"head", "head.png", 1600, 900}, {"side", "side.png", 1600, 900},
		{"underside", "underside.png", 1600, 900}, {"lengths", "lengths.png", 1600, 900},
	} {
		scene, err := m2BoltView(ctx, shot.view)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	return nil
}
