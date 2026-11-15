package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// The clutch lies with its shaft axis along +Z, the back plate at the bottom and the cover on top. The hero view cuts
// every body by the plane z = clutchMidPlane, the hub's mid-height, and removes the +Z side, so the pawl heads in the
// ring's teeth, the arms and the back plate show, with the ring's three ears outside them.
const clutchMidPlane = onewayclutch.HubFrom + onewayclutch.HubHeight/2

var (
	clutchHubColor   = solidlens.RGB(.70, .22, .12)
	clutchRingColor  = solidlens.RGB(.20, .22, .25)
	clutchPlateColor = solidlens.RGB(.30, .45, .62)
	clutchBoltColor  = solidlens.RGB(.55, .57, .60)
)

// clutchPart is one body's name, mesh and colour, in drawing order.
type clutchPart struct {
	name  string
	mesh  *decad.Mesh
	color solidlens.Color
}

// isHardware reports whether the part is a bought bolt or nut.
func (p clutchPart) isHardware() bool {
	return strings.HasPrefix(p.name, "bolt") || strings.HasPrefix(p.name, "nut")
}

// clutchSet is one built clutch's ten meshes.
type clutchSet []clutchPart

// buildClutchParts builds one clutch once and tessellates its ten bodies.
func buildClutchParts(ctx context.Context, p onewayclutch.Params) (clutchSet, error) {
	m, err := onewayclutch.Build(ctx, decad.New(), p)
	if err != nil {
		return nil, fmt.Errorf("build one-way clutch: %w", err)
	}
	parts := clutchSet{
		{"hub", nil, clutchHubColor}, {"ring", nil, clutchRingColor},
		{"back plate", nil, clutchPlateColor}, {"cover", nil, clutchPlateColor},
	}
	bodies := []*decad.Body{m.Hub, m.Ring, m.BackPlate, m.Cover}
	for i := range m.Bolts {
		parts = append(parts, clutchPart{fmt.Sprintf("bolt %d", i+1), nil, clutchBoltColor})
		bodies = append(bodies, m.Bolts[i].Body)
	}
	for i := range m.Nuts {
		parts = append(parts, clutchPart{fmt.Sprintf("nut %d", i+1), nil, clutchBoltColor})
		bodies = append(bodies, m.Nuts[i].Body)
	}
	for i := range parts {
		tol := .04
		if parts[i].isHardware() {
			tol = .01
		}
		mesh, err := bodies[i].Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return nil, fmt.Errorf("mesh clutch %s: %w", parts[i].name, err)
		}
		parts[i].mesh = mesh
	}
	return parts, nil
}

func clutchModel(mesh solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// clutchCut returns a clip plane whose cap is a lighter shade of the part's colour.
func clutchCut(plane solidlens.ClipPlane, c solidlens.Color) *solidlens.ClipPlane {
	plane.Cap = &solidlens.Material{Color: solidlens.RGB(c.R+(1-c.R)*.35, c.G+(1-c.G)*.35, c.B+(1-c.B)*.35),
		Unlit: true}
	return &plane
}

// models returns every part moved by at and by its entry of pull, cut by clip when it is not nil. A part wholly in the
// removed half is left out.
func (s clutchSet) models(at r3.Vec, pull map[string]r3.Vec, clip *solidlens.ClipPlane) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, part := range s {
		if clip != nil && clipRemovesAll(part.mesh, *clip) {
			continue
		}
		move, err := r3.Translation(at.Add(pull[part.name]))
		if err != nil {
			return nil, err
		}
		m := clutchModel(rotatedMesh{TriangleSource: part.mesh, move: move}, part.color)
		if clip != nil {
			m.Clip = clutchCut(*clip, part.color)
		}
		out = append(out, m)
	}
	return out, nil
}

// clipRemovesAll reports whether an axis-aligned clip plane leaves nothing of the mesh, because the whole mesh lies on
// the removed side.
func clipRemovesAll(mesh *decad.Mesh, clip solidlens.ClipPlane) bool {
	low, _ := meshBounds(mesh)
	switch {
	case clip.Normal.Z == 1:
		return low.Z >= clip.Origin.Z
	case clip.Normal.X == 1:
		return low.X >= clip.Origin.X
	}
	return false
}

