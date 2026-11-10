package main

import (
	"context"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
)

var (
	planetaryBoltColor = solidlens.RGB(0.2, 0.22, 0.26)
	planetaryNutColor  = solidlens.RGB(0.8, 0.62, 0.2)
)

// planetaryFastenerNormal is the normal of the cut plane through the gearbox axis and the first bolt, which lies at 45° on the
// bolt circle. Its normal points toward the camera's side, which is the half the cut removes.
var planetaryFastenerNormal = solidlens.Vec{X: math.Sqrt2 / 2, Y: -math.Sqrt2 / 2}

// planetaryFastenerPart is one body of a fastener image.
type planetaryFastenerPart struct {
	name  string
	body  *decad.Body
	color solidlens.Color
	// lift raises the body along the axis, for the exploded image.
	lift float64
}

// planetaryFastenerScene cuts every part along the plane through the first bolt and draws it with camera. The
// first bolt of each cover and its nut are the ones the plane passes through.
func planetaryFastenerScene(ctx context.Context, parts []planetaryFastenerPart, camera solidlens.Camera) (solidlens.Scene, error) {
	var models []solidlens.Model
	for _, part := range parts {
		mesh, err := planetaryMesh(ctx, planetaryBody{name: part.name, body: part.body})
		if err != nil {
			return solidlens.Scene{}, err
		}
		model := solidlens.Model{
			Mesh:     placedMesh{TriangleSource: mesh, scale: 1, offset: r3.NewVec(0, 0, part.lift)},
			Material: solidlens.Matte(part.color),
			Edges:    planetaryEdges,
		}
		model = planetaryCut(model, part.color)
		model.Clip.Normal = planetaryFastenerNormal
		models = append(models, model)
	}
	return bevelScene(camera, models), nil
}

// planetaryFastenerCamera looks square at the cut from distance, aimed at the point on the cut plane at
// height z and distance radius from the axis, with lift raising the camera.
func planetaryFastenerCamera(radius, z, distance, lift float64) solidlens.Camera {
	at := solidlens.Vec{X: radius * math.Sqrt2 / 2, Y: radius * math.Sqrt2 / 2, Z: z}
	return solidlens.Camera{
		Position: solidlens.Vec{
			X: at.X + planetaryFastenerNormal.X*distance,
			Y: at.Y + planetaryFastenerNormal.Y*distance,
			Z: z + lift,
		},
		Target: at, Up: solidlens.Vec{Z: 1}, FOV: 35,
	}
}

// planetaryFastenerScenes builds the one-stage gearbox once and returns the section and the exploded scenes.
func planetaryFastenerScenes(ctx context.Context) (section, exploded solidlens.Scene, err error) {
	box, _, err := buildPlanetaryGearbox(ctx, 1)
	if err != nil {
		return solidlens.Scene{}, solidlens.Scene{}, err
	}
	return planetaryFastenerScenesFrom(ctx, box)
}

func planetaryFastenerScenesFrom(ctx context.Context, box planetarygearbox.Gearbox) (section, exploded solidlens.Scene, err error) {
	if section, err = planetaryFastenerSection(ctx, box); err != nil {
		return solidlens.Scene{}, solidlens.Scene{}, err
	}
	if exploded, err = planetaryFastenerExploded(ctx, box); err != nil {
		return solidlens.Scene{}, solidlens.Scene{}, err
	}
	return section, exploded, nil
}

// planetaryFastenerSection draws the one-stage housing cut through the first bolt of each cover: both bolts, both
// nuts, both covers and the ring, in place.
func planetaryFastenerSection(ctx context.Context, box planetarygearbox.Gearbox) (solidlens.Scene, error) {
	parts := []planetaryFastenerPart{
		{"ring", box.Ring, planetaryRingColor, 0},
		{"front cover", box.FrontCover, planetaryCoverColor, 0},
		{"rear cover", box.RearCover, planetaryCoverColor, 0},
		{"front bolt", box.Bolts[0], planetaryBoltColor, 0},
		{"rear bolt", box.Bolts[4], planetaryBoltColor, 0},
		{"front nut", box.Nuts[0], planetaryNutColor, 0},
		{"rear nut", box.Nuts[4], planetaryNutColor, 0},
	}
	return planetaryFastenerScene(ctx, parts, planetaryFastenerCamera(30, 6, 34, 0))
}

// planetaryFastenerExploded pulls the rear end of the housing apart along the first bolt's axis, in the order
// they go together: the ring, its nut, the cover and the bolt.
func planetaryFastenerExploded(ctx context.Context, box planetarygearbox.Gearbox) (solidlens.Scene, error) {
	parts := []planetaryFastenerPart{
		{"ring", box.Ring, planetaryRingColor, 0},
		{"rear nut", box.Nuts[4], planetaryNutColor, 8},
		{"rear cover", box.RearCover, planetaryCoverColor, 16},
		{"rear bolt", box.Bolts[4], planetaryBoltColor, 28},
	}
	return planetaryFastenerScene(ctx, parts, planetaryFastenerCamera(32, 21, 80, 12))
}

func init() {
	presets["planetarygearbox/fasteners-section"] = preset{width: 1600, height: 900,
		scene: func(ctx context.Context) (solidlens.Scene, error) {
			section, _, err := planetaryFastenerScenes(ctx)
			return section, err
		}}
	presets["planetarygearbox/fasteners-exploded"] = preset{width: 1600, height: 900,
		scene: func(ctx context.Context) (solidlens.Scene, error) {
			_, exploded, err := planetaryFastenerScenes(ctx)
			return exploded, err
		}}
}
