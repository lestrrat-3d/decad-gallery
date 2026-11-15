package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// minPressWall is the thinnest wall round a press-fit bore: docs/model-pipeline.md Corners and roots.
const minPressWall = 1.6

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

// arborCone is a cone of an arbor's meridian: its two end points (radius, z).
type arborCone struct{ a, b [2]float64 }

// coneBetween returns the cone of pts that joins the points at index i and i+1.
func coneBetween(pts meridianPoints, i int) arborCone { return arborCone{pts[i], pts[i+1]} }

// gapTo returns the normal distance from the meridian point (r, z) to the cone, measured to the cone's segment.
func (c arborCone) gapTo(r, z float64) float64 {
	return distPointSeg(Pt{r, z}, Pt{c.a[0], c.a[1]}, Pt{c.b[0], c.b[1]})
}

// coneFaces returns the cone faces of body with a half angle of 30 degrees.
func coneFaces(t *testing.T, body *decad.Body) []decad.Cone {
	t.Helper()
	var out []decad.Cone
	for _, f := range body.Faces() {
		c, ok := f.Surface().(decad.Cone)
		if !ok {
			continue
		}
		// A chamfer is a cone too, at 45 degrees; only the 30 degree cones are the arbor's and its mates'.
		if math.Abs(c.HalfAngle.Base()-math.Pi/6) < 1e-6 {
			out = append(out, c)
		}
	}
	return out
}

// coneZ returns the z at radius r on the cone surface c.
func coneZ(c decad.Cone, r float64) float64 {
	return c.Origin.Z + c.Axis.Z*(r-c.Radius.Base())/math.Tan(c.HalfAngle.Base())
}

// requireCone requires a cone face on body, of 30 degrees half angle, that passes through both end points of want.
func requireCone(t *testing.T, body *decad.Body, want arborCone, what string) {
	t.Helper()
	for _, c := range coneFaces(t, body) {
		if math.Abs(coneZ(c, want.a[0])-want.a[1]) < 1e-6 && math.Abs(coneZ(c, want.b[0])-want.b[1]) < 1e-6 {
			return
		}
	}
	require.Failf(t, "missing cone", "%s has no 30 degree cone from (r %.4f, z %.4f) to (r %.4f, z %.4f)", what,
		want.a[0], want.a[1], want.b[0], want.b[1])
}

// coneFloorGap returns the least normal distance from the cone face of body that passes the cone c's line, sampled
// over radii lo to hi, to the segment of c.
func coneFloorGap(t *testing.T, body *decad.Body, c arborCone, lo, hi float64) float64 {
	t.Helper()
	least := math.Inf(1)
	for _, f := range coneFaces(t, body) {
		for i := 0; i <= 100; i++ {
			r := lo + (hi-lo)*float64(i)/100
			least = math.Min(least, c.gapTo(r, coneZ(f, r)))
		}
	}
	require.False(t, math.IsInf(least, 1), "the body has a cone face")
	return least
}

// requireConeCollar requires a collar of the shaft's own body from its end face at far to where its cone meets the
// smaller diameter at near: the end face and the cone are on the arbor, the whole length is at least 3.0 mm, and a
// round section of radius 3.7 at z round.
func requireConeCollar(t *testing.T, arbor *decad.Body, axis Pt, far, near, round float64, cone arborCone, what string) {
	t.Helper()
	hasPlane(t, arbor, far, what+": end face")
	requireCone(t, arbor, cone, what)
	require.GreaterOrEqual(t, math.Abs(far-near), minCollar-1e-9, "%s: whole length to the cone's end", what)
	lo, hi, off := sectionRadii(t, arbor, round, axis)
	require.InDelta(t, CollarRadius, lo, 0.01+off, "%s: radius", what)
	require.InDelta(t, CollarRadius, hi, 0.01+off, "%s: radius", what)
}

// boreChamferReach is the radius where a 605ZZ inner ring's bore chamfer ends on its face. The repository holds no
// chamfer figure, so this is the design's worst case: a chamfer that reaches radius 3.0 (0.5 mm on the 2.5 mm bore).
const boreChamferReach = 3.0

// minThrustLand is the narrowest ring of an arbor step that must reach the inner ring's flat face beyond the chamfer.
const minThrustLand = 0.3

// requireBodyStep requires a body step of the arbor at z: a planar face there, a round section of radius 3.3 beside it, a
// land of at least minThrustLand on the inner ring's face beyond the bore chamfer, and the carried part's face at least
// 1.0 mm from the flush side.
func requireBodyStep(t *testing.T, arbor *decad.Body, axis Pt, z, mid, partFace float64, what string) {
	t.Helper()
	hasPlane(t, arbor, z, what+": step face")
	lo, hi, off := sectionRadii(t, arbor, mid, axis)
	require.InDelta(t, bodyRadius, hi, 0.01+off, "%s: round radius", what)
	require.Greater(t, lo, ArborRadius, "%s: the flat stays clear of the seat", what)
	require.GreaterOrEqual(t, bodyRadius-boreChamferReach, minThrustLand-1e-9, "%s: land beyond the bore chamfer", what)
	require.GreaterOrEqual(t, math.Abs(partFace-z), 1.0, "%s: carried part's face to the flush side", what)
}

