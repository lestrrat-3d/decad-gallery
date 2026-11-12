package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font/basicfont"
)

// Escapement animation: one pendulum period in 36 frames at 960 × 540 pixels.
const (
	escapementMotionFrames = 36
	escapementMotionWidth  = 960
	escapementMotionHeight = 540
	// escapementMotionDelay is the time of one frame in hundredths of a second.
	escapementMotionDelay = 4
)

// escapementMover turns one body about a vertical axis by the angle the state gives it relative to the rest pose.
type escapementMover struct {
	name  string
	about r3.Vec
	angle func(rest, now escapement.State) float64
}

// escapementMovers lists the moving bodies of the escapement: the wheel and its arbor turn clockwise by φ, the anchor,
// its arbor and the crutch clockwise by ψ, the pendulum and its bearing clockwise by β. A clockwise turn by a is a
// rotation by -a.
func (b *escapementBuild) escapementMovers() []escapementMover {
	g := b.geometry
	o, p, s := r3.NewVec(g.O.X, g.O.Y, 0), r3.NewVec(g.P.X, g.P.Y, 0), r3.NewVec(g.S.X, g.S.Y, 0)
	wheel := func(rest, now escapement.State) float64 { return -(now.WheelAngle - rest.WheelAngle) }
	anchor := func(rest, now escapement.State) float64 { return -(now.AnchorAngle - rest.AnchorAngle) }
	pendulum := func(rest, now escapement.State) float64 { return -(now.PendulumAngle - rest.PendulumAngle) }
	return []escapementMover{
		{escapement.EscapeWheelName, o, wheel}, {escapement.EscapeArborName, o, wheel},
		{escapement.AnchorName, p, anchor}, {escapement.PalletArborName, p, anchor}, {escapement.CrutchName, p, anchor},
		{escapement.PendulumName, s, pendulum}, {escapement.PendulumBearingName, s, pendulum},
	}
}

// motionModels returns the models of the bodies of the animation at a state.
func (b *escapementBuild) motionModels(f bodyFilter, movers []escapementMover, now escapement.State) ([]solidlens.Model, error) {
	turn := map[string]escapementMover{}
	for _, m := range movers {
		turn[m.name] = m
	}
	var out []solidlens.Model
	for _, nb := range b.named {
		if !f(nb.Name) {
			continue
		}
		move := r3.Identity()
		if m, ok := turn[nb.Name]; ok {
			t, err := r3.RotationAround(m.about, r3.NewVec(0, 0, 1), units.Radians(m.angle(b.state, now)))
			if err != nil {
				return nil, fmt.Errorf("rotate %s: %w", nb.Name, err)
			}
			move = t
		}
		out = append(out, b.escapementModel(nb.Name, move))
	}
	return out, nil
}

// renderEscapementMotion writes the animation of one period: Phase k/36 for each frame, seen from the front with the
// front plate, the front bearings, the bolts and the drive left out.
func renderEscapementMotion(ctx context.Context, output string, width, height, frames int) error {
	b, err := buildEscapement(ctx)
	if err != nil {
		return err
	}
	keep := without(append(append([]string{}, frontParts...), driveParts...)...)
	movers := b.escapementMovers()
	// The camera frames the swing's two ends so nothing leaves the image.
	var framing []solidlens.Model
	for _, phase := range []float64{0, 0.5} {
		s, err := b.geometry.StateAt(phase)
		if err != nil {
			return err
		}
		models, err := b.motionModels(keep, movers, s)
		if err != nil {
			return err
		}
		framing = append(framing, models...)
	}
	camera := escapementCamera(framing, escapementFrontDir, float64(width)/float64(height))
	background := solidlens.RGB(.94, .96, .98).NRGBA()
	var rendered []image.Image
	for frame := range frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		state, err := b.geometry.StateAt(float64(frame) / float64(frames))
		if err != nil {
			return err
		}
		models, err := b.motionModels(keep, movers, state)
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		if err := solidlens.RenderPNG(ctx, &buffer, bevelScene(camera, models), solidlens.Settings{Width: width, Height: height}); err != nil {
			return fmt.Errorf("render frame %d: %w", frame, err)
		}
		img, err := png.Decode(&buffer)
		if err != nil {
			return fmt.Errorf("decode frame %d: %w", frame, err)
		}
		canvas := image.NewRGBA(img.Bounds())
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
		draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
		caption(canvas, basicfont.Face7x13, fmt.Sprintf("Pendulum %5.1f deg    Anchor %5.1f deg    Wheel %6.2f deg",
			state.PendulumAngle*180/math.Pi, state.AnchorAngle*180/math.Pi,
			state.WheelAngle*180/math.Pi), 8, height-8)
		rendered = append(rendered, canvas)
	}
	colors := adaptivePalette(rendered, 255)
	clear := uint8(len(colors))
	colors = append(colors, color.RGBA{})
	nearest := map[color.RGBA]uint8{}
	var full []*image.Paletted
	for _, img := range rendered {
		indexed := image.NewPaletted(img.Bounds(), colors)
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				index, ok := nearest[c]
				if !ok {
					index = uint8(colors.Index(c))
					nearest[c] = index
				}
				indexed.SetColorIndex(x, y, index)
			}
		}
		full = append(full, indexed)
	}
	animation := &gif.GIF{LoopCount: 0}
	for i, indexed := range full {
		frame := indexed
		if i > 0 {
			frame = changedBox(full[i-1], indexed, clear)
		}
		animation.Image = append(animation.Image, frame)
		animation.Delay = append(animation.Delay, escapementMotionDelay)
		animation.Disposal = append(animation.Disposal, gif.DisposalNone)
	}
	animation.Config = image.Config{ColorModel: colors, Width: width, Height: height}
	return writeWheelGIF(output, animation)
}
