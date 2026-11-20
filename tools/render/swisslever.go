package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// swissMargin is the free border of every framed Swiss lever view, as a fraction of the image.
const swissMargin = 0.06

// buildSwissMovement builds the movement. It is a variable so that a local scratch run can swap in a build that leaves
// out an operation decad does not accept yet; the committed code always uses the real Build.
var buildSwissMovement = swisslever.Build

// swissColor colours a body by the group it belongs to.
func swissColor(name string) solidlens.Color {
	switch {
	case strings.HasSuffix(name, "Plate"):
		return solidlens.RGB(0.38, 0.45, 0.52)
	case strings.HasSuffix(name, "Bearing"):
		return solidlens.RGB(0.72, 0.74, 0.78)
	case strings.HasSuffix(name, "Arbor"), name == swisslever.BalanceStaffName:
		return solidlens.RGB(0.3, 0.32, 0.36)
	case strings.HasPrefix(name, "Bolt"), strings.HasPrefix(name, "Nut"):
		return solidlens.RGB(0.55, 0.57, 0.6)
	case name == swisslever.EscapeWheelName:
		return solidlens.RGB(0.78, 0.5, 0.18)
	case name == swisslever.LeverName:
		return solidlens.RGB(0.7, 0.28, 0.2)
	case name == swisslever.EntryStoneName, name == swisslever.ExitStoneName:
		return solidlens.RGB(0.85, 0.12, 0.2)
	case name == swisslever.BalanceWheelName:
		return solidlens.RGB(0.17, 0.5, 0.55)
	case name == swisslever.SafetyRollerName, name == swisslever.RollerTableName:
		return solidlens.RGB(0.35, 0.68, 0.72)
	case name == swisslever.HairspringName:
		return solidlens.RGB(0.62, 0.38, 0.62)
	case name == swisslever.HairspringRetainerName, name == swisslever.WindingKnobName,
		name == swisslever.KnobCollarName:
		return solidlens.RGB(0.45, 0.58, 0.3)
	}
	return solidlens.RGB(0.65, 0.47, 0.2)
}

// swissBuild is the default movement with every body tessellated once.
type swissBuild struct {
	movement *swisslever.Movement
	geometry swisslever.Geometry
	named    []swisslever.NamedBody
	meshes   map[string]*decad.Mesh
}

func buildSwiss(ctx context.Context) (*swissBuild, error) {
	p := swisslever.DefaultParams()
	m, err := buildSwissMovement(ctx, swisslever.NewDocument(), p)
	if err != nil {
		return nil, fmt.Errorf("build swiss lever escapement: %w", err)
	}
	g, err := swisslever.Layout(p)
	if err != nil {
		return nil, err
	}
	b := &swissBuild{movement: m, geometry: g, named: m.Named(), meshes: map[string]*decad.Mesh{}}
	for _, nb := range b.named {
		mesh, err := nb.Body.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate %s: %w", nb.Name, err)
		}
		b.meshes[nb.Name] = mesh
	}
	return b, nil
}

// swissModel returns the model of one body, moved by move.
func (b *swissBuild) swissModel(name string, move r3.Transform) solidlens.Model {
	return solidlens.Model{
		Mesh:     rotatedMesh{TriangleSource: b.meshes[name], move: move},
		Material: solidlens.Matte(swissColor(name)),
		Edges:    solidlens.Edges{Enabled: true, Color: solidlens.RGB(0.025, 0.07, 0.11), Width: 1.2},
	}
}

// models returns the models of the bodies f keeps, at rest or moved by the exploded offsets.
func (b *swissBuild) models(f bodyFilter, offsets map[string]r3.Vec) ([]solidlens.Model, error) {
	var out []solidlens.Model
	for _, nb := range b.named {
		if !f(nb.Name) {
			continue
		}
		move := r3.Identity()
		if offset, ok := offsets[nb.Name]; ok {
			t, err := r3.Translation(offset)
			if err != nil {
				return nil, fmt.Errorf("offset of %s: %w", nb.Name, err)
			}
			move = t
		}
		out = append(out, b.swissModel(nb.Name, move))
	}
	return out, nil
}

// swissFrontParts are the bodies in front of the movement that a view of its mechanism leaves out.
var swissFrontParts = func() []string {
	out := []string{swisslever.FrontPlateName, swisslever.FrontEscapeBearingName, swisslever.FrontPalletBearingName,
		swisslever.FrontBalanceBearingName}
	for i := range 4 {
		out = append(out, swisslever.BoltName(i))
	}
	return out
}()

// swissCamera looks at the models' centre from direction dir (the way the camera lies from the target), with +Y up,
// and frames the models with the shared margin.
func swissCamera(models []solidlens.Model, dir r3.Vec, aspect float64) solidlens.Camera {
	low, high := modelBounds(models)
	centre := low.Add(high).Scale(0.5)
	u, _ := dir.Normalize()
	camera := solidlens.Camera{
		Position: solidlens.Vec(centre.Add(u.Scale(1200))), Target: solidlens.Vec(centre),
		Up: solidlens.Vec{Y: 1}, FOV: 30, Near: 1, Far: 6000,
	}
	return fitCamera(camera, models, aspect, swissMargin)
}

