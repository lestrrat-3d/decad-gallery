package swisslever

// CI: isolated-tests

import (
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/stretchr/testify/require"
)

// TestFits checks every row of the Fits table of instructions.md on the built, posed movement. A press or seat fit is
// read as the section gap within the section's mesh bound. A class fit requires the gap's lower bound to reach the
// class minimum. The rows of the fork and guard (F1, F2, F4, G1 to G3), the escapement (E1, E2) and the pin root (G4)
// are measured on the real sections over the whole swing by TestForkAction, TestEscapementAction and TestPinRootGuard;
// rows T3, S3 and L1 that touch the hairspring's hub or the balance boss read the boss only when decad builds it.
func TestFits(t *testing.T) {
	_, m := sharedDefault(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	// T1: each arbor is exactly the 5.0 mm bore of its bearings.
	for _, e := range []struct {
		body *decad.Body
		at   Pt
		z    float64
		what string
	}{
		{m.EscapeArbor, g.O, -2.5, "escape arbor, back seat"}, {m.EscapeArbor, g.O, 29.5, "escape arbor, front seat"},
		{m.PalletArbor, g.P, -2.5, "pallet arbor, back seat"}, {m.PalletArbor, g.P, 29.5, "pallet arbor, front seat"},
		{m.BalanceStaff, g.B, -2.5, "balance staff, back seat"}, {m.BalanceStaff, g.B, 29.5, "balance staff, front seat"},
	} {
		lo, hi, off := seatRadii(t, e.body, e.z, e.at)
		require.InDelta(t, ArborRadius, hi, 0.01+off, e.what)
		require.InDelta(t, ArborRadius, lo, 0.01+off, e.what)
	}

	// T2: the shoulders and body steps stand 0.25 from the faces of the bearings they face.
	_, backTop := zRange(t, m.BackEscapeBearing)
	frontLow, _ := zRange(t, m.FrontEscapeBearing)
	require.InDelta(t, 0, backTop, 2e-3)
	require.InDelta(t, FrontPlateInner, frontLow, 2e-3)
	for _, arbor := range []*decad.Body{m.EscapeArbor, m.PalletArbor, m.BalanceStaff} {
		hasPlane(t, arbor, collarBackZ, "T2 step facing the back bearing")
		require.InDelta(t, 0.25, collarBackZ-backTop, 2e-3, "T2 rear step to the back bearing")
	}
	for _, arbor := range []*decad.Body{m.EscapeArbor, m.PalletArbor, m.BalanceStaff} {
		hasPlane(t, arbor, collarFrontZ, "T2 step facing the front bearing")
		require.InDelta(t, 0.25, frontLow-collarFrontZ, 2e-3, "T2 front step to the front bearing")
	}

	// T3, P1, P2: press fits take 0.05 mm all round.
	radial(t, m.EscapeWheel, m.EscapeArbor, 11.85, 0.05, 0, "T3 escape wheel on its arbor")
	radial(t, m.Lever, m.PalletArbor, 8, 0.05, 0, "T3 lever on its arbor")
	radial(t, m.SafetyRoller, m.BalanceStaff, 7, 0.05, 0, "T3 safety roller on the staff")
	radial(t, m.RollerTable, m.BalanceStaff, 12, 0.05, 0, "T3 roller table on the staff")
	radial(t, m.BalanceWheel, m.BalanceStaff, 16, 0.05, 0, "T3 balance wheel on the staff")
	radial(t, m.HairspringRetainer, m.BalanceStaff, 24, 0.05, 0, "T3 retainer on the staff")
	radial(t, m.EntryStone, m.Lever, 8, 0.05, 0, "P1 entry stone in its slot")
	radial(t, m.ExitStone, m.Lever, 8, 0.05, 0, "P1 exit stone in its slot")
	radial(t, m.KnobCollar, m.EscapeArbor, 35, 0.05, 0, "P2 knob collar on the arbor end")
	radial(t, m.WindingKnob, m.EscapeArbor, 36.8, 0.05, 0, "P2 knob on the arbor end (in its floor)")
	radial(t, m.WindingKnob, m.KnobCollar, 35, 0.05, 0, "P2 knob collar in the knob's counterbore")

	// T4p: the press stops. A bore corner that faces a cone stands 0.05 from it, measured normal to the cone; a face
	// that faces a face stands 0.05 from it.
	escapeCone := arborCollarCone(escapeArborPoints, 4)
	palletCone := arborCollarCone(palletArborPoints, 4)
	require.InDelta(t, 0.05, escapeCone.gapTo(bodyPressRadius, wheelHigh), 1e-3, "T4p escape wheel corner to its cone")
	hasPlane(t, m.EscapeWheel, wheelHigh, "T4p escape wheel front face")
	require.InDelta(t, 0.05, palletCone.gapTo(bodyPressRadius, leverLow), 1e-3, "T4p lever corner to its cone")
	hasPlane(t, m.Lever, leverLow, "T4p lever rear face")
	require.InDelta(t, 0.05, palletCone.gapTo(bodyPressRadius, safetyLow), 1e-3, "T4p safety roller corner to its cone")
	hasPlane(t, m.SafetyRoller, safetyLow, "T4p safety roller rear face")
	axial(t, m.RollerTable, tableLow, m.SafetyRoller, safetyLow+safetyLength, 0.05, "T4p roller table to the safety roller")
	axial(t, m.BalanceWheel, balanceLow, m.RollerTable, tableLow+tableThickness, 0.05, "T4p balance wheel to the roller table")
	axial(t, m.WindingKnob, keyedCounterLow+keyedCounterLong, m.KnobCollar, collarLow+collarLong, 0.05,
		"T4p knob floor to the Knob Collar")

	// S1: the hairspring hub's D bore on the staff, Tight; S2: the eye on the stud, Regular.
	// The spring's mesh bound is wide, so both are read from the spring's own vertices.
	spring, _ := sectionLoops(t, m.Hairspring, 21)
	staff, staffOff := sectionLoops(t, m.BalanceStaff, 21)
	stud, studOff := sectionLoops(t, m.FrontPlate, 21)
	require.InDelta(t, 0.25, springGap(spring, g.B, 0, 4, staff), 0.03+staffOff, "S1 hairspring hub on the staff")
	require.GreaterOrEqual(t, springGap(spring, g.Stud, 0, 4.5, stud), regularFit, "S2 hairspring eye on the stud")
	require.InDelta(t, 0.4, springGap(spring, g.Stud, 0, 4.5, stud), 0.03+studOff, "S2 hairspring eye on the stud")
	// S3: the hub to the balance boss and the retainer, Tight.
	springLo, springHi := zRange(t, m.Hairspring)
	require.InDelta(t, springLow, springLo, 3e-3)
	_, balanceTop := zRange(t, m.BalanceWheel)
	require.InDelta(t, balanceHigh+balanceBossHeight, balanceTop, 3e-3, "the boss tops the wheel")
	require.GreaterOrEqual(t, springLo-balanceTop, 0.25-3e-3, "S3 hub to the balance wheel's top")
	retainerLo, _ := zRange(t, m.HairspringRetainer)
	require.InDelta(t, 0.25, retainerLo-springHi, 3e-3, "S3 hub to the retainer")

	// L1: the hairspring's coils to the balance wheel's front face, and the stud's tip to the wheel, Loose.
	require.GreaterOrEqual(t, springLo-balanceHigh, 1.25-3e-3, "L1 coils to the balance wheel's front face")
	require.InDelta(t, 0.55, studTip-balanceHigh, 1e-9, "L1 stud tip to the balance wheel's front face")

	// F3: the pin's tip to the dart's top face, Loose.
	pinLo, _ := zRange(t, m.RollerTable)
	hasPlane(t, m.Lever, leverBandSplit, "F3 dart's top face")
	require.InDelta(t, 0.55, pinLo-leverBandSplit, 3e-3, "F3 pin tip to the dart's top face")

	// E4: the escape wheel's rear face to the lever plate; the stones' front ends past the wheel's front face.
	axial(t, m.EscapeWheel, wheelLow, m.Lever, leverHigh, 0.55, "E4 escape wheel to the lever plate")
	_, stoneTop := zRange(t, m.EntryStone)
	require.InDelta(t, 0.55, stoneTop-wheelHigh, 3e-3, "E4 stones past the wheel's front face")
	// E5: the wheel's heel circle to the pallet arbor body, Loose, 1.117.
	radial(t, m.EscapeWheel, m.PalletArbor, 11.85, 1.117, looseFit, "E5 heel circle to the pallet arbor")

	// R1: the lever's front face to the roller table; the fork corners to the pin's root round, Loose.
	axial(t, m.Lever, leverHigh, m.RollerTable, tableLow, 1.05, "R1 lever to the roller table")
	// R2: the balance rim to the pallet arbor body, Loose, 2.70.
	radial(t, m.BalanceWheel, m.PalletArbor, 15, 2.70, looseFit, "R2 balance rim to the pallet arbor")
	// R3: the lever to the back plate; the balance wheel to the lever.
	axial(t, m.Lever, leverLow, m.BackPlate, backPlateHigh, 3.30, "R3 lever to the back plate")
	axial(t, m.BalanceWheel, balanceLow, m.Lever, leverHigh, 3.60, "R3 balance wheel to the lever")

	// K1: the Knob Collar to the front plate's outer face, Tight; the knob to the front plate, Loose.
	axial(t, m.KnobCollar, collarLow, m.FrontPlate, frontPlateHigh, 0.25, "K1 Knob Collar to the front plate")
	axial(t, m.WindingKnob, knobLow, m.FrontPlate, frontPlateHigh, 1.30, "K1 knob to the front plate")

	// B1: each 605ZZ in its 0.05 mm seat.
	for _, e := range []struct {
		bearing, plate *decad.Body
		z              float64
		what           string
	}{
		{m.BackEscapeBearing, m.BackPlate, -2.5, "back escape"}, {m.BackPalletBearing, m.BackPlate, -2.5, "back pallet"},
		{m.BackBalanceBearing, m.BackPlate, -2.5, "back balance"},
		{m.FrontEscapeBearing, m.FrontPlate, 30.5, "front escape"}, {m.FrontPalletBearing, m.FrontPlate, 30.5, "front pallet"},
		{m.FrontBalanceBearing, m.FrontPlate, 30.5, "front balance"},
	} {
		radial(t, e.bearing, e.plate, e.z, 0.05, 0, "B1 "+e.what+" bearing seat")
	}
}

// TestFitsByMesh requires every pair of bodies that is neither a fit by design nor a working contact of the
// escapement to stand at least the smallest class clearance (Tight 0.2) apart at the default pose, proven by the
// meshes' lower bound.
func TestFitsByMesh(t *testing.T) {
	l := localParts(t)
	g := &l.g
	s, err := g.StateAt(DefaultParams().Phase)
	require.NoError(t, err)
	table, err := poses(g, s)
	require.NoError(t, err)
	slots := l.m.slots()
	meshes := make([]meshgap.Mesh, len(slots))
	for i, sl := range slots {
		meshes[i] = l.posedMesh(t, sl.name, table[sl.name])
	}
	for i := range slots {
		for j := i + 1; j < len(slots); j++ {
			a, b := slots[i].name, slots[j].name
			if byDesign(a, b) || byOutline(a, b) {
				continue
			}
			// A nut stands 0.2 in its cavity, the nominal part's fit, so a mesh proves it only apart from the plate.
			need := tightFit
			if strings.HasPrefix(a, "Nut ") || strings.HasPrefix(b, "Nut ") {
				need = 0
			}
			gap, ok, err := meshes[i].Exceeds(meshes[j], need)
			require.NoError(t, err)
			require.True(t, ok, "%s and %s stand %v apart, under %v", a, b, gap, need)
		}
	}
}
