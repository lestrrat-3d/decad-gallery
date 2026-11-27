package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

type cubeRenderPart struct {
	kind     string
	mesh     *decad.Mesh
	dir      r3.Vec
	distance float64
}

func renderCubePuzzleReadme(ctx context.Context, outputDir string) error {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(ctx, doc, cubepuzzle.DefaultParams())
	if err != nil {
		return fmt.Errorf("build cube: %w", err)
	}
	var parts []cubeRenderPart
	add := func(kind string, body *decad.Body, direction r3.Vec, distance float64) error {
		mesh, err := body.Tessellate(ctx, units.Millimeters(.1), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return fmt.Errorf("mesh %s: %w", kind, err)
		}
		parts = append(parts, cubeRenderPart{kind: kind, mesh: mesh, dir: direction, distance: distance})
		return nil
	}
	if err := add("hub", a.Hub, r3.Vec{}, 0); err != nil {
		return err
	}
	axes := []r3.Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: 1}, {Z: -1}}
	for i, axis := range axes {
		for _, item := range []struct {
			kind string
			body *decad.Body
			d    float64
		}{
			{"center", a.Centers[i], 25}, {"disk", a.Disks[i], 50},
			{"flanged-sleeve", a.FlangedSleeves[i], 20}, {"sleeve", a.Sleeves[i], 35},
		} {
			if err := add(item.kind, item.body, axis, item.d); err != nil {
				return err
			}
		}
	}
	for i, body := range a.Edges {
		pairs := [][2]r3.Vec{{{X: 1}, {Y: 1}}, {{X: 1}, {Z: 1}}, {{Y: 1}, {Z: 1}}}
		x, y := -1.0, -1.0
		if i%4 >= 2 {
			x = 1
		}
		if i%2 == 1 {
			y = 1
		}
		if err := add("edge", body, pairs[i/4][0].Scale(x).Add(pairs[i/4][1].Scale(y)), 25); err != nil {
			return err
		}
	}
	for i, body := range a.Corners {
		x, y, z := -1.0, -1.0, -1.0
		if i >= 4 {
			x = 1
		}
		if i%4 >= 2 {
			y = 1
		}
		if i%2 == 1 {
			z = 1
		}
		if err := add("corner", body, r3.Vec{X: x, Y: y, Z: z}, 25); err != nil {
			return err
		}
	}
	// Bolt and nut sources start at +Z; the other five follow the signed-axis order.
	for i, axis := range []r3.Vec{{Z: 1}, {X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: -1}} {
		if err := add("bolt", a.Bolts[i], axis, 65); err != nil {
			return err
		}
		if err := add("nut", a.Nuts[i], axis, 9); err != nil {
			return err
		}
	}
	for _, view := range []struct {
		name     string
		eye      solidlens.Vec
		exploded bool
		width    int
		height   int
	}{
		{"hero.png", solidlens.Vec{X: 150, Y: -210, Z: 170}, false, 1600, 900},
		{"index.png", solidlens.Vec{X: 150, Y: -210, Z: 170}, false, 480, 270},
		{"back.png", solidlens.Vec{X: -160, Y: 210, Z: -150}, false, 1600, 900},
		{"side.png", solidlens.Vec{X: 160, Y: -5, Z: 3}, false, 1600, 900},
		{"exploded.png", solidlens.Vec{X: 260, Y: -340, Z: 280}, true, 1600, 900},
		{"exploded-back.png", solidlens.Vec{X: -260, Y: 340, Z: -280}, true, 1600, 900},
	} {
		models, err := cubeModels(parts, view.exploded, "")
		if err != nil {
			return err
		}
		camera := solidlens.Camera{Position: view.eye, Up: solidlens.Vec{Z: 1}, FOV: 35}
		scene := renderkit.Scene(renderkit.FitCamera(camera, models, float64(view.width)/float64(view.height), .08), models)
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, view.name), scene, view.width, view.height); err != nil {
			return err
		}
	}
	var mount []cubeRenderPart
	for _, part := range parts {
		if part.kind == "hub" || (part.dir.Z == 1 && part.dir.X == 0 && part.dir.Y == 0) {
			switch part.kind {
			case "nut":
				part.distance = 8
			case "flanged-sleeve":
				part.distance = 15
			case "center":
				part.distance = 30
			case "sleeve":
				part.distance = 55
			case "disk":
				part.distance = 60
			case "bolt":
				part.distance = 100
			}
			mount = append(mount, part)
		}
	}
	models, err := cubeModels(mount, true, "")
	if err != nil {
		return err
	}
	camera := solidlens.Camera{Position: solidlens.Vec{X: 150, Y: -210, Z: 170}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	scene := renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .08), models)
	if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "mount.png"), scene, 1600, 900); err != nil {
		return err
	}
	for _, kind := range []string{"hub", "center", "edge", "corner", "disk", "flanged-sleeve", "sleeve", "bolt", "nut"} {
		models, err := cubeModels(parts, false, kind)
		if err != nil {
			return err
		}
		camera := solidlens.Camera{Position: solidlens.Vec{X: 150, Y: -210, Z: 170}, Up: solidlens.Vec{Z: 1}, FOV: 35}
		scene := renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .08), models)
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "parts", kind+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}

// cubeModels colors only triangles on the six real outer faces. Other triangles
// retain the part material; no geometry is added to represent the markings.
func cubeModels(parts []cubeRenderPart, exploded bool, only string) ([]solidlens.Model, error) {
	colors := []solidlens.Color{
		solidlens.RGB(.09, .10, .12), solidlens.RGB(.82, .10, .08), solidlens.RGB(.96, .42, .06),
		solidlens.RGB(.08, .24, .76), solidlens.RGB(.06, .55, .19), solidlens.RGB(.93, .93, .91), solidlens.RGB(.96, .82, .06),
	}
	var models []solidlens.Model
	for _, part := range parts {
		if only != "" && part.kind != only {
			continue
		}
		vertices := append([]r3.Vec(nil), part.mesh.Vertices()...)
		var groups [7][][3]int
		for _, tri := range part.mesh.Triangles() {
			color := 0
			if part.kind == "center" || part.kind == "edge" || part.kind == "corner" || part.kind == "disk" {
				for axis := range 3 {
					for _, sign := range []float64{1, -1} {
						onFace := true
						for _, index := range tri {
							v := vertices[index]
							coord := [3]float64{v.X, v.Y, v.Z}[axis]
							if math.Abs(coord-sign*30) > 1e-5 {
								onFace = false
							}
						}
						if onFace {
							color = 1 + 2*axis
							if sign < 0 {
								color++
							}
						}
					}
				}
			}
			groups[color] = append(groups[color], tri)
		}
		if exploded {
			for i := range vertices {
				vertices[i] = vertices[i].Add(part.dir.Scale(part.distance))
			}
		}
		for i, triangles := range groups {
			if len(triangles) == 0 {
				continue
			}
			mesh, err := solidlens.NewMesh(vertices, triangles)
			if err != nil {
				return nil, fmt.Errorf("render %s: %w", part.kind, err)
			}
			color := colors[i]
			if i == 0 && part.kind != "center" && part.kind != "edge" && part.kind != "corner" && part.kind != "disk" {
				color = solidlens.RGB(.54, .58, .62)
			}
			models = append(models, solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color)})
		}
		if only != "" {
			break
		}
	}
	return models, nil
}
