package clamparm_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

const derivedTol = 1e-9

func requireVec(t *testing.T, want, got r3.Vec) {
	t.Helper()
	require.InDelta(t, want.X, got.X, derivedTol, "X of %v", got)
	require.InDelta(t, want.Y, got.Y, derivedTol, "Y of %v", got)
	require.InDelta(t, want.Z, got.Z, derivedTol, "Z of %v", got)
}

// requireTip requires the bolt's tip point, built from the head, the direction and an independent distance.
func requireTip(t *testing.T, want r3.Vec, head, dir r3.Vec, dist float64) {
	t.Helper()
	requireVec(t, want, head.Add(dir.Scale(dist)))
}

// requireSiteTips checks every bolt's tip against the design values: each stays past its nut and inside the outer face.
func requireSiteTips(t *testing.T, jointH r3.Vec, d clamparm.Dimensions) {
	t.Helper()
	// J1: head at z = 0, tip z 38.5 (past the nut's top at 35.96, inside the knob's face at 40.26).
	requireTip(t, r3.NewVec(30, 0, 38.5), d.PanSite.Head, d.PanSite.Dir, 38.5)
	require.Equal(t, 38.5, d.PanSite.Head.Z+d.PanSite.Tip())
	// J2 to J4: heads at y = -10, tips at y 18.5 (past the nut's top at 17.7, inside the outer face at 22).
	for i, s := range d.JointSites {
		requireVec(t, r3.NewVec(0, 1, 0), s.Dir)
		require.InDelta(t, 18.5, s.Head.Y+s.Dir.Y*28.5, derivedTol, "J%d", i+2)
		require.InDelta(t, 28.5, s.Tip(), derivedTol, "J%d", i+2)
	}
	// Stud: 23.5 above the head's origin plane z, 13.5 above the interface face.
	requireTip(t, r3.NewVec(jointH.X+40, 0, jointH.Z+23.5), d.StudSite.Head, d.StudSite.Dir, 43.5)
	require.InDelta(t, 43.5, d.StudSite.Tip(), derivedTol)
}

// requireNoResidue requires that every entry of v within 1e-9 of 0 or of an integer is exactly it.
func requireNoResidue(t *testing.T, what string, v r3.Vec) {
	t.Helper()
	for i, x := range []float64{v.X, v.Y, v.Z} {
		near := math.Round(x)
		if math.Abs(x-near) < derivedTol {
			require.Equal(t, near, x, "%s entry %d is a rounding residue", what, i)
		}
	}
}
