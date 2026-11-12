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
	"spinningwheel/hero":          {width: 1600, height: 900, scene: spinningWheelView("hero")},
	"spinningwheel/rear":          {width: 1600, height: 900, scene: spinningWheelView("rear")},
	"handcrankfan/hero":           {width: 1600, height: 900, scene: handCrankFanHero},
	"handcrankfan/exploded":       {width: 1600, height: 900, scene: handCrankFanExploded},
	"ballbearing/hero":            {width: 1600, height: 900, scene: bearing608Hero},
	"ballbearing/605zz":           {width: 1600, height: 900, scene: bearing605Hero},
	"m2nut/hero":                  {width: 1600, height: 900, scene: m2NutHero},
	"m2nut/top":                   {width: 1600, height: 900, scene: m2NutViewScene("top")},
	"m2nut/side":                  {width: 1600, height: 900, scene: m2NutViewScene("side")},
	"m2nut/low":                   {width: 1600, height: 900, scene: m2NutViewScene("low")},
	"m2nut/withbolt":              {width: 1600, height: 900, scene: m2NutViewScene("withbolt")},
	"pinring/hero":                {width: 1600, height: 900, scene: pinRingHero},
	"pinring/top":                 {width: 1600, height: 900, scene: pinRingViewScene("top")},
	"pinring/side":                {width: 1600, height: 900, scene: pinRingViewScene("side")},
	"pinring/low":                 {width: 1600, height: 900, scene: pinRingViewScene("low")},
	"m2bolt/hero":                 {width: 1600, height: 900, scene: m2BoltHero},
	"m2bolt/head":                 {width: 1600, height: 900, scene: m2BoltViewScene("head")},
	"m2bolt/side":                 {width: 1600, height: 900, scene: m2BoltViewScene("side")},
	"m2bolt/underside":            {width: 1600, height: 900, scene: m2BoltViewScene("underside")},
	"m2bolt/lengths":              {width: 1600, height: 900, scene: m2BoltViewScene("lengths")},
	"spiralspring/hero":           {width: 1600, height: 900, scene: spiralSpringHero},
	"spiralspring/top":            {width: 1600, height: 900, scene: spiralSpringViewScene("top")},
	"spiralspring/low":            {width: 1600, height: 900, scene: spiralSpringViewScene("low")},
	"spiralspring/root":           {width: 1600, height: 900, scene: spiralSpringViewScene("root")},
	"spiralspring/sizes":          {width: 1600, height: 900, scene: spiralSpringViewScene("sizes")},
	"screwgear/hero":              {width: 1600, height: 900, scene: screwGearView("hero")},
	"screwgear/top":               {width: 1600, height: 900, scene: screwGearView("top")},
	"screwgear/low":               {width: 1600, height: 900, scene: screwGearView("low")},
	"screwgear/cutaway":           {width: 1600, height: 900, scene: screwGearView("cutaway")},
	"screwgear/exploded":          {width: 1600, height: 900, scene: screwGearView("exploded")},
	"spurgear/hero":               {width: 1600, height: 900, scene: spurGearHero},
	"spurgear/top":                {width: 1600, height: 900, scene: spurGearTop},
	"spurgear/low":                {width: 1600, height: 900, scene: spurGearLow},
	"bevelgear/hero":              {width: 1600, height: 900, scene: bevelGearHero},
	"bevelgear/top":               {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearTop},
	"bevelgear/low":               {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearLow},
	"bevelgear/high":              {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearHigh},
	"bevelgear/straight-top":      {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearStraightTop},
	"bevelgear/straight-low":      {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearStraightLow},
	"bevelgear/straight-high":     {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearStraightHigh},
	"bevelgear/exploded-spiral":   {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearExplodedSpiral},
	"bevelgear/exploded-straight": {width: bevelViewWidth, height: bevelViewHeight, scene: bevelGearExplodedStraight},
	"cycloidal/hero":              {width: 1600, height: 900, scene: cycloidalHero},
	"cycloidal/exploded":          {width: 1600, height: 900, scene: cycloidalExploded},
	"planetarygearbox/hero":       {width: 1600, height: 900, scene: planetaryGearboxHero},
	"helicalgear/hero":            {width: 1600, height: 900, scene: helicalGearHero},
	"helicalgear/top":             {width: 1600, height: 900, scene: helicalGearTop},
	"helicalgear/low":             {width: 1600, height: 900, scene: helicalGearLow},
	"herringbonegear/hero":        {width: 1600, height: 900, scene: herringboneGearHero},
	"herringbonegear/top":         {width: 1600, height: 900, scene: herringboneGearTop},
	"herringbonegear/low":         {width: 1600, height: 900, scene: herringboneGearLow},
}

