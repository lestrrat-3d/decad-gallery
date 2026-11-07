package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// cycloidalLayout is the exploded layout of the two-disc drive. Each part moves along a staircase 34 mm along
// X and 16 mm along Z per step.
var cycloidalLayout = map[string]int{
	cycloidal.HousingName:     -2,
	cycloidal.DiscName(0):     -1,
	cycloidal.DiscName(1):     0,
	cycloidal.CamName:         1,
	cycloidal.OutputPlateName: 2,
}

var cycloidalStep = r3.NewVec(34, 0, 16)

var cycloidalColors = map[string]solidlens.Color{
	cycloidal.HousingName:     solidlens.RGB(0.38, 0.45, 0.52),
	cycloidal.DiscName(0):     solidlens.RGB(0.17, 0.42, 0.6),
	cycloidal.DiscName(1):     solidlens.RGB(0.25, 0.52, 0.7),
	cycloidal.CamName:         solidlens.RGB(0.72, 0.4, 0.14),
	cycloidal.OutputPlateName: solidlens.RGB(0.55, 0.62, 0.3),
}

var cycloidalHeroCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 130, Y: -390, Z: 200},
	Target:   solidlens.Vec{Z: 8},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

var cycloidalExplodedCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 0, Y: -420, Z: 150},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

type cycloidalView struct {
	shot   string
	camera solidlens.Camera
}

