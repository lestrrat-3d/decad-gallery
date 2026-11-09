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
	box, err := planetarygearbox.Build(ctx, planetarygearbox.NewDocument(), params)
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
		out = append(out,
			planetaryBody{planetarygearbox.CarrierName(k), stage.Carrier, planetaryCarrierColor, false},
			planetaryBody{planetarygearbox.ShaftName(k, len(box.Stages)), stage.Shaft, planetaryShaftColor, false})
	}
	return out
}

// The exploded view pulls the gearbox apart along its axis. Each stage rises planetaryExplodedStep over the one
// below, and inside a stage the planets' bearings, the carrier and the shaft each rise a little more, so every
// part sits clear of the ones it joins. The ring floats above the last stage, the rear cover and its bearing above
// that, and the output shaft above them all; the front cover, its bearing and the input shaft hang below the first
// stage.
const (
	planetaryExplodedStep    = 48
	planetaryExplodedBearing = 14
	planetaryExplodedCarrier = 24
	planetaryExplodedShaft   = 36
	planetaryExplodedRing    = 64
)

// planetaryExplodedLifts returns how far the exploded view raises each body of box.
func planetaryExplodedLifts(box planetarygearbox.Gearbox) map[*decad.Body]float64 {
	lifts := map[*decad.Body]float64{
		box.InputShaft:   -34,
		box.FrontBearing: -24,
		box.FrontCover:   -12,
	}
	top := 0.0
	for k, stage := range box.Stages {
		base := float64(k) * planetaryExplodedStep
		top = base
		lifts[stage.Sun] = base
		for i, planet := range stage.Planets {
			lifts[planet] = base
			lifts[stage.Bearings[i]] = base + planetaryExplodedBearing
		}
		lifts[stage.Carrier] = base + planetaryExplodedCarrier
		lifts[stage.Shaft] = base + planetaryExplodedShaft
	}
	// The output shaft passes through the ring and both rear parts, so it rises above them all to stay visible.
	lifts[box.Stages[len(box.Stages)-1].Shaft] = top + planetaryExplodedRing + 56
	lifts[box.Ring] = top + planetaryExplodedRing
	lifts[box.RearCover] = top + planetaryExplodedRing + 28
	lifts[box.RearBearing] = top + planetaryExplodedRing + 42
	return lifts
}

// planetaryExplodedCamera looks at the exploded gearbox from the front and above; fitCamera frames it.
var planetaryExplodedCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 230, Y: -420, Z: 190},
	Target:   solidlens.Vec{Z: 66},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryExplodedScene draws the whole two-stage gearbox pulled apart along its axis, with nothing cut.
func planetaryExplodedScene(ctx context.Context, box planetarygearbox.Gearbox, width, height int) (solidlens.Scene, error) {
	lifts := planetaryExplodedLifts(box)
	var models []solidlens.Model
	for _, body := range planetaryBodies(box) {
		mesh, err := planetaryMesh(ctx, body)
		if err != nil {
			return solidlens.Scene{}, err
		}
		model := planetaryModel(mesh, body, false)
		model.Mesh = placedMesh{TriangleSource: mesh, scale: 1, offset: r3.NewVec(0, 0, lifts[body.body])}
		models = append(models, model)
	}
	return bevelScene(fitCamera(planetaryExplodedCamera, models, float64(width)/float64(height), bevelViewMargin), models), nil
}

func init() {
	presets["planetarygearbox/exploded"] = preset{width: planetaryExplodedWidth, height: planetaryExplodedHeight,
		scene: func(ctx context.Context) (solidlens.Scene, error) {
			box, _, err := buildPlanetaryGearbox(ctx, 2)
			if err != nil {
				return solidlens.Scene{}, err
			}
			return planetaryExplodedScene(ctx, box, planetaryExplodedWidth, planetaryExplodedHeight)
		}}
}

// The exploded view is taller than wide, so it has its own image size.
const (
	planetaryExplodedWidth  = 900
	planetaryExplodedHeight = 1200
)

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

// planetaryView is a named camera for the assembled gearbox. The gearbox axis is +Z, the input shaft sticks out of
// the bottom and the output shaft out of the top.
type planetaryView struct {
	// shot is the preset suffix and the image file name.
	shot   string
	camera solidlens.Camera
}

