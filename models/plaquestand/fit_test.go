package plaquestand_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// gapMesh builds a meshgap mesh of body's verified tessellation at tol.
func gapMesh(t *testing.T, body *decad.Body, tol float64) meshgap.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	g, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return g
}

// TestPlaqueStandClearance proves the Loose fit of the plaque in the slots: the proven lower bound of the gap between
// the plaque and every other body meets the Loose class minimum in every case.
func TestPlaqueStandClearance(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			plaque := gapMesh(t, m.Plaque, 0.01)
			others := map[string]*decad.Body{
				"bracket 1": m.Brackets[0], "bracket 2": m.Brackets[1], "rail": m.Rail,
				"bolt 1": m.Bolts[0].Body, "bolt 2": m.Bolts[1].Body, "nut 1": m.Nuts[0].Body, "nut 2": m.Nuts[1].Body,
			}
			for name, body := range others {
				ok, err := plaque.ExceedsClearance(gapMesh(t, body, 0.01), plaquestand.LooseFit)
				require.NoError(t, err)
				require.True(t, ok, "the plaque clears the %s by more than %g mm", name, plaquestand.LooseFit)
			}
		})
	}
}

// TestPlaqueStandRailFit proves the Regular fit of the rail in the channels, and that the rail sits in them: the rail
// clears each bracket by the Regular minimum, and does not clear it by more than the joint gap plus 0.05 mm.
func TestPlaqueStandRailFit(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			rail := gapMesh(t, m.Rail, 0.01)
			for k, b := range m.Brackets {
				bracket := gapMesh(t, b, 0.01)
				ok, err := rail.ExceedsClearance(bracket, plaquestand.RegularFit)
				require.NoError(t, err)
				require.True(t, ok, "the rail clears bracket %d by more than %g mm", k+1, plaquestand.RegularFit)
				loose, err := rail.ExceedsClearance(bracket, plaquestand.JointFit+0.05)
				require.NoError(t, err)
				require.False(t, loose, "the rail sits in bracket %d's channel", k+1)
			}
		})
	}
}

// TestPlaqueStandStability reads the tip angles off the built bodies. The plaque's centre of gravity, leaned inside the
// slot clearance by the play angle, must stay MinTipAngle inside each tipping edge.
func TestPlaqueStandStability(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			dm := m.Dimensions
			g0, err := m.Plaque.Centroid()
			require.NoError(t, err)
			first, err := m.Brackets[0].Bounds()
			require.NoError(t, err)
			last, err := m.Brackets[1].Bounds()
			require.NoError(t, err)
			rail, err := m.Rail.Bounds()
			require.NoError(t, err)
			require.Greater(t, rail.Min.Z, 0.0, "the rail never touches the desk")
			// The footprint is the union of the two brackets' bounds.
			box := first
			box.Max = last.Max

			// lean tilts the plaque's centre of gravity about the seat's line (parallel to X) by a, positive toward +Y.
			alpha := dm.PlayAngle * math.Pi / 180
			dy, dz := g0.Value.Y-dm.Seat.Y, g0.Value.Z-dm.Seat.Z
			lean := func(a float64) (y, z float64) {
				return dm.Seat.Y + dy*math.Cos(a) + dz*math.Sin(a), dm.Seat.Z - dy*math.Sin(a) + dz*math.Cos(a)
			}
			backY, backZ := lean(alpha)
			frontY, frontZ := lean(-alpha)
			require.Greater(t, backY, frontY)

			floor := box.Min.Z
			frontEdge := box.Min.Y + plaquestand.CornerRadius
			backEdge := box.Max.Y - plaquestand.CornerRadius
			front := math.Atan((frontY-frontEdge)/(frontZ-floor)) * 180 / math.Pi
			back := math.Atan((backEdge-backY)/(backZ-floor)) * 180 / math.Pi
			side := math.Min(g0.Value.X-(box.Min.X+plaquestand.ChamferSize), (box.Max.X-plaquestand.ChamferSize)-g0.Value.X)
			sideTip := math.Atan(side/(frontZ-floor)) * 180 / math.Pi

			for name, got := range map[string]float64{"front": front, "back": back, "side": sideTip} {
				require.GreaterOrEqual(t, got, plaquestand.MinTipAngle-0.01, "%s tip angle", name)
			}
			require.InDelta(t, dm.TipFront, front, 0.01)
			require.InDelta(t, dm.TipBack, back, 0.01)
			require.InDelta(t, dm.TipSide, sideTip, 0.01)
		})
	}
}
