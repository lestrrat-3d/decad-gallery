package render

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font/basicfont"
)

// clickerState is one pose of the click indexer: the shaft angle, the rotor's drop, the button's drop and the digit shown.
type clickerState struct {
	theta, rotorDrop, buttonDrop float64
	digit                        int
}

// motionStates lists the frames: ten presses, digit 0 to 9 and back to 0.
func motionStates() []clickerState {
	var out []clickerState
	for k := range 10 {
		add := func(s float64, down bool) {
			var u, turn float64
			if down {
				u, turn = sevensegment.PressPose(s)
			} else {
				u, turn = sevensegment.ReleasePose(s)
			}
			shown := k
			if turn >= sevensegment.SegmentsDoneBy {
				shown = (k + 1) % 10
			}
			out = append(out, clickerState{theta: 36*float64(k) + turn, rotorDrop: u, buttonDrop: s, digit: shown})
		}
		for _, s := range []float64{0, 1.5, 3, 4.5, 6} {
			add(s, true)
		}
		for _, s := range []float64{4.5, 3, 1.5, 0} {
			add(s, false)
		}
	}
	return out
}

func nest(src solidlens.TriangleSource, t r3.Transform) solidlens.TriangleSource {
	return renderkit.TransformedMesh{TriangleSource: src, Transform: t}
}

// motionModels returns the models of one state. face selects the face view (left panel), otherwise the cut through the axis.
func motionModels(s partSet, springs map[float64]*decad.Mesh, dm sevensegment.Dimensions, st clickerState,
	face bool) ([]solidlens.Model, error) {
	turn, err := r3.RotationAround(r3.Vec{Z: sevensegment.ShaftZ}, r3.Vec{Y: 1}, units.Degrees(st.theta-dm.Theta))
	if err != nil {
		return nil, err
	}
	rotorDown, err := r3.Translation(r3.Vec{Y: st.rotorDrop})
	if err != nil {
		return nil, err
	}
	buttonDown, err := r3.Translation(r3.Vec{Y: st.buttonDrop})
	if err != nil {
		return nil, err
	}
	var out []solidlens.Model
	add := func(src solidlens.TriangleSource, c solidlens.Color) {
		m := renderkit.PartModel(src, c)
		if !face {
			m.Clip = sevenCut(sevenAxisCut, c)
		}
		out = append(out, m)
	}
	keep := func(name string) bool {
		switch {
		case face:
			return true
		case strings.HasPrefix(name, "Slider ") || strings.HasPrefix(name, "Flag ") || strings.HasPrefix(name, "Cam ") ||
			strings.HasPrefix(name, "Bolt") || strings.HasPrefix(name, "Nut") || name == "Guide Plate" ||
			name == "Top Plate" || name == "Cradle Plate" || name == "Rear Wall" || name == "Rear Bearing":
			return false
		}
		return true
	}
	if face {
		keep = func(name string) bool {
			if name == "Spring" || name == "Hub" || name == "Rotor" {
				return false // hidden inside the housing from the front
			}
			return true
		}
	}
	for _, p := range s {
		if !keep(p.name) {
			continue
		}
		if strings.HasPrefix(p.name, "Spring") {
			continue
		}
		var src solidlens.TriangleSource = p.mesh
		switch {
		case p.name == "Hub" || p.name == "Shaft" || strings.HasPrefix(p.name, "Cam "):
			src = nest(src, turn)
		case p.name == "Rotor":
			src = nest(nest(src, turn), rotorDown)
		case p.name == "Button":
			src = nest(src, buttonDown)
		case strings.HasPrefix(p.name, "Flag "):
			seg := sevenFlagSegment(p.name)
			t, err := sevenFlagMove(seg, dm.FlagAngle(seg, st.theta)-dm.FlagAngle(seg, dm.Theta))
			if err != nil {
				return nil, err
			}
			src = nest(src, t)
		case strings.HasPrefix(p.name, "Slider "):
			seg := sevensegment.Segments()[p.name[len(p.name)-1]-'a']
			built, _ := sevensegment.Lit(dm.Digit, seg)
			dz := dm.Lift(seg, st.theta)
			if built {
				dz -= sevensegment.Travel
			}
			t, err := r3.Translation(r3.Vec{Z: dz})
			if err != nil {
				return nil, err
			}
			src = nest(src, t)
		}
		add(src, p.color)
	}
	if !face {
		// The spring, at the length between the hub's top face and the rotor's plate, nearest a built one.
		length := sevensegment.SpringRestLength - st.rotorDrop
		best, bestD := 0.0, math.Inf(1)
		for l := range springs {
			if d := math.Abs(l - length); d < bestD {
				best, bestD = l, d
			}
		}
		add(nest(springs[best], turn), sevenHardware)
	}
	return out, nil
}

