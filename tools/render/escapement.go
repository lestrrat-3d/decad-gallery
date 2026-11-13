package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// escapementMargin is the free border of every framed escapement view, as a fraction of the image.
const escapementMargin = 0.06

// escapementColor colours a body by the group it belongs to.
func escapementColor(name string) solidlens.Color {
	switch {
	case strings.HasSuffix(name, "Plate"):
		return solidlens.RGB(0.38, 0.45, 0.52)
	case strings.HasSuffix(name, "Bearing"):
		return solidlens.RGB(0.72, 0.74, 0.78)
	case strings.HasSuffix(name, "Arbor"), strings.HasSuffix(name, "Collar"), name == escapement.SuspensionPinName:
		return solidlens.RGB(0.3, 0.32, 0.36)
	case strings.HasPrefix(name, "Bolt"), strings.HasPrefix(name, "Nut"):
		return solidlens.RGB(0.55, 0.57, 0.6)
	case name == escapement.EscapeWheelName, name == escapement.GreatWheelName:
		return solidlens.RGB(0.78, 0.5, 0.18)
	case name == escapement.AnchorName:
		return solidlens.RGB(0.7, 0.28, 0.2)
	case name == escapement.PendulumName, name == escapement.CrutchName, name == escapement.SuspensionCapName:
		return solidlens.RGB(0.17, 0.5, 0.55)
	case name == escapement.MainspringName:
		return solidlens.RGB(0.62, 0.38, 0.62)
	case name == escapement.PawlCarrierName, name == escapement.SpringSpacerName, name == escapement.SpringAnchorPinName,
		name == escapement.WindingKnobName:
		return solidlens.RGB(0.45, 0.58, 0.3)
	}
	return solidlens.RGB(0.65, 0.47, 0.2)
}

// escapementBuild is the default movement with every body tessellated once.
type escapementBuild struct {
	doc      *decad.Document
	params   escapement.Params
	movement *escapement.Movement
	state    escapement.State
	geometry escapement.Geometry
	named    []escapement.NamedBody
	meshes   map[string]*decad.Mesh
}

func buildEscapement(ctx context.Context) (*escapementBuild, error) {
	p := escapement.DefaultParams()
	doc := escapement.NewDocument()
	m, err := escapement.Build(ctx, doc, p)
	if err != nil {
		return nil, fmt.Errorf("build escapement: %w", err)
	}
	g, err := escapement.Layout(p)
	if err != nil {
		return nil, err
	}
	b := &escapementBuild{doc: doc, params: p, movement: m, state: m.State, geometry: g, named: m.Named(), meshes: map[string]*decad.Mesh{}}
	for _, nb := range b.named {
		mesh, err := nb.Body.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, fmt.Errorf("tessellate %s: %w", nb.Name, err)
		}
		b.meshes[nb.Name] = mesh
	}
	return b, nil
}

// rotatedMesh draws a mesh moved by a rigid transform.
type rotatedMesh struct {
	solidlens.TriangleSource
	move r3.Transform
}

func (m rotatedMesh) Vertices() []r3.Vec {
	source := m.TriangleSource.Vertices()
	out := make([]r3.Vec, len(source))
	for i, v := range source {
		out[i] = m.move.Apply(v)
	}
	return out
}

// escapementModel returns the model of one body, moved by move and offset along Z.
func (b *escapementBuild) escapementModel(name string, move r3.Transform) solidlens.Model {
	return solidlens.Model{
		Mesh:     rotatedMesh{TriangleSource: b.meshes[name], move: move},
		Material: solidlens.Matte(escapementColor(name)),
		Edges:    solidlens.Edges{Enabled: true, Color: solidlens.RGB(0.025, 0.07, 0.11), Width: 1.2},
	}
}

// bodyFilter chooses the bodies a view draws.
type bodyFilter func(name string) bool

func everything(string) bool { return true }

func without(names ...string) bodyFilter {
	hidden := map[string]struct{}{}
	for _, n := range names {
		hidden[n] = struct{}{}
	}
	return func(name string) bool { _, h := hidden[name]; return !h }
}

// frontParts are the bodies in front of the escapement that a view of it leaves out.
var frontParts = func() []string {
	out := []string{escapement.FrontPlateName, escapement.FrontEscapeBearingName, escapement.FrontPalletBearingName,
		escapement.FrontGreatWheelBearingName}
	for i := range 4 {
		out = append(out, escapement.BoltName(i))
	}
	return out
}()

