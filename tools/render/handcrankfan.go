package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// handCrankFanHero builds the default fan and views it from the operator's side: the rotor faces the camera and
// the crank is on the right.
func handCrankFanHero(ctx context.Context) (solidlens.Scene, error) {
	return handCrankFanScene(ctx, false)
}

// handCrankFanExploded is the same fan pulled apart by Decad's parts.Set.Explode along the joints the model declares.
func handCrankFanExploded(ctx context.Context) (solidlens.Scene, error) {
	return handCrankFanScene(ctx, true)
}

func handCrankFanScene(ctx context.Context, exploded bool) (solidlens.Scene, error) {
	fan, err := buildHandCrankFan(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return fan.scene(ctx, exploded)
}

// handCrankFan is one build of the default fan, the source of every image of it.
type handCrankFan struct {
	doc *decad.Document
	a   *handcrankfan.Assembly
}

func buildHandCrankFan(ctx context.Context) (handCrankFan, error) {
	doc := handcrankfan.NewDocument()
	a, err := handcrankfan.Build(ctx, doc, handcrankfan.DefaultParams())
	if err != nil {
		return handCrankFan{}, fmt.Errorf("build hand-crank fan: %w", err)
	}
	return handCrankFan{doc: doc, a: a}, nil
}

// scene shows the whole fan from the operator's side, or pulled apart along its joints.
func (f handCrankFan) scene(ctx context.Context, exploded bool) (solidlens.Scene, error) {
	offsets := map[string]r3.Vec{}
	if exploded {
		set, err := f.a.Set(f.doc)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("set up hand-crank fan parts: %w", err)
		}
		layout, err := f.a.Explode(ctx, set)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("explode hand-crank fan: %w", err)
		}
		for _, p := range layout.Parts {
			offsets[p.Name] = p.Offset
		}
	}
	meshes, err := handCrankFanMeshes(ctx, f.a)
	if err != nil {
		return solidlens.Scene{}, err
	}
	models := make([]solidlens.Model, len(meshes))
	for i, mesh := range meshes {
		models[i] = mesh.model
		if exploded {
			models[i].Mesh = placedMesh{TriangleSource: mesh.model.Mesh, scale: 1, offset: offsets[mesh.part.Name]}
		}
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 330, Y: -560, Z: 330},
		Target: solidlens.Vec{X: 20, Y: -40, Z: 120}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	if exploded {
		camera = solidlens.Camera{Position: solidlens.Vec{X: 440, Y: -620, Z: 560},
			Target: solidlens.Vec{X: 60, Y: -60, Z: 120}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .06), models), nil
}

// handCrankFanPart is one unique part of the fan, as the README's parts table lists it: the image's name, the part of
// the assembly to show, and the direction to look at it from. Blades, bolts and bearings are shown once.
type handCrankFanPart struct {
	shot, part string
	from       r3.Vec
}

var (
	partFront  = r3.Vec{X: .55, Y: -.75, Z: .4} // from the front, right and above, the rotor's side
	partFace   = r3.Vec{X: -.8, Y: -.5, Z: .35} // from the left, which shows the face of a gear on the X axis
	partSide   = r3.Vec{X: .8, Y: -.4, Z: .3}   // from the right, which shows the face of the crank arm
	partRotor  = r3.Vec{X: .4, Y: -.85, Z: .35} // from the front, along the fan axis
	partCavity = r3.Vec{X: .6, Y: .5, Z: .6}    // from behind and above, into the open top and onto the seams
	partPinion = r3.Vec{X: .4, Y: -.8, Z: .4}
)

var handCrankFanParts = []handCrankFanPart{
	{"gearbox-base", "gearbox-base", partCavity},
	{"gearbox-post", "gearbox-post", partFace},
	{"gearbox-crank-wall", "gearbox-crank-wall", partCavity},
	{"gearbox-fan-wall", "gearbox-fan-wall", partCavity},
	{"gearbox-bolt", "gearbox-bolt-1", partFront},
	{"gearbox-screw", "gearbox-screw-1", partFront},
	{"grip", "handle", partFront},
	{"grip-screw", "handle-bolt-1", partFront},
	{"crank-gear", "crank-gear", partFace},
	{"drive-gear", "drive-gear", partFace},
	{"driving-gear", "driving-gear", partFace},
	{"pinion", "pinion", partPinion},
	{"crank-shaft", "crank-shaft", partFront},
	{"drive-shaft", "drive-shaft", partFront},
	{"fan-shaft", "fan-shaft", partFront},
	{"bearing", "bearing-crank-post", partFace},
	{"crank-arm", "crank-arm", partSide},
	{"crank-pin", "crank-pin", partFront},
	{"crank-pin-screw", "crank-pin-bolt", partFront},
	{"crank-handle", "crank-grip", partFront},
	{"fan-hub", "fan-hub", partRotor},
	{"fan-blade", "fan-blade-1", partRotor},
	{"fan-bolt", "fan-bolt-1", partRotor},
}

