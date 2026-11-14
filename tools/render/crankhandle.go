package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The crank lies on its back face with the shaft axis along +Z, the arm along +Y and the set screw's tail face toward
// -Y. Every view shows the default 605ZZ crank except "sizes", which adds the 608-2RS crank beside it.
var crankHandleCameras = map[string]solidlens.Camera{
	"hero":     {Position: solidlens.Vec{X: 70, Y: -75, Z: 75}, Target: solidlens.Vec{Y: 20, Z: 10}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"back":     {Position: solidlens.Vec{X: -70, Y: -60, Z: -55}, Target: solidlens.Vec{Y: 20, Z: 10}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"side":     {Position: solidlens.Vec{X: 130, Y: 20, Z: 25}, Target: solidlens.Vec{Y: 20, Z: 18}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"tail":     {Position: solidlens.Vec{X: 35, Y: -95, Z: 18}, Target: solidlens.Vec{Y: 20, Z: 12}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"sizes":    {Position: solidlens.Vec{X: 80, Y: -110, Z: 110}, Target: solidlens.Vec{X: 25, Y: 20, Z: 10}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"cutaway":  {Position: solidlens.Vec{X: 6, Y: -14, Z: 44}, Target: solidlens.Vec{Y: -3, Z: 5}, Up: solidlens.Vec{Y: 1}, FOV: 30},
	"exploded": {Position: solidlens.Vec{X: 140, Y: -25, Z: 55}, Target: solidlens.Vec{Y: 5, Z: 12}, Up: solidlens.Vec{Z: 1}, FOV: 35},
}

// The exploded view pulls the nut and then the bolt out of the tail face along -Y, the way they go in.
const (
	crankExplodeNut  = 12.0
	crankExplodeBolt = 26.0
)

var (
	crankColor = solidlens.RGB(.22, .45, .75)
	boltColor  = solidlens.RGB(.5, .52, .56)
)

// crankMeshes is one built crank: the crank, bolt and nut meshes.
type crankMeshes struct {
	crank, bolt, nut *decad.Mesh
	dm               crankhandle.Dimensions
	p                crankhandle.Params
}

func buildCrankMeshes(ctx context.Context, p crankhandle.Params) (crankMeshes, error) {
	m, err := crankhandle.Build(ctx, decad.New(), p)
	if err != nil {
		return crankMeshes{}, fmt.Errorf("build crank handle: %w", err)
	}
	out := crankMeshes{dm: m.Dimensions, p: p}
	for _, b := range []struct {
		body *decad.Body
		dst  **decad.Mesh
		tol  float64
	}{{m.Crank, &out.crank, .05}, {m.Bolt.Body, &out.bolt, .01}, {m.Nut.Body, &out.nut, .01}} {
		mesh, err := b.body.Tessellate(ctx, units.Millimeters(b.tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return crankMeshes{}, fmt.Errorf("mesh crank handle: %w", err)
		}
		*b.dst = mesh
	}
	return out, nil
}

func crankModel(mesh solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// shifted moves a mesh by v.
func shifted(mesh solidlens.TriangleSource, v r3.Vec) (solidlens.TriangleSource, error) {
	move, err := r3.Translation(v)
	if err != nil {
		return nil, err
	}
	return rotatedMesh{TriangleSource: mesh, move: move}, nil
}

// models returns the crank, bolt and nut models, with the nut and bolt pulled out along -Y by nut and bolt.
func (c crankMeshes) models(at r3.Vec, nut, bolt float64) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, part := range []struct {
		mesh  *decad.Mesh
		pull  float64
		color solidlens.Color
	}{{c.crank, 0, crankColor}, {c.nut, nut, boltColor}, {c.bolt, bolt, boltColor}} {
		mesh, err := shifted(part.mesh, at.Add(r3.Vec{Y: -part.pull}))
		if err != nil {
			return nil, err
		}
		out = append(out, crankModel(mesh, part.color))
	}
	return out, nil
}

func crankHandleHero(ctx context.Context) (solidlens.Scene, error) {
	return crankHandleView(ctx, "hero")
}

func crankHandleViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return crankHandleView(ctx, name) }
}

// crankHandleView builds the default crank and draws one view.
func crankHandleView(ctx context.Context, name string) (solidlens.Scene, error) {
	c, err := buildCrankMeshes(ctx, crankhandle.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, err
	}
	var large *crankMeshes
	if name == "sizes" {
		l, err := buildCrankMeshes(ctx, crankhandle.Params6082RS())
		if err != nil {
			return solidlens.Scene{}, err
		}
		large = &l
	}
	return c.view(name, large)
}

// view draws one named view. large is the 608-2RS crank of the "sizes" view.
func (c crankMeshes) view(name string, large *crankMeshes) (solidlens.Scene, error) {
	camera, ok := crankHandleCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown crank handle view %q", name)
	}
	var nut, bolt float64
	if name == "exploded" {
		nut, bolt = crankExplodeNut, crankExplodeBolt
	}
	models, err := c.models(r3.Vec{}, nut, bolt)
	if err != nil {
		return solidlens.Scene{}, err
	}
	if large != nil {
		// The 608-2RS crank stands 15 mm clear of the default one along +X.
		more, err := large.models(r3.Vec{X: c.dm.EndRadius + large.dm.HubRadius + 15}, 0, 0)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, more...)
	}
	switch name {
	case "cutaway":
		// Cut every body by the plane through the set screw's axis and remove the half toward +Z, so the view from
		// above shows the D bore, the screw in its hole and the nut in its cavity.
		for i := range models {
			color := crankColor
			if i > 0 {
				color = boltColor
			}
			models[i].Clip = &solidlens.ClipPlane{
				Origin: solidlens.Vec{Z: c.dm.ScrewHeight},
				Normal: solidlens.Vec{Z: 1},
				Cap: &solidlens.Material{Color: solidlens.RGB(color.R+(1-color.R)*.35, color.G+(1-color.G)*.35,
					color.B+(1-color.B)*.35), Unlit: true},
			}
		}
		return bevelScene(camera, models), nil
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

// crankPartScene draws one part alone, from the hero's side.
func crankPartScene(mesh *decad.Mesh, color solidlens.Color, camera solidlens.Camera) solidlens.Scene {
	models := []solidlens.Model{crankModel(mesh, color)}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models)
}

// renderCrankHandleReadme builds the default crank once, and the 608-2RS crank once for the size comparison, and
// writes every README image.
func renderCrankHandleReadme(ctx context.Context, outputDir string) error {
	c, err := buildCrankMeshes(ctx, crankhandle.DefaultParams())
	if err != nil {
		return err
	}
	large, err := buildCrankMeshes(ctx, crankhandle.Params6082RS())
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"back", "back.png", 1600, 900}, {"side", "side.png", 1600, 900}, {"tail", "tail.png", 1600, 900},
		{"sizes", "sizes.png", 1600, 900}, {"cutaway", "cutaway.png", 1600, 900},
		{"exploded", "exploded.png", 1600, 900},
	} {
		var with *crankMeshes
		if shot.view == "sizes" {
			with = &large
		}
		scene, err := c.view(shot.view, with)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	part := solidlens.Camera{Position: solidlens.Vec{X: 60, Y: -70, Z: 60}, Up: solidlens.Vec{Z: 1}, FOV: 35}
	for _, p := range []struct {
		file  string
		mesh  *decad.Mesh
		color solidlens.Color
	}{{"crank.png", c.crank, crankColor}, {"bolt.png", c.bolt, boltColor}, {"nut.png", c.nut, boltColor}} {
		low, high := meshBounds(p.mesh)
		camera := part
		center := low.Add(high).Scale(.5)
		span := high.Sub(low).Len()
		camera.Target = solidlens.Vec(center)
		camera.Position = solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: .8 * span}))
		if err := renderPNG(ctx, filepath.Join(outputDir, "parts", p.file), crankPartScene(p.mesh, p.color, camera),
			480, 270); err != nil {
			return err
		}
	}
	return nil
}
