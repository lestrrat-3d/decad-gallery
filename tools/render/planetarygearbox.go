package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

var (
	planetaryRingColor    = solidlens.RGB(0.38, 0.45, 0.52)
	planetaryCoverColor   = solidlens.RGB(0.5, 0.42, 0.55)
	planetarySunColor     = solidlens.RGB(0.72, 0.4, 0.14)
	planetaryPlanetColor  = solidlens.RGB(0.17, 0.42, 0.6)
	planetaryBearingColor = solidlens.RGB(0.62, 0.64, 0.68)
	planetaryShaftColor   = solidlens.RGB(0.45, 0.47, 0.5)
	planetaryCarrierColor = solidlens.RGB(0.55, 0.62, 0.3)
)

// planetaryEdges outlines every part.
var planetaryEdges = solidlens.Edges{
	Enabled: true,
	Color:   solidlens.RGB(0.025, 0.07, 0.11),
	Width:   1.5,
}

// buildPlanetaryGearbox builds the gearbox every image shows with the given stage count.
func buildPlanetaryGearbox(ctx context.Context, stages int) (planetarygearbox.Gearbox, planetarygearbox.Params, error) {
	params := planetarygearbox.DefaultParams()
	params.Stages = stages
	box, err := planetarygearbox.Build(ctx, decad.New(), params)
	if err != nil {
		return planetarygearbox.Gearbox{}, params, fmt.Errorf("build planetary gearbox: %w", err)
	}
	return box, params, nil
}

// planetaryBody is one body of the assembled gearbox as the images draw it.
type planetaryBody struct {
	name  string
	body  *decad.Body
	color solidlens.Color
	// housing marks the ring and the covers, which the cutaway views cut open.
	housing bool
}

// planetaryBodies lists every body of box with its colour. The ring and the two covers are the housing.
func planetaryBodies(box planetarygearbox.Gearbox) []planetaryBody {
	out := []planetaryBody{
		{planetarygearbox.RingName, box.Ring, planetaryRingColor, true},
		{planetarygearbox.FrontCoverName, box.FrontCover, planetaryCoverColor, true},
		{planetarygearbox.RearCoverName, box.RearCover, planetaryCoverColor, true},
		{planetarygearbox.FrontBearingName, box.FrontBearing, planetaryBearingColor, false},
		{planetarygearbox.RearBearingName, box.RearBearing, planetaryBearingColor, false},
		{planetarygearbox.InputShaftName, box.InputShaft, planetaryShaftColor, false},
	}
	for k, stage := range box.Stages {
		out = append(out, planetaryBody{planetarygearbox.SunName(k), stage.Sun, planetarySunColor, false})
		for i, planet := range stage.Planets {
			out = append(out,
				planetaryBody{planetarygearbox.PlanetName(k, i), planet, planetaryPlanetColor, false},
				planetaryBody{planetarygearbox.BearingName(k, i), stage.Bearings[i], planetaryBearingColor, false})
		}
		out = append(out, planetaryBody{planetarygearbox.CarrierName(k), stage.Carrier, planetaryCarrierColor, false})
	}
	return out
}

// planetaryMesh tessellates body for drawing.
func planetaryMesh(ctx context.Context, body planetaryBody) (solidlens.TriangleSource, error) {
	mesh, err := body.body.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		return nil, fmt.Errorf("tessellate %s: %w", body.name, err)
	}
	return mesh, nil
}

// planetaryCut cuts a model open along the plane through the axis that holds the X axis, removing the half
// toward negative Y, which is the side the cutaway cameras look from. The cut face is the body's colour,
// lightened and unlit, so it reads as a flat cut.
func planetaryCut(model solidlens.Model, color solidlens.Color) solidlens.Model {
	model.Clip = &solidlens.ClipPlane{
		Normal: solidlens.Vec{Y: -1},
		Cap: &solidlens.Material{
			Color: solidlens.RGB(color.R+(1-color.R)*0.35, color.G+(1-color.G)*0.35, color.B+(1-color.B)*0.35),
			Unlit: true,
		},
	}
	return model
}