// planetaryViews are the cameras of the assembled, uncut gearbox: obliquely at the output end from above, at the
// input end from below, and from the side, where both shaft ends show.
var planetaryViews = []planetaryView{
	{"output", solidlens.Camera{Position: solidlens.Vec{X: 150, Y: -250, Z: 240}, Target: solidlens.Vec{Z: 18},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
	{"input", solidlens.Camera{Position: solidlens.Vec{X: -150, Y: -250, Z: -190}, Target: solidlens.Vec{Z: 8},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
	{"side", solidlens.Camera{Position: solidlens.Vec{X: 40, Y: -340, Z: 40}, Target: solidlens.Vec{Z: 12},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
}

// planetaryAssembledScene draws the whole two-stage gearbox with nothing cut, from view.
func planetaryAssembledScene(ctx context.Context, box planetarygearbox.Gearbox, view planetaryView) (solidlens.Scene, error) {
	var models []solidlens.Model
	for _, body := range planetaryBodies(box) {
		mesh, err := planetaryMesh(ctx, body)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, planetaryModel(mesh, body, false))
	}
	return bevelScene(fitCamera(view.camera, models, 16.0/9, bevelViewMargin), models), nil
}

func init() {
	for _, view := range planetaryViews {
		presets["planetarygearbox/assembled-"+view.shot] = preset{width: 1200, height: 675,
			scene: func(ctx context.Context) (solidlens.Scene, error) {
				box, _, err := buildPlanetaryGearbox(ctx, 2)
				if err != nil {
					return solidlens.Scene{}, err
				}
				return planetaryAssembledScene(ctx, box, view)
			}}
	}
}

// planetaryPart is one unique part of the gearbox, as the README's parts table lists it.
type planetaryPart struct {
	// shot is the preset suffix and the image file name.
	shot string
	// body picks the part from a two-stage gearbox.
	body  func(planetarygearbox.Gearbox) *decad.Body
	color solidlens.Color
	// view picks the camera: the default looks from the front and above, side from almost level with the
	// part so a shaft shows its collar, and below from under the part so a carrier shows its pins.
	view partView
}

// partView is a camera choice for a part image.
type partView int

const (
	frontView partView = iota
	sideView
	belowView
)

// planetaryParts lists the unique parts of a gearbox: the ring, one cover (the front cover is the rear one
// turned over), the sun, one planet, one 605ZZ bearing (the planets and the covers use the same part), the
// carrier, and the three shafts: the input shaft, a link shaft and the output shaft.
var planetaryParts = []planetaryPart{
	{"ring", func(b planetarygearbox.Gearbox) *decad.Body { return b.Ring }, planetaryRingColor, frontView},
	{"cover", func(b planetarygearbox.Gearbox) *decad.Body { return b.RearCover }, planetaryCoverColor, frontView},
	{"sun", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Sun }, planetarySunColor, frontView},
	{"planet", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Planets[0] }, planetaryPlanetColor, frontView},
	{"bearing", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Bearings[0] }, planetaryBearingColor, frontView},
	{"carrier", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Carrier }, planetaryCarrierColor, belowView},
	{"input-shaft", func(b planetarygearbox.Gearbox) *decad.Body { return b.InputShaft }, planetaryShaftColor, sideView},
	{"link-shaft", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[0].Shaft }, planetaryShaftColor, sideView},
	{"output-shaft", func(b planetarygearbox.Gearbox) *decad.Body { return b.Stages[1].Shaft }, planetaryShaftColor, sideView},
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

// planetaryPartSideCamera looks at a long part from almost level with it, so a shaft shows its keyway and its collar.
var planetaryPartSideCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: 22},
	Target:   solidlens.Vec{},
	Up:       solidlens.Vec{Z: 1},
	FOV:      35,
}

// planetaryPartBelowCamera looks at a part from under its lower face, so a carrier shows its pins and their
// collars standing on the plate's underside.
var planetaryPartBelowCamera = solidlens.Camera{
	Position: solidlens.Vec{X: 70, Y: -150, Z: -90},
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
	switch part.view {
	case sideView:
		camera = planetaryPartSideCamera
	case belowView:
		camera = planetaryPartBelowCamera
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

// renderPlanetaryGearboxReadme writes the index image, the hero, the assembled views, one image per unique part and
// the motion animation. The part images go in the parts directory beside them.
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
	for _, view := range planetaryViews {
		scene, err := planetaryAssembledScene(ctx, box, view)
		if err != nil {
			return err
		}
		name := filepath.Join(outputDir, "assembled-"+view.shot+".png")
		if err := renderPNG(ctx, name, scene, 1200, 675); err != nil {
			return fmt.Errorf("render planetary gearbox assembled %s: %w", view.shot, err)
		}
	}
	section, fastenerExploded, err := planetaryFastenerScenes(ctx)
	if err != nil {
		return err
	}
	for name, scene := range map[string]solidlens.Scene{
		"fasteners-section.png": section, "fasteners-exploded.png": fastenerExploded,
	} {
		if err := renderPNG(ctx, filepath.Join(outputDir, name), scene, 1600, 900); err != nil {
			return fmt.Errorf("render planetary gearbox %s: %w", name, err)
		}
	}
	exploded, err := planetaryExplodedScene(ctx, box, planetaryExplodedWidth, planetaryExplodedHeight)
	if err != nil {
		return err
	}
	if err := renderPNG(ctx, filepath.Join(outputDir, "exploded.png"), exploded, planetaryExplodedWidth,
		planetaryExplodedHeight); err != nil {
		return fmt.Errorf("render planetary gearbox exploded view: %w", err)
	}
	return renderPlanetaryMotion(ctx, filepath.Join(outputDir, "motion.gif"), planetaryMotionWidth, planetaryMotionHeight,
		planetaryMotionFrames)
}
