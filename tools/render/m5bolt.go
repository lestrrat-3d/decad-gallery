package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m5bolt"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// m5BoltPitch is the spacing between bolts in the length comparison.
const m5BoltPitch = 12.0

var m5BoltCameras = map[string]solidlens.Camera{
	"hero":    {Position: solidlens.Vec{X: 22, Y: -24, Z: 18}, Target: solidlens.Vec{X: -9}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"head":    {Position: solidlens.Vec{X: 40, Y: -14, Z: 12}, Target: solidlens.Vec{}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"side":    {Position: solidlens.Vec{X: -10, Y: -80, Z: 4}, Target: solidlens.Vec{X: -10}, Up: solidlens.Vec{Z: 1}, FOV: 20},
	"lengths": {Position: solidlens.Vec{X: -35, Y: 40, Z: 150}, Target: solidlens.Vec{X: -35, Y: 40}, Up: solidlens.Vec{Y: 1}, FOV: 30},
}

func m5BoltHero(ctx context.Context) (solidlens.Scene, error) { return m5BoltView(ctx, "hero") }

func m5BoltViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return m5BoltView(ctx, name) }
}

// m5BoltView builds one view. Bolts lie on their sides along X with the head toward +X.
// Every view but "lengths" shows the default length.
func m5BoltView(ctx context.Context, name string) (solidlens.Scene, error) {
	lengths := []float64{m5bolt.DefaultParams().Length}
	if name == "lengths" {
		lengths = m5bolt.StandardLengths()
	}
	doc := decad.New()
	var models []solidlens.Model
	for i, length := range lengths {
		// The frame maps the bolt's +Z, from underside to head, to world +X.
		frame, err := r3.NewFrame(r3.Vec{Y: float64(i) * m5BoltPitch}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
		}
		m, err := m5bolt.Build(ctx, doc, m5bolt.Params{Length: length, Placement: &frame})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build M5x%g bolt: %w", length, err)
		}
		mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh M5x%g bolt: %w", length, err)
		}
		models = append(models, solidlens.Model{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}})
	}
	camera, ok := m5BoltCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown M5 bolt view %q", name)
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// renderM5BoltReadme writes the index image and every README view.
func renderM5BoltReadme(ctx context.Context, outputDir string) error {
	return renderReadmeViews(ctx, outputDir, []readmeView{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"head", "head.png", 1600, 900}, {"side", "side.png", 1600, 900},
		{"lengths", "lengths.png", 1600, 900},
	}, m5BoltView, renderPNG)
}
