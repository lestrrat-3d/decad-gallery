// The render tool turns gallery models into images.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
)

type preset struct {
	width, height int
	scene         func(context.Context) (solidlens.Scene, error)
}

var presets = map[string]preset{
	"screwgear/hero":          {width: 1600, height: 900, scene: screwGearView("hero")},
	"screwgear/top":           {width: 1600, height: 900, scene: screwGearView("top")},
	"screwgear/low":           {width: 1600, height: 900, scene: screwGearView("low")},
	"screwgear/cutaway":       {width: 1600, height: 900, scene: screwGearView("cutaway")},
	"spurgear/hero":           {width: 1600, height: 900, scene: spurGearHero},
	"spurgear/top":            {width: 1600, height: 900, scene: spurGearTop},
	"spurgear/low":            {width: 1600, height: 900, scene: spurGearLow},
	"bevelgear/hero":          {width: 1600, height: 900, scene: bevelGearHero},
	"bevelgear/top":           {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearTop},
	"bevelgear/low":           {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearLow},
	"bevelgear/high":          {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearHigh},
	"bevelgear/straight-top":  {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearStraightTop},
	"bevelgear/straight-low":  {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearStraightLow},
	"bevelgear/straight-high": {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearStraightHigh},
	"cycloidal/hero":          {width: 1600, height: 900, scene: cycloidalHero},
	"helicalgear/hero":        {width: 1600, height: 900, scene: helicalGearHero},
	"helicalgear/top":         {width: 1600, height: 900, scene: helicalGearTop},
	"helicalgear/low":         {width: 1600, height: 900, scene: helicalGearLow},
	"herringbonegear/hero":    {width: 1600, height: 900, scene: herringboneGearHero},
	"herringbonegear/top":     {width: 1600, height: 900, scene: herringboneGearTop},
	"herringbonegear/low":     {width: 1600, height: 900, scene: herringboneGearLow},
}

func main() {
	name := flag.String("model", "spurgear", "model directory name")
	shot := flag.String("shot", "hero", "image preset")
	output := flag.String("out", "", "PNG output path (required)")
	width := flag.Int("width", 0, "override preset width in pixels")
	height := flag.Int("height", 0, "override preset height in pixels")
	flag.Parse()
	if err := run(context.Background(), *name, *shot, *output, *width, *height); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, model, shot, output string, width, height int) error {
	if shot == "readme" {
		if width != 0 || height != 0 {
			return fmt.Errorf("readme output uses fixed image sizes")
		}
		return renderReadme(ctx, model, output)
	}
	if model == "gallery" && shot == "hero" {
		if width == 0 {
			width = galleryWidth
		}
		if height == 0 {
			height = width * galleryHeight / galleryWidth
		}
		return renderGallery(ctx, output, width, height)
	}
	selected, ok := presets[model+"/"+shot]
	if !ok {
		return fmt.Errorf("unknown image preset %q for model %q", shot, model)
	}
	if output == "" {
		return fmt.Errorf("-out is required")
	}
	if width < 0 || height < 0 {
		return fmt.Errorf("image dimensions must be positive")
	}
	if width == 0 {
		width = selected.width
	}
	if height == 0 {
		height = selected.height
	}
	scene, err := selected.scene(ctx)
	if err != nil {
		return fmt.Errorf("build %s/%s scene: %w", model, shot, err)
	}
	return renderPNG(ctx, output, scene, width, height)
}

// renderReadme builds a model once and writes its index image and extra views.
func renderReadme(ctx context.Context, model, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("-out is required")
	}
	if model == "bevelgear" {
		return renderBevelReadme(ctx, outputDir)
	}
	if model == "cycloidal" {
		return renderCycloidalReadme(ctx, outputDir)
	}
	if model == "screwgear" {
		return renderScrewGearReadme(ctx, outputDir)
	}
	hero, ok := presets[model+"/hero"]
	if !ok {
		return fmt.Errorf("unknown model %q", model)
	}
	scene, err := hero.scene(ctx)
	if err != nil {
		return fmt.Errorf("build %s/hero scene: %w", model, err)
	}
	if err := renderPNG(ctx, filepath.Join(outputDir, "index.png"), scene, 480, 270); err != nil {
		return fmt.Errorf("render %s index image: %w", model, err)
	}
	var cameras map[string]solidlens.Camera
	switch model {
	case "helicalgear":
		cameras = helicalCameras
	case "herringbonegear":
		cameras = herringboneCameras
	}
	for _, shot := range []string{"top", "low", "high"} {
		camera, ok := cameras[shot]
		if !ok {
			continue
		}
		scene.Camera = camera
		if err := renderPNG(ctx, filepath.Join(outputDir, shot+".png"), scene, 1600, 900); err != nil {
			return fmt.Errorf("render %s/%s image: %w", model, shot, err)
		}
	}
	return nil
}

func renderPNG(ctx context.Context, output string, scene solidlens.Scene, width, height int) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".render-*.png")
	if err != nil {
		return fmt.Errorf("create temporary image: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()
	if err := solidlens.RenderPNG(ctx, file, scene, solidlens.Settings{Width: width, Height: height}); err != nil {
		return fmt.Errorf("render PNG: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close rendered PNG: %w", err)
	}
	if err := os.Chmod(file.Name(), 0o644); err != nil {
		return fmt.Errorf("set rendered PNG permissions: %w", err)
	}
	if err := os.Rename(file.Name(), output); err != nil {
		return fmt.Errorf("save rendered PNG: %w", err)
	}
	return nil
}
