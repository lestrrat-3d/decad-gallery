package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Bearing and seat dimensions in mm. The six bearings are 608-2RS (8 × 22 × 7). Each sits in a round seat 0.05 larger
// than its outside radius (the bearing-seat rule), behind a relief bore that frees the inner ring.
const (
	bearingWidth     = 7.0
	bearingRadius    = 11.0
	seatRadius       = bearingRadius + seatClearance // 11.05
	reliefRadius     = 9.5
	shaftHoleRadius  = 4.2
	frontBearingFrom = 2.85
	rearBearingFrom  = 27.75
)

// bearingSite is one bearing: its part name, the axis it carries, the Y of its start face and the holder.
type bearingSite struct {
	name, holder string
	axis         func(State) r3.Vec
	start        float64
}

var bearingSites = [...]bearingSite{
	{"bearing-rotor-front", "gearbox-front", func(s State) r3.Vec { return s.Rotor }, frontBearingFrom},
	{"bearing-intermediate-front", "gearbox-front", func(s State) r3.Vec { return s.Intermediate }, frontBearingFrom},
	{"bearing-crank-front", "gearbox-front", func(s State) r3.Vec { return s.Crank }, frontBearingFrom},
	{"bearing-rotor-rear", "gearbox-rear", func(s State) r3.Vec { return s.Rotor }, rearBearingFrom},
	{"bearing-intermediate-rear", "gearbox-rear", func(s State) r3.Vec { return s.Intermediate }, rearBearingFrom},
	{"bearing-crank-rear", "gearbox-rear", func(s State) r3.Vec { return s.Crank }, rearBearingFrom},
}

// bearings builds the purchased bearings.
func (b *builder) bearings() {
	for _, site := range bearingSites {
		b.part(site.name, BearingsContainerName, "fixed", func() *decad.Body {
			return b.bearing(shaftAxis{site.axis(b.a.State)}, site.start)
		})
	}
}

// bearing builds the 608-2RS envelope that sits on the axis from Y start to start + 7.
func (b *builder) bearing(a shaftAxis, start float64) *decad.Body {
	if b.err != nil {
		return nil
	}
	f := a.across(start)
	bp := ballbearing.DefaultParams()
	bp.Placement = &f
	m, err := ballbearing.Build(b.ctx, b.doc, bp)
	b.fail("build bearing", err)
	if err != nil {
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, m.Sketch)
	return m.Body
}
