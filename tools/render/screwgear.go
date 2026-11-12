package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// Tessellation tolerances in millimeters. The cage is a faceted body whose mesh bound is 0.189 mm,
// so decad refuses anything finer; the ribbons use the helical gear's 0.1 mm.
const (
	screwGearRibbonTolerance = 0.1
	screwGearCageTolerance   = 0.2
)

// screwGearCameras look at the default mechanism. The sleeve sits at the origin with its axis along Z, and the
// two ribbons cross it at 80 degrees. fitCamera centers and sizes each view on the sleeve and the stretch of
// ribbon next to it (see screwGearFocus).
var screwGearCameras = map[string]solidlens.Camera{
	"hero": {
		Position: solidlens.Vec{X: 60, Y: -300, Z: 110},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
	"top": {
		Position: solidlens.Vec{Z: 400},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Y: 1},
		FOV:      30,
	},
	"low": {
		Position: solidlens.Vec{X: -40, Y: -300, Z: -60},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
	"cutaway": {
		Position: solidlens.Vec{X: 90, Y: -300, Z: 130},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
	"exploded": {
		Position: solidlens.Vec{X: 120, Y: -420, Z: 330},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
	"part": {
		Position: solidlens.Vec{X: 60, Y: -300, Z: 110},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	},
}

// screwGearExplodeDistance is how far each ribbon is pulled out along its own axis, in millimeters. The ribbons
// are 178.5 mm long and the sleeve is 18.5 mm in radius, so 115 mm takes each ribbon's near end clear of the sleeve.
const screwGearExplodeDistance = 115.0

// screwGearExplodeLift is how far the ribbons are moved apart along the sleeve's axis, so they do not overlap
// in the exploded view.
const screwGearExplodeLift = 40.0

// screwGearPartNames are the file names of the three bodies in images/parts, in model order.
var screwGearPartNames = [3]string{"ribbon-a", "ribbon-b", "sleeve"}

// screwGearColors are ribbon A, ribbon B and the cage.
var screwGearColors = [3]solidlens.Color{
	solidlens.RGB(0.14, 0.4, 0.58),
	solidlens.RGB(0.72, 0.4, 0.14),
	solidlens.RGB(0.5, 0.56, 0.62),
}

// screwGearModels builds the default mechanism and returns ribbon A, ribbon B and the cage as three meshes.
// Ribbons cannot be boolean operands, so each body is its own model.
func screwGearModels(ctx context.Context) ([]solidlens.Model, error) {
	mechanism, err := screwgear.Build(ctx, screwgear.NewDocument(), screwgear.DefaultParams())
	if err != nil {
		return nil, fmt.Errorf("build screw gear: %w", err)
	}
	return screwGearMeshes(ctx, mechanism)
}

// screwGearMeshes tessellates the three bodies of a built mechanism into solidlens models.
func screwGearMeshes(ctx context.Context, mechanism *screwgear.Mechanism) ([]solidlens.Model, error) {
	var models []solidlens.Model
	bodies := []*decad.Body{mechanism.GearA, mechanism.GearB, mechanism.Cage}
	tolerances := []float64{screwGearRibbonTolerance, screwGearRibbonTolerance, screwGearCageTolerance}
	for i, body := range bodies {
		mesh, err := body.Tessellate(ctx, units.Millimeters(tolerances[i]), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate screw gear body %d: %w", i, err)
		}
		models = append(models, solidlens.Model{
			Mesh:     mesh,
			Material: solidlens.Matte(screwGearColors[i]),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
	}
	return models, nil
}

// screwGearFocus is the part of the mechanism a view is framed on: the vertices within this distance of the
// origin. The ribbons are far longer than the sleeve, so framing on every vertex would shrink the sleeve to a speck.
const screwGearFocus = 70.0

// screwGearScene frames the camera on the sleeve and the ribbon next to it.
func screwGearScene(camera solidlens.Camera, models []solidlens.Model, aspect float64) solidlens.Scene {
	var near []r3.Vec
	for _, model := range models {
		for _, v := range model.Mesh.Vertices() {
			if v.Dot(v) <= screwGearFocus*screwGearFocus {
				near = append(near, v)
			}
		}
	}
	focus := []solidlens.Model{{Mesh: pointCloud(near)}}
	return bevelScene(fitCamera(camera, focus, aspect, bevelViewMargin), models)
}

// screwGearExplode pulls each ribbon out along its own axis and lifts the two apart along the sleeve's axis.
// The sleeve stays in place. The axes lie at plus and minus half the crossing angle from X in the unplaced default
// mechanism, so the two ribbons end up on either side of the sleeve.
func screwGearExplode(models []solidlens.Model, crossAngle float64) []solidlens.Model {
	half := crossAngle / 2
	dirs := [2]r3.Vec{
		r3.NewVec(math.Cos(half), math.Sin(half), 0),
		r3.NewVec(math.Cos(half), -math.Sin(half), 0),
	}
	out := make([]solidlens.Model, len(models))
	copy(out, models)
	for i, dir := range dirs {
		lift := screwGearExplodeLift
		if i == 0 {
			lift = -lift
		}
		offset := dir.Scale(screwGearExplodeDistance).Add(r3.NewVec(0, 0, lift))
		out[i].Mesh = placedMesh{TriangleSource: models[i].Mesh, scale: 1, offset: offset}
	}
	return out
}

// pointCloud is a vertex list that stands in for a mesh when only extents matter.
type pointCloud []r3.Vec

func (p pointCloud) Vertices() []r3.Vec { return p }

func (p pointCloud) Triangles() [][3]int { return nil }

// screwGearCutaway cuts all three bodies open along the plane through the sleeve's axis and the reference
// direction ê, the world XZ plane when the mechanism is unplaced. The half toward negative Y is removed, which is the
// side the cutaway camera looks from, so the view shows the inside of the sleeve and the ribbons passing through its
// bores. The cut faces are flat, in a lighter shade of each body's colour.
func screwGearCutaway(models []solidlens.Model) []solidlens.Model {
	out := make([]solidlens.Model, len(models))
	for i, model := range models {
		capColor := screwGearColors[i]
		model.Clip = &solidlens.ClipPlane{
			Normal: solidlens.Vec{Y: -1},
			Cap: &solidlens.Material{
				Color: solidlens.RGB(capColor.R+(1-capColor.R)*0.35, capColor.G+(1-capColor.G)*0.35, capColor.B+(1-capColor.B)*0.35),
				Unlit: true,
			},
		}
		out[i] = model
	}
	return out
}

// screwGearExplodedScene frames every vertex of the pulled-apart mechanism, unlike the other views.
func screwGearExplodedScene(models []solidlens.Model) solidlens.Scene {
	models = screwGearExplode(models, screwgear.DefaultParams().CrossAngle)
	return bevelScene(fitCamera(screwGearCameras["exploded"], models, 16.0/9, bevelViewMargin), models)
}

// screwGearPartScene shows one body alone. The sleeve is framed like the assembled views. A ribbon is framed whole
// and seen from the side of its own axis, which lies at plus or minus half the crossing angle from X.
func screwGearPartScene(models []solidlens.Model, i int) solidlens.Scene {
	if i == 2 {
		return screwGearScene(screwGearCameras["part"], models[i:i+1], 16.0/9)
	}
	half := screwgear.DefaultParams().CrossAngle / 2
	if i == 1 {
		half = -half
	}
	side := solidlens.Vec{X: math.Sin(half), Y: -math.Cos(half)}
	camera := solidlens.Camera{
		Position: solidlens.Vec{X: side.X * 300, Y: side.Y * 300, Z: 90},
		Target:   solidlens.Vec{},
		Up:       solidlens.Vec{Z: 1},
		FOV:      30,
	}
	own := models[i : i+1]
	return bevelScene(fitCamera(camera, own, 16.0/9, bevelViewMargin), own)
}

func screwGearView(shot string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		models, err := screwGearModels(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		switch shot {
		case "cutaway":
			models = screwGearCutaway(models)
		case "exploded":
			return screwGearExplodedScene(models), nil
		}
		return screwGearScene(screwGearCameras[shot], models, 16.0/9), nil
	}
}

// renderScrewGearReadme builds the mechanism once and writes index.png, hero.png, top.png and low.png.
func renderScrewGearReadme(ctx context.Context, outputDir string) error {
	mechanism, err := screwgear.Build(ctx, screwgear.NewDocument(), screwgear.DefaultParams())
	if err != nil {
		return fmt.Errorf("build screw gear: %w", err)
	}
	models, err := screwGearMeshes(ctx, mechanism)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name, shot    string
		width, height int
	}{
		{"index.png", "hero", 480, 270},
		{"hero.png", "hero", 1600, 900},
		{"top.png", "top", 1600, 900},
		{"low.png", "low", 1600, 900},
	} {
		scene := screwGearScene(screwGearCameras[image.shot], models, 16.0/9)
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), scene, image.width, image.height); err != nil {
			return fmt.Errorf("render screwgear %s: %w", image.name, err)
		}
	}
	exploded := screwGearExplodedScene(models)
	if err := renderPNG(ctx, filepath.Join(outputDir, "exploded.png"), exploded, 1600, 900); err != nil {
		return fmt.Errorf("render screwgear exploded.png: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "parts"), 0o755); err != nil {
		return fmt.Errorf("create screwgear parts directory: %w", err)
	}
	for i, name := range screwGearPartNames {
		path := filepath.Join(outputDir, "parts", name+".png")
		if err := renderPNG(ctx, path, screwGearPartScene(models, i), 480, 270); err != nil {
			return fmt.Errorf("render screwgear parts/%s.png: %w", name, err)
		}
	}
	cutaway := screwGearScene(screwGearCameras["cutaway"], screwGearCutaway(models), 16.0/9)
	if err := renderPNG(ctx, filepath.Join(outputDir, "cutaway.png"), cutaway, 1600, 900); err != nil {
		return fmt.Errorf("render screwgear cutaway.png: %w", err)
	}
	return nil
}
