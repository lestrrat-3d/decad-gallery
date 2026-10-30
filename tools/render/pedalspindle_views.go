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

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var pedalSpindleSheetShots = []string{"hero-pair", "drive-pair", "mesh-pairs", "cycle", "heel-support", "coupling"}

func isPedalSpindleSheet(shot string) bool {
	for _, name := range pedalSpindleSheetShots {
		if shot == name {
			return true
		}
	}
	return false
}

func pedalSpindleSheetSize(shot string) (int, int) {
	switch shot {
	case "mesh-pairs":
		return 2000, 1400
	case "cycle":
		return 2000, 1000
	default:
		return 2000, 1100
	}
}

func runPedalSpindleSheet(ctx context.Context, shot, output string, width, height int) error {
	if output == "" {
		return fmt.Errorf("-out is required")
	}
	w, h := pedalSpindleSheetSize(shot)
	if width == 0 {
		width = w
	}
	if height == 0 {
		height = h
	}
	meshes, err := pedalSpindleModels(ctx)
	if err != nil {
		return err
	}
	return renderPedalSpindleSheet(ctx, meshes, shot, output, width, height)
}

type spindlePanel struct {
	title, caption string
	scene          solidlens.Scene
}

func renderPedalSpindleSheet(ctx context.Context, meshes []pedalSpindleMesh, shot, output string, width, height int) error {
	panels, columns, err := pedalSpindlePanels(ctx, meshes, shot)
	if err != nil {
		return err
	}
	rows := (len(panels) + columns - 1) / columns
	if width < columns*160 || height < rows*180 || width > 16384 || height > 16384 {
		return fmt.Errorf("pedal spindle sheet dimensions cannot fit its panels: %d x %d", width, height)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: solidlens.RGB(.88, .92, .95).NRGBA()}, image.Point{}, draw.Src)
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return fmt.Errorf("parse caption font: %w", err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 18, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return fmt.Errorf("create caption font: %w", err)
	}
	defer func() { _ = face.Close() }()
	pw, ph := width/columns, height/rows
	for i, panel := range panels {
		origin := image.Pt((i%columns)*pw, (i/columns)*ph)
		if shot != "coupling" {
			panel.scene.Camera = fitCamera(panel.scene.Camera, panel.scene.Models, float64(pw)/float64(ph-100), .08)
		}
		var buffer bytes.Buffer
		if err := solidlens.RenderPNG(ctx, &buffer, panel.scene, solidlens.Settings{Width: pw, Height: ph - 100}); err != nil {
			return fmt.Errorf("render %s panel %d: %w", shot, i, err)
		}
		img, err := png.Decode(&buffer)
		if err != nil {
			return fmt.Errorf("decode %s panel %d: %w", shot, i, err)
		}
		draw.Draw(canvas, image.Rect(origin.X, origin.Y+45, origin.X+pw, origin.Y+ph-55), img, image.Point{}, draw.Src)
		drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.RGBA{R: 30, G: 43, B: 54, A: 255}), Face: face}
		drawer.Dot = fixed.P(origin.X+20, origin.Y+30)
		drawer.DrawString(panel.title)
		drawer.Dot = fixed.P(origin.X+20, origin.Y+ph-24)
		drawer.DrawString(panel.caption)
	}
	return writePNG(output, canvas)
}

