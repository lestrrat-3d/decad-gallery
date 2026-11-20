package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The fan stands with its shaft axis along +Z, the hub's back face at z = 0 and the front face at z = Size.HubLength, blade 1 along
// +Y and the set screw toward -Y. Every camera has +Y up, so blade 1 points up in every view.
var (
	turbineHubColor      = solidlens.RGB(.16, .52, .70) // the hand-crank fan's rotor colour
	turbineBladeColor    = solidlens.RGB(.30, .68, .85) // the hand-crank fan's blade colour
	turbineHardwareColor = solidlens.RGB(.55, .57, .60)
	turbineHarnessColor  = solidlens.RGB(.80, .80, .78)
)

// turbineDistance is the camera's distance from its target, in mm.
const turbineDistance = 380

// turbineCamera looks at target from the direction dir, with +Y up.
func turbineCamera(dir, target r3.Vec) solidlens.Camera {
	d, _ := dir.Normalize()
	return solidlens.Camera{Position: solidlens.Vec(target.Add(d.Scale(turbineDistance))), Target: solidlens.Vec(target),
		Up: solidlens.Vec{Y: 1}, FOV: 35}
}

// turbineScene lights models like renderkit.Scene, but turns the lights to shine toward +Z when the camera is behind the
// fan (z below its target). The shared lights shine toward -Z, so they leave the back faces dark from there.
func turbineScene(camera solidlens.Camera, models []solidlens.Model) solidlens.Scene {
	scene := renderkit.Scene(camera, models)
	if camera.Position.Z >= camera.Target.Z {
		return scene
	}
	for i := range scene.DirectionalLights {
		scene.DirectionalLights[i].Direction.Z = -scene.DirectionalLights[i].Direction.Z
	}
	return scene
}

// turbineExplode is each part's offset in the exploded view: every blade outward along its radial line, each blade bolt out
// of the front face, each blade nut out of its slot along the radial line, the set screw out along -Y and its nut out of
// the back face. The blades move 35 s, the blade bolts BladeBoltLength + 14, the blade nuts 18 j and the set screw 32 j,
// for the blade scale s and the hub scale j of sz; the set screw's nut moves 14 (30, 30, 18 and 32 mm at 128).
func turbineExplode(sz turbinefan.Size) map[string]r3.Vec {
	j := sz.HubScale
	out := map[string]r3.Vec{"hub": {}, "setscrew-bolt": {Y: -32 * j}, "setscrew-nut": {Z: -14}}
	for k := 1; k <= turbinefan.BladeCount; k++ {
		a := turbinefan.BladeAngle(k)
		u := r3.Vec{X: math.Cos(a), Y: math.Sin(a)}
		out[fmt.Sprintf("blade-%d", k)] = u.Scale(35 * sz.Scale)
		out[fmt.Sprintf("bolt-%d", k)] = r3.Vec{Z: sz.BladeBoltLength + 14}
		out[fmt.Sprintf("nut-%d", k)] = u.Scale(18 * j)
	}
	return out
}

// turbineMeshes is one built fan's meshes by part name.
type turbineMeshes struct {
	mesh map[string]*decad.Mesh
	dm   turbinefan.Dimensions
}

var turbinePartOrder = []string{
	"hub", "blade-1", "blade-2", "blade-3", "bolt-1", "nut-1", "bolt-2", "nut-2", "bolt-3", "nut-3", "setscrew-bolt", "setscrew-nut",
}

