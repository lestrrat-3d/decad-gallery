package pedalspindle

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

type Part struct {
	Name, Group string
	Body        *decad.Body
}

type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
}

// Build requires a nonnil context and document.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil {
		return nil, fmt.Errorf("pedalspindle: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("pedalspindle: document must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("pedalspindle: context: %w", err)
	}
	state, err := Motion(p)
	if err != nil {
		return nil, fmt.Errorf("pedalspindle: parameters: %w", err)
	}
	assembly := &Assembly{State: state}
	b := &builder{ctx: ctx, doc: doc, assembly: assembly}
	if err := b.buildParts(p); err != nil {
		return nil, err
	}
	if len(assembly.Parts) != 48 || len(assembly.Sketches) != 58 {
		return nil, fmt.Errorf("pedalspindle: expected 48 parts and 58 sketches, got %d and %d", len(assembly.Parts), len(assembly.Sketches))
	}
	return assembly, nil
}

func (b *builder) placedPart(name, group string, body *decad.Body, center r3.Vec, y, angle float64) error {
	body, err := b.place(name, body, center, y, angle)
	if err != nil {
		return err
	}
	return b.part(name, group, body)
}

func (b *builder) buildParts(p Params) error {
	s := p.Scale
	st := b.assembly.State
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	boxes := []struct {
		name, group            string
		x0, x1, y0, y1, z0, z1 float64
	}{
		{"base", "frame", -270, 150, -240, 110, 0, 18},
		{"left-leg", "frame", -180, -156, 0, 18, 18, 290},
		{"right-leg", "frame", 80, 104, 0, 18, 18, 290},
		{"guard-left", "guard", -179, -175, -94, 0, 291, 524},
		{"guard-right", "guard", 100, 104, -94, 0, 291, 524},
		{"guard-bottom", "guard", -175, 100, -94, 0, 291, 295},
		{"guard-top", "guard", -175, 100, -94, 0, 520, 524},
	}
	for _, v := range boxes {
		body, err := b.rectangle(v.name, 0, 0, (v.x1-v.x0)*s, (v.y1-v.y0)*s, (v.z1-v.z0)*s, nil)
		if err != nil {
			return err
		}
		body, err = b.transform(v.name, body, r3.NewVec(v.x0*s, v.y0*s, v.z0*s), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
		if err != nil {
			return err
		}
		if err = b.part(v.name, v.group, body); err != nil {
			return err
		}
	}
	plates := []struct {
		name, group               string
		x0, x1, z0, z1, y, length float64
		holes                     []hole
	}{
		{"bearing-board", "frame", -185, 110, 290, 525, 18, 18,
			[]hole{{95 * s, 90 * s, 22}, {197.3 * s, 90 * s, 22}, {197.3 * s, 192.3 * s, 14}}},
		{"guard-front", "guard", -175, 100, 295, 520, -90, 4,
			[]hole{{85 * s, 85 * s, 8.8}, {187.3 * s, 85 * s, 8.8}, {187.3 * s, 187.3 * s, 5.8}}},
		{"heel-bracket-front", "frame", -250, -210, 18, 84, -214, 8, []hole{{20 * s, 42 * s, 8.2}}},
		{"heel-bracket-rear", "frame", -250, -210, 18, 84, -128, 8, []hole{{20 * s, 42 * s, 8.2}}},
	}
	for _, v := range plates {
		body, err := b.rectangle(v.name, 0, 0, (v.x1-v.x0)*s, (v.z1-v.z0)*s, v.length*s, v.holes)
		if err != nil {
			return err
		}
		if err = b.placedPart(v.name, v.group, body, r3.NewVec(v.x0*s, 0, v.z0*s), v.y*s, 0); err != nil {
			return err
		}
	}
	if err := b.buildBearings(s); err != nil {
		return err
	}
	input, err := b.inputShaft(s)
	if err != nil {
		return err
	}
	if err = b.placedPart("input-shaft", "steel", input, st.Input, 65*s, a); err != nil {
		return err
	}
	intermediate, err := b.intermediateShaft(s)
	if err != nil {
		return err
	}
	if err = b.placedPart("intermediate-shaft", "steel", intermediate, st.Intermediate, 26*s, st.IntermediateAngle); err != nil {
		return err
	}
	spindle, err := b.flatSpindle(s)
	if err != nil {
		return err
	}
	if err = b.placedPart("spindle", "steel", spindle, st.Output, 30*s, st.SpindleAngle); err != nil {
		return err
	}
	if err = b.buildGears(s, a); err != nil {
		return err
	}
	if err = b.buildFlywheel(s, a); err != nil {
		return err
	}
	if err = b.buildLinkage(s, a); err != nil {
		return err
	}
	return b.buildRingsAndPins(s)
}

func (b *builder) buildBearings(s float64) error {
	st := b.assembly.State
	items := []struct {
		name   string
		center r3.Vec
		y      float64
		small  bool
	}{
		{"input-bearing-front", st.Input, 7, false}, {"input-bearing-rear", st.Input, 18 * s, false},
		{"intermediate-bearing-front", st.Intermediate, 7, false}, {"intermediate-bearing-rear", st.Intermediate, 18 * s, false},
		{"output-bearing-front", st.Output, 5, true}, {"output-bearing-rear", st.Output, 18 * s, true},
		{"heel-bearing-front", st.Heel, -210*s + 7, false}, {"heel-bearing-rear", st.Heel, -140 * s, false},
		{"footman-bearing-crank", st.CrankPin, -125*s + 2.5, true},
		{"footman-bearing-toe", st.Toe, -125*s + 2.5, true},
	}
	for _, v := range items {
		p := ballbearing.DefaultParams()
		if v.small {
			p = ballbearing.Params605ZZ()
		}
		frame, err := r3.NewFrame(r3.NewVec(v.center.X, v.y, v.center.Z), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
		if err != nil {
			return fmt.Errorf("%s: bearing frame: %w", v.name, err)
		}
		p.Placement = &frame
		model, err := ballbearing.Build(b.ctx, b.doc, p)
		if err != nil {
			return fmt.Errorf("%s: bearing construction: %w", v.name, err)
		}
		b.assembly.Sketches = append(b.assembly.Sketches, model.Sketch)
		if err = b.part(v.name, "bearing", model.Body); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) buildGears(s, a float64) error {
	p := spurgear.DefaultParams()
	p.Module = 3 * s
	p.Teeth = 51
	p.PressureAngle = 20 * math.Pi / 180
	p.Thickness = 12 * s
	p.Samples = 15
	p.BoreDiameter = 0
	p.CapChamfer = 0
	large, err := spurgear.Build(b.ctx, b.doc, p)
	if err != nil {
		return fmt.Errorf("large gear prototype: build: %w", err)
	}
	large, err = b.dBore("large-gear", large, 4.1, 3.1, p.Thickness)
	if err != nil {
		return err
	}
	input, err := decad.Copy(b.ctx, large)
	if err != nil {
		return fmt.Errorf("input-gear: copy: %w", err)
	}
	p.Teeth = 17
	small, err := spurgear.Build(b.ctx, b.doc, p)
	if err != nil {
		return fmt.Errorf("small gear prototype: build: %w", err)
	}
	intermediate, err := decad.Copy(b.ctx, small)
	if err != nil {
		return fmt.Errorf("intermediate-pinion: copy: %w", err)
	}
	intermediate, err = b.dBore("intermediate-pinion", intermediate, 4.1, 3.1, p.Thickness)
	if err != nil {
		return err
	}
	output, err := b.dBore("output-pinion", small, 2.6, 1.85, p.Thickness)
	if err != nil {
		return err
	}
	st := b.assembly.State
	items := []struct {
		name     string
		body     *decad.Body
		center   r3.Vec
		y, angle float64
	}{
		{"input-gear", input, st.Input, -30 * s, a},
		{"intermediate-pinion", intermediate, st.Intermediate, -30 * s, st.IntermediateAngle},
		{"intermediate-gear", large, st.Intermediate, -60 * s, st.IntermediateAngle},
		{"output-pinion", output, st.Output, -60 * s, st.SpindleAngle},
	}
	for _, v := range items {
		if err = b.placedPart(v.name, "gear", v.body, v.center, v.y, v.angle); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) buildFlywheel(s, a float64) error {
	center := b.assembly.State.Input
	rim, err := b.disc("flywheel-rim", 155*s, 270*s, 18*s)
	if err != nil {
		return err
	}
	if err = b.placedPart("flywheel-rim", "drive", rim, center, 53*s, 0); err != nil {
		return err
	}
	hub, err := b.disc("flywheel-hub", 17*s, 0, 18*s)
	if err != nil {
		return err
	}
	hub, err = b.dBore("flywheel-hub", hub, 4.1, 3.1, 18*s)
	if err != nil {
		return err
	}
	if err = b.placedPart("flywheel-hub", "drive", hub, center, 53*s, a); err != nil {
		return err
	}
	for k := 0; k < 4; k++ {
		name := fmt.Sprintf("flywheel-spoke-%d", k)
		body, err := b.rectangle(name, 15*s, -6*s, 137*s, 6*s, 10*s, nil)
		if err != nil {
			return err
		}
		if err = b.placedPart(name, "drive", body, center, 49*s, a+float64(k)*math.Pi/2); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) buildLinkage(s, a float64) error {
	st := b.assembly.State
	crank, err := b.rectangle("crank", -9*s, -9*s, 34*s, 9*s, 8*s, []hole{{x: 25 * s, diameter: 5.2}})
	if err != nil {
		return err
	}
	crank, err = b.dBore("crank", crank, 4.1, 3.1, 8*s)
	if err != nil {
		return err
	}
	if err = b.placedPart("crank", "drive", crank, st.Input, -105*s, a); err != nil {
		return err
	}
	treadle, err := b.treadle(s)
	if err != nil {
		return err
	}
	if err = b.placedPart("treadle", "pedal", treadle, st.Heel, -140*s, st.PedalAngle); err != nil {
		return err
	}
	footman, err := b.rectangle("footman", -12*s, -12*s, 332*s, 12*s, 8*s,
		[]hole{{diameter: 14}, {x: 320 * s, diameter: 14}})
	if err != nil {
		return err
	}
	angle := math.Atan2(st.CrankPin.Z-st.Toe.Z, st.CrankPin.X-st.Toe.X)
	return b.placedPart("footman", "pedal", footman, st.Toe, -121*s, angle)
}

func (b *builder) buildRingsAndPins(s float64) error {
	st := b.assembly.State
	items := []struct {
		name                        string
		center                      r3.Vec
		radius, diameter, y, length float64
	}{
		{"input-hub", st.Input, 12 * s, 8.2, -44 * s, 8 * s},
		{"intermediate-pinion-hub", st.Intermediate, 12 * s, 8.2, -44 * s, 8 * s},
		{"intermediate-gear-hub", st.Intermediate, 12 * s, 8.2, -74 * s, 8 * s},
		{"output-hub", st.Output, 12 * s, 5.2, -74 * s, 8 * s},
		{"crank-pin", st.CrankPin, 2.5, 0, -105 * s, 27 * s},
		{"heel-pin", st.Heel, 4, 0, -120 * s, 110 * s},
		{"toe-pin", st.Toe, 2.5, 0, -118 * s, 97 * s},
		{"heel-collar-front", st.Heel, 8 * s, 8.2, -222 * s, 6 * s},
		{"heel-collar-rear", st.Heel, 8 * s, 8.2, -122 * s, 6 * s},
		{"heel-spacer-front", st.Heel, 6, 8.2, -210.1 * s, 3.9 * s},
		{"heel-spacer-rear", st.Heel, 6, 8.2, -136 * s, 3.9 * s},
	}
	for _, v := range items {
		body, err := b.disc(v.name, v.radius, v.diameter, v.length)
		if err != nil {
			return err
		}
		if err = b.placedPart(v.name, "steel", body, v.center, v.y, 0); err != nil {
			return err
		}
	}
	return nil
}