func pedalSpindlePanels(ctx context.Context, meshes []pedalSpindleMesh, shot string) ([]spindlePanel, int, error) {
	if shot == "coupling" {
		var gear, assembled []solidlens.Model
		for _, mesh := range meshes {
			if mesh.name == "input-gear" {
				gear = append(gear, mesh.model)
				assembled = append(assembled, mesh.model)
			}
			if mesh.name == "input-shaft" {
				assembled = append(assembled, mesh.model)
			}
		}
		target := solidlens.Vec{X: -90, Y: -36, Z: 380}
		front := solidlens.Camera{Position: solidlens.Vec{X: -90, Y: -1000, Z: 380}, Target: target, Up: solidlens.Vec{Z: 1}, FOV: 2}
		oblique := solidlens.Camera{Position: solidlens.Vec{X: 500, Y: -1100, Z: 650},
			Target: solidlens.Vec{X: -90, Y: -70, Z: 380}, Up: solidlens.Vec{Z: 1}, FOV: 5}
		return []spindlePanel{
			{"Input gear: actual D bore", "The flat transmits torque; outer teeth are cropped in this close-up", bevelScene(front, gear)},
			{"Assembled shaft and gear", "The shaft has a matching flat; its bearing journal stays round", bevelScene(oblique, assembled)},
		}, 2, nil
	}
	if shot == "heel-support" {
		var models []solidlens.Model
		for _, mesh := range meshes {
			switch mesh.name {
			case "treadle", "heel-pin", "heel-bracket-front", "heel-bracket-rear", "heel-bearing-front", "heel-bearing-rear",
				"heel-collar-front", "heel-collar-rear", "heel-spacer-front", "heel-spacer-rear":
				models = append(models, mesh.model)
			}
		}
		target := solidlens.Vec{X: -230, Y: -175, Z: 60}
		oblique := solidlens.Camera{Position: solidlens.Vec{X: -700, Y: -800, Z: 440}, Target: target, Up: solidlens.Vec{Z: 1}, FOV: 30}
		end := solidlens.Camera{Position: solidlens.Vec{X: -1600, Y: -175, Z: 60}, Target: target, Up: solidlens.Vec{Z: 1}, FOV: 30}
		return []spindlePanel{
			{"Heel: two base-mounted supports", "Two 608 bearings carry the treadle on an 8 mm through-shaft", bevelScene(oblique, models)},
			{"End view: shaft retention and pedal clearance", "Outer collars retain the shaft; inner spacers limit pedal end play", bevelScene(end, models)},
		}, 2, nil
	}
	if shot == "cycle" {
		return pedalSpindleCycle(ctx, meshes)
	}
	if shot == "mesh-pairs" {
		var panels []spindlePanel
		for stage, names := range [][]string{{"input-gear", "intermediate-pinion"}, {"intermediate-gear", "output-pinion"}} {
			var models []solidlens.Model
			for _, mesh := range meshes {
				if mesh.name == names[0] || mesh.name == names[1] {
					models = append(models, mesh.model)
				}
			}
			for _, view := range []string{"front", "oblique"} {
				x, z := -38.85, 380.0
				if stage == 1 {
					x, z = 12.3, 431.15
				}
				camera := solidlens.Camera{Position: solidlens.Vec{X: x, Y: -1400, Z: z},
					Target: solidlens.Vec{X: x, Y: -50, Z: z}, Up: solidlens.Vec{Z: 1}, FOV: 25}
				if view == "oblique" {
					camera.Position = solidlens.Vec{X: x + 500, Y: -1000, Z: z + 300}
				}
				caption := "Blue 51T -> gold 17T; rear pair at Y = -30 to -42 mm"
				if stage == 1 {
					caption = "Gold 51T -> orange 17T; front pair at Y = -60 to -72 mm"
				}
				panels = append(panels, spindlePanel{fmt.Sprintf("Stage %d: %s / 3:1 speed increase", stage+1, view),
					caption, bevelScene(camera, models)})
			}
		}
		return panels, 2, nil
	}
	base := "hero"
	if shot == "drive-pair" {
		base = "drive"
	}
	front := pedalSpindleScene(meshes, base)
	rear := front
	rear.Camera.Position.Y = 1400
	rear.Camera.Position.X = -820
	frontCaption := "Blue input; gold compound shaft; orange output"
	rearCaption := "Both gold gears share one intermediate shaft"
	if shot == "drive-pair" {
		frontCaption += "; supports and bearings omitted"
		rearCaption += "; supports and bearings omitted"
	}
	return []spindlePanel{
		{"Front: spindle and crank side", frontCaption, front},
		{"Rear: flywheel side", rearCaption, rear},
	}, 2, nil
}

