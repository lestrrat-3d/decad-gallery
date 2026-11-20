package main

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// fastenerSet holds the render settings of one catalog preset.
type fastenerSet struct {
	preset      string                      // catalog name
	length      float64                     // bolt length in single-bolt views and under the nut in "withbolt"
	pitch       float64                     // spacing between bolts in the "lengths" view
	boltCameras map[string]solidlens.Camera // keys: hero, head, side, lengths, and underside for M2
	nutCameras  map[string]solidlens.Camera // keys: hero, top, side, withbolt, and low for M2
}

// The nut stands on its lower face with its axis along +Z, except in "withbolt", where a bolt lies along X.
// Bolts lie on their sides along X with the head toward +X.
var fastenerSets = map[string]fastenerSet{
	"m2": {
		preset: "M2", length: 10, pitch: 6,
		boltCameras: map[string]solidlens.Camera{
			"hero":      {Position: solidlens.Vec{X: 9, Y: -9, Z: 7}, Target: solidlens.Vec{X: -2}, Up: solidlens.Vec{Z: 1}, FOV: 35},
			"head":      {Position: solidlens.Vec{X: 30, Y: -8, Z: 6}, Target: solidlens.Vec{}, Up: solidlens.Vec{Z: 1}, FOV: 30},
			"side":      {Position: solidlens.Vec{X: -4, Y: -40, Z: 2}, Target: solidlens.Vec{X: -4}, Up: solidlens.Vec{Z: 1}, FOV: 20},
			"underside": {Position: solidlens.Vec{X: -22, Y: -14, Z: -10}, Target: solidlens.Vec{X: -2}, Up: solidlens.Vec{Z: 1}, FOV: 35},
			"lengths":   {Position: solidlens.Vec{X: -6, Y: 8, Z: 45}, Target: solidlens.Vec{X: -6, Y: 12}, Up: solidlens.Vec{Y: 1}, FOV: 30},
		},
		nutCameras: map[string]solidlens.Camera{
			"hero":     {Position: solidlens.Vec{X: 7, Y: -9, Z: 7}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
			"top":      {Position: solidlens.Vec{X: 1, Y: -1.5, Z: 20}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Y: 1}, FOV: 25},
			"side":     {Position: solidlens.Vec{X: 0, Y: -30, Z: .8}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Z: 1}, FOV: 20},
			"low":      {Position: solidlens.Vec{X: 7, Y: -9, Z: -7}, Target: solidlens.Vec{Z: .8}, Up: solidlens.Vec{Z: 1}, FOV: 30},
			"withbolt": {Position: solidlens.Vec{X: 9, Y: -12, Z: 9}, Target: solidlens.Vec{X: -4}, Up: solidlens.Vec{Z: 1}, FOV: 35},
		},
	},
	"m5": {
		preset: "M5", length: 25, pitch: 12,
		boltCameras: map[string]solidlens.Camera{
			"hero":    {Position: solidlens.Vec{X: 22, Y: -24, Z: 18}, Target: solidlens.Vec{X: -9}, Up: solidlens.Vec{Z: 1}, FOV: 35},
			"head":    {Position: solidlens.Vec{X: 40, Y: -14, Z: 12}, Target: solidlens.Vec{}, Up: solidlens.Vec{Z: 1}, FOV: 30},
			"side":    {Position: solidlens.Vec{X: -10, Y: -80, Z: 4}, Target: solidlens.Vec{X: -10}, Up: solidlens.Vec{Z: 1}, FOV: 20},
			"lengths": {Position: solidlens.Vec{X: -35, Y: 40, Z: 150}, Target: solidlens.Vec{X: -35, Y: 40}, Up: solidlens.Vec{Y: 1}, FOV: 30},
		},
		nutCameras: map[string]solidlens.Camera{
			"hero":     {Position: solidlens.Vec{X: 11, Y: -14, Z: 11}, Target: solidlens.Vec{Z: 2.3}, Up: solidlens.Vec{Z: 1}, FOV: 30},
			"top":      {Position: solidlens.Vec{X: 1, Y: -2, Z: 30}, Target: solidlens.Vec{Z: 2.3}, Up: solidlens.Vec{Y: 1}, FOV: 25},
			"side":     {Position: solidlens.Vec{X: 0, Y: -45, Z: 2.3}, Target: solidlens.Vec{Z: 2.3}, Up: solidlens.Vec{Z: 1}, FOV: 20},
			"withbolt": {Position: solidlens.Vec{X: 22, Y: -28, Z: 20}, Target: solidlens.Vec{X: -12}, Up: solidlens.Vec{Z: 1}, FOV: 35},
		},
	},
}

// fastenerShot maps one image (file stem) to its preset set, part ("bolt" or "nut") and view.
type fastenerShot struct{ shot, set, part, view string }

