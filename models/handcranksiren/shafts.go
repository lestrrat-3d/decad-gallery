package handcranksiren

import "github.com/lestrrat-3d/decad"

// Shaft and collar dimensions in mm. Each shaft is a D-shaft; the rotor shaft and the crank shaft also carry a collar.
const (
	collarRadius = 5.5

	rotorShaftFrom = -25.25
	rotorShaftTo   = 35.70
	rotorCollar    = 10.1 // the collar spans [10.1, 12.1]
	interShaftFrom = 2.90
	interShaftTo   = 34.70
	crankShaftFrom = 2.90
	crankShaftTo   = 45.35
	crankCollar    = 19.1 // the collar spans [19.1, 20.1]
)

// shafts builds the three shafts.
func (b *builder) shafts() {
	rotor, inter, crank := b.axes()
	b.part("crank-shaft", ShaftsContainerName, "crank", func() *decad.Body {
		return b.collared(crank, crankShaftFrom, crankShaftTo, crankCollar, 1)
	})
	b.part("intermediate-shaft", ShaftsContainerName, "intermediate", func() *decad.Body {
		return b.dShaft(inter, interShaftFrom, interShaftTo)
	})
	b.part("rotor-shaft", ShaftsContainerName, "rotor", func() *decad.Body {
		return b.collared(rotor, rotorShaftFrom, rotorShaftTo, rotorCollar, 2)
	})
}

// collared builds a D-shaft with a collar of the given length from Y collar. The flat is cut before the union, because
// Decad refuses to cut after it.
func (b *builder) collared(a shaftAxis, from, to, collar, length float64) *decad.Body {
	shaft := b.dShaft(a, from, to)
	return b.union(shaft, b.cylinder(a.across(collar), collarRadius, length))
}

// axes returns the rotor, intermediate and crank axes.
func (b *builder) axes() (rotor, inter, crank shaftAxis) {
	s := b.a.State
	return shaftAxis{s.Rotor}, shaftAxis{s.Intermediate}, shaftAxis{s.Crank}
}
