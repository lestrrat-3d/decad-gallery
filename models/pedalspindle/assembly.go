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

// Part names an independently removable solid and its rendering group.
type Part struct {
	Name, Group string
	Body        *decad.Body
}

// Assembly records the placed parts, every owned sketch and the solved pose.
type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
}

// Build adds the 43 assembly parts to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*Assembly, error) {
	if ctx == nil {
		return nil, fmt.Errorf("pedalspindle: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("pedalspindle: document must not be nil")
	}
	state, err := Motion(p)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("pedalspindle: before creating parts: %w", err)
	}
	assembly := &Assembly{State: state, Parts: make([]Part, 0, 43), Sketches: make([]*sketch.Sketch, 0, 39)}
	c := &construction{ctx: ctx, doc: doc, assembly: assembly}
	if err := c.frame(p.Scale); err != nil {
		return nil, err
	}
	if err := c.bearings(p.Scale, state); err != nil {
		return nil, err
	}
	if err := c.drive(p, state); err != nil {
		return nil, err
	}
	if err := c.gears(p, state); err != nil {
		return nil, err
	}
	if err := c.linkage(p, state); err != nil {
		return nil, err
	}
	if len(assembly.Parts) != 43 || len(assembly.Sketches) != 39 {
		return nil, fmt.Errorf("pedalspindle: assembly has %d parts and %d sketches, want 43 and 39",
			len(assembly.Parts), len(assembly.Sketches))
	}
	for _, part := range assembly.Parts {
		if !part.Body.IsSolid() {
			return nil, fmt.Errorf("pedalspindle: %s: final body is not solid", part.Name)
		}
	}
	return assembly, nil
}

func (c *construction) frame(s float64) error {
	for _, b := range []struct {
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
	} {
		if err := c.box(b.name, b.group, b.x1, b.x2, b.y1, b.y2, b.z1, b.z2, s); err != nil {
			return err
		}
	}
	if err := c.plate("bearing-board", "frame", r3.NewVec(-185*s, 18*s, 290*s), 0,
		0, 0, 295*s, 235*s, 18*s, []hole{{95 * s, 90 * s, 22}, {197.3 * s, 90 * s, 22}, {197.3 * s, 192.3 * s, 14}}); err != nil {
		return err
	}
	if err := c.plate("guard-front", "guard", r3.NewVec(-175*s, -90*s, 295*s), 0,
		0, 0, 275*s, 225*s, 4*s, []hole{{85 * s, 85 * s, 8.8}, {187.3 * s, 85 * s, 8.8}, {187.3 * s, 187.3 * s, 5.8}}); err != nil {
		return err
	}
	return c.plate("heel-bracket", "frame", r3.NewVec(-244*s, -132*s, 18*s), 0,
		0, 0, 28*s, 58*s, 8*s, []hole{{14 * s, 42 * s, 5.2}})
}

func (c *construction) bearings(s float64, state State) error {
	for _, spec := range []struct {
		name   string
		center r3.Vec
		y      float64
		small  bool
	}{
		{"input-bearing-front", state.Input, 7, false},
		{"input-bearing-rear", state.Input, 18 * s, false},
		{"intermediate-bearing-front", state.Intermediate, 7, false},
		{"intermediate-bearing-rear", state.Intermediate, 18 * s, false},
		{"output-bearing-front", state.Output, 5, true},
		{"output-bearing-rear", state.Output, 18 * s, true},
		{"heel-bearing-front", state.Heel, -210*s + 5, true},
		{"heel-bearing-rear", state.Heel, -140 * s, true},
		{"footman-bearing-crank", state.CrankPin, -125*s + 2.5, true},
		{"footman-bearing-toe", state.Toe, -125*s + 2.5, true},
	} {
		p := ballbearing.DefaultParams()
		if spec.small {
			p = ballbearing.Params605ZZ()
		}
		f, err := xzFrame(r3.NewVec(spec.center.X, spec.y, spec.center.Z), 0)
		if err != nil {
			return fmt.Errorf("pedalspindle: %s: create bearing frame: %w", spec.name, err)
		}
		p.Placement = &f
		model, err := ballbearing.Build(c.ctx, c.doc, p)
		if err != nil {
			return fmt.Errorf("pedalspindle: %s: build bearing: %w", spec.name, err)
		}
		c.assembly.Sketches = append(c.assembly.Sketches, model.Sketch)
		if err := c.append(spec.name, "bearing", model.Body); err != nil {
			return err
		}
	}
	return nil
}

