package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The hero image pulls the default two-stage gearbox apart along its axis: each stage rises by
// planetaryStageLift over the one below, each carrier lifts planetaryCarrierLift off its planets,
// and the ring floats planetaryRingLift above the stack's first stage.
const (
	planetaryStageLift   = 30
	planetaryCarrierLift = 17
	planetaryRingLift    = 78
)

var (
	planetaryRingColor    = solidlens.RGB(0.38, 0.45, 0.52)
	planetarySunColor     = solidlens.RGB(0.72, 0.4, 0.14)
	planetaryPlanetColor  = solidlens.RGB(0.17, 0.42, 0.6)
	planetaryCarrierColor = solidlens.RGB(0.55, 0.62, 0.3)
)

// buildPlanetaryGearbox builds the default gearbox with two stages, the layout of every image.
func buildPlanetaryGearbox(ctx context.Context) (planetarygearbox.Gearbox, error) {
	params := planetarygearbox.DefaultParams()
	params.Stages = 2
	box, err := planetarygearbox.Build(ctx, decad.New(), params)
	if err != nil {
		return planetarygearbox.Gearbox{}, fmt.Errorf("build planetary gearbox: %w", err)
	}
	return box, nil
}

// planetaryModels returns one model per body of box, moved to the exploded layout.
func planetaryModels(ctx context.Context, box planetarygearbox.Gearbox) ([]solidlens.Model, error) {
	var models []solidlens.Model
	add := func(name string, body *decad.Body, color solidlens.Color, lift float64) error {
		mesh, err := body.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return fmt.Errorf("tessellate %s: %w", name, err)
		}
		models = append(models, solidlens.Model{
			Mesh:     placedMesh{TriangleSource: mesh, scale: 1, offset: r3.NewVec(0, 0, lift)},
			Material: solidlens.Matte(color),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
		return nil
	}
	if err := add(planetarygearbox.RingName, box.Ring, planetaryRingColor, planetaryRingLift); err != nil {
		return nil, err
	}
	for k, stage := range box.Stages {
		lift := float64(k) * planetaryStageLift
		if err := add(planetarygearbox.SunName(k), stage.Sun, planetarySunColor, lift); err != nil {
			return nil, err
		}
		for i, planet := range stage.Planets {
			if err := add(planetarygearbox.PlanetName(k, i), planet, planetaryPlanetColor, lift); err != nil {
				return nil, err
			}
		}
		if err := add(planetarygearbox.CarrierName(k), stage.Carrier, planetaryCarrierColor,
			lift+planetaryCarrierLift); err != nil {
			return nil, err
		}
	}
	return models, nil
}

// planetaryHeroCamera looks at the stack from the front and above; fitCamera centers it.
var planetaryHeroCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 90, Y: -340, Z: 120},
	Target:   solidlens.Vec{Z: 50},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

func planetaryGearboxHero(ctx context.Context) (solidlens.Scene, error) {
	box, err := buildPlanetaryGearbox(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return planetaryHeroScene(ctx, box)
}

func planetaryHeroScene(ctx context.Context, box planetarygearbox.Gearbox) (solidlens.Scene, error) {
	models, err := planetaryModels(ctx, box)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(fitCamera(planetaryHeroCamera, models, 16.0/9, bevelViewMargin), models), nil
}

// planetaryPart is one unique part of the gearbox, as the README's parts table lists it.
type planetaryPart struct {
	// shot is the preset suffix and the image file name.
	shot string
	// body picks the part from a two-stage gearbox.
	body  func(planetarygearbox.Gearbox) *decad.Body
	color solidlens.Color
	// side looks at the part from almost level with it, so a carrier shows its pins and its stub shaft.
	side bool
}

// planetaryParts lists the unique parts of a gearbox: the ring, the sun, one planet, a carrier whose
// stub shaft drives the next stage, and the last carrier with its output shaft.
var planetaryParts = []planetaryPart{
	{"ring", func(b planetarygearbox.Gearbox) *decad.Body { return b.Ring }, planetaryRingColor, false},
	{"sun", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Sun }, planetarySunColor, false},
	{"planet", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Planets[0] }, planetaryPlanetColor, false},
	{"linking-carrier", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Carrier }, planetaryCarrierColor, true},
	{"output-carrier", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[1].Carrier }, planetaryCarrierColor, true},
}

// planetaryPartCamera looks at one part from the front and above; fitCamera frames it.
var planetaryPartCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: 110},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryPartSideCamera looks at a carrier from almost level with its plate, so the pins show
// below it and the stub shaft above it.
var planetaryPartSideCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: 22},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryPartScene shows one part of box alone, in place, framed in the middle of the image.
func planetaryPartScene(ctx context.Context, box planetarygearbox.Gearbox, part planetaryPart) (solidlens.Scene, error) {
	mesh, err := part.body(box).Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("tessellate %s: %w", part.shot, err)
	}
	models := []solidlens.Model{{
		Mesh:     placedMesh{TriangleSource: mesh, scale: 1},
		Material: solidlens.Matte(part.color),
		Edges: solidlens.Edges{
			Enabled: true,
			Color:   solidlens.RGB(0.025, 0.07, 0.11),
			Width:   1.5,
		},
	}}
	camera := planetaryPartCamera
	if part.side {
		camera = planetaryPartSideCamera
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, bevelViewMargin), models), nil
}

// planetaryPartPreset builds a part preset that builds the two-stage gearbox for that one part.
func planetaryPartPreset(part planetaryPart) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		box, err := buildPlanetaryGearbox(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return planetaryPartScene(ctx, box, part)
	}
}

// renderPlanetaryGearboxReadme writes the index image, the hero and one image per unique part from one
// build. The part images go in the parts directory beside them.
func renderPlanetaryGearboxReadme(ctx context.Context, outputDir string) error {
	box, err := buildPlanetaryGearbox(ctx)
	if err != nil {
		return err
	}
	scene, err := planetaryHeroScene(ctx, box)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name          string
		width, height int
	}{{"index.png", 480, 270}, {"hero.png", 1600, 900}} {
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), scene, image.width, image.height); err != nil {
			return fmt.Errorf("render planetary gearbox %s: %w", image.name, err)
		}
	}
	for _, part := range planetaryParts {
		scene, err := planetaryPartScene(ctx, box, part)
		if err != nil {
			return err
		}
		name := filepath.Join(outputDir, "parts", part.shot+".png")
		if err := renderPNG(ctx, name, scene, 480, 270); err != nil {
			return fmt.Errorf("render planetary gearbox part %s: %w", part.shot, err)
		}
	}
	return nil
}
