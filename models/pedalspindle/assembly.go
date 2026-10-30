package pedalspindle

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
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
		return nil, fmt.Errorf("pedalspindle: before construction: %w", err)
	}
	assembly := &Assembly{State: state}
	b := builder{ctx: ctx, doc: doc, assembly: assembly, scale: p.Scale}
	s := p.Scale
	a := math.Mod(p.CrankAngle, 2*math.Pi)
	b.frame()
	b.gears(a, state.IntermediateAngle, state.SpindleAngle)
	input := b.disc(4, 0, 181*s, 0)
	input = b.flats(input, 4, 3, [][2]float64{{169 * s, 181*s + 1}, {85 * s, 148 * s}, {-1, 35 * s}})
	b.add("input-shaft", "steel", b.place(input, state.Input.X, 65*s, state.Input.Z, a))
	intermediate := b.disc(4, 0, 126*s, 0)
	intermediate = b.flats(intermediate, 4, 3, [][2]float64{{46 * s, 109 * s}})
	b.add("intermediate-shaft", "steel", b.place(intermediate, state.Intermediate.X, 26*s, state.Intermediate.Z, state.IntermediateAngle))
	spindle := b.flats(b.spindle(), 2.5, 1.75, [][2]float64{{80 * s, 114 * s}})
	b.add("spindle", "steel", b.place(spindle, state.Output.X, 30*s, state.Output.Z, state.SpindleAngle))
	b.ring("flywheel-rim", "drive", state.Input.X, state.Input.Z, 35*s, 53*s, 310*s, 270*s, 0)
	hub := b.dBore(b.disc(17*s, 0, 18*s, 0), 4.1, 3.1, 18*s)
	b.add("flywheel-hub", "drive", b.place(hub, state.Input.X, 53*s, state.Input.Z, a))
	for i := range 4 {
		spoke := b.rect(15*s, -6*s, 137*s, 6*s, 10*s, 0)
		b.add(fmt.Sprintf("flywheel-spoke-%d", i), "drive", b.place(spoke, state.Input.X, 49*s, state.Input.Z, a+float64(i)*math.Pi/2))
	}
	for _, h := range []struct {
		name             string
		x, z, y0, y1, id float64
	}{
		{"input-hub", -90, 380, -52, -44, 8.2}, {"intermediate-pinion-hub", 12.3, 380, -52, -44, 8.2},
		{"intermediate-gear-hub", 12.3, 380, -82, -74, 8.2}, {"output-hub", 12.3, 482.3, -82, -74, 5.2},
	} {
		b.ring(h.name, "steel", h.x*s, h.z*s, h.y0*s, h.y1*s, 24*s, h.id, 0)
	}
	crank := b.rect(-9*s, -9*s, 34*s, 9*s, 8*s, 0, hole{25 * s, 0, 5.2})
	crank = b.dBore(crank, 4.1, 3.1, 8*s)
	b.add("crank", "drive", b.place(crank, state.Input.X, -105*s, state.Input.Z, a))
	b.ring("crank-pin", "steel", state.CrankPin.X, state.CrankPin.Z, -132*s, -105*s, 5, 0, 0)
	b.add("treadle", "pedal", b.place(b.treadle(), state.Heel.X, -140*s, state.Heel.Z, state.PedalAngle))
	rodAngle := math.Atan2(state.CrankPin.Z-state.Toe.Z, state.CrankPin.X-state.Toe.X)
	rod := b.rect(-12*s, -12*s, 332*s, 12*s, 8*s, 0, hole{0, 0, 14}, hole{320 * s, 0, 14})
	b.add("footman", "pedal", b.place(rod, state.Toe.X, -121*s, state.Toe.Z, rodAngle))
	b.ring("heel-pin", "steel", state.Heel.X, state.Heel.Z, -230*s, -120*s, 8, 0, 0)
	b.ring("toe-pin", "steel", state.Toe.X, state.Toe.Z, -215*s, -118*s, 5, 0, 0)
	b.heelSupports()
	b.bearings()
	b.guards()
	if b.err != nil {
		return nil, b.err
	}
	if len(assembly.Parts) != 61 || len(assembly.Sketches) != 72 {
		return nil, fmt.Errorf("pedalspindle: constructed %d parts and %d sketches; expected 61 and 72", len(assembly.Parts), len(assembly.Sketches))
	}
	return assembly, nil
}