// The exploded view pulls the parts apart along the shaft axis, in the order they go together: the nuts out of the back
// plate's hex pockets, the back plate, the hub, the ring, the cover, and the bolts, 30 mm apart.
var clutchExplode = map[string]r3.Vec{
	"nut 1": {Z: -30}, "nut 2": {Z: -30}, "nut 3": {Z: -30}, "back plate": {}, "hub": {Z: 30}, "ring": {Z: 60},
	"cover": {Z: 90}, "bolt 1": {Z: 120}, "bolt 2": {Z: 120}, "bolt 3": {Z: 120},
}

var (
	clutchMidCut = solidlens.ClipPlane{Origin: solidlens.Vec{Z: clutchMidPlane}, Normal: solidlens.Vec{Z: 1}}
	// clutchAxisCut cuts by the plane x = 0, which holds the shaft axis, and removes the half toward +X.
	clutchAxisCut = solidlens.ClipPlane{Normal: solidlens.Vec{X: 1}}
)

var clutchCameras = map[string]solidlens.Camera{
	"hero":    {Position: solidlens.Vec{X: 40, Y: -60, Z: 85}, Target: solidlens.Vec{Z: clutchMidPlane}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"front":   {Position: solidlens.Vec{X: 50, Y: -70, Z: 95}, Target: solidlens.Vec{Z: clutchMidPlane}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"back":    {Position: solidlens.Vec{X: -50, Y: -70, Z: -75}, Target: solidlens.Vec{Z: clutchMidPlane}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"side":    {Position: solidlens.Vec{X: 140, Y: -30, Z: 25}, Target: solidlens.Vec{Z: clutchMidPlane}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	"section": {Position: solidlens.Vec{Z: 220}, Target: solidlens.Vec{Z: clutchMidPlane}, Up: solidlens.Vec{Y: 1}, FOV: 30},
	"cutaway": {Position: solidlens.Vec{X: 150, Y: -25, Z: 30}, Target: solidlens.Vec{Z: clutchMidPlane}, Up: solidlens.Vec{Z: 1}, FOV: 35},
	// The shaft axis runs left to right on the screen, the back plate's side at the left, seen a little from below.
	"exploded": {Position: solidlens.Vec{X: -260, Y: 120, Z: -40}, Target: solidlens.Vec{Z: 55}, Up: solidlens.Vec{Y: 1},
		FOV: 35},
}

// view draws one named view of a single clutch.
func (s clutchSet) view(name string) (solidlens.Scene, error) {
	camera, ok := clutchCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown one-way clutch view %q", name)
	}
	var clip *solidlens.ClipPlane
	var pull map[string]r3.Vec
	switch name {
	case "hero", "section":
		clip = &clutchMidCut
	case "cutaway":
		clip = &clutchAxisCut
	case "exploded":
		pull = clutchExplode
	}
	models, err := s.models(r3.Vec{}, pull, clip)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .08), models), nil
}

// sideBySide draws the section of each clutch in the sets, the sets spaced 60 mm apart along +X, seen from above.
func sideBySide(sets []clutchSet) (solidlens.Scene, error) {
	var models []solidlens.Model
	for i, s := range sets {
		more, err := s.models(r3.Vec{X: float64(i) * 60}, nil, &clutchMidCut)
		if err != nil {
			return solidlens.Scene{}, err
		}
		models = append(models, more...)
	}
	mid := float64(len(sets)-1) * 30
	camera := solidlens.Camera{Position: solidlens.Vec{X: mid, Z: 240}, Target: solidlens.Vec{X: mid, Z: clutchMidPlane},
		Up: solidlens.Vec{Y: 1}, FOV: 30}
	return bevelScene(fitCamera(camera, models, 16.0/9, .06), models), nil
}

func oneWayClutchHero(ctx context.Context) (solidlens.Scene, error) {
	return oneWayClutchView("hero")(ctx)
}

// oneWayClutchView returns the preset for a single-clutch view of the default clutch.
func oneWayClutchView(name string) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		s, err := buildClutchParts(ctx, onewayclutch.DefaultParams())
		if err != nil {
			return solidlens.Scene{}, err
		}
		return s.view(name)
	}
}