// fastenerShots lists every image in README order.
var fastenerShots = []fastenerShot{
	{"hero", "m2", "bolt", "hero"},
	{"m2-bolt-head", "m2", "bolt", "head"}, {"m2-bolt-side", "m2", "bolt", "side"},
	{"m2-bolt-underside", "m2", "bolt", "underside"}, {"m2-bolt-lengths", "m2", "bolt", "lengths"},
	{"m2-nut-hero", "m2", "nut", "hero"}, {"m2-nut-top", "m2", "nut", "top"}, {"m2-nut-side", "m2", "nut", "side"},
	{"m2-nut-low", "m2", "nut", "low"}, {"m2-nut-withbolt", "m2", "nut", "withbolt"},
	{"m5-bolt-hero", "m5", "bolt", "hero"}, {"m5-bolt-head", "m5", "bolt", "head"},
	{"m5-bolt-side", "m5", "bolt", "side"}, {"m5-bolt-lengths", "m5", "bolt", "lengths"},
	{"m5-nut-hero", "m5", "nut", "hero"}, {"m5-nut-top", "m5", "nut", "top"}, {"m5-nut-side", "m5", "nut", "side"},
	{"m5-nut-withbolt", "m5", "nut", "withbolt"},
}

func init() {
	for _, s := range fastenerShots {
		presets["fastener/"+s.shot] = preset{width: 1600, height: 900, scene: fastenerScene(s)}
	}
}

func fastenerScene(s fastenerShot) func(context.Context) (solidlens.Scene, error) {
	return func(ctx context.Context) (solidlens.Scene, error) { return fastenerView(ctx, s) }
}

func fastenerModel(mesh *decad.Mesh) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(solidlens.RGB(.5, .52, .56)),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// fastenerView builds one image.
func fastenerView(ctx context.Context, s fastenerShot) (solidlens.Scene, error) {
	set, ok := fastenerSets[s.set]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown fastener set %q", s.set)
	}
	switch s.part {
	case "bolt":
		return fastenerBoltView(ctx, set, s.view)
	case "nut":
		return fastenerNutView(ctx, set, s.view)
	}
	return solidlens.Scene{}, fmt.Errorf("unknown fastener part %q", s.part)
}

// fastenerBoltView builds one bolt view. Every view but "lengths" shows the set's length.
func fastenerBoltView(ctx context.Context, set fastenerSet, name string) (solidlens.Scene, error) {
	camera, ok := set.boltCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown %s bolt view %q", set.preset, name)
	}
	lengths := []float64{set.length}
	if name == "lengths" {
		preset, ok := fastener.Lookup(set.preset)
		if !ok {
			return solidlens.Scene{}, fmt.Errorf("catalog has no %s preset", set.preset)
		}
		lengths = preset.Bolt.StandardLengths
	}
	doc := decad.New()
	var models []solidlens.Model
	for i, length := range lengths {
		// The frame maps the bolt's +Z, from underside to head, to world +X.
		frame, err := r3.NewFrame(r3.Vec{Y: float64(i) * set.pitch}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
		}
		m, err := fastener.BuildBolt(ctx, doc, fastener.BoltParams{Preset: set.preset, Length: length, Placement: &frame})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build %sx%g bolt: %w", set.preset, length, err)
		}
		mesh, err := m.Body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh %sx%g bolt: %w", set.preset, length, err)
		}
		models = append(models, fastenerModel(mesh))
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// fastenerNutView builds one view of the nut, or of the nut on a bolt for "withbolt".
func fastenerNutView(ctx context.Context, set fastenerSet, name string) (solidlens.Scene, error) {
	camera, ok := set.nutCameras[name]
	if !ok {
		return solidlens.Scene{}, fmt.Errorf("unknown %s nut view %q", set.preset, name)
	}
	doc := decad.New()
	params := fastener.NutParams{Preset: set.preset}
	var bodies []*decad.Body
	if name == "withbolt" {
		// Both frames map their +Z to world +X. The bolt's head underside is at x = 0 and its tip at x = -length; the
		// nut sits 1 mm from the tip.
		boltFrame, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create bolt frame: %w", err)
		}
		bolt, err := fastener.BuildBolt(ctx, doc, fastener.BoltParams{Preset: set.preset, Length: set.length, Placement: &boltFrame})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("build %s bolt: %w", set.preset, err)
		}
		bodies = append(bodies, bolt.Body)
		nutFrame, err := r3.NewFrame(r3.Vec{X: -(set.length - 1)}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("create nut frame: %w", err)
		}
		params.Placement = &nutFrame
	}
	nut, err := fastener.BuildNut(ctx, doc, params)
	if err != nil {
		return solidlens.Scene{}, fmt.Errorf("build %s nut: %w", set.preset, err)
	}
	bodies = append(bodies, nut.Body)
	var models []solidlens.Model
	for _, body := range bodies {
		mesh, err := body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return solidlens.Scene{}, fmt.Errorf("mesh %s view: %w", name, err)
		}
		models = append(models, fastenerModel(mesh))
	}
	return bevelScene(fitCamera(camera, models, 16.0/9, .1), models), nil
}

// renderFastenerReadme writes the index image and every README view.
func renderFastenerReadme(ctx context.Context, outputDir string) error {
	var views []readmeView
	for _, s := range fastenerShots {
		views = append(views, readmeView{s.shot, s.shot + ".png", 1600, 900})
		if s.shot == "hero" {
			views = append(views, readmeView{"hero", "index.png", 480, 270})
		}
	}
	byShot := map[string]fastenerShot{}
	for _, s := range fastenerShots {
		byShot[s.shot] = s
	}
	return renderReadmeViews(ctx, outputDir, views, func(ctx context.Context, shot string) (solidlens.Scene, error) {
		s, ok := byShot[shot]
		if !ok {
			return solidlens.Scene{}, fmt.Errorf("unknown fastener shot %q", shot)
		}
		return fastenerView(ctx, s)
	}, renderPNG)
}