// turbineClutchPartOrder is the clutch-mount fan's parts: no set screw, and the two adapter bolts and nuts. The adapter
// and the ear bolts belong to the clutch's set.
var turbineClutchPartOrder = []string{
	"hub", "blade-1", "blade-2", "blade-3", "bolt-1", "nut-1", "bolt-2", "nut-2", "bolt-3", "nut-3",
	"adapterbolt-1", "adapternut-1", "adapterbolt-2", "adapternut-2",
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
	bodies := map[string]*decad.Body{"hub": m.Hub}
	if m.SetScrew != nil {
		bodies["setscrew-bolt"], bodies["setscrew-nut"] = m.SetScrew.Body, m.SetScrewNut.Body
	}
	for k := range m.AdapterBolts {
		if m.AdapterBolts[k] == nil {
			continue
		}
		bodies[fmt.Sprintf("adapterbolt-%d", k+1)] = m.AdapterBolts[k].Body
		bodies[fmt.Sprintf("adapternut-%d", k+1)] = m.AdapterNuts[k].Body
	}
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
		mesh, err := body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyNone))
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
	sz := c.dm.Size
	hubMid := r3.Vec{Z: sz.HubMid}
	switch name {
	case "hero", "front", "back", "side", "exploded", "exploded-back":
		dirs := map[string]r3.Vec{
			"hero": {X: .55, Y: .35, Z: 1}, "front": {X: .08, Y: .08, Z: 1}, "back": {X: -.35, Y: -.25, Z: -1},
			"side": {X: 1, Y: -.55, Z: .15}, "exploded": {X: .5, Y: .5, Z: 1}, "exploded-back": {X: -.5, Y: .45, Z: -1},
		}
		var pull map[string]r3.Vec
		if strings.HasPrefix(name, "exploded") {
			pull = turbineExplode(c.dm.Size)
		}
		models, err := c.models(turbinePartOrder, r3.Vec{}, pull)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return turbineScene(renderkit.FitCamera(turbineCamera(dirs[name], hubMid), models, 16.0/9, .06), models), nil
	case "sizes":
		// The 605ZZ hub and the 608ZZ hub, 2 HubRadius + 12 apart along X (49.8 mm at 192), seen from behind.
		half := sz.HubRadius + 6
		models, err := c.models([]string{"hub"}, r3.Vec{X: -half}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		more, err := large.models([]string{"hub"}, r3.Vec{X: half}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, more...)
		return turbineScene(renderkit.FitCamera(turbineCamera(r3.Vec{X: -.2, Y: -.35, Z: -1}, hubMid), models, 16.0/9, .08), models), nil
	case "setscrew":
		// The hub, the blade bolts and nuts, and the set screw and its nut, cut by the plane z = SetScrewZ through the set
		// screw's line, the half toward the front removed, seen from the front: the D bore, the screw's counterbore, hole
		// and nut slot, the sockets with the studs in them, and the blade bolts.
		parts := []string{"hub", "bolt-1", "bolt-2", "bolt-3", "nut-1", "nut-2", "nut-3", "setscrew-bolt", "setscrew-nut"}
		models, err := c.models(parts, r3.Vec{}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		for i, part := range parts {
			models[i] = clipped(models[i], turbinePartColor(part), solidlens.Vec{Z: turbinefan.SetScrewZ}, solidlens.Vec{Z: 1})
		}
		box := sz.HubRadius + 6
		lo := r3.Vec{X: -box, Y: -box, Z: -1}
		hi := r3.Vec{X: box, Y: box, Z: turbinefan.SetScrewZ + 1}
		frame := []solidlens.Model{{Mesh: boxCorners{lo, hi}}}
		target := r3.Vec{Z: turbinefan.SetScrewZ}
		return turbineScene(renderkit.FitCamera(turbineCamera(r3.Vec{X: .04, Y: .06, Z: 1}, target), frame, 16.0/9, .04), models), nil
	case "joint":
		// The hub, blade 1, blade bolt 1 with its nut, and the set screw and its nut, cut by the plane x = 0 through the
		// shaft axis and blade 1's radial line, the half toward +X removed, seen from +X.
		parts := []string{"hub", "blade-1", "bolt-1", "nut-1", "setscrew-bolt", "setscrew-nut"}
		models, err := c.models(parts, r3.Vec{}, nil)
		if err != nil {
			return solidlens.Scene{}, err
		}
		for i, part := range parts {
			models[i] = clipped(models[i], turbinePartColor(part), solidlens.Vec{}, solidlens.Vec{X: 1})
		}
		lo := r3.Vec{X: -1, Y: -(sz.HubRadius + 11), Z: -2}
		hi := r3.Vec{X: 1, Y: sz.HubRadius + 16, Z: sz.HubLength + 2}
		frame := []solidlens.Model{{Mesh: boxCorners{lo, hi}}}
		target := r3.Vec{Y: 2.5 * sz.HubScale, Z: sz.HubMid}
		return turbineScene(renderkit.FitCamera(turbineCamera(r3.Vec{X: 1, Y: .01, Z: .02}, target), frame, 16.0/9, .05), models), nil
	}
	return solidlens.Scene{}, fmt.Errorf("unknown turbine fan view %q", name)
}

// turbinePrint is one blade and its print harness, meshed, in the print frame: bed z = 0, +Z up, +X from stud to tip.
type turbinePrint struct {
	blade, harness *decad.Mesh
}

// buildTurbinePrintMeshes builds the blade and harness of p with BuildBladePrint and tessellates the blade at 0.05 mm
// and the harness at 0.2 mm.
func buildTurbinePrintMeshes(ctx context.Context, p turbinefan.Params) (turbinePrint, error) {
	bp, err := turbinefan.BuildBladePrint(ctx, turbinefan.NewDocument(), p)
	if err != nil {
		return turbinePrint{}, fmt.Errorf("build turbine blade print: %w", err)
	}
	var out turbinePrint
	// The cradle's cuts cross two faceted tools, so its held mesh bound is 0.09 to 0.14; it meshes at 0.2.
	for _, part := range []struct {
		body *decad.Body
		mesh **decad.Mesh
		tol  float64
	}{{bp.Blade, &out.blade, .05}, {bp.Harness.Body, &out.harness, .2}} {
		mesh, err := part.body.Tessellate(ctx, units.Millimeters(part.tol), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return turbinePrint{}, fmt.Errorf("mesh turbine blade print: %w", err)
		}
		*part.mesh = mesh
	}
	return out, nil
}

func turbinePrintViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		p, err := buildTurbinePrintMeshes(ctx, turbinefan.DefaultParams())
		if err != nil {
			return solidlens.Scene{}, err
		}
		return p.view(name)
	}
}

