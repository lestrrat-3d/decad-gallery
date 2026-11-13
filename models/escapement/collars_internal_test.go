package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// minCollar is the shortest collar: instructions.md "Arbor collars", docs/model-pipeline.md Arbor collars.
const minCollar = 3.0

// sectionRadii returns the least and greatest distance from axis to the points of the section of body at z, over the
// loop that reaches farthest from the axis, and the section's OffSurface bound.
func sectionRadii(t *testing.T, body *decad.Body, z float64, axis Pt) (float64, float64, float64) {
	t.Helper()
	o := sectionAt(t, body, z)
	loop := outerLoop2(o.Loops, axis)
	lo, hi := math.Inf(1), 0.0
	for _, p := range loop {
		lo, hi = math.Min(lo, p.Len()), math.Max(hi, p.Len())
	}
	return lo, hi, o.OffSurface.Base()
}

// requireShaftCollar requires a collar of the shaft's own body between z0 and z1: both end faces on the arbor, a
// length of at least 3.0 mm and a round section of radius 3.7 at mid-length.
func requireShaftCollar(t *testing.T, arbor *decad.Body, axis Pt, z0, z1 float64, what string) {
	t.Helper()
	hasPlane(t, arbor, z0, what+": rear end face")
	hasPlane(t, arbor, z1, what+": front end face")
	require.GreaterOrEqual(t, z1-z0, minCollar-1e-9, "%s: length", what)
	lo, hi, off := sectionRadii(t, arbor, (z0+z1)/2, axis)
	require.InDelta(t, CollarRadius, lo, 0.01+off, "%s: radius", what)
	require.InDelta(t, CollarRadius, hi, 0.01+off, "%s: radius", what)
}

// requireRing requires a collar ring: a z extent of at least 3.0 mm and an outer radius of 3.7 about axis.
func requireRing(t *testing.T, ring *decad.Body, axis Pt, z0 float64, what string) {
	t.Helper()
	bb, err := ring.Bounds()
	require.NoError(t, err)
	require.InDelta(t, z0, bb.Min.Z, 2e-3, "%s: rear face", what)
	require.GreaterOrEqual(t, bb.Max.Z-bb.Min.Z, minCollar-2e-3, "%s: length", what)
	require.InDelta(t, axis.X+CollarRadius, bb.Max.X, 2e-3, "%s: outer radius", what)
	require.InDelta(t, axis.X-CollarRadius, bb.Min.X, 2e-3, "%s: outer radius", what)
	require.InDelta(t, axis.Y+CollarRadius, bb.Max.Y, 2e-3, "%s: outer radius", what)
}

