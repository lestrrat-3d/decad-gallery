package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/features/dovetail"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The default dovetail demo: two blocks 21.5 mm wide, 16 mm long along Y and 10 mm tall, the first below z = 0 and the
// second above it, the tail on the first block rising 6 mm into the second. The slide direction is +Y.
var dovetailCameras = map[string]solidlens.Camera{
	"hero":     {Position: solidlens.Vec{X: 38, Y: -42, Z: 30}, Target: solidlens.Vec{Y: 8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"end":      {Position: solidlens.Vec{X: 0, Y: -90, Z: 0}, Target: solidlens.Vec{Y: 8}, Up: solidlens.Vec{Z: 1}, FOV: 20},
	"rear":     {Position: solidlens.Vec{X: -38, Y: 58, Z: 30}, Target: solidlens.Vec{Y: 8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"low":      {Position: solidlens.Vec{X: -38, Y: -42, Z: -30}, Target: solidlens.Vec{Y: 8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
	"exploded": {Position: solidlens.Vec{X: 52, Y: -34, Z: 30}, Target: solidlens.Vec{Y: 8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
}

// dovetailExplode is how far each block moves along the slide direction in the exploded view, in mm: the first block
// toward -Y and the second toward +Y, so the halves part by 30 mm, clear of each other's 16 mm length.
const dovetailExplode = 15.0

func dovetailHero(ctx context.Context) (solidlens.Scene, error) { return dovetailView(ctx, "hero") }

func dovetailViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return dovetailView(ctx, name) }
}

// dovetailView builds one view of the default dovetail demo. The tail block is orange and the socket block blue.
func dovetailView(ctx context.Context, name string) (solidlens.Scene, error) {
	// The part views use the hero camera and show one block.
	only := ""
	switch name {
	case "tail-block", "socket-block":
		only, name = name, "hero"
	}
	camera, ok := dovetailCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown dovetail view %q", name)
	}
	demo, err := dovetail.BuildDemo(ctx, decad.New(), dovetail.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build dovetail demo: %w", err)
	}
	type part struct {
		body   *decad.Body
		colour solidlens.Color
		pull   float64
	}
	pull := dovetailExplode
	if name != "exploded" {
		pull = 0
	}
	parts := []part{{demo.TailBlock, solidlens.RGB(.86, .52, .22), 0}, {demo.SocketBlock, solidlens.RGB(.32, .5, .72), 0}}
	// The first block moves toward -Y and the second toward +Y.
	if demo.TailBlock == demo.First {
		parts[0].pull, parts[1].pull = -pull, pull
	} else {
		parts[0].pull, parts[1].pull = pull, -pull
	}
	var models []solidlens.Model
	for i, p := range parts {
		if only == "tail-block" && i != 0 || only == "socket-block" && i != 1 {
			continue
		}
		body := p.body
		if p.pull != 0 {
			move, err := r3.Translation(r3.Vec{Y: p.pull})
			if err != nil {
				return solidlens.Scene{}, fmt.Errorf("dovetail %s view: move part %d: %w", name, i, err)
			}
			if body, err = decad.Transform(ctx, body, move); err != nil {
				return solidlens.Scene{}, fmt.Errorf("dovetail %s view: move part %d: %w", name, i, err)
			}
		}
		mesh, err := body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh dovetail %s view: %w", name, err)
		}
		models = append(models, solidlens.Model{Mesh: mesh, Material: solidlens.Matte(p.colour),
			Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}})
	}
	margin := .1
	if name == "exploded" {
		margin = .04
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, margin), models), nil
}

// renderDovetailReadme writes the index image and every README view.
func renderDovetailReadme(ctx context.Context, outputDir string) error {
	return renderReadmeViews(ctx, outputDir, []readmeView{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"end", "end.png", 1600, 900}, {"rear", "rear.png", 1600, 900}, {"low", "low.png", 1600, 900},
		{"exploded", "exploded.png", 1600, 900},
		{"tail-block", "parts/tail-block.png", 480, 270}, {"socket-block", "parts/socket-block.png", 480, 270},
	}, dovetailView, renderPNG)
}
