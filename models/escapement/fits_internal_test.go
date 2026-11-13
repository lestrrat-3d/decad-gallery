package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// planeHeights returns the heights of the planar faces of body whose normal is along Z.
func planeHeights(body *decad.Body) []float64 {
	var out []float64
	for _, f := range body.Faces() {
		pl, ok := f.Surface().(decad.Plane)
		if ok && math.Abs(math.Abs(pl.Frame.N().Z)-1) < 1e-9 {
			out = append(out, pl.Frame.Origin().Z)
		}
	}
	return out
}

func hasPlane(t *testing.T, body *decad.Body, z float64, what string) {
	t.Helper()
	for _, h := range planeHeights(body) {
		if math.Abs(h-z) < 1e-6 {
			return
		}
	}
	require.Failf(t, "missing face", "%s has no planar face at z %v", what, z)
}

// axial checks a gap between a planar face of a at za and one of b at zb.
func axial(t *testing.T, a *decad.Body, za float64, b *decad.Body, zb, want float64, what string) {
	t.Helper()
	hasPlane(t, a, za, what)
	hasPlane(t, b, zb, what)
	require.InDelta(t, want, math.Abs(zb-za), 1e-9, what)
}

// bearingZ returns a bearing body's lower and upper cap heights.
func bearingZ(t *testing.T, body *decad.Body) (float64, float64) {
	t.Helper()
	b, err := body.Bounds()
	require.NoError(t, err)
	return b.Min.Z, b.Max.Z
}

// radial requires the section gap between a and b at z: at least min (a class minimum), or, for a press or seat fit,
// want within the measurement bound when min is zero.
func radial(t *testing.T, a, b *decad.Body, z, want, min float64, what string) {
	t.Helper()
	oa, ob := sectionAt(t, a, z), sectionAt(t, b, z)
	c, err := oa.Clearance(ob)
	require.NoError(t, err, what)
	require.False(t, c.Overlaps, what)
	bound := c.Distance.Bound.Base()
	if min > 0 {
		require.GreaterOrEqual(t, c.Lower().Base(), min, "%s: gap %v ± %v", what, c.Distance.Value.Base(), bound)
		return
	}
	require.InDelta(t, want, c.Distance.Value.Base(), bound+0.01, "%s: gap ± %v", what, bound)
}

