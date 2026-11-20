package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

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

// plaqueStandPartOrder is every body of a built stand by name: the printed brackets and rail, the bought bolts and
// nuts, and the reference plaque.
var plaqueStandPartOrder = []string{
	"bracket-1", "bracket-2", "rail", "bolt-1", "bolt-2", "nut-1", "nut-2", "plaque",
}

func plaqueStandPartColor(name string) solidlens.Color {
	switch {
	case name == "plaque":
		return plaqueSheetColor
	case strings.Contains(name, "bolt"), strings.Contains(name, "nut"):
		return crankBoltColor
	}
	return plaqueStandColor
}

// plaqueStandMeshes is one built stand's meshes by part name. The stand lies in its local frame: the span along +X,
// front to back along +Y, up along +Z.
type plaqueStandMeshes struct {
	mesh map[string]*decad.Mesh
	p    plaquestand.Params
	dm   plaquestand.Dimensions
}

func buildPlaqueStandMeshes(ctx context.Context, p plaquestand.Params) (plaqueStandMeshes, error) {
	m, err := plaquestand.Build(ctx, decad.New(), p)
	if err != nil {
		return plaqueStandMeshes{}, fmt.Errorf("build plaque stand: %w", err)
	}
	out := plaqueStandMeshes{mesh: map[string]*decad.Mesh{}, p: p, dm: m.Dimensions}
	bodies := map[string]*decad.Body{
		"bracket-1": m.Brackets[0], "bracket-2": m.Brackets[1], "rail": m.Rail, "plaque": m.Plaque,
		"bolt-1": m.Bolts[0].Body, "bolt-2": m.Bolts[1].Body, "nut-1": m.Nuts[0].Body, "nut-2": m.Nuts[1].Body,
	}
	for name, body := range bodies {
		tol := .05
		if plaqueStandPartColor(name) == crankBoltColor {
			tol = .01
		}
		mesh, err := body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return plaqueStandMeshes{}, fmt.Errorf("mesh plaque stand %s: %w", name, err)
		}
		out.mesh[name] = mesh
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
	s, d := m.dm.Span, m.dm.Depth
	up := solidlens.Vec{Z: 1}
	switch name {
	case "hero":
		return solidlens.Camera{Position: solidlens.Vec{X: s/2 - 350, Y: -500, Z: 330},
			Target: solidlens.Vec{X: s / 2, Y: d / 2, Z: 100}, Up: up, FOV: 35}, nil
	case "exploded":
		return solidlens.Camera{Position: solidlens.Vec{X: s/2 + 300, Y: d + 450, Z: 120},
			Target: solidlens.Vec{X: s / 2, Y: d / 2, Z: 100}, Up: up, FOV: 35}, nil
	case "frame":
		return solidlens.Camera{Position: solidlens.Vec{X: s/2 + 250, Y: -260, Z: 220},
			Target: solidlens.Vec{X: s / 2, Y: d / 2, Z: 20}, Up: up, FOV: 35}, nil
	case "side":
		return solidlens.Camera{Position: solidlens.Vec{X: s/2 + 800, Y: d / 2, Z: 120},
			Target: solidlens.Vec{X: s / 2, Y: d / 2, Z: 120}, Up: up, FOV: 25}, nil
	case "back":
		return solidlens.Camera{Position: solidlens.Vec{X: s/2 + 300, Y: d + 500, Z: 300},
			Target: solidlens.Vec{X: s / 2, Y: d / 2, Z: 100}, Up: up, FOV: 35}, nil
	case "print":
		// The brackets lie on their x = 0 faces: (x, y, z) goes to (-z, y, x).
		return solidlens.Camera{Position: solidlens.Vec{X: 250, Y: -250, Z: 300},
			Target: solidlens.Vec{X: -m.dm.Height / 2, Y: d / 2, Z: 0}, Up: up, FOV: 35}, nil
	case "joint":
		// From -X at the rail's height, onto the cut through bracket 1's mid-plane.
		return solidlens.Camera{Position: solidlens.Vec{X: -400, Y: d / 2, Z: 30},
			Target: solidlens.Vec{X: plaquestand.BracketWidth / 2, Y: d / 2, Z: 30}, Up: up, FOV: 25}, nil
	case "sizes":
		return solidlens.Camera{Position: solidlens.Vec{X: 450, Y: -750, Z: 600},
			Target: solidlens.Vec{X: 450, Y: 80, Z: 100}, Up: up, FOV: 35}, nil
	case "part-plaque":
		return m.partCamera("plaque", -.6, -1.1, .5), nil
	}
	if part, ok := strings.CutPrefix(name, "part-"); ok {
		for _, n := range plaqueStandPartOrder {
			if strings.HasPrefix(n, part) {
				return m.partCamera(n, .9, -.9, .8), nil
			}
		}
	}
	return solidlens.Camera{}, fmt.Errorf("unknown plaque stand view %q", name)
}

// partCamera looks at one part from its bounds' centre plus (x, y, z) times the bounds' diagonal.
func (m plaqueStandMeshes) partCamera(name string, x, y, z float64) solidlens.Camera {
	low, high := meshBounds(m.mesh[name])
	center, span := low.Add(high).Scale(.5), high.Sub(low).Len()
	return solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: x * span, Y: y * span, Z: z * span})),
		Target: solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
}