var cycloidalViews = []cycloidalView{
	{"output", solidlens.Camera{Position: solidlens.Vec{X: 150, Y: -250, Z: 240}, Target: solidlens.Vec{Z: 12},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
	{"input", solidlens.Camera{Position: solidlens.Vec{X: -150, Y: -250, Z: -190}, Target: solidlens.Vec{Z: 8},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
	{"side", solidlens.Camera{Position: solidlens.Vec{X: 40, Y: -340, Z: 40}, Target: solidlens.Vec{Z: 12},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
}

type cycloidalPart struct {
	shot string
	name string
}

// cycloidalParts lists every separate body in the two-disc, six-pin drive.
var cycloidalParts = []cycloidalPart{
	{"disc-1", cycloidal.DiscName(0)},
	{"disc-2", cycloidal.DiscName(1)},
	{"housing", cycloidal.HousingName},
	{"cam", cycloidal.CamName},
	{"output-plate", cycloidal.OutputPlateName},
	{"output-pin-1", cycloidal.OutputPinName(0)},
	{"output-pin-2", cycloidal.OutputPinName(1)},
	{"output-pin-3", cycloidal.OutputPinName(2)},
	{"output-pin-4", cycloidal.OutputPinName(3)},
	{"output-pin-5", cycloidal.OutputPinName(4)},
	{"output-pin-6", cycloidal.OutputPinName(5)},
}

func init() {
	for _, view := range cycloidalViews {
		presets["cycloidal/assembled-"+view.shot] = preset{width: 1200, height: 675,
			scene: cycloidalAssembledPreset(view)}
	}
	for _, part := range cycloidalParts {
		presets["cycloidal/part-"+part.shot] = preset{width: 480, height: 270,
			scene: cycloidalPartPreset(part)}
	}
}

func buildCycloidalDrive(ctx context.Context) (cycloidal.Drive, error) {
	params := cycloidal.DefaultParams()
	params.DiscCount = 2
	drive, err := cycloidal.Build(ctx, decad.New(decad.WithChainTolerance(units.Millimeters(0.1))), params)
	if err != nil {
		return cycloidal.Drive{}, fmt.Errorf("build cycloidal drive: %w", err)
	}
	return drive, nil
}

func cycloidalBodyModel(ctx context.Context, named cycloidal.NamedBody, offset r3.Vec) (solidlens.Model, error) {
	mesh, err := named.Body.Tessellate(ctx, units.Millimeters(0.1), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		return solidlens.Model{}, fmt.Errorf("tessellate %s: %w", named.Name, err)
	}
	color, ok := cycloidalColors[named.Name]
	if !ok {
		color = solidlens.RGB(0.65, 0.47, 0.2)
	}
	return solidlens.Model{
		Mesh:     placedMesh{TriangleSource: mesh, scale: 1, offset: offset},
		Material: solidlens.Matte(color),
		Edges: solidlens.Edges{
			Enabled: true,
			Color:   solidlens.RGB(0.025, 0.07, 0.11),
			Width:   1.5,
		},
	}, nil
}

func cycloidalModels(ctx context.Context, drive cycloidal.Drive, exploded bool) ([]solidlens.Model, error) {
	var models []solidlens.Model
	for _, named := range drive.Named() {
		var offset r3.Vec
		if exploded {
			step, ok := cycloidalLayout[named.Name]
			if !ok {
				step = cycloidalLayout[cycloidal.OutputPlateName]
			}
			offset = cycloidalStep.Scale(float64(step))
		}
		model, err := cycloidalBodyModel(ctx, named, offset)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func cycloidalScene(ctx context.Context, drive cycloidal.Drive, camera solidlens.Camera, exploded bool) (solidlens.Scene, error) {
	models, err := cycloidalModels(ctx, drive, exploded)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, bevelViewMargin), models), nil
}

func cycloidalHero(ctx context.Context) (solidlens.Scene, error) {
	drive, err := buildCycloidalDrive(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return cycloidalScene(ctx, drive, cycloidalHeroCamera, false)
}

func cycloidalExploded(ctx context.Context) (solidlens.Scene, error) {
	drive, err := buildCycloidalDrive(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return cycloidalScene(ctx, drive, cycloidalExplodedCamera, true)
}

func cycloidalAssembledScene(ctx context.Context, drive cycloidal.Drive, view cycloidalView) (solidlens.Scene, error) {
	return cycloidalScene(ctx, drive, view.camera, false)
}

func cycloidalAssembledPreset(view cycloidalView) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		drive, err := buildCycloidalDrive(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return cycloidalAssembledScene(ctx, drive, view)
	}
}

func cycloidalPartScene(ctx context.Context, drive cycloidal.Drive, part cycloidalPart) (solidlens.Scene, error) {
	for _, named := range drive.Named() {
		if named.Name != part.name {
			continue
		}
		model, err := cycloidalBodyModel(ctx, named, r3.Vec{})
		if err != nil {
			return solidlens.Scene{}, err
		}
		models := []solidlens.Model{model}
		camera := solidlens.Camera{
			Position: solidlens.Vec{X: 120, Y: -200, Z: 130},
			Target:   solidlens.Vec{},
			Up:       solidlens.Vec{Z: 1},
			FOV:      35,
		}
		return bevelScene(fitCamera(camera, models, 16.0/9, bevelViewMargin), models), nil
	}
	return solidlens.Scene{}, fmt.Errorf("cycloidal part %q not found", part.name)
}

func cycloidalPartPreset(part cycloidalPart) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		drive, err := buildCycloidalDrive(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return cycloidalPartScene(ctx, drive, part)
	}
}

// renderCycloidalReadme writes the index, hero, assembled views, exploded view and every body image.
func renderCycloidalReadme(ctx context.Context, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("-out is required")
	}
	drive, err := buildCycloidalDrive(ctx)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name          string
		width, height int
		camera        solidlens.Camera
		exploded      bool
	}{
		{"hero.png", 1600, 900, cycloidalHeroCamera, false},
		{"index.png", 480, 270, cycloidalHeroCamera, false},
		{"exploded.png", 1600, 900, cycloidalExplodedCamera, true},
	} {
		scene, err := cycloidalScene(ctx, drive, image.camera, image.exploded)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), scene, image.width, image.height); err != nil {
			return fmt.Errorf("render cycloidal %s: %w", image.name, err)
		}
	}
	for _, view := range cycloidalViews {
		scene, err := cycloidalAssembledScene(ctx, drive, view)
		if err != nil {
			return err
		}
		name := filepath.Join(outputDir, "assembled-"+view.shot+".png")
		if err := renderPNG(ctx, name, scene, 1200, 675); err != nil {
			return fmt.Errorf("render cycloidal assembled %s: %w", view.shot, err)
		}
	}
	for _, part := range cycloidalParts {
		scene, err := cycloidalPartScene(ctx, drive, part)
		if err != nil {
			return err
		}
		name := filepath.Join(outputDir, "parts", part.shot+".png")
		if err := renderPNG(ctx, name, scene, 480, 270); err != nil {
			return fmt.Errorf("render cycloidal part %s: %w", part.name, err)
		}
	}
	return nil
}