func init() {
	for _, part := range handCrankFanParts {
		presets["handcrankfan/part-"+part.shot] = preset{width: 480, height: 270, scene: handCrankFanPartPreset(part)}
	}
}

// partScene shows one part of the fan alone, in place, framed in the middle of the image.
func (f handCrankFan) partScene(ctx context.Context, part handCrankFanPart) (solidlens.Scene, error) {
	for _, p := range f.a.Parts {
		if p.Name != part.part {
			continue
		}
		mesh, err := handCrankFanMesh(ctx, p)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models := []solidlens.Model{mesh.model}
		low, high := models[0].Mesh.Vertices()[0], models[0].Mesh.Vertices()[0]
		for _, v := range models[0].Mesh.Vertices() {
			low = r3.Vec{X: math.Min(low.X, v.X), Y: math.Min(low.Y, v.Y), Z: math.Min(low.Z, v.Z)}
			high = r3.Vec{X: math.Max(high.X, v.X), Y: math.Max(high.Y, v.Y), Z: math.Max(high.Z, v.Z)}
		}
		centre := low.Add(high).Scale(.5)
		direction, _ := part.from.Normalize()
		camera := solidlens.Camera{Position: solidlens.Vec(centre.Add(direction.Scale(3 * high.Sub(low).Len()))),
			Target: solidlens.Vec(centre), Up: solidlens.Vec{Z: 1}, FOV: 35}
		return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
	}
	return solidlens.Scene{}, fmt.Errorf("hand-crank fan has no part %q", part.part)
}

// handCrankFanPartPreset builds the fan for that one part.
func handCrankFanPartPreset(part handCrankFanPart) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		fan, err := buildHandCrankFan(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return fan.partScene(ctx, part)
	}
}

// renderHandCrankFanReadme writes the hero, the index, the exploded view and one image per unique part from one build.
// The part images go in the parts directory beside them.
func renderHandCrankFanReadme(ctx context.Context, outputDir string) error {
	fan, err := buildHandCrankFan(ctx)
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		name          string
		exploded      bool
		width, height int
	}{
		{"hero.png", false, 1600, 900}, {"index.png", false, 480, 270}, {"exploded.png", true, 1600, 900},
	} {
		scene, err := fan.scene(ctx, shot.exploded)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.name), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	for _, part := range handCrankFanParts {
		scene, err := fan.partScene(ctx, part)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, "parts", part.shot+".png"), scene, 480, 270); err != nil {
			return fmt.Errorf("render hand-crank fan part %s: %w", part.shot, err)
		}
	}
	return nil
}

// fanMesh is one part of the fan with its mesh, colour and outlines.
type fanMesh struct {
	part  handcrankfan.Part
	model solidlens.Model
}

// handCrankFanMeshes meshes every part of the assembly from its real Decad body.
func handCrankFanMeshes(ctx context.Context, a *handcrankfan.Assembly) ([]fanMesh, error) {
	meshes := make([]fanMesh, 0, len(a.Parts))
	for _, part := range a.Parts {
		mesh, err := handCrankFanMesh(ctx, part)
		if err != nil {
			return nil, err
		}
		meshes = append(meshes, mesh)
	}
	return meshes, nil
}

// handCrankFanMesh meshes one part from its real Decad body and colours it by its group.
func handCrankFanMesh(ctx context.Context, part handcrankfan.Part) (fanMesh, error) {
	// The crank shaft's grooved end is meshed at 0.2 mm, as the tests check it, and the pinion's spline bore holds its
	// mesh to 0.103 mm.
	tolerance := .1
	switch part.Name {
	case "crank-shaft":
		tolerance = .2
	case "pinion", "driving-gear":
		tolerance = .12
	}
	mesh, err := part.Body.Tessellate(ctx, units.Millimeters(tolerance), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		return fanMesh{}, fmt.Errorf("mesh hand-crank fan %s: %w", part.Name, err)
	}
	color := solidlens.RGB(.55, .58, .62)
	switch part.Group {
	case "stand", "housing":
		color = solidlens.RGB(.42, .47, .55)
	case "gear":
		color = solidlens.RGB(.80, .62, .20)
	case "rotor":
		color = solidlens.RGB(.16, .52, .70)
	case "blade":
		color = solidlens.RGB(.30, .68, .85)
	case "hardware":
		color = solidlens.RGB(.88, .90, .92)
	case "bearing":
		color = solidlens.RGB(.72, .76, .80)
	case "arm", "pin":
		color = solidlens.RGB(.70, .22, .12)
	case "grip":
		color = solidlens.RGB(.12, .12, .14)
	}
	return fanMesh{part: part, model: solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1}}}, nil
}