// TestArborCollars checks each collar of the movement on the built bodies: its length, its diameter, the counterbore
// it enters, the body steps and the gaps between the carried part and the flush side.
func TestArborCollars(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	// Collars of the shafts' own bodies.
	// Each collar's whole length runs to where its cone meets the smaller diameter.
	escapeCone, palletCone := coneBetween(escapeArborPoints, 4), coneBetween(palletArborPoints, 4)
	greatCone := coneBetween(greatArborPoints, 4)
	requireConeCollar(t, m.EscapeArbor, g.O, 27.75, escapeConeLow, 27, escapeCone, "escape front collar")
	requireConeCollar(t, m.PalletArbor, g.P, 0.25, palletConeHigh, 10, palletCone, "pallet shoulder")
	requireConeCollar(t, m.GreatWheelArbor, g.G, 0.25, greatCollarEnd, 1.2, greatCone, "great wheel collar")
	requireCone(t, m.GreatWheelArbor, coneBetween(greatArborPoints, 6), "great wheel journal step")
	requireCone(t, m.PalletArbor, coneBetween(palletArborPoints, 8), "pallet end step")
	requireCone(t, m.GreatWheelArbor, coneBetween(greatArborPoints, 10), "great wheel end step")
	requireConeCollar(t, m.SuspensionPin, g.S, 39.80, frontHigh, 38, coneBetween(pinPoints, 2), "suspension shoulder")
	requireCone(t, m.SuspensionPin, coneBetween(pinPoints, 6), "suspension pin end step")

	// Body steps of the shafts: the end of the body section the parts are pressed on, facing the bearing with the Tight
	// gap. The escape wheel's pinion rear face is at 1.5, the anchor's front face at 26.
	requireBodyStep(t, m.EscapeArbor, g.O, 0.25, 12, 1.5, "escape body step")
	requireBodyStep(t, m.PalletArbor, g.P, 27.75, 24, 26, "pallet body step")
	requireBodyStep(t, m.GreatWheelArbor, g.G, 27.75, 24, 20.5, "great wheel body step")

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
		require.InDelta(t, endPressFlat, boreReach(t, e.ring, 36, e.axis), 0.02, "%s: bore chord", e.name)
		require.InDelta(t, CollarRadius, hi, 0.01+off, "%s: outer radius", e.name)
		// Walls: 3.7 - 2.05 = 1.65 round and 3.3 - 1.55 = 1.75 at the flats, over the 1.6 minimum.
		require.GreaterOrEqual(t, CollarRadius-endPressRadius, minPressWall-1e-9, "%s: wall round the bore", e.name)
		require.GreaterOrEqual(t, keyedRingFlat-endPressFlat, minPressWall, "%s: wall at the flats", e.name)
	}

	// Counterbores: the part's floor stands 0.05 from the collar and the wall stands 0.05 from it all round (press fits,
	// the parts turn with the shaft). The collar enters each counterbore by at least 1 mm of its length, and the part
	// keeps at least 1 mm of material under it. A cone has no stop face: the escape wheel's floor corner and the anchor's
	// rear bore corner stand 0.05 from the cone, measured normal to it, and the great wheel's floor corner at its bore
	// stands at least 0.25 from the collar's cone.
	hasPlane(t, m.EscapeWheel, wheelCounterFloor, "escape wheel front counterbore floor")
	require.InDelta(t, 0.05, escapeCone.gapTo(bodyPressRadius, wheelCounterFloor), 1e-3, "escape wheel floor corner to the cone")
	hasPlane(t, m.Anchor, anchorLow, "anchor rear face")
	require.InDelta(t, 0.05, palletCone.gapTo(bodyPressRadius, anchorLow), 1e-3, "anchor bore corner to the cone")
	hasPlane(t, m.GreatWheel, greatCounterFloor, "great wheel counterbore floor")
	require.GreaterOrEqual(t, greatCone.gapTo(greatBore/2, greatCounterFloor), 0.25, "great wheel floor corner to the cone")
	radial(t, m.GreatWheel, m.GreatWheelArbor, 1.7, 0.4, 0.35, "great wheel counterbore wall")
	require.GreaterOrEqual(t, greatCollarEnd-1.5, 1.0, "great wheel collar depth in the wheel")
	require.GreaterOrEqual(t, ratchetFloor-greatCounterFloor, 1.0, "wheel material under its counterbore floor")
	require.GreaterOrEqual(t, escapeCone.gapTo(wheelCounterRadius, 24.5), 0.05, "escape wheel counterbore wall to the cone")
	radial(t, m.EscapeWheel, m.EscapeArbor, 24.8, 0, 0.05, "escape wheel front counterbore wall")
	require.GreaterOrEqual(t, 25.0-escapeConeLow, 1.0-1e-9, "escape front collar depth in the wheel")
	require.GreaterOrEqual(t, wheelCounterFloor-22.0, 1.0, "wheel material under its counterbore floor")

	// Collar to flush side: each collar and step stands 0.25 from the bearing it faces.
	_, backTop := bearingZ(t, m.BackEscapeBearing)
	frontZ, _ := bearingZ(t, m.FrontPalletBearing)
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "escape body step to the back bearing")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "pallet shoulder to the back bearing")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "great wheel collar to the back bearing")
	require.InDelta(t, 0.25, frontZ-27.75, 2e-3, "pallet body step to the front bearing")
	require.InDelta(t, 0.25, frontZ-27.75, 2e-3, "great wheel body step to the front bearing")

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
