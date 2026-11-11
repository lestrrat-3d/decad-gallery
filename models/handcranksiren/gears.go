package handcranksiren

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/gearroot"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Gear train dimensions in mm. Stage 1 is a 50-tooth crank gear driving a 16-tooth pinion on the intermediate axis. Stage
// 2 is a 48-tooth gear on the same axis driving a 15-tooth rotor pinion. The gears of one stage share a plane.
const (
	gearModule   = 1.5
	gearThick    = 6.0
	gearBacklash = 0.15
	gearMargin   = 0.02 // measurement margin: the root goes this much deeper and the tip this much lower

	plane1Front = 20.6 // crank-gear, intermediate-pinion
	plane2Front = 12.6 // intermediate-gear, rotor-pinion

	boreClearance = 0.25 // tightFit: the gear's D-bore is this much larger than the shaft
	boreFrom      = 1.0  // the bore tool starts this far before the gear's front face
	boreLength    = 8.0

	// A stage mark is one through hole (stage 1) or two (stage 2) on the face band, so the 50- and 48-tooth gears and
	// the 16- and 15-tooth pinions, which look alike, can be told apart.
	markRadius = 6.6  // the holes' circle about the gear axis: clear of the D-bore and of every root circle
	markHole   = 0.75 // hole radius
	markSpread = 25.0 // angle between neighbouring holes, in degrees
)

// gearSite is one gear of the train.
type gearSite struct {
	name   string
	teeth  int
	axis   func(State) r3.Vec
	front  float64
	motion string
	// toward is the axis whose direction the gear's tooth zero points at; a driven gear is turned half a tooth so that
	// its gap points there.
	toward func(State) r3.Vec
	driven bool
	cap    float64 // CapChamfer
	stage  int     // the number of mark holes
}

var gearSites = [...]gearSite{
	{"crank-gear", 50, func(s State) r3.Vec { return s.Crank }, plane1Front, "crank",
		func(s State) r3.Vec { return s.Intermediate }, false, 0.2, 1},
	{"intermediate-pinion", 16, func(s State) r3.Vec { return s.Intermediate }, plane1Front, "intermediate",
		func(s State) r3.Vec { return s.Crank }, true, 0, 1},
	{"intermediate-gear", 48, func(s State) r3.Vec { return s.Intermediate }, plane2Front, "intermediate",
		func(s State) r3.Vec { return s.Rotor }, false, 0.2, 2},
	{"rotor-pinion", 15, func(s State) r3.Vec { return s.Rotor }, plane2Front, "rotor",
		func(s State) r3.Vec { return s.Intermediate }, true, 0, 2},
}

// gears builds the four gears.
func (b *builder) gears() {
	for _, site := range gearSites {
		b.part(site.name, GearPairsContainerName, site.motion, func() *decad.Body { return b.gear(site) })
	}
}

// gear builds one gear at crank angle 0: tooth zero centred on U, which points at the mate (a driven gear is turned
// half a tooth), then the D-bore.
func (b *builder) gear(site gearSite) *decad.Body {
	if b.err != nil {
		return nil
	}
	axis := site.axis(b.a.State)
	u := site.toward(b.a.State).Sub(axis)
	u.Y = 0
	u = u.Scale(1 / u.Len())
	if site.driven {
		xf, err := r3.RotationAround(r3.Vec{}, r3.Vec{Y: 1}, units.Radians(math.Pi/float64(site.teeth)))
		b.fail("create half-tooth turn", err)
		if err != nil {
			return nil
		}
		u = xf.ApplyDir(u)
	}
	f := frame(axis.Add(r3.Vec{Y: site.front}), u, r3.Vec{Y: 1}.Cross(u))
	gp := spurgear.DefaultParams()
	gp.Module, gp.Teeth, gp.Thickness, gp.BoreDiameter, gp.Placement = gearModule, site.teeth, gearThick, 0, &f
	gp.Backlash, gp.CapChamfer = gearBacklash, site.cap
	body, err := spurgear.Variant{Define: gearroot.Define(gearroot.Clearance(gearModule), gearMargin)}.Build(b.ctx, b.doc, gp)
	b.fail("build spur gear", err)
	if err != nil {
		return nil
	}
	body = b.cut(body, b.dBoreTool(shaftAxis{axis}, site.front-boreFrom, boreLength, boreClearance))
	return b.markHoles(body, shaftAxis{axis}, site)
}

// markHoles cuts site.stage through holes on the +V side of the gear's section frame, opposite the D flat, spread
// symmetrically about the V axis.
func (b *builder) markHoles(body *decad.Body, a shaftAxis, site gearSite) *decad.Body {
	for i := 0; i < site.stage; i++ {
		angle := (90 + markSpread*(float64(i)-float64(site.stage-1)/2)) * math.Pi / 180
		s := b.section(a.across(site.front - boreFrom))
		circle(s, markRadius*math.Cos(angle), markRadius*math.Sin(angle), markHole)
		body = b.cut(body, b.extrude(s, 0, boreLength))
	}
	return body
}