// clutchDirectionParams are the default clutch and its mirror, for the directions image.
func clutchDirectionParams() []onewayclutch.Params {
	cw := onewayclutch.DefaultParams()
	ccw := cw
	ccw.Freewheel = onewayclutch.CounterClockwise
	return []onewayclutch.Params{cw, ccw}
}

// clutchSizeParams are the clutches of the three shaft presets, in table order.
func clutchSizeParams() []onewayclutch.Params {
	var out []onewayclutch.Params
	for _, shaft := range onewayclutch.Shafts() {
		p := onewayclutch.DefaultParams()
		p.Shaft = shaft.Name
		out = append(out, p)
	}
	return out
}

func buildClutchSets(ctx context.Context, params []onewayclutch.Params) ([]clutchSet, error) {
	var out []clutchSet
	for _, p := range params {
		s, err := buildClutchParts(ctx, p)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// oneWayClutchSideBySide returns the preset that draws the sections of the clutches built from params.
func oneWayClutchSideBySide(params func() []onewayclutch.Params) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		sets, err := buildClutchSets(ctx, params())
		if err != nil {
			return solidlens.Scene{}, err
		}
		return sideBySide(sets)
	}
}

// renderOneWayClutchReadme builds the default clutch once, its counter-clockwise mirror once and the other two shaft
// presets once, and writes every README image.
func renderOneWayClutchReadme(ctx context.Context, outputDir string) error {
	def, err := buildClutchParts(ctx, onewayclutch.DefaultParams())
	if err != nil {
		return err
	}
	for _, shot := range []struct {
		view, file    string
		width, height int
	}{
		{"hero", "hero.png", 1600, 900}, {"hero", "index.png", 480, 270},
		{"front", "front.png", 1600, 900}, {"back", "back.png", 1600, 900}, {"side", "side.png", 1600, 900},
		{"section", "section.png", 1600, 900}, {"cutaway", "cutaway.png", 1600, 900},
		{"exploded", "exploded.png", 1600, 900},
	} {
		scene, err := def.view(shot.view)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, shot.width, shot.height); err != nil {
			return err
		}
	}
	// The directions image holds the clockwise clutch already built; the sizes image holds the default 605ZZ clutch.
	ccw, err := buildClutchParts(ctx, clutchDirectionParams()[1])
	if err != nil {
		return err
	}
	var sizes []clutchSet
	for _, p := range clutchSizeParams() {
		if p.Shaft == onewayclutch.DefaultParams().Shaft {
			sizes = append(sizes, def)
			continue
		}
		s, err := buildClutchParts(ctx, p)
		if err != nil {
			return err
		}
		sizes = append(sizes, s)
	}
	for _, shot := range []struct {
		file string
		sets []clutchSet
	}{{"directions.png", []clutchSet{def, ccw}}, {"sizes.png", sizes}} {
		scene, err := sideBySide(shot.sets)
		if err != nil {
			return err
		}
		if err := renderPNG(ctx, filepath.Join(outputDir, shot.file), scene, 1600, 900); err != nil {
			return err
		}
	}
	// One image per distinct part: the three bolts are alike, and so are the three nuts.
	for _, name := range []string{"hub", "ring", "back plate", "cover", "bolt 1", "nut 1"} {
		for _, part := range def {
			if part.name != name {
				continue
			}
			low, high := meshBounds(part.mesh)
			center := low.Add(high).Scale(.5)
			span := high.Sub(low).Len()
			// The back plate is seen from below, so its three hex nut pockets show.
			lift := .8 * span
			if name == "back plate" {
				lift = -.8 * span
			}
			camera := solidlens.Camera{
				Position: solidlens.Vec(center.Add(r3.Vec{X: .9 * span, Y: -.9 * span, Z: lift})),
				Target:   solidlens.Vec(center), Up: solidlens.Vec{Z: 1}, FOV: 35}
			models := []solidlens.Model{clutchModel(part.mesh, part.color)}
			scene := bevelScene(fitCamera(camera, models, 16.0/9, .1), models)
			file := strings.TrimSuffix(strings.ReplaceAll(name, " ", "-"), "-1")
			if err := renderPNG(ctx, filepath.Join(outputDir, "parts", file+".png"), scene, 480, 270); err != nil {
				return err
			}
		}
	}
	return nil
}
