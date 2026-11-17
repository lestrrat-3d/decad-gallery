package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
)

// Shaft and collar dimensions in mm. Each shaft is a D-shaft; the rotor shaft and the crank shaft also carry a collar.
const (
	collarRadius = 5.5

	rotorShaftFrom = -25.25
	rotorShaftTo   = 35.70
	rotorCollar    = 10.1 // the collar spans [10.1, 12.1]
	interShaftFrom = 2.90
	interShaftTo   = 34.70
	crankShaftFrom = 2.90
	// crankShaftTo is the arm's back face (armFace) plus the hub's depth to its bore floor, less 0.25.
	crankShaftTo = armFace + crankhandle.HubLength - crankhandle.BoreFloor - 0.25
	// crankBackFlatFrom is the Y where the crank shaft's -Z flat reaches full depth, 0.25 before the arm's back face.
	crankBackFlatFrom = armFace - 0.25
	crankCollar       = 19.1 // the collar spans [19.1, 20.1]
)

// shafts builds the three shafts.
func (b *builder) shafts() {
	rotor, inter, crank := b.axes()
	b.part("crank-shaft", ShaftsContainerName, "crank", func() *decad.Body {
		return b.crankShaft(crank)
	})
	b.part("intermediate-shaft", ShaftsContainerName, "intermediate", func() *decad.Body {
		return b.dShaft(inter, interShaftFrom, interShaftTo)
	})
	b.part("rotor-shaft", ShaftsContainerName, "rotor", func() *decad.Body {
		return b.collared(rotor, rotorShaftFrom, rotorShaftTo, rotorCollar, 2)
	})
}

// crankShaft builds the crank shaft: a D-shaft with a second flat on the -Z side over the crank hub's bore, a double
// flat that ends in a 30 degree ramp (backFlatTool), and the collar. Both cuts precede the union, because Decad refuses
// a cut after it.
func (b *builder) crankShaft(a shaftAxis) *decad.Body {
	shaft := b.dShaft(a, crankShaftFrom, crankShaftTo)
	shaft = b.cut(shaft, b.backFlatTool(a))
	return b.union(shaft, b.cylinder(a.across(crankCollar), collarRadius, 1))
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