func (b *builder) treadle() *decad.Body {
	s := b.section(false, 0)
	if b.err != nil {
		return nil
	}
	k := b.scale
	coords := [][2]float64{{-18, -18}, {18, -18}, {18, -12}, {212, -12}, {212, 12}, {18, 12}, {18, 18}, {-18, 18}}
	for i := range coords {
		coords[i][0] *= k
		coords[i][1] *= k
	}
	ps, ls := polygon(s, coords, false)
	for _, i := range []int{0, 2, 4, 6} {
		s.AddConstraint(sketch.NewHorizontal(ls[i]))
	}
	for _, i := range []int{1, 3, 5, 7} {
		s.AddConstraint(sketch.NewVertical(ls[i]))
	}
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[0], -18*k), sketch.NewVerticalDistance(s.Origin(), ps[0], -18*k))
	for i, d := range []float64{36, 194, -194} {
		j := 2 * i
		s.AddConstraint(sketch.NewHorizontalDistance(ps[j], ps[j+1], d*k))
	}
	for i, d := range []float64{6, 24, 6} {
		j := 2*i + 1
		s.AddConstraint(sketch.NewVerticalDistance(ps[j], ps[j+1], d*k))
	}
	holes(s, []hole{{0, 0, 22}, {200 * k, 0, 5.2}})
	return b.extrude(s, 2, 70*k)
}

func (b *builder) heelSupports() {
	s := b.scale
	for _, p := range []struct {
		name   string
		y0, y1 float64
	}{
		{"heel-bracket-front", -222, -214}, {"heel-bracket-rear", -136, -128},
	} {
		body := b.rect(-250*s, 18*s, -210*s, 84*s, (p.y1-p.y0)*s, 0, hole{-230 * s, 60 * s, 8.2})
		b.add(p.name, "frame", b.place(body, 0, p.y1*s, 0, 0))
	}
	b.ring("heel-collar-front", "steel", -230*s, 60*s, -228*s, -222*s, 16*s, 8.2, 0)
	b.ring("heel-collar-rear", "steel", -230*s, 60*s, -128*s, -122*s, 16*s, 8.2, 0)
	b.ring("heel-spacer-front", "steel", -230*s, 60*s, -214*s, -210.1*s, 12, 8.2, 0)
	b.ring("heel-spacer-rear", "steel", -230*s, 60*s, -139.9*s, -136*s, 12, 8.2, 0)
}

func (b *builder) bearing(name string, x, z, yBack float64, small bool) {
	if b.err != nil {
		return
	}
	p := ballbearing.DefaultParams()
	if small {
		p = ballbearing.Params605ZZ()
	}
	f, err := r3.NewFrame(r3.NewVec(x, yBack, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
	if err != nil {
		b.err = fmt.Errorf("bearing placement: %w", err)
		b.add(name, "bearing", nil)
		return
	}
	p.Placement = &f
	m, err := ballbearing.Build(b.ctx, b.doc, p)
	if err != nil {
		b.err = fmt.Errorf("construct bearing: %w", err)
		b.add(name, "bearing", nil)
		return
	}
	b.assembly.Sketches = append(b.assembly.Sketches, m.Sketch)
	b.add(name, "bearing", m.Body)
}

func (b *builder) bearings() {
	s := b.scale
	for _, p := range []struct {
		name  string
		x, z  float64
		small bool
	}{
		{"input", -90, 380, false}, {"intermediate", 12.3, 380, false}, {"output", 12.3, 482.3, true},
	} {
		width := 7.0
		if p.small {
			width = 5
		}
		b.bearing(p.name+"-bearing-front", p.x*s, p.z*s, -95*s+width, p.small)
		b.bearing(p.name+"-bearing-rear", p.x*s, p.z*s, 18*s, p.small)
	}
	b.bearing("heel-bearing-front", -230*s, 60*s, -210*s+7, false)
	b.bearing("heel-bearing-rear", -230*s, 60*s, -140*s, false)
	state := b.assembly.State
	b.bearing("footman-bearing-crank", state.CrankPin.X, state.CrankPin.Z, -125*s+2.5, true)
	b.bearing("footman-bearing-toe", state.Toe.X, state.Toe.Z, -125*s+2.5, true)
}

func (b *builder) guards() {
	s := b.scale
	front := b.rect(-179*s, 294*s, 104*s, 514*s, 3*s, 0,
		hole{-90 * s, 380 * s, 8.8}, hole{12.3 * s, 380 * s, 8.8}, hole{12.3 * s, 482.3 * s, 5.8})
	b.add("guard-front", "guard", b.place(front, 0, -101*s, 0, 0))
	b.box("guard-left", "guard", -183, -179, -104, 0, 290, 518)
	b.box("guard-right", "guard", 104, 108, -104, 0, 290, 518)
	b.box("guard-bottom", "guard", -179, 104, -104, 0, 290, 294)
	b.box("guard-top", "guard", -179, 104, -104, 0, 514, 518)
}
