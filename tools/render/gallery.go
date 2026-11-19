package main

import (
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
	// name identifies the tile in the cache.
	name  string
	scene func(context.Context) (solidlens.Scene, error)
	cell  image.Rectangle
}

// galleryTiles puts the spinning wheel in a tall cell on the left and the other seventeen models in a grid of four rows beside it, filled row by row. The top row holds five 208 × 225 pixel cells: the spur, helical,
// herringbone and bevel gears and the seven-segment display. The other rows have four 260 × 225 pixel cells. The second row holds the planetary gearbox, cycloidal drive, plaque stand and clamp
// arm. The third row holds the hand-crank fan, hand-crank siren, Graham escapement and Swiss lever escapement. The
// bottom row holds the crank handle, one-way clutch, spiral spring and turbine fan.
var galleryTiles = []galleryTile{
	{"spinningwheel", spinningWheelView("hero"), image.Rect(0, 0, 560, 900)},
	{"spurgear", spurGearHero, image.Rect(560, 0, 768, 225)},
	{"helicalgear", helicalGearHero, image.Rect(768, 0, 976, 225)},
	{"herringbonegear", herringboneGearHero, image.Rect(976, 0, 1184, 225)},
	{"bevelgear", bevelGearHero, image.Rect(1184, 0, 1392, 225)},
	{"sevensegment", sevenSegmentHero, image.Rect(1392, 0, 1600, 225)},
	{"planetarygearbox", planetaryGearboxHero, image.Rect(560, 225, 820, 450)},
	{"cycloidal", cycloidalHero, image.Rect(820, 225, 1080, 450)},
	{"plaquestand", plaqueStandHero, image.Rect(1080, 225, 1340, 450)},
	{"clamparm", clampArmView("hero"), image.Rect(1340, 225, 1600, 450)},
	{"handcrankfan", handCrankFanHero, image.Rect(560, 450, 820, 675)},
	{"handcranksiren", handCrankSirenHero, image.Rect(820, 450, 1080, 675)},
	{"escapement", escapementHero, image.Rect(1080, 450, 1340, 675)},
	{"swisslever", swissHero, image.Rect(1340, 450, 1600, 675)},
	{"crankhandle", crankHandleHero, image.Rect(560, 675, 820, 900)},
	{"onewayclutch", oneWayClutchHero, image.Rect(820, 675, 1080, 900)},
	{"spiralspring", spiralSpringHero, image.Rect(1080, 675, 1340, 900)},
	{"turbinefan", turbineFanHero, image.Rect(1340, 675, 1600, 900)},
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
	cache := newGalleryCache(ctx, galleryFresh)
	return renderGalleryWithCache(ctx, output, width, height, galleryTiles, cache)
}

func renderGalleryWithCache(ctx context.Context, output string, width, height int, tiles []galleryTile, cache galleryCache) error {
	if output == "" {
		return fmt.Errorf("-out is required")
	}
	if width*galleryHeight != height*galleryWidth {
		return fmt.Errorf("gallery image must be 16:9, got %d × %d", width, height)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, tile := range tiles {
		if err := ctx.Err(); err != nil {
			return err
		}
		cell := scaleRect(tile.cell, width)
		margin := galleryMargin * width / galleryWidth
		rendered, background, ok := cache.read(tile.name, width, height, tile.cell)
		if !ok {
			scene, err := tile.scene(ctx)
			if err != nil {
				return fmt.Errorf("build gallery tile %d scene: %w", i, err)
			}
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
			rendered, background, err = renderTile(ctx, scene, int(math.Round(galleryProbeWidth*fit)))
			if err != nil {
				return fmt.Errorf("render gallery tile %d: %w", i, err)
			}
			cache.write(tile.name, width, height, tile.cell, rendered)
		}
		if i == 0 {
			draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		}
		drawn := foregroundBounds(rendered, background)
		origin := image.Pt(cell.Min.X+(cell.Dx()-drawn.Dx())/2, cell.Min.Y+(cell.Dy()-drawn.Dy())/2)
		drawForeground(canvas, rendered, drawn, origin, background)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return writePNG(output, canvas)
}

// renderTile renders scene at a 16:9 size of the given width and returns the image and its background colour.
func renderTile(ctx context.Context, scene solidlens.Scene, width int) (image.Image, color.Color, error) {
	settings := solidlens.Settings{Width: width, Height: width * galleryHeight / galleryWidth}
	rendered, err := solidlens.Render(ctx, scene, settings)
	if err != nil {
		return nil, nil, err
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