func (c *construction) drive(p Params, state State) error {
	s := p.Scale
	for _, spec := range []struct {
		name, group             string
		center                  r3.Vec
		y, radius, bore, length float64
	}{
		{"input-shaft", "steel", state.Input, 65 * s, 4, 0, 181 * s},
		{"intermediate-shaft", "steel", state.Intermediate, 26 * s, 4, 0, 109 * s},
		{"flywheel-rim", "drive", state.Input, 53 * s, 155 * s, 270 * s, 18 * s},
		{"flywheel-hub", "drive", state.Input, 53 * s, 17 * s, 8.2, 18 * s},
		{"input-hub", "steel", state.Input, -44 * s, 12 * s, 8.2, 8 * s},
		{"intermediate-pinion-hub", "steel", state.Intermediate, -44 * s, 12 * s, 8.2, 8 * s},
		{"intermediate-gear-hub", "steel", state.Intermediate, -74 * s, 12 * s, 8.2, 8 * s},
		{"output-hub", "steel", state.Output, -74 * s, 12 * s, 5.2, 8 * s},
	} {
		if err := c.cylinder(spec.name, spec.group, spec.center, spec.y, spec.radius, spec.bore, spec.length); err != nil {
			return err
		}
	}
	if err := c.spindle(state.Output, s); err != nil {
		return err
	}
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	for k := 0; k < 4; k++ {
		name := fmt.Sprintf("flywheel-spoke-%d", k)
		if err := c.plate(name, "drive", r3.NewVec(state.Input.X, 49*s, state.Input.Z), a+float64(k)*math.Pi/2,
			15*s, -6*s, 137*s, 6*s, 10*s, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *construction) gears(p Params, state State) error {
	s := p.Scale
	params := spurgear.DefaultParams()
	params.Module = 3 * s
	params.Thickness = 12 * s
	params.PressureAngle = 20 * math.Pi / 180
	params.Samples = 15
	params.CapChamfer = 0
	params.Teeth, params.BoreDiameter = 51, 8.2
	large, err := spurgear.Build(c.ctx, c.doc, params)
	if err != nil {
		return fmt.Errorf("pedalspindle: input-gear: build 51-tooth prototype: %w", err)
	}
	input, err := decad.Copy(c.ctx, large)
	if err != nil {
		return fmt.Errorf("pedalspindle: input-gear: copy 51-tooth prototype: %w", err)
	}
	params.Teeth = 17
	small, err := spurgear.Build(c.ctx, c.doc, params)
	if err != nil {
		return fmt.Errorf("pedalspindle: intermediate-pinion: build 17-tooth prototype: %w", err)
	}
	params.BoreDiameter = 5.2
	output, err := spurgear.Build(c.ctx, c.doc, params)
	if err != nil {
		return fmt.Errorf("pedalspindle: output-pinion: build 17-tooth prototype: %w", err)
	}
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	for _, spec := range []struct {
		name     string
		body     *decad.Body
		center   r3.Vec
		y, angle float64
	}{
		{"input-gear", input, state.Input, -30 * s, a},
		{"intermediate-pinion", small, state.Intermediate, -30 * s, state.IntermediateAngle},
		{"intermediate-gear", large, state.Intermediate, -60 * s, state.IntermediateAngle},
		{"output-pinion", output, state.Output, -60 * s, state.SpindleAngle},
	} {
		f, err := xzFrame(r3.NewVec(spec.center.X, spec.y, spec.center.Z), spec.angle)
		if err != nil {
			return fmt.Errorf("pedalspindle: %s: create gear frame: %w", spec.name, err)
		}
		b, err := c.place(spec.name, spec.body, f)
		if err != nil {
			return err
		}
		if err := c.append(spec.name, "gear", b); err != nil {
			return err
		}
	}
	return nil
}

func (c *construction) linkage(p Params, state State) error {
	s := p.Scale
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	if err := c.plate("crank", "drive", r3.NewVec(state.Input.X, -105*s, state.Input.Z), a,
		-9*s, -9*s, 34*s, 9*s, 8*s, []hole{{0, 0, 8.2}, {25 * s, 0, 5.2}}); err != nil {
		return err
	}
	if err := c.plate("treadle", "pedal", r3.NewVec(state.Heel.X, -140*s, state.Heel.Z), state.PedalAngle,
		-12*s, -12*s, 212*s, 12*s, 70*s, []hole{{0, 0, 14}, {200 * s, 0, 5.2}}); err != nil {
		return err
	}
	angle := math.Atan2(state.CrankPin.Z-state.Toe.Z, state.CrankPin.X-state.Toe.X)
	if err := c.plate("footman", "pedal", r3.NewVec(state.Toe.X, -121*s, state.Toe.Z), angle,
		-12*s, -12*s, 332*s, 12*s, 8*s, []hole{{0, 0, 14}, {320 * s, 0, 14}}); err != nil {
		return err
	}
	for _, spec := range []struct {
		name      string
		center    r3.Vec
		y, length float64
	}{
		{"crank-pin", state.CrankPin, -105 * s, 27 * s},
		{"heel-pin", state.Heel, -135 * s, 80 * s},
		{"toe-pin", state.Toe, -118 * s, 97 * s},
	} {
		if err := c.cylinder(spec.name, "steel", spec.center, spec.y, 2.5, 0, spec.length); err != nil {
			return err
		}
	}
	return nil
}
