package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math"
	"path/filepath"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
)

// fanAnimationFrames is the number of frames in one crank turn. The 32-tooth gear has a tooth pitch of 11.25°, so a
// step well below it keeps the teeth from seeming to run backward. It is a variable so that the -frames flag can
// shorten the animation for tests.
var fanAnimationFrames = 72

// handCrankFanAnimation turns the crank one full turn, and so the fan two turns, in two panels: the crank side on the
// left and the gear box from above and behind on the right. The meshes come from one build of the real fan; each frame
// turns the crank parts about the crank axis and the fan parts about the fan axis by rigid transforms, as the model's
// own pose does.
func renderHandCrankFanAnimation(ctx context.Context, out string, width, height, frames int) error {
	if filepath.Ext(out) != ".gif" {
		return fmt.Errorf("animation requires a .gif output path")
	}
	if frames < 2 {
		return fmt.Errorf("animation needs at least two frames")
	}
	doc := handcrankfan.NewDocument()
	a, err := handcrankfan.Build(ctx, doc, handcrankfan.DefaultParams())
	if err != nil {
		return fmt.Errorf("build hand-crank fan: %w", err)
	}
	meshes, err := handCrankFanMeshes(ctx, a)
	if err != nil {
		return err
	}
	face, err := wheelCaptionFont(14)
	if err != nil {
		return err
	}
	defer func() { _ = face.Close() }()

	panelWidth, panelHeight := width/2, height-110
	aspect := float64(panelWidth) / float64(panelHeight)
	var box, upper []solidlens.Model
	for _, mesh := range meshes {
		switch mesh.part.Name {
		case "gearbox", "driving-gear", "pinion":
			box = append(box, mesh.model)
		}
		if mesh.part.Name != "handle" {
			upper = append(upper, mesh.model)
		}
	}
	// The crank panel must hold the arm's whole sweep, so it is fitted to four poses a quarter turn apart.
	for _, turn := range []float64{math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
		posed, err := handCrankFanPose(meshes, a.State, turn)
		if err != nil {
			return err
		}
		for i, mesh := range meshes {
			if mesh.part.Name != "handle" {
				upper = append(upper, posed[i])
			}
		}
	}
	crankSide := fitCamera(solidlens.Camera{Position: solidlens.Vec{X: 700, Y: -260, Z: 250},
		Target: solidlens.Vec{X: 20, Y: -45, Z: 175}, Up: solidlens.Vec{Z: 1}, FOV: 30}, upper, aspect, .04)
	gearSide := fitCamera(solidlens.Camera{Position: solidlens.Vec{X: -300, Y: 380, Z: 560},
		Target: solidlens.Vec{X: 5, Y: -10, Z: 168}, Up: solidlens.Vec{Z: 1}, FOV: 30}, box, aspect, .12)

	animation := &gif.GIF{LoopCount: 0, Config: image.Config{ColorModel: color.Palette(palette.Plan9), Width: width, Height: height}}
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		turn := 2 * math.Pi * float64(frame) / float64(frames)
		posed, err := handCrankFanPose(meshes, a.State, turn)
		if err != nil {
			return err
		}
		panels := make([]wheelPanel, 2)
		for i, camera := range []solidlens.Camera{crankSide, gearSide} {
			panels[i].scene = bevelScene(camera, posed)
		}
		crank := math.Mod(turn*180/math.Pi, 360)
		panels[0].title = "Crank side: one turn of the arm"
		panels[0].caption = fmt.Sprintf("Crank %03.0f deg. The fan turns twice as fast.", crank)
		panels[1].title = "Gears: the 32-tooth gear drives the 16-tooth pinion"
		panels[1].caption = fmt.Sprintf("Gear %03.0f deg, pinion and fan %03.0f deg.", crank, math.Mod(2*crank, 360))
		canvas, err := wheelPanelCanvas(ctx, panels, width, height, face)
		if err != nil {
			return err
		}
		indexed := image.NewPaletted(canvas.Bounds(), palette.Plan9)
		draw.Draw(indexed, indexed.Bounds(), canvas, image.Point{}, draw.Src)
		animation.Image = append(animation.Image, indexed)
		animation.Delay = append(animation.Delay, 5)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	return writeWheelGIF(out, animation)
}

// handCrankFanPose turns the crank parts about the crank axis through the apex by turn, and the fan parts about the
// fan axis by twice that, from the pose the meshes were built at. The fixed parts stay.
func handCrankFanPose(meshes []fanMesh, state handcrankfan.State, turn float64) ([]solidlens.Model, error) {
	crank, err := r3.RotationAround(state.Apex, r3.Vec{X: -1}, units.Radians(turn))
	if err != nil {
		return nil, fmt.Errorf("turn the crank: %w", err)
	}
	fan, err := r3.RotationAround(state.Apex, r3.Vec{Y: -1}, units.Radians(handcrankfan.Ratio*turn))
	if err != nil {
		return nil, fmt.Errorf("turn the fan: %w", err)
	}
	posed := make([]solidlens.Model, len(meshes))
	for i, mesh := range meshes {
		posed[i] = mesh.model
		var xf r3.Transform
		switch mesh.part.Motion {
		case "crank":
			xf = crank
		case "fan":
			xf = fan
		default:
			continue
		}
		vertices := mesh.model.Mesh.Vertices()
		for j, v := range vertices {
			vertices[j] = xf.Apply(v)
		}
		triangles, err := solidlens.NewMesh(vertices, mesh.model.Mesh.Triangles())
		if err != nil {
			return nil, fmt.Errorf("pose %s: %w", mesh.part.Name, err)
		}
		posed[i].Mesh = triangles
	}
	return posed, nil
}
