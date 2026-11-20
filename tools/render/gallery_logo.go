package main

import (
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/DejaVuSerif.ttf
var galleryLogoFont []byte

// renderGalleryHero adds the centered wordmark after rendering the Decad bodies.
func renderGalleryHero(ctx context.Context, output string, scene solidlens.Scene, width, height int) error {
	canvas, err := solidlens.Render(ctx, scene, solidlens.Settings{Width: width, Height: height})
	if err != nil {
		return fmt.Errorf("render gallery scene: %w", err)
	}
	if err := drawGalleryLogo(canvas); err != nil {
		return fmt.Errorf("draw gallery logo: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".render-*.png")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, canvas); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Chmod(file.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(file.Name(), output)
}

func drawGalleryLogo(canvas *image.RGBA) error {
	parsed, err := opentype.Parse(galleryLogoFont)
	if err != nil {
		return err
	}
	scale := float64(canvas.Bounds().Dx()) / 1600
	mainFace, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 146 * scale, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer func() { _ = mainFace.Close() }()
	subFace, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 44 * scale, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer func() { _ = subFace.Close() }()
	ink := image.NewUniform(color.RGBA{R: 83, G: 57, B: 37, A: 255})
	center := canvas.Bounds().Dx() / 2
	title := "DECAD"
	titleWidth := font.MeasureString(mainFace, title).Round()
	d := font.Drawer{Dst: canvas, Src: ink, Face: mainFace, Dot: fixed.P(center-titleWidth/2, int(425*scale))}
	d.DrawString(title)
	subtitle := []rune("GALLERY")
	tracking := int(10 * scale)
	width := 0
	for _, r := range subtitle {
		width += font.MeasureString(subFace, string(r)).Round()
	}
	width += tracking * (len(subtitle) - 1)
	x := center - width/2
	d.Face = subFace
	for _, r := range subtitle {
		d.Dot = fixed.P(x, int(492*scale))
		d.DrawString(string(r))
		x += font.MeasureString(subFace, string(r)).Round() + tracking
	}
	lineY := int(476 * scale)
	lineHeight := max(1, int(2*scale))
	for _, line := range [][2]int{{int(480 * scale), center - width/2 - int(30*scale)}, {center + width/2 + int(30*scale), int(1120 * scale)}} {
		draw.Draw(canvas, image.Rect(line[0], lineY, line[1], lineY+lineHeight), ink, image.Point{}, draw.Src)
	}
	return nil
}