// pedalSpindleCycle copies the real bodies into four solved poses. No gear or linkage is redrawn.
func pedalSpindleCycle(ctx context.Context, meshes []pedalSpindleMesh) ([]spindlePanel, int, error) {
	baseParams := pedalspindle.DefaultParams()
	base, err := pedalspindle.Motion(baseParams)
	if err != nil {
		return nil, 0, err
	}
	var panels []spindlePanel
	for i := range 4 {
		p := baseParams
		p.CrankAngle = float64(i) * math.Pi / 2
		state, err := pedalspindle.Motion(p)
		if err != nil {
			return nil, 0, err
		}
		posed, err := pedalSpindlePose(ctx, meshes, base, state, p.CrankAngle-baseParams.CrankAngle)
		if err != nil {
			return nil, 0, err
		}
		var models []solidlens.Model
		for _, mesh := range posed {
			models = append(models, mesh.model)
		}
		camera := solidlens.Camera{Position: solidlens.Vec{X: -90, Y: -2000, Z: 300},
			Target: solidlens.Vec{X: -90, Z: 300}, Up: solidlens.Vec{Z: 1}, FOV: 25}
		panels = append(panels, spindlePanel{fmt.Sprintf("%d deg: one continuous crank turn", i*90),
			fmt.Sprintf("Toe height %.1f mm; treadle only rocks", state.Toe.Z), bevelScene(camera, models)})
	}
	return panels, 4, nil
}

// pedalSpindlePose places the real linkage and bearing bodies without redrawing their geometry.
func pedalSpindlePose(ctx context.Context, meshes []pedalSpindleMesh, base, state pedalspindle.State, delta float64) ([]pedalSpindleMesh, error) {
	var posed []pedalSpindleMesh
	for _, mesh := range meshes {
		center, angle, offset, keep := cyclePlacement(mesh.name, base, state, delta)
		if !keep {
			continue
		}
		rotation, err := r3.RotationAround(center, r3.NewVec(0, -1, 0), units.Radians(angle))
		if err != nil {
			return nil, err
		}
		translation, err := r3.Translation(offset)
		if err != nil {
			return nil, err
		}
		xf, err := rotation.Then(translation)
		if err != nil {
			return nil, err
		}
		body, err := decad.Copy(ctx, mesh.body, decad.WithCopyPlacement(xf))
		if err != nil {
			return nil, fmt.Errorf("pose %s: %w", mesh.name, err)
		}
		triangles, err := body.Tessellate(ctx, units.Millimeters(.15), decad.WithVerification(decad.VerifyNone))
		if err != nil {
			return nil, err
		}
		mesh.body = body
		mesh.model.Mesh = triangles
		posed = append(posed, mesh)
	}
	return posed, nil
}

func cyclePlacement(name string, base, state pedalspindle.State, delta float64) (r3.Vec, float64, r3.Vec, bool) {
	switch name {
	case "crank", "input-shaft", "flywheel-hub", "flywheel-spoke-0", "flywheel-spoke-1", "flywheel-spoke-2", "flywheel-spoke-3":
		return base.Input, delta, r3.Vec{}, true
	case "treadle":
		return base.Heel, state.PedalAngle - base.PedalAngle, r3.Vec{}, true
	case "footman":
		oldAngle := math.Atan2(base.CrankPin.Z-base.Toe.Z, base.CrankPin.X-base.Toe.X)
		newAngle := math.Atan2(state.CrankPin.Z-state.Toe.Z, state.CrankPin.X-state.Toe.X)
		return base.Toe, newAngle - oldAngle, state.Toe.Sub(base.Toe), true
	case "crank-pin", "footman-bearing-crank":
		return r3.Vec{}, 0, state.CrankPin.Sub(base.CrankPin), true
	case "toe-pin", "footman-bearing-toe":
		return r3.Vec{}, 0, state.Toe.Sub(base.Toe), true
	case "heel-pin", "heel-bracket-front", "heel-bracket-rear", "heel-collar-front", "heel-collar-rear",
		"heel-spacer-front", "heel-spacer-rear", "flywheel-rim",
		"input-bearing-front", "input-bearing-rear", "heel-bearing-front", "heel-bearing-rear":
		return r3.Vec{}, 0, r3.Vec{}, true
	default:
		return r3.Vec{}, 0, r3.Vec{}, false
	}
}
