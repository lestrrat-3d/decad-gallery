package main

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/lestrrat-3d/units"
	"golang.org/x/image/font/basicfont"
)

// Swiss lever animation: one balance period in 36 frames at 960 × 540 pixels.
const (
	swissMotionFrames = 36
	swissMotionWidth  = 960
	swissMotionHeight = 540
	// swissMotionDelay is the time of one frame in hundredths of a second.
	swissMotionDelay = 4
)

// swissMover turns one body about a vertical axis by the math angle (counter-clockwise from +Z) that angle gives it.
type swissMover struct {
	name  string
	about r3.Vec
	angle func(now swisslever.State) float64
}

// swissMovers lists the moving bodies. The built bodies stand at the build pose of the movement's own State, which
// swisslever.Build backs off the last contact by swisslever.BuildBackoff; each mover turns a body from that pose to
// the state of the frame. The wheel and the lever turn clockwise by φ and ψ, the balance counter-clockwise by θ. A
// clockwise turn by a is a rotation by -a. The hairspring stays relaxed.
func (b *swissBuild) swissMovers() ([]swissMover, error) {
	g := b.geometry
	built := b.movement.State
	sign := 1.0
	if built.Phase > 0.5 {
		sign = -1
	}
	psi := built.LeverAngle
	switch built.LeverStage {
	case swisslever.LeverBanking:
		if psi > 0 {
			psi -= swisslever.BuildBackoff
		} else {
			psi += swisslever.BuildBackoff
		}
	case swisslever.LeverUnlock:
		psi += swisslever.BuildBackoff * sign
	case swisslever.LeverImpulse:
		psi -= swisslever.BuildBackoff * sign
	default:
		return nil, fmt.Errorf("unknown lever stage %q", built.LeverStage)
	}
	phi, _, err := g.WheelAt(psi, built.Phase <= 0.5)
	if err != nil {
		return nil, fmt.Errorf("wheel angle at the build pose: %w", err)
	}
	wheelBuilt := phi - swisslever.BuildBackoff
	wheel := func(now swisslever.State) float64 { return -(now.WheelAngle - wheelBuilt) }
	lever := func(now swisslever.State) float64 { return -(now.LeverAngle - psi) }
	balance := func(now swisslever.State) float64 { return now.BalanceAngle - built.BalanceAngle }
	o, p, s := r3.NewVec(g.O.X, g.O.Y, 0), r3.NewVec(g.P.X, g.P.Y, 0), r3.NewVec(g.B.X, g.B.Y, 0)
	return []swissMover{
		{swisslever.EscapeWheelName, o, wheel}, {swisslever.EscapeArborName, o, wheel},
		{swisslever.WindingKnobName, o, wheel}, {swisslever.KnobCollarName, o, wheel},
		{swisslever.LeverName, p, lever}, {swisslever.PalletArborName, p, lever},
		{swisslever.EntryStoneName, p, lever}, {swisslever.ExitStoneName, p, lever},
		{swisslever.BalanceStaffName, s, balance}, {swisslever.SafetyRollerName, s, balance},
		{swisslever.RollerTableName, s, balance}, {swisslever.BalanceWheelName, s, balance},
		{swisslever.HairspringRetainerName, s, balance},
	}, nil
}

// motionModels returns the models of the bodies f keeps, moved to state now.
func (b *swissBuild) motionModels(f bodyFilter, movers []swissMover, now swisslever.State) ([]solidlens.Model, error) {
	turn := map[string]swissMover{}
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
			t, err := r3.RotationAround(m.about, r3.NewVec(0, 0, 1), units.Radians(m.angle(now)))
			if err != nil {
				return nil, fmt.Errorf("rotate %s: %w", nb.Name, err)
			}
			move = t
		}
		out = append(out, b.swissModel(nb.Name, move))
	}
	return out, nil
}

// renderSwissMotion writes the animation of one balance period: Phase k/36 for each frame, seen from the front with
// the front plate, the front bearings, the bolts, the knob and its collar left out so the mechanism shows.
func renderSwissMotion(ctx context.Context, output string, width, height, frames int) error {
	b, err := buildSwiss(ctx)
	if err != nil {
		return err
	}
	hidden := append(append([]string{}, swissFrontParts...), swisslever.WindingKnobName, swisslever.KnobCollarName)
	keep := without(hidden...)
	movers, err := b.swissMovers()
	if err != nil {
		return err
	}
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
	camera := swissCamera(framing, swissFrontDir, float64(width)/float64(height))
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
		img, err := solidlens.Render(ctx, renderkit.Scene(camera, models), solidlens.Settings{Width: width, Height: height})
		if err != nil {
			return fmt.Errorf("render frame %d: %w", frame, err)
		}
		canvas := image.NewRGBA(img.Bounds())
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
		draw.Draw(canvas, canvas.Bounds(), img, image.Point{}, draw.Over)
		const toDegrees = 180 / math.Pi
		caption(canvas, basicfont.Face7x13, fmt.Sprintf("Balance %6.1f deg    Lever %5.1f deg    Wheel %6.2f deg",
			state.BalanceAngle*toDegrees, state.LeverAngle*toDegrees, state.WheelAngle*toDegrees), 8, height-8)
		rendered = append(rendered, canvas)
	}
	return writeWheelGIF(output, encodeMotionGIF(rendered, width, height, swissMotionDelay))
}