// view draws the print job toward the bodies' bounds centre, +Z up: "print" from the stud end, the pressure side and
// above; "print-side" from the right hand's suction side (+Y), a little from above, so the tip is on the left and the
// stud on the right; "print-end" from the
// tip end along the radial line, a little from above.
func (p turbinePrint) view(name string) (solidlens.Scene, error) {
	dirs := map[string]r3.Vec{
		"print": {X: -.6, Y: -1, Z: .55}, "print-side": {X: 0, Y: 1, Z: .2}, "print-end": {X: 1, Y: -.05, Z: .25},
	}
	dir, ok := dirs[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown turbine print view %q", name)
	}
	models := []solidlens.Model{turbineModel(p.blade, turbineBladeColor), turbineModel(p.harness, turbineHarnessColor)}
	lo, hi := meshBounds(p.harness)
	blo, bhi := meshBounds(p.blade)
	lo = r3.Vec{X: math.Min(lo.X, blo.X), Y: math.Min(lo.Y, blo.Y), Z: math.Min(lo.Z, blo.Z)}
	hi = r3.Vec{X: math.Max(hi.X, bhi.X), Y: math.Max(hi.Y, bhi.Y), Z: math.Max(hi.Z, bhi.Z)}
	center := lo.Add(hi).Scale(.5)
	d, _ := dir.Normalize()
	camera := solidlens.Camera{Position: solidlens.Vec(center.Add(d.Scale(4 * hi.Sub(lo).Len()))),
		Target: solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
	return renderkit.Scene(renderkit.FitCamera(camera, models, 16.0/9, .06), models), nil
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
		{"back", "back.png", 1600, 900}, {"side", "side.png", 1600, 900}, {"setscrew", "setscrew.png", 1600, 900},
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
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	// One image per distinct part: the three blades are alike, and so are the three blade bolts and the four nuts.
	for _, item := range [][2]string{{"hub", "hub"}, {"blade-1", "blade"}, {"bolt-1", "bolt"}, {"setscrew-bolt", "setscrew"}, {"nut-1", "nut"}} {
		name := item[0]
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
		scene := turbineScene(renderkit.FitCamera(camera, models, 16.0/9, .1), models)
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "parts", item[1]+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return renderTurbineClutchReadme(ctx, outputDir)
}

// The clutch-mount views draw the real clutch with its adapter and the fan placed on it, both in the clutch frame: the
// shaft axis along +Z, the clutch's back plate at z = 0, the adapter against the ring's front end and the hub's back face
// one seam in front of the adapter.
// clutchViewTarget returns the stack's middle along Z, for the cameras: 12 at 128 and 17 at 192.
func clutchViewTarget(sz turbinefan.Size) float64 { return 12 + (sz.HubLength-20)/2 }

// clutchHubLead is how far ahead of the clutch frame the hub moves in the exploded view: 40 s for the blade scale s.
func clutchHubLead(sz turbinefan.Size) float64 { return 40 * sz.Scale }

// clutchMountExplode is each fan part's offset in the exploded view, in the order the parts go together along +Z: the
// adapter 20 mm ahead of the clutch, the hub 40 s ahead (each blade 35 s and each blade nut 18 j out along its radial
// line), each blade bolt 40 s + BladeBoltLength + 14 ahead, each adapter bolt 6 mm ahead, which parks it between the
// clutch and the adapter, and each adapter nut 40 s ahead and 18 mm out along its line. The scales s and j are sz's.
func clutchMountExplode(sz turbinefan.Size) map[string]r3.Vec {
	lead := clutchHubLead(sz)
	out := map[string]r3.Vec{"hub": {Z: lead}}
	for k := 1; k <= turbinefan.BladeCount; k++ {
		a := turbinefan.BladeAngle(k) + turbinefan.ClutchTurn
		u := r3.Vec{X: math.Cos(a), Y: math.Sin(a)}
		out[fmt.Sprintf("blade-%d", k)] = u.Scale(35 * sz.Scale).Add(r3.Vec{Z: lead})
		out[fmt.Sprintf("nut-%d", k)] = u.Scale(18 * sz.HubScale).Add(r3.Vec{Z: lead})
		out[fmt.Sprintf("bolt-%d", k)] = r3.Vec{Z: lead + sz.BladeBoltLength + 14}
	}
	for k := 1; k <= 2; k++ {
		a := onewayclutch.EarAngles[k] * math.Pi / 180
		out[fmt.Sprintf("adapterbolt-%d", k)] = r3.Vec{Z: 6}
		out[fmt.Sprintf("adapternut-%d", k)] = r3.Vec{X: 18 * math.Cos(a), Y: 18 * math.Sin(a), Z: lead}
	}
	return out
}

// clutchMountSetExplode is the offset of each part of the clutch set in the exploded view: the adapter 20 mm ahead, each ear
// nut 30 mm behind the ring and 14 mm out along its ear, each ear bolt max(110, the blade bolt offset + 40) ahead, so it
// clears the blade bolts. The clutch itself stays in place.
func clutchMountSetExplode(sz turbinefan.Size) map[string]r3.Vec {
	out := map[string]r3.Vec{"adapter": {Z: 20}}
	earBolt := math.Max(110, clutchHubLead(sz)+sz.BladeBoltLength+14+40)
	for k := 1; k <= onewayclutch.Ears; k++ {
		e := onewayclutch.EarAngles[k-1] * math.Pi / 180
		out[fmt.Sprintf("mount nut %d", k)] = r3.Vec{X: 14 * math.Cos(e), Y: 14 * math.Sin(e), Z: -30}
		out[fmt.Sprintf("mount bolt %d", k)] = r3.Vec{Z: earBolt}
	}
	return out
}

// buildClutchMount builds the default clutch with its adapter, and the fan matched to it, once each.
func buildClutchMount(ctx context.Context) (clutchSet, turbineMeshes, error) {
	cp := onewayclutch.DefaultParams()
	set, err := buildClutchAdapterParts(ctx, cp)
	if err != nil {
		return nil, turbineMeshes{}, err
	}
	fp, err := turbinefan.ParamsForClutch(cp)
	if err != nil {
		return nil, turbineMeshes{}, err
	}
	fan, err := buildTurbineMeshes(ctx, fp)
	if err != nil {
		return nil, turbineMeshes{}, err
	}
	return set, fan, nil
}

// clutchMountCut cuts by the plane through the shaft axis and the 330 degree direction of the clutch frame, which holds
// ear 3, adapter bolt 2 (the fan's 150 degree hole) and the clutch's 150 degree bolt, and removes the half toward its
// normal (60 degrees).
var clutchMountCut = solidlens.ClipPlane{Normal: solidlens.Vec{X: math.Cos(math.Pi / 3), Y: math.Sin(math.Pi / 3)}}

// clutchMountView draws one clutch-mount view: "clutch", "clutch-back", "clutch-cutaway" or "clutch-exploded".
func clutchMountView(name string, set clutchSet, fan turbineMeshes) (solidlens.Scene, error) {
	var clip *solidlens.ClipPlane
	var pull, clutchPull map[string]r3.Vec
	sz := fan.dm.Size
	target := r3.Vec{Z: clutchViewTarget(sz)}
	var camera solidlens.Camera
	switch name {
	case "clutch":
		camera = turbineCamera(r3.Vec{X: .55, Y: .35, Z: 1}, target)
	case "clutch-back":
		camera = turbineCamera(r3.Vec{X: -.45, Y: -.3, Z: -1}, target)
	case "clutch-cutaway":
		clip = &clutchMountCut
		// Look from the removed side along the normal n, shifted 20 mm along -t, the plane's other horizontal direction.
		n, t := r3.Vec{X: clutchMountCut.Normal.X, Y: clutchMountCut.Normal.Y}, r3.Vec{X: -clutchMountCut.Normal.Y, Y: clutchMountCut.Normal.X}
		camera = solidlens.Camera{Position: solidlens.Vec(target.Add(n.Scale(turbineDistance)).Add(t.Scale(-20)).Add(r3.Vec{Z: 20})),
			Target: solidlens.Vec(target), Up: solidlens.Vec{Z: 1}, FOV: 35}
	case "clutch-exploded":
		pull, clutchPull = clutchMountExplode(sz), clutchMountSetExplode(sz)
		camera = turbineCamera(r3.Vec{X: 1, Y: .25, Z: .45}, r3.Vec{Z: 40 * sz.Scale})
	default:
		return solidlens.Scene{}, fmt.Errorf("unknown clutch-mount view %q", name)
	}
	// The clutch moves as one unit; only the adapter, its bolts and its nuts take part in the pull.
	models, err := set.models(r3.Vec{}, clutchPull, clip)
	if err != nil {
		return solidlens.Scene{}, err
	}
	names := turbineClutchPartOrder
	if clip != nil {
		// A part wholly on the removed side is left out.
		names = nil
		for _, part := range turbineClutchPartOrder {
			if !clipRemovesAll(fan.mesh[part], *clip) {
				names = append(names, part)
			}
		}
	}
	more, err := fan.models(names, r3.Vec{}, pull)
	if err != nil {
		return solidlens.Scene{}, err
	}
	if clip != nil {
		for i, part := range names {
			more[i] = clipped(more[i], turbinePartColor(part), solidlens.Vec{}, clutchMountCut.Normal)
		}
	}
	models = append(models, more...)
	return turbineScene(renderkit.FitCamera(camera, models, 16.0/9, .06), models), nil
}

// turbineClutchViewScene returns the preset for a clutch-mount view.
func turbineClutchViewScene(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		set, fan, err := buildClutchMount(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return clutchMountView(name, set, fan)
	}
}

// renderTurbineClutchReadme writes the four clutch-mount images and the images of the parts it adds.
func renderTurbineClutchReadme(ctx context.Context, outputDir string) error {
	set, fan, err := buildClutchMount(ctx)
	if err != nil {
		return err
	}
	for _, view := range []string{"clutch", "clutch-back", "clutch-cutaway", "clutch-exploded"} {
		scene, err := clutchMountView(view, set, fan)
		if err != nil {
			return err
		}
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, view+".png"), scene, 1600, 900); err != nil {
			return err
		}
	}
	// The ear bolt is the adapter's M2 x 25, drawn from the clutch's set.
	for _, item := range [][2]string{{"adapterbolt-1", "adapter-bolt"}, {"mount bolt 1", "ear-bolt"}} {
		name := item[0]
		mesh := fan.mesh[name]
		if mesh == nil {
			for _, part := range set {
				if part.name == name {
					mesh = part.mesh
				}
			}
		}
		low, high := meshBounds(mesh)
		center := low.Add(high).Scale(.5)
		span := high.Sub(low).Len()
		camera := solidlens.Camera{Position: solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: .8 * span})),
			Target: solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
		models := []solidlens.Model{turbineModel(mesh, turbineHardwareColor)}
		scene := turbineScene(renderkit.FitCamera(camera, models, 16.0/9, .1), models)
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, "parts", item[1]+".png"), scene, 480, 270); err != nil {
			return err
		}
	}
	return nil
}
