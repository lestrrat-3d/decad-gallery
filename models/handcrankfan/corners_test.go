package handcrankfan_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBevelToothRootCorners records that the driving gear's tooth roots have no fillet. Each of its teeth leaves one
// sharp inside corner of about 83° on its root cone, in every section through the teeth. The gear's flank is an
// involute that starts below the root, and a fillet tangent to it needs a sketch constraint and a decad join that do
// not exist yet. "Corners and roots" does not exempt the corners, so they are known findings, and the model stays on
// CORNER_PENDING until they are rounded. The pinion's roots are filleted and read by requirePinionRoots. The test
// fails when the driving gear's roots are rounded or the geometry moves.
func TestBevelToothRootCorners(t *testing.T) {
	_, parts, a := sharedDefault(t)
	t.Run("driving-gear", func(t *testing.T) {
		// 56 teeth, x 14.14 to 16.14, root cone r 23.41 at x 14.14.
		toothRoots(t, parts["driving-gear"].Body, r3.Vec{X: 1}, a.State.Apex, 56,
			[]float64{14.14, 15.14, 16.14}, func(float64) int { return 6 }, 23.41, 14.14, 1.655, 82, 84)
	})
}

// TestFanHubIsTurbinePreset requires the fan's rotor to be turbinefan's 608ZZ preset apart from the pitch, the hand and
// the placement and the rotor's size (128 mm, not the preset's default), so TestTurbineFanCorners' reading of the 608ZZ
// hub holds for the hub built here.
func TestFanHubIsTurbinePreset(t *testing.T) {
	got := handcrankfan.FanParams(handcrankfan.DefaultParams())
	require.NotNil(t, got.Placement)
	require.Equal(t, turbinefan.RightHand, got.Hand)
	require.Equal(t, 128.0, got.RotorDiameter, "the rotor keeps turbinefan's smallest size")
	got.Placement, got.BladePitch, got.Hand = nil, turbinefan.Params608ZZ().BladePitch, turbinefan.RightHand
	got.RotorDiameter = turbinefan.Params608ZZ().RotorDiameter
	require.Equal(t, turbinefan.Params608ZZ(), got)
}

func TestHandCrankFanCorners(t *testing.T) {
	_, parts, _ := sharedDefault(t)
	requireCorners(t, handcrankfan.DefaultParams(), parts)
}

// TestPressFitWalls requires at least 1.6 mm of wall round the bore of every separate printed part pressed on a shaft
// (docs/model-pipeline.md "Corners and roots", Root width): the hub through its socket floors, the arm and the two
// spur gears. The bevel pair is not measured here.
func TestPressFitWalls(t *testing.T) {
	_, parts, a := sharedDefault(t)
	for _, c := range []struct {
		name string
		axis r3.Vec
		at   float64 // along the print axis
		x    bool    // the print axis is X, otherwise Y
	}{
		{"fan-hub", a.State.Apex, -86, false},
		{"crank-arm", a.State.Crank, 52, true},
		{"crank-gear", a.State.Crank, -6.2, true},
		{"drive-gear", a.State.Apex, -6.2, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			plane, centre := atY(t, c.at), r3.Vec{X: c.axis.X, Y: c.at, Z: c.axis.Z}
			if c.x {
				plane, centre = atX(t, c.at), r3.Vec{X: c.at, Y: c.axis.Y, Z: c.axis.Z}
			}
			section, err := parts[c.name].Body.SectionOutline(t.Context(), plane, units.Millimeters(fitTolerance))
			require.NoError(t, err)
			wall := boreWall(t, section.Loops, centre)
			require.GreaterOrEqual(t, wall, minPressWall-0.01, "%s: wall round the bore is %.3f", c.name, wall)
		})
	}
}
