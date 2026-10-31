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

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Rigidly transform vertices tessellated from the actual Decad bodies.
// No replacement linkage or flyer geometry is drawn for the animation.
func spinningWheelPose(meshes []wheelMesh, base, state spinningwheel.State, wheelDelta float64) ([]wheelMesh, error) {
	posed := make([]wheelMesh, len(meshes))
	for i, mesh := range meshes {
		posed[i] = mesh
		if mesh.part.Motion == "fixed" {
			continue
		}
		xf, err := spinningwheel.PoseTransform(base, state, mesh.part.Motion, wheelDelta)
		if err != nil {
			return nil, err
		}
		vertices := mesh.model.Mesh.Vertices()
		for j, v := range vertices {
			vertices[j] = xf.Apply(v)
		}
		triangles, err := solidlens.NewMesh(vertices, mesh.model.Mesh.Triangles())
		if err != nil {
			return nil, err
		}
		posed[i].model.Mesh = triangles
	}
	return posed, nil
}

func renderSpinningWheelAnimation(ctx context.Context, meshes []wheelMesh, base spinningwheel.State, out string, width, height, frames int) error {
	if filepath.Ext(out) != ".gif" {
		return fmt.Errorf("animation requires a .gif output path")
	}
	p := spinningwheel.DefaultParams()
	// One relative bobbin turn per wheel turn makes the GIF loop without a phase jump.
	p.YarnFeedPerTurn = 2 * math.Pi * 31
	face, err := wheelCaptionFont(14)
	if err != nil {
		return err
	}
	defer func() { _ = face.Close() }()
	var cameras [2]solidlens.Camera
	for i, shot := range []string{"hero", "flyer"} {
		scene := spinningWheelScene(meshes, shot)
		cameras[i] = fitCamera(scene.Camera, scene.Models, float64(width/2)/float64(height-110), .14)
	}
	animation := &gif.GIF{LoopCount: 0, Config: image.Config{ColorModel: color.Palette(palette.Plan9), Width: width, Height: height}}
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.WheelAngle = 2 * math.Pi * float64(frame) / float64(frames)
		state, err := spinningwheel.Motion(p)
		if err != nil {
			return err
		}
		posed, err := spinningWheelPose(meshes, base, state, p.WheelAngle-math.Pi/3)
		if err != nil {
			return err
		}
		for i := range posed {
			if posed[i].part.Motion != "fixed" {
				continue
			}
			switch posed[i].part.Group {
			case "frame", "head-frame", "adjustment":
				posed[i].model.Material = solidlens.Matte(solidlens.RGB(.45, .47, .49))
			}
		}
		panels := make([]wheelPanel, 2)
		for i, shot := range []string{"hero", "flyer"} {
			scene := spinningWheelScene(posed, shot)
			scene.Camera = cameras[i]
			panels[i].scene = scene
		}
		panels[0].title = "Grey frame stays fixed; the treadle rocks."
		panels[0].caption = fmt.Sprintf("Wheel %03.0f deg. A/B fixed axes; C/D joints; S gimbal.", p.WheelAngle*180/math.Pi)
		panels[1].title = "Red flyer: 7 turns. Orange bobbin marker: 6 turns."
		panels[1].caption = "Prescribed take-up: 195 mm/turn. Forces are not simulated."
		canvas, err := wheelPanelCanvas(ctx, panels, width, height, face)
		if err != nil {
			return err
		}
		annotateWheelPivots(canvas, cameras[0], state, width/2, height-110, face)
		indexed := image.NewPaletted(canvas.Bounds(), palette.Plan9)
		draw.Draw(indexed, indexed.Bounds(), canvas, image.Point{}, draw.Src)
		animation.Image = append(animation.Image, indexed)
		animation.Delay = append(animation.Delay, 8)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	return writeSpindleGIF(out, animation)
}

func annotateWheelPivots(canvas draw.Image, camera solidlens.Camera, state spinningwheel.State, width, height int, face font.Face) {
	// These are the actual three-dimensional joint positions. The heel and wheel axes are perpendicular.
	wheelAxis := state.Wheel
	wheelAxis.Y = state.CrankPin.Y
	position := func(p r3.Vec) image.Point {
		return projectAnimationPoint(camera, p, width, height).Add(image.Pt(0, 50))
	}
	fixedColor := color.RGBA{R: 15, G: 80, B: 210, A: 255}
	movingColor := color.RGBA{R: 0, G: 140, B: 65, A: 255}
	dashedAnimationLine(canvas, position(wheelAxis), position(state.Heel), fixedColor)
	dashedAnimationLine(canvas, position(wheelAxis.Add(r3.Vec{Y: -35})), position(wheelAxis.Add(r3.Vec{Y: 35})), fixedColor)
	dashedAnimationLine(canvas, position(state.Heel.Add(r3.Vec{X: -55})), position(state.Heel.Add(r3.Vec{X: 55})), fixedColor)
	for _, mark := range []struct {
		name  string
		point r3.Vec
		ink   color.Color
	}{
		{"C", state.CrankPin, movingColor}, {"D", state.Toe, movingColor},
		{"S", state.RodHead, movingColor},
		{"A", wheelAxis, fixedColor}, {"B", state.Heel, fixedColor},
	} {
		p := position(mark.point)
		animationMarker(canvas, p, 7, mark.ink)
		draw.Draw(canvas, image.Rect(p.X+8, p.Y-21, p.X+26, p.Y-2), image.NewUniform(color.White), image.Point{}, draw.Src)
		drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(mark.ink), Face: face, Dot: fixed.P(p.X+10, p.Y-6)}
		drawer.DrawString(mark.name)
	}
}

func runSpinningWheelAnimation(ctx context.Context, out string, width, height int) error {
	if width == 0 {
		width = 1200
	}
	if height == 0 {
		height = 800
	}
	if width < 800 || height < 600 || width > 1600 || height > 1200 {
		return fmt.Errorf("animation dimensions must be within 800x600 and 1600x1200")
	}
	meshes, base, err := spinningWheelModels(ctx)
	if err != nil {
		return err
	}
	return renderSpinningWheelAnimation(ctx, meshes, base, out, width, height, 96)
}
