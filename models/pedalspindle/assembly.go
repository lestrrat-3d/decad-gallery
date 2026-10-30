package pedalspindle

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/sketch"
)

// Part associates a separate solid with its stable name and renderer group.
type Part struct {
	Name, Group string
	Body        *decad.Body
}

// Assembly contains the parts, all locally owned sketches, and solved motion.
type Assembly struct {
	Parts    []Part
	Sketches []*sketch.Sketch
	State    State
}

// Build constructs the assembly. ctx and doc must not be nil.
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
		return nil, fmt.Errorf("pedalspindle: build: %w", err)
	}
	assembly := &Assembly{State: state}
	b := &construction{ctx: ctx, doc: doc, scale: p.Scale, world: sketch.NewWorld(), assembly: assembly}
	if err := b.frame(); err != nil {
		return nil, err
	}
	if err := b.shafts(); err != nil {
		return nil, err
	}
	if err := b.gears(p); err != nil {
		return nil, err
	}
	if err := b.flywheel(p); err != nil {
		return nil, err
	}
	if err := b.linkage(p); err != nil {
		return nil, err
	}
	if err := b.guard(); err != nil {
		return nil, err
	}
	return assembly, nil
}

func (b *construction) frame() error {
	for _, part := range []struct {
		name                   string
		x1, x2, y1, y2, z1, z2 float64
	}{
		{"base", -270, 150, -240, 110, 0, 18},
		{"left-leg", -180, -156, 0, 18, 18, 290},
		{"right-leg", 80, 104, 0, 18, 18, 290},
	} {
		if err := b.box(part.name, "frame", part.x1, part.x2, part.y1, part.y2, part.z1, part.z2); err != nil {
			return err
		}
	}
	holes := []sectionHole{{95, 90, 16}, {197.3, 90, 16}, {197.3, 192.3, 16}}
	return b.rectangleXZ("bearing-board", "frame", 0, 0, 295, 235, -185, 18, 290, 0, 18, holes)
}

