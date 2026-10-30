package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type wheelPanel struct {
	scene          solidlens.Scene
	title, caption string
}

func wheelCaptionFont(size float64) (font.Face, error) {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func spinningWheelPanels(meshes []wheelMesh, shot string) ([]wheelPanel, error) {
	switch shot {
	case "whorl":
		for _, mesh := range meshes {
			if mesh.part.Name != "flyer-drive-whorl" {
				continue
			}
			box, err := mesh.part.Body.Bounds()
			if err != nil {
				return nil, err
			}
			centre := box.Min.Add(box.Max).Scale(.5)
			oblique := solidlens.Camera{Position: centre.Add(solidlens.Vec{X: -240, Y: 400, Z: 220}), Target: centre, Up: solidlens.Vec{Z: 1}, FOV: 30}
			boreTarget := centre
			boreTarget.Y = box.Max.Y
			bore := solidlens.Camera{Position: boreTarget.Add(solidlens.Vec{Y: 470}), Target: boreTarget, Up: solidlens.Vec{Z: 1}, FOV: 2}
			models := []solidlens.Model{mesh.model}
			return []wheelPanel{
				{bevelScene(oblique, models), "One-piece whorl: four V grooves", "The grooves give 7:1, 9:1, 12:1 and 17:1 speed ratios."},
				{bevelScene(bore, models), "Actual D-shaped through-bore", "The flat mates with the 8 mm shaft; this close-up crops the rim."},
			}, nil
		}
		return nil, fmt.Errorf("whorl sheet needs the actual flyer-drive-whorl body")
	case "front-back":
		return []wheelPanel{
			{spinningWheelScene(meshes, "hero"), "Operator side: treadle, wheel and flyer", "The open frame carries both sides of the wheel axle."},
			{spinningWheelScene(meshes, "rear"), "Rear: drive band and adjustable mother-of-all", "The single band drives the keyed flyer whorl; the bobbin turns independently."},
		}, nil
	case "flyer":
		return []wheelPanel{
			{spinningWheelScene(meshes, "flyer"), "Flyer and bobbin: two different rotations", "Red flyer guides yarn; teal yarn pack turns with the bobbin."},
			{spinningWheelScene(meshes, "orifice"), "Orifice: actual hollow shaft, clipped open", "The leader enters the 10 mm bore and exits through the radial port."},
		}, nil
	case "brake":
		return []wheelPanel{
			{spinningWheelScene(meshes, "brake"), "Scotch tension: band around the bobbin's rear flange", "The stationary spring and adjusting screw pull the two brake-band tails."},
			{spinningWheelScene(meshes, "flyer"), "Flyer arms stop before the brake band", "The bobbin brake creates drag; it does not lock the bobbin to the flyer."},
		}, nil
	}
	return nil, fmt.Errorf("unknown spinning wheel sheet %q", shot)
}

func wheelPanelCanvas(ctx context.Context, panels []wheelPanel, width, height int, face font.Face) (*image.RGBA, error) {
	if len(panels) != 2 || width < 640 || height < 360 || width > 4000 || height > 2400 {
		return nil, fmt.Errorf("wheel sheet needs two panels and dimensions within 640x360 and 4000x2400")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(solidlens.RGB(.88, .92, .95).NRGBA()), image.Point{}, draw.Src)
	pw, ph := width/2, height-110
	for i, panel := range panels {
		var buffer bytes.Buffer
		if err := solidlens.RenderPNG(ctx, &buffer, panel.scene, solidlens.Settings{Width: pw, Height: ph}); err != nil {
			return nil, err
		}
		img, err := png.Decode(&buffer)
		if err != nil {
			return nil, err
		}
		draw.Draw(canvas, image.Rect(i*pw, 50, (i+1)*pw, 50+ph), img, image.Point{}, draw.Src)
		drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.RGBA{R: 26, G: 38, B: 46, A: 255}), Face: face}
		drawer.Dot = fixed.P(i*pw+16, 30)
		drawer.DrawString(panel.title)
		drawer.Dot = fixed.P(i*pw+16, height-24)
		drawer.DrawString(panel.caption)
	}
	return canvas, nil
}

func renderSpinningWheelSheet(ctx context.Context, meshes []wheelMesh, shot, out string, width, height int) error {
	panels, err := spinningWheelPanels(meshes, shot)
	if err != nil {
		return err
	}
	for i := range panels {
		if shot == "whorl" && i == 1 {
			continue
		}
		panels[i].scene.Camera = fitCamera(panels[i].scene.Camera, panels[i].scene.Models, float64(width/2)/float64(height-110), .08)
	}
	face, err := wheelCaptionFont(18)
	if err != nil {
		return err
	}
	defer func() { _ = face.Close() }()
	canvas, err := wheelPanelCanvas(ctx, panels, width, height, face)
	if err != nil {
		return err
	}
	return writePNG(out, canvas)
}

func runSpinningWheelSheet(ctx context.Context, shot, out string, width, height int) error {
	if out == "" {
		return fmt.Errorf("-out is required")
	}
	if width == 0 {
		width = 2000
	}
	if height == 0 {
		height = 1100
	}
	meshes, _, err := spinningWheelModels(ctx)
	if err != nil {
		return err
	}
	return renderSpinningWheelSheet(ctx, meshes, shot, out, width, height)
}