func TestFits(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	const regular = 0.35

	// T1: each arbor and the suspension pin are exactly the 5.0 mm bore of their bearings.
	for _, e := range []struct {
		body   *decad.Body
		at     Pt
		z      float64
		bounds string
	}{
		{m.EscapeArbor, g.O, -2.5, "escape arbor, back seat"}, {m.EscapeArbor, g.O, 30.5, "escape arbor, front seat"},
		{m.PalletArbor, g.P, -2.5, "pallet arbor, back seat"}, {m.PalletArbor, g.P, 30.5, "pallet arbor, front seat"},
		{m.GreatWheelArbor, g.G, -2.5, "great wheel arbor, back seat"},
		{m.GreatWheelArbor, g.G, 30.5, "great wheel arbor, front seat"},
		{m.SuspensionPin, g.S, 41.5, "suspension pin, pendulum bearing"},
	} {
		o := sectionAt(t, e.body, e.z)
		lo, hi := math.Inf(1), 0.0
		for _, p := range loopsBy(o)[0] {
			r := p.Sub(e.at).Len()
			lo, hi = math.Min(lo, r), math.Max(hi, r)
		}
		require.InDelta(t, 2.5, hi, 0.01+o.OffSurface.Base(), e.bounds)
		require.InDelta(t, 2.5, lo, 0.01+o.OffSurface.Base(), e.bounds)
	}

	// T2: collars, bosses, the shoulder and the cap stand 0.25 from the bearing faces.
	frontZ, _ := bearingZ(t, m.FrontEscapeBearing)
	_, backTop := bearingZ(t, m.BackEscapeBearing)
	require.InDelta(t, 28, frontZ, 2e-3)
	require.InDelta(t, 0, backTop, 2e-3)
	axial(t, m.EscapeArbor, 27.75, m.FrontEscapeBearing, frontZ, 0.25, "T2 escape collar to the front bearing")
	hasPlane(t, m.EscapeArbor, 0.25, "T2 escape arbor's body step")
	hasPlane(t, m.EscapeRearCollar, 0.25, "T2 escape rear collar")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "T2 escape rear collar to the back bearing")
	hasPlane(t, m.PalletArbor, 0.25, "T2 pallet shoulder")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3, "T2 pallet shoulder to the back pallet bearing")
	hasPlane(t, m.PalletArbor, 27.75, "T2 pallet arbor's body step")
	hasPlane(t, m.EscapeArbor, 27.75, "T2 escape arbor's front collar")
	hasPlane(t, m.GreatWheelArbor, 27.75, "T2 great wheel arbor's body step")
	hasPlane(t, m.GreatWheelArbor, 0.25, "T2 great wheel collar")
	require.InDelta(t, 0.25, 0.25-backTop, 2e-3)
	hasPlane(t, m.SpringSpacer, 27.75, "T2 spring spacer")
	require.InDelta(t, 0.25, frontZ-27.75, 2e-3, "S5 spacer to the front great wheel bearing")
	hasPlane(t, m.PalletFrontCollar, 27.75, "T2 pallet front collar")
	require.InDelta(t, 0.25, frontZ-27.75, 2e-3, "T2 pallet front collar to the front pallet bearing")
	pendLow, pendHigh := bearingZ(t, m.PendulumBearing)
	axial(t, m.SuspensionPin, 38.75, m.Pendulum, 38.75, 0, "T2 shoulder")
	require.InDelta(t, 0.25, pendLow-38.75, 2e-3, "T2 shoulder to the pendulum bearing")
	hasPlane(t, m.SuspensionCap, 44.25, "T2 cap")
	require.InDelta(t, 0.25, 44.25-pendHigh, 2e-3, "T2 cap to the pendulum bearing")

	// T3, P1: press fits take 0.05 mm all round.
	radial(t, m.EscapeWheel, m.EscapeArbor, 10, 0.05, 0, "T3 escape wheel on its arbor")
	radial(t, m.Anchor, m.PalletArbor, 24, 0.05, 0, "T3 anchor on its arbor")
	radial(t, m.EscapeRearCollar, m.EscapeArbor, 1.75, 0.05, 0, "T3 escape rear collar on its arbor")
	radial(t, m.PalletFrontCollar, m.PalletArbor, 26.25, 0.05, 0, "T3 pallet front collar on its arbor")
	radial(t, m.PawlCarrier, m.GreatWheelArbor, 7, 0.05, 0, "T3 pawl carrier on its arbor")
	radial(t, m.SpringSpacer, m.GreatWheelArbor, 22, 0.05, 0, "T3 spring spacer on its arbor")
	radial(t, m.Crutch, m.PalletArbor, 36, 0.05, 0, "P1 crutch on its arbor")
	radial(t, m.WindingKnob, m.GreatWheelArbor, 36, 0.05, 0, "P1 knob on its arbor")
	radial(t, m.SuspensionCap, m.SuspensionPin, 45, 0.05, 0, "P1 cap on the pin")
	radial(t, m.FrontPlate, m.SuspensionPin, 32, 0.05, 0, "P1 pin in the front plate")

	// T4, T4p: axial gaps along an arbor.
	axial(t, m.GreatWheel, 3.5, m.GreatWheelArbor, 3.25, 0.25, "T4 great wheel counterbore floor to its collar")
	axial(t, m.PawlCarrier, 5.25, m.GreatWheel, 5.0, 0.25, "T4 pawl layer to the ratchet floor")
	axial(t, m.EscapeWheel, 23.95, m.EscapeArbor, 24.0, 0.05, "T4p escape wheel counterbore floor to its collar")
	axial(t, m.EscapeRearCollar, 3.25, m.EscapeWheel, 3.30, 0.05, "T4p escape rear collar to the pinion counterbore floor")
	axial(t, m.PalletFrontCollar, 24.75, m.Anchor, 24.70, 0.05, "T4p pallet front collar to the anchor counterbore floor")
	axial(t, m.SpringSpacer, 17.5, m.GreatWheelArbor, 17.45, 0.05, "T4p spring spacer to the arbor step")
	axial(t, m.Anchor, 21, m.PalletArbor, 20.95, 0.05, "T4p anchor to the pallet collar")

	// T5: the crutch and knob stand 0.25 off the front plate.
	axial(t, m.Crutch, 34.25, m.FrontPlate, 34, 0.25, "T5 crutch")
	axial(t, m.WindingKnob, 34.25, m.FrontPlate, 34, 0.25, "T5 knob")

	// C1: each collar ring in the counterbore it enters, press 0.05 all round.
	radial(t, m.EscapeWheel, m.EscapeArbor, 24.2, 0.05, 0, "C1 escape front collar in the wheel's counterbore")
	radial(t, m.EscapeRearCollar, m.EscapeWheel, 2.4, 0.05, 0, "C1 escape rear collar in the pinion's counterbore")
	radial(t, m.PalletFrontCollar, m.Anchor, 25.2, 0.05, 0, "C1 pallet front collar in the anchor's counterbore")

	// C1: the great wheel turns on its collar in the counterbore, Regular 0.40 radially.
	radial(t, m.GreatWheel, m.GreatWheelArbor, 2.5, 0.4, regular, "C1 great wheel counterbore on its collar")

	// T6: the crutch pin in the pendulum's slot, 0.25 each side (checked in TestCoupling by pose); here the build pose.
	radial(t, m.Crutch, m.Pendulum, 40, 0.25, 0, "T6 crutch pin in the slot")

	// R1: the great wheel's 6.9 bore on the 6.1 arbor section.
	radial(t, m.GreatWheel, m.GreatWheelArbor, 4, 0.4, regular, "R1 great wheel bore")

	// L1, L3, L4: axial Loose gaps.
	axial(t, m.Pendulum, 37.75, m.Crutch, 37.25, 0.5, "L3 pendulum to the crutch")
	axial(t, m.Pendulum, 37.75, m.WindingKnob, 37.25, 0.5, "L3 pendulum to the knob")
	axial(t, m.Anchor, 26, m.FrontPlate, 28, 2.0, "L4 anchor to the front plate")
	axial(t, m.EscapeWheel, 25, m.FrontPlate, 28, 3.0, "L4 escape wheel to the front plate")

	// L2, L5, L6: radial Loose gaps.
	radial(t, m.SpringSpacer, m.EscapeWheel, 23.5, 2.05, 2.05-0.1, "L2 spring spacer to the wheel tips")
	radial(t, m.GreatWheelArbor, m.EscapeWheel, 23.5, 2.95, 2.95-0.1, "L2 great wheel arbor to the wheel tips")
	radial(t, m.EscapeWheel, m.PawlCarrier, 7, 22.0, 22.0-0.1, "L5 pinion tips to the pawl layer")
	radial(t, m.SuspensionPin, m.Pendulum, 38.2, 1.3, 1.3-0.1, "L6 shoulder in the lip bore")

	// B1: each 605ZZ in its 0.05 mm seat.
	for _, e := range []struct {
		bearing, plate *decad.Body
		z              float64
		what           string
	}{
		{m.BackEscapeBearing, m.BackPlate, -2.5, "back escape"}, {m.BackPalletBearing, m.BackPlate, -2.5, "back pallet"},
		{m.BackGreatWheelBearing, m.BackPlate, -2.5, "back great wheel"},
		{m.FrontEscapeBearing, m.FrontPlate, 30.5, "front escape"}, {m.FrontPalletBearing, m.FrontPlate, 30.5, "front pallet"},
		{m.FrontGreatWheelBearing, m.FrontPlate, 30.5, "front great wheel"},
		{m.PendulumBearing, m.Pendulum, 41.5, "pendulum"},
	} {
		radial(t, e.bearing, e.plate, e.z, 0.05, 0, "B1 "+e.what+" bearing seat")
	}
}
