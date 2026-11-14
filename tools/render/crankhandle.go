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

// The crank lies on the arm's back face with the shaft axis along +Z, the arm along +Y and the handle standing up on
// the pin at (0, 55). Every view shows the default 605ZZ crank except "sizes", which adds the 608-2RS crank beside it.
var crankHandleCameras = map[string]solidlens.Camera{
	"hero":     {Position: solidlens.Vec{X: 95, Y: -60, Z: 85}, Target: solidlens.Vec{Y: 28, Z: 20}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"back":     {Position: solidlens.Vec{X: -70, Y: -55, Z: -60}, Target: solidlens.Vec{Y: 28, Z: 20}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"side":     {Position: solidlens.Vec{X: 150, Y: 28, Z: 30}, Target: solidlens.Vec{Y: 28, Z: 28}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"sizes":    {Position: solidlens.Vec{X: 90, Y: -110, Z: 110}, Target: solidlens.Vec{X: 18, Y: 28, Z: 20}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"cutaway":  {Position: solidlens.Vec{X: 150, Y: 20, Z: 40}, Target: solidlens.Vec{Y: 28, Z: 28}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"exploded": {Position: solidlens.Vec{X: 190, Y: -60, Z: 80}, Target: solidlens.Vec{Y: 30, Z: 55}, Up: solidlens.Vec{Z: 1}, FOV: 35},
}

// The exploded view pulls the parts apart along the pin axis, the way they go together: the bolt out of the arm's back
// face, the nut out of the pin's stub, the handle off the arm, and the pin out of the handle.
var crankExplode = map[string]float64{"arm": 0, "bolt": -18, "nut": 16, "handle": 32, "pin": 75}

var (
	crankArmColor    = solidlens.RGB(.70, .22, .12) // the hand-crank siren's crank colour
	crankHandleColor = solidlens.RGB(.20, .22, .25)
	crankBoltColor   = solidlens.RGB(.55, .57, .60)
)

// crankMeshes is one built crank's meshes by part name.
type crankMeshes struct {
	mesh map[string]*decad.Mesh
	p    crankhandle.Params
}

var crankPartOrder = []string{"arm", "pin", "handle", "bolt", "nut"}

func crankPartColor(name string) solidlens.Color {
	switch name {
	case "handle":
		return crankHandleColor
	case "bolt", "nut":
		return crankBoltColor
	}
	return crankArmColor
}

func buildCrankMeshes(ctx context.Context, p crankhandle.Params) (crankMeshes, error) {
	m, err := crankhandle.Build(ctx, decad.New(), p)
	if err != nil {
		return crankMeshes{}, fmt.Errorf("build crank handle: %w", err)
	}
	out := crankMeshes{mesh: map[string]*decad.Mesh{}, p: p}
	for name, body := range map[string]*decad.Body{
		"arm": m.Arm, "pin": m.Pin, "handle": m.Handle, "bolt": m.Bolt.Body, "nut": m.Nut.Body,
	} {
		tol := .05
		if name == "bolt" || name == "nut" {
			tol = .01
		}
		mesh, err := body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return crankMeshes{}, fmt.Errorf("mesh crank %s: %w", name, err)
		}
		out.mesh[name] = mesh
	}
	return out, nil
}

func crankModel(mesh solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// models returns every part moved by at, each part also pulled along +Z by its entry of pull.
func (c crankMeshes) models(at r3.Vec, pull map[string]float64) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, name := range crankPartOrder {
		move, err := r3.Translation(at.Add(r3.Vec{Z: pull[name]}))
		if err != nil {
			return nil, err
		}
		out = append(out, crankModel(rotatedMesh{TriangleSource: c.mesh[name], move: move}, crankPartColor(name)))
	}
	return out, nil
}

func crankHandleHero(ctx context.Context) (solidlens.Scene, error) {
	return crankHandleView(ctx, "hero")
}

func crankHandleViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return crankHandleView(ctx, name) }
}

// crankHandleView builds the default crank, and the 608-2RS crank for "sizes", and draws one view.
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
	var pull map[string]float64
	if name == "exploded" {
		pull = crankExplode
	}
	models, err := c.models(r3.Vec{}, pull)
	if err != nil {
		return solidlens.Scene{}, err
	}
	if large != nil {
		// The 608-2RS crank lies 15 mm clear of the default one along +X.
		more, err := large.models(r3.Vec{X: 2*crankhandle.ArmRadius + 15}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, more...)
	}
	if name == "cutaway" {
		// Cut every body by the plane x = 0, which holds the shaft axis and the pin axis, and remove the half toward +X,
		// so the view from +X shows the D bore, the pin clamped in the arm, and the handle round the pin.
		for i, name := range crankPartOrder {
			color := crankPartColor(name)
			models[i].Clip = &solidlens.ClipPlane{
				Normal: solidlens.Vec{X: 1},
				Cap: &solidlens.Material{Color: solidlens.RGB(color.R+(1-color.R)*.35, color.G+(1-color.G)*.35,
					color.B+(1-color.B)*.35), Unlit: true},
			}
		}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
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
		{"back", "back.png", 1600, 900}, {"side", "side.png", 1600, 900},
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
	for _, name := range crankPartOrder {
		mesh := c.mesh[name]
		low, high := meshBounds(mesh)
		center := low.Add(high).Scale(.5)
		span := high.Sub(low).Len()
		camera := solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: .8 * span})),
			Target: solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
		models := []solidlens.Model{crankModel(mesh, crankPartColor(name))}
		scene := bevelScene(fitCamera(camera, models, 16.0/9, .1), models)
		if err := renderPNG(ctx, filepath.Join(outputDir, "parts", name+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}
