package render

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"strings"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font/basicfont"
)

// Seven-segment animation: the camshaft turns once, 3 degrees a frame, at 960 × 540 pixels. The left panel is the face seen at an angle
// and the right the cutaway through the axis. The sliders follow the cams by Dimensions.Lift.
const (
	sevenMotionFrames = 120
	sevenMotionWidth  = 960
	sevenMotionHeight = 540
	sevenMotionDelay  = 4
)

// sevenMotionModels returns the models of the set at camshaft angle theta, the camshaft parts turned and the sliders
// lifted from the pose they were built in (the default digit).
func sevenMotionModels(s partSet, dm sevensegment.Dimensions, theta float64, top bool) ([]solidlens.Model, error) {
	turn, err := r3.RotationAround(r3.Vec{Z: sevensegment.ShaftZ}, r3.Vec{Y: 1}, units.Degrees(theta-dm.Theta))
	if err != nil {
		return nil, err
	}
	var out []solidlens.Model
	for _, p := range s {
		isSlider := strings.HasPrefix(p.name, "Slider ")
		onShaft := p.name == "Shaft" || strings.HasPrefix(p.name, "Cam ") || p.name == "Ratchet Wheel" ||
			p.name == "Collar Ring" || p.name == "Knob"
		isFlag := strings.HasPrefix(p.name, "Flag ")
		if top && !isFlag && p.name != "Top Plate" && !sevenFaceBolt(p.name) {
			continue
		}
		if !top && !onShaft && renderkit.ClipRemovesAll(p.mesh, sevenAxisCut) {
			continue // wholly on the removed side of the cut
		}
		var src solidlens.TriangleSource = p.mesh
		switch {
		case onShaft:
			src = renderkit.TransformedMesh{TriangleSource: p.mesh, Transform: turn}
		case isFlag:
			seg := sevenFlagSegment(p.name)
			turn, err := sevenFlagMove(seg, dm.FlagAngle(seg, theta)-dm.FlagAngle(seg, dm.Theta))
			if err != nil {
				return nil, err
			}
			src = renderkit.TransformedMesh{TriangleSource: p.mesh, Transform: turn}
		case isSlider:
			seg := sevensegment.Segments()[p.name[len(p.name)-1]-'a']
			built, _ := sevensegment.Lit(dm.Digit, seg)
			dz := dm.Lift(seg, theta)
			if built {
				dz -= sevensegment.Travel
			}
			move, err := r3.Translation(r3.Vec{Z: dz})
			if err != nil {
				return nil, err
			}
			src = renderkit.TransformedMesh{TriangleSource: p.mesh, Transform: move}
		}
		m := renderkit.PartModel(src, p.color)
		if !top {
			m.Clip = sevenCut(sevenAxisCut, p.color)
		}
		out = append(out, m)
	}
	return out, nil
}

// sevenRestDigit names the digit the camshaft rests on at theta, or "between" while a ramp is moving a segment.
func sevenRestDigit(theta float64) string {
	for k := range 10 {
		if d := theta - 36*float64(k); d >= -6.5 && d <= 8 || d-360 >= -6.5 && d-360 <= 8 {
			return fmt.Sprint(k)
		}
	}
	return "between"
}

// renderSevenSegmentMotion writes the animation to output.
func renderSevenSegmentMotion(ctx context.Context, output string, width, height, frames int) error {
	s, err := buildSevenSegmentSet(ctx)
	if err != nil {
		return err
	}
	dm, err := sevensegment.DefaultParams().Derive()
	if err != nil {
		return err
	}
	half := width / 2
	var cameras [2]solidlens.Camera
	for i, name := range []string{"oblique45", "cutaway"} {
		models, err := sevenMotionModels(s, dm, dm.Theta, i == 0)
		if err != nil {
			return err
		}
		cameras[i] = renderkit.FitCamera(sevenCameras[name], models, float64(half)/float64(height), .06)
	}
	background := solidlens.RGB(.94, .96, .98).NRGBA()
	var rendered []image.Image
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		theta := 360 * float64(frame) / float64(frames)
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
		for i := range cameras {
			models, err := sevenMotionModels(s, dm, theta, i == 0)
			if err != nil {
				return err
			}
			img, err := solidlens.Render(ctx, renderkit.Scene(cameras[i], models), solidlens.Settings{Width: half, Height: height})
			if err != nil {
				return fmt.Errorf("render frame %d: %w", frame, err)
			}
			draw.Draw(canvas, image.Rect(i*half, 0, (i+1)*half, height), img, image.Point{}, draw.Over)
		}
		renderkit.Caption(canvas, basicfont.Face7x13, fmt.Sprintf("Camshaft %5.1f deg    Digit %s", theta, sevenRestDigit(theta)), 8,
			height-8)
		rendered = append(rendered, canvas)
	}
	return renderkit.WriteGIF(output, renderkit.EncodeMotionGIF(rendered, width, height, sevenMotionDelay))
}
