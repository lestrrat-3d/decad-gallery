package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

var (
	plaqueStandColor  = solidlens.RGB(.22, .40, .65)
	plaqueSheetColor  = solidlens.RGB(.80, .81, .84)
	plaqueStandMinMax = [2]plaquestand.Params{
		{PlaqueWidth: 150, PlaqueHeight: 120, PlaqueThickness: 2, Recline: 10, LipHeight: 6, SlotClearance: 0.55},
		{PlaqueWidth: 400, PlaqueHeight: 350, PlaqueThickness: 15, Recline: 25, LipHeight: 20, SlotClearance: 1.5},
	}
)

// plaqueStandMeshes is one built stand's meshes. The stand lies in its local frame: width along +X, front to back
// along +Y, up along +Z.
type plaqueStandMeshes struct {
	stand, plaque *decad.Mesh
	p             plaquestand.Params
	dm            plaquestand.Dimensions
}

func buildPlaqueStandMeshes(ctx context.Context, p plaquestand.Params) (plaqueStandMeshes, error) {
	m, err := plaquestand.Build(ctx, decad.New(), p)
	if err != nil {
		return plaqueStandMeshes{}, fmt.Errorf("build plaque stand: %w", err)
	}
	out := plaqueStandMeshes{p: p, dm: m.Dimensions}
	for _, part := range []struct {
		name string
		body *decad.Body
		dst  **decad.Mesh
	}{{"stand", m.Stand, &out.stand}, {"plaque", m.Plaque, &out.plaque}} {
		mesh, err := part.body.Tessellate(ctx, units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return plaqueStandMeshes{}, fmt.Errorf("mesh plaque stand %s: %w", part.name, err)
		}
		*part.dst = mesh
	}
	return out, nil
}

func plaqueStandModel(mesh solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

func moveBy(v r3.Vec) r3.Transform {
	t, _ := r3.Translation(v)
	return t
}

// camera returns the named view's camera in the stand's local frame, as the starting point for fitCamera.
func (m plaqueStandMeshes) camera(name string) (solidlens.Camera, error) {
	ws, d := m.dm.StandWidth, m.dm.Depth
	up := solidlens.Vec{Z: 1}
	switch name {
	case "hero", "exploded":
		return solidlens.Camera{Position: solidlens.Vec{X: ws/2 - 350, Y: -500, Z: 330},
			Target: solidlens.Vec{X: ws / 2, Y: d / 2, Z: 100}, Up: up, FOV: 35}, nil
	case "stand", "part-stand":
		return solidlens.Camera{Position: solidlens.Vec{X: ws/2 + 250, Y: -260, Z: 220},
			Target: solidlens.Vec{X: ws / 2, Y: d / 2, Z: 20}, Up: up, FOV: 35}, nil
	case "side":
		return solidlens.Camera{Position: solidlens.Vec{X: ws/2 + 800, Y: d / 2, Z: 120},
			Target: solidlens.Vec{X: ws / 2, Y: d / 2, Z: 120}, Up: up, FOV: 25}, nil
	case "back":
		return solidlens.Camera{Position: solidlens.Vec{X: ws/2 + 300, Y: d + 500, Z: 300},
			Target: solidlens.Vec{X: ws / 2, Y: d / 2, Z: 100}, Up: up, FOV: 35}, nil
	case "print":
		// The stand lies on its x = 0 face: (x, y, z) goes to (-z, y, x), so it spans x' in [-Height, 0] and z' in [0, W_s].
		return solidlens.Camera{Position: solidlens.Vec{X: 170, Y: -150, Z: 260},
			Target: solidlens.Vec{X: -m.dm.Height / 2, Y: d / 2, Z: ws / 2}, Up: up, FOV: 35}, nil
	case "sizes":
		return solidlens.Camera{Position: solidlens.Vec{X: 450, Y: -750, Z: 600},
			Target: solidlens.Vec{X: 450, Y: 80, Z: 100}, Up: up, FOV: 35}, nil
	case "part-plaque":
		low, high := meshBounds(m.plaque)
		center, span := low.Add(high).Scale(.5), high.Sub(low).Len()
		return solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: -.6 * span, Y: -1.1 * span, Z: .5 * span})),
			Target: solidlens.Vec(center), Up: up, FOV: 35}, nil
	}
	return solidlens.Camera{}, fmt.Errorf("unknown plaque stand view %q", name)
}

