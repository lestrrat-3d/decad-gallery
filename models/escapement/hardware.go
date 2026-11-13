package escapement

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Bearing lower-cap heights in mm: the back bearings sit in the back plate's seats, the front ones against the
// front plate's seats, the pendulum bearing in the pendulum's pocket.
const (
	backBearingZ     = -5.0
	frontBearingZ    = 28.0
	pendulumBearingZ = 40.05
)

// Hardware holds the purchased parts: seven 605ZZ bearings, four bolts and four nuts.
type Hardware struct {
	BackEscape, BackPallet, BackGreat    *decad.Body
	FrontEscape, FrontPallet, FrontGreat *decad.Body
	Pendulum                             *decad.Body
	Bolts, Nuts                          []*decad.Body
}

// bearing builds a 605ZZ envelope with its lower cap centre at (at, z) and its axis along +Z.
func (b *builder) bearing(at Pt, z float64) (*decad.Body, error) {
	f, err := r3.NewFrame(r3.NewVec(at.X, at.Y, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("bearing frame: %w", err)
	}
	p := ballbearing.Params605ZZ()
	p.Placement = &f
	m, err := ballbearing.Build(b.ctx, b.doc, p)
	if err != nil {
		return nil, fmt.Errorf("bearing: %w", err)
	}
	b.sketches = append(b.sketches, m.Sketch)
	return m.Body, nil
}

// buildHardware builds the bearings, bolts and nuts at their places.
func (b *builder) buildHardware() (*Hardware, error) {
	g := &b.g
	var h Hardware
	for _, e := range []struct {
		dst **decad.Body
		at  Pt
		z   float64
	}{
		{&h.BackEscape, g.O, backBearingZ}, {&h.BackPallet, g.P, backBearingZ}, {&h.BackGreat, g.G, backBearingZ},
		{&h.FrontEscape, g.O, frontBearingZ}, {&h.FrontPallet, g.P, frontBearingZ}, {&h.FrontGreat, g.G, frontBearingZ},
		{&h.Pendulum, g.S, pendulumBearingZ},
	} {
		body, err := b.bearing(e.at, e.z)
		if err != nil {
			return nil, err
		}
		*e.dst = body
	}
	for i, s := range sites() {
		bolt, err := m2fastener.PlaceBolt(b.ctx, b.doc, s)
		if err != nil {
			return nil, fmt.Errorf("bolt %d: %w", i+1, err)
		}
		b.sketches = append(b.sketches, bolt.Profile, bolt.Socket)
		h.Bolts = append(h.Bolts, bolt.Body)
		nut, err := m2fastener.PlaceNut(b.ctx, b.doc, s)
		if err != nil {
			return nil, fmt.Errorf("nut %d: %w", i+1, err)
		}
		b.sketches = append(b.sketches, nut.Hexagon, nut.LowerChamfer, nut.UpperChamfer)
		h.Nuts = append(h.Nuts, nut.Body)
	}
	return &h, nil
}

// m2Tip is the distance from a bolt's head top to its tip.
var m2Tip = sites()[0].Tip()
