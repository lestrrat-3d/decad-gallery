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

// Part names one separately removable assembly body and its renderer group.
type Part struct {
	Name, Group string
	Body        *decad.Body
}

// Assembly returns placed parts, every owned or imported bearing sketch, and the motion pose.
type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
}

// Build constructs the assembly. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil {
		return nil, fmt.Errorf("pedalspindle: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("pedalspindle: nil document")
	}
	state, err := Motion(p)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("pedalspindle: context: %w", err)
	}
	a := &Assembly{State: state}
	b := &construction{ctx: ctx, doc: doc, assembly: a, world: sketch.NewWorld()}
	if err := b.frameAndGuard(p.Scale); err != nil {
		return nil, err
	}
	if err := b.driveAndLinkage(p); err != nil {
		return nil, err
	}
	if err := b.bearings(p.Scale); err != nil {
		return nil, err
	}
	if err := b.gears(p); err != nil {
		return nil, err
	}
	if len(a.Parts) != 48 || len(a.Sketches) != 44 {
		return nil, fmt.Errorf("pedalspindle: assembly counts: got %d parts and %d sketches, want 48 and 44", len(a.Parts), len(a.Sketches))
	}
	for _, part := range a.Parts {
		if !part.Body.IsSolid() {
			return nil, fmt.Errorf("%s: final body is not solid", part.Name)
		}
	}
	return a, nil
}

