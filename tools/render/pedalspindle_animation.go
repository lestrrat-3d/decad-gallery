package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// This is prescribed kinematic motion, not a simulation of foot forces or flywheel inertia.
func runPedalSpindleAnimation(ctx context.Context, output string, width, height int) error {
	if output == "" || filepath.Ext(output) != ".gif" {
		return fmt.Errorf("animation requires a .gif output path")
	}
	if width == 0 {
		width = 1200
	}
	if height == 0 {
		height = 820
	}
	if width < 800 || height < 600 || width > 1600 || height > 1200 {
		return fmt.Errorf("animation dimensions must be within 800x600 and 1600x1200")
	}
	meshes, err := pedalSpindleModels(ctx)
	if err != nil {
		return err
	}
	return renderPedalSpindleAnimation(ctx, meshes, output, width, height)
}

func renderPedalSpindleAnimation(ctx context.Context, meshes []pedalSpindleMesh, output string, width, height int) error {
	p := pedalspindle.DefaultParams()
	base, err := pedalspindle.Motion(p)
	if err != nil {
		return err
	}
	p.CrankAngle = 0
	start, err := pedalspindle.Motion(p)
	if err != nil {
		return err
	}
	first, err := pedalSpindlePose(ctx, meshes, base, start, -math.Pi/3)
	if err != nil {
		return err
	}
	panelWidth, panelHeight := width/2, height-210
	var cameras [2]solidlens.Camera
	for panel := range 2 {
		models := animationModels(first, meshes, panel == 0)
		camera := solidlens.Camera{Position: solidlens.Vec{X: -90, Y: -2000, Z: 280},
			Target: solidlens.Vec{X: -90, Z: 280}, Up: solidlens.Vec{Z: 1}, FOV: 25}
		if panel == 1 {
			camera.Position = solidlens.Vec{X: 450, Y: -1800, Z: 520}
		}
		// Fit once. Every frame uses these same cameras, so fixed objects cannot drift.
		cameras[panel] = fitCamera(camera, models, float64(panelWidth)/float64(panelHeight), .09)
	}
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 17, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	defer func() { _ = face.Close() }()
	colors := append(color.Palette(nil), palette.Plan9...)
	colors[0] = solidlens.RGB(.88, .92, .95).NRGBA()
	colors[1] = color.RGBA{R: 30, G: 43, B: 54, A: 255}
	colors[255] = color.RGBA{R: 190, G: 35, B: 180, A: 255}
	colors[3] = color.White
	colors[4] = color.RGBA{G: 150, B: 80, A: 255}
	animation := &gif.GIF{LoopCount: 0, Config: image.Config{ColorModel: colors, Width: width, Height: height}}
	for frame := range 48 {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.CrankAngle = 2 * math.Pi * float64(frame) / 48
		state, err := pedalspindle.Motion(p)
		if err != nil {
			return err
		}
		posed, err := pedalSpindlePose(ctx, meshes, base, state, p.CrankAngle-math.Pi/3)
		if err != nil {
			return err
		}
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(colors[0]), image.Point{}, draw.Src)
		for panel := range 2 {
			models := animationModels(posed, meshes, panel == 0)
			scene := bevelScene(cameras[panel], models)
			var buffer bytes.Buffer
			if err := solidlens.RenderPNG(ctx, &buffer, scene, solidlens.Settings{Width: panelWidth, Height: panelHeight}); err != nil {
				return fmt.Errorf("render animation frame %d: %w", frame, err)
			}
			img, err := png.Decode(&buffer)
			if err != nil {
				return err
			}
			origin := image.Pt(panel*panelWidth, 90)
			draw.Draw(canvas, image.Rect(origin.X, origin.Y, origin.X+panelWidth, origin.Y+panelHeight), img, image.Point{}, draw.Src)
		}
		// Reserve the final entry for fixed-axis annotations so body shading cannot imitate a pivot mark.
		indexed := image.NewPaletted(canvas.Bounds(), colors[:255])
		draw.Draw(indexed, indexed.Bounds(), canvas, image.Point{}, draw.Src)
		indexed.Palette = colors
		text := func(x, y int, value string, c color.Color) {
			d := font.Drawer{Dst: indexed, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
			d.DrawString(value)
		}
		text(20, 28, "Crank-rocker: fixed frame and moving links", colors[1])
		text(20, 64, "Frame stays fixed; gear train omitted", colors[1])
		text(panelWidth+20, 64, "Oblique linkage; dashed A-B is a fixed reference", colors[1])
		for panel := range 2 {
			origin := image.Pt(panel*panelWidth, 90)
			point := func(v r3.Vec) image.Point {
				return projectAnimationPoint(cameras[panel], v, panelWidth, panelHeight).Add(origin)
			}
			a := point(r3.NewVec(state.Input.X, -116, state.Input.Z))
			b := point(r3.NewVec(state.Heel.X, -215, state.Heel.Z))
			if panel == 1 {
				dashedAnimationLine(indexed, a, b, colors[1])
			}
			c := point(r3.NewVec(state.CrankPin.X, -132, state.CrankPin.Z))
			d := point(r3.NewVec(state.Toe.X, -215, state.Toe.Z))
			for _, mark := range []struct {
				name string
				p    image.Point
			}{{"C", c}, {"D", d}} {
				animationMarker(indexed, mark.p, 5, colors[4])
				text(mark.p.X+9, mark.p.Y+17, mark.name, colors[1])
			}
			// Keep fixed-center marks visible when the crank label passes near them.
			for _, mark := range []struct {
				name string
				p    image.Point
			}{{"A", a}, {"B", b}} {
				animationMarker(indexed, mark.p, 10, colors[255])
				// An opaque label backing keeps antialiased lettering independent of moving bodies behind it.
				draw.Draw(indexed, image.Rect(mark.p.X+12, mark.p.Y-32, mark.p.X+33, mark.p.Y-9),
					image.NewUniform(colors[0]), image.Point{}, draw.Src)
				text(mark.p.X+14, mark.p.Y-12, mark.name, colors[255])
			}
		}
		text(20, height-80, "Grey: fixed frame. Blue: crank/wheel. Red: footman. Orange: treadle.", colors[1])
		text(20, height-52, "A/B: fixed bearing centers. C/D: moving bearing joints.", colors[1])
		text(20, height-24, fmt.Sprintf("Crank %03.0f deg. Constant-speed illustration; forces and inertia are not simulated.", p.CrankAngle*180/math.Pi), colors[1])
		animation.Image = append(animation.Image, indexed)
		animation.Delay = append(animation.Delay, 8)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	return writeSpindleGIF(output, animation)
}

func animationModels(posed, source []pedalSpindleMesh, frameVisible bool) []solidlens.Model {
	var models []solidlens.Model
	for _, mesh := range posed {
		model := mesh.model
		switch mesh.name {
		case "footman":
			model.Material = solidlens.Matte(solidlens.RGB(.68, .10, .06))
		case "treadle":
			model.Material = solidlens.Matte(solidlens.RGB(.86, .43, .08))
		case "heel-bracket":
			model.Material = solidlens.Matte(solidlens.RGB(.36, .39, .42))
		case "flywheel-spoke-0":
			model.Material = solidlens.Matte(solidlens.RGB(.12, .53, .72))
		}
		models = append(models, model)
	}
	if frameVisible {
		for _, mesh := range source {
			if mesh.group != "frame" || mesh.name == "heel-bracket" {
				continue
			}
			model := mesh.model
			model.Material = solidlens.Matte(solidlens.RGB(.36, .39, .42))
			models = append(models, model)
		}
	}
	return models
}

func projectAnimationPoint(camera solidlens.Camera, p r3.Vec, width, height int) image.Point {
	forward, _ := r3.Vec(camera.Target).Sub(r3.Vec(camera.Position)).Normalize()
	right, _ := forward.Cross(r3.Vec(camera.Up)).Normalize()
	up := right.Cross(forward)
	d := p.Sub(r3.Vec(camera.Position))
	focal := float64(height) / (2 * math.Tan(camera.FOV*math.Pi/360))
	return image.Pt(int(math.Round(float64(width)/2+focal*d.Dot(right)/d.Dot(forward))),
		int(math.Round(float64(height)/2-focal*d.Dot(up)/d.Dot(forward))))
}

func animationMarker(dst draw.Image, p image.Point, radius int, c color.Color) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			d2 := x*x + y*y
			if (d2 <= radius*radius && d2 >= (radius-2)*(radius-2)) || x == 0 || y == 0 {
				dst.Set(p.X+x, p.Y+y, c)
			}
		}
	}
}

func dashedAnimationLine(dst draw.Image, a, b image.Point, c color.Color) {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	steps := int(math.Hypot(dx, dy))
	for i := 0; i <= steps; i++ {
		if i%12 >= 6 {
			continue
		}
		u := float64(i) / float64(steps)
		dst.Set(a.X+int(math.Round(u*dx)), a.Y+int(math.Round(u*dy)), c)
	}
}

func writeSpindleGIF(output string, animation *gif.GIF) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".animation-*.gif")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()
	if err := gif.EncodeAll(file, animation); err != nil {
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
