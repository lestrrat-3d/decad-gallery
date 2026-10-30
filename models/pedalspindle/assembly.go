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

// Part identifies one separate solid in the spindle assembly.
type Part struct {
	Name, Group string
	Body        *decad.Body
}

// Assembly contains the placed parts, recorded sketches, and solved pose.
type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
}

// Build constructs the complete spindle assembly. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, params Params) (*Assembly, error) {
	if ctx == nil {
		return nil, fmt.Errorf("pedalspindle: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("pedalspindle: document must not be nil")
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("pedalspindle: build context: %w", err)
	}
	state, err := Motion(params)
	if err != nil {
		return nil, err
	}
	assembly := &Assembly{State: state}
	b := construction{ctx: ctx, doc: doc, assembly: assembly}
	s := params.Scale
	a := math.Mod(params.CrankAngle, 2*math.Pi)
	if err := b.frame(s); err != nil {
		return nil, err
	}
	if err := b.shafts(s); err != nil {
		return nil, err
	}
	if err := b.gears(s, a); err != nil {
		return nil, err
	}
	if err := b.flywheel(s, a); err != nil {
		return nil, err
	}
	if err := b.hubs(s); err != nil {
		return nil, err
	}
	if err := b.linkage(s, a); err != nil {
		return nil, err
	}
	if err := b.bearings(s); err != nil {
		return nil, err
	}
	if len(assembly.Parts) != 43 || len(assembly.Sketches) != 39 {
		return nil, fmt.Errorf("pedalspindle: assembly has %d parts and %d sketches; expected 43 and 39",
			len(assembly.Parts), len(assembly.Sketches))
	}
	for _, part := range assembly.Parts {
		if !part.Body.IsSolid() {
			return nil, fmt.Errorf("pedalspindle: %s: completed body is not solid", part.Name)
		}
	}
	return assembly, nil
}

func (b *construction) frame(s float64) error {
	boxes := []struct {
		name, group            string
		x1, x2, y1, y2, z1, z2 float64
	}{
		{"base", "frame", -270, 150, -240, 110, 0, 18},
		{"left-leg", "frame", -180, -156, 0, 18, 18, 290},
		{"right-leg", "frame", 80, 104, 0, 18, 18, 290},
		{"guard-left", "guard", -179, -175, -94, 0, 291, 524},
		{"guard-right", "guard", 100, 104, -94, 0, 291, 524},
		{"guard-bottom", "guard", -175, 100, -94, 0, 291, 295},
		{"guard-top", "guard", -175, 100, -94, 0, 520, 524},
	}
	for _, box := range boxes {
		if err := b.box(box.name, box.group, box.x1*s, box.x2*s,
			box.y1*s, box.y2*s, box.z1*s, box.z2*s); err != nil {
			return err
		}
	}
	state := b.assembly.State
	boardHoles := []circularHole{
		{state.Input.X + 185*s, state.Input.Z - 290*s, 22},
		{state.Intermediate.X + 185*s, state.Intermediate.Z - 290*s, 22},
		{state.Output.X + 185*s, state.Output.Z - 290*s, 14},
	}
	if err := b.xzRectangle("bearing-board", "frame", r3.NewVec(-185*s, 18*s, 290*s),
		0, 0, 0, 295*s, 235*s, 18*s, boardHoles); err != nil {
		return err
	}
	guardHoles := []circularHole{
		{state.Input.X + 175*s, state.Input.Z - 295*s, 8.8},
		{state.Intermediate.X + 175*s, state.Intermediate.Z - 295*s, 8.8},
		{state.Output.X + 175*s, state.Output.Z - 295*s, 5.8},
	}
	if err := b.xzRectangle("guard-front", "guard", r3.NewVec(-175*s, -90*s, 295*s),
		0, 0, 0, 275*s, 225*s, 4*s, guardHoles); err != nil {
		return err
	}
	return b.xzRectangle("heel-bracket", "frame", r3.NewVec(-244*s, -112*s, 18*s),
		0, 0, 0, 28*s, 58*s, 8*s, []circularHole{{14 * s, 42 * s, 5.2}})
}

func (b *construction) shafts(s float64) error {
	state := b.assembly.State
	if err := b.xzRing("input-shaft", "steel", state.Input, 65*s, 4, 0, 181*s); err != nil {
		return err
	}
	if err := b.xzRing("intermediate-shaft", "steel", state.Intermediate, 26*s, 4, 0, 109*s); err != nil {
		return err
	}
	return b.spindle(s, state.Output)
}