func (b *construction) shafts() error {
	for _, part := range []struct {
		name string
		x, z float64
	}{
		{"input-bush", -90, 380}, {"intermediate-bush", 12.3, 380}, {"output-bush", 12.3, 482.3},
	} {
		if err := b.roundXZ(part.name, "bush", 16, 10.2, part.x, part.z, -1, 19); err != nil {
			return err
		}
	}
	if err := b.roundXZ("input-shaft", "steel", 10, 0, -90, 380, -116, 65); err != nil {
		return err
	}
	if err := b.roundXZ("intermediate-shaft", "steel", 10, 0, 12.3, 380, -83, 26); err != nil {
		return err
	}
	if err := b.spindle(); err != nil {
		return err
	}
	for _, part := range []struct {
		name         string
		x, z, y1, y2 float64
	}{
		{"input-hub", -90, 380, -52, -44}, {"intermediate-pinion-hub", 12.3, 380, -52, -44},
		{"intermediate-gear-hub", 12.3, 380, -82, -74}, {"output-hub", 12.3, 482.3, -82, -74},
	} {
		if err := b.roundXZ(part.name, "steel", 24, 10.2, part.x, part.z, part.y1, part.y2); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) gears(p Params) error {
	gp := spurgear.Params{Module: 3 * p.Scale, PressureAngle: 20 * math.Pi / 180, Thickness: 12 * p.Scale,
		Samples: 15, BoreDiameter: 10.2 * p.Scale, CapChamfer: 0}
	gp.Teeth = 51
	large, err := spurgear.Build(b.ctx, b.doc, gp)
	if err != nil {
		return fmt.Errorf("input-gear: build 51-tooth prototype: %w", err)
	}
	gp.Teeth = 17
	small, err := spurgear.Build(b.ctx, b.doc, gp)
	if err != nil {
		return fmt.Errorf("intermediate-pinion: build 17-tooth prototype: %w", err)
	}
	a := math.Remainder(p.CrankAngle, 2*math.Pi)
	phaseB := math.Pi + math.Pi/17 - 3*a
	phaseC := 3*math.Pi + math.Pi/17 - 3*phaseB
	for _, part := range []struct {
		name           string
		body           *decad.Body
		copyPrototype  bool
		x, y, z, angle float64
	}{
		{"input-gear", large, true, -90, -30, 380, a},
		{"intermediate-pinion", small, true, 12.3, -30, 380, phaseB},
		{"intermediate-gear", large, false, 12.3, -60, 380, phaseB},
		{"output-pinion", small, false, 12.3, -60, 482.3, phaseC},
	} {
		body := part.body
		if part.copyPrototype {
			body, err = decad.Copy(b.ctx, part.body)
			if err != nil {
				return fmt.Errorf("%s: copy gear prototype: %w", part.name, err)
			}
		}
		if err := b.placeXZ(part.name, "gear", body, part.x, part.y, part.z, part.angle); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) flywheel(p Params) error {
	if err := b.roundXZ("flywheel-rim", "drive", 310, 270, -90, 380, 35, 53); err != nil {
		return err
	}
	if err := b.roundXZ("flywheel-hub", "drive", 34, 10.2, -90, 380, 35, 53); err != nil {
		return err
	}
	a := math.Remainder(p.CrankAngle, 2*math.Pi)
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("flywheel-spoke-%d", i)
		if err := b.rectangleXZ(name, "drive", 15, -6, 137, 6, -90, 49, 380, a+float64(i)*math.Pi/2, 10, nil); err != nil {
			return err
		}
	}
	return nil
}

func (b *construction) linkage(p Params) error {
	s := b.assembly.State
	k := b.scale
	a := math.Remainder(p.CrankAngle, 2*math.Pi)
	if err := b.rectangleXZ("crank", "drive", -9, -9, 34, 9, -90, -105, 380, a, 8,
		[]sectionHole{{0, 0, 10.2}, {25, 0, 8.4}}); err != nil {
		return err
	}
	if err := b.roundXZ("crank-pin", "steel", 8, 0, s.CrankPin.X/k, s.CrankPin.Z/k, -116, -94); err != nil {
		return err
	}
	if err := b.rectangleXZ("treadle", "pedal", -12, -8, 212, 8, -230, -120, 60, s.PedalAngle, 90,
		[]sectionHole{{0, 0, 8.4}, {200, 0, 8.4}}); err != nil {
		return err
	}
	rodAngle := math.Atan2(s.CrankPin.Z-s.Toe.Z, s.CrankPin.X-s.Toe.X)
	if err := b.rectangleXZ("footman", "pedal", -10, -9, 330, 9, s.Toe.X/k, -97, s.Toe.Z/k, rodAngle, 8,
		[]sectionHole{{0, 0, 8.4}, {320, 0, 8.4}}); err != nil {
		return err
	}
	if err := b.roundXZ("heel-pin", "steel", 8, 0, -230, 60, -215, -115); err != nil {
		return err
	}
	if err := b.roundXZ("toe-pin", "steel", 8, 0, s.Toe.X/k, s.Toe.Z/k, -215, -94); err != nil {
		return err
	}
	return b.rectangleXZ("heel-bracket", "frame", 0, 0, 28, 58, -244, -112, 18, 0, 8, []sectionHole{{14, 42, 8.4}})
}

func (b *construction) guard() error {
	if err := b.rectangleXZ("guard-front", "guard", 0, 0, 275, 225, -175, -90, 295, 0, 4,
		[]sectionHole{{85, 85, 10.8}, {187.3, 85, 10.8}, {187.3, 187.3, 10.8}}); err != nil {
		return err
	}
	for _, part := range []struct {
		name           string
		x1, x2, z1, z2 float64
	}{
		{"guard-left", -179, -175, 291, 524}, {"guard-right", 100, 104, 291, 524},
		{"guard-bottom", -175, 100, 291, 295}, {"guard-top", -175, 100, 520, 524},
	} {
		if err := b.box(part.name, "guard", part.x1, part.x2, -94, 0, part.z1, part.z2); err != nil {
			return err
		}
	}
	return nil
}