// renderSevenSegmentMotion writes the labelled preview animation to output. It is not the final model.
func renderSevenSegmentMotion(ctx context.Context, output string, width, height int) error {
	s, err := buildSevenSegmentSet(ctx)
	if err != nil {
		return err
	}
	dm, err := sevensegment.DefaultParams().Derive()
	if err != nil {
		return err
	}
	springs := map[float64]*decad.Mesh{}
	for _, l := range []float64{6, 6.5, 7, 7.5, 8} {
		doc := decad.New()
		body, err := sevensegment.BuildSpring(ctx, doc, l)
		if err != nil {
			return fmt.Errorf("spring %g: %w", l, err)
		}
		if springs[l], err = body.Tessellate(ctx, units.Millimeters(.05)); err != nil {
			return fmt.Errorf("mesh spring %g: %w", l, err)
		}
	}
	states := motionStates()
	half := width / 2
	// Knob end up: the camera's up is -Y. Left: the face from +Z, turned a little toward +X. Right: the cut through the axis
	// from +X, close on the indexer.
	faceCam := solidlens.Camera{Position: solidlens.Vec{X: 70, Y: -40, Z: 330}, Target: solidlens.Vec{Y: -30, Z: 40},
		Up: solidlens.Vec{Y: -1}, FOV: 30}
	cutCam := solidlens.Camera{Position: solidlens.Vec{X: 300, Y: -66, Z: sevensegment.ShaftZ},
		Target: solidlens.Vec{Y: -66, Z: sevensegment.ShaftZ},
		Up:     solidlens.Vec{Y: -1}, FOV: 30}
	rest := states[0]
	rest.buttonDrop = 0
	var cams [2]solidlens.Camera
	for i, base := range []solidlens.Camera{faceCam, cutCam} {
		models, err := motionModels(s, springs, dm, rest, i == 0)
		if err != nil {
			return err
		}
		if i == 1 { // fit the cut to the indexer only
			var near []solidlens.Model
			for _, m := range models {
				near = append(near, m)
			}
			models = near
		}
		cams[i] = renderkit.FitCamera(base, models, float64(half)/float64(height), .06)
	}
	background := solidlens.RGB(.94, .96, .98).NRGBA()
	var rendered []image.Image
	for n, st := range states {
		if err := ctx.Err(); err != nil {
			return err
		}
		canvas := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
		for i := range cams {
			models, err := motionModels(s, springs, dm, st, i == 0)
			if err != nil {
				return err
			}
			img, err := solidlens.Render(ctx, renderkit.Scene(cams[i], models), solidlens.Settings{Width: half, Height: height})
			if err != nil {
				return fmt.Errorf("render frame %d: %w", n, err)
			}
			draw.Draw(canvas, image.Rect(i*half, 0, (i+1)*half, height), img, image.Point{}, draw.Over)
		}
		renderkit.Caption(canvas, basicfont.Face7x13, "Seven-segment display: push-button click indexer", 8, 16)
		renderkit.Caption(canvas, basicfont.Face7x13, fmt.Sprintf("Digit %d   button down %.1f mm   shaft %.1f deg", st.digit,
			st.buttonDrop, st.theta), 8, height-8)
		rendered = append(rendered, canvas)
	}
	return renderkit.WriteGIF(output, renderkit.EncodeMotionGIF(rendered, width, height, 12))
}