// models returns the parts that keep reports, each moved by at, and by its own entry of pull.
func (m plaqueStandMeshes) models(at r3.Vec, pull map[string]r3.Vec, keep func(string) bool) []solidlens.Model {
	var out []solidlens.Model
	for _, name := range plaqueStandPartOrder {
		if !keep(name) {
			continue
		}
		mesh := rotatedMesh{TriangleSource: m.mesh[name], move: moveBy(at.Add(pull[name]))}
		out = append(out, plaqueStandModel(mesh, plaqueStandPartColor(name)))
	}
	return out
}

func allParts(string) bool        { return true }
func withoutPlaque(n string) bool { return n != "plaque" }

// explode pulls the parts apart the way they go together: the nuts down and out of their pockets, the rail up out of
// the channels, the bolts higher still, and the plaque up the slot axis.
func (m plaqueStandMeshes) explode() map[string]r3.Vec {
	return map[string]r3.Vec{
		"nut-1": {X: -30, Z: -25}, "nut-2": {X: 30, Z: -25}, "rail": {Z: 25}, "bolt-1": {Z: 45}, "bolt-2": {Z: 45},
		"plaque": m.dm.Axis.Scale(m.dm.BackHeight + 80),
	}
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
		models = m.models(r3.Vec{}, nil, allParts)
	case "frame":
		models = m.models(r3.Vec{}, nil, withoutPlaque)
	case "exploded":
		models = m.models(r3.Vec{}, m.explode(), allParts)
	case "joint":
		return m.jointView(camera), nil
	case "print":
		models, err = m.printModels()
		if err != nil {
			return solidlens.Scene{}, err
		}
	case "sizes":
		// The min, default and max cases stand left to right along +X, 40 mm between their plaques' bounding boxes.
		cursor := 0.0
		for _, c := range []*plaqueStandMeshes{small, &m, large} {
			x0 := (c.dm.Span - c.p.PlaqueWidth) / 2
			models = append(models, c.models(r3.Vec{X: cursor - x0}, nil, allParts)...)
			cursor += c.p.PlaqueWidth + 40
		}
		camera.Target.X, camera.Position.X = (cursor-40)/2, (cursor-40)/2
	default:
		part := strings.TrimPrefix(name, "part-")
		models = m.models(r3.Vec{}, nil, func(n string) bool { return n == part || n == part+"-1" })
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

// jointView cuts every body by the plane x = BracketWidth/2, through bracket 1's mid-plane and its bolt, removes the
// outer half and looks into the cut from -X. The camera frames bracket 1.
func (m plaqueStandMeshes) jointView(camera solidlens.Camera) solidlens.Scene {
	models := m.models(r3.Vec{}, nil, allParts)
	for i, name := range plaqueStandPartOrder {
		color := plaqueStandPartColor(name)
		models[i].Clip = &solidlens.ClipPlane{
			Origin: solidlens.Vec{X: plaquestand.BracketWidth / 2}, Normal: solidlens.Vec{X: -1},
			Cap: &solidlens.Material{Color: solidlens.RGB(color.R+(1-color.R)*.35, color.G+(1-color.G)*.35,
				color.B+(1-color.B)*.35), Unlit: true},
		}
	}
	frame := []solidlens.Model{{Mesh: boxCorners{
		{X: plaquestand.BracketWidth/2 - 1, Y: -1, Z: -1},
		{X: plaquestand.BracketWidth/2 + 1, Y: m.dm.Depth + 1, Z: m.dm.Height + 10},
	}}}
	return bevelScene(fitCamera(camera, frame, 16.0/9, .06), models)
}

// printModels lays the printed parts as they print: bracket 1 twice on its x = 0 face with its profile in the bed
// plane, (x, y, z) going to (-z, y, x), the second D + 10 behind the first, and the rail flat on its bottom face 10 mm in
// front of them.
func (m plaqueStandMeshes) printModels() ([]solidlens.Model, error) {
	turn, err := r3.Rotation(r3.Vec{Y: 1}, units.Radians(-math.Pi/2))
	if err != nil {
		return nil, err
	}
	var out []solidlens.Model
	for k := range 2 {
		move, err := turn.Then(moveBy(r3.Vec{Y: float64(k) * (m.dm.Depth + 10)}))
		if err != nil {
			return nil, err
		}
		out = append(out, plaqueStandModel(rotatedMesh{TriangleSource: m.mesh["bracket-1"], move: move}, plaqueStandColor))
	}
	rail := moveBy(r3.Vec{
		X: -m.dm.Height/2 - m.dm.Span/2,
		Y: -10 - (m.dm.RailCenter + plaquestand.RailWidth/2),
		Z: -(plaquestand.BaseThickness - plaquestand.RailHeight + plaquestand.JointFit),
	})
	return append(out, plaqueStandModel(rotatedMesh{TriangleSource: m.mesh["rail"], move: rail}, plaqueStandColor)), nil
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
		{"frame", "frame.png", 1600, 900}, {"joint", "joint.png", 1600, 900}, {"side", "side.png", 1600, 900},
		{"back", "back.png", 1600, 900}, {"exploded", "exploded.png", 1600, 900}, {"print", "print.png", 1600, 900},
		{"sizes", "sizes.png", 1600, 900},
		{"part-bracket", "parts/bracket.png", 480, 270}, {"part-rail", "parts/rail.png", 480, 270},
		{"part-bolt", "parts/bolt.png", 480, 270}, {"part-nut", "parts/nut.png", 480, 270},
		{"part-plaque", "parts/plaque.png", 480, 270},
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