// TestArborCollars checks each collar of the movement on the built bodies: its length, its diameter, the counterbore
// it enters and the gaps between the carried part and the flush side.
func TestArborCollars(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	// Collars of the shafts' own bodies.
	requireShaftCollar(t, m.EscapeArbor, g.O, 24.0, 27.75, "escape front collar")
	requireShaftCollar(t, m.PalletArbor, g.P, 0.25, 20.95, "pallet shoulder")
	requireShaftCollar(t, m.GreatWheelArbor, g.G, 0.25, 3.25, "great wheel collar")

	// The keyed rings in their own frames (the built movement turns them): the arc reaches 3.7 on the -X side and the
	// chord stands at 3.3 on the +X side.
	b := testBuilder(t)
	for name, build := range map[string]func() (*decad.Body, error){"crutch collar": b.buildCrutchCollar,
		"knob collar": b.buildKnobCollar} {
		ring, err := build()
		require.NoError(t, err)
		bb, err := ring.Bounds()
		require.NoError(t, err)
		axisX := 0.0
		require.InDelta(t, axisX+keyedRingFlat, bb.Max.X, 2e-3, "%s: outer chord", name)
		require.InDelta(t, axisX-CollarRadius, bb.Min.X, 2e-3, "%s: outer arc", name)
	}

	// Collar rings.
	requireRing(t, m.EscapeRearCollar, g.O, 0.25, "escape rear collar")
	requireRing(t, m.PalletFrontCollar, g.P, 24.75, "pallet front collar")
	for _, e := range []struct {
		ring *decad.Body
		axis Pt
		name string
	}{{m.CrutchCollar, g.P, "crutch collar"}, {m.KnobCollar, g.G, "knob collar"}} {
		bb, err := e.ring.Bounds()
		require.NoError(t, err)
		require.InDelta(t, 34.25, bb.Min.Z, 2e-3, "%s: rear face", e.name)
		require.GreaterOrEqual(t, bb.Max.Z-bb.Min.Z, minCollar-2e-3, "%s: length", e.name)
		// A keyed D ring: the arc reaches radius 3.7 on the -X side and the chord stands at 3.3 on the +X side.
		_, hi, off := sectionRadii(t, e.ring, 36, e.axis)
		require.InDelta(t, 2.05, boreReach(t, e.ring, 36, e.axis), 0.02, "%s: bore chord", e.name)
		require.InDelta(t, CollarRadius, hi, 0.01+off, "%s: outer radius", e.name)
		// Wall at the flats: 3.3 - 2.05 = 1.25.
		require.GreaterOrEqual(t, keyedRingFlat-pressFlat, 1.2, "%s: wall at the flats", e.name)
	}

	// Counterbores: the part's floor stops 0.05 from the collar and the wall stands 0.05 from it all round (press fits,
	// the parts turn with the shaft). The collar enters each counterbore by at least 1 mm of its length, and the part
	// keeps at least 1 mm of material under it.
	axial(t, m.EscapeWheel, 23.95, m.EscapeArbor, 24.0, 0.05, "escape wheel front counterbore floor")
	axial(t, m.EscapeRearCollar, 3.25, m.EscapeWheel, 3.30, 0.05, "pinion rear counterbore floor")
	axial(t, m.PalletFrontCollar, 24.75, m.Anchor, 24.70, 0.05, "anchor front counterbore floor")
	axial(t, m.GreatWheel, 3.5, m.GreatWheelArbor, 3.25, 0.25, "great wheel counterbore floor")
	radial(t, m.GreatWheel, m.GreatWheelArbor, 2.5, 0.4, 0.35, "great wheel counterbore wall")
	require.GreaterOrEqual(t, 3.25-1.5, 1.0, "great wheel collar depth in the wheel")
	require.GreaterOrEqual(t, ratchetFloor-greatCounterFloor, 1.0, "wheel material under its counterbore floor")
	radial(t, m.EscapeWheel, m.EscapeArbor, 24.2, 0.05, 0, "escape wheel front counterbore wall")
	radial(t, m.EscapeRearCollar, m.EscapeWheel, 2.4, 0.05, 0, "pinion rear counterbore wall")
	radial(t, m.PalletFrontCollar, m.Anchor, 25.2, 0.05, 0, "anchor front counterbore wall")
	require.GreaterOrEqual(t, 25.0-24.0, 1.0-1e-9, "escape front collar depth in the wheel")
	require.GreaterOrEqual(t, 3.25-1.5, 1.0, "escape rear collar depth in the pinion")
	require.GreaterOrEqual(t, 26.0-24.75, 1.0, "pallet front collar depth in the anchor")
	require.GreaterOrEqual(t, 23.95-22.0, 1.0, "wheel material under its counterbore floor")
	require.GreaterOrEqual(t, 3.30-1.5, 1.0, "pinion material under its counterbore floor")
	require.GreaterOrEqual(t, 24.70-21.0, 1.0, "anchor material under its counterbore floor")

	// Collar to flush side: each ring and shoulder stands 0.25 from the bearing it faces.
	_, backTop := bearingZ(t, m.BackEscapeBearing)
	frontZ, _ := bearingZ(t, m.FrontPalletBearing)
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "escape rear collar to the back bearing")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "pallet shoulder to the back bearing")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "great wheel collar to the back bearing")
	require.InDelta(t, 0.25, frontZ-27.75, 2e-3, "pallet front collar to the front bearing")

	// Carried part to flush side: the pinion's rear face is 1.5 from the back plate's inner face, and the anchor's front
	// face is 2.0 from the front plate's inner face.
	axial(t, m.EscapeWheel, 1.5, m.BackPlate, 0, 1.5, "pinion rear face to the back plate")
	axial(t, m.GreatWheel, 1.5, m.BackPlate, 0, 1.5, "great wheel rear face to the back plate")
	axial(t, m.Crutch, 35.30, m.FrontPlate, 34, 1.30, "crutch rear face to the front plate")
	axial(t, m.WindingKnob, 35.30, m.FrontPlate, 34, 1.30, "knob rear face to the front plate")
	// Outside collars start 0.25 in front of the plate's outer face, and stay outside its lip bore.
	hasPlane(t, m.CrutchCollar, 34.25, "crutch collar rear face")
	hasPlane(t, m.KnobCollar, 34.25, "knob collar rear face")
	hasPlane(t, m.PalletArbor, 34.25, "the pallet arbor flat starts where the crutch collar does")
	axial(t, m.Anchor, 26, m.FrontPlate, 28, 2.0, "anchor front face to the front plate")
}
