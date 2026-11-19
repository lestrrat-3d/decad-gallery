package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSafetyRollerSketch requires the roller outline (roller arc, crescent arc, both lip rounds) to verify fully
// constrained with DOF 0 and to leave one valid region without holes.
func TestSafetyRollerSketch(t *testing.T) {
	b := partBuilder(t)
	s, err := b.safetySketch()
	require.NoError(t, err)
	_, err = b.profile(s, 0, "safety roller")
	require.NoError(t, err)
	requireSketchesConstrained(t, b, 1)
}

func TestRollerTable(t *testing.T) {
	b := partBuilder(t)
	table, err := b.buildRollerTable()
	require.NoError(t, err)
	require.True(t, table.IsSolid())
	requireSketchesConstrained(t, b, 2)
	// Disc z 10.85 to 13.35, pin tip at z 6.35.
	bb, err := table.Bounds()
	require.NoError(t, err)
	require.InDelta(t, tableLow-pinHeight, bb.Min.Z, 3e-3)
	require.InDelta(t, tableLow+tableThickness, bb.Max.Z, 3e-3)
	require.InDelta(t, tableRadius, math.Max(bb.Max.X, bb.Max.Y), 3e-3)
	pin := Polar(PinCentreRadius, b.g.PinAngle)
	require.True(t, partInside(t, table, pin.X, pin.Y, 8.0), "pin")
	far := Polar(PinCentreRadius+PinRadius+0.2, b.g.PinAngle)
	require.False(t, partInside(t, table, far.X, far.Y, 8.0), "beside the pin")
	require.True(t, partInside(t, table, -6, 0, 12.0), "disc")
	require.False(t, partInside(t, table, 2.4, 0, 12.0), "bore")
	require.True(t, partInside(t, table, 2.8, 0, 12.0), "bore flat")
}

func TestSafetyRoller(t *testing.T) {
	b := partBuilder(t)
	roller, err := b.buildSafetyRoller()
	require.NoError(t, err)
	require.True(t, roller.IsSolid())
	requireSketchesConstrained(t, b, 2)
	requireBoundsZ(t, roller, safetyLow, safetyLow+safetyLength, SafetyRadius)

	const z = 7.0
	c := Polar(CrescentOffset, b.g.PinAngle)
	require.False(t, partInside(t, roller, c.X, c.Y, z), "crescent centre is open")
	near := Polar(CrescentOffset-CrescentRadius+0.3, b.g.PinAngle)
	require.False(t, partInside(t, roller, near.X, near.Y, z), "crescent near side")
	// The wall between the crescent and the bore is 1.62 thick, along the pin's ray.
	wall := Polar(CrescentOffset-CrescentRadius-0.3, b.g.PinAngle)
	require.True(t, partInside(t, roller, wall.X, wall.Y, z), "wall in front of the crescent")
	opposite := Polar(5.5, b.g.PinAngle+math.Pi)
	require.True(t, partInside(t, roller, opposite.X, opposite.Y, z), "roller opposite the crescent")
	require.False(t, partInside(t, roller, 2.4, 0, z), "bore")
	require.True(t, partInside(t, roller, 2.8, 0, z), "bore flat")

	// The wall round the bore is 1.6 or more everywhere, from the section's loops: the crescent loop to the bore loop.
	loops, off := sectionLoops(t, roller, z)
	require.Len(t, loops, 2, "the roller's outline and the bore")
	var bore outline
	for _, l := range loops {
		if polyDist(l, outline{{0, 0}}) < 3.4 {
			bore = l
		}
	}
	require.NotNil(t, bore)
	gap := polyDist(loops[0], loops[1])
	require.GreaterOrEqual(t, gap, 1.6-0.02-off, "wall round the bore")
	require.InDelta(t, CrescentOffset-CrescentRadius-BodyBoreRadius, gap, 0.02+off)
}

func TestRollerCorners(t *testing.T) {
	b := partBuilder(t)
	roller, err := b.buildSafetyRoller()
	require.NoError(t, err)
	requireCorners(t, roller)
	table, err := b.buildRollerTable()
	require.NoError(t, err)
	requireCorners(t, table)
}

// TestPinRootGuard checks G4: the impulse pin's root round stands at least 0.274 from the safety roller. Both parts
// turn with the staff, so the gap is the same at every balance angle. The built gap is about 1.62, not the 0.274 that
// instructions.md lists, so the check is a lower bound.
func TestPinRootGuard(t *testing.T) {
	b := partBuilder(t)
	table, err := b.buildRollerTable()
	require.NoError(t, err)
	roller, err := b.buildSafetyRoller()
	require.NoError(t, err)
	// The root round is widest at the table's rear face; read the table just inside it and the roller at its top.
	root, off1 := sectionLoops(t, table, tableLow-0.01)
	top, off2 := sectionLoops(t, roller, safetyLow+safetyLength-0.01)
	require.NotEmpty(t, root)
	gap := loopsGap(root, top)
	require.GreaterOrEqual(t, gap, 0.274-0.03-off1-off2)
}
