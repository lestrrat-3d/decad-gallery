package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

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
// face, the nut out of the pin's stub, the handle off the arm, and the pin out of the handle. The cross bolts and their
// nuts go with the pin, each pulled out along its own line: the bolts toward +Y, out of their pockets, and the nuts
// toward -Y.
var crankExplode = map[string]r3.Vec{
	"arm": {}, "bolt": {Z: -18}, "nut": {Z: 16}, "handle": {Z: 32}, "pin": {Z: 75},
	"cross-bolt-1": {Y: 14, Z: 75}, "cross-bolt-2": {Y: 14, Z: 75}, "cross-nut-1": {Y: -14, Z: 75}, "cross-nut-2": {Y: -14, Z: 75},
}

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

var crankPartOrder = []string{"arm", "pin", "handle", "bolt", "nut", "cross-bolt-1", "cross-nut-1", "cross-bolt-2", "cross-nut-2"}

// isCrankHardware reports whether the part is a bought bolt or nut.
func isCrankHardware(name string) bool {
	return strings.Contains(name, "bolt") || strings.Contains(name, "nut")
}

func crankPartColor(name string) solidlens.Color {
	switch {
	case name == "handle":
		return crankHandleColor
	case isCrankHardware(name):
		return crankBoltColor
	}
	return crankArmColor
}

func buildCrankMeshes(ctx context.Context, p crankhandle.Params) (crankMeshes, error) {
	m, err := crankhandle.Build(ctx, crankhandle.NewDocument(), p)
	if err != nil {
		return crankMeshes{}, fmt.Errorf("build crank handle: %w", err)
	}
	out := crankMeshes{mesh: map[string]*decad.Mesh{}, p: p}
	bodies := map[string]*decad.Body{
		"arm": m.Arm, "pin": m.Pin, "handle": m.Handle, "bolt": m.Bolt.Body, "nut": m.Nut.Body,
	}
	for i := range m.CrossBolts {
		bodies[fmt.Sprintf("cross-bolt-%d", i+1)] = m.CrossBolts[i].Body
		bodies[fmt.Sprintf("cross-nut-%d", i+1)] = m.CrossNuts[i].Body
	}
	for name, body := range bodies {
		tol := .05
		if isCrankHardware(name) {
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

// models returns every part moved by at, each part also pulled by its entry of pull.
func (c crankMeshes) models(at r3.Vec, pull map[string]r3.Vec) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, name := range crankPartOrder {
		move, err := r3.Translation(at.Add(pull[name]))
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
	if name == "split" || name == "print" {
		return c.splitView(name)
	}
	camera, ok := crankHandleCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown crank handle view %q", name)
	}
	var pull map[string]r3.Vec
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
		{"split", "split.png", 1600, 900}, {"print", "print.png", 1600, 900},
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
	// One image per distinct part: the cross nuts are the pin bolt's nut, and the two cross bolts are alike.
	for _, name := range []string{"arm", "pin", "handle", "bolt", "nut", "cross-bolt-1"} {
		mesh := c.mesh[name]
		low, high := meshBounds(mesh)
		center := low.Add(high).Scale(.5)
		span := high.Sub(low).Len()
		camera := solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: .8 * span})),
			Target: solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
		models := []solidlens.Model{crankModel(mesh, crankPartColor(name))}
		scene := bevelScene(fitCamera(camera, models, 16.0/9, .1), models)
		file := strings.TrimSuffix(name, "-1")
		if err := renderPNG(ctx, filepath.Join(outputDir, "parts", file+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}

// splitView draws the pin's two printed halves. The pin's mesh moves into a view frame: X along the pin axis (local
// +Z), Y against SplitNormal (local -Y) and Z along local +X, up, so the seam is Y = 0. solidlens.ClipMesh cuts the
// real mesh by the seam into the halves; the model keeps the pin whole.
//
//   - split: the halves pulled 7 mm off the seam, the cross bolts pulled out of their pockets in the +Y half, the nuts
//     in their cavities in the -Y half, and the pin bolt and its nut on the seam, where they sit;
//   - print: both halves lying on their cut faces on the bed, as they print.
func (c crankMeshes) splitView(name string) (solidlens.Scene, error) {
	frame, err := r3.NewFrame(r3.Vec{Y: c.p.Throw}, r3.Vec{Z: 1}, r3.Vec{Y: -1})
	if err != nil {
		return solidlens.Scene{}, err
	}
	place, err := r3.FromFrame(frame)
	if err != nil {
		return solidlens.Scene{}, err
	}
	local, err := place.Inverse()
	if err != nil {
		return solidlens.Scene{}, err
	}
	moved := func(name string, then r3.Transform) (solidlens.TriangleSource, error) {
		xf, err := local.Then(then)
		if err != nil {
			return nil, err
		}
		return rotatedMesh{TriangleSource: c.mesh[name], move: xf}, nil
	}
	pinView := rotatedMesh{TriangleSource: c.mesh["pin"], move: local}
	cut := solidlens.Material{Color: solidlens.RGB(.92, .62, .42), Unlit: true}
	// half returns the half on the side (+1: view +Y, the pin's -Y half, with the nut cavities; -1: the +Y half).
	half := func(side float64, move r3.Transform) (solidlens.Model, error) {
		clipped, err := solidlens.ClipMesh(pinView, solidlens.ClipPlane{Normal: solidlens.Vec{Y: -side}})
		if err != nil {
			return solidlens.Model{}, err
		}
		m := crankModel(rotatedMesh{TriangleSource: clipped, move: move}, crankArmColor)
		m.BackMaterial = &cut
		return m, nil
	}
	shift := func(v r3.Vec) r3.Transform {
		t, _ := r3.Translation(v)
		return t
	}
	var models []solidlens.Model
	var camera solidlens.Camera
	switch name {
	case "split":
		gap := 7.0
		for _, side := range []float64{1, -1} {
			m, err := half(side, shift(r3.Vec{Y: side * gap}))
			if err != nil {
				return solidlens.Scene{}, err
			}
			models = append(models, m)
		}
		for part, at := range map[string]r3.Vec{
			"cross-bolt-1": {Y: -gap - 9}, "cross-bolt-2": {Y: -gap - 9},
			"cross-nut-1": {Y: gap}, "cross-nut-2": {Y: gap}, "bolt": {}, "nut": {},
		} {
			mesh, err := moved(part, shift(at))
			if err != nil {
				return solidlens.Scene{}, err
			}
			models = append(models, crankModel(mesh, crankBoltColor))
		}
		camera = solidlens.Camera{Position: solidlens.Vec{X: -45, Y: -75, Z: 60}, Target: solidlens.Vec{X: 28},
			Up: solidlens.Vec{Z: 1}, FOV: 30}
	case "print":
		// Turning a half about X by ±90° brings its cut face to Z = 0 facing down, the half above it.
		for _, side := range []float64{1, -1} {
			turn, err := r3.Rotation(r3.Vec{X: 1}, units.Radians(side*math.Pi/2))
			if err != nil {
				return solidlens.Scene{}, err
			}
			move, err := turn.Then(shift(r3.Vec{Y: side * 10}))
			if err != nil {
				return solidlens.Scene{}, err
			}
			m, err := half(side, move)
			if err != nil {
				return solidlens.Scene{}, err
			}
			models = append(models, m)
		}
		camera = solidlens.Camera{Position: solidlens.Vec{X: -40, Y: -70, Z: 55}, Target: solidlens.Vec{X: 28, Z: 2},
			Up: solidlens.Vec{Z: 1}, FOV: 30}
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}