var (
	swissHeroDir  = r3.NewVec(0.3, 0.25, 1)
	swissFrontDir = r3.NewVec(0, 0, 1)
	swissAngleDir = r3.NewVec(1, 0.6, 1)
	swissSideDir  = r3.NewVec(1, 0, 0)
	swissBackDir  = r3.NewVec(0, 0, -1)
)

// heroScene shows the whole movement at the default pose from the front and a little above: the plates' windows let
// the mechanism show through.
func (b *swissBuild) heroScene() (solidlens.Scene, error) { return b.viewScene(swissHeroDir) }

func (b *swissBuild) viewScene(dir r3.Vec) (solidlens.Scene, error) {
	models, err := b.models(everything, nil)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(swissCamera(models, dir, 16.0/9), models), nil
}

// backScene looks at the movement from behind. The shared lights shine toward -Z, so they leave the back faces dark;
// this scene turns them to shine toward +Z. The frame is square because the movement is taller than it is wide.
func (b *swissBuild) backScene() (solidlens.Scene, error) {
	models, err := b.models(everything, nil)
	if err != nil {
		return solidlens.Scene{}, err
	}
	scene := bevelScene(swissCamera(models, swissBackDir, 1), models)
	for i := range scene.DirectionalLights {
		scene.DirectionalLights[i].Direction.Z = -scene.DirectionalLights[i].Direction.Z
	}
	return scene, nil
}

// swissCloseupHalfHeight is half the height, in mm, of the close-up's view.
const swissCloseupHalfHeight = 40.0

// closeupScene is a near-orthographic view from behind of the pallets, the fork, the dart, the safety roller and the
// impulse pin at the default Phase 0.25, where the pin lies on the line of centres. The pin stands out of the rear face
// of the roller table into the fork, so only a view from the rear shows it. The view leaves out what would hide the
// lever or float in the air without its seat: the back plate, its bearings and nuts, the front plate and its bearings,
// the bolts, the balance wheel, the hairspring and its retainer, the knob and its collar.
func (b *swissBuild) closeupScene() (solidlens.Scene, error) {
	hide := append(append([]string{}, swissFrontParts...), swisslever.BackPlateName,
		swisslever.BackEscapeBearingName, swisslever.BackPalletBearingName, swisslever.BackBalanceBearingName,
		swisslever.BalanceWheelName, swisslever.HairspringName, swisslever.HairspringRetainerName,
		swisslever.WindingKnobName, swisslever.KnobCollarName)
	for i := range 4 {
		hide = append(hide, swisslever.NutName(i))
	}
	models, err := b.models(without(hide...), nil)
	if err != nil {
		return solidlens.Scene{}, err
	}
	const distance = 800.0
	g := b.geometry
	centre := r3.NewVec(0, (g.P.Y+g.B.Y)/2, 8)
	camera := solidlens.Camera{
		Position: solidlens.Vec(centre.Add(r3.NewVec(0, 0, -distance))), Target: solidlens.Vec(centre),
		Up: solidlens.Vec{Y: 1}, FOV: 2 * math.Atan(swissCloseupHalfHeight/distance) * 180 / math.Pi, Near: 1, Far: 6000,
	}
	scene := bevelScene(camera, models)
	// The shared lights shine toward -Z and leave the rear faces dark, so turn them to shine toward +Z.
	for i := range scene.DirectionalLights {
		scene.DirectionalLights[i].Direction.Z = -scene.DirectionalLights[i].Direction.Z
	}
	return scene, nil
}

// swissPart names one image of a distinct body.
type swissPart struct {
	shot   string
	bodies []string
	dir    r3.Vec
}

// swissParts lists the images of the README's parts table: one per distinct body. The six bearings, the four bolts
// and the four nuts are identical, and the Pallet Arbor and the Balance Staff are the same part built twice.
var swissParts = []swissPart{
	{"back-plate", []string{swisslever.BackPlateName}, r3.NewVec(0.5, 0.4, -1)},
	{"front-plate", []string{swisslever.FrontPlateName}, r3.NewVec(0.5, 0.4, 1)},
	{"bearing", []string{swisslever.BackEscapeBearingName}, r3.NewVec(0.5, 0.5, 1)},
	{"escape-arbor", []string{swisslever.EscapeArborName}, r3.NewVec(1, 0.3, 0.5)},
	{"pallet-arbor", []string{swisslever.PalletArborName}, r3.NewVec(1, 0.3, 0.5)},
	{"escape-wheel", []string{swisslever.EscapeWheelName}, r3.NewVec(0.5, 0.4, 1)},
	{"lever", []string{swisslever.LeverName}, r3.NewVec(0.5, 0.4, 1)},
	{"entry-stone", []string{swisslever.EntryStoneName}, r3.NewVec(0.6, 0.5, 1)},
	{"exit-stone", []string{swisslever.ExitStoneName}, r3.NewVec(0.6, 0.5, 1)},
	{"safety-roller", []string{swisslever.SafetyRollerName}, r3.NewVec(0.5, 0.4, 1)},
	{"roller-table", []string{swisslever.RollerTableName}, r3.NewVec(0.5, 0.4, 1)},
	{"balance-wheel", []string{swisslever.BalanceWheelName}, r3.NewVec(0.5, 0.4, 1)},
	{"hairspring", []string{swisslever.HairspringName}, r3.NewVec(0.4, 0.5, 1)},
	{"hairspring-retainer", []string{swisslever.HairspringRetainerName}, r3.NewVec(0.5, 0.4, 1)},
	{"winding-knob", []string{swisslever.WindingKnobName}, r3.NewVec(0.5, 0.4, 1)},
	{"knob-collar", []string{swisslever.KnobCollarName}, r3.NewVec(0.5, 0.4, 1)},
	{"bolt", []string{swisslever.BoltName(0)}, r3.NewVec(1, 0.3, 0.5)},
	{"nut", []string{swisslever.NutName(0)}, r3.NewVec(0.5, 0.5, 1)},
}