// models returns the stand and the plaque (each when drawn), moved by at, with the plaque also pulled by pull.
func (m plaqueStandMeshes) models(at, pull r3.Vec, stand, plaque bool) []solidlens.Model {
	var out []solidlens.Model
	if stand {
		out = append(out, plaqueStandModel(rotatedMesh{TriangleSource: m.stand, move: moveBy(at)}, plaqueStandColor))
	}
	if plaque {
		out = append(out, plaqueStandModel(rotatedMesh{TriangleSource: m.plaque, move: moveBy(at.Add(pull))},
			plaqueSheetColor))
	}
	return out
}

// view draws one named view of the stand m. small and large are the min and max cases, which only "sizes" reads.
func (m plaqueStandMeshes) view(name string, small, large *plaqueStandMeshes) (solidlens.Scene, error) {
	camera, err := m.camera(name)
	if err != nil {
		return solidlens.Scene{}, err
	}
	var models []solidlens.Model
	switch name {
	case "hero", "side", "back":
		models = m.models(r3.Vec{}, r3.Vec{}, true, true)
	case "stand", "part-stand":
		models = m.models(r3.Vec{}, r3.Vec{}, true, false)
	case "part-plaque":
		models = m.models(r3.Vec{}, r3.Vec{}, false, true)
	case "exploded":
		models = m.models(r3.Vec{}, m.dm.Axis.Scale(m.dm.BackHeight+40), true, true)
	case "print":
		turn, err := r3.Rotation(r3.Vec{Y: 1}, units.Radians(-math.Pi/2))
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = []solidlens.Model{plaqueStandModel(rotatedMesh{TriangleSource: m.stand, move: turn}, plaqueStandColor)}
	case "sizes":
		// The min, default and max cases stand left to right along +X, 40 mm between their plaques' bounding boxes.
		cursor := 0.0
		for _, c := range []*plaqueStandMeshes{small, &m, large} {
			x0 := (c.dm.StandWidth - c.p.PlaqueWidth) / 2
			models = append(models, c.models(r3.Vec{X: cursor - x0}, r3.Vec{}, true, true)...)
			cursor += c.p.PlaqueWidth + 40
		}
		camera.Target.X, camera.Position.X = (cursor-40)/2, (cursor-40)/2
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

func plaqueStandHero(ctx context.Context) (solidlens.Scene, error) {
	return plaqueStandView(ctx, "hero")
}

func plaqueStandViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return plaqueStandView(ctx, name) }
}

// plaqueStandView builds the default stand, and the min and max cases for "sizes", and draws one view.
func plaqueStandView(ctx context.Context, name string) (solidlens.Scene, error) {
	m, err := buildPlaqueStandMeshes(ctx, plaquestand.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, err
	}
	if name != "sizes" {
		return m.view(name, nil, nil)
	}
	small, err := buildPlaqueStandMeshes(ctx, plaqueStandMinMax[0])
	if err != nil {
		return solidlens.Scene{}, err
	}
	large, err := buildPlaqueStandMeshes(ctx, plaqueStandMinMax[1])
	if err != nil {
		return solidlens.Scene{}, err
	}
	return m.view(name, &small, &large)
}

// renderPlaqueStandReadme builds the default stand once, the min and max cases once each, and writes every README image.
func renderPlaqueStandReadme(ctx context.Context, outputDir string) error {
	m, err := buildPlaqueStandMeshes(ctx, plaquestand.DefaultParams())
	if err != nil {
		return err
	}
	small, err := buildPlaqueStandMeshes(ctx, plaqueStandMinMax[0])
	if err != nil {
		return err
	}
	large, err := buildPlaqueStandMeshes(ctx, plaqueStandMinMax[1])
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"stand", "stand.png", 1600, 900}, {"side", "side.png", 1600, 900}, {"back", "back.png", 1600, 900},
		{"exploded", "exploded.png", 1600, 900}, {"print", "print.png", 1600, 900}, {"sizes", "sizes.png", 1600, 900},
		{"part-stand", "parts/stand.png", 480, 270}, {"part-plaque", "parts/plaque.png", 480, 270},
	} {
		scene, err := m.view(shot.view, &small, &large)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	return nil
}
