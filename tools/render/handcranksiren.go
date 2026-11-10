package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// handCrankSiren is one build of the default siren, the source of every image of it.
type handCrankSiren struct {
	a *handcranksiren.Assembly
}

func buildHandCrankSiren(ctx context.Context) (handCrankSiren, error) {
	a, err := handcranksiren.Build(ctx, handcranksiren.NewDocument(), handcranksiren.DefaultParams())
	if err != nil {
		return handCrankSiren{}, fmt.Errorf("build hand-crank siren: %w", err)
	}
	return handCrankSiren{a: a}, nil
}

// sirenView is a named camera for the whole assembled siren. The horn points to -Y, the crank to +Y and Z is up.
type sirenView struct {
	shot   string
	camera solidlens.Camera
}

var (
	// The hero looks from the front, left and above, with the horn mouth and the drum in front of the gearbox.
	handCrankSirenHeroCamera = solidlens.Camera{Position: solidlens.Vec{X: -330, Y: -560, Z: 330},
		Target: solidlens.Vec{X: 45, Y: -20, Z: 80}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	handCrankSirenCutawayCamera = solidlens.Camera{Position: solidlens.Vec{X: -480, Y: -300, Z: 230},
		Target: solidlens.Vec{X: 0, Y: -60, Z: 84}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	handCrankSirenExplodedCamera = solidlens.Camera{Position: solidlens.Vec{X: -520, Y: -420, Z: 420},
		Target: solidlens.Vec{X: 45, Y: 20, Z: 90}, Up: solidlens.Vec{Z: 1}, FOV: 35}
)

// handCrankSirenViews are the finished siren from other angles: the rear, where the crank is, the right side and the top.
var handCrankSirenViews = []sirenView{
	{"rear", solidlens.Camera{Position: solidlens.Vec{X: 330, Y: 520, Z: 300}, Target: solidlens.Vec{X: 45, Y: -20, Z: 80},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
	{"side", solidlens.Camera{Position: solidlens.Vec{X: 700, Y: -20, Z: 90}, Target: solidlens.Vec{X: 45, Y: -20, Z: 80},
		Up: solidlens.Vec{Z: 1}, FOV: 35}},
	{"top", solidlens.Camera{Position: solidlens.Vec{X: 45, Y: -20, Z: 700}, Target: solidlens.Vec{X: 45, Y: -20, Z: 80},
		Up: solidlens.Vec{Y: 1}, FOV: 35}},
}

// handCrankSirenExplode is how far the exploded view moves a part, by name. Everything is coaxial along Y, so the offsets
// are fixed: gearbox-front and the drum nuts stay where they are.
func handCrankSirenExplode(name string) r3.Vec {
	switch {
	case strings.HasPrefix(name, "gearbox-front-bolt-"):
		return r3.Vec{Y: -25}
	case strings.HasSuffix(name, "-shaft"):
		return r3.Vec{Z: 160}
	case name == "drum" || strings.HasPrefix(name, "drum-bolt-") || strings.HasPrefix(name, "horn-bolt-"):
		return r3.Vec{Y: -35}
	case name == "rotor":
		return r3.Vec{Y: -65}
	case name == "stator":
		return r3.Vec{Y: -95}
	case name == "horn" || strings.HasPrefix(name, "horn-nut-"):
		return r3.Vec{Y: -135}
	case strings.HasPrefix(name, "bearing-") && strings.HasSuffix(name, "-front"):
		return r3.Vec{Y: 20}
	case name == "intermediate-gear" || name == "rotor-pinion":
		return r3.Vec{Y: 40}
	case name == "crank-gear" || name == "intermediate-pinion":
		return r3.Vec{Y: 60}
	case name == "gearbox-frame" || strings.HasPrefix(name, "gearbox-front-nut-") || strings.HasPrefix(name, "gearbox-rear-nut-"):
		return r3.Vec{Y: 85}
	case strings.HasPrefix(name, "bearing-") && strings.HasSuffix(name, "-rear"):
		return r3.Vec{Y: 110}
	case name == "gearbox-rear" || strings.HasPrefix(name, "gearbox-rear-bolt-"):
		return r3.Vec{Y: 135}
	case name == "crank-arm" || name == "crank-pin-bolt":
		return r3.Vec{Y: 165}
	case name == "crank-handle":
		return r3.Vec{Y: 195}
	case name == "crank-pin" || name == "crank-pin-nut":
		return r3.Vec{Y: 235}
	}
	return r3.Vec{}
}

// sirenMesh is one part of the siren with its model.
type sirenMesh struct {
	part  handcranksiren.Part
	model solidlens.Model
}

// handCrankSirenMeshes meshes every part of the assembly from its real Decad body.
func handCrankSirenMeshes(ctx context.Context, a *handcranksiren.Assembly) ([]sirenMesh, error) {
	meshes := make([]sirenMesh, 0, len(a.Parts))
	for _, part := range a.Parts {
		mesh, err := handCrankSirenMesh(ctx, part)
		if err != nil {
			return nil, err
		}
		meshes = append(meshes, mesh)
	}
	return meshes, nil
}

// handCrankSirenColors are the part colours by top-level container.
var handCrankSirenColors = map[string]solidlens.Color{
	handcranksiren.HousingContainerName:   solidlens.RGB(.42, .47, .55),
	handcranksiren.StatorContainerName:    solidlens.RGB(.50, .56, .62),
	handcranksiren.RotorContainerName:     solidlens.RGB(.16, .52, .70),
	handcranksiren.HornContainerName:      solidlens.RGB(.78, .52, .16),
	handcranksiren.GearPairsContainerName: solidlens.RGB(.80, .62, .20),
	handcranksiren.ShaftsContainerName:    solidlens.RGB(.55, .58, .62),
	handcranksiren.BearingsContainerName:  solidlens.RGB(.72, .76, .80),
	handcranksiren.CrankContainerName:     solidlens.RGB(.70, .22, .12),
	handcranksiren.BoltsContainerName:     solidlens.RGB(.88, .90, .92),
	handcranksiren.NutsContainerName:      solidlens.RGB(.88, .90, .92),
}

// handCrankSirenMesh meshes one part from its real Decad body at 0.1 mm. The 48-tooth and 50-tooth gears use 0.12 when
// Decad refuses 0.1.
func handCrankSirenMesh(ctx context.Context, part handcranksiren.Part) (sirenMesh, error) {
	mesh, err := part.Body.Tessellate(ctx, units.Millimeters(.1), decad.WithVerification(decad.VerifyNone))
	if err != nil && (part.Name == "crank-gear" || part.Name == "intermediate-gear") {
		mesh, err = part.Body.Tessellate(ctx, units.Millimeters(.12), decad.WithVerification(decad.VerifyNone))
	}
	if err != nil {
		return sirenMesh{}, fmt.Errorf("mesh hand-crank siren %s: %w", part.Name, err)
	}
	color, ok := handCrankSirenColors[part.Group]
	if !ok {
		color = solidlens.RGB(.55, .58, .62)
	}
	return sirenMesh{part: part, model: solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1}}}, nil
}

// sceneFrom draws the siren from camera. exploded moves each part by its offset. cutaway clips the drum, the stator and
// the horn by the plane X = 0 and removes the -X half, so the rotor, the vanes and the ports show.
func (s handCrankSiren) sceneFrom(ctx context.Context, camera solidlens.Camera, exploded, cutaway bool) (solidlens.Scene, error) {
	meshes, err := handCrankSirenMeshes(ctx, s.a)
	if err != nil {
		return solidlens.Scene{}, err
	}
	models := make([]solidlens.Model, len(meshes))
	for i, m := range meshes {
		models[i] = m.model
		if exploded {
			models[i].Mesh = placedMesh{TriangleSource: m.model.Mesh, scale: 1, offset: handCrankSirenExplode(m.part.Name)}
		}
		if cutaway && (m.part.Name == "drum" || m.part.Name == "stator" || m.part.Name == "horn") {
			c := handCrankSirenColors[m.part.Group]
			models[i].Clip = &solidlens.ClipPlane{
				Normal: solidlens.Vec{X: -1},
				Cap: &solidlens.Material{
					Color: solidlens.RGB(c.R+(1-c.R)*0.35, c.G+(1-c.G)*0.35, c.B+(1-c.B)*0.35),
					Unlit: true,
				},
			}
		}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .06), models), nil
}

func handCrankSirenPreset(camera solidlens.Camera, exploded, cutaway bool) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		siren, err := buildHandCrankSiren(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return siren.sceneFrom(ctx, camera, exploded, cutaway)
	}
}