func (b *swissBuild) partScene(part swissPart) (solidlens.Scene, error) {
	keep := map[string]struct{}{}
	for _, n := range part.bodies {
		keep[n] = struct{}{}
	}
	models, err := b.models(func(name string) bool { _, ok := keep[name]; return ok }, nil)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelScene(swissCamera(models, part.dir, 16.0/9), models), nil
}

// swissSceneOf adapts a scene method of a fresh build to a preset.
func swissSceneOf(f func(*swissBuild) (solidlens.Scene, error)) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) {
		b, err := buildSwiss(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return f(b)
	}
}

func addSwissPresets(presets map[string]renderkit.Preset) {
	view := func(dir r3.Vec) func(*swissBuild) (solidlens.Scene, error) {
		return func(b *swissBuild) (solidlens.Scene, error) { return b.viewScene(dir) }
	}
	presets["hero"] = renderkit.Preset{Width: 1600, Height: 900, Scene: swissSceneOf((*swissBuild).heroScene)}
	presets["index"] = renderkit.Preset{Width: 480, Height: 270, Scene: swissSceneOf((*swissBuild).heroScene)}
	presets["assembled-front"] = renderkit.Preset{Width: 1200, Height: 675, Scene: swissSceneOf(view(swissFrontDir))}
	presets["assembled-angle"] = renderkit.Preset{Width: 1200, Height: 675, Scene: swissSceneOf(view(swissAngleDir))}
	presets["assembled-side"] = renderkit.Preset{Width: 1200, Height: 675, Scene: swissSceneOf(view(swissSideDir))}
	presets["assembled-back"] = renderkit.Preset{Width: 1000, Height: 1000, Scene: swissSceneOf((*swissBuild).backScene)}
	presets["closeup"] = renderkit.Preset{Width: 1200, Height: 675, Scene: swissSceneOf((*swissBuild).closeupScene)}
	presets["exploded"] = renderkit.Preset{Width: swissExplodedWidth, Height: swissExplodedHeight,
		Scene: swissSceneOf((*swissBuild).explodedScene)}
	for _, part := range swissParts {
		presets["part-"+part.shot] = renderkit.Preset{Width: 480, Height: 270,
			Scene: swissSceneOf(func(b *swissBuild) (solidlens.Scene, error) { return b.partScene(part) })}
	}
}

// renderSwissReadme writes every still of the model's README from one build.
func renderSwissReadme(ctx context.Context, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("-out is required")
	}
	b, err := buildSwiss(ctx)
	if err != nil {
		return err
	}
	type image struct {
		name          string
		width, height int
		scene         func() (solidlens.Scene, error)
	}
	images := []image{
		{"hero.png", 1600, 900, b.heroScene},
		{"index.png", 480, 270, b.heroScene},
		{"assembled-front.png", 1200, 675, func() (solidlens.Scene, error) { return b.viewScene(swissFrontDir) }},
		{"assembled-angle.png", 1200, 675, func() (solidlens.Scene, error) { return b.viewScene(swissAngleDir) }},
		{"assembled-side.png", 1200, 675, func() (solidlens.Scene, error) { return b.viewScene(swissSideDir) }},
		{"assembled-back.png", 1000, 1000, b.backScene},
		{"closeup.png", 1200, 675, b.closeupScene},
		{"exploded.png", swissExplodedWidth, swissExplodedHeight, b.explodedScene},
	}
	for _, part := range swissParts {
		images = append(images, image{filepath.Join("parts", part.shot+".png"), 480, 270,
			func() (solidlens.Scene, error) { return b.partScene(part) }})
	}
	for _, img := range images {
		scene, err := img.scene()
		if err != nil {
			return fmt.Errorf("scene of swisslever %s: %w", img.name, err)
		}
		if err := renderkit.RenderPNG(ctx, filepath.Join(outputDir, img.name), scene, img.width, img.height); err != nil {
			return fmt.Errorf("render swisslever %s: %w", img.name, err)
		}
	}
	return nil
}