func (b *construction) frameAndGuard(s float64) error {
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
	for _, part := range boxes {
		if err := b.box(part.name, part.group, part.x1*s, part.x2*s, part.y1*s, part.y2*s, part.z1*s, part.z2*s); err != nil {
			return err
		}
	}
	plates := []struct {
		name, group            string
		x1, x2, z1, z2, y1, y2 float64
		holes                  []hole
	}{
		{"bearing-board", "frame", -185, 110, 290, 525, 0, 18,
			[]hole{{95 * s, 90 * s, 22}, {197.3 * s, 90 * s, 22}, {197.3 * s, 192.3 * s, 14}}},
		{"guard-front", "guard", -175, 100, 295, 520, -94, -90,
			[]hole{{85 * s, 85 * s, 8.8}, {187.3 * s, 85 * s, 8.8}, {187.3 * s, 187.3 * s, 5.8}}},
		{"heel-bracket-front", "frame", -250, -210, 18, 84, -222, -214, []hole{{20 * s, 42 * s, 8.2}}},
		{"heel-bracket-rear", "frame", -250, -210, 18, 84, -136, -128, []hole{{20 * s, 42 * s, 8.2}}},
	}
	for _, part := range plates {
		body, err := b.rectangle(part.name, 0, 0, (part.x2-part.x1)*s, (part.z2-part.z1)*s, (part.y2-part.y1)*s, part.holes)
		if err != nil {
			return err
		}
		if err := b.addXZ(part.name, part.group, body, r3.NewVec(part.x1*s, part.y2*s, part.z1*s), 0); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) driveAndLinkage(p Params) error {
	s := p.Scale
	state := b.assembly.State
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	rounds := []struct {
		name, group          string
		center               r3.Vec
		radius, bore, y1, y2 float64
	}{
		{"input-shaft", "steel", state.Input, 4, 0, -116 * s, 65 * s},
		{"intermediate-shaft", "steel", state.Intermediate, 4, 0, -83 * s, 26 * s},
		{"flywheel-rim", "drive", state.Input, 155 * s, 270 * s, 35 * s, 53 * s},
		{"flywheel-hub", "drive", state.Input, 17 * s, 8.2, 35 * s, 53 * s},
		{"input-hub", "steel", state.Input, 12 * s, 8.2, -52 * s, -44 * s},
		{"intermediate-pinion-hub", "steel", state.Intermediate, 12 * s, 8.2, -52 * s, -44 * s},
		{"intermediate-gear-hub", "steel", state.Intermediate, 12 * s, 8.2, -82 * s, -74 * s},
		{"output-hub", "steel", state.Output, 12 * s, 5.2, -82 * s, -74 * s},
		{"crank-pin", "steel", state.CrankPin, 2.5, 0, -132 * s, -105 * s},
		{"heel-pin", "steel", state.Heel, 4, 0, -230 * s, -120 * s},
		{"toe-pin", "steel", state.Toe, 2.5, 0, -215 * s, -118 * s},
		{"heel-collar-front", "steel", state.Heel, 8 * s, 8.2, -228 * s, -222 * s},
		{"heel-collar-rear", "steel", state.Heel, 8 * s, 8.2, -128 * s, -122 * s},
		{"heel-spacer-front", "steel", state.Heel, 6, 8.2, -214 * s, -210.1 * s},
		{"heel-spacer-rear", "steel", state.Heel, 6, 8.2, -139.9 * s, -136 * s},
	}
	for _, part := range rounds {
		body, err := b.round(part.name, part.radius, part.bore, part.y2-part.y1)
		if err != nil {
			return err
		}
		if err := b.addXZ(part.name, part.group, body, r3.NewVec(part.center.X, part.y2, part.center.Z), 0); err != nil {
			return err
		}
	}
	spindle, err := b.spindle(s)
	if err != nil {
		return err
	}
	if err := b.addXZ("spindle", "steel", spindle, r3.NewVec(state.Output.X, 30*s, state.Output.Z), 0); err != nil {
		return err
	}
	for k := 0; k < 4; k++ {
		name := fmt.Sprintf("flywheel-spoke-%d", k)
		body, err := b.rectangle(name, 15*s, -6*s, 137*s, 6*s, 10*s, nil)
		if err != nil {
			return err
		}
		if err := b.addXZ(name, "drive", body, r3.NewVec(state.Input.X, 49*s, state.Input.Z), a+float64(k)*math.Pi/2); err != nil {
			return err
		}
	}
	crank, err := b.rectangle("crank", -9*s, -9*s, 34*s, 9*s, 8*s, []hole{{diameter: 8.2}, {x: 25 * s, diameter: 5.2}})
	if err != nil {
		return err
	}
	if err := b.addXZ("crank", "drive", crank, r3.NewVec(state.Input.X, -105*s, state.Input.Z), a); err != nil {
		return err
	}
	treadle, err := b.treadle(s)
	if err != nil {
		return err
	}
	if err := b.addXZ("treadle", "pedal", treadle, r3.NewVec(state.Heel.X, -140*s, state.Heel.Z), state.PedalAngle); err != nil {
		return err
	}
	footman, err := b.rectangle("footman", -12*s, -12*s, 332*s, 12*s, 8*s, []hole{{diameter: 14}, {x: 320 * s, diameter: 14}})
	if err != nil {
		return err
	}
	angle := math.Atan2(state.CrankPin.Z-state.Toe.Z, state.CrankPin.X-state.Toe.X)
	return b.addXZ("footman", "pedal", footman, r3.NewVec(state.Toe.X, -121*s, state.Toe.Z), angle)
}

func (b *construction) bearings(s float64) error {
	state := b.assembly.State
	parts := []struct {
		name   string
		center r3.Vec
		small  bool
		back   float64
	}{
		{"input-bearing-front", state.Input, false, 7},
		{"input-bearing-rear", state.Input, false, 18 * s},
		{"intermediate-bearing-front", state.Intermediate, false, 7},
		{"intermediate-bearing-rear", state.Intermediate, false, 18 * s},
		{"output-bearing-front", state.Output, true, 5},
		{"output-bearing-rear", state.Output, true, 18 * s},
		{"heel-bearing-front", state.Heel, false, -210*s + 7},
		{"heel-bearing-rear", state.Heel, false, -140 * s},
		{"footman-bearing-crank", state.CrankPin, true, -125*s + 2.5},
		{"footman-bearing-toe", state.Toe, true, -125*s + 2.5},
	}
	for _, part := range parts {
		params := ballbearing.DefaultParams()
		if part.small {
			params = ballbearing.Params605ZZ()
		}
		frame, err := xzFrame(r3.NewVec(part.center.X, part.back, part.center.Z), 0)
		if err != nil {
			return fmt.Errorf("%s: bearing frame: %w", part.name, err)
		}
		params.Placement = &frame
		model, err := ballbearing.Build(b.ctx, b.doc, params)
		if err != nil {
			return fmt.Errorf("%s: imported bearing: %w", part.name, err)
		}
		b.assembly.Sketches = append(b.assembly.Sketches, model.Sketch)
		if err := b.add(part.name, "bearing", model.Body); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) gears(p Params) error {
	params := spurgear.DefaultParams()
	params.Module = 3 * p.Scale
	params.Thickness = 12 * p.Scale
	params.Samples = 15
	params.PressureAngle = 20 * math.Pi / 180
	params.CapChamfer = 0
	params.BoreDiameter = 8.2
	params.Teeth = 51
	large, err := spurgear.Build(b.ctx, b.doc, params)
	if err != nil {
		return fmt.Errorf("51-tooth gear prototype: imported gear: %w", err)
	}
	input, err := decad.Copy(b.ctx, large)
	if err != nil {
		return fmt.Errorf("input-gear: copy 51-tooth prototype: %w", err)
	}
	state := b.assembly.State
	if err := b.addXZ("input-gear", "gear", input, r3.NewVec(state.Input.X, -30*p.Scale, state.Input.Z), math.Mod(p.CrankAngle, 2*math.Pi)); err != nil {
		return err
	}
	if err := b.addXZ("intermediate-gear", "gear", large, r3.NewVec(state.Intermediate.X, -60*p.Scale, state.Intermediate.Z), state.IntermediateAngle); err != nil {
		return err
	}
	params.Teeth = 17
	for _, part := range []struct {
		name              string
		center            r3.Vec
		back, angle, bore float64
	}{
		{"intermediate-pinion", state.Intermediate, -30 * p.Scale, state.IntermediateAngle, 8.2},
		{"output-pinion", state.Output, -60 * p.Scale, state.SpindleAngle, 5.2},
	} {
		params.BoreDiameter = part.bore
		body, err := spurgear.Build(b.ctx, b.doc, params)
		if err != nil {
			return fmt.Errorf("%s: imported gear: %w", part.name, err)
		}
		if err := b.addXZ(part.name, "gear", body, r3.NewVec(part.center.X, part.back, part.center.Z), part.angle); err != nil {
			return err
		}
	}
	return nil
}
