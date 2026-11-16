package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The fan stands with its shaft axis along +Z, the hub's back face at z = 0 and the front face at z = 20, blade 1 along
// +Y and the clamp tab toward -Y. Every camera has +Y up, so blade 1 points up in every view.
var (
	turbineHubColor      = solidlens.RGB(.16, .52, .70) // the hand-crank fan's rotor colour
	turbineBladeColor    = solidlens.RGB(.30, .68, .85) // the hand-crank fan's blade colour
	turbineHardwareColor = solidlens.RGB(.55, .57, .60)
)

// turbineDistance is the camera's distance from its target, in mm.
const turbineDistance = 380

// turbineCamera looks at target from the direction dir, with +Y up.
func turbineCamera(dir, target r3.Vec) solidlens.Camera {
	d, _ := dir.Normalize()
	return solidlens.Camera{Position: solidlens.Vec(target.Add(d.Scale(turbineDistance))), Target: solidlens.Vec(target),
		Up: solidlens.Vec{Y: 1}, FOV: 35}
}

// turbineExplode is each part's offset in the exploded view: every blade outward along its radial line, each blade
// bolt out of the front face and each blade nut out of the back, the clamp bolt out of the +X side face and its nut out
// of the -X side face.
func turbineExplode() map[string]r3.Vec {
	out := map[string]r3.Vec{"hub": {}, "clamp-bolt": {X: 25}, "clamp-nut": {X: -18}}
	for k := 1; k <= turbinefan.BladeCount; k++ {
		a := turbinefan.BladeAngle(k)
		out[fmt.Sprintf("blade-%d", k)] = r3.Vec{X: 35 * math.Cos(a), Y: 35 * math.Sin(a)}
		out[fmt.Sprintf("bolt-%d", k)] = r3.Vec{Z: 30}
		out[fmt.Sprintf("nut-%d", k)] = r3.Vec{Z: -14}
	}
	return out
}

// turbineMeshes is one built fan's meshes by part name.
type turbineMeshes struct {
	mesh map[string]*decad.Mesh
	dm   turbinefan.Dimensions
}

var turbinePartOrder = []string{
	"hub", "blade-1", "blade-2", "blade-3", "bolt-1", "nut-1", "bolt-2", "nut-2", "bolt-3", "nut-3", "clamp-bolt", "clamp-nut",
}

func isTurbineHardware(name string) bool {
	return strings.Contains(name, "bolt") || strings.Contains(name, "nut")
}

func turbinePartColor(name string) solidlens.Color {
	switch {
	case strings.HasPrefix(name, "blade"):
		return turbineBladeColor
	case isTurbineHardware(name):
		return turbineHardwareColor
	}
	return turbineHubColor
}

// buildTurbineMeshes builds the fan once and tessellates its printed parts at 0.05 mm and its hardware at 0.01 mm.
func buildTurbineMeshes(ctx context.Context, p turbinefan.Params) (turbineMeshes, error) {
	m, err := turbinefan.Build(ctx, turbinefan.NewDocument(), p)
	if err != nil {
		return turbineMeshes{}, fmt.Errorf("build turbine fan: %w", err)
	}
	out := turbineMeshes{mesh: map[string]*decad.Mesh{}, dm: m.Dimensions}
	bodies := map[string]*decad.Body{"hub": m.Hub, "clamp-bolt": m.ClampBolt.Body, "clamp-nut": m.ClampNut.Body}
	for k := range m.Blades {
		bodies[fmt.Sprintf("blade-%d", k+1)] = m.Blades[k]
		bodies[fmt.Sprintf("bolt-%d", k+1)] = m.BladeBolts[k].Body
		bodies[fmt.Sprintf("nut-%d", k+1)] = m.BladeNuts[k].Body
	}
	for name, body := range bodies {
		tol := .05
		if isTurbineHardware(name) {
			tol = .01
		}
		mesh, err := body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return turbineMeshes{}, fmt.Errorf("mesh turbine fan %s: %w", name, err)
		}
		out.mesh[name] = mesh
	}
	return out, nil
}

func turbineModel(mesh solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// clipped returns the model cut by plane, the half toward the plane's normal removed, with a light cap of its colour.
func clipped(m solidlens.Model, color solidlens.Color, origin, normal solidlens.Vec) solidlens.Model {
	m.Clip = &solidlens.ClipPlane{Origin: origin, Normal: normal,
		Cap: &solidlens.Material{Color: solidlens.RGB(color.R+(1-color.R)*.35, color.G+(1-color.G)*.35,
			color.B+(1-color.B)*.35), Unlit: true}}
	return m
}

// models returns the named parts, each moved by at and by its entry of pull.
func (c turbineMeshes) models(names []string, at r3.Vec, pull map[string]r3.Vec) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, name := range names {
		move, err := r3.Translation(at.Add(pull[name]))
		if err != nil {
			return nil, err
		}
		out = append(out, turbineModel(rotatedMesh{TriangleSource: c.mesh[name], move: move}, turbinePartColor(name)))
	}
	return out, nil
}

func turbineFanHero(ctx context.Context) (solidlens.Scene, error) { return turbineFanView(ctx, "hero") }

func turbineFanViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return turbineFanView(ctx, name) }
}

// turbineFanView builds the default fan, and the 608ZZ fan for "sizes", and draws one view.
func turbineFanView(ctx context.Context, name string) (solidlens.Scene, error) {
	c, err := buildTurbineMeshes(ctx, turbinefan.DefaultParams())
	if err != nil {
		return solidlens.Scene{}, err
	}
	var large *turbineMeshes
	if name == "sizes" {
		l, err := buildTurbineMeshes(ctx, turbinefan.Params608ZZ())
		if err != nil {
			return solidlens.Scene{}, err
		}
		large = &l
	}
	return c.view(name, large)
}