// handCrankSirenPart is one unique part as the README's parts table lists it: the image's name, the part to show, the parts
// drawn with it and the direction to look from.
type handCrankSirenPart struct {
	shot, part string
	with       []string
	from       r3.Vec
}

var (
	sirenFront = r3.Vec{X: -.55, Y: -.75, Z: .4} // from the front, left and above
	sirenRear  = r3.Vec{X: .55, Y: .75, Z: .4}   // from behind, right and above
	sirenSide  = r3.Vec{X: .85, Y: -.3, Z: .3}   // from the right
	sirenFace  = r3.Vec{X: -.3, Y: -.9, Z: .3}   // onto a front face
)

var handCrankSirenParts = []handCrankSirenPart{
	{"gearbox-front", "gearbox-front", nil, sirenRear},
	{"gearbox-frame", "gearbox-frame", nil, sirenFront},
	{"gearbox-rear", "gearbox-rear", nil, sirenFront},
	{"drum", "drum", nil, sirenRear},
	{"stator", "stator", nil, sirenFace},
	{"rotor", "rotor", nil, sirenRear},
	{"horn", "horn", nil, sirenFront},
	{"crank-gear", "crank-gear", nil, sirenFace},
	{"intermediate-pinion", "intermediate-pinion", nil, sirenFace},
	{"intermediate-gear", "intermediate-gear", nil, sirenFace},
	{"rotor-pinion", "rotor-pinion", nil, sirenFace},
	{"crank-shaft", "crank-shaft", nil, sirenSide},
	{"intermediate-shaft", "intermediate-shaft", nil, sirenSide},
	{"rotor-shaft", "rotor-shaft", nil, sirenSide},
	{"bearing", "bearing-rotor-front", nil, sirenFace},
	{"crank-arm", "crank-arm", nil, sirenSide},
	{"crank-pin", "crank-pin", nil, sirenSide},
	{"crank-handle", "crank-handle", nil, sirenSide},
	{"gearbox-bolt", "gearbox-front-bolt-1", []string{"gearbox-front-nut-1"}, sirenSide},
	{"drum-bolt", "drum-bolt-1", []string{"drum-nut-1"}, sirenSide},
	{"horn-bolt", "horn-bolt-1", []string{"horn-nut-1"}, sirenSide},
	{"crank-pin-bolt", "crank-pin-bolt", []string{"crank-pin-nut"}, sirenSide},
}