// driveParts are the spring drive's bodies.
var driveParts = []string{escapement.GreatWheelName, escapement.PawlCarrierName, escapement.MainspringName,
	escapement.SpringSpacerName, escapement.SpringAnchorPinName, escapement.WindingKnobName,
	escapement.GreatWheelArborName, escapement.BackGreatWheelBearingName}

// models returns the models of the bodies f keeps, at rest or moved by the offsets of layout, which must be the
// movement's exploded layout when exploded is true.
func (b *escapementBuild) models(f bodyFilter, layout *parts.Exploded) []solidlens.Model {
	offsets := map[string]r3.Vec{}
	if layout != nil {
		for _, part := range layout.Parts {
			offsets[part.Name] = part.Offset
		}
	}
	var out []solidlens.Model
	for _, nb := range b.named {
		if !f(nb.Name) {
			continue
		}
		move := r3.Identity()
		if t, err := r3.Translation(offsets[nb.Name]); err == nil {
			move = t
		}
		out = append(out, b.escapementModel(nb.Name, move))
	}
	return out
}

// explode runs the movement's derived explode over the build's document.
func (b *escapementBuild) explode(ctx context.Context) (*parts.Exploded, error) {
	set, err := b.movement.Set(b.doc, b.params)
	if err != nil {
		return nil, fmt.Errorf("set up escapement parts: %w", err)
	}
	layout, err := b.movement.Explode(ctx, set)
	if err != nil {
		return nil, fmt.Errorf("explode escapement: %w", err)
	}
	return layout, nil
}

// escapementCamera looks at the models' centre from direction dir (the way the camera lies from the target), with
// +Y up, and frames the models with the shared margin.
func escapementCamera(models []solidlens.Model, dir r3.Vec, aspect float64) solidlens.Camera {
	low, high := modelBounds(models)
	centre := low.Add(high).Scale(0.5)
	u, _ := dir.Normalize()
	camera := solidlens.Camera{
		Position: solidlens.Vec(centre.Add(u.Scale(900))), Target: solidlens.Vec(centre),
		Up: solidlens.Vec{Y: 1}, FOV: 30, Near: 1, Far: 4000,
	}
	return fitCamera(camera, models, aspect, escapementMargin)
}