func main() {
	name := flag.String("model", "spurgear", "model directory name")
	shot := flag.String("shot", "hero", "image preset")
	output := flag.String("out", "", "PNG output path (required)")
	width := flag.Int("width", 0, "override preset width in pixels")
	height := flag.Int("height", 0, "override preset height in pixels")
	frames := flag.Int("frames", 0, "frames in the hand-crank fan animation, 0 for the preset")
	flag.Parse()
	if *frames > 0 {
		fanAnimationFrames = *frames
	}
	if err := run(context.Background(), *name, *shot, *output, *width, *height); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, model, shot, output string, width, height int) error {
	if model == "spinningwheel-preview" {
		return renderSpinningWheelPreview(ctx, shot, output, width, height)
	}
	if shot == "readme" {
		if width != 0 || height != 0 {
			return fmt.Errorf("readme output uses fixed image sizes")
		}
		return renderReadme(ctx, model, output)
	}
	if model == "spinningwheel" {
		switch shot {
		case "front-back", "layout", "flyer", "brake", "whorl":
			return runSpinningWheelSheet(ctx, shot, output, width, height)
		case "animation":
			return runSpinningWheelAnimation(ctx, output, width, height)
		}
	}
	if model == "handcrankfan" && shot == "animation" {
		if width == 0 {
			width = 1000
		}
		if height == 0 {
			height = 560
		}
		return renderHandCrankFanAnimation(ctx, output, width, height, fanAnimationFrames)
	}
	if model == "planetarygearbox" && shot == "motion" {
		if width == 0 {
			width = planetaryMotionWidth
		}
		if height == 0 {
			height = planetaryMotionHeight
		}
		return renderPlanetaryMotion(ctx, output, width, height, planetaryMotionFrames)
	}
	if model == "escapement" && shot == "motion" {
		if width == 0 {
			width = escapementMotionWidth
		}
		if height == 0 {
			height = escapementMotionHeight
		}
		return renderEscapementMotion(ctx, output, width, height, escapementMotionFrames)
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
	if model == "spinningwheel" {
		return renderSpinningWheelReadme(ctx, outputDir)
	}
	if model == "bevelgear" {
		return renderBevelReadme(ctx, outputDir)
	}
	if model == "cycloidal" {
		return renderCycloidalReadme(ctx, outputDir)
	}
	if model == "planetarygearbox" {
		return renderPlanetaryGearboxReadme(ctx, outputDir)
	}
	if model == "screwgear" {
		return renderScrewGearReadme(ctx, outputDir)
	}
	if model == "handcrankfan" {
		return renderHandCrankFanReadme(ctx, outputDir)
	}
	if model == "handcranksiren" {
		return renderHandCrankSirenReadme(ctx, outputDir)
	}
	if model == "escapement" {
		return renderEscapementReadme(ctx, outputDir)
	}
	if model == "m2nut" {
		return renderM2NutReadme(ctx, outputDir)
	}
	if model == "pinring" {
		return renderPinRingReadme(ctx, outputDir)
	}
	if model == "m2bolt" {
		return renderM2BoltReadme(ctx, outputDir)
	}
	if model == "spiralspring" {
		return renderSpiralSpringReadme(ctx, outputDir)
	}
	if model == "ballbearing" {
		return renderBearingReadme(ctx, outputDir)
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