func init() {
	presets["handcranksiren/hero"] = preset{width: 1600, height: 900, scene: handCrankSirenPreset(handCrankSirenHeroCamera, false, false)}
	presets["handcranksiren/cutaway"] = preset{width: 1600, height: 900, scene: handCrankSirenPreset(handCrankSirenCutawayCamera, false, true)}
	presets["handcranksiren/exploded"] = preset{width: 1600, height: 900, scene: handCrankSirenPreset(handCrankSirenExplodedCamera, true, false)}
	for _, view := range handCrankSirenViews {
		presets["handcranksiren/view-"+view.shot] = preset{width: 1200, height: 675, scene: handCrankSirenPreset(view.camera, false, false)}
	}
	for _, part := range handCrankSirenParts {
		presets["handcranksiren/part-"+part.shot] = preset{width: 480, height: 270, scene: func(ctx context.Context) (solidlens.Scene, error) {
			siren, err := buildHandCrankSiren(ctx)
			if err != nil {
				return solidlens.Scene{}, err
			}
			return siren.partScene(ctx, part)
		}}
	}
}

// partScene shows one part, with the parts named in its with field, framed in the middle of the image.
func (s handCrankSiren) partScene(ctx context.Context, part handCrankSirenPart) (solidlens.Scene, error) {
	var models []solidlens.Model
	for _, name := range append([]string{part.part}, part.with...) {
		found := false
		for _, p := range s.a.Parts {
			if p.Name != name {
				continue
			}
			mesh, err := handCrankSirenMesh(ctx, p)
			if err != nil {
				return solidlens.Scene{}, err
			}
			models = append(models, mesh.model)
			found = true
			break
		}
		if !found {
			return solidlens.Scene{}, fmt.Errorf("hand-crank siren has no part %q", name)
		}
	}
	first := models[0].Mesh.Vertices()[0]
	low, high := first, first
	for _, model := range models {
		for _, v := range model.Mesh.Vertices() {
			low = r3.Vec{X: math.Min(low.X, v.X), Y: math.Min(low.Y, v.Y), Z: math.Min(low.Z, v.Z)}
			high = r3.Vec{X: math.Max(high.X, v.X), Y: math.Max(high.Y, v.Y), Z: math.Max(high.Z, v.Z)}
		}
	}
	centre := low.Add(high).Scale(.5)
	direction, _ := part.from.Normalize()
	camera := solidlens.Camera{Position: solidlens.Vec(centre.Add(direction.Scale(3 * high.Sub(low).Len()))),
		Target: solidlens.Vec(centre), Up: solidlens.Vec{Z: 1}, FOV: 35}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

// renderHandCrankSirenReadme writes the hero, the index, the exploded view, the cutaway, the other views and one image per
// unique part from one build. The part images go in the parts directory beside them.
func renderHandCrankSirenReadme(ctx context.Context, outputDir string) error {
	siren, err := buildHandCrankSiren(ctx)
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		name              string
		camera            solidlens.Camera
		exploded, cutaway bool
		width, height     int
	}{
		{"hero.png", handCrankSirenHeroCamera, false, false, 1600, 900},
		{"index.png", handCrankSirenHeroCamera, false, false, 480, 270},
		{"exploded.png", handCrankSirenExplodedCamera, true, false, 1600, 900},
		{"cutaway.png", handCrankSirenCutawayCamera, false, true, 1600, 900},
	} {
		scene, err := siren.sceneFrom(ctx, shot.camera, shot.exploded, shot.cutaway)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.name), scene, shot.width, shot.height); err != nil {
			return fmt.Errorf("render hand-crank siren %s: %w", shot.name, err)
		}
	}
	for _, view := range handCrankSirenViews {
		scene, err := siren.sceneFrom(ctx, view.camera, false, false)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, "view-"+view.shot+".png"), scene, 1200, 675); err != nil {
			return fmt.Errorf("render hand-crank siren view %s: %w", view.shot, err)
		}
	}
	for _, part := range handCrankSirenParts {
		scene, err := siren.partScene(ctx, part)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, "parts", part.shot+".png"), scene, 480, 270); err != nil {
			return fmt.Errorf("render hand-crank siren part %s: %w", part.shot, err)
		}
	}
	return nil
}