// planetaryModel returns the model of one body. cut cuts the housing open.
func planetaryModel(mesh solidlens.TriangleSource, body planetaryBody, cut bool) solidlens.Model {
	model := solidlens.Model{Mesh: mesh, Material: solidlens.Matte(body.color), Edges: planetaryEdges}
	if cut && body.housing {
		model = planetaryCut(model, body.color)
	}
	return model
}

// planetaryHeroCamera looks at the assembled gearbox from the cut side, in front and above; fitCamera
// frames it.
var planetaryHeroCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 60, Y: -300, Z: 130},
	Target:   solidlens.Vec{Z: 8},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryHeroScene draws the assembled two-stage gearbox with the ring and both covers cut open, so the
// planets, the carriers, the shafts and the bearings show inside.
func planetaryHeroScene(ctx context.Context, box planetarygearbox.Gearbox) (solidlens.Scene, error) {
	var models []solidlens.Model
	for _, body := range planetaryBodies(box) {
		mesh, err := planetaryMesh(ctx, body)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, planetaryModel(mesh, body, true))
	}
	return bevelScene(fitCamera(planetaryHeroCamera, models, 16.0/9, bevelViewMargin), models), nil
}

func planetaryGearboxHero(ctx context.Context) (solidlens.Scene, error) {
	box, _, err := buildPlanetaryGearbox(ctx, 2)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return planetaryHeroScene(ctx, box)
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

// planetaryParts lists the unique parts of a gearbox: the ring, one cover (the front cover is the rear one
// turned over), the sun, one planet, one 608ZZ bearing (the planets and the covers use the same part), the
// input shaft, a carrier whose stub shaft drives the next stage, and the last carrier with its output shaft.
var planetaryParts = []planetaryPart{
	{"ring", func(b planetarygearbox.Gearbox) *decad.Body { return b.Ring }, planetaryRingColor, false},
	{"cover", func(b planetarygearbox.Gearbox) *decad.Body { return b.RearCover }, planetaryCoverColor, false},
	{"sun", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Sun }, planetarySunColor, false},
	{"planet", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Planets[0] }, planetaryPlanetColor, false},
	{"bearing", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Bearings[0] }, planetaryBearingColor, false},
	{"input-shaft", func(b planetarygearbox.Gearbox) *decad.Body { return b.InputShaft }, planetaryShaftColor, true},
	{"linking-carrier", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Carrier }, planetaryCarrierColor, true},
	{"output-carrier", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[1].Carrier }, planetaryCarrierColor, true},
}

func init() {
	for _, part := range planetaryParts {
		presets["planetarygearbox/part-"+part.shot] = preset{width: 480, height: 270, scene: planetaryPartPreset(part)}
	}
}

// planetaryPartCamera looks at one part from the front and above; fitCamera frames it.
var planetaryPartCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: 110},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryPartSideCamera looks at a long part from almost level with it, so a carrier shows its pins below the
// arms and its stub shaft above them.
var planetaryPartSideCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: 22},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryPartScene shows one part of box alone, in place, framed in the middle of the image.
func planetaryPartScene(ctx context.Context, box planetarygearbox.Gearbox, part planetaryPart) (solidlens.Scene, error) {
	body := planetaryBody{name: part.shot, body: part.body(box), color: part.color}
	mesh, err := planetaryMesh(ctx, body)
	if err != nil {
		return solidlens.Scene{}, err
	}
	models := []solidlens.Model{planetaryModel(mesh, body, false)}
	camera := planetaryPartCamera
	if part.side {
		camera = planetaryPartSideCamera
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, bevelViewMargin), models), nil
}

// planetaryPartPreset builds a part preset that builds the two-stage gearbox for that one part.
func planetaryPartPreset(part planetaryPart) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		box, _, err := buildPlanetaryGearbox(ctx, 2)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return planetaryPartScene(ctx, box, part)
	}
}

// renderPlanetaryGearboxReadme writes the index image, the hero, one image per unique part and the motion
// animation. The part images go in the parts directory beside them.
func renderPlanetaryGearboxReadme(ctx context.Context, outputDir string) error {
	box, _, err := buildPlanetaryGearbox(ctx, 2)
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
	return renderPlanetaryMotion(ctx, filepath.Join(outputDir, "motion.gif"), 640, 360, planetaryMotionFrames)
}
