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
	"spurgear/hero":  {width: 1600, height: 900, scene: spurGearHero},
	"spurgear/top":   {width: 1600, height: 900, scene: spurGearTop},
	"spurgear/low":   {width: 1600, height: 900, scene: spurGearLow},
	"bevelgear/hero": {width: 1600, height: 900, scene: bevelGearHero},
	"bevelgear/top":  {width: 1600, height: 900, scene: bevelGearTop},
	"bevelgear/low":  {width: 1600, height: 900, scene: bevelGearLow},
	"bevelgear/high": {width: 1600, height: 900, scene: bevelGearHigh},
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
