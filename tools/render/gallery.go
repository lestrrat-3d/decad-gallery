package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
)

// galleryTile places one model's hero scene on the gallery image. The rectangle is in pixels of the
// 1600 × 900 gallery image; the scene is rendered at the rectangle's size.
type galleryTile struct {
	scene func(context.Context) (solidlens.Scene, error)
	area  image.Rectangle
}

// galleryTiles puts the three involute gears in a row above the cycloidal drive and the bevel gear pairs. The
// gear tiles overlap by their empty margins; renderGallery draws only the pixels that differ from the background.
var galleryTiles = []galleryTile{
	{spurGearHero, image.Rect(-48, 15, 688, 429)},
	{helicalGearHero, image.Rect(432, 15, 1168, 429)},
	{herringboneGearHero, image.Rect(912, 15, 1648, 429)},
	{cycloidalHero, image.Rect(0, 430, 800, 880)},
	{bevelGearHero, image.Rect(800, 430, 1600, 880)},
}

const (
	galleryWidth  = 1600
	galleryHeight = 900
)

// renderGallery builds every model's hero scene, renders each as a tile, and writes them as one PNG. The
// layout scales with width; height must keep the 16:9 shape of the tiles.
func renderGallery(ctx context.Context, output string, width, height int) error {
	if output == "" {
		return fmt.Errorf("-out is required")
	}
	if width*galleryHeight != height*galleryWidth {
		return fmt.Errorf("gallery image must be 16:9, got %d × %d", width, height)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, tile := range galleryTiles {
		area := scaleRect(tile.area, width)
		scene, err := tile.scene(ctx)
		if err != nil {
			return fmt.Errorf("build gallery tile %d scene: %w", i, err)
		}
		var buffer bytes.Buffer
		settings := solidlens.Settings{Width: area.Dx(), Height: area.Dy()}
		if err := solidlens.RenderPNG(ctx, &buffer, scene, settings); err != nil {
			return fmt.Errorf("render gallery tile %d: %w", i, err)
		}
		rendered, err := png.Decode(&buffer)
		if err != nil {
			return fmt.Errorf("decode gallery tile %d: %w", i, err)
		}
		background := color.RGBAModel.Convert(rendered.At(0, 0))
		if i == 0 {
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		}
		drawForeground(canvas, rendered, area.Min, background)
	}
	return writePNG(output, canvas)
}

// scaleRect scales a rectangle given in gallery pixels to an image width.
func scaleRect(r image.Rectangle, width int) image.Rectangle {
	scale := func(v int) int { return v * width / galleryWidth }
	return image.Rect(scale(r.Min.X), scale(r.Min.Y), scale(r.Max.X), scale(r.Max.Y))
}

// drawForeground copies the pixels of src that differ from background onto dst at origin.
func drawForeground(dst *image.RGBA, src image.Image, origin image.Point, background color.Color) {
	bounds := src.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := src.At(x, y)
			if pixel == background {
				continue
			}
			dst.Set(origin.X+x-bounds.Min.X, origin.Y+y-bounds.Min.Y, pixel)
		}
	}
}

// writePNG writes img to output through a temporary file, as renderPNG does.
func writePNG(output string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".render-*.png")
	if err != nil {
		return fmt.Errorf("create temporary image: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close PNG: %w", err)
	}
	if err := os.Chmod(file.Name(), 0o644); err != nil {
		return fmt.Errorf("set PNG permissions: %w", err)
	}
	if err := os.Rename(file.Name(), output); err != nil {
		return fmt.Errorf("save PNG: %w", err)
	}
	return nil
}