func modelBounds(models []solidlens.Model) (low, high r3.Vec) {
	low = r3.NewVec(math.Inf(1), math.Inf(1), math.Inf(1))
	high = r3.NewVec(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, m := range models {
		for _, v := range m.Mesh.Vertices() {
			low = r3.NewVec(math.Min(low.X, v.X), math.Min(low.Y, v.Y), math.Min(low.Z, v.Z))
			high = r3.NewVec(math.Max(high.X, v.X), math.Max(high.Y, v.Y), math.Max(high.Z, v.Z))
		}
	}
	return low, high
}

var (
	escapementHeroDir  = r3.NewVec(0.35, 0.25, 1)
	escapementFrontDir = r3.NewVec(0, 0, 1)
	escapementAngleDir = r3.NewVec(1, 0.6, 1)
	escapementSideDir  = r3.NewVec(1, 0, 0)
	escapementBackDir  = r3.NewVec(0, 0, -1)
	// escapementExplodeDir is the exploded view's camera, on the side of the layer grid where no part hides a part of
	// another layer (parts.Exploded.Spread.View is the opposite of a direction in that set).
	escapementExplodeDir = r3.NewVec(-1, 0, 1.2)
)

// heroScene shows the whole movement at the default pose: the open plates let the wheel, anchor and drive show through.
func (b *escapementBuild) heroScene() solidlens.Scene {
	models := b.models(everything, nil)
	return bevelScene(escapementCamera(models, escapementHeroDir, 16.0/9), models)
}

func (b *escapementBuild) viewScene(dir r3.Vec) solidlens.Scene {
	models := b.models(everything, nil)
	return bevelScene(escapementCamera(models, dir, 16.0/9), models)
}

// backScene looks at the movement from behind. The shared lights shine toward -Z, so they leave the back faces dark;
// this scene turns them to shine toward +Z. The frame is square because the movement is taller than it is wide.
func (b *escapementBuild) backScene() solidlens.Scene {
	models := b.models(everything, nil)
	scene := bevelScene(escapementCamera(models, escapementBackDir, 1), models)
	for i := range scene.DirectionalLights {
		scene.DirectionalLights[i].Direction.Z = -scene.DirectionalLights[i].Direction.Z
	}
	return scene
}

// escapementGuideMinSize is the longest box edge, in mm, a part needs to get a guide line in the exploded view. It
// leaves the nuts, bolts and small bearings out, which would clutter the image. A part that moved along its own axis
// gets no line either, because its place in the stack already shows where it goes.
const escapementGuideMinSize = 13.0

// explodedScene is the movement pulled apart by the derived explode, framed square because the layer grid is, with a dotted guide line from each moved part
// back to where it assembles.
func (b *escapementBuild) explodedScene(ctx context.Context) (solidlens.Scene, error) {
	layout, err := b.explode(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	models := b.models(everything, layout)
	scene := bevelScene(escapementCamera(models, escapementExplodeDir, 16.0/9), models)
	for _, guide := range layout.Guides {
		size, err := guide.Size.In(units.Millimeter)
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("size of the guide of %s: %w", guide.Part, err)
		}
		if guide.Inline || size < escapementGuideMinSize {
			continue
		}
		scene.Lines = append(scene.Lines, solidlens.Line{
			Points: []solidlens.Vec{solidlens.Vec(guide.From), solidlens.Vec(guide.To)},
			Color:  solidlens.RGB(0.1, 0.12, 0.16), Width: 1.5, Dash: []float64{6, 5}})
	}
	return scene, nil
}

// closeupScene is a near-orthographic front view of the pallets and the teeth between them at Phase 0, where the entry
// pallet holds a tooth: the front plate, the crutch, the pendulum and the other parts in front of the anchor are left out.
func (b *escapementBuild) closeupScene() (solidlens.Scene, error) {
	hide := append(append([]string{}, frontParts...), escapement.CrutchName, escapement.PendulumName,
		escapement.PendulumBearingName, escapement.SuspensionPinName, escapement.SuspensionCapName,
		escapement.WindingKnobName)
	locked, err := b.geometry.StateAt(0)
	if err != nil {
		return solidlens.Scene{}, err
	}
	models, err := b.motionModels(without(hide...), b.escapementMovers(), locked)
	if err != nil {
		return solidlens.Scene{}, err
	}
	const distance = 600.0
	centre := r3.NewVec(0, b.geometry.P.Y-24, 25)
	halfHeight := 24.0
	camera := solidlens.Camera{
		Position: solidlens.Vec(centre.Add(r3.NewVec(0, 0, distance))), Target: solidlens.Vec(centre),
		Up: solidlens.Vec{Y: 1}, FOV: 2 * math.Atan(halfHeight/distance) * 180 / math.Pi, Near: 1, Far: 4000,
	}
	return bevelScene(camera, models), nil
}

// escapementPart names one image of a distinct body.
type escapementPart struct {
	shot   string
	bodies []string
	dir    r3.Vec
}

var escapementParts = []escapementPart{
	{"back-plate", []string{escapement.BackPlateName}, r3.NewVec(0.5, 0.4, 1)},
	{"front-plate", []string{escapement.FrontPlateName}, r3.NewVec(0.5, 0.4, 1)},
	{"escape-wheel", []string{escapement.EscapeWheelName}, r3.NewVec(0.5, 0.4, 1)},
	{"anchor", []string{escapement.AnchorName}, r3.NewVec(0.5, 0.4, 1)},
	{"crutch", []string{escapement.CrutchName}, r3.NewVec(0.5, 0.4, 1)},
	{"pendulum", []string{escapement.PendulumName}, r3.NewVec(0.4, 0.3, 1)},
	{"suspension-pin", []string{escapement.SuspensionPinName}, r3.NewVec(1, 0.3, 0.6)},
	{"suspension-cap", []string{escapement.SuspensionCapName}, r3.NewVec(0.5, 0.4, 1)},
	{"great-wheel", []string{escapement.GreatWheelName}, r3.NewVec(0.5, 0.4, 1)},
	{"pawl-carrier", []string{escapement.PawlCarrierName}, r3.NewVec(0.5, 0.6, 1)},
	{"mainspring", []string{escapement.MainspringName}, r3.NewVec(0.5, 0.4, 1)},
	{"spring-spacer", []string{escapement.SpringSpacerName}, r3.NewVec(1, 0.3, 0.5)},
	{"spring-anchor-pin", []string{escapement.SpringAnchorPinName}, r3.NewVec(1, 0.3, 0.5)},
	{"winding-knob", []string{escapement.WindingKnobName}, r3.NewVec(0.5, 0.4, 1)},
	{"escape-arbor", []string{escapement.EscapeArborName}, r3.NewVec(1, 0.3, 0.5)},
	{"pallet-arbor", []string{escapement.PalletArborName}, r3.NewVec(1, 0.3, 0.5)},
	{"great-wheel-arbor", []string{escapement.GreatWheelArborName}, r3.NewVec(1, 0.3, 0.5)},
	{"arbor-collar", []string{escapement.EscapeRearCollarName}, r3.NewVec(1, 0.3, 0.5)},
	{"bearing", []string{escapement.BackEscapeBearingName}, r3.NewVec(0.5, 0.5, 1)},
	{"bolt", []string{escapement.BoltName(0), escapement.NutName(0)}, r3.NewVec(1, 0.3, 0.5)},
}

func (b *escapementBuild) partScene(part escapementPart) solidlens.Scene {
	keep := map[string]struct{}{}
	for _, n := range part.bodies {
		keep[n] = struct{}{}
	}
	models := b.models(func(name string) bool { _, ok := keep[name]; return ok }, nil)
	return bevelScene(escapementCamera(models, part.dir, 16.0/9), models)
}

func init() {
	cached := func(f func(*escapementBuild) solidlens.Scene) func(context.Context) (solidlens.Scene, error) {
		return func(ctx context.Context) (solidlens.Scene, error) {
			b, err := buildEscapement(ctx)
			if err != nil {
				return solidlens.Scene{}, err
			}
			return f(b), nil
		}
	}
	presets["escapement/hero"] = preset{width: 1600, height: 900, scene: cached((*escapementBuild).heroScene)}
	presets["escapement/index"] = preset{width: 480, height: 270, scene: cached((*escapementBuild).heroScene)}
	presets["escapement/assembled-front"] = preset{width: 1200, height: 675, scene: cached(func(b *escapementBuild) solidlens.Scene { return b.viewScene(escapementFrontDir) })}
	presets["escapement/assembled-angle"] = preset{width: 1200, height: 675, scene: cached(func(b *escapementBuild) solidlens.Scene { return b.viewScene(escapementAngleDir) })}
	presets["escapement/assembled-side"] = preset{width: 1200, height: 675, scene: cached(func(b *escapementBuild) solidlens.Scene { return b.viewScene(escapementSideDir) })}
	presets["escapement/assembled-back"] = preset{width: 1000, height: 1000, scene: cached((*escapementBuild).backScene)}
	presets["escapement/closeup"] = preset{width: 1200, height: 675, scene: func(ctx context.Context) (solidlens.Scene, error) {
		b, err := buildEscapement(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return b.closeupScene()
	}}
	presets["escapement/exploded"] = preset{width: 1600, height: 900, scene: func(ctx context.Context) (solidlens.Scene, error) {
		b, err := buildEscapement(ctx)
		if err != nil {
			return solidlens.Scene{}, err
		}
		return b.explodedScene(ctx)
	}}
	for _, part := range escapementParts {
		presets["escapement/part-"+part.shot] = preset{width: 480, height: 270,
			scene: cached(func(b *escapementBuild) solidlens.Scene { return b.partScene(part) })}
	}
}

// renderEscapementReadme writes every still of the model's README from one build.
func renderEscapementReadme(ctx context.Context, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("-out is required")
	}
	b, err := buildEscapement(ctx)
	if err != nil {
		return err
	}
	closeup, err := b.closeupScene()
	if err != nil {
		return err
	}
	exploded, err := b.explodedScene(ctx)
	if err != nil {
		return err
	}
	for _, image := range []struct {
		name          string
		width, height int
		scene         solidlens.Scene
	}{
		{"hero.png", 1600, 900, b.heroScene()},
		{"index.png", 480, 270, b.heroScene()},
		{"assembled-front.png", 1200, 675, b.viewScene(escapementFrontDir)},
		{"assembled-angle.png", 1200, 675, b.viewScene(escapementAngleDir)},
		{"assembled-side.png", 1200, 675, b.viewScene(escapementSideDir)},
		{"assembled-back.png", 1000, 1000, b.backScene()},
		{"closeup.png", 1200, 675, closeup},
		{"exploded.png", 1600, 900, exploded},
	} {
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), image.scene, image.width, image.height); err != nil {
			return fmt.Errorf("render escapement %s: %w", image.name, err)
		}
	}
	for _, part := range escapementParts {
		name := filepath.Join(outputDir, "parts", part.shot+".png")
		if err := renderPNG(ctx, name, b.partScene(part), 480, 270); err != nil {
			return fmt.Errorf("render escapement part %s: %w", part.shot, err)
		}
	}
	return nil
}

// escapementHero builds the movement and returns the hero scene, for the gallery tile.
func escapementHero(ctx context.Context) (solidlens.Scene, error) {
	b, err := buildEscapement(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return b.heroScene(), nil
}