func (b *construction) gears(s, a float64) error {
	params := spurgear.Params{
		Module: 3 * s, Teeth: 51, PressureAngle: 20 * math.Pi / 180,
		Thickness: 12 * s, Samples: 15, BoreDiameter: 8.2, CapChamfer: 0,
	}
	large, err := spurgear.Build(b.ctx, b.doc, params)
	if err != nil {
		return fmt.Errorf("pedalspindle: input/intermediate gear prototype: %w", err)
	}
	params.Teeth = 17
	intermediatePinion, err := spurgear.Build(b.ctx, b.doc, params)
	if err != nil {
		return fmt.Errorf("pedalspindle: intermediate pinion prototype: %w", err)
	}
	params.BoreDiameter = 5.2
	outputPinion, err := spurgear.Build(b.ctx, b.doc, params)
	if err != nil {
		return fmt.Errorf("pedalspindle: output pinion prototype: %w", err)
	}
	inputGear, err := decad.Copy(b.ctx, large)
	if err != nil {
		return fmt.Errorf("pedalspindle: input-gear: copy large prototype: %w", err)
	}
	state := b.assembly.State
	placements := []struct {
		name     string
		body     *decad.Body
		center   r3.Vec
		y, angle float64
	}{
		{"input-gear", inputGear, state.Input, -30 * s, a},
		{"intermediate-pinion", intermediatePinion, state.Intermediate, -30 * s, state.IntermediateAngle},
		{"intermediate-gear", large, state.Intermediate, -60 * s, state.IntermediateAngle},
		{"output-pinion", outputPinion, state.Output, -60 * s, state.SpindleAngle},
	}
	for _, gear := range placements {
		frame, err := xzFrame(r3.NewVec(gear.center.X, gear.y, gear.center.Z), gear.angle)
		if err != nil {
			return fmt.Errorf("pedalspindle: %s: create gear frame: %w", gear.name, err)
		}
		if err := b.appendPart(gear.name, "gear", gear.body, nil, frame); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) flywheel(s, angle float64) error {
	center := b.assembly.State.Input
	if err := b.xzRing("flywheel-rim", "drive", center, 53*s, 155*s, 270*s, 18*s); err != nil {
		return err
	}
	if err := b.xzRing("flywheel-hub", "drive", center, 53*s, 17*s, 8.2, 18*s); err != nil {
		return err
	}
	for i := range 4 {
		name := fmt.Sprintf("flywheel-spoke-%d", i)
		if err := b.xzRectangle(name, "drive", r3.NewVec(center.X, 49*s, center.Z),
			angle+float64(i)*math.Pi/2, 15*s, -6*s, 137*s, 6*s, 10*s, nil); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) hubs(s float64) error {
	state := b.assembly.State
	parts := []struct {
		name    string
		center  r3.Vec
		y, bore float64
	}{
		{"input-hub", state.Input, -44 * s, 8.2},
		{"intermediate-pinion-hub", state.Intermediate, -44 * s, 8.2},
		{"intermediate-gear-hub", state.Intermediate, -74 * s, 8.2},
		{"output-hub", state.Output, -74 * s, 5.2},
	}
	for _, part := range parts {
		if err := b.xzRing(part.name, "steel", part.center, part.y, 12*s, part.bore, 8*s); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) linkage(s, angle float64) error {
	state := b.assembly.State
	if err := b.xzRectangle("crank", "drive", r3.NewVec(state.Input.X, -105*s, state.Input.Z),
		angle, -9*s, -9*s, 34*s, 9*s, 8*s,
		[]circularHole{{0, 0, 8.2}, {25 * s, 0, 5.2}}); err != nil {
		return err
	}
	if err := b.xzRing("crank-pin", "steel", state.CrankPin, -94*s, 2.5, 0, 22*s); err != nil {
		return err
	}
	if err := b.xzRectangle("treadle", "pedal", r3.NewVec(state.Heel.X, -120*s, state.Heel.Z),
		state.PedalAngle, -12*s, -12*s, 212*s, 12*s, 90*s,
		[]circularHole{{0, 0, 14}, {200 * s, 0, 5.2}}); err != nil {
		return err
	}
	rodAngle := math.Atan2(state.CrankPin.Z-state.Toe.Z, state.CrankPin.X-state.Toe.X)
	if err := b.xzRectangle("footman", "pedal", r3.NewVec(state.Toe.X, -97*s, state.Toe.Z),
		rodAngle, -12*s, -12*s, 332*s, 12*s, 8*s,
		[]circularHole{{0, 0, 14}, {320 * s, 0, 14}}); err != nil {
		return err
	}
	if err := b.xzRing("heel-pin", "steel", state.Heel, -115*s, 2.5, 0, 100*s); err != nil {
		return err
	}
	return b.xzRing("toe-pin", "steel", state.Toe, -94*s, 2.5, 0, 121*s)
}

func (b *construction) bearings(s float64) error {
	state := b.assembly.State
	parts := []struct {
		name   string
		center r3.Vec
		backY  float64
		is608  bool
	}{
		{"input-bearing-front", state.Input, 7, true},
		{"input-bearing-rear", state.Input, 18 * s, true},
		{"intermediate-bearing-front", state.Intermediate, 7, true},
		{"intermediate-bearing-rear", state.Intermediate, 18 * s, true},
		{"output-bearing-front", state.Output, 5, false},
		{"output-bearing-rear", state.Output, 18 * s, false},
		{"heel-bearing-front", state.Heel, -210*s + 5, false},
		{"heel-bearing-rear", state.Heel, -120 * s, false},
		{"footman-bearing-crank", state.CrankPin, -101*s + 2.5, false},
		{"footman-bearing-toe", state.Toe, -101*s + 2.5, false},
	}
	for _, part := range parts {
		frame, err := xzFrame(r3.NewVec(part.center.X, part.backY, part.center.Z), 0)
		if err != nil {
			return fmt.Errorf("pedalspindle: %s: create bearing frame: %w", part.name, err)
		}
		params := ballbearing.Params605ZZ()
		if part.is608 {
			params = ballbearing.DefaultParams()
		}
		params.Placement = &frame
		bearing, err := ballbearing.Build(b.ctx, b.doc, params)
		if err != nil {
			return fmt.Errorf("pedalspindle: %s: build purchased bearing: %w", part.name, err)
		}
		if !bearing.Body.IsSolid() {
			return fmt.Errorf("pedalspindle: %s: imported bearing body is not solid", part.name)
		}
		b.assembly.Parts = append(b.assembly.Parts,
			Part{Name: part.name, Group: "bearing", Body: bearing.Body})
		b.assembly.Sketches = append(b.assembly.Sketches, bearing.Sketch)
	}
	return nil
}
