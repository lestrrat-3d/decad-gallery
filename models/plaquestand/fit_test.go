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

// TestPlaqueStandClearance proves the Loose fit of the plaque in the slot: the proven lower bound of the gap between
// the two bodies meets the Loose class minimum in every case.
func TestPlaqueStandClearance(t *testing.T) {
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			_, m := requireBuilt(t, tc.params)
			plaque, stand := gapMesh(t, m.Plaque, 0.01), gapMesh(t, m.Stand, 0.01)
			ok, err := plaque.ExceedsClearance(stand, plaquestand.LooseFit)
			require.NoError(t, err)
			require.True(t, ok, "the plaque clears the stand by more than %g mm", plaquestand.LooseFit)
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
			box, err := m.Stand.Bounds()
			require.NoError(t, err)

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
