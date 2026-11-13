package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
)

// galleryTile places one model's hero scene on the gallery image. The cell is a rectangle in pixels of the
// 1600 × 900 gallery image; renderGallery scales the model's drawn pixels to fill the cell, centred, so that
// every model takes the same share of the image whatever the size and shape of its scene.
type galleryTile struct {
	scene func(context.Context) (solidlens.Scene, error)
	cell  image.Rectangle
}

// galleryTiles puts the spinning wheel in a tall cell on the left and the other ten models in a grid of four
// columns beside it: the three involute gears and the bevel gear pair across the top, the planetary gearbox, cycloidal
// drive, hand-crank fan and hand-crank siren in the middle, and the escapement and spiral spring centred below.
var galleryTiles = []galleryTile{
	{spinningWheelView("hero"), image.Rect(0, 0, 560, 900)},
	{spurGearHero, image.Rect(560, 0, 820, 300)},
	{helicalGearHero, image.Rect(820, 0, 1080, 300)},
	{herringboneGearHero, image.Rect(1080, 0, 1340, 300)},
	{bevelGearHero, image.Rect(1340, 0, 1600, 300)},
	{planetaryGearboxHero, image.Rect(560, 300, 820, 600)},
	{cycloidalHero, image.Rect(820, 300, 1080, 600)},
	{handCrankFanHero, image.Rect(1080, 300, 1340, 600)},
	{handCrankSirenHero, image.Rect(1340, 300, 1600, 600)},
	{escapementHero, image.Rect(690, 600, 950, 900)},
	{spiralSpringHero, image.Rect(1210, 600, 1470, 900)},
}

const (
	galleryWidth  = 1600
	galleryHeight = 900
	// galleryMargin is the space in gallery pixels kept free on each side of a model inside its cell.
	galleryMargin = 14
	// galleryProbeWidth is the width of the first, trial render that measures a model's drawn size.
	galleryProbeWidth = 800
)

// renderGallery builds every model's hero scene, renders each to fill its cell, and writes them as one PNG. The
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
		scene, err := tile.scene(ctx)
		if err != nil {
			return fmt.Errorf("build gallery tile %d scene: %w", i, err)
		}
		cell := scaleRect(tile.cell, width)
		margin := galleryMargin * width / galleryWidth
		// A trial render measures the model's drawn size; the second render is made at the scale that fits the cell.
		probe, _, err := renderTile(ctx, scene, galleryProbeWidth)
		if err != nil {
			return fmt.Errorf("render gallery tile %d: %w", i, err)
		}
		drawn := foregroundBounds(probe, probe.At(0, 0))
		if drawn.Empty() {
			return fmt.Errorf("gallery tile %d draws nothing", i)
		}
		fit := min(float64(cell.Dx()-2*margin)/float64(drawn.Dx()), float64(cell.Dy()-2*margin)/float64(drawn.Dy()))
		rendered, background, err := renderTile(ctx, scene, int(math.Round(galleryProbeWidth*fit)))
		if err != nil {
			return fmt.Errorf("render gallery tile %d: %w", i, err)
		}
		if i == 0 {
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		}
		drawn = foregroundBounds(rendered, background)
		origin := image.Pt(cell.Min.X+(cell.Dx()-drawn.Dx())/2, cell.Min.Y+(cell.Dy()-drawn.Dy())/2)
		drawForeground(canvas, rendered, drawn, origin, background)
	}
	return writePNG(output, canvas)
}

// renderTile renders scene at a 16:9 size of the given width and returns the image and its background colour.
func renderTile(ctx context.Context, scene solidlens.Scene, width int) (image.Image, color.Color, error) {
	var buffer bytes.Buffer
	settings := solidlens.Settings{Width: width, Height: width * galleryHeight / galleryWidth}
	if err := solidlens.RenderPNG(ctx, &buffer, scene, settings); err != nil {
		return nil, nil, err
	}
	rendered, err := png.Decode(&buffer)
	if err != nil {
		return nil, nil, fmt.Errorf("decode image: %w", err)
	}
	return rendered, color.RGBAModel.Convert(rendered.At(0, 0)), nil
}

// foregroundBounds returns the smallest rectangle holding every pixel of img that differs from background.
func foregroundBounds(img image.Image, background color.Color) image.Rectangle {
	bounds := img.Bounds()
	out := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if color.RGBAModel.Convert(img.At(x, y)) == background {
				continue
			}
			out.Min.X, out.Min.Y = min(out.Min.X, x), min(out.Min.Y, y)
			out.Max.X, out.Max.Y = max(out.Max.X, x+1), max(out.Max.Y, y+1)
		}
	}
	if out.Min.X > out.Max.X {
		return image.Rectangle{}
	}
	return out
}

// scaleRect scales a rectangle given in gallery pixels to an image width.
func scaleRect(r image.Rectangle, width int) image.Rectangle {
	scale := func(v int) int { return v * width / galleryWidth }
	return image.Rect(scale(r.Min.X), scale(r.Min.Y), scale(r.Max.X), scale(r.Max.Y))
}

// drawForeground copies the pixels of src inside area that differ from background onto dst, placing the area's
// top left corner at origin.
func drawForeground(dst *image.RGBA, src image.Image, area image.Rectangle, origin image.Point, background color.Color) {
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			pixel := src.At(x, y)
			if color.RGBAModel.Convert(pixel) == background {
				continue
			}
			dst.Set(origin.X+x-area.Min.X, origin.Y+y-area.Min.Y, pixel)
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
