package main

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// bevelSpiralCameras frame the 31/31 spiral pair alone.
var bevelSpiralCameras = map[string]solidlens.Camera{
	"hero": {
		Position: solidlens.Vec{X: -43, Y: -43, Z: 43},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Z: 1},
		FOV:      35,
	},
	"top": {
		Position: solidlens.Vec{X: 12, Y: 12, Z: 100},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Y: 1},
		FOV:      42,
	},
	"low": {
		Position: solidlens.Vec{X: -60, Y: -30, Z: 22},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Z: 1},
		FOV:      40,
	},
	"high": {
		Position: solidlens.Vec{X: -50, Y: -20, Z: 75},
		Target:   solidlens.Vec{X: 12, Y: 12},
		Up:       solidlens.Vec{Z: 1},
		FOV:      38,
	},
}

// straightCamera frames the straight pair from one of the README angles. The camera looks at the
// center of the pair's bounding box from a distance set by the box's diagonal.
func straightCamera(shot string, low, high r3.Vec) solidlens.Camera {
	center := low.Add(high).Scale(0.5)
	span := high.Sub(low).Len()
	at := func(x, y, z float64) solidlens.Vec {
		return solidlens.Vec{X: center.X + 1.4*x*span, Y: center.Y + 1.4*y*span, Z: center.Z + 1.4*z*span}
	}
	target := solidlens.Vec(center)
	switch shot {
	case "top":
		return solidlens.Camera{Position: at(0, 0, 1.6), Target: target, Up: solidlens.Vec{Y: 1}, FOV: 30}
	case "low":
		return solidlens.Camera{Position: at(-0.9, -0.7, 0.3), Target: target, Up: solidlens.Vec{Z: 1}, FOV: 30}
	default:
		return solidlens.Camera{Position: at(-0.6, -0.4, 0.9), Target: target, Up: solidlens.Vec{Z: 1}, FOV: 30}
	}
}

// bevelHeroGap is the clear distance along X between the two pairs in the hero scene.
const bevelHeroGap = 30.0

// bevelHeroEdgeTolerance is the Edges.Tolerance of the hero scene in millimetres. It is about half a
// pixel at the 1600 by 900 hero size. Left at its default, SolidLens would scale it to the image, and
// the 480 by 270 index image would get 0.40 mm, which breaks the straight teeth's outline into dashes.
const bevelHeroEdgeTolerance = 0.12