// view draws one named view. large is the 608ZZ fan of the "sizes" view.
func (c turbineMeshes) view(name string, large *turbineMeshes) (solidlens.Scene, error) {
	hubMid := r3.Vec{Z: turbinefan.HubMid}
	switch name {
	case "hero", "front", "back", "side", "exploded", "exploded-back":
		dirs := map[string]r3.Vec{
			"hero": {X: .55, Y: .35, Z: 1}, "front": {X: .08, Y: .08, Z: 1}, "back": {X: -.35, Y: -.25, Z: -1},
			"side": {X: 1, Y: -.15, Z: .1}, "exploded": {X: .5, Y: .5, Z: 1}, "exploded-back": {X: -.5, Y: .45, Z: -1},
		}
		var pull map[string]r3.Vec
		if strings.HasPrefix(name, "exploded") {
			pull = turbineExplode()
		}
		models, err := c.models(turbinePartOrder, r3.Vec{}, pull)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return bevelScene(fitCamera(turbineCamera(dirs[name], hubMid), models, 16.0/9, .06), models), nil
	case "sizes":
		// The 605ZZ hub and the 608ZZ hub, 40 mm apart along X, seen from behind.
		models, err := c.models([]string{"hub"}, r3.Vec{X: -20}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		more, err := large.models([]string{"hub"}, r3.Vec{X: 20}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, more...)
		return bevelScene(fitCamera(turbineCamera(r3.Vec{X: -.2, Y: -.35, Z: -1}, hubMid), models, 16.0/9, .08), models), nil
	case "clamp":
		// Every body cut by the plane z = 10 through the clamp bolt's line, the half toward the front removed, seen from
		// the front: the hub's bore and slit, the sockets with the studs in them and the blade bolts.
		models, err := c.models(turbinePartOrder, r3.Vec{}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		for i, part := range turbinePartOrder {
			models[i] = clipped(models[i], turbinePartColor(part), solidlens.Vec{Z: turbinefan.HubMid}, solidlens.Vec{Z: 1})
		}
		lo := r3.Vec{X: -turbinefan.HubRadius - 10, Y: -c.dm.Reach - 10, Z: -1}
		hi := r3.Vec{X: turbinefan.HubRadius + 10, Y: turbinefan.HubRadius + 10, Z: turbinefan.HubMid + 1}
		frame := []solidlens.Model{{Mesh: boxCorners{lo, hi}}}
		return bevelScene(fitCamera(turbineCamera(r3.Vec{X: .04, Y: .06, Z: 1}, hubMid), frame, 16.0/9, .04), models), nil
	case "joint":
		// The hub, blade 1, blade bolt 1 with its nut, and the clamp bolt and nut, cut by the plane x = 0 through the
		// shaft axis and blade 1's radial line, the half toward +X removed, seen from +X.
		parts := []string{"hub", "blade-1", "bolt-1", "nut-1", "clamp-bolt", "clamp-nut"}
		models, err := c.models(parts, r3.Vec{}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		for i, part := range parts {
			models[i] = clipped(models[i], turbinePartColor(part), solidlens.Vec{}, solidlens.Vec{X: 1})
		}
		lo, hi := r3.Vec{X: -1, Y: -25, Z: -2}, r3.Vec{X: 1, Y: 30, Z: 22}
		frame := []solidlens.Model{{Mesh: boxCorners{lo, hi}}}
		return bevelScene(fitCamera(turbineCamera(r3.Vec{X: 1, Y: .01, Z: .02}, r3.Vec{Y: 2.5, Z: 10}), frame, 16.0/9, .05), models), nil
	}
	return solidlens.Scene{}, fmt.Errorf("unknown turbine fan view %q", name)
}

// renderTurbineFanReadme builds the default fan once, and the 608ZZ fan once for the size comparison, and writes every
// README image.
func renderTurbineFanReadme(ctx context.Context, outputDir string) error {
	c, err := buildTurbineMeshes(ctx, turbinefan.DefaultParams())
	if err != nil {
		return err
	}
	large, err := buildTurbineMeshes(ctx, turbinefan.Params608ZZ())
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270}, {"front", "front.png", 1600, 900},
		{"back", "back.png", 1600, 900}, {"side", "side.png", 1600, 900}, {"clamp", "clamp.png", 1600, 900},
		{"joint", "joint.png", 1600, 900}, {"exploded", "exploded.png", 1600, 900}, {"exploded-back", "exploded-back.png", 1600, 900}, {"sizes", "sizes.png", 1600, 900},
	} {
		var with *turbineMeshes
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
	// One image per distinct part: the three blades are alike, and so are the four bolts and the four nuts.
	for _, name := range []string{"hub", "blade-1", "bolt-1", "nut-1"} {
		mesh := c.mesh[name]
		low, high := meshBounds(mesh)
		center := low.Add(high).Scale(.5)
		span := high.Sub(low).Len()
		camera := solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: .8 * span})),
			Target: solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
		if name == "hub" {
			// Seen from the front, the face with the three blade bolts, with blade 1 up.
			camera = solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: .5 * span, Y: .2 * span, Z: 1.2 * span})),
				Target: solidlens.Vec(center), Up: solidlens.Vec{Y: 1}, FOV: 35}
		}
		models := []solidlens.Model{turbineModel(mesh, turbinePartColor(name))}
		scene := bevelScene(fitCamera(camera, models, 16.0/9, .1), models)
		file := strings.TrimSuffix(name, "-1")
		if err := renderPNG(ctx, filepath.Join(outputDir, "parts", file+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}