func bevelGearHero(ctx context.Context) (solidlens.Scene, error) {
	pairs, err := buildBevelPairs(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return pairs.heroScene(), nil
}

func bevelGearTop(ctx context.Context) (solidlens.Scene, error) {
	return bevelSpiralView(ctx, "top")
}

func bevelGearLow(ctx context.Context) (solidlens.Scene, error) {
	return bevelSpiralView(ctx, "low")
}

func bevelGearHigh(ctx context.Context) (solidlens.Scene, error) {
	return bevelSpiralView(ctx, "high")
}

func bevelGearStraightTop(ctx context.Context) (solidlens.Scene, error) {
	return bevelStraightView(ctx, "top")
}

func bevelGearStraightLow(ctx context.Context) (solidlens.Scene, error) {
	return bevelStraightView(ctx, "low")
}

func bevelGearStraightHigh(ctx context.Context) (solidlens.Scene, error) {
	return bevelStraightView(ctx, "high")
}

func bevelSpiralView(ctx context.Context, shot string) (solidlens.Scene, error) {
	spiral, err := buildSpiralBevelPair(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	return bevelViewScene(bevelSpiralCameras[shot], spiral.models(spiralColor, 1, r3.Vec{})), nil
}

func bevelStraightView(ctx context.Context, shot string) (solidlens.Scene, error) {
	straight, err := buildStraightBevelPair(ctx)
	if err != nil {
		return solidlens.Scene{}, err
	}
	straightMin, straightMax := straight.bounds(1)
	return bevelViewScene(straightCamera(shot, straightMin, straightMax), straight.models(straightColor, 1, r3.Vec{})), nil
}

// renderBevelReadme builds each pair once and writes every bevel gear README image.
func renderBevelReadme(ctx context.Context, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("-out is required")
	}
	pairs, err := buildBevelPairs(ctx)
	if err != nil {
		return err
	}
	hero := pairs.heroScene()
	spiral := pairs.spiral.models(spiralColor, 1, r3.Vec{})
	straight := pairs.straight.models(straightColor, 1, r3.Vec{})
	straightMin, straightMax := pairs.straight.bounds(1)
	images := []struct {
		name          string
		scene         solidlens.Scene
		width, height int
	}{
		{"index.png", hero, 480, 270},
		{"hero.png", hero, 1600, 900},
		{"top.png", bevelViewScene(bevelSpiralCameras["top"], spiral), bevelViewWidth, bevelViewHeight},
		{"low.png", bevelViewScene(bevelSpiralCameras["low"], spiral), bevelViewWidth, bevelViewHeight},
		{"high.png", bevelViewScene(bevelSpiralCameras["high"], spiral), bevelViewWidth, bevelViewHeight},
		{"straight-top.png", bevelViewScene(straightCamera("top", straightMin, straightMax), straight), bevelViewWidth, bevelViewHeight},
		{"straight-low.png", bevelViewScene(straightCamera("low", straightMin, straightMax), straight), bevelViewWidth, bevelViewHeight},
		{"straight-high.png", bevelViewScene(straightCamera("high", straightMin, straightMax), straight), bevelViewWidth, bevelViewHeight},
	}
	for _, image := range images {
		if err := renderPNG(ctx, filepath.Join(outputDir, image.name), image.scene, image.width, image.height); err != nil {
			return fmt.Errorf("render bevelgear %s: %w", image.name, err)
		}
	}
	return nil
}

var (
	spiralColor   = solidlens.RGB(0.17, 0.42, 0.6)
	straightColor = solidlens.RGB(0.72, 0.4, 0.14)
)

// bevelPairMesh holds the verified meshes of one built pair.
type bevelPairMesh struct {
	pinion, driving *decad.Mesh
}

type bevelPairs struct {
	spiral, straight bevelPairMesh
}

func buildBevelPairs(ctx context.Context) (bevelPairs, error) {
	spiral, err := buildSpiralBevelPair(ctx)
	if err != nil {
		return bevelPairs{}, err
	}
	straight, err := buildStraightBevelPair(ctx)
	if err != nil {
		return bevelPairs{}, err
	}
	return bevelPairs{spiral: spiral, straight: straight}, nil
}

// heroScene shows the spiral pair enlarged to the straight pair's size, with the straight pair
// moved along +X until the two bounding boxes are bevelHeroGap apart.
func (p bevelPairs) heroScene() solidlens.Scene {
	spiralMin, spiralMax := p.spiral.bounds(bevelSpiralScale)
	straightMin, straightMax := p.straight.bounds(1)
	offset := r3.NewVec(spiralMax.X-straightMin.X+bevelHeroGap, spiralMin.Y-straightMin.Y, 0)
	models := p.spiral.models(spiralColor, bevelSpiralScale, r3.Vec{})
	models = append(models, p.straight.models(straightColor, 1, offset)...)
	for i := range models {
		models[i].Edges.Tolerance = bevelHeroEdgeTolerance
	}
	low := r3.NewVec(spiralMin.X, spiralMin.Y, math.Min(spiralMin.Z, straightMin.Z+offset.Z))
	high := r3.NewVec(straightMax.X+offset.X, math.Max(spiralMax.Y, straightMax.Y+offset.Y),
		math.Max(spiralMax.Z, straightMax.Z+offset.Z))
	center := low.Add(high).Scale(0.5)
	span := high.Sub(low).Len()
	camera := solidlens.Camera{
		Position: solidlens.Vec{X: center.X - 0.1*span, Y: center.Y - 1.15*span, Z: center.Z + 0.55*span},
		Target:   solidlens.Vec(center),
		Up:       solidlens.Vec{Z: 1},
		FOV:      29,
	}
	return bevelScene(centerHorizontally(camera, models), models)
}

// bevelViewWidth and bevelViewHeight are the pixel size of each README "More views" image.
const (
	bevelViewWidth  = 1200
	bevelViewHeight = 1000
)

// bevelViewMargin is the share of each image dimension left empty on both sides of a view's models.
const bevelViewMargin = 0.06

// bevelViewScene frames the models in the middle of a bevelViewWidth by bevelViewHeight image.
func bevelViewScene(camera solidlens.Camera, models []solidlens.Model) solidlens.Scene {
	return bevelScene(fitCamera(camera, models, float64(bevelViewWidth)/bevelViewHeight, bevelViewMargin), models)
}

// fitCamera slides the camera parallel to the image plane until the models' projected extents are centered,
// then narrows or widens its field of view until the extents leave margin of the image free on every side.
// The camera keeps its viewing direction and distance.
func fitCamera(camera solidlens.Camera, models []solidlens.Model, aspect, margin float64) solidlens.Camera {
	position, target := r3.Vec(camera.Position), r3.Vec(camera.Target)
	forward, ok := target.Sub(position).Normalize()
	if !ok {
		return camera
	}
	right, ok := forward.Cross(r3.Vec(camera.Up)).Normalize()
	if !ok {
		return camera
	}
	up := right.Cross(forward)
	var vertices []r3.Vec
	for _, model := range models {
		vertices = append(vertices, model.Mesh.Vertices()...)
	}
	var halfX, halfY float64
	for range 20 {
		minX, maxX := math.Inf(1), math.Inf(-1)
		minY, maxY := math.Inf(1), math.Inf(-1)
		var depthSum float64
		for _, v := range vertices {
			d := v.Sub(position)
			depth := d.Dot(forward)
			x, y := d.Dot(right)/depth, d.Dot(up)/depth
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
			depthSum += depth
		}
		meanDepth := depthSum / float64(len(vertices))
		shiftX := (minX + maxX) / 2 * meanDepth
		shiftY := (minY + maxY) / 2 * meanDepth
		shift := right.Scale(shiftX).Add(up.Scale(shiftY))
		position, target = position.Add(shift), target.Add(shift)
		halfX, halfY = (maxX-minX)/2, (maxY-minY)/2
	}
	// The vertical field of view FOV satisfies tan(FOV/2) = half the image height in image-plane units.
	tanHalf := math.Max(halfX/aspect, halfY) / (1 - 2*margin)
	camera.Position, camera.Target = solidlens.Vec(position), solidlens.Vec(target)
	camera.FOV = 2 * math.Atan(tanHalf) * 180 / math.Pi
	return camera
}

// centerHorizontally slides the camera sideways, parallel to the image plane, until the leftmost and
// rightmost model vertices sit the same distance from the image's left and right edges.
func centerHorizontally(camera solidlens.Camera, models []solidlens.Model) solidlens.Camera {
	position, target := r3.Vec(camera.Position), r3.Vec(camera.Target)
	forward, ok := target.Sub(position).Normalize()
	if !ok {
		return camera
	}
	right, ok := forward.Cross(r3.Vec(camera.Up)).Normalize()
	if !ok {
		return camera
	}
	var vertices []r3.Vec
	for _, model := range models {
		vertices = append(vertices, model.Mesh.Vertices()...)
	}
	for range 20 {
		left, rightmost := math.Inf(1), math.Inf(-1)
		var leftDepth, rightDepth float64
		for _, v := range vertices {
			d := v.Sub(position)
			depth := d.Dot(forward)
			x := d.Dot(right) / depth
			if x < left {
				left, leftDepth = x, depth
			}
			if x > rightmost {
				rightmost, rightDepth = x, depth
			}
		}
		shift := (left + rightmost) / 2 * (leftDepth + rightDepth) / 2
		position = position.Add(right.Scale(shift))
		target = target.Add(right.Scale(shift))
	}
	camera.Position, camera.Target = solidlens.Vec(position), solidlens.Vec(target)
	return camera
}

// buildSpiralBevelPair builds the source-default bored 31/31 spiral pair.
func buildSpiralBevelPair(ctx context.Context) (bevelPairMesh, error) {
	pair, err := bevelgear.BuildPair(ctx, decad.New(), bevelgear.DefaultParams())
	if err != nil {
		return bevelPairMesh{}, fmt.Errorf("build spiral bevel gear pair: %w", err)
	}
	return tessellateBevelPair(ctx, pair)
}

// buildStraightBevelPair builds the bored 32/16 straight-tooth pair at Module 4.
func buildStraightBevelPair(ctx context.Context) (bevelPairMesh, error) {
	params := bevelgear.DefaultParams()
	params.Module = 4
	params.Teeth, params.MateTeeth = 32, 16
	params.Samples = 15
	params.SpiralAngle = 0
	pair, err := bevelgear.BuildPair(ctx, decad.New(), params)
	if err != nil {
		return bevelPairMesh{}, fmt.Errorf("build straight bevel gear pair: %w", err)
	}
	return tessellateBevelPair(ctx, pair)
}

func tessellateBevelPair(ctx context.Context, pair bevelgear.GearPair) (bevelPairMesh, error) {
	var meshes bevelPairMesh
	for _, member := range []struct {
		name string
		body *decad.Body
		mesh **decad.Mesh
	}{
		{"pinion", pair.Pinion, &meshes.pinion},
		{"driving", pair.Driving, &meshes.driving},
	} {
		mesh, err := member.body.Tessellate(ctx, units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		if err != nil {
			return bevelPairMesh{}, fmt.Errorf("tessellate %s gear: %w", member.name, err)
		}
		*member.mesh = mesh
	}
	return meshes, nil
}

// bounds returns the bounding box of both members after enlarging them by scale about the origin.
func (m bevelPairMesh) bounds(scale float64) (low, high r3.Vec) {
	first := true
	for _, mesh := range []*decad.Mesh{m.pinion, m.driving} {
		for _, v := range mesh.Vertices() {
			v = v.Scale(scale)
			if first {
				low, high, first = v, v, false
				continue
			}
			low = r3.NewVec(math.Min(low.X, v.X), math.Min(low.Y, v.Y), math.Min(low.Z, v.Z))
			high = r3.NewVec(math.Max(high.X, v.X), math.Max(high.Y, v.Y), math.Max(high.Z, v.Z))
		}
	}
	return low, high
}

func (m bevelPairMesh) models(color solidlens.Color, scale float64, offset r3.Vec) []solidlens.Model {
	models := make([]solidlens.Model, 0, 2)
	for _, mesh := range []*decad.Mesh{m.pinion, m.driving} {
		var source solidlens.TriangleSource = mesh
		if scale != 1 || offset != (r3.Vec{}) {
			source = placedMesh{TriangleSource: mesh, scale: scale, offset: offset}
		}
		models = append(models, solidlens.Model{
			Mesh:     source,
			Material: solidlens.Matte(color),
			Edges: solidlens.Edges{
				Enabled: true,
				Color:   solidlens.RGB(0.025, 0.07, 0.11),
				Width:   1.5,
			},
		})
	}
	return models
}

// bevelSpiralScale enlarges the Module 1 spiral pair to the straight pair's Module 4 size in the hero.
const bevelSpiralScale = 4

// placedMesh draws a mesh enlarged about the origin, which is the pair's pitch apex, then moved by offset.
type placedMesh struct {
	solidlens.TriangleSource
	scale  float64
	offset r3.Vec
}

func (m placedMesh) Vertices() []r3.Vec {
	source := m.TriangleSource.Vertices()
	placed := make([]r3.Vec, len(source))
	for i, v := range source {
		placed[i] = v.Scale(m.scale).Add(m.offset)
	}
	return placed
}

func bevelScene(camera solidlens.Camera, models []solidlens.Model) solidlens.Scene {
	return solidlens.Scene{
		Camera: camera,
		Models: models,
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
			{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
		},
		Background: solidlens.RGB(0.88, 0.92, 0.95),
	}
}
